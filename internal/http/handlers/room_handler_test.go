package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
	"github.com/internships-backend/test-backend-bober-17/mocks"
)

func TestCreateRoom_UserRole_Returns403(t *testing.T) {
	roomSvc := mocks.NewMockRoomService(t)
	h := newRouter(nil, roomSvc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":"Room A"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCreateRoom_EmptyName_Returns400(t *testing.T) {
	roomSvc := mocks.NewMockRoomService(t)
	h := newRouter(nil, roomSvc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateRoom_ZeroCapacity_Returns400(t *testing.T) {
	roomSvc := mocks.NewMockRoomService(t)
	h := newRouter(nil, roomSvc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":"Room A","capacity":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateRoom_HappyPath_Returns201(t *testing.T) {
	want := model.Room{ID: uuid.New(), Name: "Room A"}
	roomSvc := mocks.NewMockRoomService(t)
	roomSvc.On("CreateRoom", mockCtx(), "Room A", (*string)(nil), (*int)(nil)).Return(want, nil)
	h := newRouter(nil, roomSvc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":"Room A"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), want.ID.String())
}

func TestListRooms_HappyPath_Returns200(t *testing.T) {
	want := []model.Room{{ID: uuid.New(), Name: "Room A"}}
	roomSvc := mocks.NewMockRoomService(t)
	roomSvc.On("ListRooms", mockCtx()).Return(want, nil)
	h := newRouter(nil, roomSvc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Room A")
}
