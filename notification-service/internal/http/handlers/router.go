package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/middleware"
)

const requestTimeout = 4 * time.Second

func NewRouter(jwtSecret string, hub sseHub, tokens tokenStore, svc notificationService) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Metrics)

	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	sseH := newSSEHandler(hub, tokens)
	notifH := &notificationHandler{svc: svc}

	// SSE stream — без timeout и Recoverer: соединение долгоживущее,
	// заголовки committed после первого flush, Recoverer запишет мусор в поток.
	// Панику в SSE-хендлере перехватывает встроенный recover Go (закрывает соединение).
	r.Get("/_info", infoHandler)
	r.Get("/notifications/stream", sseH.streamNotifications)

	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.Recoverer)
		r.Use(chimiddleware.Timeout(requestTimeout))
		r.Use(middleware.Auth(jwtSecret))

		r.Post("/sse-token", sseH.issueToken)
		r.Get("/notifications", notifH.list)
		r.Post("/notifications/{id}/read", notifH.markRead)
		r.Post("/notifications/read-all", notifH.markAllRead)
	})

	return r
}
