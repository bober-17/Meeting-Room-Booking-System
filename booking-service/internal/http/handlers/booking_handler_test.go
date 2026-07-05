package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	"github.com/bober-17/meeting-room-booking-system/booking-service/mocks"
)

func TestCreateBooking_MissingSlotID_Returns400(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateBooking_InvalidSlotID_Returns400(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":"not-a-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateBooking_SlotNotFound_Returns404(t *testing.T) {
	slotID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CreateBooking", mockCtx(), slotID, userID, false).Return(model.Booking{}, model.ErrSlotNotFound)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	body := fmt.Sprintf(`{"slotId":"%s"}`, slotID)
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCreateBooking_SlotInPast_Returns400(t *testing.T) {
	slotID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CreateBooking", mockCtx(), slotID, userID, false).Return(model.Booking{}, model.ErrSlotInPast)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	body := fmt.Sprintf(`{"slotId":"%s"}`, slotID)
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateBooking_SlotAlreadyBooked_Returns409(t *testing.T) {
	slotID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CreateBooking", mockCtx(), slotID, userID, false).Return(model.Booking{}, model.ErrSlotAlreadyBooked)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	body := fmt.Sprintf(`{"slotId":"%s"}`, slotID)
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestCreateBooking_AdminRole_Returns403(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":"`+uuid.New().String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCreateBooking_HappyPath_Returns201(t *testing.T) {
	slotID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	want := model.Booking{ID: uuid.New(), SlotID: slotID, UserID: userID, Status: model.BookingStatusActive}
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CreateBooking", mockCtx(), slotID, userID, false).Return(want, nil)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	body := fmt.Sprintf(`{"slotId":"%s"}`, slotID)
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), want.ID.String())
}

// CancelBooking

func TestCancelBooking_InvalidBookingID_Returns400(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, "/bookings/not-a-uuid/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCancelBooking_BookingNotFound_Returns404(t *testing.T) {
	bookingID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CancelBooking", mockCtx(), bookingID, userID).Return(model.Booking{}, model.ErrBookingNotFound)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/bookings/%s/cancel", bookingID), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCancelBooking_Forbidden_Returns403(t *testing.T) {
	bookingID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CancelBooking", mockCtx(), bookingID, userID).Return(model.Booking{}, model.ErrForbidden)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/bookings/%s/cancel", bookingID), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCancelBooking_HappyPath_Returns200(t *testing.T) {
	bookingID := uuid.New()
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	want := model.Booking{ID: bookingID, UserID: userID, Status: model.BookingStatusCancelled}
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("CancelBooking", mockCtx(), bookingID, userID).Return(want, nil)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/bookings/%s/cancel", bookingID), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "cancelled")
}

// ListBookings (admin)

func TestListBookings_InvalidPage_Returns400(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/list?page=0", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListBookings_PageSizeTooLarge_Returns400(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/list?pageSize=101", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListBookings_UserRole_Returns403(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestListBookings_HappyPath_Returns200(t *testing.T) {
	want := []model.Booking{{ID: uuid.New(), Status: model.BookingStatusActive}}
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("ListBookings", mockCtx(), 1, 20).Return(want, 1, nil)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/list", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "pagination")
}

// ListUserBookings

func TestListUserBookings_AdminRole_Returns403(t *testing.T) {
	bookingSvc := mocks.NewMockBookingService(t)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestListUserBookings_HappyPath_Returns200(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	want := []model.Booking{{
		ID:     uuid.New(),
		UserID: userID,
		Status: model.BookingStatusActive,
		SlotID: uuid.New(),
	}}
	bookingSvc := mocks.NewMockBookingService(t)
	bookingSvc.On("ListUserBookings", mockCtx(), userID).Return(want, nil)
	h := newRouter(nil, nil, nil, nil, bookingSvc)

	req := httptest.NewRequest(http.MethodGet, "/bookings/my", nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), want[0].ID.String())
}
