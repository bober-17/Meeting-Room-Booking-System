package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
)

// Фиксированные UUID для dummyLogin — должны совпадать с seed-миграцией
// migrations/0006_seed_dummy_users.up.sql, чтобы user_id в токене
// соответствовал реальной записи в таблице users.
var (
	dummyAdminID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	dummyUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// sentinelHash — фиктивный хеш для защиты от timing oracle в Login:
// bcrypt-сравнение выполняется даже когда email не найден в БД.
var sentinelHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("sentinel"), bcrypt.DefaultCost)
	if err != nil {
		panic("auth: init sentinel hash: " + err.Error())
	}
	sentinelHash = h
}

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

// Аккаунты без пароля (созданные через DummyLogin) не могут войти через этот метод.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrInvalidCredentials) {
			// timing protection: всегда тратим ~то же время, что и bcrypt-сравнение
			_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(password))
			return "", fmt.Errorf("login: %w", model.ErrInvalidCredentials)
		}
		return "", fmt.Errorf("login: %w", err)
	}

	if user.Password == nil {
		return "", fmt.Errorf("login: %w", model.ErrInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", fmt.Errorf("login: bcrypt: %w", err)
		}
		return "", fmt.Errorf("login: %w", model.ErrInvalidCredentials)
	}

	token, err := pkgjwt.GenerateToken(user.ID, string(user.Role), s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	return token, nil
}
