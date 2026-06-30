//go:build integration

package notification_test

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

	reponotification "github.com/bober-17/meeting-room-booking-system/notification-service/internal/repo/notification"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/repo/testutil"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
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

func TestCreate_Idempotent(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)
	bookingID := uuid.New()
	future := time.Now().UTC().Add(time.Hour)

	n := model.Notification{
		ID:        uuid.New().String(),
		UserID:    testutil.UserID.String(),
		Type:      model.TypeBookingCreated,
		BookingID: bookingID.String(),
		RoomName:  "Room A",
		SlotStart: future,
		SlotEnd:   future.Add(30 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	inserted, err := repo.Create(ctx, n)
	require.NoError(t, err)
	assert.True(t, inserted, "первая вставка должна вернуть true")

	// Симулируем повторную доставку из Kafka: новый ID, те же booking_id + type.
	// ON CONFLICT (booking_id, type) должен проигнорировать дубль.
	n2 := n
	n2.ID = uuid.New().String()
	inserted2, err := repo.Create(ctx, n2)
	require.NoError(t, err, "повторная доставка из Kafka не должна возвращать ошибку")
	assert.False(t, inserted2, "дубль должен вернуть false")

	ns, total, err := repo.ListByUserID(ctx, testutil.UserID.String(), 10, 0, false)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, ns, 1)
}

func TestListByUserID_Pagination(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)
	future := time.Now().UTC().Add(time.Hour)

	for i := 0; i < 5; i++ {
		_, err := testutil.InsertNotification(ctx, pool, testutil.UserID, uuid.New(), string(model.TypeBookingCreated), future.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	ns, total, err := repo.ListByUserID(ctx, testutil.UserID.String(), 3, 0, false)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, ns, 3)

	ns2, _, err := repo.ListByUserID(ctx, testutil.UserID.String(), 3, 3, false)
	require.NoError(t, err)
	assert.Len(t, ns2, 2)
}

func TestMarkAsRead_NotFound(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)

	err := repo.MarkAsRead(ctx, uuid.New().String(), testutil.UserID.String())
	assert.ErrorIs(t, err, model.ErrNotificationNotFound)
}

func TestMarkAsRead_WrongUser_NotFound(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)
	future := time.Now().UTC().Add(time.Hour)

	id, err := testutil.InsertNotification(ctx, pool, testutil.UserID, uuid.New(), string(model.TypeBookingCreated), future)
	require.NoError(t, err)

	otherUser := uuid.New()
	markErr := repo.MarkAsRead(ctx, id.String(), otherUser.String())
	assert.ErrorIs(t, markErr, model.ErrNotificationNotFound)
}

func TestMarkAllAsRead(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)
	future := time.Now().UTC().Add(time.Hour)

	for i := 0; i < 3; i++ {
		_, err := testutil.InsertNotification(ctx, pool, testutil.UserID, uuid.New(), string(model.TypeBookingCreated), future.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	require.NoError(t, repo.MarkAllAsRead(ctx, testutil.UserID.String()))

	ns, _, err := repo.ListByUserID(ctx, testutil.UserID.String(), 10, 0, false)
	require.NoError(t, err)
	for _, n := range ns {
		assert.True(t, n.IsRead)
	}
}

func TestListByUserID_UnreadOnly(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testutil.TruncateTables(ctx, pool))

	repo := reponotification.New(pool)
	future := time.Now().UTC().Add(time.Hour)

	// 3 уведомления, из них 1 прочитано
	ids := make([]uuid.UUID, 3)
	for i := range ids {
		id, err := testutil.InsertNotification(ctx, pool, testutil.UserID, uuid.New(), string(model.TypeBookingCreated), future.Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
		ids[i] = id
	}
	require.NoError(t, repo.MarkAsRead(ctx, ids[0].String(), testutil.UserID.String()))

	ns, total, err := repo.ListByUserID(ctx, testutil.UserID.String(), 10, 0, true)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "total должен считать только непрочитанные")
	assert.Len(t, ns, 2)
	for _, n := range ns {
		assert.False(t, n.IsRead)
	}
}
