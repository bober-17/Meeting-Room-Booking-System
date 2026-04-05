//go:build integration

package booking_test

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repobooking "github.com/internships-backend/test-backend-bober-17/internal/repo/booking"
	"github.com/internships-backend/test-backend-bober-17/internal/repo/testutil"
	"github.com/internships-backend/test-backend-bober-17/internal/model"
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

func TestCreateBooking_UniqueViolation_ReturnsSlotAlreadyBooked(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room A")
	require.NoError(t, err)

	future := time.Now().UTC().Add(time.Hour)
	slotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	repo := repobooking.New(pool)

	_, err = repo.CreateBooking(ctx, slotID, testutil.UserUserID)
	require.NoError(t, err)

	_, err = repo.CreateBooking(ctx, slotID, testutil.AdminUserID)
	assert.True(t, errors.Is(err, model.ErrSlotAlreadyBooked), "got: %v", err)
}

func TestCreateBooking_SlotNotFound(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := repobooking.New(pool)

	_, err := repo.CreateBooking(ctx, uuid.New(), testutil.UserUserID)
	assert.True(t, errors.Is(err, model.ErrSlotNotFound), "got: %v", err)
}

func TestCancelBooking_Idempotent(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room B")
	require.NoError(t, err)

	future := time.Now().UTC().Add(time.Hour)
	slotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	repo := repobooking.New(pool)

	booking, err := repo.CreateBooking(ctx, slotID, testutil.UserUserID)
	require.NoError(t, err)

	first, err := repo.CancelBooking(ctx, booking.ID, testutil.UserUserID)
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusCancelled, first.Status)

	second, err := repo.CancelBooking(ctx, booking.ID, testutil.UserUserID)
	require.NoError(t, err)
	assert.Equal(t, model.BookingStatusCancelled, second.Status)
	assert.Equal(t, first.ID, second.ID)
}

func TestCancelBooking_Forbidden(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room C")
	require.NoError(t, err)

	future := time.Now().UTC().Add(time.Hour)
	slotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	repo := repobooking.New(pool)

	booking, err := repo.CreateBooking(ctx, slotID, testutil.UserUserID)
	require.NoError(t, err)

	_, err = repo.CancelBooking(ctx, booking.ID, testutil.AdminUserID)
	assert.True(t, errors.Is(err, model.ErrForbidden), "got: %v", err)
}

func TestListUserBookings_OnlyFutureActive(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room D")
	require.NoError(t, err)

	past := time.Now().UTC().Add(-2 * time.Hour)
	future := time.Now().UTC().Add(2 * time.Hour)

	pastSlotID, err := testutil.InsertSlot(ctx, pool, roomID, past, past.Add(30*time.Minute))
	require.NoError(t, err)

	futureSlotID, err := testutil.InsertSlot(ctx, pool, roomID, future, future.Add(30*time.Minute))
	require.NoError(t, err)

	cancelledSlotID, err := testutil.InsertSlot(ctx, pool, roomID, future.Add(time.Hour), future.Add(90*time.Minute))
	require.NoError(t, err)

	repo := repobooking.New(pool)

	_, err = repo.CreateBooking(ctx, pastSlotID, testutil.UserUserID)
	require.NoError(t, err)

	_, err = repo.CreateBooking(ctx, futureSlotID, testutil.UserUserID)
	require.NoError(t, err)

	cancelledBooking, err := repo.CreateBooking(ctx, cancelledSlotID, testutil.UserUserID)
	require.NoError(t, err)
	_, err = repo.CancelBooking(ctx, cancelledBooking.ID, testutil.UserUserID)
	require.NoError(t, err)

	bookings, err := repo.ListUserBookings(ctx, testutil.UserUserID)
	require.NoError(t, err)

	require.Len(t, bookings, 1)
	assert.Equal(t, futureSlotID, bookings[0].SlotID)
	assert.Equal(t, model.BookingStatusActive, bookings[0].Status)
}
