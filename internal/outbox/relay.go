package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	outboxrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/outbox"
)

const (
	pollInterval       = 2 * time.Second
	writerBatchTimeout = 100 * time.Millisecond
	drainTimeout       = 10 * time.Second
)

type Relay struct {
	repo   *outboxrepo.Repo
	writer *kafka.Writer
	logger *slog.Logger
}

func NewRelay(repo *outboxrepo.Repo, brokers []string, topic string, logger *slog.Logger) *Relay {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{}, // все события одной брони в один partition
		BatchTimeout:           writerBatchTimeout,
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
	}
	return &Relay{repo: repo, writer: writer, logger: logger}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	defer r.writer.Close()

	for {
		select {
		case <-ctx.Done():
			drainCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
			r.processBatch(drainCtx)
			cancel()
			r.logger.Info("outbox relay stopped")
			return
		case <-ticker.C:
			r.processBatch(ctx)
		}
	}
}

func (r *Relay) processBatch(ctx context.Context) {
	var published int
	err := r.repo.ProcessBatch(ctx, func(records []outboxrepo.Record) error {
		msgs := make([]kafka.Message, 0, len(records))
		for _, rec := range records {
			key := bookingIDFromPayload(rec.Payload)
			if key == "" {
				r.logger.Warn("outbox: missing booking_id in payload, message will be unkeyed", "outbox_id", rec.ID)
			}
			msgs = append(msgs, kafka.Message{Key: []byte(key), Value: rec.Payload})
		}
		published = len(msgs)
		return r.writer.WriteMessages(ctx, msgs...)
	})
	if err != nil {
		r.logger.Error("outbox process batch", "err", err)
		return
	}
	if published > 0 {
		r.logger.Info("outbox relay published", "count", published)
	}
}

func bookingIDFromPayload(payload []byte) string {
	var v struct {
		BookingID string `json:"booking_id"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return ""
	}
	return v.BookingID
}
