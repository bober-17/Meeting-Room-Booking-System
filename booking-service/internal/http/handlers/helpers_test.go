package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/http/handlers"
	"github.com/bober-17/meeting-room-booking-system/booking-service/internal/model"
	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
	"github.com/bober-17/meeting-room-booking-system/booking-service/mocks"
)

const testJWTSecret = "test-secret"

func newRouter(
	authSvc *mocks.MockAuthService,
	roomSvc *mocks.MockRoomService,
	scheduleSvc *mocks.MockScheduleService,
	slotSvc *mocks.MockSlotService,
	bookingSvc *mocks.MockBookingService,
) http.Handler {
	return handlers.NewRouter(testJWTSecret, authSvc, roomSvc, scheduleSvc, slotSvc, bookingSvc)
}

func adminToken(t interface{ Fatal(...interface{}) }) string {
	token, err := pkgjwt.GenerateToken(uuid.MustParse("00000000-0000-0000-0000-000000000001"), string(model.RoleAdmin), testJWTSecret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func userToken(t interface{ Fatal(...interface{}) }) string {
	token, err := pkgjwt.GenerateToken(uuid.MustParse("00000000-0000-0000-0000-000000000002"), string(model.RoleUser), testJWTSecret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func doRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// mockCtx возвращает matcher, принимающий любой context.Context.
// Используется в mock.On вместо конкретного контекста,
// потому что chi добавляет свои значения в контекст запроса.
func mockCtx() any {
	return mock.MatchedBy(func(_ context.Context) bool { return true })
}
