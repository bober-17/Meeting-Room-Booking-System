package auth_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/auth"
	"github.com/bober-17/meeting-room-booking-system/booking-service/mocks"
)

const testSecret = "test-secret"

func newService(t *testing.T, repo *mocks.MockUserRepository) *auth.Service {
	t.Helper()
	return auth.New(repo, testSecret, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// DummyLogin

func TestDummyLogin_AdminReturnsFixedUUID(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)

	token, err := newService(t, repo).DummyLogin(context.Background(), model.RoleAdmin)
	require.NoError(t, err)

	userID, role, err := pkgjwt.ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), userID)
	assert.Equal(t, string(model.RoleAdmin), role)
}

func TestDummyLogin_UserReturnsFixedUUID(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)

	token, err := newService(t, repo).DummyLogin(context.Background(), model.RoleUser)
	require.NoError(t, err)

	userID, role, err := pkgjwt.ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000002"), userID)
	assert.Equal(t, string(model.RoleUser), role)
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)

	_, err := newService(t, repo).DummyLogin(context.Background(), model.Role("superadmin"))
	require.Error(t, err)
}

// Register

func TestRegister_PasswordIsHashed(t *testing.T) {
	const rawPassword = "password123"
	repo := mocks.NewMockUserRepository(t)

	repo.On("CreateUser", context.Background(), "u@example.com", mock.MatchedBy(func(hash string) bool {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(rawPassword)) == nil
	}), model.RoleUser).Return(model.User{ID: uuid.New(), Email: "u@example.com", Role: model.RoleUser}, nil)

	_, err := newService(t, repo).Register(context.Background(), "u@example.com", rawPassword, model.RoleUser)
	require.NoError(t, err)
}

func TestRegister_EmailTaken(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)

	repo.On("CreateUser", context.Background(), "taken@example.com", mock.AnythingOfType("string"), model.RoleUser).
		Return(model.User{}, model.ErrEmailTaken)

	_, err := newService(t, repo).Register(context.Background(), "taken@example.com", "pass", model.RoleUser)
	assert.ErrorIs(t, err, model.ErrEmailTaken)
}

// Login

func TestLogin_Success(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	wantID := uuid.New()
	hashStr := string(hash)
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetUserByEmail", context.Background(), "u@example.com").
		Return(model.User{ID: wantID, Email: "u@example.com", Password: &hashStr, Role: model.RoleUser}, nil)

	token, err := newService(t, repo).Login(context.Background(), "u@example.com", "secret")
	require.NoError(t, err)

	gotID, role, err := pkgjwt.ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, wantID, gotID)
	assert.Equal(t, string(model.RoleUser), role)
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	require.NoError(t, err)

	hashStr := string(hash)
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetUserByEmail", context.Background(), "u@example.com").
		Return(model.User{ID: uuid.New(), Password: &hashStr, Role: model.RoleUser}, nil)

	_, err = newService(t, repo).Login(context.Background(), "u@example.com", "wrong")
	assert.ErrorIs(t, err, model.ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetUserByEmail", context.Background(), "nobody@example.com").
		Return(model.User{}, model.ErrInvalidCredentials)

	_, err := newService(t, repo).Login(context.Background(), "nobody@example.com", "pass")
	assert.ErrorIs(t, err, model.ErrInvalidCredentials)
}

func TestLogin_DummyUserHasNoPassword(t *testing.T) {
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetUserByEmail", context.Background(), "admin@test.com").
		Return(model.User{ID: uuid.New(), Email: "admin@test.com", Password: nil, Role: model.RoleAdmin}, nil)

	_, err := newService(t, repo).Login(context.Background(), "admin@test.com", "anything")
	assert.ErrorIs(t, err, model.ErrInvalidCredentials)
}
