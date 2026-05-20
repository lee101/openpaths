package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/valyala/fasthttp"
)

const openAIOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"

type OpenAIOAuthHandler struct {
	pkQ *queries.ProviderKeyQueries
}

func NewOpenAIOAuthHandler(pkQ *queries.ProviderKeyQueries) *OpenAIOAuthHandler {
	return &OpenAIOAuthHandler{pkQ: pkQ}
}

type openAIOAuthPendingState struct {
	UserID       string
	CodeVerifier string
	RedirectURI  string
	ExpiresAt    time.Time
}

var openAIOAuthStates = struct {
	sync.Mutex
	items map[string]openAIOAuthPendingState
}{items: make(map[string]openAIOAuthPendingState)}

func (h *OpenAIOAuthHandler) HandleStart(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	codeVerifier, codeChallenge, err := newPKCEPair()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI sign-in")
		return
	}
	state, err := randomURLToken(32)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI sign-in")
		return
	}
	redirectURI := publicBaseURL(ctx) + "/account/openai/callback"
	openAIOAuthStates.Lock()
	openAIOAuthStates.items[state] = openAIOAuthPendingState{
		UserID:       userID,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}
	openAIOAuthStates.Unlock()

	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", openAIOAuthClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("scope", "openid profile email offline_access api.connectors.read api.connectors.invoke")
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	query.Set("id_token_add_organizations", "true")
	query.Set("codex_cli_simplified_flow", "true")
	query.Set("state", state)
	query.Set("originator", "openpaths")

	writeJSON(ctx, 200, map[string]any{
		"auth_url": openAIOAuthIssuer() + "/oauth/authorize?" + query.Encode(),
	})
}

func (h *OpenAIOAuthHandler) HandleCallback(ctx *fasthttp.RequestCtx) {
	if string(ctx.QueryArgs().Peek("error")) != "" {
		redirectOpenAICallback(ctx, "error", "OpenAI sign-in was cancelled or rejected")
		return
	}
	state := string(ctx.QueryArgs().Peek("state"))
	code := string(ctx.QueryArgs().Peek("code"))
	if state == "" || code == "" {
		redirectOpenAICallback(ctx, "error", "OpenAI sign-in callback was missing required fields")
		return
	}
	openAIOAuthStates.Lock()
	pending, ok := openAIOAuthStates.items[state]
	delete(openAIOAuthStates.items, state)
	openAIOAuthStates.Unlock()
	if !ok || time.Now().After(pending.ExpiresAt) {
		redirectOpenAICallback(ctx, "error", "OpenAI sign-in expired. Start again from Account.")
		return
	}

	authJSON, err := completeOpenAIOAuth(ctx, code, pending.RedirectURI, pending.CodeVerifier)
	if err != nil {
		redirectOpenAICallback(ctx, "error", "OpenAI sign-in failed: "+err.Error())
		return
	}
	if err := h.pkQ.UpsertAuthJSON(ctx, pending.UserID, openAIMaxPlanProvider, authJSON); err != nil {
		redirectOpenAICallback(ctx, "error", "OpenAI sign-in could not be saved")
		return
	}
	redirectOpenAICallback(ctx, "success", "OpenAI Max plan sign-in saved")
}

func completeOpenAIOAuth(ctx context.Context, code, redirectURI, codeVerifier string) (string, error) {
	tokens, err := exchangeOpenAICodeForTokens(ctx, code, redirectURI, codeVerifier)
	if err != nil {
		return "", err
	}
	apiKey, err := exchangeOpenAIIDTokenForAPIKey(ctx, tokens.IDToken)
	if err != nil {
		return "", err
	}
	auth := map[string]any{
		"auth_mode":      "chatgpt",
		"OPENAI_API_KEY": apiKey,
		"tokens": map[string]any{
			"id_token":      tokens.IDToken,
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
		},
		"last_refresh": time.Now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func exchangeOpenAICodeForTokens(ctx context.Context, code, redirectURI, codeVerifier string) (*openAIRefreshTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", openAIOAuthClientID)
	form.Set("code_verifier", codeVerifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIOAuthTokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange returned HTTP %d", resp.StatusCode)
	}
	var parsed openAIRefreshTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.IDToken == "" || parsed.AccessToken == "" || parsed.RefreshToken == "" {
		return nil, fmt.Errorf("token exchange response was incomplete")
	}
	return &parsed, nil
}

func redirectOpenAICallback(ctx *fasthttp.RequestCtx, status, message string) {
	target := "/account?openai_auth=" + url.QueryEscape(status)
	if message != "" {
		target += "&openai_auth_message=" + url.QueryEscape(message)
	}
	ctx.Response.Header.Set("Location", target)
	ctx.SetStatusCode(fasthttp.StatusFound)
}

func publicBaseURL(ctx *fasthttp.RequestCtx) string {
	if base := strings.TrimRight(os.Getenv("OPENPATHS_PUBLIC_URL"), "/"); base != "" {
		return base
	}
	proto := string(ctx.Request.Header.Peek("X-Forwarded-Proto"))
	if proto == "" {
		proto = "https"
		if ctx.IsTLS() {
			proto = "https"
		}
	}
	host := string(ctx.Request.Header.Peek("X-Forwarded-Host"))
	if host == "" {
		host = string(ctx.Host())
	}
	return proto + "://" + host
}

func openAIOAuthIssuer() string {
	if issuer := strings.TrimRight(os.Getenv("OPENAI_OAUTH_ISSUER"), "/"); issuer != "" {
		return issuer
	}
	return "https://auth.openai.com"
}

func openAIOAuthTokenEndpoint() string {
	if endpoint := strings.TrimSpace(os.Getenv("CODEX_REFRESH_TOKEN_URL_OVERRIDE")); endpoint != "" {
		return endpoint
	}
	return openAIOAuthIssuer() + "/oauth/token"
}

func newPKCEPair() (string, string, error) {
	verifier, err := randomURLToken(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
