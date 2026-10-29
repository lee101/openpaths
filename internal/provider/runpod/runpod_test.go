package runpod

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

func TestGenerateVideoPVideo(t *testing.T) {
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("auth header = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/p-video/run":
			var body struct{ Input map[string]any }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Input["resolution"] != "1080p" || body.Input["duration"] != float64(5) || body.Input["prompt"] != "fox" {
				t.Errorf("input = %v", body.Input)
			}
			_, _ = w.Write([]byte(`{"id":"j1","status":"IN_QUEUE"}`))
		case "/p-video/status/j1":
			polls++
			_, _ = w.Write([]byte(`{"id":"j1","status":"COMPLETED","output":{"result":"https://v/x.mp4","cost":0.2}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	p := New(openai.NewCompatible("runpod", "k", "", nil), "k")
	p.apiBase, p.pollInterval = srv.URL, time.Millisecond
	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "p-video", Prompt: "fox", Duration: "5", Resolution: "1080p"})
	if err != nil || resp.VideoURL != "https://v/x.mp4" || polls != 1 {
		t.Fatalf("resp=%+v err=%v polls=%d", resp, err, polls)
	}
}

func TestVideoInputEditRequiresVideo(t *testing.T) {
	if _, err := videoInput("p-video-edit", &model.VideoGenerationRequest{Prompt: "x"}); err == nil {
		t.Fatal("expected error without video")
	}
	in, err := videoInput("p-video-edit", &model.VideoGenerationRequest{Prompt: "x", VideoURL: "https://v/a.mp4"})
	if err != nil || in["video"] != "https://v/a.mp4" {
		t.Fatalf("in=%v err=%v", in, err)
	}
}
