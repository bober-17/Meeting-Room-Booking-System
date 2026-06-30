package slot

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

type RoomRepository interface {
	RoomExists(ctx context.Context, roomID uuid.UUID) (bool, error)
}

type ScheduleRepository interface {
	GetScheduleByRoomID(ctx context.Context, roomID uuid.UUID) (model.Schedule, error)
}

type SlotRepository interface {
	EnsureSlots(ctx context.Context, slots []model.Slot) error
	ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, from, to time.Time) ([]model.Slot, error)
}
