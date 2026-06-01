package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

// OnRegisterFunc is called after a successful user registration with (userID, email).
type OnRegisterFunc func(userID, email string)

type AuthHandler struct {
	userQ      *queries.UserQueries
	creditQ    *queries.CreditQueries
	apiKeyQ    *queries.APIKeyQueries
	jwt        *auth.JWTService
	onRegister OnRegisterFunc
}

func NewAuthHandler(userQ *queries.UserQueries, creditQ *queries.CreditQueries, apiKeyQ *queries.APIKeyQueries, jwt *auth.JWTService) *AuthHandler {
	return &AuthHandler{userQ: userQ, creditQ: creditQ, apiKeyQ: apiKeyQ, jwt: jwt}
}

// SetOnRegister sets a callback invoked after successful registration.
func (h *AuthHandler) SetOnRegister(fn OnRegisterFunc) {
	h.onRegister = fn
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
	Token  string `json:"token"`
	APIKey string `json:"api_key,omitempty"`
	User   any    `json:"user"`
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

	// Auto-create a default API key for the new user
	rawKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate API key")
		return
	}
	if _, err := h.apiKeyQ.Create(ctx, user.ID, keyHash, keyPrefix, "Default"); err != nil {
		writeError(ctx, 500, "server_error", "Failed to create API key")
		return
	}

	setSessionCookie(ctx, rawKey)
	writeJSON(ctx, 201, authResponse{
		Token:  rawKey,
		APIKey: rawKey,
		User: map[string]any{
			"id":       user.ID,
			"email":    user.Email,
			"name":     user.Name,
			"is_admin": user.IsAdmin,
		},
	})

	if h.onRegister != nil {
		h.onRegister(user.ID, user.Email)
	}
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

	keys, err := h.apiKeyQ.ListByUser(ctx, user.ID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list API keys")
		return
	}
	if activeAPIKeyCount(keys) > 0 {
		if h.jwt == nil {
			writeError(ctx, 500, "server_error", "JWT not configured")
			return
		}
		token, err := h.jwt.Generate(user.ID, user.Email)
		if err != nil {
			writeError(ctx, 500, "server_error", "Failed to generate session token")
			return
		}
		setSessionCookie(ctx, token)
		writeJSON(ctx, 200, authResponse{
			Token: token,
			User: map[string]any{
				"id":       user.ID,
				"email":    user.Email,
				"name":     user.Name,
				"is_admin": user.IsAdmin,
			},
		})
		return
	}

	rawKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate session key")
		return
	}
	if _, err := h.apiKeyQ.Create(ctx, user.ID, keyHash, keyPrefix, "Default"); err != nil {
		writeError(ctx, 500, "server_error", "Failed to create session key")
		return
	}

	setSessionCookie(ctx, rawKey)
	writeJSON(ctx, 200, authResponse{
		Token:  rawKey,
		APIKey: rawKey,
		User: map[string]any{
			"id":       user.ID,
			"email":    user.Email,
			"name":     user.Name,
			"is_admin": user.IsAdmin,
		},
	})
}

func activeAPIKeyCount(keys []model.APIKey) int {
	n := 0
	for _, k := range keys {
		if !k.Revoked {
			n++
		}
	}
	return n
}

// HandleLogout clears the session cookie.
func (h *AuthHandler) HandleLogout(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Set-Cookie", "op_session=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0")
	writeJSON(ctx, 200, map[string]any{"ok": true})
}

func setSessionCookie(ctx *fasthttp.RequestCtx, key string) {
	ctx.Response.Header.Set("Set-Cookie",
		"op_session="+key+"; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000")
}
