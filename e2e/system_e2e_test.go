//go:build e2e_system

package e2e_system_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	bookingBase      = envOr("BOOKING_URL", "http://localhost:8080")
	notificationBase = envOr("NOTIFICATION_URL", "http://localhost:8081")

	// apiClient с таймаутом для обычных REST-запросов.
	// SSE-соединение открывается через http.DefaultClient без таймаута.
	apiClient = &http.Client{Timeout: 10 * time.Second}
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestSystem_BookingCreated_NotificationViaSSE — сквозной тест двух сервисов:
// POST /bookings/create → Outbox → Kafka → notification-service → SSE.
func TestSystem_BookingCreated_NotificationViaSSE(t *testing.T) {
	adminToken := dummyLogin(t, "admin")
	userToken := dummyLogin(t, "user")

	sseToken := issueSSEToken(t, userToken)

	// Контекст закроет SSE-соединение когда тест завершится.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ready := make(chan struct{})
	notifications := make(chan map[string]any, 1)
	go consumeSSE(ctx, notificationBase+"/notifications/stream?token="+sseToken, ready, notifications)

	// Ждём event: connected — соединение гарантированно установлено.
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("SSE connection not established within timeout")
	}

	bookingID := createRoomAndBook(t, adminToken, userToken)
	t.Logf("created booking %s, waiting for SSE notification...", bookingID)

	select {
	case n := <-notifications:
		assert.Equal(t, "booking.created", n["type"])
		assert.Equal(t, bookingID, n["booking_id"])
	case <-ctx.Done():
		t.Fatal("notification not received within 20s after booking creation")
	}
}

// consumeSSE читает SSE-поток.
// ready закрывается при получении event: connected.
// out получает распарсенные данные каждого event: notification.
func consumeSSE(ctx context.Context, url string, ready chan<- struct{}, out chan<- map[string]any) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var eventType, data string
	var readySent bool

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		case line == "":
			if eventType == "connected" && !readySent {
				close(ready)
				readySent = true
			}
			if eventType == "notification" && data != "" {
				var n map[string]any
				if json.Unmarshal([]byte(data), &n) == nil {
					select {
					case out <- n:
					default:
					}
				}
			}
			eventType, data = "", ""
		}
	}
}

func dummyLogin(t *testing.T, role string) string {
	t.Helper()
	resp := postJSON(t, bookingBase+"/dummyLogin", map[string]string{"role": role}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Token string `json:"token"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.Token)
	return body.Token
}

func issueSSEToken(t *testing.T, userToken string) string {
	t.Helper()
	resp := postJSON(t, notificationBase+"/sse-token", nil, userToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Token string `json:"token"`
	}
	decodeJSON(t, resp, &body)
	require.NotEmpty(t, body.Token)
	return body.Token
}

func createRoomAndBook(t *testing.T, adminToken, userToken string) (bookingID string) {
	t.Helper()

	resp := postJSON(t, bookingBase+"/rooms/create",
		map[string]any{"name": "System E2E Room"}, adminToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var roomBody struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
	}
	decodeJSON(t, resp, &roomBody)
	roomID := roomBody.Room.ID

	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	resp = postJSON(t,
		bookingBase+fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		map[string]any{
			"daysOfWeek": []int{isoWeekday(tomorrow)},
			"startTime":  "09:00",
			"endTime":    "18:00",
		},
		adminToken,
	)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	req, _ := http.NewRequest(http.MethodGet,
		bookingBase+fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, tomorrow.Format("2006-01-02")),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+userToken)
	slotsResp, err := apiClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, slotsResp.StatusCode)
	var slotsBody struct {
		Slots []struct {
			ID string `json:"id"`
		} `json:"slots"`
	}
	decodeJSON(t, slotsResp, &slotsBody)
	require.NotEmpty(t, slotsBody.Slots, "no slots available for tomorrow")

	resp = postJSON(t, bookingBase+"/bookings/create",
		map[string]any{"slotId": slotsBody.Slots[0].ID},
		userToken,
	)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var bookBody struct {
		Booking struct {
			ID string `json:"id"`
		} `json:"booking"`
	}
	decodeJSON(t, resp, &bookBody)
	return bookBody.Booking.ID
}

func postJSON(t *testing.T, url string, body any, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := apiClient.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(target))
}

func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}
