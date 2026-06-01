package fal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// TestLiveMeshyV6ImageTo3D exercises the real fal-ai/meshy/v6/image-to-3d
// endpoint end-to-end before deploy. Guarded behind RUN_LIVE_MESHY3D=1.
//
//	RUN_LIVE_MESHY3D=1 FAL_KEY=... go test ./internal/provider/fal -run TestLiveMeshyV6ImageTo3D -v -timeout 8m
func TestLiveMeshyV6ImageTo3D(t *testing.T) {
	if os.Getenv("RUN_LIVE_MESHY3D") != "1" {
		t.Skip("set RUN_LIVE_MESHY3D=1 to run the live Meshy v6 image-to-3d integration test")
	}
	apiKey := os.Getenv("FAL_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("FAL_API_KEY")
	}
	if apiKey == "" {
		t.Fatal("FAL_KEY or FAL_API_KEY is required for the live Meshy v6 test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	p := New(apiKey)
	resp, err := p.Generate3D(ctx, &model.Model3DGenerationRequest{
		Model:    "fal-ai/meshy/v6/image-to-3d",
		ImageURL: "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL == "" {
		t.Fatalf("missing model_glb.url; resp = %#v", resp)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 80 {
		t.Fatalf("unexpected billing = %#v", resp.Billing)
	}
	t.Logf("meshy v6 GLB: %s", resp.ModelGLB.URL)
}
