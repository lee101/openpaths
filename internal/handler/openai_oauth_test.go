package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOpenAIOAuthLoginTicketSurvivesHandlerRestart(t *testing.T) {
	beforeRestart := NewOpenAIOAuthHandler(nil, "stable-test-secret")
	loginID, err := beforeRestart.sealLoginState(openAIOAuthLoginState{
		Version:      openAIOAuthTicketVersion,
		Mode:         openAIDeviceLoginMode,
		UserID:       "user-1",
		DeviceAuthID: "device-123",
		UserCode:     "ABCD-EFGH",
		StartedAt:    time.Now().Add(-time.Minute).Unix(),
		ExpiresAt:    time.Now().Add(14 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("sealLoginState: %v", err)
	}

	// A newly constructed handler models an API restart or another replica.
	afterRestart := NewOpenAIOAuthHandler(nil, "stable-test-secret")
	state, err := afterRestart.openLoginState(loginID, "user-1", openAIDeviceLoginMode)
	if err != nil {
		t.Fatalf("openLoginState after restart: %v", err)
	}
	if state.DeviceAuthID != "device-123" || state.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected recovered state: %+v", state)
	}
}

func TestOpenAIOAuthLoginTicketRejectsTamperingWrongUserAndExpiry(t *testing.T) {
	h := NewOpenAIOAuthHandler(nil, "stable-test-secret")
	valid := openAIOAuthLoginState{
		Version:   openAIOAuthTicketVersion,
		Mode:      openAIBrowserLoginMode,
		UserID:    "user-1",
		StartedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
	}
	loginID, err := h.sealLoginState(valid)
	if err != nil {
		t.Fatalf("sealLoginState: %v", err)
	}

	tamperAt := len(loginID) / 2
	replacement := byte('A')
	if loginID[tamperAt] == replacement {
		replacement = 'B'
	}
	tampered := loginID[:tamperAt] + string(replacement) + loginID[tamperAt+1:]
	if _, err := h.openLoginState(tampered, "user-1", openAIBrowserLoginMode); !errors.Is(err, errOpenAIOAuthTicketInvalid) {
		t.Fatalf("tampered ticket error = %v, want invalid", err)
	}
	if _, err := h.openLoginState(loginID, "user-2", openAIBrowserLoginMode); !errors.Is(err, errOpenAIOAuthTicketInvalid) {
		t.Fatalf("wrong-user ticket error = %v, want invalid", err)
	}
	if _, err := h.openLoginState(loginID, "user-1", openAIDeviceLoginMode); !errors.Is(err, errOpenAIOAuthTicketInvalid) {
		t.Fatalf("wrong-mode ticket error = %v, want invalid", err)
	}

	valid.ExpiresAt = time.Now().Add(-time.Second).Unix()
	expiredID, err := h.sealLoginState(valid)
	if err != nil {
		t.Fatalf("seal expired state: %v", err)
	}
	if _, err := h.openLoginState(expiredID, "user-1", openAIBrowserLoginMode); !errors.Is(err, errOpenAIOAuthTicketExpired) {
		t.Fatalf("expired ticket error = %v, want expired", err)
	}
}

func TestOpenAIBrowserAuthorizationURLMatchesCodexFlow(t *testing.T) {
	t.Setenv("OPENAI_OAUTH_ISSUER", "https://auth.example.test")
	parsed, err := url.Parse(openAIBrowserAuthorizationURL("challenge-123", "state-123"))
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	query := parsed.Query()
	if parsed.String() == "" || parsed.Path != "/oauth/authorize" {
		t.Fatalf("unexpected authorization URL: %s", parsed.String())
	}
	for key, want := range map[string]string{
		"client_id":                  openAIOAuthClientID,
		"redirect_uri":               openAIOAuthLocalRedirectURI,
		"code_challenge":             "challenge-123",
		"code_challenge_method":      "S256",
		"state":                      "state-123",
		"id_token_add_organizations": "true",
		"codex_cli_simplified_flow":  "true",
		"originator":                 "openpaths",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if !strings.Contains(query.Get("scope"), "offline_access") || !strings.Contains(query.Get("scope"), "api.connectors.invoke") {
		t.Fatalf("scope = %q", query.Get("scope"))
	}
}

func TestParseOpenAIAuthorizationInput(t *testing.T) {
	tests := []struct {
		input, code, state string
	}{
		{"http://localhost:1455/auth/callback?code=abc&state=state-1", "abc", "state-1"},
		{"?code=abc&state=state-1", "abc", "state-1"},
		{"abc#state-1", "abc", "state-1"},
		{"abc", "abc", ""},
	}
	for _, tt := range tests {
		code, state := parseOpenAIAuthorizationInput(tt.input)
		if code != tt.code || state != tt.state {
			t.Errorf("parseOpenAIAuthorizationInput(%q) = (%q, %q), want (%q, %q)", tt.input, code, state, tt.code, tt.state)
		}
	}
}

func TestCompleteOpenAIOAuthStoresCodexAuthJSON(t *testing.T) {
	var sawCodeExchange, sawAPIKeyExchange bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		switch values.Get("grant_type") {
		case "authorization_code":
			sawCodeExchange = true
			if values.Get("code") != "code-123" || values.Get("code_verifier") != "verifier-123" {
				t.Fatalf("unexpected code exchange values: %s", string(body))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id_token":      "id-token",
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
			})
		case "urn:ietf:params:oauth:grant-type:token-exchange":
			sawAPIKeyExchange = true
			if values.Get("subject_token") != "id-token" || values.Get("requested_token") != "openai-api-key" {
				t.Fatalf("unexpected api key exchange values: %s", string(body))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "sk-openai"})
		default:
			t.Fatalf("unexpected grant_type %q", values.Get("grant_type"))
		}
	}))
	defer srv.Close()
	t.Setenv("OPENAI_OAUTH_ISSUER", srv.URL)

	authJSON, err := completeOpenAIOAuth(t.Context(), "code-123", "https://openpaths.test/account/openai/callback", "verifier-123")
	if err != nil {
		t.Fatalf("completeOpenAIOAuth error: %v", err)
	}
	if !sawCodeExchange || !sawAPIKeyExchange {
		t.Fatalf("sawCodeExchange=%v sawAPIKeyExchange=%v, want both true", sawCodeExchange, sawAPIKeyExchange)
	}
	var auth map[string]any
	if err := json.Unmarshal([]byte(authJSON), &auth); err != nil {
		t.Fatalf("auth json: %v", err)
	}
	if auth["auth_mode"] != "chatgpt" || auth["OPENAI_API_KEY"] != "sk-openai" {
		t.Fatalf("unexpected auth root: %+v", auth)
	}
	tokens, ok := auth["tokens"].(map[string]any)
	if !ok {
		t.Fatalf("tokens missing from auth json: %+v", auth)
	}
	if tokens["refresh_token"] != "refresh-token" || tokens["id_token"] != "id-token" {
		t.Fatalf("unexpected tokens: %+v", tokens)
	}
}

func TestOpenAIDeviceAuthHelpers(t *testing.T) {
	var sawUserCode, sawPendingPoll, sawSuccessPoll bool
	pollCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/deviceauth/usercode":
			sawUserCode = true
			_ = json.NewEncoder(w).Encode(map[string]string{
				"device_auth_id": "device-123",
				"user_code":      "ABCD-EFGH",
				"interval":       "2",
			})
		case "/api/accounts/deviceauth/token":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"device_auth_id":"device-123"`) || !strings.Contains(string(body), `"user_code":"ABCD-EFGH"`) {
				t.Fatalf("unexpected device poll body: %s", string(body))
			}
			pollCount++
			if pollCount == 1 {
				sawPendingPoll = true
				w.WriteHeader(http.StatusForbidden)
				return
			}
			sawSuccessPoll = true
			_ = json.NewEncoder(w).Encode(map[string]string{
				"authorization_code": "auth-code-123",
				"code_challenge":     "challenge-123",
				"code_verifier":      "verifier-123",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	t.Setenv("OPENAI_OAUTH_ISSUER", srv.URL)

	deviceCode, err := requestOpenAIDeviceUserCode(t.Context())
	if err != nil {
		t.Fatalf("requestOpenAIDeviceUserCode error: %v", err)
	}
	if deviceCode.DeviceAuthID != "device-123" || deviceCode.UserCode != "ABCD-EFGH" || deviceCode.IntervalSeconds != 2 {
		t.Fatalf("unexpected device code: %+v", deviceCode)
	}
	if _, err := pollOpenAIDeviceToken(t.Context(), deviceCode.DeviceAuthID, deviceCode.UserCode); !errors.Is(err, errOpenAIDevicePending) {
		t.Fatalf("first poll err = %v, want pending", err)
	}
	tokenResp, err := pollOpenAIDeviceToken(t.Context(), deviceCode.DeviceAuthID, deviceCode.UserCode)
	if err != nil {
		t.Fatalf("second poll error: %v", err)
	}
	if tokenResp.AuthorizationCode != "auth-code-123" || tokenResp.CodeVerifier != "verifier-123" {
		t.Fatalf("unexpected token response: %+v", tokenResp)
	}
	if !sawUserCode || !sawPendingPoll || !sawSuccessPoll {
		t.Fatalf("sawUserCode=%v sawPendingPoll=%v sawSuccessPoll=%v", sawUserCode, sawPendingPoll, sawSuccessPoll)
	}
}

func TestCompleteOpenAIDeviceAuthUsesDeviceCallback(t *testing.T) {
	var sawCodeExchange, sawAPIKeyExchange bool
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse form: %v", err)
		}
		switch values.Get("grant_type") {
		case "authorization_code":
			sawCodeExchange = true
			if values.Get("code") != "device-code-123" || values.Get("code_verifier") != "device-verifier-123" {
				t.Fatalf("unexpected code exchange values: %s", string(body))
			}
			if values.Get("redirect_uri") != srv.URL+"/deviceauth/callback" {
				t.Fatalf("redirect_uri = %q", values.Get("redirect_uri"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id_token":      "id-token",
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
			})
		case "urn:ietf:params:oauth:grant-type:token-exchange":
			sawAPIKeyExchange = true
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "sk-openai"})
		default:
			t.Fatalf("unexpected grant_type %q", values.Get("grant_type"))
		}
	}))
	defer srv.Close()
	t.Setenv("OPENAI_OAUTH_ISSUER", srv.URL)

	authJSON, err := completeOpenAIDeviceAuth(t.Context(), &openAIDeviceTokenResp{
		AuthorizationCode: "device-code-123",
		CodeVerifier:      "device-verifier-123",
	})
	if err != nil {
		t.Fatalf("completeOpenAIDeviceAuth error: %v", err)
	}
	if !sawCodeExchange || !sawAPIKeyExchange {
		t.Fatalf("sawCodeExchange=%v sawAPIKeyExchange=%v, want both true", sawCodeExchange, sawAPIKeyExchange)
	}
	var auth map[string]any
	if err := json.Unmarshal([]byte(authJSON), &auth); err != nil {
		t.Fatalf("auth json: %v", err)
	}
	if auth["OPENAI_API_KEY"] != "sk-openai" {
		t.Fatalf("unexpected auth: %+v", auth)
	}
}
