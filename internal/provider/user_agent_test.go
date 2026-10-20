package provider

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type uaCapture struct{ ua string }

func (c *uaCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	c.ua = req.Header.Get("User-Agent")
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func TestUserAgentTransportSetsDefault(t *testing.T) {
	base := &uaCapture{}
	tr := userAgentTransport{base: base, ua: defaultUserAgent}

	req, _ := http.NewRequest(http.MethodGet, "https://example.test/", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if base.ua != defaultUserAgent {
		t.Fatalf("User-Agent = %q, want %q", base.ua, defaultUserAgent)
	}
}

func TestUserAgentTransportPreservesRequestUA(t *testing.T) {
	base := &uaCapture{}
	tr := userAgentTransport{base: base, ua: defaultUserAgent}

	req, _ := http.NewRequest(http.MethodGet, "https://example.test/", nil)
	req.Header.Set("User-Agent", "custom/9")
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if base.ua != "custom/9" {
		t.Fatalf("User-Agent = %q, want custom/9", base.ua)
	}
}
