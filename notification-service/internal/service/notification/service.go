package notification

import (
	"context"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
	"github.com/bober-17/meeting-room-booking-system/shared/events"
)

type Service struct {
	repo Repository
	hub  Hub
}

func New(repo Repository, hub Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

func (s *Service) CreateFromEvent(_ context.Context, _ events.BookingEvent) error {
	panic("not implemented")
}

func (s *Service) ListByUserID(_ context.Context, _ string, _, _ int) ([]model.Notification, int, error) {
	panic("not implemented")
}

func (s *Service) MarkAsRead(_ context.Context, _, _ string) error {
	panic("not implemented")
}

func (s *Service) MarkAllAsRead(_ context.Context, _ string) error {
	panic("not implemented")
}
