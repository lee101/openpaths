package server

import (
	"encoding/json"
	"testing"
)

func TestDiscoveryDocsAreValidJSON(t *testing.T) {
	for name, b := range map[string][]byte{"openapi": buildOpenAPI(), "catalog": buildAPICatalog(), "card": buildMCPCard()} {
		var v map[string]any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var v map[string]any
	_ = json.Unmarshal(buildOpenAPI(), &v)
	if v["openapi"] != "3.1.0" {
		t.Fatal("version")
	}
	if len(v["paths"].(map[string]any)) < 5 {
		t.Fatal("paths")
	}
}

func TestStaticCacheControl(t *testing.T) {
	if staticCacheControl("/logo.webp", 200) == "" || staticCacheControl("/v1/x.json", 200) != "" || staticCacheControl("/docs", 200) != "" || staticCacheControl("/a.js", 404) != "" {
		t.Fatal("cache policy")
	}
	if !isDiscoveryPath("/.well-known/api-catalog") || isDiscoveryPath("/docs") {
		t.Fatal("paths")
	}
}
