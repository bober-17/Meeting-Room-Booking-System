package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	pkgjwt "github.com/internships-backend/test-backend-bober-17/internal/pkg/jwt"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userID"
	ContextKeyRole   contextKey = "role"

	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	// TODO: вынести response-хелперы в internal/http/response, чтобы middleware и handlers использовали одно место
	unauthorizedBody = `{"error":{"code":"UNAUTHORIZED","message":"unauthorized"}}`
)

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

func UserIDFromContext(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(ContextKeyUserID).(uuid.UUID)
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
