package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	"github.com/bober-17/meeting-room-booking-system/booking-service/mocks"
)

func slotsURL(roomID uuid.UUID, date string) string {
	return fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date)
}

func TestListSlots_InvalidRoomID_Returns400(t *testing.T) {
	slotSvc := mocks.NewMockSlotService(t)
	h := newRouter(nil, nil, nil, slotSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/rooms/not-a-uuid/slots/list?date=2024-06-10", nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListSlots_MissingDate_Returns400(t *testing.T) {
	slotSvc := mocks.NewMockSlotService(t)
	h := newRouter(nil, nil, nil, slotSvc, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/rooms/%s/slots/list", uuid.New()), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListSlots_BadDateFormat_Returns400(t *testing.T) {
	slotSvc := mocks.NewMockSlotService(t)
	h := newRouter(nil, nil, nil, slotSvc, nil)

	req := httptest.NewRequest(http.MethodGet, slotsURL(uuid.New(), "10-06-2024"), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListSlots_RoomNotFound_Returns404(t *testing.T) {
	roomID := uuid.New()
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	slotSvc := mocks.NewMockSlotService(t)
	slotSvc.On("ListAvailableSlots", mockCtx(), roomID, date).Return(nil, model.ErrRoomNotFound)
	h := newRouter(nil, nil, nil, slotSvc, nil)

	req := httptest.NewRequest(http.MethodGet, slotsURL(roomID, "2024-06-10"), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListSlots_HappyPath_Returns200(t *testing.T) {
	roomID := uuid.New()
	date := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	want := []model.Slot{
		{ID: uuid.New(), RoomID: roomID, StartAt: date.Add(9 * time.Hour), EndAt: date.Add(9*time.Hour + 30*time.Minute)},
	}
	slotSvc := mocks.NewMockSlotService(t)
	slotSvc.On("ListAvailableSlots", mockCtx(), roomID, date).Return(want, nil)
	h := newRouter(nil, nil, nil, slotSvc, nil)

	req := httptest.NewRequest(http.MethodGet, slotsURL(roomID, "2024-06-10"), nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), want[0].ID.String())
}
