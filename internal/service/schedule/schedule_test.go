package schedule_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/internal/service/schedule"
)

type repoMock struct {
	createSchedule func(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error)
}

func (m *repoMock) CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error) {
	return m.createSchedule(ctx, roomID, daysOfWeek, startTime, endTime)
}

func newService(repo schedule.ScheduleRepository) *schedule.Service {
	return schedule.New(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// CreateSchedule

func TestCreateSchedule_Success(t *testing.T) {
	roomID := uuid.New()
	want := model.Schedule{
		ID:         uuid.New(),
		RoomID:     roomID,
		DaysOfWeek: []int{1, 2, 3},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}

	svc := newService(&repoMock{
		createSchedule: func(_ context.Context, _ uuid.UUID, _ []int, _, _ string) (model.Schedule, error) {
			return want, nil
		},
	})

	got, err := svc.CreateSchedule(context.Background(), roomID, want.DaysOfWeek, want.StartTime, want.EndTime)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCreateSchedule_RoomNotFound(t *testing.T) {
	svc := newService(&repoMock{
		createSchedule: func(_ context.Context, _ uuid.UUID, _ []int, _, _ string) (model.Schedule, error) {
			return model.Schedule{}, model.ErrRoomNotFound
		},
	})

	_, err := svc.CreateSchedule(context.Background(), uuid.New(), []int{1}, "09:00", "18:00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrRoomNotFound))
}

func TestCreateSchedule_ScheduleExists(t *testing.T) {
	svc := newService(&repoMock{
		createSchedule: func(_ context.Context, _ uuid.UUID, _ []int, _, _ string) (model.Schedule, error) {
			return model.Schedule{}, model.ErrScheduleExists
		},
	})

	_, err := svc.CreateSchedule(context.Background(), uuid.New(), []int{1}, "09:00", "18:00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrScheduleExists))
}

func TestCreateSchedule_UnexpectedError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(&repoMock{
		createSchedule: func(_ context.Context, _ uuid.UUID, _ []int, _, _ string) (model.Schedule, error) {
			return model.Schedule{}, repoErr
		},
	})

	_, err := svc.CreateSchedule(context.Background(), uuid.New(), []int{1}, "09:00", "18:00")
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}
