package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/middleware"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/metrics"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

const (
	heartbeatInterval = 15 * time.Second
	sseWriteTimeout   = 30 * time.Second
)

type sseHub interface {
	Subscribe(userID string) (<-chan model.Notification, func(), error)
}

type tokenStore interface {
	Issue(userID string) string
	Consume(token string) (string, bool)
}

type sseHandler struct {
	hub    sseHub
	tokens tokenStore
}

func newSSEHandler(hub sseHub, tokens tokenStore) *sseHandler {
	return &sseHandler{hub: hub, tokens: tokens}
}

type issueTokenResponse struct {
	Token string `json:"token"`
}

func (h *sseHandler) issueToken(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	token := h.tokens.Issue(userID.String())
	respondJSON(w, http.StatusOK, issueTokenResponse{Token: token})
}

func (h *sseHandler) streamNotifications(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusUnauthorized, codeUnauthorized, "token is required")
		return
	}

	userID, ok := h.tokens.Consume(token)
	if !ok {
		respondError(w, http.StatusUnauthorized, codeUnauthorized, "invalid or expired token")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, codeInternalError, "streaming not supported")
		return
	}

	// Снимаем WriteTimeout сервера — SSE-соединение долгоживущее, 5s дедлайн его убьёт.
	// Вместо него ставим per-write дедлайн перед каждой записью чтобы избежать зомби-горутин
	// когда клиент завис без TCP RST (NAT/proxy) и TCP receive window обнуляется.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		respondError(w, http.StatusInternalServerError, codeInternalError, "failed to configure stream")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, unsubscribe, err := h.hub.Subscribe(userID)
	if err != nil {
		respondError(w, http.StatusTooManyRequests, codeTooManyConnections, "too many active SSE connections")
		return
	}
	defer unsubscribe()
	metrics.SSEConnectionsActive.Inc()
	defer metrics.SSEConnectionsActive.Dec()

	_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {}\n\n"); err != nil {
		return
	}
	flusher.Flush()
	_ = rc.SetWriteDeadline(time.Time{})

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case n := <-ch:
			data, err := json.Marshal(n)
			if err != nil {
				return
			}
			_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
			if _, err := fmt.Fprintf(w, "event: notification\ndata: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
			_ = rc.SetWriteDeadline(time.Time{})

		case <-ticker.C:
			_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
			_ = rc.SetWriteDeadline(time.Time{})

		case <-r.Context().Done():
			return
		}
	}
}
