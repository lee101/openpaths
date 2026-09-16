package handler

import (
	fws "github.com/fasthttp/websocket"
	gws "github.com/gorilla/websocket"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRealtimeUsage(t *testing.T) {
	payload := []byte(`{
  "type":"response.done",
  "response":{"id":"resp_1","usage":{
    "input_tokens":3100,"output_tokens":3500,
    "input_token_details":{"text_tokens":1000,"audio_tokens":2000,"image_tokens":100,"cached_tokens":700,
      "cached_tokens_details":{"text_tokens":400,"audio_tokens":250,"image_tokens":50}},
    "output_token_details":{"text_tokens":500,"audio_tokens":3000}
  }}
}`)
	id, usage, completed, err := parseRealtimeUsage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !completed || id != "resp_1" {
		t.Fatalf("completed = %v, id = %q", completed, id)
	}
	if usage.TextInputTokens != 1000 || usage.CachedTextInputTokens != 400 ||
		usage.AudioInputTokens != 2000 || usage.CachedAudioInputTokens != 250 ||
		usage.ImageInputTokens != 100 || usage.CachedImageInputTokens != 50 ||
		usage.TextOutputTokens != 500 || usage.AudioOutputTokens != 3000 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestParseRealtimeUsageFallbacks(t *testing.T) {
	payload := []byte(`{"type":"response.done","response":{"id":"resp_2","usage":{"input_tokens":100,"output_tokens":20,"input_token_details":{"cached_tokens":30},"output_token_details":{}}}}`)
	_, usage, completed, err := parseRealtimeUsage(payload)
	if err != nil || !completed {
		t.Fatalf("completed = %v, err = %v", completed, err)
	}
	if usage.TextInputTokens != 100 || usage.CachedTextInputTokens != 30 || usage.TextOutputTokens != 20 {
		t.Fatalf("unexpected fallback usage: %+v", usage)
	}
}

func TestParseRealtimeUsageRejectsMissingUsage(t *testing.T) {
	_, _, _, err := parseRealtimeUsage([]byte(`{"type":"response.done","response":{"id":"resp_3"}}`))
	if err == nil {
		t.Fatal("expected missing usage to fail")
	}
}

func TestMakeRealtimeURL(t *testing.T) {
	got, err := makeRealtimeURL("https://api.openai.com", "gpt-realtime-2.1-mini")
	if err != nil {
		t.Fatal(err)
	}
	want := "wss://api.openai.com/v1/realtime?model=gpt-realtime-2.1-mini"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestValidateRealtimeModels(t *testing.T) {
	google := &model.ModelConfig{ID: "gemini-3.8-live-extended-thinking", Provider: "google", ProviderModelID: "gemini-3.8-live-extended-thinking"}
	openai := &model.ModelConfig{ID: "gpt-realtime-2.1-mini", Provider: "openai", ProviderModelID: "gpt-realtime-2.1-mini"}
	for _, tc := range []struct {
		name, payload string
		cfg           *model.ModelConfig
		first, valid  bool
	}{
		{"gemini setup", `{"setup":{"model":"models/gemini-3.8-live-extended-thinking"}}`, google, true, true},
		{"gemini mismatch", `{"setup":{"model":"models/gemini-3.8-flash"}}`, google, true, false},
		{"gemini requires setup", `{"realtimeInput":{"audio":{}}}`, google, true, false},
		{"gemini second setup", `{"setup":{"model":"models/gemini-3.8-live-extended-thinking"}}`, google, false, false},
		{"gemini unmetered search", `{"setup":{"model":"models/gemini-3.8-live-extended-thinking","tools":[{"googleSearch":{}}]}}`, google, true, false},
		{"openai setup", `{"type":"session.update","session":{"type":"realtime"}}`, openai, true, true},
		{"openai later model switch", `{"type":"session.update","session":{"model":"gpt-realtime-2.1"}}`, openai, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRealtimeClientMessage([]byte(tc.payload), tc.cfg, tc.first)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
	if supportedRealtimeModel(&model.ModelConfig{Provider: "google", ProviderModelID: "gemini-3.8-flash"}) {
		t.Fatal("non-live model accepted")
	}
}

func TestGeminiUsageModalities(t *testing.T) {
	usage, ok := parseGeminiUsage([]byte(`{"usageMetadata":{"promptTokenCount":350,"responseTokenCount":90,"thoughtsTokenCount":20,"promptTokensDetails":[{"modality":"TEXT","tokenCount":100},{"modality":"AUDIO","tokenCount":200},{"modality":"VIDEO","tokenCount":30}],"responseTokensDetails":[{"modality":"TEXT","tokenCount":10},{"modality":"AUDIO","tokenCount":80}]}}`))
	if !ok || usage.TextInputTokens != 120 || usage.AudioInputTokens != 200 || usage.ImageInputTokens != 30 || usage.TextOutputTokens != 30 || usage.AudioOutputTokens != 80 {
		t.Fatalf("incorrect modality billing: %+v, present=%v", usage, ok)
	}
	if _, ok := parseGeminiUsage([]byte(`{"serverContent":{"turnComplete":true}}`)); ok {
		t.Fatal("turn completion without usage billed")
	}
	url, err := makeGeminiRealtimeURL("https://generativelanguage.googleapis.com")
	if err != nil || url != "wss://generativelanguage.googleapis.com"+geminiRealtimePath {
		t.Fatalf("URL=%q err=%v", url, err)
	}
}

func TestRealtimeOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin string
		valid  bool
	}{{"", true}, {"https://openpaths.io", true}, {"https://evil.example", false}, {"null", false}, {"https://openpaths.io.evil.example", false}} {
		var ctx fasthttp.RequestCtx
		ctx.Request.SetRequestURI("https://openpaths.io/v1/realtime")
		ctx.Request.Header.Set("Origin", tc.origin)
		if realtimeSameOrigin(&ctx) != tc.valid {
			t.Errorf("origin %q accepted=%v", tc.origin, !tc.valid)
		}
	}
}

func TestRealtimeRelayStreamsAndStops(t *testing.T) {
	for _, provider := range []string{"openai", "google"} {
		t.Run(provider, func(t *testing.T) {
			upstreamClosed := make(chan struct{})
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				u := gws.Upgrader{}
				ws, err := u.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer ws.Close()
				defer close(upstreamClosed)
				if _, _, err = ws.ReadMessage(); err != nil {
					return
				}
				_ = ws.WriteMessage(gws.TextMessage, []byte(`{"audio":"streamed"}`))
				for {
					if _, _, err = ws.ReadMessage(); err != nil {
						return
					}
				}
			}))
			defer up.Close()
			cfg := &model.ModelConfig{ID: "gpt-realtime-2.1-mini", Provider: provider, ProviderModelID: "gpt-realtime-2.1-mini"}
			setup := `{"type":"session.update","session":{"type":"realtime"}}`
			if provider == "google" {
				cfg.ID = "gemini-3.8-live-extended-thinking"
				cfg.ProviderModelID = cfg.ID
				setup = `{"setup":{"model":"models/gemini-3.8-live-extended-thinking"}}`
			}
			h := NewRealtimeHandler(nil, nil, nil, nil)
			relayStopped := make(chan struct{})
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			server := &fasthttp.Server{DisableHeaderNamesNormalizing: true, Handler: func(ctx *fasthttp.RequestCtx) {
				if middleware.RealtimeCredentialFromSubprotocol(ctx) != "testsecret" {
					t.Error("handshake credential missing")
				}

				_ = h.upgrader.Upgrade(ctx, func(client *fws.Conn) {
					defer close(relayStopped)
					h.relay(client, "ws"+strings.TrimPrefix(up.URL, "http"), "test-key", "user", "", cfg, true, requestApp{}, time.Now())
				})
			}}
			go server.Serve(listener)
			defer server.Shutdown()
			dialer := gws.Dialer{Subprotocols: []string{"openpaths-realtime", "openpaths-api-key.testsecret"}}
			ws, _, err := dialer.Dial("ws://"+listener.Addr().String(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if ws.Subprotocol() != "openpaths-realtime" {
				t.Fatal("credential subprotocol leaked or protocol not negotiated")
			}
			_ = ws.WriteMessage(gws.TextMessage, []byte(setup))
			_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, payload, err := ws.ReadMessage()
			if err != nil || string(payload) != `{"audio":"streamed"}` {
				t.Fatalf("payload=%s err=%v", payload, err)
			}
			ws.Close()
			select {
			case <-relayStopped:
			case <-time.After(3 * time.Second):
				t.Fatal("relay did not release both pumps")
			}
			select {
			case <-upstreamClosed:
			case <-time.After(3 * time.Second):
				t.Fatal("provider session left open")
			}
		})
	}
}
