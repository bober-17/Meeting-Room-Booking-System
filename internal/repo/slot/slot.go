package slot

import (
	"context"
	"fmt"
	"time"

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

// EnsureSlots вставляет слоты пакетно через unnest().
// ON CONFLICT DO NOTHING гарантирует идемпотентность: повторный вызов для той же даты — no-op.
func (r *Repo) EnsureSlots(ctx context.Context, slots []model.Slot) error {
	if len(slots) == 0 {
		return nil
	}

	const q = `
		INSERT INTO slots (id, room_id, start_at, end_at)
		SELECT * FROM unnest($1::uuid[], $2::uuid[], $3::timestamptz[], $4::timestamptz[])
		ON CONFLICT (room_id, start_at) DO NOTHING`

	ids := make([]string, len(slots))
	roomIDs := make([]string, len(slots))
	starts := make([]time.Time, len(slots))
	ends := make([]time.Time, len(slots))

	for i, s := range slots {
		ids[i] = s.ID.String()
		roomIDs[i] = s.RoomID.String()
		starts[i] = s.StartAt
		ends[i] = s.EndAt
	}

	if _, err := r.pool.Exec(ctx, q, ids, roomIDs, starts, ends); err != nil {
		return fmt.Errorf("ensure slots: %w", err)
	}

	return nil
}

// ListAvailableByRoomAndDate возвращает слоты без активной брони в диапазоне [from, to).
// Использует индекс UNIQUE(room_id, start_at) для range scan.
func (r *Repo) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, from, to time.Time) ([]model.Slot, error) {
	const q = `
		SELECT s.id, s.room_id, s.start_at, s.end_at
		FROM slots s
		LEFT JOIN bookings b ON b.slot_id = s.id AND b.status = 'active'
		WHERE s.room_id = $1
		  AND s.start_at >= $2
		  AND s.start_at < $3
		  AND b.id IS NULL
		ORDER BY s.start_at`

	rows, err := r.pool.Query(ctx, q, roomID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list available slots: %w", err)
	}
	defer rows.Close()

	var result []model.Slot
	for rows.Next() {
		var s model.Slot
		if err := rows.Scan(&s.ID, &s.RoomID, &s.StartAt, &s.EndAt); err != nil {
			return nil, fmt.Errorf("list available slots scan: %w", err)
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list available slots rows: %w", err)
	}

	return result, nil
}
