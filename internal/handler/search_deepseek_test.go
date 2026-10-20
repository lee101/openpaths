package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/valyala/fasthttp"
)

func runSearch(t *testing.T, h *SearchHandler, body string) (int, map[string]any) {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/search")
	req.Header.SetContentType("application/json")
	req.SetBodyString(body)
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "u1")
	h.HandleSearch(ctx)
	var out map[string]any
	_ = json.Unmarshal(ctx.Response.Body(), &out)
	return ctx.Response.StatusCode(), out
}

func newDeepSeekTestHandler(base, key string) *SearchHandler {
	return NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour))).
		SetDeepSeekProvider(SearchProviderConfig{BaseURL: base, APIKey: key, Enabled: true})
}

func TestDeepSeekSearchProvider(t *testing.T) {
	var got map[string]any
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tools/search" || r.Header.Get("Authorization") != "Bearer k" {
			t.Fatalf("bad request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"results":[{"title":"T","url":"https://a.test","snippet":"s","summary":"sum","content":"","date":"2026-06-01T00:00:00+00:00"}]}`))
	}))
	defer up.Close()
	st, out := runSearch(t, newDeepSeekTestHandler(up.URL, "k"), `{"provider":"deepseek","query":"q","numResults":3,"need_content":true}`)
	if st != 200 || got["count"] != float64(3) || got["need_content"] != true {
		t.Fatalf("status=%d upstream=%v out=%v", st, got, out)
	}
	rs := out["results"].([]any)
	if len(rs) != 1 || rs[0].(map[string]any)["text"] != "sum" {
		t.Fatalf("results = %v", rs)
	}
}

func TestOpenPathsAgentSearch(t *testing.T) {
	round := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/tools/search":
			_, _ = w.Write([]byte(`{"results":[{"title":"Src","url":"https://src.test","summary":"fact"}]}`))
		case "/chat/completions":
			round++
			if round == 1 {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"c1","function":{"name":"web_search","arguments":"{\"query\":\"x\"}"}}]}}]}`))
				return
			}
			var body struct {
				Messages []map[string]any `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			last := body.Messages[len(body.Messages)-1]
			if last["role"] != "tool" || !strings.Contains(last["content"].(string), "https://src.test") {
				t.Fatalf("tool result not fed back: %v", last)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"answer [1]"}}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer up.Close()
	st, out := runSearch(t, newDeepSeekTestHandler(up.URL, "k"), `{"provider":"openpaths","query":"q"}`)
	if st != 200 || out["answer"] != "answer [1]" || len(out["citations"].([]any)) != 1 {
		t.Fatalf("status=%d out=%v", st, out)
	}
}

// Live check against the real DeepSeek API; run with DEEPSEEK_API_KEY set.
func TestOpenPathsAgentSearchLive(t *testing.T) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" || os.Getenv("LIVE_SEARCH") == "" {
		t.Skip("set LIVE_SEARCH=1 and DEEPSEEK_API_KEY")
	}
	h := newDeepSeekTestHandler("https://api.deepseek.com", key)
	st, out := runSearch(t, h, `{"provider":"deepseek","query":"DeepSeek V4 release","numResults":3}`)
	if st != 200 || len(out["results"].([]any)) == 0 {
		t.Fatalf("deepseek: %d %v", st, out)
	}
	st, out = runSearch(t, h, `{"provider":"openpaths","query":"What is DeepSeek V4 and when was it released?"}`)
	t.Logf("openpaths: %d answer=%v citations=%d queries=%v", st, out["answer"], len(out["citations"].([]any)), out["searchQueries"])
	if st != 200 || out["answer"] == "" {
		t.Fatalf("openpaths: %d %v", st, out)
	}
}
