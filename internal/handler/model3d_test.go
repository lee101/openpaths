package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestPixal3DRequestCostTracksTextureTier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		textureSize int
		want        int64
	}{
		{textureSize: 0, want: 3000},
		{textureSize: 1024, want: 3000},
		{textureSize: 2048, want: 4200},
		{textureSize: 4096, want: 4200},
	}

	for _, tc := range cases {
		if got := pixal3DRequestCost(tc.textureSize); got != tc.want {
			t.Fatalf("texture %d: cost = %d, want %d", tc.textureSize, got, tc.want)
		}
	}
}

func TestModel3DRequestCacheKeyIgnoresAsync(t *testing.T) {
	t.Parallel()

	base := model.Model3DGenerationRequest{
		Model:       "pixal3d-image-to-3d",
		ImageURL:    "https://example.com/input.png",
		TextureSize: 1024,
		Async:       false,
	}
	async := base
	async.Async = true

	if model3DRequestCacheKey(base) != model3DRequestCacheKey(async) {
		t.Fatal("expected async flag not to affect 3D request cache key")
	}
}

func TestModel3DJobCacheCoalescesIdenticalRequests(t *testing.T) {
	t.Parallel()

	cache := newModel3DJobCache()
	req := model.Model3DGenerationRequest{
		Model:       "pixal3d-image-to-3d",
		ImageURL:    "https://example.com/input.png",
		TextureSize: 1024,
	}

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
