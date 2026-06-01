package fal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// TestLiveTripoP1ImageTo3D exercises the real tripo3d/p1/image-to-3d endpoint
// end-to-end before deploy. Guarded behind RUN_LIVE_TRIPO3D=1.
//
//	RUN_LIVE_TRIPO3D=1 FAL_KEY=... go test ./internal/provider/fal -run TestLiveTripoP1ImageTo3D -v -timeout 8m
func TestLiveTripoP1ImageTo3D(t *testing.T) {
	if os.Getenv("RUN_LIVE_TRIPO3D") != "1" {
		t.Skip("set RUN_LIVE_TRIPO3D=1 to run the live Tripo p1 image-to-3d integration test")
	}
	apiKey := os.Getenv("FAL_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("FAL_API_KEY")
	}
	if apiKey == "" {
		t.Fatal("FAL_KEY or FAL_API_KEY is required for the live Tripo test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	p := New(apiKey)
	resp, err := p.Generate3D(ctx, &model.Model3DGenerationRequest{
		Model:    "tripo3d/p1/image-to-3d",
		ImageURL: "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL == "" {
		t.Fatalf("missing glb url; resp = %#v", resp)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 50 {
		t.Fatalf("unexpected billing = %#v (want 50c with default textures)", resp.Billing)
	}
	t.Logf("tripo p1 GLB: %s", resp.ModelGLB.URL)
}
