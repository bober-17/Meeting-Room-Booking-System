package schedule

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type Service struct {
	repo ScheduleRepository
	log  *slog.Logger
}

func New(repo ScheduleRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error) {
	schedule, err := s.repo.CreateSchedule(ctx, roomID, daysOfWeek, startTime, endTime)
	if err != nil {
		return model.Schedule{}, fmt.Errorf("schedule service: %w", err)
	}

	return schedule, nil
}
