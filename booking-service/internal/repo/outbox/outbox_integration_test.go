//go:build integration

package outbox_test

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repooutbox "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/outbox"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/testutil"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	pool, err = testutil.NewPool(ctx)
	if err != nil {
		log.Fatalf("setup test pool: %v", err)
	}

	if err := testutil.MigrateUp("../../../migrations"); err != nil {
		pool.Close()
		log.Fatalf("migrate up: %v", err)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func insertPendingRecord(ctx context.Context, t *testing.T, eventType string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO outbox (id, event_type, payload) VALUES ($1, $2, $3)`,
		id, eventType, `{"booking_id":"test"}`,
	)
	require.NoError(t, err)
	return id
}

func countPending(ctx context.Context, t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE sent_at IS NULL`).Scan(&n))
	return n
}

func countSent(ctx context.Context, t *testing.T) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE sent_at IS NOT NULL`).Scan(&n))
	return n
}

func TestProcessBatch_HappyPath_MarksRecordsSent(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	insertPendingRecord(ctx, t, "booking.created")
	insertPendingRecord(ctx, t, "booking.created")

	repo := repooutbox.New(pool)

	var received []repooutbox.Record
	err := repo.ProcessBatch(ctx, func(records []repooutbox.Record) error {
		received = records
		return nil
	})
	require.NoError(t, err)

	assert.Len(t, received, 2)
	assert.Equal(t, 0, countPending(ctx, t))
	assert.Equal(t, 2, countSent(ctx, t))
}

func TestProcessBatch_PublishError_RecordsRemainPending(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	insertPendingRecord(ctx, t, "booking.created")

	repo := repooutbox.New(pool)

	publishErr := errors.New("kafka unavailable")
	err := repo.ProcessBatch(ctx, func(records []repooutbox.Record) error {
		return publishErr
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, publishErr)

	assert.Equal(t, 1, countPending(ctx, t))
	assert.Equal(t, 0, countSent(ctx, t))
}

func TestProcessBatch_EmptyBatch_PublishNotCalled(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := repooutbox.New(pool)

	publishCalled := false
	err := repo.ProcessBatch(ctx, func(records []repooutbox.Record) error {
		publishCalled = true
		return nil
	})
	require.NoError(t, err)
	assert.False(t, publishCalled)
}

func TestProcessBatch_AlreadySent_NotReprocessed(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	id := insertPendingRecord(ctx, t, "booking.created")

	_, err := pool.Exec(ctx, `UPDATE outbox SET sent_at = NOW() WHERE id = $1`, id)
	require.NoError(t, err)

	repo := repooutbox.New(pool)

	publishCalled := false
	err = repo.ProcessBatch(ctx, func(records []repooutbox.Record) error {
		publishCalled = true
		return nil
	})
	require.NoError(t, err)
	assert.False(t, publishCalled)
}
