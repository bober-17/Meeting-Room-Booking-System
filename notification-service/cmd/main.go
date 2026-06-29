package main

import (
	"log/slog"
	"os"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("starting notification-service", "port", cfg.ServerPort)

	// TODO фаза 3: подключение к БД, миграции
	// TODO фаза 4: Kafka consumer
	// TODO фаза 5: SSE hub, HTTP server
}
