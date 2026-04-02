package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internships-backend/test-backend-bober-17/internal/config"
	"github.com/internships-backend/test-backend-bober-17/internal/http/handlers"
)

const (
	serverReadTimeout  = 5 * time.Second
	serverWriteTimeout = 5 * time.Second
	serverIdleTimeout  = 60 * time.Second
	shutdownTimeout    = 5 * time.Second
	migrationsPath     = "file:///migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	// Подключение к БД
	pool, err := pgxpool.New(context.Background(), cfg.DSN())
	if err != nil {
		logger.Error("connecting to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("pinging database", "err", err)
		os.Exit(1)
	}

	// Запуск миграций
	m, err := migrate.New(migrationsPath, cfg.MigrateDSN())
	if err != nil {
		logger.Error("creating migrator", "err", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("running migrations", "err", err)
		os.Exit(1)
	}

	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		logger.Error("closing migrator", "srcErr", srcErr, "dbErr", dbErr)
	}

	logger.Info("migrations applied")

	// TODO: инициализировать репозитории (auth, room, schedule, slot, booking)
	// TODO: инициализировать сервисы (auth, room, schedule, slot, booking)
	// TODO: инициализировать ConferenceClient
	// TODO: передать сервисы в handlers.NewRouter

	router := handlers.NewRouter(cfg.JWTSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	go func() {
		logger.Info("server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "err", err)
	}

	logger.Info("server stopped")
}
