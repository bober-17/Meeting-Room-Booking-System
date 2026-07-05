package notification

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
	"github.com/bober-17/meeting-room-booking-system/shared/events"
)

type Service struct {
	repo   Repository
	hub    Hub
	logger *slog.Logger
}

func New(repo Repository, hub Hub, logger *slog.Logger) *Service {
	return &Service{repo: repo, hub: hub, logger: logger}
}

func (s *Service) CreateFromEvent(ctx context.Context, event events.BookingEvent) error {
	var notifType model.NotificationType
	switch event.EventType {
	case events.EventTypeBookingCreated:
		notifType = model.TypeBookingCreated
	case events.EventTypeBookingCancelled:
		notifType = model.TypeBookingCancelled
	default:
		s.logger.WarnContext(ctx, "unknown event type, skipping",
			slog.String("event_type", string(event.EventType)),
			slog.String("event_id", event.EventID),
		)
		return nil
	}

	n := model.Notification{
		ID:        uuid.New().String(),
		UserID:    event.UserID,
		Type:      notifType,
		BookingID: event.BookingID,
		RoomName:  event.RoomName,
		SlotStart: event.SlotStart,
		SlotEnd:   event.SlotEnd,
		CreatedAt: time.Now().UTC(),
	}

	inserted, err := s.repo.Create(ctx, n)
	if err != nil {
		return err
	}

	if inserted && s.hub != nil {
		s.hub.Broadcast(event.UserID, n)
	}

	return nil
}

func (s *Service) ListByUserID(ctx context.Context, userID string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error) {
	return s.repo.ListByUserID(ctx, userID, limit, offset, unreadOnly)
}

func (s *Service) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkAsRead(ctx, id, userID)
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}
