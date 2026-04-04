package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/internships-backend/test-backend-bober-17/internal/model"
)

// maxPasswordLen — bcrypt молча обрезает пароль на 72 байтах,
// поэтому длиннее не принимаем, чтобы не было двух разных паролей с одним хешем.
const maxPasswordLen = 72

type dummyLoginRequest struct {
	Role string `json:"role"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthService interface {
	DummyLogin(ctx context.Context, role model.Role) (string, error)
	Register(ctx context.Context, email, password string, role model.Role) (model.User, error)
	Login(ctx context.Context, email, password string) (string, error)
}

type authHandler struct {
	svc AuthService
}

func newAuthHandler(svc AuthService) *authHandler {
	return &authHandler{svc: svc}
}

func (h *authHandler) dummyLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req dummyLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	role := model.Role(req.Role)
	if !role.Valid() {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "role must be admin or user"},
		})
		return
	}

	token, err := h.svc.DummyLogin(r.Context(), role)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, tokenResponse{Token: token})
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid email"},
		})
		return
	}
	if req.Password == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "password is required"},
		})
		return
	}
	if len(req.Password) > maxPasswordLen {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "password must not exceed 72 characters"},
		})
		return
	}

	role := model.Role(req.Role)
	if !role.Valid() {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "role must be admin or user"},
		})
		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.Password, role)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, registerResponse{User: toUserResponse(user)})
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "invalid request body"},
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorBody{Code: codeInvalidRequest, Message: "email and password are required"},
		})
		return
	}

	// Формат email намеренно не валидируем: если он некорректен,
	// репозиторий вернёт ErrInvalidCredentials — результат тот же, 401.
	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, tokenResponse{Token: token})
}

type tokenResponse struct {
	Token string `json:"token"`
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type registerResponse struct {
	User userResponse `json:"user"`
}

func toUserResponse(u model.User) userResponse {
	return userResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
