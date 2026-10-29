package seedream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatUsesAPIV3Path(t *testing.T) {
	var path, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, auth = r.URL.Path, r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(model.ChatCompletionResponse{})
	}))
	defer srv.Close()
	p := New("k", srv.URL+"/api/v3")
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "deepseek-v4-1-flash-261017", Messages: []model.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/v3/chat/completions" || auth != "Bearer k" {
		t.Fatalf("path=%q auth=%q", path, auth)
	}
}

func TestGenerateImage(t *testing.T) {
	var got map[string]any
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"abc"}]}`))
	}))
	defer srv.Close()
	p := New("k", srv.URL+"/api/v3")
	res, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "seedream-4-5-251128", Prompt: "cat", Size: "2K"})
	if err != nil || len(res.Data) != 1 {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if path != "/api/v3/images/generations" || got["watermark"] != false || got["size"] != "2K" {
		t.Fatalf("path=%q body=%v", path, got)
	}
}

func TestGenerateVideoPollsTask(t *testing.T) {
	polls := 0
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if r.URL.Path != "/api/v3/contents/generations/tasks" {
				t.Errorf("path %q", r.URL.Path)
			}
			_ = json.NewDecoder(r.Body).Decode(&sent)
			_, _ = w.Write([]byte(`{"id":"cgt-1"}`))
			return
		}
		polls++
		if polls < 2 {
			_, _ = w.Write([]byte(`{"id":"cgt-1","status":"running"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"cgt-1","status":"succeeded","content":{"video_url":"https://v/x.mp4"}}`))
	}))
	defer srv.Close()
	p := New("k", srv.URL+"/api/v3")
	p.pollInterval = time.Millisecond
	res, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "dreamina-seedance-2-5-260628", Prompt: "a fox", ImageURL: "https://i/a.jpg", Duration: "5", Ratio: "16:9",
	})
	if err != nil || res.VideoURL != "https://v/x.mp4" {
		t.Fatalf("res=%v err=%v", res, err)
	}
	content := sent["content"].([]any)
	if len(content) != 2 || content[1].(map[string]any)["role"] != "first_frame" || sent["duration"] != float64(5) || sent["watermark"] != false {
		t.Fatalf("sent=%v", sent)
	}
}
