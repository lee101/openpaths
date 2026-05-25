package queries

import "testing"

func TestNormalizeAppURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "https URL", raw: " https://example.com/app?x=1#token ", want: "https://example.com/app?x=1"},
		{name: "http URL", raw: "http://localhost:3000", want: "http://localhost:3000"},
		{name: "missing host", raw: "https:///no-host", want: ""},
		{name: "javascript URL rejected", raw: "javascript:alert(1)", want: ""},
		{name: "relative URL rejected", raw: "/apps/client", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAppURL(tt.raw); got != tt.want {
				t.Fatalf("NormalizeAppURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSlugForApp(t *testing.T) {
	tests := []struct {
		name string
		app  string
		url  string
		want string
	}{
		{name: "uses name", app: "Hermes Agent", url: "https://nousresearch.com", want: "hermes-agent"},
		{name: "uses host from URL", app: "", url: "https://www.example.com/app", want: "www-example-com"},
		{name: "strips punctuation", app: "Dify.AI", url: "https://dify.ai", want: "dify-ai"},
		{name: "fallback", app: "", url: "", want: "app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlugForApp(tt.app, tt.url); got != tt.want {
				t.Fatalf("SlugForApp(%q, %q) = %q, want %q", tt.app, tt.url, got, tt.want)
			}
		})
	}
}

func TestFaviconURLRejectsUnsafeURL(t *testing.T) {
	if got := FaviconURL("javascript:alert(1)"); got != "" {
		t.Fatalf("FaviconURL returned %q for unsafe URL, want empty", got)
	}
}
