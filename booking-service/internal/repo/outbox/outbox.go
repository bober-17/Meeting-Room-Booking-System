package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const fetchBatchSize = 100

type Record struct {
	ID        string
	EventType string
	Payload   []byte
}

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ProcessBatch выбирает до 100 неотправленных записей внутри транзакции
// (FOR UPDATE SKIP LOCKED), вызывает publish, затем помечает их sent_at.
// Если publish возвращает ошибку — транзакция откатывается, записи будут
// перепрочитаны при следующем poll.
func (r *Repo) ProcessBatch(ctx context.Context, publish func([]Record) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("outbox begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `
		SELECT id, event_type, payload
		FROM outbox
		WHERE sent_at IS NULL
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`,
		fetchBatchSize,
	)
	if err != nil {
		return fmt.Errorf("outbox fetch pending: %w", err)
	}

	var records []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.EventType, &rec.Payload); err != nil {
			rows.Close()
			return fmt.Errorf("outbox scan: %w", err)
		}
		records = append(records, rec)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("outbox rows: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	if err := publish(records); err != nil {
		return err
	}

	ids := make([]string, len(records))
	for i, rec := range records {
		ids[i] = rec.ID
	}

	if _, err := tx.Exec(ctx, `
		UPDATE outbox SET sent_at = $1
		WHERE id = ANY($2::uuid[])`,
		time.Now().UTC(), ids,
	); err != nil {
		return fmt.Errorf("outbox mark sent: %w", err)
	}

	return tx.Commit(ctx)
}
