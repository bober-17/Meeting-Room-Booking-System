package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
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

func respondError(w http.ResponseWriter, err error) { //nolint:unused
	code, message, status := mapError(err)

	if status == http.StatusInternalServerError {
		slog.Error("internal error", "err", err)
	}

	respondJSON(w, status, errorResponse{
		Error: errorBody{Code: code, Message: message},
	})
}

func mapError(err error) (code, message string, status int) {
	switch {
	case errors.Is(err, model.ErrRoomNotFound):
		return "ROOM_NOT_FOUND", model.ErrRoomNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrSlotNotFound):
		return "SLOT_NOT_FOUND", model.ErrSlotNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrBookingNotFound):
		return "BOOKING_NOT_FOUND", model.ErrBookingNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrSlotAlreadyBooked):
		return "SLOT_ALREADY_BOOKED", model.ErrSlotAlreadyBooked.Error(), http.StatusConflict
	case errors.Is(err, model.ErrScheduleExists):
		return "SCHEDULE_EXISTS", model.ErrScheduleExists.Error(), http.StatusConflict
	case errors.Is(err, model.ErrForbidden):
		return "FORBIDDEN", model.ErrForbidden.Error(), http.StatusForbidden
	case errors.Is(err, model.ErrSlotInPast):
		return "INVALID_REQUEST", model.ErrSlotInPast.Error(), http.StatusBadRequest
	case errors.Is(err, model.ErrEmailTaken):
		return "INVALID_REQUEST", model.ErrEmailTaken.Error(), http.StatusBadRequest
	case errors.Is(err, model.ErrInvalidCredentials):
		return "UNAUTHORIZED", model.ErrInvalidCredentials.Error(), http.StatusUnauthorized
	default:
		return "INTERNAL_ERROR", "internal server error", http.StatusInternalServerError
	}
}
