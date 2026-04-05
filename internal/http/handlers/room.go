package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

const minRoomCapacity = 1

type RoomService interface {
	CreateRoom(ctx context.Context, name string, description *string, capacity *int) (model.Room, error)
	ListRooms(ctx context.Context) ([]model.Room, error)
}

type roomHandler struct {
	svc RoomService
}

func newRoomHandler(svc RoomService) *roomHandler {
	return &roomHandler{svc: svc}
}

type createRoomRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Capacity    *int    `json:"capacity"`
}

type roomResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Capacity    *int    `json:"capacity"`
	CreatedAt   string  `json:"createdAt"`
}

type createRoomResponse struct {
	Room roomResponse `json:"room"`
}

type listRoomsResponse struct {
	Rooms []roomResponse `json:"rooms"`
}

func toRoomResponse(r model.Room) roomResponse {
	return roomResponse{
		ID:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		Capacity:    r.Capacity,
		CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *roomHandler) createRoom(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	if req.Name == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "name is required"},
		})
		return
	}

	if req.Capacity != nil && *req.Capacity < minRoomCapacity {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "capacity must be at least 1"},
		})
		return
	}

	room, err := h.svc.CreateRoom(r.Context(), req.Name, req.Description, req.Capacity)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, createRoomResponse{Room: toRoomResponse(room)})
}

func (h *roomHandler) listRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.svc.ListRooms(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	resp := make([]roomResponse, 0, len(rooms))
	for _, room := range rooms {
		resp = append(resp, toRoomResponse(room))
	}

	respondJSON(w, http.StatusOK, listRoomsResponse{Rooms: resp})
}
