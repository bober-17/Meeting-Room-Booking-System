package room

import (
	"context"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type RoomRepository interface {
	CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error)
	ListRooms(ctx context.Context) ([]model.Room, error)
}
