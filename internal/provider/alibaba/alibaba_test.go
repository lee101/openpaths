package alibaba

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestVideoPayloadTextToVideo(t *testing.T) {
	audio := false
	payload, err := videoPayload(&model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "kitten", Resolution: "480p", AspectRatio: "auto", Duration: "auto", GenerateAudio: &audio})
	if err != nil {
		t.Fatal(err)
	}
	params := payload["parameters"].(map[string]any)
	if params["resolution"] != "480P" || params["ratio"] != "adaptive" || params["duration"] != -1 || params["audio"] != false || params["watermark"] != false {
		t.Fatalf("params %#v", params)
	}
	if _, ok := payload["input"].(map[string]any)["media"]; ok {
		t.Fatal("text-to-video must not send media")
	}
}

func TestVideoPayloadFrames(t *testing.T) {
	payload, err := videoPayload(&model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "p", ImageURL: "https://x/a.png", EndImageURL: "https://x/b.png", Duration: "45"})
	if err != nil {
		t.Fatal(err)
	}
	items := payload["input"].(map[string]any)["media"].([]media)
	if len(items) != 2 || items[0] != (media{"first_frame", "https://x/a.png"}) || items[1] != (media{"last_frame", "https://x/b.png"}) {
		t.Fatalf("media %#v", items)
	}
	if payload["parameters"].(map[string]any)["duration"] != 30 {
		t.Fatal("duration must clamp to 30")
	}
}

func TestVideoPayloadReferences(t *testing.T) {
	raw := json.RawMessage(`{"media":[{"type":"link","url":"https://example.com"}]}`)
	payload, err := videoPayload(&model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "Image 1 dances to Audio 1", ImageURL: "https://x/a.png", ReferenceImageURLs: []string{"https://x/b.png"}, VideoURL: "https://x/v.mp4", AudioURLs: []string{"https://x/s.mp3"}, Input: raw, PromptExpansionMode: "off"})
	if err != nil {
		t.Fatal(err)
	}
	items := payload["input"].(map[string]any)["media"].([]media)
	want := []media{{"reference_image", "https://x/a.png"}, {"reference_image", "https://x/b.png"}, {"reference_video", "https://x/v.mp4"}, {"reference_audio", "https://x/s.mp3"}, {"link", "https://example.com"}}
	if len(items) != len(want) {
		t.Fatalf("media %#v", items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("media[%d] = %#v want %#v", i, items[i], want[i])
		}
	}
	if payload["parameters"].(map[string]any)["prompt_extend"] != true {
		t.Fatal("link input forces prompt_extend")
	}
}

func TestVideoPayloadRejectsInvalidCombos(t *testing.T) {
	for name, req := range map[string]*model.VideoGenerationRequest{
		"frame+ref":  {Model: "wan3.0-video", Prompt: "p", EndImageURL: "https://x/e.png", VideoURL: "https://x/v.mp4"},
		"too many":   {Model: "wan3.0-video", Prompt: "p", VideoURLs: []string{"1", "2", "3", "4", "5", "6"}},
		"empty":      {Model: "wan3.0-video"},
		"file+link":  {Model: "wan3.0-video", Prompt: "p", Input: json.RawMessage(`{"media":[{"type":"file","url":"https://x/a.pdf"},{"type":"link","url":"https://x"}]}`)},
		"bad native": {Model: "wan3.0-video", Prompt: "p", Input: json.RawMessage(`{"media":[{"type":"mask","url":"https://x"}]}`)},
	} {
		if _, err := videoPayload(req); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestGenerateVideoSubmitsAndPolls(t *testing.T) {
	var polls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services/aigc/video-generation/video-synthesis":
			if r.Header.Get("X-DashScope-Async") != "enable" {
				t.Errorf("missing async header")
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"model":"wan3.0-video"`) {
				t.Errorf("body %s", body)
			}
			io.WriteString(w, `{"request_id":"r","output":{"task_id":"t1","task_status":"PENDING"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tasks/t1":
			if atomic.AddInt32(&polls, 1) == 1 {
				io.WriteString(w, `{"output":{"task_id":"t1","task_status":"RUNNING"}}`)
				return
			}
			io.WriteString(w, `{"output":{"task_id":"t1","task_status":"SUCCEEDED","video_url":"https://oss/v.mp4"},"usage":{"duration":7.5,"SR":720}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	p := New("key", srv.URL)
	p.pollInterval = time.Millisecond
	req := &model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "kitten", Resolution: "HD"}
	resp, err := p.GenerateVideo(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://oss/v.mp4" || resp.DurationSeconds != 7.5 || resp.BackendUsed != "alibaba" || req.Resolution != "720P" {
		t.Fatalf("resp %#v resolution %q", resp, req.Resolution)
	}
}

func TestGenerateVideoFailedTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			io.WriteString(w, `{"output":{"task_id":"t1","task_status":"PENDING"}}`)
			return
		}
		io.WriteString(w, `{"output":{"task_id":"t1","task_status":"FAILED","code":"DataInspectionFailed","message":"blocked"}}`)
	}))
	defer srv.Close()
	p := New("key", srv.URL)
	p.pollInterval = time.Millisecond
	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "DataInspectionFailed") {
		t.Fatalf("err %v", err)
	}
}

func TestSubmitRejectsBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"code":"InvalidParameter","message":"bad ratio"}`)
	}))
	defer srv.Close()
	_, err := New("key", srv.URL).GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "wan3.0-video", Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "InvalidParameter bad ratio") {
		t.Fatalf("err %v", err)
	}
}
