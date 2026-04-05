package room

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error) {
	const q = `
		INSERT INTO rooms (name, description, capacity)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, capacity, created_at`

	var room model.Room
	err := r.pool.QueryRow(ctx, q, name, description, capacity).Scan(
		&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt,
	)
	if err != nil {
		return model.Room{}, fmt.Errorf("create room: %w", err)
	}

	return room, nil
}

func (r *Repo) RoomExists(ctx context.Context, roomID uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, roomID).Scan(&exists); err != nil {
		return false, fmt.Errorf("room exists: %w", err)
	}

	return exists, nil
}

func (r *Repo) ListRooms(ctx context.Context) ([]model.Room, error) {
	const q = `SELECT id, name, description, capacity, created_at FROM rooms ORDER BY created_at`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		var room model.Room
		if err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity, &room.CreatedAt); err != nil {
			return nil, fmt.Errorf("list rooms scan: %w", err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list rooms rows: %w", err)
	}

	return rooms, nil
}
