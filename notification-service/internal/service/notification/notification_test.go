package notification_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
	svc "github.com/bober-17/meeting-room-booking-system/notification-service/internal/service/notification"
	"github.com/bober-17/meeting-room-booking-system/notification-service/mocks"
	"github.com/bober-17/meeting-room-booking-system/shared/events"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func matchNotification(event events.BookingEvent, expectedType model.NotificationType) any {
	return mock.MatchedBy(func(n model.Notification) bool {
		return n.UserID == event.UserID &&
			n.BookingID == event.BookingID &&
			n.RoomName == event.RoomName &&
			n.Type == expectedType &&
			!n.IsRead
	})
}

func bookingEvent(eventType events.EventType) events.BookingEvent {
	return events.BookingEvent{
		EventID:    uuid.New().String(),
		EventType:  eventType,
		BookingID:  uuid.New().String(),
		UserID:     uuid.New().String(),
		RoomName:   "Переговорная 1",
		SlotStart:  time.Now().UTC().Add(time.Hour),
		SlotEnd:    time.Now().UTC().Add(2 * time.Hour),
		OccurredAt: time.Now().UTC(),
	}
}

func TestCreateFromEvent_BookingCreated_InsertsAndBroadcasts(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	event := bookingEvent(events.EventTypeBookingCreated)
	matcher := matchNotification(event, model.TypeBookingCreated)

	repo.On("Create", context.Background(), matcher).Return(true, nil)
	hub.On("Broadcast", event.UserID, matcher).Return()

	require.NoError(t, s.CreateFromEvent(context.Background(), event))
}

func TestCreateFromEvent_BookingCancelled_InsertsAndBroadcasts(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	event := bookingEvent(events.EventTypeBookingCancelled)
	matcher := matchNotification(event, model.TypeBookingCancelled)

	repo.On("Create", context.Background(), matcher).Return(true, nil)
	hub.On("Broadcast", event.UserID, matcher).Return()

	require.NoError(t, s.CreateFromEvent(context.Background(), event))
}

func TestCreateFromEvent_Duplicate_NoBroadcast(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	event := bookingEvent(events.EventTypeBookingCreated)

	repo.On("Create", context.Background(), matchNotification(event, model.TypeBookingCreated)).Return(false, nil)

	err := s.CreateFromEvent(context.Background(), event)
	require.NoError(t, err)
}

func TestCreateFromEvent_DBError_ReturnsError(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	event := bookingEvent(events.EventTypeBookingCreated)
	dbErr := assert.AnError

	repo.On("Create", context.Background(), matchNotification(event, model.TypeBookingCreated)).Return(false, dbErr)

	err := s.CreateFromEvent(context.Background(), event)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateFromEvent_UnknownType_ReturnsNil(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	event := bookingEvent("booking.unknown")

	// repo и hub не должны вызываться
	err := s.CreateFromEvent(context.Background(), event)
	require.NoError(t, err)
}

func TestListByUserID_DelegatesParams(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	userID := uuid.New().String()
	want := []model.Notification{{ID: uuid.New().String(), UserID: userID}}
	repo.On("ListByUserID", context.Background(), userID, 10, 0, true).Return(want, 1, nil)

	ns, total, err := s.ListByUserID(context.Background(), userID, 10, 0, true)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, want, ns)
}

func TestMarkAsRead_NotFound_ReturnsError(t *testing.T) {
	repo := mocks.NewMockRepository(t)
	hub := mocks.NewMockHub(t)
	s := svc.New(repo, hub, discardLogger)

	id := uuid.New().String()
	userID := uuid.New().String()
	repo.On("MarkAsRead", context.Background(), id, userID).Return(model.ErrNotificationNotFound)

	err := s.MarkAsRead(context.Background(), id, userID)
	assert.ErrorIs(t, err, model.ErrNotificationNotFound)
}
