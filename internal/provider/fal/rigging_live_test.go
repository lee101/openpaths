package fal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// TestLiveRigMesh exercises the real fal-ai/meshy/rigging endpoint end-to-end so
// we can confirm the integration works BEFORE deploying. Guarded behind
// RUN_LIVE_RIGGING=1 and a real fal key (FAL_KEY or FAL_API_KEY).
//
//	RUN_LIVE_RIGGING=1 FAL_KEY=... go test ./internal/provider/fal -run TestLiveRigMesh -v -timeout 5m
func TestLiveRigMesh(t *testing.T) {
	if os.Getenv("RUN_LIVE_RIGGING") != "1" {
		t.Skip("set RUN_LIVE_RIGGING=1 to run the live meshy rigging integration test")
	}
	apiKey := os.Getenv("FAL_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("FAL_API_KEY")
	}
	if apiKey == "" {
		t.Fatal("FAL_KEY or FAL_API_KEY is required for the live rigging test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	p := New(apiKey)
	resp, err := p.RigMesh(ctx, &model.MeshRiggingRequest{
		Model: "fal-ai/meshy/rigging",
		// A clearly-humanoid GLB (three.js Soldier) — fal's docs sample and the
		// low-poly CesiumMan both fail meshy pose estimation intermittently.
		ModelURL: "https://threejs.org/examples/models/gltf/Soldier.glb",
	})
	if err != nil {
		t.Fatalf("RigMesh() error = %v", err)
	}
	if resp.RiggedCharacterGLB.URL == "" {
		t.Fatalf("missing rigged_character_glb.url; resp = %#v", resp)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 20 {
		t.Fatalf("unexpected billing = %#v", resp.Billing)
	}
	t.Logf("rigged GLB: %s (task %s)", resp.RiggedCharacterGLB.URL, resp.RigTaskID)
}
