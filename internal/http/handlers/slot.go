package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

const dateParseLayout = "2006-01-02"

type SlotService interface {
	ListAvailableSlots(ctx context.Context, roomID uuid.UUID, date time.Time) ([]model.Slot, error)
}

type slotHandler struct {
	svc SlotService
}

func newSlotHandler(svc SlotService) *slotHandler {
	return &slotHandler{svc: svc}
}

type slotResponse struct {
	ID     string `json:"id"`
	RoomID string `json:"roomId"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

type listSlotsResponse struct {
	Slots []slotResponse `json:"slots"`
}

func toSlotResponse(s model.Slot) slotResponse {
	return slotResponse{
		ID:     s.ID.String(),
		RoomID: s.RoomID.String(),
		Start:  s.StartAt.UTC().Format(time.RFC3339),
		End:    s.EndAt.UTC().Format(time.RFC3339),
	}
}

func (h *slotHandler) listSlots(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid roomId"},
		})
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "date query parameter is required"},
		})
		return
	}

	date, err := time.Parse(dateParseLayout, dateStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "date must be in YYYY-MM-DD format"},
		})
		return
	}

	slots, err := h.svc.ListAvailableSlots(r.Context(), roomID, date)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := listSlotsResponse{Slots: make([]slotResponse, len(slots))}
	for i, s := range slots {
		resp.Slots[i] = toSlotResponse(s)
	}

	respondJSON(w, http.StatusOK, resp)
}
