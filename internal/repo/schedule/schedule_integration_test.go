//go:build integration

package schedule_test

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	reposched "github.com/internships-backend/test-backend-bober-17/internal/repo/schedule"
	"github.com/internships-backend/test-backend-bober-17/internal/repo/testutil"
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

func TestCreateSchedule_TimeRoundTrip(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room A")
	require.NoError(t, err)

	repo := reposched.New(pool)

	got, err := repo.CreateSchedule(ctx, roomID, []int{1, 3, 5}, "09:00", "17:30")
	require.NoError(t, err)

	assert.Equal(t, "09:00", got.StartTime)
	assert.Equal(t, "17:30", got.EndTime)
	assert.Equal(t, []int{1, 3, 5}, got.DaysOfWeek)
	assert.Equal(t, roomID, got.RoomID)
}

func TestCreateSchedule_ScheduleExists(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	roomID, err := testutil.InsertRoom(ctx, pool, "Room B")
	require.NoError(t, err)

	repo := reposched.New(pool)

	_, err = repo.CreateSchedule(ctx, roomID, []int{1}, "09:00", "17:00")
	require.NoError(t, err)

	_, err = repo.CreateSchedule(ctx, roomID, []int{2}, "08:00", "18:00")
	assert.True(t, errors.Is(err, model.ErrScheduleExists), "got: %v", err)
}
