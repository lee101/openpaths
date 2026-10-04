package bfl

import (
	"context"
	"encoding/json"
	"github.com/openpaths/openpaths/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFlux3ImageNativeInputsAndPolling(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-key") != "test-key" {
			t.Error("missing API key header")
		}
		switch r.URL.Path {
		case "/v1/flux-3-image":
			json.NewDecoder(r.Body).Decode(&got)
			w.Write([]byte(`{"id":"flux3","polling_url":"/poll","cost":0.607}`))
		case "/poll":
			w.Write([]byte(`{"status":"Ready","result":{"sample":"https://example.com/generated.jpg"}}`))
		default:
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	var req model.ImageGenerationRequest
	err := json.Unmarshal([]byte(`{"model":"flux-3-image","prompt":"Edit image 1 using image 2","images":["https://example.com/a.png","base64-image"],"resolution":"4k","safety_tolerance":0,"grounding":false}`), &req)
	if err != nil {
		t.Fatal(err)
	}
	p := New("test-key", server.URL)
	resp, err := p.GenerateImage(context.Background(), &req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://example.com/generated.jpg" {
		t.Fatalf("response %#v", resp)
	}
	if got["resolution"] != "4k" || got["grounding"] != false || got["safety_tolerance"] != float64(0) || got["aspect_ratio"] != "auto" || len(got["images"].([]any)) != 2 {
		t.Fatalf("input %#v", got)
	}
	for _, key := range []string{"width", "height", "output_format", "disable_pup", "input_image"} {
		if _, ok := got[key]; ok {
			t.Errorf("unexpected key %s", key)
		}
	}
}

func TestFlux3ImageDefaultsAndLimits(t *testing.T) {
	req := model.ImageGenerationRequest{Prompt: "a forest"}
	payload, err := flux3ImagePayload(&req)
	if err != nil || payload["resolution"] != "1k" || payload["grounding"] != true || payload["safety_tolerance"] != 2 {
		t.Fatalf("defaults %#v err %v", payload, err)
	}
	req.ReferenceImageURLs = make([]string, 11)
	for i := range req.ReferenceImageURLs {
		req.ReferenceImageURLs[i] = "https://example.com/a.png"
	}
	if _, err := flux3ImagePayload(&req); err == nil {
		t.Fatal("accepted eleven references")
	}
	req.ReferenceImageURLs = nil
	req.SafetyTolerance = "5"
	if _, err := flux3ImagePayload(&req); err == nil {
		t.Fatal("accepted safety 5")
	}
	req.SafetyTolerance = "2"
	req.Resolution = "8k"
	if _, err := flux3ImagePayload(&req); err == nil {
		t.Fatal("accepted unsupported resolution")
	}
}
