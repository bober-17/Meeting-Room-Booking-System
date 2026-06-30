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

func scheduleURL(roomID uuid.UUID) string {
	return fmt.Sprintf("/rooms/%s/schedule/create", roomID)
}

func TestCreateSchedule_UserRole_Returns403(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCreateSchedule_InvalidRoomID_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/rooms/not-a-uuid/schedule/create",
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_EmptyDaysOfWeek_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_DayOutOfRange_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[0],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_DuplicateDays_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[1,1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_BadStartTimeFormat_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"not-a-time","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_EndTimeBeforeStartTime_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"10:00","endTime":"09:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_MinutesNotGranular_Returns400(t *testing.T) {
	scheduleSvc := mocks.NewMockScheduleService(t)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(uuid.New()),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:15","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateSchedule_RoomNotFound_Returns404(t *testing.T) {
	roomID := uuid.New()
	scheduleSvc := mocks.NewMockScheduleService(t)
	scheduleSvc.On("CreateSchedule", mockCtx(), roomID, []int{1}, "09:00", "10:00").
		Return(model.Schedule{}, model.ErrRoomNotFound)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(roomID),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCreateSchedule_AlreadyExists_Returns409(t *testing.T) {
	roomID := uuid.New()
	scheduleSvc := mocks.NewMockScheduleService(t)
	scheduleSvc.On("CreateSchedule", mockCtx(), roomID, []int{1}, "09:00", "10:00").
		Return(model.Schedule{}, model.ErrScheduleExists)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(roomID),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestCreateSchedule_HappyPath_Returns201(t *testing.T) {
	roomID := uuid.New()
	want := model.Schedule{ID: uuid.New(), RoomID: roomID, DaysOfWeek: []int{1}, StartTime: "09:00", EndTime: "10:00"}
	scheduleSvc := mocks.NewMockScheduleService(t)
	scheduleSvc.On("CreateSchedule", mockCtx(), roomID, []int{1}, "09:00", "10:00").Return(want, nil)
	h := newRouter(nil, nil, scheduleSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, scheduleURL(roomID),
		strings.NewReader(`{"daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), want.ID.String())
}
