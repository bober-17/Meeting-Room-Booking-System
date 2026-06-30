package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/config"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/handlers"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/postgres"
	notifRepo "github.com/bober-17/meeting-room-booking-system/notification-service/internal/repo/notification"
	notifService "github.com/bober-17/meeting-room-booking-system/notification-service/internal/service/notification"
)

const (
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 5 * time.Second
	serverIdleTimeout  = 60 * time.Second
	shutdownTimeout    = 5 * time.Second
	migrationsPath     = "file:///migrations"
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

	repo := notifRepo.New(pool)
	svc := notifService.New(repo, nil)

	_ = svc // будет передан в handlers в фазе 6

	router := handlers.NewRouter()

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	logger.Info("server stopped")

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
