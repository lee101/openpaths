package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestParseMeshRiggingRequestValidation(t *testing.T) {
	t.Parallel()

	if _, _, err := parseMeshRiggingRequest([]byte(`{}`)); err == nil {
		t.Fatal("expected error when model_url is missing")
	}
	if _, _, err := parseMeshRiggingRequest([]byte(`{"model_url":"/local/file.glb"}`)); err == nil {
		t.Fatal("expected error for non-public model_url")
	}

	req, cost, err := parseMeshRiggingRequest([]byte(`{"model_url":"https://example.com/in.glb"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.Model != "meshy-rigging" {
		t.Fatalf("default model = %q", req.Model)
	}
	if cost != 2000 {
		t.Fatalf("base cost = %d, want 2000", cost)
	}
}

func TestRiggingRequestCostAnimationSurcharge(t *testing.T) {
	t.Parallel()

	if got := riggingRequestCost(false); got != 2000 {
		t.Fatalf("base cost = %d, want 2000", got)
	}
	if got := riggingRequestCost(true); got != 3200 {
		t.Fatalf("animation cost = %d, want 3200", got)
	}
}

func TestParseMeshRiggingRequestAnimationCost(t *testing.T) {
	t.Parallel()

	_, cost, err := parseMeshRiggingRequest([]byte(`{"model_url":"https://example.com/in.glb","enable_animation":true}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cost != 3200 {
		t.Fatalf("animation cost = %d, want 3200", cost)
	}
}

func TestRiggingJobCacheCoalescesIdenticalRequests(t *testing.T) {
	t.Parallel()

	cache := newRiggingJobCache()
	req := model.MeshRiggingRequest{Model: "meshy-rigging", ModelURL: "https://example.com/in.glb"}

	first, cached := cache.getOrCreate(req)
	if cached {
		t.Fatal("first request should create a job")
	}
	second, cached := cache.getOrCreate(req)
	if !cached {
		t.Fatal("second identical request should reuse the job")
	}
	if second.ID != first.ID {
		t.Fatalf("expected reuse of job %q, got %q", first.ID, second.ID)
	}
}

func TestRiggingRequestCacheKeyIgnoresAsync(t *testing.T) {
	t.Parallel()

	base := model.MeshRiggingRequest{Model: "meshy-rigging", ModelURL: "https://example.com/in.glb"}
	async := base
	async.Async = true
	if riggingRequestCacheKey(base) != riggingRequestCacheKey(async) {
		t.Fatal("async flag must not affect rigging cache key")
	}
}

func TestModel3DRequestCostForModelFamily(t *testing.T) {
	t.Parallel()

	req := func(textureSize int, shouldTexture *bool) model.Model3DGenerationRequest {
		return model.Model3DGenerationRequest{TextureSize: textureSize, ShouldTexture: shouldTexture}
	}
	on, off := true, false

	pixal := &model.ModelConfig{ID: "pixal3d-image-to-3d", ProviderModelID: "fal-ai/pixal3d", PricePerRequest: 0.30}
	if got := model3DRequestCostFor(pixal, req(1024, nil)); got != 3000 {
		t.Fatalf("pixal 1024 cost = %d, want 3000", got)
	}
	if got := model3DRequestCostFor(pixal, req(4096, nil)); got != 4200 {
		t.Fatalf("pixal 4096 cost = %d, want 4200 (texture-tiered, not flat price_per_request)", got)
	}

	meshy := &model.ModelConfig{ID: "meshy-v6-image-to-3d", ProviderModelID: "fal-ai/meshy/v6/image-to-3d", PricePerRequest: 0.80}
	if got := model3DRequestCostFor(meshy, req(1024, nil)); got != 8000 {
		t.Fatalf("meshy cost = %d, want 8000", got)
	}
	if got := model3DRequestCostFor(meshy, req(4096, nil)); got != 8000 {
		t.Fatalf("meshy cost must be flat regardless of texture: got %d", got)
	}

	tripo := &model.ModelConfig{ID: "tripo-p1-image-to-3d", ProviderModelID: "tripo3d/p1/image-to-3d", PricePerRequest: 0.50}
	if got := model3DRequestCostFor(tripo, req(0, nil)); got != 5000 {
		t.Fatalf("tripo default (texture on) cost = %d, want 5000", got)
	}
	if got := model3DRequestCostFor(tripo, req(0, &on)); got != 5000 {
		t.Fatalf("tripo texture-on cost = %d, want 5000", got)
	}
	if got := model3DRequestCostFor(tripo, req(0, &off)); got != 4000 {
		t.Fatalf("tripo texture-off cost = %d, want 4000", got)
	}
}
