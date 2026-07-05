package handlers_test

import (
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

func TestDummyLogin_InvalidRole_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"superadmin"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDummyLogin_InvalidBody_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDummyLogin_Admin_Returns200WithToken(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("DummyLogin", mockCtx(), model.RoleAdmin).Return("jwt-token", nil)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/dummyLogin", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "token")
}

func TestRegister_EmptyEmail_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"","password":"pass","role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegister_InvalidEmail_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"notanemail","password":"pass","role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegister_EmptyPassword_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"u@example.com","password":"","role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegister_PasswordTooLong_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	longPassword := strings.Repeat("a", 73)
	body := `{"email":"u@example.com","password":"` + longPassword + `","role":"user"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegister_InvalidRole_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"u@example.com","password":"pass","role":"god"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLogin_EmptyFields_Returns400(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"","password":""}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRegister_HappyPath_Returns201(t *testing.T) {
	want := model.User{ID: uuid.New(), Email: "u@example.com", Role: model.RoleUser}
	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("Register", mockCtx(), "u@example.com", "pass", model.RoleUser).Return(want, nil)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"u@example.com","password":"pass","role":"user"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), want.ID.String())
}

func TestLogin_HappyPath_Returns200WithToken(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("Login", mockCtx(), "u@example.com", "pass").Return("jwt-token", nil)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"u@example.com","password":"pass"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "token")
}

func TestLogin_WrongCredentials_Returns401(t *testing.T) {
	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("Login", mockCtx(), "u@example.com", "wrong").Return("", model.ErrInvalidCredentials)
	h := newRouter(authSvc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"u@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := doRequest(h, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
