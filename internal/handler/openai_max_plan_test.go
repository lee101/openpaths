package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
		case "application/json":
			refreshSeen = true
			var req map[string]string
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("refresh json: %v", err)
			}
			if req["grant_type"] != "refresh_token" || req["refresh_token"] != "refresh-old" {
				t.Fatalf("unexpected refresh request: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id_token":      "id-new",
				"access_token":  "access-new",
				"refresh_token": "refresh-new",
			})
		case "application/x-www-form-urlencoded":
			exchangeSeen = true
			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("exchange form: %v", err)
			}
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
}

func TestParseOpenAIMaxPlanCredentialsDedupes(t *testing.T) {
	raw := `[{"OPENAI_API_KEY":"sk-one"},{"OPENAI_API_KEY":"sk-one"}]`
	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 1 {
		t.Fatalf("len(creds) = %d, want 1", len(creds))
	}
}
