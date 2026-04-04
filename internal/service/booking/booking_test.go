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
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/internal/service/booking"
)

type bookingRepoMock struct {
	createBooking        func(ctx context.Context, slotID, userID uuid.UUID) (model.Booking, error)
	getBookingByID       func(ctx context.Context, id uuid.UUID) (model.Booking, error)
	cancelBooking        func(ctx context.Context, id, userID uuid.UUID) (model.Booking, error)
	listBookings         func(ctx context.Context, page, pageSize int) ([]model.Booking, int, error)
	listUserBookings     func(ctx context.Context, userID uuid.UUID) ([]model.Booking, error)
	updateConferenceLink func(ctx context.Context, bookingID uuid.UUID, link string) error
}

func (m *bookingRepoMock) CreateBooking(ctx context.Context, slotID, userID uuid.UUID) (model.Booking, error) {
	return m.createBooking(ctx, slotID, userID)
}
func (m *bookingRepoMock) GetBookingByID(ctx context.Context, id uuid.UUID) (model.Booking, error) {
	return m.getBookingByID(ctx, id)
}
func (m *bookingRepoMock) CancelBooking(ctx context.Context, id, userID uuid.UUID) (model.Booking, error) {
	return m.cancelBooking(ctx, id, userID)
}
func (m *bookingRepoMock) ListBookings(ctx context.Context, page, pageSize int) ([]model.Booking, int, error) {
	return m.listBookings(ctx, page, pageSize)
}
func (m *bookingRepoMock) ListUserBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error) {
	return m.listUserBookings(ctx, userID)
}
func (m *bookingRepoMock) UpdateConferenceLink(ctx context.Context, bookingID uuid.UUID, link string) error {
	return m.updateConferenceLink(ctx, bookingID, link)
}

type slotRepoMock struct {
	getSlotByID func(ctx context.Context, id uuid.UUID) (model.Slot, error)
}

func (m *slotRepoMock) GetSlotByID(ctx context.Context, id uuid.UUID) (model.Slot, error) {
	return m.getSlotByID(ctx, id)
}

type conferenceClientMock struct {
	createLink func(ctx context.Context, bookingID uuid.UUID) (string, error)
}

func (m *conferenceClientMock) CreateLink(ctx context.Context, bookingID uuid.UUID) (string, error) {
	return m.createLink(ctx, bookingID)
}

func newService(br booking.BookingRepository, sr booking.SlotRepository, cc booking.ConferenceClient) *booking.Service {
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
	svc := newService(
		&bookingRepoMock{},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return model.Slot{}, model.ErrSlotNotFound
		}},
		&conferenceClientMock{},
	)

	_, err := svc.CreateBooking(context.Background(), uuid.New(), uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotNotFound))
}

func TestCreateBooking_SlotRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&bookingRepoMock{},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return model.Slot{}, repoErr
		}},
		&conferenceClientMock{},
	)

	_, err := svc.CreateBooking(context.Background(), uuid.New(), uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestCreateBooking_SlotInPast(t *testing.T) {
	svc := newService(
		&bookingRepoMock{},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slotInPast(), nil
		}},
		&conferenceClientMock{},
	)

	_, err := svc.CreateBooking(context.Background(), uuid.New(), uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotInPast))
}

func TestCreateBooking_SlotAlreadyBooked(t *testing.T) {
	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return model.Booking{}, model.ErrSlotAlreadyBooked
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slotInFuture(), nil
		}},
		&conferenceClientMock{},
	)

	_, err := svc.CreateBooking(context.Background(), uuid.New(), uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrSlotAlreadyBooked))
}

func TestCreateBooking_BookingRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return model.Booking{}, repoErr
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slotInFuture(), nil
		}},
		&conferenceClientMock{},
	)

	_, err := svc.CreateBooking(context.Background(), uuid.New(), uuid.New(), false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestCreateBooking_HappyPath_NoConferenceLink(t *testing.T) {
	slot := slotInFuture()
	userID := uuid.New()
	want := model.Booking{
		ID:     uuid.New(),
		SlotID: slot.ID,
		UserID: userID,
		Status: model.BookingStatusActive,
	}

	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, slotID, uid uuid.UUID) (model.Booking, error) {
				assert.Equal(t, slot.ID, slotID)
				assert.Equal(t, userID, uid)
				return want, nil
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slot, nil
		}},
		&conferenceClientMock{},
	)

	got, err := svc.CreateBooking(context.Background(), slot.ID, userID, false)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Nil(t, got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_Success(t *testing.T) {
	slot := slotInFuture()
	bookingID := uuid.New()
	expectedLink := "https://conference.example/meeting/" + bookingID.String()
	created := model.Booking{ID: bookingID, SlotID: slot.ID, Status: model.BookingStatusActive}

	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return created, nil
			},
			updateConferenceLink: func(_ context.Context, bID uuid.UUID, link string) error {
				assert.Equal(t, bookingID, bID)
				assert.Equal(t, expectedLink, link)
				return nil
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slot, nil
		}},
		&conferenceClientMock{
			createLink: func(_ context.Context, bID uuid.UUID) (string, error) {
				assert.Equal(t, bookingID, bID)
				return expectedLink, nil
			},
		},
	)

	got, err := svc.CreateBooking(context.Background(), slot.ID, uuid.New(), true)
	require.NoError(t, err)
	require.NotNil(t, got.ConferenceLink)
	assert.Equal(t, expectedLink, *got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_ClientError_BookingStillCreated(t *testing.T) {
	// сбой внешнего сервиса — бронь создана, conferenceLink остаётся nil
	slot := slotInFuture()
	created := model.Booking{ID: uuid.New(), SlotID: slot.ID, Status: model.BookingStatusActive}

	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return created, nil
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slot, nil
		}},
		&conferenceClientMock{
			createLink: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "", errors.New("conference service unavailable")
			},
		},
	)

	got, err := svc.CreateBooking(context.Background(), slot.ID, uuid.New(), true)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Nil(t, got.ConferenceLink)
}

func TestCreateBooking_WithConferenceLink_UpdateError_BookingStillCreated(t *testing.T) {
	// конференция создана, но UPDATE упал — бронь валидна, conferenceLink nil
	slot := slotInFuture()
	created := model.Booking{ID: uuid.New(), SlotID: slot.ID, Status: model.BookingStatusActive}

	svc := newService(
		&bookingRepoMock{
			createBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return created, nil
			},
			updateConferenceLink: func(_ context.Context, _ uuid.UUID, _ string) error {
				return errors.New("db unavailable")
			},
		},
		&slotRepoMock{getSlotByID: func(_ context.Context, _ uuid.UUID) (model.Slot, error) {
			return slot, nil
		}},
		&conferenceClientMock{
			createLink: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "https://conference.example/meeting/xxx", nil
			},
		},
	)

	got, err := svc.CreateBooking(context.Background(), slot.ID, uuid.New(), true)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Nil(t, got.ConferenceLink)
}

// CancelBooking

func TestCancelBooking_BookingNotFound(t *testing.T) {
	svc := newService(
		&bookingRepoMock{
			cancelBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return model.Booking{}, model.ErrBookingNotFound
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	_, err := svc.CancelBooking(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrBookingNotFound))
}

func TestCancelBooking_Forbidden(t *testing.T) {
	svc := newService(
		&bookingRepoMock{
			cancelBooking: func(_ context.Context, _, _ uuid.UUID) (model.Booking, error) {
				return model.Booking{}, model.ErrForbidden
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	_, err := svc.CancelBooking(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrForbidden))
}

func TestCancelBooking_AlreadyCancelled_Idempotent(t *testing.T) {
	// повторная отмена возвращает бронь без ошибки
	bookingID := uuid.New()
	userID := uuid.New()
	cancelled := model.Booking{ID: bookingID, UserID: userID, Status: model.BookingStatusCancelled}

	svc := newService(
		&bookingRepoMock{
			cancelBooking: func(_ context.Context, id, uid uuid.UUID) (model.Booking, error) {
				assert.Equal(t, bookingID, id)
				assert.Equal(t, userID, uid)
				return cancelled, nil
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	got, err := svc.CancelBooking(context.Background(), bookingID, userID)
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusCancelled, got.Status)
}

func TestCancelBooking_HappyPath(t *testing.T) {
	bookingID := uuid.New()
	userID := uuid.New()
	want := model.Booking{ID: bookingID, UserID: userID, Status: model.BookingStatusCancelled}

	svc := newService(
		&bookingRepoMock{
			cancelBooking: func(_ context.Context, id, uid uuid.UUID) (model.Booking, error) {
				assert.Equal(t, bookingID, id)
				assert.Equal(t, userID, uid)
				return want, nil
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	got, err := svc.CancelBooking(context.Background(), bookingID, userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// ListBookings

func TestListBookings_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&bookingRepoMock{
			listBookings: func(_ context.Context, _, _ int) ([]model.Booking, int, error) {
				return nil, 0, repoErr
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	_, _, err := svc.ListBookings(context.Background(), 1, 20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListBookings_HappyPath(t *testing.T) {
	want := []model.Booking{
		{ID: uuid.New(), Status: model.BookingStatusActive},
		{ID: uuid.New(), Status: model.BookingStatusCancelled},
	}

	svc := newService(
		&bookingRepoMock{
			listBookings: func(_ context.Context, page, pageSize int) ([]model.Booking, int, error) {
				assert.Equal(t, 2, page)
				assert.Equal(t, 10, pageSize)
				return want, 42, nil
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	got, total, err := svc.ListBookings(context.Background(), 2, 10)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, 42, total)
}

// ListUserBookings

func TestListUserBookings_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&bookingRepoMock{
			listUserBookings: func(_ context.Context, _ uuid.UUID) ([]model.Booking, error) {
				return nil, repoErr
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	_, err := svc.ListUserBookings(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListUserBookings_HappyPath(t *testing.T) {
	userID := uuid.New()
	want := []model.Booking{
		{ID: uuid.New(), UserID: userID, Status: model.BookingStatusActive},
	}

	svc := newService(
		&bookingRepoMock{
			listUserBookings: func(_ context.Context, uid uuid.UUID) ([]model.Booking, error) {
				assert.Equal(t, userID, uid)
				return want, nil
			},
		},
		&slotRepoMock{},
		&conferenceClientMock{},
	)

	got, err := svc.ListUserBookings(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
