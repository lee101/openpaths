package fal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateVideo_Flux3MaxSelectsEndpointFromInputShape(t *testing.T) {
	paths := []string{}
	bodies := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, r.URL.Path)
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/flux3.mp4"}, "seed": 7})
	}))
	defer server.Close()
	p := New("test-key")
	p.baseURL, p.client = server.URL, server.Client()

	audio, seed := false, 42
	requests := []*model.VideoGenerationRequest{
		{Model: "blackforestlabs/flux-3-max/text-to-video", Prompt: "a red panda", Duration: "auto"},
		{Model: "blackforestlabs/flux-3-max/text-to-video", Prompt: "animate", ImageURL: "https://example.com/a.png", EndImageURL: "https://example.com/b.png", Duration: "12", Resolution: "1080P", AspectRatio: "9:16", ReasoningLevel: "low", GenerateAudio: &audio, Seed: &seed},
		{Model: "blackforestlabs/flux-3-max/text-to-video", Prompt: "move", Duration: "10", Keyframes: []model.VideoKeyframe{{ImageURL: "https://example.com/a.png", FrameIndex: 0}, {ImageURL: "https://example.com/b.png", FrameIndex: 240}}},
	}
	for _, req := range requests {
		if _, err := p.GenerateVideo(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}

	if paths[0] != "/blackforestlabs/flux-3-max/text-to-video" || bodies[0]["duration"] != float64(5) || bodies[0]["resolution"] != "720p" || bodies[0]["aspect_ratio"] != "auto" {
		t.Fatalf("text request = %s %#v", paths[0], bodies[0])
	}
	for _, key := range []string{"image_url", "keyframes", "generate_audio", "reasoning_level", "seed"} {
		if _, ok := bodies[0][key]; ok {
			t.Fatalf("text request must omit %s: %#v", key, bodies[0])
		}
	}
	if paths[1] != "/blackforestlabs/flux-3-max/image-to-video" || bodies[1]["image_url"] != "https://example.com/a.png" || bodies[1]["end_image_url"] != "https://example.com/b.png" {
		t.Fatalf("image request = %s %#v", paths[1], bodies[1])
	}
	if bodies[1]["duration"] != float64(12) || bodies[1]["resolution"] != "1080p" || bodies[1]["aspect_ratio"] != "9:16" ||
		bodies[1]["reasoning_level"] != "low" || bodies[1]["generate_audio"] != false || bodies[1]["seed"] != float64(42) {
		t.Fatalf("image controls = %#v", bodies[1])
	}
	frames, _ := bodies[2]["keyframes"].([]any)
	if paths[2] != "/blackforestlabs/flux-3-max/keyframes-to-video" || len(frames) != 2 {
		t.Fatalf("keyframes request = %s %#v", paths[2], bodies[2])
	}
	if _, ok := bodies[2]["image_url"]; ok {
		t.Fatalf("keyframes request must omit image_url: %#v", bodies[2])
	}
}

func TestFlux3MaxVideoInputRejects(t *testing.T) {
	cases := map[string]*model.VideoGenerationRequest{
		"short":           {Prompt: "x", Duration: "4"},
		"long":            {Prompt: "x", Duration: "21"},
		"resolution":      {Prompt: "x", Resolution: "480p"},
		"aspect":          {Prompt: "x", AspectRatio: "5:4"},
		"reasoning":       {Prompt: "x", ReasoningLevel: "medium"},
		"end only":        {Prompt: "x", EndImageURL: "https://example.com/b.png"},
		"mixed":           {Prompt: "x", ImageURL: "https://example.com/a.png", Keyframes: []model.VideoKeyframe{{ImageURL: "https://example.com/b.png"}}},
		"index too high":  {Prompt: "x", Keyframes: []model.VideoKeyframe{{ImageURL: "https://example.com/b.png", FrameIndex: 121}}},
		"duplicate index": {Prompt: "x", Keyframes: []model.VideoKeyframe{{ImageURL: "https://example.com/a.png"}, {ImageURL: "https://example.com/b.png"}}},
		"empty keyframe":  {Prompt: "x", Keyframes: []model.VideoKeyframe{{}}},
		"too many frames": {Prompt: "x", Keyframes: make([]model.VideoKeyframe, 11)},
		"missing prompt":  {},
	}
	for name, req := range cases {
		if _, _, err := flux3MaxVideoInput(req); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
