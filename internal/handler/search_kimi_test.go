package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/metrics"
)

func newKimiTestHandler(base string) *SearchHandler {
	return NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour))).
		SetKimiProvider(SearchProviderConfig{BaseURL: base, APIKey: "k", Enabled: true})
}

func TestKimiSearchBasicAndPro(t *testing.T) {
	var got map[string]any
	var path string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		got = nil
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"search_results":[{"title":"T","url":"https://a.test","snippet":"s","text":"","chunks":[{"text":"c1","score":1},{"text":"c2","score":0.5}]}]}`))
	}))
	defer up.Close()
	h := newKimiTestHandler(up.URL)

	st, out := runSearch(t, h, `{"provider":"kimi","query":"q","numResults":50,"include_content":true}`)
	if st != 200 || path != "/v1/tools/search" || got["limit"] != float64(20) || got["include_content"] != true || got["text_query"] != "q" {
		t.Fatalf("basic status=%d path=%s upstream=%v out=%v", st, path, got, out)
	}

	st, out = runSearch(t, h, `{"provider":"kimi-pro","query":"q","includeDomains":["a.com","b.com","c.com","d.com","e.com","f.com"],"startPublishedDate":"2026-01-01T00:00:00.000Z"}`)
	sites, _ := got["sites"].([]any)
	tw, _ := got["time_window"].(map[string]any)
	if st != 200 || path != "/v1/tools/search_pro" || len(sites) != 5 || tw["start"] != "2026-01-01" {
		t.Fatalf("pro status=%d path=%s upstream=%v out=%v", st, path, got, out)
	}
	rs := out["results"].([]any)
	if rs[0].(map[string]any)["text"] != "c1\n\nc2" {
		t.Fatalf("chunks not joined: %v", rs)
	}
}

func TestKimiFetch(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tools/fetch" {
			t.Fatalf("path %s", r.URL.Path)
		}
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["url"] == "https://empty.test" {
			_, _ = w.Write([]byte(`{"url":"https://empty.test","markdown":"  ","title":""}`))
			return
		}
		_, _ = w.Write([]byte(`{"url":"https://a.test","markdown":"# Hi","title":"A"}`))
	}))
	defer up.Close()
	h := newKimiTestHandler(up.URL)
	st, out := runSearch(t, h, `{"provider":"kimi-fetch","query":"https://a.test"}`)
	if st != 200 || out["answer"] != "# Hi" {
		t.Fatalf("status=%d out=%v", st, out)
	}
	if st, _ = runSearch(t, h, `{"provider":"kimi-fetch","query":"https://empty.test"}`); st != 404 {
		t.Fatalf("blank markdown should 404, got %d", st)
	}
}
