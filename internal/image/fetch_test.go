package image

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchImageBoundedAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte("abcd"))
		case "/big":
			w.Write([]byte(strings.Repeat("x", 100)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := maxFetchImageBytes
	maxFetchImageBytes = 10
	defer func() { maxFetchImageBytes = old }()
	if b, err := FetchImage(srv.URL + "/ok"); err != nil || string(b) != "abcd" {
		t.Fatalf("ok: %q %v", b, err)
	}
	if _, err := FetchImage(srv.URL + "/big"); err == nil {
		t.Fatal("oversize accepted")
	}
	if _, err := FetchImage(srv.URL + "/missing"); err == nil {
		t.Fatal("404 accepted")
	}
}
