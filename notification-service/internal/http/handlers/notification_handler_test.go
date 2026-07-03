package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/http/middleware"
	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/notification-service/internal/pkg/jwt"
)

func withAuthRequest(userID uuid.UUID, role string, req *http.Request, h http.Handler) *httptest.ResponseRecorder {
	claims := &pkgjwt.Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID: userID,
		Role:   role,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testSecret))
	req.Header.Set("Authorization", "Bearer "+signed)

	rr := httptest.NewRecorder()
	middleware.Auth(testSecret)(h).ServeHTTP(rr, req)
	return rr
}

// stubNotifSvc фиксирует аргументы вызовов и возвращает настроенные значения.
type stubNotifSvc struct {
	listLimit      int
	listOffset     int
	listUnreadOnly bool
	listResult     []model.Notification
	listTotal      int
	markReadErr    error
}

func (s *stubNotifSvc) ListByUserID(_ context.Context, _ string, limit, offset int, unreadOnly bool) ([]model.Notification, int, error) {
	s.listLimit = limit
	s.listOffset = offset
	s.listUnreadOnly = unreadOnly
	return s.listResult, s.listTotal, nil
}

func (s *stubNotifSvc) MarkAsRead(_ context.Context, _, _ string) error { return s.markReadErr }
func (s *stubNotifSvc) MarkAllAsRead(_ context.Context, _ string) error  { return nil }

func TestParseIntParam(t *testing.T) {
	cases := []struct {
		s, desc    string
		def, min, max, want int
	}{
		{"", "empty→default", 20, 1, 100, 20},
		{"abc", "invalid→default", 20, 1, 100, 20},
		{"0", "below min→default", 20, 1, 100, 20},
		{"50", "valid", 20, 1, 100, 50},
		{"150", "above max→clamp", 20, 1, 100, 100},
		{"5", "no upper bound", 0, 0, -1, 5},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.want, parseIntParam(tc.s, tc.def, tc.min, tc.max))
		})
	}
}

func TestListNotifications_DefaultParams(t *testing.T) {
	svc := &stubNotifSvc{listResult: make([]model.Notification, 0)}
	h := &notificationHandler{svc: svc}

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rr := withAuthRequest(uuid.New(), "user", req, http.HandlerFunc(h.list))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 20, svc.listLimit)
	assert.Equal(t, 0, svc.listOffset)
	assert.False(t, svc.listUnreadOnly)

	var resp listNotificationsResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.NotNil(t, resp.Notifications)
}

func TestListNotifications_ClampsLimitAndPassesParams(t *testing.T) {
	svc := &stubNotifSvc{listResult: make([]model.Notification, 0)}
	h := &notificationHandler{svc: svc}

	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=999&offset=5&unread_only=true", nil)
	rr := withAuthRequest(uuid.New(), "user", req, http.HandlerFunc(h.list))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 100, svc.listLimit)
	assert.Equal(t, 5, svc.listOffset)
	assert.True(t, svc.listUnreadOnly)
}

func TestMarkRead_NotFound(t *testing.T) {
	svc := &stubNotifSvc{markReadErr: model.ErrNotificationNotFound}
	h := &notificationHandler{svc: svc}

	req := httptest.NewRequest(http.MethodPost, "/notifications/some-id/read", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := withAuthRequest(uuid.New(), "user", req, http.HandlerFunc(h.markRead))

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
