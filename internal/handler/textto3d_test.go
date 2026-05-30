package handler

import (
	"encoding/json"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestParseTextTo3DRequestDefaults(t *testing.T) {
	t.Parallel()

	req, err := parseTextTo3DRequest([]byte(`{"prompt":"a sword"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Model != "pixal3d-image-to-3d" {
		t.Fatalf("3D model default = %q, want pixal3d-image-to-3d", req.Model)
	}
	if req.ImageModel != "openpaths/auto-image" {
		t.Fatalf("image model default = %q, want openpaths/auto-image", req.ImageModel)
	}
	if req.TextureSize != 1024 {
		t.Fatalf("texture default = %d, want 1024", req.TextureSize)
	}
	if req.Resolution != 1024 {
		t.Fatalf("resolution default = %d, want 1024", req.Resolution)
	}
}

func TestParseTextTo3DRequestRequiresPromptOrImage(t *testing.T) {
	t.Parallel()

	if _, err := parseTextTo3DRequest([]byte(`{}`)); err == nil {
		t.Fatal("expected error when neither prompt nor image_url is provided")
	}

	if _, err := parseTextTo3DRequest([]byte(`{"image_url":"https://example.com/a.png"}`)); err != nil {
		t.Fatalf("image_url alone should be valid: %v", err)
	}

	if _, err := parseTextTo3DRequest([]byte(`{"image_url":"ftp://nope"}`)); err == nil {
		t.Fatal("expected error for non-http image_url")
	}
}

func TestTextTo3DEstimateCostSumsImageAndModel3D(t *testing.T) {
	t.Parallel()

	// No billing engine: estimate falls back to the 3D leg only.
	h := &TextTo3DHandler{}
	got := h.estimateCost(model.TextTo3DGenerationRequest{TextureSize: 1024})
	if got != pixal3DRequestCost(1024) {
		t.Fatalf("estimate without billing = %d, want %d", got, pixal3DRequestCost(1024))
	}

	// With an image_url the image leg is skipped entirely.
	got = h.estimateCost(model.TextTo3DGenerationRequest{TextureSize: 2048, ImageURL: "https://example.com/a.png"})
	if got != pixal3DRequestCost(2048) {
		t.Fatalf("estimate with image_url = %d, want %d", got, pixal3DRequestCost(2048))
	}
}

func TestTextTo3DJobCacheCoalescesIdenticalRequests(t *testing.T) {
	t.Parallel()

	cache := newTextTo3DJobCache()
	req := model.TextTo3DGenerationRequest{Model: "pixal3d-image-to-3d", Prompt: "a sword", TextureSize: 1024}

	first, cached := cache.getOrCreate(req)
	if cached {
		t.Fatal("first request should create a job")
	}
	second, cached := cache.getOrCreate(req)
	if !cached {
		t.Fatal("second identical request should reuse the job")
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate request to reuse job %q, got %q", first.ID, second.ID)
	}
}

func TestTextTo3DJobPayloadShape(t *testing.T) {
	t.Parallel()

	cache := newTextTo3DJobCache()
	job, _ := cache.getOrCreate(model.TextTo3DGenerationRequest{Prompt: "a sword", TextureSize: 1024})
	payload := textTo3DJobPayload(job, false)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded["object"] != "3d.text_generation.job" {
		t.Fatalf("object = %v, want 3d.text_generation.job", decoded["object"])
	}
	if decoded["id"] == "" || decoded["id"] == nil {
		t.Fatal("expected a job id in the payload")
	}
}
