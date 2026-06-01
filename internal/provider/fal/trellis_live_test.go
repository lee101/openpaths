package fal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// TestLiveTrellis2Retexture exercises the real fal-ai/trellis-2/retexture
// endpoint end-to-end before deploy. Guarded behind RUN_LIVE_TRELLIS=1.
//
//	RUN_LIVE_TRELLIS=1 FAL_KEY=... go test ./internal/provider/fal -run TestLiveTrellis2Retexture -v -timeout 8m
func TestLiveTrellis2Retexture(t *testing.T) {
	if os.Getenv("RUN_LIVE_TRELLIS") != "1" {
		t.Skip("set RUN_LIVE_TRELLIS=1 to run the live Trellis-2 retexture integration test")
	}
	apiKey := os.Getenv("FAL_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("FAL_API_KEY")
	}
	if apiKey == "" {
		t.Fatal("FAL_KEY or FAL_API_KEY is required for the live Trellis test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	p := New(apiKey)
	resp, err := p.Generate3D(ctx, &model.Model3DGenerationRequest{
		Model:      "fal-ai/trellis-2/retexture",
		ImageURL:   "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
		MeshURL:    "https://threejs.org/examples/models/gltf/Soldier.glb",
		Resolution: 512,
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL == "" {
		t.Fatalf("missing model_glb.url; resp = %#v", resp)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 20 {
		t.Fatalf("unexpected billing = %#v (want 20c at 512p)", resp.Billing)
	}
	t.Logf("trellis retexture GLB: %s", resp.ModelGLB.URL)
}
