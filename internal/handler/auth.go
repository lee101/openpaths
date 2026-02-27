package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/auth"
	"github.com/openpath/openpath/internal/db/queries"
)

type AuthHandler struct {
	userQ   *queries.UserQueries
	creditQ *queries.CreditQueries
	jwt     *auth.JWTService
}

func NewAuthHandler(userQ *queries.UserQueries, creditQ *queries.CreditQueries, jwt *auth.JWTService) *AuthHandler {
	return &AuthHandler{userQ: userQ, creditQ: creditQ, jwt: jwt}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

// HandleRegister handles POST /auth/register.
func (h *AuthHandler) HandleRegister(ctx *fasthttp.RequestCtx) {
	var req registerRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(ctx, 400, "invalid_request", "email and password are required")
		return
	}

	if len(req.Password) < 8 {
		writeError(ctx, 400, "invalid_request", "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to hash password")
		return
	}

	user, err := h.userQ.Create(ctx, req.Email, hash, req.Name)
	if err != nil {
		writeError(ctx, 409, "conflict", "Email already registered")
		return
	}

	// Initialize credit balance
	if err := h.creditQ.InitBalance(ctx, user.ID); err != nil {
		writeError(ctx, 500, "server_error", "Failed to initialize balance")
		return
	}

	token, err := h.jwt.Generate(user.ID, user.Email)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate token")
		return
	}

	writeJSON(ctx, 201, authResponse{
		Token: token,
		User: map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// HandleLogin handles POST /auth/login.
func (h *AuthHandler) HandleLogin(ctx *fasthttp.RequestCtx) {
	var req loginRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(ctx, 400, "invalid_request", "email and password are required")
		return
	}

	user, err := h.userQ.GetByEmail(ctx, req.Email)
	if err != nil {
		writeError(ctx, 401, "auth_error", "Invalid email or password")
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(ctx, 401, "auth_error", "Invalid email or password")
		return
	}

	if user.Disabled {
		writeError(ctx, 403, "auth_error", "Account is disabled")
		return
	}

	token, err := h.jwt.Generate(user.ID, user.Email)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate token")
		return
	}

	writeJSON(ctx, 200, authResponse{
		Token: token,
		User: map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}
