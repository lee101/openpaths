package openrouter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	oai "github.com/openpaths/openpaths/internal/provider/openai"
)

const okBody = `{"id":"c","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`

type capture struct {
	header http.Header
	body   string
}

func captureServer(c *capture, stream bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.header = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		c.body = string(b)
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, okBody)
	}))
}

func chatReq(session string) *model.ChatCompletionRequest {
	return &model.ChatCompletionRequest{
		Model:     "anthropic/claude-sonnet-5",
		Messages:  []model.ChatMessage{{Role: "user", Content: "hi"}},
		SessionID: session,
	}
}

func TestOpenRouterForwardsSessionHeaderNonStream(t *testing.T) {
	var c capture
	srv := captureServer(&c, false)
	defer srv.Close()
	if _, err := New("k", srv.URL).ChatCompletion(t.Context(), chatReq("sess-1")); err != nil {
		t.Fatal(err)
	}
	if got := c.header.Get("x-session-id"); got != "sess-1" {
		t.Fatalf("x-session-id = %q", got)
	}
	if strings.Contains(c.body, "session") {
		t.Fatalf("session leaked into body: %s", c.body)
	}
}

func TestOpenRouterForwardsSessionHeaderStream(t *testing.T) {
	var c capture
	srv := captureServer(&c, true)
	defer srv.Close()
	ch, err := New("k", srv.URL).ChatCompletionStream(t.Context(), chatReq("sess-2"))
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	if got := c.header.Get("x-session-id"); got != "sess-2" {
		t.Fatalf("x-session-id = %q", got)
	}
	if strings.Contains(c.body, "session") {
		t.Fatalf("session leaked into body: %s", c.body)
	}
}

func TestOpenRouterOmitsSessionHeaderWhenUnset(t *testing.T) {
	var c capture
	srv := captureServer(&c, false)
	defer srv.Close()
	if _, err := New("k", srv.URL).ChatCompletion(t.Context(), chatReq("")); err != nil {
		t.Fatal(err)
	}
	if _, has := c.header["X-Session-Id"]; has {
		t.Fatal("header set without session")
	}
}

func TestOtherProvidersNeverSeeSessionID(t *testing.T) {
	for name, build := range map[string]func(url string) *oai.OpenAIProvider{
		"openai": func(u string) *oai.OpenAIProvider { return oai.New("k", u) },
		"compat": func(u string) *oai.OpenAIProvider { return oai.NewCompatible("deepseek", "k", u, nil) },
	} {
		for _, stream := range []bool{false, true} {
			var c capture
			srv := captureServer(&c, stream)
			p := build(srv.URL)
			if stream {
				ch, err := p.ChatCompletionStream(t.Context(), chatReq("sess-3"))
				if err != nil {
					t.Fatal(err)
				}
				for range ch {
				}
			} else if _, err := p.ChatCompletion(t.Context(), chatReq("sess-3")); err != nil {
				t.Fatal(err)
			}
			srv.Close()
			if _, has := c.header["X-Session-Id"]; has {
				t.Fatalf("%s stream=%v: header leaked", name, stream)
			}
			if strings.Contains(c.body, "sess-3") || strings.Contains(c.body, "session") {
				t.Fatalf("%s stream=%v: body leaked: %s", name, stream, c.body)
			}
		}
	}
}
