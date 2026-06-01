package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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

type openAIDevicePendingState struct {
	UserID          string
	DeviceAuthID    string
	UserCode        string
	IntervalSeconds int
	ExpiresAt       time.Time
}

var openAIOAuthStates = struct {
	sync.Mutex
	items map[string]openAIOAuthPendingState
}{items: make(map[string]openAIOAuthPendingState)}

var openAIDeviceStates = struct {
	sync.Mutex
	items map[string]openAIDevicePendingState
}{items: make(map[string]openAIDevicePendingState)}

func (h *OpenAIOAuthHandler) HandleStart(ctx *fasthttp.RequestCtx) {
	h.HandleDeviceStart(ctx)
}

func (h *OpenAIOAuthHandler) HandleDeviceStart(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	deviceCode, err := requestOpenAIDeviceUserCode(ctx)
	if err != nil {
		writeError(ctx, 502, "openai_auth_error", "Failed to start OpenAI device sign-in: "+err.Error())
		return
	}
	loginID, err := randomURLToken(24)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI device sign-in")
		return
	}
	expiresAt := time.Now().Add(15 * time.Minute)
	openAIDeviceStates.Lock()
	openAIDeviceStates.items[loginID] = openAIDevicePendingState{
		UserID:          userID,
		DeviceAuthID:    deviceCode.DeviceAuthID,
		UserCode:        deviceCode.UserCode,
		IntervalSeconds: deviceCode.IntervalSeconds,
		ExpiresAt:       expiresAt,
	}
	openAIDeviceStates.Unlock()

	writeJSON(ctx, 200, map[string]any{
		"login_id":         loginID,
		"verification_url": openAIOAuthIssuer() + "/codex/device",
		"user_code":        deviceCode.UserCode,
		"interval_seconds": deviceCode.IntervalSeconds,
		"expires_at":       expiresAt.UTC().Format(time.RFC3339),
	})
}

type openAIDevicePollReq struct {
	LoginID string `json:"login_id"`
}

func (h *OpenAIOAuthHandler) HandleDevicePoll(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	var req openAIDevicePollReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if req.LoginID == "" {
		writeError(ctx, 400, "invalid_request", "login_id is required")
		return
	}

	openAIDeviceStates.Lock()
	pending, ok := openAIDeviceStates.items[req.LoginID]
	if ok && (pending.UserID != userID || time.Now().After(pending.ExpiresAt)) {
		delete(openAIDeviceStates.items, req.LoginID)
		ok = false
	}
	openAIDeviceStates.Unlock()
	if !ok {
		writeError(ctx, 404, "not_found", "OpenAI device sign-in was not found or expired")
		return
	}

	codeResp, err := pollOpenAIDeviceToken(ctx, pending.DeviceAuthID, pending.UserCode)
	if errors.Is(err, errOpenAIDevicePending) {
		writeJSON(ctx, fasthttp.StatusAccepted, map[string]any{
			"status":  "pending",
			"message": "Waiting for OpenAI device authorization",
		})
		return
	}
	if err != nil {
		openAIDeviceStates.Lock()
		delete(openAIDeviceStates.items, req.LoginID)
		openAIDeviceStates.Unlock()
		writeError(ctx, 502, "openai_auth_error", "OpenAI device sign-in failed: "+err.Error())
		return
	}

	authJSON, err := completeOpenAIDeviceAuth(ctx, codeResp)
	if err != nil {
		openAIDeviceStates.Lock()
		delete(openAIDeviceStates.items, req.LoginID)
		openAIDeviceStates.Unlock()
		writeError(ctx, 502, "openai_auth_error", "OpenAI device sign-in failed: "+err.Error())
		return
	}
	if err := h.pkQ.UpsertAuthJSON(ctx, userID, openAIMaxPlanProvider, authJSON); err != nil {
		writeError(ctx, 500, "server_error", "OpenAI device sign-in could not be saved")
		return
	}
	openAIDeviceStates.Lock()
	delete(openAIDeviceStates.items, req.LoginID)
	openAIDeviceStates.Unlock()
	writeJSON(ctx, 200, map[string]any{
		"status":  "success",
		"message": "OpenAI Max plan sign-in saved",
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

type openAIDeviceUserCode struct {
	DeviceAuthID    string
	UserCode        string
	IntervalSeconds int
}

type openAIDeviceUserCodeResp struct {
	DeviceAuthID string          `json:"device_auth_id"`
	UserCode     string          `json:"user_code"`
	UserCodeAlt  string          `json:"usercode"`
	Interval     json.RawMessage `json:"interval"`
}

type openAIDeviceTokenResp struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeChallenge     string `json:"code_challenge"`
	CodeVerifier      string `json:"code_verifier"`
}

var errOpenAIDevicePending = errors.New("openai device authorization pending")

func requestOpenAIDeviceUserCode(ctx context.Context) (*openAIDeviceUserCode, error) {
	body, _ := json.Marshal(map[string]string{"client_id": openAIOAuthClientID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIDeviceAuthBaseURL()+"/deviceauth/usercode", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device code request returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed openAIDeviceUserCodeResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	userCode := strings.TrimSpace(parsed.UserCode)
	if userCode == "" {
		userCode = strings.TrimSpace(parsed.UserCodeAlt)
	}
	intervalSeconds := parseOpenAIDeviceInterval(parsed.Interval)
	if intervalSeconds <= 0 {
		intervalSeconds = 5
	}
	if strings.TrimSpace(parsed.DeviceAuthID) == "" || userCode == "" {
		return nil, fmt.Errorf("device code response was incomplete")
	}
	return &openAIDeviceUserCode{
		DeviceAuthID:    strings.TrimSpace(parsed.DeviceAuthID),
		UserCode:        userCode,
		IntervalSeconds: intervalSeconds,
	}, nil
}

func pollOpenAIDeviceToken(ctx context.Context, deviceAuthID, userCode string) (*openAIDeviceTokenResp, error) {
	body, _ := json.Marshal(map[string]string{
		"device_auth_id": deviceAuthID,
		"user_code":      userCode,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIDeviceAuthBaseURL()+"/deviceauth/token", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == fasthttp.StatusForbidden || resp.StatusCode == fasthttp.StatusNotFound {
		return nil, errOpenAIDevicePending
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device token request returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed openAIDeviceTokenResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if strings.TrimSpace(parsed.AuthorizationCode) == "" || strings.TrimSpace(parsed.CodeVerifier) == "" {
		return nil, fmt.Errorf("device token response was incomplete")
	}
	return &parsed, nil
}

func completeOpenAIDeviceAuth(ctx context.Context, codeResp *openAIDeviceTokenResp) (string, error) {
	if codeResp == nil {
		return "", fmt.Errorf("device token response was empty")
	}
	redirectURI := openAIOAuthIssuer() + "/deviceauth/callback"
	return completeOpenAIOAuth(ctx, codeResp.AuthorizationCode, redirectURI, codeResp.CodeVerifier)
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

func openAIDeviceAuthBaseURL() string {
	if endpoint := strings.TrimRight(os.Getenv("OPENAI_DEVICE_AUTH_BASE_URL"), "/"); endpoint != "" {
		return endpoint
	}
	return openAIOAuthIssuer() + "/api/accounts"
}

func parseOpenAIDeviceInterval(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		var parsed int
		if _, scanErr := fmt.Sscanf(strings.TrimSpace(asString), "%d", &parsed); scanErr == nil {
			return parsed
		}
		return 0
	}
	var asNumber int
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return asNumber
	}
	return 0
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
