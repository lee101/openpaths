package appnz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func newLocalServer(t *testing.T, health string, audio []byte) (*httptest.Server, *map[string]any) {
	t.Helper()
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(health))
		case "/v1/audio/speech":
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Errorf("auth header = %q", got)
			}
			json.NewDecoder(r.Body).Decode(&captured)
			w.Write(audio)
		default:
			w.WriteHeader(404)
		}
	}))
	return srv, &captured
}

func TestGenerateSpeechLocal(t *testing.T) {
	srv, captured := newLocalServer(t, `{"busy":false,"queue_depth":0}`, []byte("AUDIOBYTES"))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Model: "cosyvoice3-0.5b", Input: "hello world", VoiceID: "clone-abc", Speed: 1.2, Language: "en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Audio != base64.StdEncoding.EncodeToString([]byte("AUDIOBYTES")) {
		t.Errorf("audio = %q", resp.Audio)
	}
	if resp.Format != "mp3" || resp.Characters != len("hello world") {
		t.Errorf("format=%q chars=%d", resp.Format, resp.Characters)
	}
	c := *captured
	if c["model"] != "cosyvoice3-0.5b" || c["input"] != "hello world" || c["voice"] != "clone-abc" || c["language"] != "en" || c["response_format"] != "mp3" {
		t.Errorf("payload = %v", c)
	}
}

func TestGenerateSpeechOffloadsToRunpod(t *testing.T) {
	srv, _ := newLocalServer(t, `{"busy":true,"queue_depth":9}`, []byte("LOCAL"))
	defer srv.Close()

	rp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/ep1/runsync" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer rp-key" {
			t.Errorf("auth header = %q", got)
		}
		var body map[string]map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["input"]["voice"] != "narrator" {
			t.Errorf("input = %v", body["input"])
		}
		json.NewEncoder(w).Encode(runpodJob{ID: "j1", Status: "COMPLETED", Output: &runpodOutput{AudioB64: "UlVOUE9E", Format: "mp3"}})
	}))
	defer rp.Close()

	p := New("test-key", srv.URL, WithRunpod(rp.URL, "ep1", "rp-key"))
	p.limiter = newLimiter(30)
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Model: "cosyvoice3-0.5b", Input: "this input is definitely long enough to justify a runpod offload", Voice: "narrator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Audio != "UlVOUE9E" || resp.Format != "mp3" {
		t.Errorf("audio=%q format=%q", resp.Audio, resp.Format)
	}
}

func TestRunpodPolling(t *testing.T) {
	srv, _ := newLocalServer(t, `{"busy":true}`, nil)
	defer srv.Close()

	polls := 0
	rp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/ep1/runsync":
			json.NewEncoder(w).Encode(runpodJob{ID: "j2", Status: "IN_QUEUE"})
		case "/v2/ep1/status/j2":
			polls++
			if polls < 2 {
				json.NewEncoder(w).Encode(runpodJob{ID: "j2", Status: "IN_PROGRESS"})
			} else {
				json.NewEncoder(w).Encode(runpodJob{ID: "j2", Status: "COMPLETED", Output: &runpodOutput{AudioB64: "ZG9uZQ=="}})
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer rp.Close()

	p := New("test-key", srv.URL, WithRunpod(rp.URL, "ep1", "rp-key"))
	p.limiter = newLimiter(30)
	p.pollInterval = 5 * time.Millisecond
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Model: "cosyvoice3-0.5b", Input: "this input is definitely long enough to justify a runpod offload",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Audio != "ZG9uZQ==" || resp.Format != "mp3" || polls != 2 {
		t.Errorf("audio=%q format=%q polls=%d", resp.Audio, resp.Format, polls)
	}
}

func TestBusyWithoutRunpodFallsBackToLocal(t *testing.T) {
	srv, _ := newLocalServer(t, `{"busy":true,"queue_depth":9}`, []byte("LOCAL"))
	defer srv.Close()

	p := New("test-key", srv.URL)
	p.runpodEndpoint = ""
	p.runpodAPIKey = ""
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Model: "cosyvoice3-0.5b", Input: "this input is definitely long enough to justify a runpod offload",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Audio != base64.StdEncoding.EncodeToString([]byte("LOCAL")) {
		t.Errorf("audio = %q", resp.Audio)
	}
}

func TestDecideRoute(t *testing.T) {
	healthy := localHealth{reachable: true}
	busy := localHealth{reachable: true, busy: true}
	deep := localHealth{reachable: true, queueDepth: 5}
	down := localHealth{}

	mk := func(endpoint, key string, lim *limiter) *AppNZProvider {
		p := New("k", "http://127.0.0.1:9080")
		p.runpodEndpoint = endpoint
		p.runpodAPIKey = key
		p.limiter = lim
		return p
	}
	exhausted := &limiter{rpm: 30, tokens: 0, last: time.Now()}

	cases := []struct {
		name     string
		p        *AppNZProvider
		h        localHealth
		inputLen int
		want     route
	}{
		{"healthy local", mk("ep", "k", newLimiter(30)), healthy, 500, routeLocal},
		{"busy long input", mk("ep", "k", newLimiter(30)), busy, 500, routeRunpod},
		{"deep queue", mk("ep", "k", newLimiter(30)), deep, 500, routeRunpod},
		{"unreachable", mk("ep", "k", newLimiter(30)), down, 500, routeRunpod},
		{"busy short input", mk("ep", "k", newLimiter(30)), busy, 10, routeLocal},
		{"busy runpod disabled", mk("", "", newLimiter(30)), busy, 500, routeLocal},
		{"rate limit exhausted", mk("ep", "k", exhausted), busy, 500, routeLocal},
	}
	for _, tc := range cases {
		if got := tc.p.decideRoute(tc.h, tc.inputLen); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestLimiter(t *testing.T) {
	l := newLimiter(2)
	if !l.allow() || !l.allow() {
		t.Error("first two should pass")
	}
	if l.allow() {
		t.Error("third should be limited")
	}
}

func TestHealthCheck(t *testing.T) {
	srv, _ := newLocalServer(t, `{"busy":false}`, nil)
	defer srv.Close()
	p := New("k", srv.URL)
	if err := p.HealthCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	srv.Close()
	if err := p.HealthCheck(context.Background()); err == nil {
		t.Error("expected error after close")
	}
}
