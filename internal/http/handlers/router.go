package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/internships-backend/test-backend-bober-17/internal/http/middleware"
)

func NewRouter(jwtSecret string, authSvc AuthService) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)

	auth := newAuthHandler(authSvc)

	// Публичные маршруты — без авторизации
	r.Get("/", InfoHandler)
	r.Get("/_info", InfoHandler)

	r.Post("/dummyLogin", auth.dummyLogin)
	r.Post("/register", auth.register)
	r.Post("/login", auth.login)

	// Защищённые маршруты
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		// TODO: подключить room handlers (GET /rooms/list, POST /rooms/create)
		// TODO: подключить schedule handlers (POST /rooms/{roomId}/schedule/create)
		// TODO: подключить slot handlers (GET /rooms/{roomId}/slots/list)
		// TODO: подключить booking handlers (POST /bookings/create, GET /bookings/list, GET /bookings/my, POST /bookings/{bookingId}/cancel)
	})

	return r
}
