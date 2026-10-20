package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunStructuredClairicAndClef(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte(`{"success":true,"result":{"answers":{"a":{"value":1}},"usage":{"input_tokens":328}}}`))
	}))
	defer srv.Close()
	t.Setenv("CLOUDFLARE_EMAIL", "")
	p := New("tok", srv.URL+"/client/v4/accounts/acct/ai/run")

	res, tokens, err := p.RunStructured(context.Background(), "@cf/cloudflare/clairic", map[string]any{"model": "clairic", "state": "x", "fields": map[string]any{"a": map[string]any{"type": "text"}}})
	if err != nil || tokens != 328 || len(res) == 0 {
		t.Fatalf("res=%s tokens=%d err=%v", res, tokens, err)
	}
	if path != "/client/v4/accounts/acct/ai/run/@cf/cloudflare/clairic" {
		t.Fatalf("path = %s", path)
	}
	if _, ok := body["model"]; ok {
		t.Fatalf("clairic body should omit model: %v", body)
	}
	if _, _, err := p.RunStructured(context.Background(), "@cf/cloudflare/clef-flash", map[string]any{"state": "x", "questions": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	if body["model"] != "clef-flash" {
		t.Fatalf("clef body model = %v", body["model"])
	}
}

func TestRunStructuredErrorsAndGlobalKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Email") != "a@b.c" || r.Header.Get("X-Auth-Key") != "gk" {
			t.Errorf("headers = %v", r.Header)
		}
		w.WriteHeader(403)
		w.Write([]byte(`{"success":false,"result":null,"errors":[{"code":10000,"message":"Authentication error"}]}`))
	}))
	defer srv.Close()
	t.Setenv("CLOUDFLARE_EMAIL", "a@b.c")
	p := New("gk", srv.URL+"/client/v4/accounts/acct/ai/run")
	if _, _, err := p.RunStructured(context.Background(), "@cf/cloudflare/clef", map[string]any{}); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := New("", "").RunStructured(context.Background(), "x", nil); err == nil {
		t.Fatal("expected unconfigured error")
	}
}
