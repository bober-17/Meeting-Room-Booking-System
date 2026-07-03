package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/middleware"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/notification-service/internal/pkg/jwt"
)

const testSecret = "test-secret"

func makeToken(userID uuid.UUID, role, secret string) string {
	claims := &pkgjwt.Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: userID,
		Role:   role,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestAuth_NoHeader_Returns401(t *testing.T) {
	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

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

func TestAuth_WrongSecret_Returns401(t *testing.T) {
	token := makeToken(uuid.New(), "user", "other-secret")

	h := middleware.Auth(testSecret)(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ValidToken_SetsContextValues(t *testing.T) {
	userID := uuid.New()
	token := makeToken(userID, "admin", testSecret)

	var gotID uuid.UUID
	var gotRole string
	capture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = middleware.UserIDFromContext(r.Context())
		gotRole = middleware.RoleFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	h := middleware.Auth(testSecret)(capture)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, userID, gotID)
	assert.Equal(t, "admin", gotRole)
}
