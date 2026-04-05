package booking

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
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

func (r *Repo) CreateBooking(ctx context.Context, slotID, userID uuid.UUID) (model.Booking, error) {
	const q = `
		INSERT INTO bookings (slot_id, user_id)
		VALUES ($1, $2)
		RETURNING id, slot_id, user_id, status, conference_link, created_at`

	var b model.Booking
	err := r.pool.QueryRow(ctx, q, slotID, userID).Scan(
		&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				return model.Booking{}, fmt.Errorf("create booking: %w", model.ErrSlotAlreadyBooked)
			case pgerrcode.ForeignKeyViolation:
				return model.Booking{}, fmt.Errorf("create booking: %w", model.ErrSlotNotFound)
			}
		}
		return model.Booking{}, fmt.Errorf("create booking: %w", err)
	}

	return b, nil
}

func (r *Repo) GetBookingByID(ctx context.Context, id uuid.UUID) (model.Booking, error) {
	const q = `
		SELECT id, slot_id, user_id, status, conference_link, created_at
		FROM bookings WHERE id = $1`

	var b model.Booking
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Booking{}, fmt.Errorf("get booking: %w", model.ErrBookingNotFound)
		}
		return model.Booking{}, fmt.Errorf("get booking: %w", err)
	}

	return b, nil
}

// CancelBooking атомарно отменяет бронь через один UPDATE.
// Если UPDATE не затронул строку — делает SELECT для диагностики: 404 / 403 / уже отменена (идемпотентность).
func (r *Repo) CancelBooking(ctx context.Context, id, userID uuid.UUID) (model.Booking, error) {
	const updateQ = `
		UPDATE bookings SET status = 'cancelled'
		WHERE id = $1 AND status = 'active' AND user_id = $2
		RETURNING id, slot_id, user_id, status, conference_link, created_at`

	var b model.Booking
	err := r.pool.QueryRow(ctx, updateQ, id, userID).Scan(
		&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt,
	)
	if err == nil {
		return b, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, fmt.Errorf("cancel booking: %w", err)
	}

	// UPDATE не затронул строку — определяем причину
	existing, err := r.GetBookingByID(ctx, id)
	if err != nil {
		return model.Booking{}, err // уже оборачивает ErrBookingNotFound
	}

	if existing.UserID != userID {
		return model.Booking{}, fmt.Errorf("cancel booking: %w", model.ErrForbidden)
	}

	// Бронь уже отменена — идемпотентность, возвращаем как есть
	return existing, nil
}

func (r *Repo) ListBookings(ctx context.Context, page, pageSize int) ([]model.Booking, int, error) {
	const countQ = `SELECT COUNT(*) FROM bookings`

	var total int
	if err := r.pool.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("list bookings count: %w", err)
	}

	const q = `
		SELECT id, slot_id, user_id, status, conference_link, created_at
		FROM bookings
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, q, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var b model.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("list bookings scan: %w", err)
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list bookings rows: %w", err)
	}

	return bookings, total, nil
}

// ListUserBookings возвращает только активные брони пользователя на будущие слоты (start >= now).
// Логика выбора — DECISIONS.md п.17.
func (r *Repo) ListUserBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error) {
	const q = `
		SELECT b.id, b.slot_id, b.user_id, b.status, b.conference_link, b.created_at
		FROM bookings b
		JOIN slots s ON s.id = b.slot_id
		WHERE b.user_id = $1
		  AND b.status = 'active'
		  AND s.start_at >= NOW()
		ORDER BY s.start_at`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list user bookings: %w", err)
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var b model.Booking
		if err := rows.Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("list user bookings scan: %w", err)
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user bookings rows: %w", err)
	}

	return bookings, nil
}

func (r *Repo) UpdateConferenceLink(ctx context.Context, bookingID uuid.UUID, link string) error {
	const q = `UPDATE bookings SET conference_link = $1 WHERE id = $2`

	if _, err := r.pool.Exec(ctx, q, link, bookingID); err != nil {
		return fmt.Errorf("update conference link: %w", err)
	}

	return nil
}
