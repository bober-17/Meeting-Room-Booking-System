package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	pkgjwt "github.com/internships-backend/test-backend-bober-17/internal/pkg/jwt"
	"github.com/internships-backend/test-backend-bober-17/internal/service/auth"
)

const testSecret = "test-secret"

type repoMock struct {
	getUserByEmail func(ctx context.Context, email string) (model.User, error)
	createUser     func(ctx context.Context, email, passwordHash string, role model.Role) (model.User, error)
}

func (m *repoMock) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	return m.getUserByEmail(ctx, email)
}

func (m *repoMock) CreateUser(ctx context.Context, email, passwordHash string, role model.Role) (model.User, error) {
	return m.createUser(ctx, email, passwordHash, role)
}

func newService(repo auth.UserRepository) *auth.Service {
	return auth.New(repo, testSecret, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// DummyLogin
func TestDummyLogin_AdminReturnsFixedUUID(t *testing.T) {
	svc := newService(&repoMock{})

	token, err := svc.DummyLogin(context.Background(), model.RoleAdmin)
	require.NoError(t, err)

	userID, role, err := pkgjwt.ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), userID)
	assert.Equal(t, string(model.RoleAdmin), role)
}

func TestDummyLogin_UserReturnsFixedUUID(t *testing.T) {
	svc := newService(&repoMock{})

	token, err := svc.DummyLogin(context.Background(), model.RoleUser)
	require.NoError(t, err)

	userID, role, err := pkgjwt.ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000002"), userID)
	assert.Equal(t, string(model.RoleUser), role)
}

func TestDummyLogin_InvalidRole(t *testing.T) {
	svc := newService(&repoMock{})

	_, err := svc.DummyLogin(context.Background(), model.Role("superadmin"))
	require.Error(t, err)
}

// Register
func TestRegister_PasswordIsHashed(t *testing.T) {
	const rawPassword = "password123"

	repo := &repoMock{
		createUser: func(_ context.Context, _, passwordHash string, _ model.Role) (model.User, error) {
			// В репо должен прийти хеш, не открытый пароль
			assert.NotEqual(t, rawPassword, passwordHash)
			err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(rawPassword))
			assert.NoError(t, err, "переданный хеш должен соответствовать исходному паролю")
			return model.User{ID: uuid.New(), Email: "u@example.com", Role: model.RoleUser}, nil
		},
	}

	svc := newService(repo)
	_, err := svc.Register(context.Background(), "u@example.com", rawPassword, model.RoleUser)
	require.NoError(t, err)
}

func TestRegister_EmailTaken(t *testing.T) {
	repo := &repoMock{
		createUser: func(_ context.Context, _, _ string, _ model.Role) (model.User, error) {
			return model.User{}, model.ErrEmailTaken
		},
	}

	svc := newService(repo)
	_, err := svc.Register(context.Background(), "taken@example.com", "pass", model.RoleUser)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrEmailTaken))
}

// Login
func TestLogin_Success(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	wantID := uuid.New()
	hashStr := string(hash)
	repo := &repoMock{
		getUserByEmail: func(_ context.Context, _ string) (model.User, error) {
			return model.User{ID: wantID, Email: "u@example.com", Password: &hashStr, Role: model.RoleUser}, nil
		},
	}

	svc := newService(repo)
	token, err := svc.Login(context.Background(), "u@example.com", "secret")
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
	repo := &repoMock{
		getUserByEmail: func(_ context.Context, _ string) (model.User, error) {
			return model.User{ID: uuid.New(), Password: &hashStr, Role: model.RoleUser}, nil
		},
	}

	svc := newService(repo)
	_, err = svc.Login(context.Background(), "u@example.com", "wrong")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidCredentials))
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &repoMock{
		getUserByEmail: func(_ context.Context, _ string) (model.User, error) {
			return model.User{}, model.ErrInvalidCredentials
		},
	}

	svc := newService(repo)
	_, err := svc.Login(context.Background(), "nobody@example.com", "pass")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidCredentials))
}

func TestLogin_DummyUserHasNoPassword(t *testing.T) {
	repo := &repoMock{
		getUserByEmail: func(_ context.Context, _ string) (model.User, error) {
			// Дамми-пользователь создаётся без пароля (password = NULL)
			return model.User{ID: uuid.New(), Email: "admin@test.com", Password: nil, Role: model.RoleAdmin}, nil
		},
	}

	svc := newService(repo)
	_, err := svc.Login(context.Background(), "admin@test.com", "anything")
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidCredentials))
}
