//go:build integration

package testutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var UserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func PoolDSN() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "host=localhost port=5435 user=postgres password=postgres dbname=notifications_test sslmode=disable"
}

func MigrateDSN() string {
	if v := os.Getenv("TEST_MIGRATE_URL"); v != "" {
		return v
	}
	return "pgx5://postgres:postgres@localhost:5435/notifications_test?sslmode=disable"
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, PoolDSN())
	if err != nil {
		return nil, fmt.Errorf("create test pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping test db: %w", err)
	}
	return pool, nil
}

func MigrateUp(migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, MigrateDSN())
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.Close()
		return fmt.Errorf("migrate up: %w", err)
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil || dbErr != nil {
		return fmt.Errorf("close migrator: src=%v db=%v", srcErr, dbErr)
	}
	return nil
}

func TruncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE notifications`)
	return err
}

func InsertNotification(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, bookingID uuid.UUID, notifType string, start time.Time) (uuid.UUID, error) {
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, booking_id, room_name, slot_start, slot_end, is_read, created_at)
		VALUES ($1, $2, $3, $4, 'Room A', $5, $6, false, NOW())`,
		id, userID, notifType, bookingID, start, start.Add(30*time.Minute),
	)
	return id, err
}
