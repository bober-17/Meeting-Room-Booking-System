package room

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type Service struct {
	repo RoomRepository
	log  *slog.Logger
}

func New(repo RoomRepository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error) {
	room, err := s.repo.CreateRoom(ctx, name, description, capacity)
	if err != nil {
		return model.Room{}, fmt.Errorf("room service: %w", err)
	}

	return room, nil
}

func (s *Service) ListRooms(ctx context.Context) ([]model.Room, error) {
	rooms, err := s.repo.ListRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("room service: %w", err)
	}

	return rooms, nil
}
