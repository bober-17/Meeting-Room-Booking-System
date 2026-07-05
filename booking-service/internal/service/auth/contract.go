package auth

import (
	"context"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	CreateUser(ctx context.Context, email, passwordHash string, role model.Role) (model.User, error)
}
