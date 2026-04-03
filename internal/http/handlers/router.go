package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/internships-backend/test-backend-bober-17/internal/http/middleware"
	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

func NewRouter(
	jwtSecret string,
	authSvc AuthService,
	roomSvc RoomService,
	scheduleSvc ScheduleService,
) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)

	auth := newAuthHandler(authSvc)
	room := newRoomHandler(roomSvc)
	schedule := newScheduleHandler(scheduleSvc)

	// Публичные маршруты — без авторизации
	r.Get("/", InfoHandler)
	r.Get("/_info", InfoHandler)

	r.Post("/dummyLogin", auth.dummyLogin)
	r.Post("/register", auth.register)
	r.Post("/login", auth.login)

	// Защищённые маршруты
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		// Rooms
		r.Get("/rooms/list", room.listRooms)
		r.With(middleware.RequireRole(string(model.RoleAdmin))).
			Post("/rooms/create", room.createRoom)

		// Schedules
		r.With(middleware.RequireRole(string(model.RoleAdmin))).
			Post("/rooms/{roomId}/schedule/create", schedule.createSchedule)

		// TODO: подключить slot handlers (GET /rooms/{roomId}/slots/list)
		// TODO: подключить booking handlers (POST /bookings/create, GET /bookings/list, GET /bookings/my, POST /bookings/{bookingId}/cancel)
	})

	return r
}
