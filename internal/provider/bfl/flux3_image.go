package bfl

import (
	"fmt"
	"github.com/openpaths/openpaths/internal/model"
	"strconv"
	"strings"
)

func flux3ImagePayload(req *model.ImageGenerationRequest) (map[string]any, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = "1k"
	}
	switch resolution {
	case "768sq", "1k", "2k", "4k":
	default:
		return nil, fmt.Errorf("resolution must be 768sq, 1k, 2k, or 4k")
	}
	aspect := valueOr(strings.TrimSpace(req.AspectRatio), "auto")
	switch aspect {
	case "auto", "21:9", "2:1", "16:9", "3:2", "7:5", "4:3", "5:4", "1:1", "4:5", "3:4", "5:7", "2:3", "9:16", "1:2", "9:21":
	default:
		return nil, fmt.Errorf("unsupported FLUX 3 aspect ratio")
	}
	safety := 2
	if req.SafetyTolerance != "" {
		n, err := strconv.Atoi(req.SafetyTolerance)
		if err != nil || n < 0 || n > 4 {
			return nil, fmt.Errorf("safety_tolerance must be an integer from 0 to 4")
		}
		safety = n
	}
	images := bflInputImages(req)
	if len(req.Images) > 0 {
		images = nil
		for _, image := range req.Images {
			images = append(images, image.URL)
		}
	} else if len(req.ReferenceImageURLs) > 0 {
		images = req.ReferenceImageURLs
	} else if len(req.ImageURLs) > 0 {
		images = req.ImageURLs
	}
	if len(images) > 10 {
		return nil, fmt.Errorf("FLUX 3 Image accepts at most ten reference images")
	}
	grounding := true
	if req.Grounding != nil {
		grounding = *req.Grounding
	}
	req.Resolution = resolution
	payload := map[string]any{"prompt": prompt, "resolution": resolution, "aspect_ratio": aspect, "safety_tolerance": safety, "grounding": grounding}
	if len(images) > 0 {
		payload["images"] = images
	}
	return payload, nil
}
