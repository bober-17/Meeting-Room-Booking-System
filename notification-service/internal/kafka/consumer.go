package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/bober-17/meeting-room-booking-system/shared/events"
)

type NotificationService interface {
	CreateFromEvent(ctx context.Context, event events.BookingEvent) error
}

type Consumer struct {
	reader *kafka.Reader
	logger *slog.Logger
}

func NewConsumer(brokers []string, topic, groupID string, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		logger: logger,
	}
}

func (c *Consumer) Run(ctx context.Context, svc NotificationService) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		var event events.BookingEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.WarnContext(ctx, "invalid message payload, skipping",
				slog.String("topic", msg.Topic),
				slog.Int64("offset", msg.Offset),
				slog.String("error", err.Error()),
			)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("commit after invalid payload: %w", err)
			}
			continue
		}

		if err := svc.CreateFromEvent(ctx, event); err != nil {
			// Не останавливаем consumer на ошибке обработки — иначе одно
			// «плохое» сообщение блокирует весь топик навсегда (poison pill).
			// Коммитим и логируем; потеря уведомления предпочтительнее бесконечной петли.
			c.logger.ErrorContext(ctx, "failed to process event, skipping",
				slog.String("event_id", event.EventID),
				slog.String("error", err.Error()),
			)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("commit after processing error: %w", err)
			}
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
