package appnz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Every call app.nz makes on behalf of the gateway is credit-metered, so it must
// carry the paid tier. Free ($0) models are never routed to this provider
// (see internal/config/free_models_guard_test.go).
func TestEveryRequestCarriesPaidTier(t *testing.T) {
	var tier string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tier = r.Header.Get("X-Omniserve-Tier")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	p := New("k", srv.URL)
	if _, err := p.do(context.Background(), p.client, "GET", srv.URL+"/x", "k", nil); err != nil {
		t.Fatal(err)
	}
	if tier != "paid" {
		t.Fatalf("X-Omniserve-Tier = %q, want paid", tier)
	}
}
