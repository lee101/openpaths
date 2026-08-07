package handler

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseOpenAIMaxPlanCredentials(t *testing.T) {
	raw := `{
		"credentials": [
			{"auth_mode": "chatgpt", "OPENAI_API_KEY": "sk-one"},
			{"api_key": "sk-two"},
			{"token": "sk-three"},
			"sk-four"
		]
	}`

	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 4 {
		t.Fatalf("len(creds) = %d, want 4", len(creds))
	}
	got := map[string]bool{}
	for _, cred := range creds {
		got[cred.APIKey] = true
		if cred.ID == "" || cred.Label == "" {
			t.Fatalf("credential missing stable metadata: %+v", cred)
		}
		if strings.Contains(cred.Label, cred.ID[:8]) {
			t.Fatalf("credential label leaks stable id: %q", cred.Label)
		}
	}
	for _, want := range []string{"sk-one", "sk-two", "sk-three", "sk-four"} {
		if !got[want] {
			t.Fatalf("missing credential %q in %+v", want, got)
		}
	}
}

func TestParseOpenAIMaxPlanCredentialsIncludesRefreshToken(t *testing.T) {
	raw := `{
		"auth_mode": "chatgpt",
		"OPENAI_API_KEY": "sk-one",
		"tokens": {
			"refresh_token": "refresh-one",
			"id_token": {"email": "user@example.com"}
		}
	}`

	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 1 {
		t.Fatalf("len(creds) = %d, want 1", len(creds))
	}
	if creds[0].RefreshToken != "refresh-one" {
		t.Fatalf("RefreshToken = %q, want refresh-one", creds[0].RefreshToken)
	}
	if creds[0].AuthJSON == "" {
		t.Fatalf("AuthJSON was not retained")
	}
	if got, want := creds[0].Label, "OpenAI Codex sign-in for user@example.com"; got != want {
		t.Fatalf("Label = %q, want %q", got, want)
	}
}

func TestRefreshOpenAIMaxPlanAuthJSONRefreshesAndExchangesAPIKey(t *testing.T) {
	var refreshSeen, exchangeSeen bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.Header.Get("Content-Type") {
		case "application/x-www-form-urlencoded":
			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("token form: %v", err)
			}
			if values.Get("grant_type") == "refresh_token" {
				refreshSeen = true
				if values.Get("refresh_token") != "refresh-old" || values.Get("client_id") != openAIOAuthClientID {
					t.Fatalf("unexpected refresh request: %s", string(body))
				}
				_ = json.NewEncoder(w).Encode(map[string]string{
					"id_token":      "id-new",
					"access_token":  "access-new",
					"refresh_token": "refresh-new",
				})
				return
			}
			exchangeSeen = true
			if values.Get("subject_token") != "id-new" || values.Get("requested_token") != "openai-api-key" {
				t.Fatalf("unexpected exchange request: %s", string(body))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "sk-new"})
		default:
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
	}))
	defer srv.Close()
	t.Setenv("CODEX_REFRESH_TOKEN_URL_OVERRIDE", srv.URL)

	raw := `{"auth_mode":"chatgpt","OPENAI_API_KEY":"sk-old","tokens":{"id_token":"id-old","access_token":"access-old","refresh_token":"refresh-old"}}`
	refreshed, err := refreshOpenAIMaxPlanAuthJSON(t.Context(), raw, "refresh-old")
	if err != nil {
		t.Fatalf("refreshOpenAIMaxPlanAuthJSON error: %v", err)
	}
	if !refreshSeen || !exchangeSeen {
		t.Fatalf("refreshSeen=%v exchangeSeen=%v, want both true", refreshSeen, exchangeSeen)
	}
	if refreshed.APIKey != "sk-new" || refreshed.RefreshToken != "refresh-new" {
		t.Fatalf("unexpected refreshed result: %+v", refreshed)
	}
	var updated map[string]any
	if err := json.Unmarshal([]byte(refreshed.AuthJSON), &updated); err != nil {
		t.Fatalf("updated auth json: %v", err)
	}
	if updated["OPENAI_API_KEY"] != "sk-new" {
		t.Fatalf("OPENAI_API_KEY = %v, want sk-new", updated["OPENAI_API_KEY"])
	}
	tokens := updated["tokens"].(map[string]any)
	if tokens["refresh_token"] != "refresh-new" || tokens["access_token"] != "access-new" {
		t.Fatalf("rotated tokens were not persisted: %+v", tokens)
	}
}

func TestOpenAIAuthNeedsRefreshUsesAccessTokenExpiry(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	jwt := func(exp time.Time) string {
		payload, _ := json.Marshal(map[string]int64{"exp": exp.Unix()})
		return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	}
	auth := func(accessToken, lastRefresh string) string {
		encoded, _ := json.Marshal(map[string]any{
			"OPENAI_API_KEY": "sk-test",
			"tokens": map[string]any{
				"access_token":  accessToken,
				"refresh_token": "refresh-test",
			},
			"last_refresh": lastRefresh,
		})
		return string(encoded)
	}

	if openAIAuthNeedsRefresh(auth(jwt(now.Add(time.Hour)), now.Add(-24*time.Hour).Format(time.RFC3339)), now) {
		t.Fatal("fresh JWT should take precedence over stale last_refresh")
	}
	if !openAIAuthNeedsRefresh(auth(jwt(now.Add(5*time.Minute)), now.Format(time.RFC3339)), now) {
		t.Fatal("JWT inside the refresh window should refresh")
	}
	if openAIAuthNeedsRefresh(auth("opaque", now.Add(-time.Hour).Format(time.RFC3339)), now) {
		t.Fatal("recent last_refresh should keep an opaque token fresh")
	}
	if !openAIAuthNeedsRefresh(auth("opaque", now.Add(-4*time.Hour).Format(time.RFC3339)), now) {
		t.Fatal("stale opaque token should refresh")
	}
	if !openAIAuthNeedsRefresh(auth("opaque", ""), now) {
		t.Fatal("legacy auth without expiry metadata should refresh once")
	}
	if openAIAuthNeedsRefresh(`{"OPENAI_API_KEY":"sk-test","tokens":{"access_token":"opaque"}}`, now) {
		t.Fatal("auth without a refresh token cannot refresh")
	}
}

func TestUpdateOpenAIMaxPlanAuthJSONKeepsRefreshTokenWhenNotRotated(t *testing.T) {
	raw := `{"OPENAI_API_KEY":"sk-old","tokens":{"refresh_token":"refresh-old"}}`
	updated, err := updateOpenAIMaxPlanAuthJSON(raw, "sk-new", "id-new", "access-new", "")
	if err != nil {
		t.Fatalf("updateOpenAIMaxPlanAuthJSON: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(updated), &parsed); err != nil {
		t.Fatalf("updated auth JSON: %v", err)
	}
	if got := parsed["tokens"].(map[string]any)["refresh_token"]; got != "refresh-old" {
		t.Fatalf("refresh_token = %v, want refresh-old", got)
	}
}

func TestParseOpenAIMaxPlanCredentialsDedupes(t *testing.T) {
	raw := `[{"OPENAI_API_KEY":"sk-one"},{"OPENAI_API_KEY":"sk-one"}]`
	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 1 {
		t.Fatalf("len(creds) = %d, want 1", len(creds))
	}
}
