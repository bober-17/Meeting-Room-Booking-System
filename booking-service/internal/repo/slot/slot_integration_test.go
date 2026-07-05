//go:build integration

package slot_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	reposlot "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/slot"
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

func TestEnsureSlots_Idempotent(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room A")
	require.NoError(t, err)

	base := time.Now().UTC().Add(time.Hour).Truncate(30 * time.Minute)
	slots := []model.Slot{
		{ID: uuid.New(), RoomID: roomID, StartAt: base, EndAt: base.Add(30 * time.Minute)},
		{ID: uuid.New(), RoomID: roomID, StartAt: base.Add(30 * time.Minute), EndAt: base.Add(time.Hour)},
		{ID: uuid.New(), RoomID: roomID, StartAt: base.Add(time.Hour), EndAt: base.Add(90 * time.Minute)},
	}

	repo := reposlot.New(pool)

	require.NoError(t, repo.EnsureSlots(ctx, slots))
	require.NoError(t, repo.EnsureSlots(ctx, slots))

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM slots WHERE room_id = $1`, roomID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestListAvailable_ExcludesBooked(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room B")
	require.NoError(t, err)

	future := time.Now().UTC().Add(time.Hour)
	slotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	_, err = testutil.InsertBooking(ctx, pool, slotID, testutil.UserUserID)
	require.NoError(t, err)

	repo := reposlot.New(pool)
	from := future.Add(-time.Minute)
	to := future.Add(time.Hour)

	slots, err := repo.ListAvailableByRoomAndDate(ctx, roomID, from, to)
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestListAvailable_CancelledBookingFreesSlot(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room C")
	require.NoError(t, err)

	future := time.Now().UTC().Add(time.Hour)
	slotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	bookingID, err := testutil.InsertBooking(ctx, pool, slotID, testutil.UserUserID)
	require.NoError(t, err)

	require.NoError(t, testutil.CancelBookingDirect(ctx, pool, bookingID))

	repo := reposlot.New(pool)
	from := future.Add(-time.Minute)
	to := future.Add(time.Hour)

	slots, err := repo.ListAvailableByRoomAndDate(ctx, roomID, from, to)
	require.NoError(t, err)
	require.Len(t, slots, 1)
	assert.Equal(t, slotID, slots[0].ID)
}
