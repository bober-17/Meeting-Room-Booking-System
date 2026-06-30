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

var (
	AdminUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	UserUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

func PoolDSN() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "host=localhost port=5433 user=postgres password=postgres dbname=booking_test sslmode=disable"
}

func MigrateDSN() string {
	if v := os.Getenv("TEST_MIGRATE_URL"); v != "" {
		return v
	}
	return "pgx5://postgres:postgres@localhost:5433/booking_test?sslmode=disable"
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

// TruncateTables очищает данные между тестами. Пользователи (seeded migration) не затрагиваются.
func TruncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE bookings, slots, schedules, rooms, outbox CASCADE`)
	return err
}

// CountOutboxByEventType возвращает количество необработанных записей в outbox с данным типом события.
func CountOutboxByEventType(ctx context.Context, pool *pgxpool.Pool, eventType string) (int, error) {
	var count int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox WHERE event_type = $1 AND sent_at IS NULL`,
		eventType,
	).Scan(&count)
	return count, err
}

// OutboxBookingID возвращает booking_id из payload первой необработанной записи с данным типом события.
func OutboxBookingID(ctx context.Context, pool *pgxpool.Pool, eventType string) (string, error) {
	var bookingID string
	err := pool.QueryRow(ctx,
		`SELECT payload->>'booking_id' FROM outbox WHERE event_type = $1 AND sent_at IS NULL ORDER BY created_at ASC LIMIT 1`,
		eventType,
	).Scan(&bookingID)
	return bookingID, err
}

func InsertRoom(ctx context.Context, pool *pgxpool.Pool, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO rooms (name) VALUES ($1) RETURNING id`, name,
	).Scan(&id)
	return id, err
}

func InsertSlot(ctx context.Context, pool *pgxpool.Pool, roomID uuid.UUID, start, end time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO slots (room_id, start_at, end_at) VALUES ($1, $2, $3) RETURNING id`,
		roomID, start, end,
	).Scan(&id)
	return id, err
}

func InsertBooking(ctx context.Context, pool *pgxpool.Pool, slotID, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO bookings (slot_id, user_id) VALUES ($1, $2) RETURNING id`,
		slotID, userID,
	).Scan(&id)
	return id, err
}

func CancelBookingDirect(ctx context.Context, pool *pgxpool.Pool, bookingID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE bookings SET status = 'cancelled' WHERE id = $1`, bookingID)
	return err
}
