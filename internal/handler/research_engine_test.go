package handler

import (
	"encoding/json"
	"testing"
)

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain", `{"a":1}`, `{"a":1}`},
		{"fenced", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"prose", "Sure, here it is:\n{\"a\":1}\nThanks", `{"a":1}`},
		{"nested", `{"a":{"b":2},"c":3}`, `{"a":{"b":2},"c":3}`},
		{"brace-in-string", `{"a":"x{y}z"}`, `{"a":"x{y}z"}`},
		{"none", `no json here`, ``},
	}
	for _, c := range cases {
		if got := extractJSONObject(c.in); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestDecodeJSONRepairsTrailingComma(t *testing.T) {
	var v struct {
		Queries []string `json:"queries"`
	}
	if err := decodeJSON("```json\n{\"queries\":[\"a\",\"b\",]}\n```", &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(v.Queries) != 2 || v.Queries[0] != "a" {
		t.Fatalf("got %+v", v)
	}
}

func TestNormalizeURL(t *testing.T) {
	for _, p := range [][2]string{
		{"https://www.Example.com/path/", "example.com/path"},
		{"http://example.com", "example.com"},
		{"https://example.com/", "example.com"},
	} {
		if got := normalizeURL(p[0]); got != p[1] {
			t.Errorf("normalizeURL(%q)=%q want %q", p[0], got, p[1])
		}
	}
}

func TestGateBySourceOnly(t *testing.T) {
	src := map[string]ResearchSource{normalizeURL("https://a.com"): {URL: "https://a.com"}}
	claims := []researchClaim{
		{Claim: "kept", Source: "https://a.com"},
		{Claim: "dropped", Source: "https://b.com"},
	}
	kept := gateBySourceOnly(claims, src)
	if len(kept) != 1 || kept[0].Claim != "kept" {
		t.Fatalf("gate kept %+v", kept)
	}
}

func TestBuildCitationsNumbersUniqueSources(t *testing.T) {
	sources := []ResearchSource{
		{URL: "https://a.com", Title: "A"},
		{URL: "https://b.com", Title: "B"},
	}
	claims := []researchClaim{
		{Claim: "c1", Source: "https://a.com"},
		{Claim: "c2", Source: "https://a.com"},
		{Claim: "c3", Source: "https://b.com"},
	}
	cites, idx := buildCitations(claims, sources)
	if len(cites) != 2 {
		t.Fatalf("want 2 citations, got %d", len(cites))
	}
	if idx[normalizeURL("https://a.com")] != 1 || idx[normalizeURL("https://b.com")] != 2 {
		t.Fatalf("bad index map %+v", idx)
	}
}

func TestParsePapersResultsNestedKeys(t *testing.T) {
	raw := []byte(`{"papers":[{"title":"T","arxiv_url":"https://arxiv.org/abs/1","abstract":"sum"}]}`)
	got := parsePapersResults(raw)
	if len(got) != 1 || got[0].URL != "https://arxiv.org/abs/1" || got[0].Provider != "papers" {
		t.Fatalf("parsed %+v", got)
	}
}

func TestContentToStringHandlesParts(t *testing.T) {
	var parts any
	_ = json.Unmarshal([]byte(`[{"type":"text","text":"hello "},{"type":"text","text":"world"}]`), &parts)
	if got := contentToString(parts); got != "hello world" {
		t.Fatalf("got %q", got)
	}
	if got := contentToString("plain"); got != "plain" {
		t.Fatalf("got %q", got)
	}
}

func TestStripHTML(t *testing.T) {
	in := `<html><body><script>var x=1;</script><p>Hello <b>World</b></p></body></html>`
	got := stripHTML(in)
	if got == "" || !contains(got, "Hello") || contains(got, "<p>") || contains(got, "var x") {
		t.Fatalf("stripHTML=%q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
