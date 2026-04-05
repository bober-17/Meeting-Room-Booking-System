package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/internships-backend/test-backend-bober-17/internal/http/middleware"
	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

// requestTimeout — максимальное время обработки одного HTTP-запроса.
// Устанавливается чуть меньше serverWriteTimeout (5s), чтобы приложение успело
// вернуть корректный JSON 503 до того, как сервер принудительно закроет соединение.
const requestTimeout = 4 * time.Second

func NewRouter(
	jwtSecret string,
	authSvc AuthService,
	roomSvc RoomService,
	scheduleSvc ScheduleService,
	slotSvc SlotService,
	bookingSvc BookingService,
) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Timeout(requestTimeout))
	r.Use(chimiddleware.Recoverer)

	auth := newAuthHandler(authSvc)
	room := newRoomHandler(roomSvc)
	schedule := newScheduleHandler(scheduleSvc)
	slotH := newSlotHandler(slotSvc)
	bookingH := newBookingHandler(bookingSvc)

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

		// Slots
		r.Get("/rooms/{roomId}/slots/list", slotH.listSlots)

		// Bookings
		r.With(middleware.RequireRole(string(model.RoleUser))).
			Post("/bookings/create", bookingH.createBooking)
		r.With(middleware.RequireRole(string(model.RoleAdmin))).
			Get("/bookings/list", bookingH.listBookings)
		r.With(middleware.RequireRole(string(model.RoleUser))).
			Get("/bookings/my", bookingH.listUserBookings)
		r.With(middleware.RequireRole(string(model.RoleUser))).
			Post("/bookings/{bookingId}/cancel", bookingH.cancelBooking)
	})

	return r
}
