//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/internships-backend/test-backend-bober-17/internal/client/conference"
	"github.com/internships-backend/test-backend-bober-17/internal/http/handlers"
	authrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/auth"
	bookingrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/booking"
	roomrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/room"
	schedulerepo "github.com/internships-backend/test-backend-bober-17/internal/repo/schedule"
	slotrepo "github.com/internships-backend/test-backend-bober-17/internal/repo/slot"
	authservice "github.com/internships-backend/test-backend-bober-17/internal/service/auth"
	bookingservice "github.com/internships-backend/test-backend-bober-17/internal/service/booking"
	roomservice "github.com/internships-backend/test-backend-bober-17/internal/service/room"
	scheduleservice "github.com/internships-backend/test-backend-bober-17/internal/service/schedule"
	slotservice "github.com/internships-backend/test-backend-bober-17/internal/service/slot"
)

const e2eJWTSecret = "e2e-test-secret"

var (
	testPool   *pgxpool.Pool
	testServer *httptest.Server
	testClient *http.Client
)

func poolDSN() string {
	if v := os.Getenv("E2E_DATABASE_URL"); v != "" {
		return v
	}
	return "host=localhost port=5434 user=postgres password=postgres dbname=booking_e2e sslmode=disable"
}

func migrateDSN() string {
	if v := os.Getenv("E2E_MIGRATE_URL"); v != "" {
		return v
	}
	return "pgx5://postgres:postgres@localhost:5434/booking_e2e?sslmode=disable"
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	var err error
	testPool, err = pgxpool.New(ctx, poolDSN())
	if err != nil {
		log.Fatalf("connect pool: %v", err)
	}
	if err := testPool.Ping(ctx); err != nil {
		testPool.Close()
		log.Fatalf("ping db: %v", err)
	}

	mg, err := migrate.New("file://../migrations", migrateDSN())
	if err != nil {
		testPool.Close()
		log.Fatalf("create migrator: %v", err)
	}
	if err := mg.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		testPool.Close()
		log.Fatalf("migrate up: %v", err)
	}
	mg.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authRepo := authrepo.New(testPool)
	roomRepo := roomrepo.New(testPool)
	scheduleRepo := schedulerepo.New(testPool)
	slotRepo := slotrepo.New(testPool)
	bookingRepo := bookingrepo.New(testPool)

	confClient := conference.New()

	authSvc := authservice.New(authRepo, e2eJWTSecret, logger)
	roomSvc := roomservice.New(roomRepo, logger)
	scheduleSvc := scheduleservice.New(scheduleRepo, logger)
	slotSvc := slotservice.New(roomRepo, scheduleRepo, slotRepo, logger)
	bookingSvc := bookingservice.New(bookingRepo, slotRepo, confClient, logger)

	router := handlers.NewRouter(e2eJWTSecret, authSvc, roomSvc, scheduleSvc, slotSvc, bookingSvc)
	testServer = httptest.NewServer(router)
	testClient = testServer.Client()

	code := m.Run()

	testServer.Close()
	testPool.Close()
	os.Exit(code)
}

// truncate очищает данные между тестами. Пользователи (seeded migration) не затрагиваются.
func truncate(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `TRUNCATE bookings, slots, schedules, rooms CASCADE`)
	require.NoError(t, err)
}

func doJSON(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, testServer.URL+path, bodyReader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := testClient.Do(req)
	require.NoError(t, err)
	return resp
}

func readJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(target))
}

func dummyLogin(t *testing.T, role string) string {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/dummyLogin", map[string]string{"role": role}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var result struct {
		Token string `json:"token"`
	}
	readJSON(t, resp, &result)
	require.NotEmpty(t, result.Token)
	return result.Token
}

// isoWeekday конвертирует Go Weekday (0=Sun) в ISO (1=Mon, 7=Sun).
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// TestE2E_CreateBooking проверяет сквозной сценарий:
// создание переговорки → создание расписания → получение слотов → бронирование.
func TestE2E_CreateBooking(t *testing.T) {
	truncate(t)

	adminToken := dummyLogin(t, "admin")
	userToken := dummyLogin(t, "user")

	// 1. Создать переговорку
	roomResp := doJSON(t, http.MethodPost, "/rooms/create",
		map[string]any{"name": "E2E Room"}, adminToken)
	require.Equal(t, http.StatusCreated, roomResp.StatusCode)
	var roomBody struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
	}
	readJSON(t, roomResp, &roomBody)
	roomID := roomBody.Room.ID
	require.NotEmpty(t, roomID)

	// 2. Создать расписание для завтрашнего дня недели
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	schedResp := doJSON(t, http.MethodPost,
		fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		map[string]any{
			"roomId":     roomID,
			"daysOfWeek": []int{isoWeekday(tomorrow)},
			"startTime":  "09:00",
			"endTime":    "18:00",
		}, adminToken)
	require.Equal(t, http.StatusCreated, schedResp.StatusCode)
	schedResp.Body.Close()

	// 3. Получить доступные слоты на завтра
	date := tomorrow.Format("2006-01-02")
	slotsResp := doJSON(t, http.MethodGet,
		fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), nil, userToken)
	require.Equal(t, http.StatusOK, slotsResp.StatusCode)
	var slotsBody struct {
		Slots []struct {
			ID string `json:"id"`
		} `json:"slots"`
	}
	readJSON(t, slotsResp, &slotsBody)
	require.NotEmpty(t, slotsBody.Slots, "expected slots for tomorrow")
	slotID := slotsBody.Slots[0].ID

	// 4. Забронировать слот от имени пользователя
	bookResp := doJSON(t, http.MethodPost, "/bookings/create",
		map[string]any{"slotId": slotID}, userToken)
	require.Equal(t, http.StatusCreated, bookResp.StatusCode)
	var bookBody struct {
		Booking struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"booking"`
	}
	readJSON(t, bookResp, &bookBody)
	assert.Equal(t, "active", bookBody.Booking.Status)
	assert.NotEmpty(t, bookBody.Booking.ID)
}

// TestE2E_CancelBooking проверяет отмену брони и идемпотентность операции.
func TestE2E_CancelBooking(t *testing.T) {
	truncate(t)

	adminToken := dummyLogin(t, "admin")
	userToken := dummyLogin(t, "user")

	// Setup: переговорка + расписание + слот + бронь
	roomResp := doJSON(t, http.MethodPost, "/rooms/create",
		map[string]any{"name": "E2E Cancel Room"}, adminToken)
	require.Equal(t, http.StatusCreated, roomResp.StatusCode)
	var roomBody struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
	}
	readJSON(t, roomResp, &roomBody)
	roomID := roomBody.Room.ID

	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	schedResp := doJSON(t, http.MethodPost,
		fmt.Sprintf("/rooms/%s/schedule/create", roomID),
		map[string]any{
			"roomId":     roomID,
			"daysOfWeek": []int{isoWeekday(tomorrow)},
			"startTime":  "09:00",
			"endTime":    "18:00",
		}, adminToken)
	require.Equal(t, http.StatusCreated, schedResp.StatusCode)
	schedResp.Body.Close()

	date := tomorrow.Format("2006-01-02")
	slotsResp := doJSON(t, http.MethodGet,
		fmt.Sprintf("/rooms/%s/slots/list?date=%s", roomID, date), nil, userToken)
	require.Equal(t, http.StatusOK, slotsResp.StatusCode)
	var slotsBody struct {
		Slots []struct {
			ID string `json:"id"`
		} `json:"slots"`
	}
	readJSON(t, slotsResp, &slotsBody)
	require.NotEmpty(t, slotsBody.Slots)

	bookResp := doJSON(t, http.MethodPost, "/bookings/create",
		map[string]any{"slotId": slotsBody.Slots[0].ID}, userToken)
	require.Equal(t, http.StatusCreated, bookResp.StatusCode)
	var bookBody struct {
		Booking struct {
			ID string `json:"id"`
		} `json:"booking"`
	}
	readJSON(t, bookResp, &bookBody)
	bookingID := bookBody.Booking.ID
	require.NotEmpty(t, bookingID)

	// Отменить бронь
	cancelResp := doJSON(t, http.MethodPost,
		fmt.Sprintf("/bookings/%s/cancel", bookingID), nil, userToken)
	require.Equal(t, http.StatusOK, cancelResp.StatusCode)
	var cancelBody struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
	}
	readJSON(t, cancelResp, &cancelBody)
	assert.Equal(t, "cancelled", cancelBody.Booking.Status)

	// Идемпотентность: повторная отмена должна вернуть 200 с тем же статусом
	cancelResp2 := doJSON(t, http.MethodPost,
		fmt.Sprintf("/bookings/%s/cancel", bookingID), nil, userToken)
	require.Equal(t, http.StatusOK, cancelResp2.StatusCode)
	var cancelBody2 struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
	}
	readJSON(t, cancelResp2, &cancelBody2)
	assert.Equal(t, "cancelled", cancelBody2.Booking.Status)
}
