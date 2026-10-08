package fal

import (
	"fmt"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

var h3MaxRelightAspects = map[string]string{"adaptive": "adaptive", "21:9": "21:9", "16:9": "16:9", "4:3": "4:3", "1:1": "1:1", "3:4": "3:4", "9:16": "9:16"}

func h3MaxRelightInput(req *model.VideoGenerationRequest) (map[string]any, error) {
	video := strings.TrimSpace(req.VideoURL)
	if video == "" && req.Video != nil {
		video = strings.TrimSpace(req.Video.URL)
	}
	if video == "" && len(req.VideoURLs) > 0 {
		video = strings.TrimSpace(req.VideoURLs[0])
	}
	if video == "" {
		return nil, fmt.Errorf("video_url is required for H3 Max Relight")
	}
	var sphere string
	for _, candidate := range append(append([]string{req.ReferenceImageURL}, req.ReferenceImageURLs...), append(req.ImageURLs, req.ImageURL)...) {
		if strings.TrimSpace(candidate) != "" {
			sphere = strings.TrimSpace(candidate)
			break
		}
	}
	if sphere == "" {
		return nil, fmt.Errorf("reference_image_url (a sphere render of the target lighting) is required for H3 Max Relight")
	}
	resolution := strings.ToUpper(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = "768P"
	}
	switch resolution {
	case "480P", "768P", "1080P", "2K":
	default:
		return nil, fmt.Errorf("H3 Max Relight resolution must be 480P, 768P, 1080P or 2K")
	}
	aspect := "16:9"
	if value := strings.ToLower(strings.TrimSpace(req.AspectRatio)); value != "" && value != "auto" {
		mapped, ok := h3MaxRelightAspects[value]
		if !ok {
			return nil, fmt.Errorf("H3 Max Relight aspect_ratio must be adaptive, 21:9, 16:9, 4:3, 1:1, 3:4 or 9:16")
		}
		aspect = mapped
	} else if value == "auto" {
		aspect = "adaptive"
	}
	req.Resolution = resolution
	input := map[string]any{"video_url": video, "reference_image_url": sphere, "resolution": resolution, "aspect_ratio": aspect}
	if req.Seed != nil {
		input["seed"] = *req.Seed
	}
	if req.EnableSafetyChecker != nil {
		input["enable_safety_checker"] = *req.EnableSafetyChecker
	}
	return input, nil
}
