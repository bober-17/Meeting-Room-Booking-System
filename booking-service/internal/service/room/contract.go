package room

import (
	"context"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

type RoomRepository interface {
	CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error)
	ListRooms(ctx context.Context) ([]model.Room, error)
}
