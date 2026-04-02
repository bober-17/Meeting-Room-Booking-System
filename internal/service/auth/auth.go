package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	pkgjwt "github.com/internships-backend/test-backend-bober-17/internal/pkg/jwt"
)

// Фиксированные UUID для dummyLogin — должны совпадать с seed-миграцией
// migrations/0006_seed_dummy_users.up.sql, чтобы user_id в токене
// соответствовал реальной записи в таблице users.
var (
	dummyAdminID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	dummyUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type Service struct {
	repo      UserRepository
	jwtSecret string
	log       *slog.Logger
}

func New(repo UserRepository, jwtSecret string, log *slog.Logger) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		log:       log}
}

func (s *Service) DummyLogin(_ context.Context, role model.Role) (string, error) {
	var userID uuid.UUID
	switch role {
	case model.RoleAdmin:
		userID = dummyAdminID
	case model.RoleUser:
		userID = dummyUserID
	default:
		return "", fmt.Errorf("dummy login: unknown role %q", role)
	}

	token, err := pkgjwt.GenerateToken(userID, string(role), s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("dummy login: %w", err)
	}

	return token, nil
}

func (s *Service) Register(ctx context.Context, email, password string, role model.Role) (model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("register: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, email, string(hash), role)
	if err != nil {
		return model.User{}, fmt.Errorf("register: %w", err)
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	// Дамми-пользователи создаются через /dummyLogin без пароля (password = NULL).
	// Войти через /login с таким аккаунтом невозможно.
	if user.Password == nil {
		return "", fmt.Errorf("login: %w", model.ErrInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return "", fmt.Errorf("login: %w", model.ErrInvalidCredentials)
	}

	token, err := pkgjwt.GenerateToken(user.ID, string(user.Role), s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	return token, nil
}
