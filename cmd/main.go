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
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/internships-backend/test-backend-bober-17/internal/config"
	"github.com/internships-backend/test-backend-bober-17/internal/http/handlers"
	"github.com/internships-backend/test-backend-bober-17/internal/postgres"
	authrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/auth"
	roomrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/room"
	schedulerepo "github.com/internships-backend/test-backend-bober-17/internal/repo/schedule"
	slotrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/slot"
	authservice "github.com/internships-backend/test-backend-bober-17/internal/service/auth"
	roomservice "github.com/internships-backend/test-backend-bober-17/internal/service/room"
	scheduleservice "github.com/internships-backend/test-backend-bober-17/internal/service/schedule"
	slotservice "github.com/internships-backend/test-backend-bober-17/internal/service/slot"
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

	// Репозитории
	authRepo := authrepo.New(pool)
	roomRepo := roomrepo.New(pool)
	scheduleRepo := schedulerepo.New(pool)
	slotRepo := slotrepo.New(pool)

	// Сервисы
	authSvc := authservice.New(authRepo, cfg.JWTSecret, logger)
	roomSvc := roomservice.New(roomRepo, logger)
	scheduleSvc := scheduleservice.New(scheduleRepo, logger)
	slotSvc := slotservice.New(roomRepo, scheduleRepo, slotRepo, logger)

	// TODO: инициализировать booking репозиторий и сервис
	// TODO: инициализировать ConferenceClient

	router := handlers.NewRouter(cfg.JWTSecret, authSvc, roomSvc, scheduleSvc, slotSvc)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server: %w", err)
	case <-quit:
		logger.Info("shutting down server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
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
