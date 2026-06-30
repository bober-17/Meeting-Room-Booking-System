package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/client/conference"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/config"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/http/handlers"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/outbox"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/postgres"
	authrepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/auth"
	bookingrepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/booking"
	outboxrepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/outbox"
	roomrepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/room"
	schedulerepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/schedule"
	slotrepo "github.com/bober-17/meeting-room-booking-system/booking-service/internal/repo/slot"
	authservice "github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/auth"
	bookingservice "github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/booking"
	roomservice "github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/room"
	scheduleservice "github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/schedule"
	slotservice "github.com/bober-17/meeting-room-booking-system/booking-service/internal/service/slot"
)

const (
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 5 * time.Second
	serverIdleTimeout  = 60 * time.Second
	shutdownTimeout    = 5 * time.Second
	migrationsPath     = "file:///migrations"
	pprofAddr          = ":6060"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	pool, err := postgres.NewPool(context.Background(), cfg.DSN())
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	if err := runMigrations(migrationsPath, cfg.MigrateDSN()); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("migrations applied")

	authRepo := authrepo.New(pool)
	roomRepo := roomrepo.New(pool)
	scheduleRepo := schedulerepo.New(pool)
	slotRepo := slotrepo.New(pool)
	bookingRepo := bookingrepo.New(pool)
	outboxRepo := outboxrepo.New(pool)

	confClient := conference.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	relay := outbox.NewRelay(outboxRepo, brokers, cfg.KafkaTopicBookingEvents, logger)

	relayDone := make(chan struct{})
	go func() {
		relay.Run(ctx)
		close(relayDone)
	}()

	authSvc := authservice.New(authRepo, cfg.JWTSecret, logger)
	roomSvc := roomservice.New(roomRepo, logger)
	scheduleSvc := scheduleservice.New(scheduleRepo, logger)
	slotSvc := slotservice.New(roomRepo, scheduleRepo, slotRepo, logger)
	bookingSvc := bookingservice.New(bookingRepo, slotRepo, confClient, logger)

	router := handlers.NewRouter(cfg.JWTSecret, authSvc, roomSvc, scheduleSvc, slotSvc, bookingSvc)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	go func() {
		logger.Info("pprof starting", "addr", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			logger.Error("pprof server error", "err", err)
		}
	}()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	// HTTP останавливаем первым — после этого новых записей в outbox не будет
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	logger.Info("server stopped")

	// relay дренирует уже без риска новых записей; должен завершиться до pool.Close() (defer выше)
	select {
	case <-relayDone:
		logger.Info("outbox relay drained")
	case <-time.After(shutdownTimeout):
		logger.Warn("outbox relay drain timeout, pending records will retry on next start")
	}

	return nil
}

func runMigrations(path, dsn string) error {
	m, err := migrate.New(path, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.Close()
		return fmt.Errorf("migrate up: %w", err)
	}

	srcErr, dbErr := m.Close()
	if srcErr != nil {
		slog.Error("closing migrator source", "err", srcErr)
	}
	if dbErr != nil {
		slog.Error("closing migrator db", "err", dbErr)
	}

	return nil
}
