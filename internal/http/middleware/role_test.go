package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/internships-backend/test-backend-bober-17/internal/http/middleware"
)

func withRole(r *http.Request, role string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextKeyRole, role)
	return r.WithContext(ctx)
}

func TestRequireRole_MatchingRole_CallsNext(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := withRole(httptest.NewRequest(http.MethodGet, "/", nil), "admin")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_WrongRole_Returns403(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := withRole(httptest.NewRequest(http.MethodGet, "/", nil), "user")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_EmptyRole_Returns403(t *testing.T) {
	h := middleware.RequireRole("admin")(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireRole_MultipleAllowedRoles_BothPass(t *testing.T) {
	h := middleware.RequireRole("admin", "user")(http.HandlerFunc(okHandler))

	for _, role := range []string{"admin", "user"} {
		req := withRole(httptest.NewRequest(http.MethodGet, "/", nil), role)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "role=%s", role)
	}
}
