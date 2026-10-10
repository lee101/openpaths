package handler

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

// Alibaba signs Wan output links for 24 hours; copy them to our bucket so
// shared results and the gallery keep working.
func (h *VideoHandler) rehostExpiringVideoURL(ctx context.Context, resp *model.VideoGenerationResponse, modelID string) {
	if resp == nil || h.store == nil || !strings.HasPrefix(resp.VideoURL, "http") {
		return
	}
	dir, err := os.MkdirTemp("", "openpaths-video-rehost-*")
	if err != nil {
		log.Printf("video rehost: %v", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "output.mp4")
	n, err := downloadVideoForReencode(ctx, resp.VideoURL, path)
	if err != nil {
		log.Printf("video rehost download: %v", err)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf("video rehost open: %v", err)
		return
	}
	defer f.Close()
	safeModel := strings.NewReplacer("/", "-", " ", "-").Replace(modelID)
	url, err := h.store.Upload(ctx, safeModel+".mp4", "video/mp4", f)
	if err != nil {
		log.Printf("video rehost upload: %v", err)
		return
	}
	resp.VideoURL, resp.Bytes, resp.OutputFormat = url, n, "mp4"
}
