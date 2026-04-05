package booking_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/internal/service/booking"
	"github.com/internships-backend/test-backend-bober-17/mocks"
)

func newService(t *testing.T, br *mocks.MockBookingRepository, sr *mocks.MockBookingSlotRepository, cc *mocks.MockConferenceClient) *booking.Service {
	t.Helper()
	return booking.New(br, sr, cc, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

func slotInFuture() model.Slot {
	return model.Slot{
		ID:      uuid.New(),
		RoomID:  uuid.New(),
		StartAt: time.Now().Add(24 * time.Hour),
		EndAt:   time.Now().Add(24*time.Hour + 30*time.Minute),
	}
}

func slotInPast() model.Slot {
	return model.Slot{
		ID:      uuid.New(),
		RoomID:  uuid.New(),
		StartAt: time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC),
		EndAt:   time.Date(2020, 1, 1, 9, 30, 0, 0, time.UTC),
	}
}

// CreateBooking

func TestCreateBooking_SlotNotFound(t *testing.T) {
	slotID := uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slotID).Return(model.Slot{}, model.ErrSlotNotFound)

	_, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slotID, uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotNotFound))
}

func TestCreateBooking_SlotRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	slotID := uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slotID).Return(model.Slot{}, repoErr)

	_, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slotID, uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestCreateBooking_SlotInPast(t *testing.T) {
	slot := slotInPast()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)

	_, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotInPast))
}

func TestCreateBooking_SlotAlreadyBooked(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(model.Booking{}, model.ErrSlotAlreadyBooked)

	_, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotAlreadyBooked))
}

func TestCreateBooking_BookingRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	slot := slotInFuture()
	userID := uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(model.Booking{}, repoErr)

	_, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestCreateBooking_HappyPath_NoConferenceLink(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	want := model.Booking{ID: uuid.New(), SlotID: slot.ID, UserID: userID, Status: model.BookingStatusActive}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(want, nil)

	got, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, false)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Nil(t, got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_Success(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	created := model.Booking{ID: uuid.New(), SlotID: slot.ID, Status: model.BookingStatusActive}
	expectedLink := "https://conference.example/meeting/" + created.ID.String()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(created, nil)
	cc.On("CreateLink", mock.Anything, created.ID).Return(expectedLink, nil)
	br.On("UpdateConferenceLink", context.Background(), created.ID, expectedLink).Return(nil)

	got, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, true)
	require.NoError(t, err)
	require.NotNil(t, got.ConferenceLink)
	assert.Equal(t, expectedLink, *got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_ClientError_BookingStillCreated(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	created := model.Booking{ID: uuid.New(), SlotID: slot.ID, Status: model.BookingStatusActive}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(created, nil)
	cc.On("CreateLink", mock.Anything, created.ID).Return("", errors.New("conference service unavailable"))

	got, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, true)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Nil(t, got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_UpdateError_BookingStillCreated(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	link := "https://conference.example/meeting/xxx"
	created := model.Booking{ID: uuid.New(), SlotID: slot.ID, Status: model.BookingStatusActive}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	sr.On("GetSlotByID", context.Background(), slot.ID).Return(slot, nil)
	br.On("CreateBooking", context.Background(), slot.ID, userID).Return(created, nil)
	cc.On("CreateLink", mock.Anything, created.ID).Return(link, nil)
	br.On("UpdateConferenceLink", context.Background(), created.ID, link).Return(errors.New("db unavailable"))

	got, err := newService(t, br, sr, cc).CreateBooking(context.Background(), slot.ID, userID, true)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Nil(t, got.ConferenceLink)
}

// CancelBooking

func TestCancelBooking_BookingNotFound(t *testing.T) {
	bookingID, userID := uuid.New(), uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("CancelBooking", context.Background(), bookingID, userID).Return(model.Booking{}, model.ErrBookingNotFound)

	_, err := newService(t, br, sr, cc).CancelBooking(context.Background(), bookingID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrBookingNotFound))
}

func TestCancelBooking_Forbidden(t *testing.T) {
	bookingID, userID := uuid.New(), uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("CancelBooking", context.Background(), bookingID, userID).Return(model.Booking{}, model.ErrForbidden)

	_, err := newService(t, br, sr, cc).CancelBooking(context.Background(), bookingID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrForbidden))
}

func TestCancelBooking_AlreadyCancelled_Idempotent(t *testing.T) {
	bookingID, userID := uuid.New(), uuid.New()
	cancelled := model.Booking{ID: bookingID, UserID: userID, Status: model.BookingStatusCancelled}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("CancelBooking", context.Background(), bookingID, userID).Return(cancelled, nil)

	got, err := newService(t, br, sr, cc).CancelBooking(context.Background(), bookingID, userID)
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusCancelled, got.Status)
}

func TestCancelBooking_HappyPath(t *testing.T) {
	bookingID, userID := uuid.New(), uuid.New()
	want := model.Booking{ID: bookingID, UserID: userID, Status: model.BookingStatusCancelled}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("CancelBooking", context.Background(), bookingID, userID).Return(want, nil)

	got, err := newService(t, br, sr, cc).CancelBooking(context.Background(), bookingID, userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// ListBookings

func TestListBookings_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("ListBookings", context.Background(), 1, 20).Return([]model.Booking(nil), 0, repoErr)

	_, _, err := newService(t, br, sr, cc).ListBookings(context.Background(), 1, 20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListBookings_HappyPath(t *testing.T) {
	want := []model.Booking{
		{ID: uuid.New(), Status: model.BookingStatusActive},
		{ID: uuid.New(), Status: model.BookingStatusCancelled},
	}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("ListBookings", context.Background(), 2, 10).Return(want, 42, nil)

	got, total, err := newService(t, br, sr, cc).ListBookings(context.Background(), 2, 10)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, 42, total)
}

// ListUserBookings

func TestListUserBookings_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	userID := uuid.New()
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("ListUserBookings", context.Background(), userID).Return([]model.Booking(nil), repoErr)

	_, err := newService(t, br, sr, cc).ListUserBookings(context.Background(), userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListUserBookings_HappyPath(t *testing.T) {
	userID := uuid.New()
	want := []model.Booking{{ID: uuid.New(), UserID: userID, Status: model.BookingStatusActive}}
	br := mocks.NewMockBookingRepository(t)
	sr := mocks.NewMockBookingSlotRepository(t)
	cc := mocks.NewMockConferenceClient(t)

	br.On("ListUserBookings", context.Background(), userID).Return(want, nil)

	got, err := newService(t, br, sr, cc).ListUserBookings(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
