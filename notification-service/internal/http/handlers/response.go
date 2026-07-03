package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

const (
	codeUnauthorized       = "UNAUTHORIZED"
	codeInternalError      = "INTERNAL_ERROR"
	codeTooManyConnections = "TOO_MANY_CONNECTIONS"
	codeNotFound           = "NOT_FOUND"
	codeForbidden          = "FORBIDDEN"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("encoding response", "err", err)
	}
}

func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, model.ErrNotificationNotFound):
		return http.StatusNotFound, codeNotFound
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden, codeForbidden
	default:
		return http.StatusInternalServerError, codeInternalError
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, errorResponse{
		Error: errorBody{Code: code, Message: message},
	})
}
