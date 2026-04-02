package middleware

import (
	"log/slog"
	"net/http"
)

// TODO: вынести response-хелперы в internal/http/response, чтобы middleware и handlers использовали одно место
const forbiddenBody = `{"error":{"code":"FORBIDDEN","message":"forbidden"}}`

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if _, ok := allowed[role]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				if _, err := w.Write([]byte(forbiddenBody)); err != nil {
					slog.Error("writing forbidden response", "err", err)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
