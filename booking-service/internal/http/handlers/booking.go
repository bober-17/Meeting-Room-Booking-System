package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/http/middleware"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

type BookingService interface {
	CreateBooking(ctx context.Context, slotID, userID uuid.UUID, createConferenceLink bool) (model.Booking, error)
	CancelBooking(ctx context.Context, bookingID, userID uuid.UUID) (model.Booking, error)
	ListBookings(ctx context.Context, page, pageSize int) ([]model.Booking, int, error)
	ListUserBookings(ctx context.Context, userID uuid.UUID) ([]model.Booking, error)
}

type bookingHandler struct {
	svc BookingService
}

func newBookingHandler(svc BookingService) *bookingHandler {
	return &bookingHandler{svc: svc}
}

type createBookingRequest struct {
	SlotID               string `json:"slotId"`
	CreateConferenceLink bool   `json:"createConferenceLink"`
}

type bookingResponse struct {
	ID             string  `json:"id"`
	SlotID         string  `json:"slotId"`
	UserID         string  `json:"userId"`
	Status         string  `json:"status"`
	ConferenceLink *string `json:"conferenceLink"`
	CreatedAt      string  `json:"createdAt"`
}

type bookingResult struct {
	Booking bookingResponse `json:"booking"`
}

type paginationResponse struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type listBookingsResponse struct {
	Bookings   []bookingResponse  `json:"bookings"`
	Pagination paginationResponse `json:"pagination"`
}

type listUserBookingsResponse struct {
	Bookings []bookingResponse `json:"bookings"`
}

func toBookingResponse(b model.Booking) bookingResponse {
	return bookingResponse{
		ID:             b.ID.String(),
		SlotID:         b.SlotID.String(),
		UserID:         b.UserID.String(),
		Status:         string(b.Status),
		ConferenceLink: b.ConferenceLink,
		CreatedAt:      b.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *bookingHandler) createBooking(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	userID := middleware.UserIDFromContext(r.Context())

	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	if req.SlotID == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "slotId is required"},
		})
		return
	}

	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid slotId"},
		})
		return
	}

	booking, err := h.svc.CreateBooking(r.Context(), slotID, userID, req.CreateConferenceLink)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, bookingResult{Booking: toBookingResponse(booking)})
}

func (h *bookingHandler) cancelBooking(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	bookingID, err := uuid.Parse(chi.URLParam(r, "bookingId"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid bookingId"},
		})
		return
	}

	booking, err := h.svc.CancelBooking(r.Context(), bookingID, userID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, bookingResult{Booking: toBookingResponse(booking)})
}

func (h *bookingHandler) listBookings(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := parsePagination(w, r)
	if !ok {
		return
	}

	bookings, total, err := h.svc.ListBookings(r.Context(), page, pageSize)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := listBookingsResponse{
		Bookings: make([]bookingResponse, len(bookings)),
		Pagination: paginationResponse{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
	for i, b := range bookings {
		resp.Bookings[i] = toBookingResponse(b)
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *bookingHandler) listUserBookings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	bookings, err := h.svc.ListUserBookings(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := listUserBookingsResponse{Bookings: make([]bookingResponse, len(bookings))}
	for i, b := range bookings {
		resp.Bookings[i] = toBookingResponse(b)
	}

	respondJSON(w, http.StatusOK, resp)
}

func parsePagination(w http.ResponseWriter, r *http.Request) (page, pageSize int, ok bool) {
	page = defaultPage
	pageSize = defaultPageSize

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			respondJSON(w, http.StatusBadRequest, errorResponse{
				Error: errorBody{Code: codeInvalidRequest, Message: "page must be a positive integer"},
			})
			return 0, 0, false
		}
		page = p
	}

	if v := r.URL.Query().Get("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > maxPageSize {
			respondJSON(w, http.StatusBadRequest, errorResponse{
				Error: errorBody{Code: codeInvalidRequest, Message: "pageSize must be between 1 and 100"},
			})
			return 0, 0, false
		}
		pageSize = ps
	}

	return page, pageSize, true
}
