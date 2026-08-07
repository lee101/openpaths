package handler

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
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
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/valyala/fasthttp"
)

const (
	openAIOAuthClientID         = "app_EMoamEEZ73f0CkXaXp7hrann"
	openAIOAuthLocalRedirectURI = "http://localhost:1455/auth/callback"
	openAIOAuthScope            = "openid profile email offline_access api.connectors.read api.connectors.invoke"
	openAIOAuthTicketVersion    = 1
	openAIDeviceLoginMode       = "device"
	openAIBrowserLoginMode      = "browser"
)

var (
	errOpenAIOAuthTicketInvalid = errors.New("invalid OpenAI sign-in session")
	errOpenAIOAuthTicketExpired = errors.New("expired OpenAI sign-in session")
	openAIOAuthHTTPClient       = &http.Client{Timeout: 30 * time.Second}
)

type OpenAIOAuthHandler struct {
	pkQ       *queries.ProviderKeyQueries
	stateAEAD cipher.AEAD
}

func NewOpenAIOAuthHandler(pkQ *queries.ProviderKeyQueries, stateSecret string) *OpenAIOAuthHandler {
	h := &OpenAIOAuthHandler{pkQ: pkQ}
	if strings.TrimSpace(stateSecret) == "" {
		return h
	}
	key := sha256.Sum256([]byte("openpaths/openai-oauth-state/v1\x00" + stateSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return h
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return h
	}
	h.stateAEAD = aead
	return h
}

// openAIOAuthLoginState is sealed into the opaque login_id returned to the
// browser. The ticket is authenticated and encrypted with the application's
// JWT secret, so pending logins survive process restarts and work across API
// replicas without putting OAuth verifier/device secrets in local memory.
type openAIOAuthLoginState struct {
	Version         int    `json:"v"`
	Mode            string `json:"mode"`
	UserID          string `json:"user_id"`
	DeviceAuthID    string `json:"device_auth_id,omitempty"`
	UserCode        string `json:"user_code,omitempty"`
	CodeVerifier    string `json:"code_verifier,omitempty"`
	OAuthState      string `json:"oauth_state,omitempty"`
	RedirectURI     string `json:"redirect_uri,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	StartedAt       int64  `json:"started_at"`
	ExpiresAt       int64  `json:"expires_at"`
}

func (h *OpenAIOAuthHandler) sealLoginState(state openAIOAuthLoginState) (string, error) {
	if h.stateAEAD == nil {
		return "", errors.New("OpenAI sign-in state encryption is not configured")
	}
	plain, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, h.stateAEAD.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := h.stateAEAD.Seal(nonce, nonce, plain, []byte("openpaths-openai-oauth-v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (h *OpenAIOAuthHandler) openLoginState(loginID, userID, mode string) (*openAIOAuthLoginState, error) {
	if h.stateAEAD == nil || strings.TrimSpace(loginID) == "" {
		return nil, errOpenAIOAuthTicketInvalid
	}
	sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(loginID))
	if err != nil || len(sealed) <= h.stateAEAD.NonceSize() {
		return nil, errOpenAIOAuthTicketInvalid
	}
	nonce, ciphertext := sealed[:h.stateAEAD.NonceSize()], sealed[h.stateAEAD.NonceSize():]
	plain, err := h.stateAEAD.Open(nil, nonce, ciphertext, []byte("openpaths-openai-oauth-v1"))
	if err != nil {
		return nil, errOpenAIOAuthTicketInvalid
	}
	var state openAIOAuthLoginState
	if json.Unmarshal(plain, &state) != nil || state.Version != openAIOAuthTicketVersion || state.UserID != userID || state.Mode != mode {
		return nil, errOpenAIOAuthTicketInvalid
	}
	if state.ExpiresAt <= 0 || time.Now().Unix() >= state.ExpiresAt {
		return nil, errOpenAIOAuthTicketExpired
	}
	return &state, nil
}

func writeOpenAIOAuthTicketError(ctx *fasthttp.RequestCtx, err error) {
	if errors.Is(err, errOpenAIOAuthTicketExpired) {
		writeError(ctx, fasthttp.StatusGone, "openai_auth_expired", "OpenAI sign-in expired. Start a new sign-in.")
		return
	}
	writeError(ctx, fasthttp.StatusBadRequest, "openai_auth_invalid", "OpenAI sign-in session is invalid. Start a new sign-in.")
}

func (h *OpenAIOAuthHandler) HandleStart(ctx *fasthttp.RequestCtx) {
	// Keep the original generic route compatible with clients that already use
	// it for device auth. Browser fallback has explicit /browser/* endpoints.
	h.HandleDeviceStart(ctx)
}

func (h *OpenAIOAuthHandler) HandleDeviceStart(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	if h.stateAEAD == nil {
		writeError(ctx, 503, "openai_auth_unavailable", "OpenAI sign-in is unavailable because JWT_SECRET is not configured")
		return
	}
	deviceCode, err := requestOpenAIDeviceUserCode(ctx)
	if err != nil {
		writeError(ctx, 502, "openai_auth_error", "Failed to start OpenAI device sign-in: "+err.Error())
		return
	}
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)
	loginID, err := h.sealLoginState(openAIOAuthLoginState{
		Version:         openAIOAuthTicketVersion,
		Mode:            openAIDeviceLoginMode,
		UserID:          userID,
		DeviceAuthID:    deviceCode.DeviceAuthID,
		UserCode:        deviceCode.UserCode,
		IntervalSeconds: deviceCode.IntervalSeconds,
		StartedAt:       now.Unix(),
		ExpiresAt:       expiresAt.Unix(),
	})
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI device sign-in")
		return
	}

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

	pending, err := h.openLoginState(req.LoginID, userID, openAIDeviceLoginMode)
	if err != nil {
		writeOpenAIOAuthTicketError(ctx, err)
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
		writeError(ctx, 502, "openai_auth_error", "OpenAI device sign-in failed: "+err.Error())
		return
	}

	authJSON, err := completeOpenAIDeviceAuth(ctx, codeResp)
	if err != nil {
		writeError(ctx, 502, "openai_auth_error", "OpenAI device sign-in failed: "+err.Error())
		return
	}
	if err := h.pkQ.UpsertAuthJSON(ctx, userID, openAIMaxPlanProvider, authJSON); err != nil {
		writeError(ctx, 500, "server_error", "OpenAI device sign-in could not be saved")
		return
	}
	// If this user owns the configured shared credential, make it available
	// immediately instead of waiting for the four-hour refresh tick.
	TriggerAdminOpenAIMaxPlanRefresh()
	writeJSON(ctx, 200, map[string]any{
		"status":  "success",
		"message": "OpenAI Max plan sign-in saved",
	})
}

func (h *OpenAIOAuthHandler) HandleBrowserStart(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	if h.stateAEAD == nil {
		writeError(ctx, 503, "openai_auth_unavailable", "OpenAI sign-in is unavailable because JWT_SECRET is not configured")
		return
	}
	verifier, challenge, err := newPKCEPair()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI browser sign-in")
		return
	}
	state, err := randomURLToken(24)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI browser sign-in")
		return
	}
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)
	loginID, err := h.sealLoginState(openAIOAuthLoginState{
		Version:      openAIOAuthTicketVersion,
		Mode:         openAIBrowserLoginMode,
		UserID:       userID,
		CodeVerifier: verifier,
		OAuthState:   state,
		RedirectURI:  openAIOAuthLocalRedirectURI,
		StartedAt:    now.Unix(),
		ExpiresAt:    expiresAt.Unix(),
	})
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to prepare OpenAI browser sign-in")
		return
	}
	writeJSON(ctx, 200, map[string]any{
		"login_id":          loginID,
		"authorization_url": openAIBrowserAuthorizationURL(challenge, state),
		"redirect_uri":      openAIOAuthLocalRedirectURI,
		"expires_at":        expiresAt.UTC().Format(time.RFC3339),
	})
}

type openAIBrowserCompleteReq struct {
	LoginID     string `json:"login_id"`
	CallbackURL string `json:"callback_url"`
}

func (h *OpenAIOAuthHandler) HandleBrowserComplete(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	var req openAIBrowserCompleteReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	pending, err := h.openLoginState(req.LoginID, userID, openAIBrowserLoginMode)
	if err != nil {
		writeOpenAIOAuthTicketError(ctx, err)
		return
	}
	code, returnedState := parseOpenAIAuthorizationInput(req.CallbackURL)
	if code == "" {
		writeError(ctx, 400, "invalid_request", "Paste the localhost callback URL or authorization code")
		return
	}
	if returnedState != "" && returnedState != pending.OAuthState {
		writeError(ctx, 400, "openai_auth_state_mismatch", "OpenAI sign-in state did not match. Start a new sign-in.")
		return
	}
	authJSON, err := completeOpenAIOAuth(ctx, code, pending.RedirectURI, pending.CodeVerifier)
	if err != nil {
		writeError(ctx, 502, "openai_auth_error", "OpenAI browser sign-in failed: "+err.Error())
		return
	}
	if err := h.pkQ.UpsertAuthJSON(ctx, userID, openAIMaxPlanProvider, authJSON); err != nil {
		writeError(ctx, 500, "server_error", "OpenAI browser sign-in could not be saved")
		return
	}
	TriggerAdminOpenAIMaxPlanRefresh()
	writeJSON(ctx, 200, map[string]any{
		"status":  "success",
		"message": "OpenAI Max plan sign-in saved",
	})
}

func openAIBrowserAuthorizationURL(challenge, state string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", openAIOAuthClientID)
	values.Set("redirect_uri", openAIOAuthLocalRedirectURI)
	values.Set("scope", openAIOAuthScope)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	values.Set("state", state)
	values.Set("id_token_add_organizations", "true")
	values.Set("codex_cli_simplified_flow", "true")
	values.Set("originator", "openpaths")
	return openAIOAuthIssuer() + "/oauth/authorize?" + values.Encode()
}

func parseOpenAIAuthorizationInput(input string) (code, state string) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		return strings.TrimSpace(parsed.Query().Get("code")), strings.TrimSpace(parsed.Query().Get("state"))
	}
	if strings.Contains(value, "#") {
		parts := strings.SplitN(value, "#", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if strings.Contains(value, "code=") {
		params, _ := url.ParseQuery(strings.TrimPrefix(value, "?"))
		return strings.TrimSpace(params.Get("code")), strings.TrimSpace(params.Get("state"))
	}
	return value, ""
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
	_ = state
	_ = code
	redirectOpenAICallback(ctx, "error", "Paste the localhost callback URL into the OpenAI sign-in panel on Account to finish.")
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
	resp, err := openAIOAuthHTTPClient.Do(req)
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
	resp, err := openAIOAuthHTTPClient.Do(req)
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
	resp, err := openAIOAuthHTTPClient.Do(req)
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
