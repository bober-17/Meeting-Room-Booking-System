package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

const (
	timeParseLayout = "15:04"
	minDayOfWeek    = 1
	maxDayOfWeek    = 7
	minSlotDuration = 30 * time.Minute
)

type ScheduleService interface {
	CreateSchedule(ctx context.Context, roomID uuid.UUID, daysOfWeek []int, startTime, endTime string) (model.Schedule, error)
}

type scheduleHandler struct {
	svc ScheduleService
}

func newScheduleHandler(svc ScheduleService) *scheduleHandler {
	return &scheduleHandler{svc: svc}
}

type createScheduleRequest struct {
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type scheduleResponse struct {
	ID         string `json:"id"`
	RoomID     string `json:"roomId"`
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type createScheduleResponse struct {
	Schedule scheduleResponse `json:"schedule"`
}

func toScheduleResponse(s model.Schedule) scheduleResponse {
	return scheduleResponse{
		ID:         s.ID.String(),
		RoomID:     s.RoomID.String(),
		DaysOfWeek: s.DaysOfWeek,
		StartTime:  s.StartTime,
		EndTime:    s.EndTime,
	}
}

func (h *scheduleHandler) createSchedule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid roomId"},
		})
		return
	}

	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	if len(req.DaysOfWeek) == 0 {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "daysOfWeek is required"},
		})
		return
	}

	seen := make(map[int]struct{}, len(req.DaysOfWeek))
	for _, d := range req.DaysOfWeek {
		if d < minDayOfWeek || d > maxDayOfWeek {
			respondJSON(w, http.StatusBadRequest, errorResponse{
				Error: errorBody{Code: codeInvalidRequest, Message: "daysOfWeek values must be between 1 and 7"},
			})
			return
		}
		if _, ok := seen[d]; ok {
			respondJSON(w, http.StatusBadRequest, errorResponse{
				Error: errorBody{Code: codeInvalidRequest, Message: "daysOfWeek contains duplicate values"},
			})
			return
		}
		seen[d] = struct{}{}
	}

	startT, err := time.Parse(timeParseLayout, req.StartTime)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "startTime must be in HH:MM format"},
		})
		return
	}

	endT, err := time.Parse(timeParseLayout, req.EndTime)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "endTime must be in HH:MM format"},
		})
		return
	}

	if !endT.After(startT) {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "endTime must be after startTime"},
		})
		return
	}

	if endT.Sub(startT) < minSlotDuration {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "time window must be at least 30 minutes"},
		})
		return
	}

	schedule, err := h.svc.CreateSchedule(r.Context(), roomID, req.DaysOfWeek, req.StartTime, req.EndTime)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, createScheduleResponse{Schedule: toScheduleResponse(schedule)})
}
