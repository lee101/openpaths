package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

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
