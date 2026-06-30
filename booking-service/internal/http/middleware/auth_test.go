package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/http/middleware"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
)

const testSecret = "test-secret"

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestAuth_NoHeader_Returns401(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_MissingBearerPrefix_Returns401(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "token-without-bearer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ValidToken_CallsNext(t *testing.T) {
	token, err := pkgjwt.GenerateToken(uuid.New(), "user", testSecret)
	if err != nil {
		t.Fatal(err)
	}

	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuth_WrongSecret_Returns401(t *testing.T) {
	token, err := pkgjwt.GenerateToken(uuid.New(), "user", "other-secret")
	if err != nil {
		t.Fatal(err)
	}

	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
