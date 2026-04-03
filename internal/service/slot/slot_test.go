package slot_test

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
	"github.com/internships-backend/test-backend-bober-17/internal/service/slot"
)

type roomRepoMock struct {
	roomExists func(ctx context.Context, roomID uuid.UUID) (bool, error)
}

func (m *roomRepoMock) RoomExists(ctx context.Context, roomID uuid.UUID) (bool, error) {
	return m.roomExists(ctx, roomID)
}

type scheduleRepoMock struct {
	getScheduleByRoomID func(ctx context.Context, roomID uuid.UUID) (model.Schedule, error)
}

func (m *scheduleRepoMock) GetScheduleByRoomID(ctx context.Context, roomID uuid.UUID) (model.Schedule, error) {
	return m.getScheduleByRoomID(ctx, roomID)
}

type slotRepoMock struct {
	ensureSlots                func(ctx context.Context, slots []model.Slot) error
	listAvailableByRoomAndDate func(ctx context.Context, roomID uuid.UUID, from, to time.Time) ([]model.Slot, error)
}

func (m *slotRepoMock) EnsureSlots(ctx context.Context, slots []model.Slot) error {
	return m.ensureSlots(ctx, slots)
}

func (m *slotRepoMock) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, from, to time.Time) ([]model.Slot, error) {
	return m.listAvailableByRoomAndDate(ctx, roomID, from, to)
}

func newService(roomRepo slot.RoomRepository, schedRepo slot.ScheduleRepository, slotRepo slot.SlotRepository) *slot.Service {
	return slot.New(roomRepo, schedRepo, slotRepo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// ListAvailableSlots

func TestListAvailableSlots_RoomNotFound(t *testing.T) {
	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		}},
		&scheduleRepoMock{},
		&slotRepoMock{},
	)

	_, err := svc.ListAvailableSlots(context.Background(), uuid.New(), time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrRoomNotFound))
}

func TestListAvailableSlots_RoomRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, repoErr
		}},
		&scheduleRepoMock{},
		&slotRepoMock{},
	)

	_, err := svc.ListAvailableSlots(context.Background(), uuid.New(), time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_NoSchedule(t *testing.T) {
	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{}, model.ErrScheduleNotFound
		}},
		&slotRepoMock{},
	)

	slots, err := svc.ListAvailableSlots(context.Background(), uuid.New(), time.Now())
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestListAvailableSlots_ScheduleRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{}, repoErr
		}},
		&slotRepoMock{},
	)

	_, err := svc.ListAvailableSlots(context.Background(), uuid.New(), time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_DayNotInSchedule(t *testing.T) {
	// 2024-06-10 — понедельник (ISO=1), расписание только на вт-пт
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{
				DaysOfWeek: []int{2, 3, 4, 5},
				StartTime:  "09:00",
				EndTime:    "18:00",
			}, nil
		}},
		&slotRepoMock{},
	)

	slots, err := svc.ListAvailableSlots(context.Background(), uuid.New(), date)
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestListAvailableSlots_SundayIsISO7(t *testing.T) {
	// 2024-06-09 — воскресенье (Go Weekday()=0, ISO=7)
	// Проверяем корректное маппирование: если в расписании день 7, воскресенье должно подойти
	date := time.Date(2024, 6, 9, 0, 0, 0, 0, time.UTC)

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{
				DaysOfWeek: []int{7},
				StartTime:  "10:00",
				EndTime:    "12:00",
			}, nil
		}},
		&slotRepoMock{
			ensureSlots: func(_ context.Context, slots []model.Slot) error {
				assert.Len(t, slots, 4) // 10:00, 10:30, 11:00, 11:30
				return nil
			},
			listAvailableByRoomAndDate: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]model.Slot, error) {
				return []model.Slot{{}, {}, {}, {}}, nil
			},
		},
	)

	got, err := svc.ListAvailableSlots(context.Background(), uuid.New(), date)
	require.NoError(t, err)
	assert.Len(t, got, 4)
}

func TestListAvailableSlots_EnsureSlotsError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	// 2024-06-10 — понедельник (ISO=1)
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "10:00"}, nil
		}},
		&slotRepoMock{
			ensureSlots: func(_ context.Context, _ []model.Slot) error {
				return repoErr
			},
		},
	)

	_, err := svc.ListAvailableSlots(context.Background(), uuid.New(), date)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_ListAvailableError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	// 2024-06-10 — понедельник (ISO=1)
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "10:00"}, nil
		}},
		&slotRepoMock{
			ensureSlots: func(_ context.Context, _ []model.Slot) error { return nil },
			listAvailableByRoomAndDate: func(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]model.Slot, error) {
				return nil, repoErr
			},
		},
	)

	_, err := svc.ListAvailableSlots(context.Background(), uuid.New(), date)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_HappyPath(t *testing.T) {
	roomID := uuid.New()
	// 2024-06-10 — понедельник (ISO=1)
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)

	want := []model.Slot{
		{ID: uuid.New(), RoomID: roomID, StartAt: time.Date(2024, 6, 10, 9, 0, 0, 0, time.UTC), EndAt: time.Date(2024, 6, 10, 9, 30, 0, 0, time.UTC)},
		{ID: uuid.New(), RoomID: roomID, StartAt: time.Date(2024, 6, 10, 9, 30, 0, 0, time.UTC), EndAt: time.Date(2024, 6, 10, 10, 0, 0, 0, time.UTC)},
	}

	svc := newService(
		&roomRepoMock{roomExists: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		}},
		&scheduleRepoMock{getScheduleByRoomID: func(_ context.Context, _ uuid.UUID) (model.Schedule, error) {
			return model.Schedule{
				DaysOfWeek: []int{1, 2, 3, 4, 5},
				StartTime:  "09:00",
				EndTime:    "10:00",
			}, nil
		}},
		&slotRepoMock{
			ensureSlots: func(_ context.Context, slots []model.Slot) error {
				assert.Len(t, slots, 2)
				return nil
			},
			listAvailableByRoomAndDate: func(_ context.Context, _ uuid.UUID, from, to time.Time) ([]model.Slot, error) {
				assert.Equal(t, time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC), from)
				assert.Equal(t, time.Date(2024, 6, 11, 0, 0, 0, 0, time.UTC), to)
				return want, nil
			},
		},
	)

	got, err := svc.ListAvailableSlots(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
