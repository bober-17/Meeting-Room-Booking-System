package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/middleware"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	defaultOffset = 0
	maxOffset    = 1_000
)

type notificationService interface {
	ListByUserID(ctx context.Context, userID string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error)
	MarkAsRead(ctx context.Context, id, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
}

type notificationHandler struct {
	svc notificationService
}

type listNotificationsResponse struct {
	Notifications []model.Notification `json:"notifications"`
	Total         int                  `json:"total"`
}

func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context()).String()

	limit := parseIntParam(r.URL.Query().Get("limit"), defaultLimit, 1, maxLimit)
	offset := parseIntParam(r.URL.Query().Get("offset"), defaultOffset, 0, maxOffset)
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	ns, total, err := h.svc.ListByUserID(r.Context(), userID, limit, offset, unreadOnly)
	if err != nil {
		status, code := mapError(err)
		respondError(w, status, code, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, listNotificationsResponse{
		Notifications: ns,
		Total:         total,
	})
}

func (h *notificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.UserIDFromContext(r.Context()).String()

	if err := h.svc.MarkAsRead(r.Context(), id, userID); err != nil {
		status, code := mapError(err)
		respondError(w, status, code, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *notificationHandler) markAllRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context()).String()

	if err := h.svc.MarkAllAsRead(r.Context(), userID); err != nil {
		status, code := mapError(err)
		respondError(w, status, code, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// max <= 0 означает отсутствие верхней границы.
func parseIntParam(s string, def, min, max int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < min {
		return def
	}
	if max > 0 && v > max {
		return max
	}
	return v
}
