package schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error) {
	const q = `
		INSERT INTO schedules (room_id, days_of_week, start_time, end_time)
		VALUES ($1, $2, $3::time, $4::time)
		RETURNING id, room_id, days_of_week,
		          TO_CHAR(start_time, 'HH24:MI'),
		          TO_CHAR(end_time,   'HH24:MI'),
		          created_at`

	days := make([]int32, len(daysOfWeek))
	for i, d := range daysOfWeek {
		days[i] = int32(d)
	}

	var s model.Schedule
	var daysRaw []int32

	err := r.pool.QueryRow(ctx, q, roomID, days, startTime, endTime).Scan(
		&s.ID, &s.RoomID, &daysRaw, &s.StartTime, &s.EndTime, &s.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.ForeignKeyViolation:
				return model.Schedule{}, fmt.Errorf("create schedule: %w", model.ErrRoomNotFound)
			case pgerrcode.UniqueViolation:
				return model.Schedule{}, fmt.Errorf("create schedule: %w", model.ErrScheduleExists)
			}
		}
		return model.Schedule{}, fmt.Errorf("create schedule: %w", err)
	}

	s.DaysOfWeek = make([]int, len(daysRaw))
	for i, d := range daysRaw {
		s.DaysOfWeek[i] = int(d)
	}

	return s, nil
}
