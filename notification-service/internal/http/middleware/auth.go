package middleware

import "net/http"

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
)

// Auth — JWT middleware, аналогичен booking-service.
// Реализация будет добавлена далее (SSE + HTTP handlers).
func Auth(_ string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("not implemented")
			next.ServeHTTP(w, r)
		})
	}
}