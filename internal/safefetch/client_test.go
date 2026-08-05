package safefetch

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestPublicIPRejectsNonPublicRanges(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "::1", "fc00::1", "fe80::1"} {
		if PublicIP(net.ParseIP(raw)) {
			t.Errorf("PublicIP(%s) = true", raw)
		}
	}
	if !PublicIP(net.ParseIP("8.8.8.8")) || !PublicIP(net.ParseIP("2606:4700:4700::1111")) {
		t.Fatal("expected public resolver addresses to be allowed")
	}
}

func TestValidateURLRejectsUnsafeLiteralAndCredentials(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "http://127.0.0.1/a", "http://user:pass@example.com/a"} {
		u, _ := url.Parse(raw)
		if ValidateURL(u) == nil {
			t.Errorf("ValidateURL(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestClientBlocksLoopbackBeforeRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := NewClient(time.Second).Do(req)
	if err == nil {
		t.Fatal("loopback request unexpectedly succeeded")
	}
	if called {
		t.Fatal("unsafe request reached the loopback server")
	}
}
