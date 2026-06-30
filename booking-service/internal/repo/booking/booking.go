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

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) CreateBooking(ctx context.Context, slotID, userID uuid.UUID) (model.Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Booking{}, fmt.Errorf("create booking begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var b model.Booking
	err = tx.QueryRow(ctx, `
		INSERT INTO bookings (slot_id, user_id)
		VALUES ($1, $2)
		RETURNING id, slot_id, user_id, status, conference_link, created_at`,
		slotID, userID,
	).Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt)
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

	outboxID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (id, event_type, payload)
		SELECT
			$4,
			'booking.created',
			jsonb_build_object(
				'event_id',    $5::text,
				'event_type',  'booking.created',
				'occurred_at', NOW(),
				'booking_id',  $1::text,
				'user_id',     $2::text,
				'room_id',     r.id::text,
				'room_name',   r.name,
				'slot_start',  s.start_at,
				'slot_end',    s.end_at
			)
		FROM slots s
		JOIN rooms r ON r.id = s.room_id
		WHERE s.id = $3`,
		b.ID, b.UserID, slotID, outboxID, outboxID.String(),
	)
	if err != nil {
		return model.Booking{}, fmt.Errorf("create booking outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Booking{}, fmt.Errorf("create booking commit: %w", err)
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

func (r *Repo) CancelBooking(ctx context.Context, id, userID uuid.UUID) (model.Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Booking{}, fmt.Errorf("cancel booking begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var b model.Booking
	err = tx.QueryRow(ctx, `
		UPDATE bookings SET status = 'cancelled'
		WHERE id = $1 AND status = 'active' AND user_id = $2
		RETURNING id, slot_id, user_id, status, conference_link, created_at`,
		id, userID,
	).Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.ConferenceLink, &b.CreatedAt)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, fmt.Errorf("cancel booking: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return r.diagnoseCancelFailure(ctx, id, userID)
	}

	outboxID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox (id, event_type, payload)
		SELECT
			$4,
			'booking.cancelled',
			jsonb_build_object(
				'event_id',    $5::text,
				'event_type',  'booking.cancelled',
				'occurred_at', NOW(),
				'booking_id',  $1::text,
				'user_id',     $2::text,
				'room_id',     r.id::text,
				'room_name',   r.name,
				'slot_start',  s.start_at,
				'slot_end',    s.end_at
			)
		FROM slots s
		JOIN rooms r ON r.id = s.room_id
		WHERE s.id = $3`,
		b.ID, b.UserID, b.SlotID, outboxID, outboxID.String(),
	)
	if err != nil {
		return model.Booking{}, fmt.Errorf("cancel booking outbox: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Booking{}, fmt.Errorf("cancel booking commit: %w", err)
	}

	return b, nil
}

func (r *Repo) diagnoseCancelFailure(ctx context.Context, id, userID uuid.UUID) (model.Booking, error) {
	existing, err := r.GetBookingByID(ctx, id)
	if err != nil {
		return model.Booking{}, err
	}

	if existing.UserID != userID {
		return model.Booking{}, fmt.Errorf("cancel booking: %w", model.ErrForbidden)
	}

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
