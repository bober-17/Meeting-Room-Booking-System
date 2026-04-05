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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/internal/service/slot"
	"github.com/internships-backend/test-backend-bober-17/mocks"
)

func newService(t *testing.T, rr *mocks.MockSlotRoomRepository, sr *mocks.MockSlotScheduleRepository, slr *mocks.MockSlotRepository) *slot.Service {
	t.Helper()
	return slot.New(rr, sr, slr, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// ListAvailableSlots

func TestListAvailableSlots_RoomNotFound(t *testing.T) {
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(false, nil)

	_, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrRoomNotFound))
}

func TestListAvailableSlots_RoomRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(false, repoErr)

	_, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_NoSchedule(t *testing.T) {
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{}, model.ErrScheduleNotFound)

	slots, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, time.Now())
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestListAvailableSlots_ScheduleRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{}, repoErr)

	_, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, time.Now())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_DayNotInSchedule(t *testing.T) {
	// 2024-06-10 — понедельник (ISO=1), расписание только на вт–пт
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{
		DaysOfWeek: []int{2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}, nil)

	slots, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestListAvailableSlots_SundayIsISO7(t *testing.T) {
	// 2024-06-09 — воскресенье (Go Weekday()=0, ISO=7)
	date := time.Date(2024, 6, 9, 0, 0, 0, 0, time.UTC)
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{
		DaysOfWeek: []int{7},
		StartTime:  "10:00",
		EndTime:    "12:00",
	}, nil)
	slr.On("EnsureSlots", context.Background(), mock4slots()).Return(nil)
	slr.On("ListAvailableByRoomAndDate", context.Background(), roomID,
		time.Date(2024, 6, 9, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
	).Return(make([]model.Slot, 4), nil)

	got, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Len(t, got, 4)
}

func TestListAvailableSlots_EnsureSlotsError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	// 2024-06-10 — понедельник (ISO=1)
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}, nil)
	slr.On("EnsureSlots", context.Background(), anySlots()).Return(repoErr)

	_, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, date)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestListAvailableSlots_ListAvailableError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	// 2024-06-10 — понедельник (ISO=1)
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	roomID := uuid.New()
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}, nil)
	slr.On("EnsureSlots", context.Background(), anySlots()).Return(nil)
	slr.On("ListAvailableByRoomAndDate", context.Background(), roomID,
		time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 11, 0, 0, 0, 0, time.UTC),
	).Return([]model.Slot(nil), repoErr)

	_, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, date)
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
	rr := mocks.NewMockSlotRoomRepository(t)
	sr := mocks.NewMockSlotScheduleRepository(t)
	slr := mocks.NewMockSlotRepository(t)

	rr.On("RoomExists", context.Background(), roomID).Return(true, nil)
	sr.On("GetScheduleByRoomID", context.Background(), roomID).Return(model.Schedule{
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}, nil)
	slr.On("EnsureSlots", context.Background(), anySlots()).Return(nil)
	slr.On("ListAvailableByRoomAndDate", context.Background(), roomID,
		time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 6, 11, 0, 0, 0, 0, time.UTC),
	).Return(want, nil)

	got, err := newService(t, rr, sr, slr).ListAvailableSlots(context.Background(), roomID, date)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// anySlots возвращает matcher, принимающий любой []model.Slot.
func anySlots() interface{} {
	return mock.MatchedBy(func(_ []model.Slot) bool { return true })
}

// mock4slots возвращает matcher для ровно 4 слотов (10:00–12:00 с шагом 30 мин).
func mock4slots() interface{} {
	return mock.MatchedBy(func(s []model.Slot) bool { return len(s) == 4 })
}
