package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

const testSecret = "test-secret"

type stubTokenStore struct {
	issued string
}

func (s *stubTokenStore) Issue(_ string) string            { return s.issued }
func (s *stubTokenStore) Consume(_ string) (string, bool) { return "", false }

type stubHub struct{}

func (s *stubHub) Subscribe(_ string) (<-chan model.Notification, func(), error) {
	return nil, func() {}, nil
}

func TestIssueToken_ReturnsToken(t *testing.T) {
	userID := uuid.New()
	store := &stubTokenStore{issued: "tok-abc"}
	h := newSSEHandler(&stubHub{}, store)

	rr := withAuthRequest(userID, "user", httptest.NewRequest(http.MethodPost, "/sse-token", nil), http.HandlerFunc(h.issueToken))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var resp issueTokenResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "tok-abc", resp.Token)
}

func TestIssueToken_PassesUserIDToStore(t *testing.T) {
	userID := uuid.New()

	captureStore := &captureIssueStore{}
	h := newSSEHandler(&stubHub{}, captureStore)

	rr := withAuthRequest(userID, "user", httptest.NewRequest(http.MethodPost, "/sse-token", nil), http.HandlerFunc(h.issueToken))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, userID.String(), captureStore.got, "Issue должен получить userID из JWT")
}

type captureIssueStore struct {
	got string
}

func (c *captureIssueStore) Issue(userID string) string {
	c.got = userID
	return "tok"
}
func (c *captureIssueStore) Consume(_ string) (string, bool) { return "", false }
