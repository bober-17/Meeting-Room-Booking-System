package room_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/internal/service/room"
)

type repoMock struct {
	createRoom func(ctx context.Context, name string, description *string, capacity *int) (model.Room, error)
	listRooms  func(ctx context.Context) ([]model.Room, error)
}

func (m *repoMock) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error) {
	return m.createRoom(ctx, name, description, capacity)
}

func (m *repoMock) ListRooms(ctx context.Context) ([]model.Room, error) {
	return m.listRooms(ctx)
}

func newService(repo room.RoomRepository) *room.Service {
	return room.New(repo, slog.New(slog.NewTextHandler(os.Stderr, nil)))
}

// CreateRoom

func TestCreateRoom_Success(t *testing.T) {
	want := model.Room{ID: uuid.New(), Name: "Board Room"}

	svc := newService(&repoMock{
		createRoom: func(_ context.Context, _ string, _ *string, _ *int) (model.Room, error) {
			return want, nil
		},
	})

	got, err := svc.CreateRoom(context.Background(), want.Name, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCreateRoom_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(&repoMock{
		createRoom: func(_ context.Context, _ string, _ *string, _ *int) (model.Room, error) {
			return model.Room{}, repoErr
		},
	})

	_, err := svc.CreateRoom(context.Background(), "Room", nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

// ListRooms

func TestListRooms_ReturnsRooms(t *testing.T) {
	want := []model.Room{
		{ID: uuid.New(), Name: "Room A"},
		{ID: uuid.New(), Name: "Room B"},
	}

	svc := newService(&repoMock{
		listRooms: func(_ context.Context) ([]model.Room, error) {
			return want, nil
		},
	})

	got, err := svc.ListRooms(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListRooms_Empty(t *testing.T) {
	svc := newService(&repoMock{
		listRooms: func(_ context.Context) ([]model.Room, error) {
			return nil, nil
		},
	})

	got, err := svc.ListRooms(context.Background())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListRooms_RepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")

	svc := newService(&repoMock{
		listRooms: func(_ context.Context) ([]model.Room, error) {
			return nil, repoErr
		},
	})

	_, err := svc.ListRooms(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}
