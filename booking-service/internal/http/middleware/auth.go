package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	pkgjwt "github.com/bober-17/meeting-room-booking-system/booking-service/internal/pkg/jwt"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyRole   contextKey = "role"

	authorizationHeader = "Authorization"
	bearerPrefix        = "bearer " // lowercase: сравниваем без учёта регистра (RFC 7235 §2.1)
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

			if !strings.HasPrefix(strings.ToLower(authHeader), bearerPrefix) {
				respondUnauthorized(w)
				return
			}
			tokenString := authHeader[len(bearerPrefix):]
			if tokenString == "" {
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

func UserIDFromContext(ctx context.Context) uuid.UUID {
	v, ok := ctx.Value(ContextKeyUserID).(uuid.UUID)
	if !ok {
		panic("UserIDFromContext: Auth middleware not in chain")
	}
	return v
}

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
