package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

// maxBodyBytes — лимит тела запроса для всех хендлеров, защита от DoS.
const maxBodyBytes = 1 << 20 // 1 MB

// Коды ошибок из api.yaml.
const (
	codeInvalidRequest    = "INVALID_REQUEST"
	codeUnauthorized      = "UNAUTHORIZED"
	codeForbidden         = "FORBIDDEN"
	codeRoomNotFound      = "ROOM_NOT_FOUND"
	codeSlotNotFound      = "SLOT_NOT_FOUND"
	codeBookingNotFound   = "BOOKING_NOT_FOUND"
	codeSlotAlreadyBooked = "SLOT_ALREADY_BOOKED"
	codeScheduleExists    = "SCHEDULE_EXISTS"
	codeInternalError     = "INTERNAL_ERROR"
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

func respondError(w http.ResponseWriter, err error) {
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
		return codeRoomNotFound, model.ErrRoomNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrSlotNotFound):
		return codeSlotNotFound, model.ErrSlotNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrBookingNotFound):
		return codeBookingNotFound, model.ErrBookingNotFound.Error(), http.StatusNotFound
	case errors.Is(err, model.ErrSlotAlreadyBooked):
		return codeSlotAlreadyBooked, model.ErrSlotAlreadyBooked.Error(), http.StatusConflict
	case errors.Is(err, model.ErrScheduleExists):
		return codeScheduleExists, model.ErrScheduleExists.Error(), http.StatusConflict
	case errors.Is(err, model.ErrForbidden):
		return codeForbidden, model.ErrForbidden.Error(), http.StatusForbidden
	case errors.Is(err, model.ErrSlotInPast):
		return codeInvalidRequest, model.ErrSlotInPast.Error(), http.StatusBadRequest
	case errors.Is(err, model.ErrEmailTaken):
		return codeInvalidRequest, model.ErrEmailTaken.Error(), http.StatusBadRequest
	case errors.Is(err, model.ErrInvalidCredentials):
		return codeUnauthorized, model.ErrInvalidCredentials.Error(), http.StatusUnauthorized
	default:
		return codeInternalError, "internal server error", http.StatusInternalServerError
	}
}
