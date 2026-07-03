package notification

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

type Repo struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, n model.Notification) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, booking_id, room_name, slot_start, slot_end, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (booking_id, type) DO NOTHING`,
		n.ID, n.UserID, n.Type, n.BookingID, n.RoomName, n.SlotStart, n.SlotEnd, n.IsRead, n.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("create notification: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) ListByUserID(ctx context.Context, userID string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error) {
	filter := `WHERE user_id = $1`
	if unreadOnly {
		filter += ` AND is_read = false`
	}

	// COUNT(*) OVER() в одном запросе — нет race condition между отдельными COUNT и SELECT.
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, booking_id, room_name, slot_start, slot_end, is_read, created_at,
		        COUNT(*) OVER() AS total_count
		FROM notifications `+filter+`
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	ns := make([]model.Notification, 0)
	var total int
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.BookingID,
			&n.RoomName, &n.SlotStart, &n.SlotEnd, &n.IsRead, &n.CreatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("list notifications scan: %w", err)
		}
		ns = append(ns, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list notifications rows: %w", err)
	}

	return ns, total, nil
}

func (r *Repo) MarkAsRead(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true
		WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark as read: %w", model.ErrNotificationNotFound)
	}
	return nil
}

func (r *Repo) MarkAllAsRead(ctx context.Context, userID string) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true
		WHERE user_id = $1 AND is_read = false`,
		userID,
	); err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}
