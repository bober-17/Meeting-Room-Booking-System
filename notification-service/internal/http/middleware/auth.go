package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	pkgjwt "github.com/bober-17/meeting-room-booking-system/notification-service/internal/pkg/jwt"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"

	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	unauthorizedBody    = `{"error":{"code":"UNAUTHORIZED","message":"unauthorized"}}`
)

// Auth проверяет заголовок Authorization: Bearer <token> и кладёт userID и role в context.
// При отсутствии или невалидном токене возвращает 401 Unauthorized.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(authorizationHeader)
			if authHeader == "" {
				respondUnauthorized(w)
				return
			}

			tokenString, ok := strings.CutPrefix(authHeader, bearerPrefix)
			if !ok || tokenString == "" {
				respondUnauthorized(w)
				return
			}

			userID, role, err := pkgjwt.ParseToken(tokenString, jwtSecret)
			if err != nil {
				respondUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext извлекает userID, установленный middleware Auth.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return v
}

// RoleFromContext извлекает role, установленную middleware Auth.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(unauthorizedBody)); err != nil {
		slog.Error("writing unauthorized response", "err", err)
	}
}
