package notification

import (
	"context"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
	"github.com/bober-17/meeting-room-booking-system/shared/events"
)

type Repository interface {
	Create(ctx context.Context, n model.Notification) (bool, error)
	ListByUserID(ctx context.Context, userID string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error)
	MarkAsRead(ctx context.Context, id, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
}

type Hub interface {
	Broadcast(userID string, n model.Notification)
}

type NotificationService interface {
	CreateFromEvent(ctx context.Context, event events.BookingEvent) error
	ListByUserID(ctx context.Context, userID string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error)
	MarkAsRead(ctx context.Context, id, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
}
