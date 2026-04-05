package schedule

import (
	"context"

	"github.com/google/uuid"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type ScheduleRepository interface {
	CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error)
}
