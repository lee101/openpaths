package server

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestKeyAuthenticatedRoutesDeclareScope(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	keyChains := regexp.MustCompile(`\b(apiKeyChain|searchChain|realtimeChain|accountChain|mcpChain)\b`)
	def := regexp.MustCompile(`^\s*(apiKeyChain|searchChain|realtimeChain|accountChain|mcpChain)\s*:?=`)
	route := regexp.MustCompile(`\br\.(GET|POST|PUT|DELETE|PATCH|Handle)\(`)
	checked := 0
	for i, line := range strings.Split(string(src), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "//") || def.MatchString(line) || !keyChains.MatchString(line) {
			continue
		}
		if !route.MatchString(line) {
			t.Errorf("server.go:%d key chain used outside a single-line route registration: %s", i+1, trim)
			continue
		}
		checked++
		if !strings.Contains(line, "scoped(") {
			t.Errorf("server.go:%d route uses a key-authenticated chain without a scope: %s", i+1, trim)
		}
	}
	if checked < 50 {
		t.Fatalf("only %d key-authenticated routes found, scan is broken", checked)
	}
}
