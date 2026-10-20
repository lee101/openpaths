package fal

import (
	"fmt"
	"github.com/openpaths/openpaths/internal/model"
	"strings"
)

func h3MaxRecastInput(req *model.VideoGenerationRequest) (map[string]any, error) {
	video := strings.TrimSpace(req.VideoURL)
	if video == "" && len(req.VideoURLs) > 0 {
		video = strings.TrimSpace(req.VideoURLs[0])
	}
	refs := req.ReferenceImageURLs
	if len(refs) == 0 {
		refs = req.ImageURLs
	}
	if len(refs) == 0 && req.ImageURL != "" {
		refs = []string{req.ImageURL}
	}
	if video == "" {
		return nil, fmt.Errorf("video_url is required for H3 Max Recast")
	}
	if len(refs) < 1 || len(refs) > 4 {
		return nil, fmt.Errorf("H3 Max Recast requires one to four reference photos")
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			return nil, fmt.Errorf("reference photo URLs must not be empty")
		}
	}
	resolution := strings.ToUpper(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = "1080P"
	}
	if resolution != "768P" && resolution != "1080P" {
		return nil, fmt.Errorf("H3 Max Recast resolution must be 768P or 1080P")
	}
	req.Resolution = resolution
	input := map[string]any{"video_url": video, "reference_image_urls": refs, "resolution": resolution}
	if strings.TrimSpace(req.Prompt) != "" {
		input["prompt"] = req.Prompt
	}
	if req.Seed != nil {
		input["seed"] = *req.Seed
	}
	return input, nil
}
