package auth

import (
	"context"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	CreateUser(ctx context.Context, email, passwordHash string, role model.Role) (model.User, error)
}
