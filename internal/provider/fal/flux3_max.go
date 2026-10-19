package fal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

const (
	flux3MaxApp          = "blackforestlabs/flux-3-max"
	flux3MaxFPS          = 24
	flux3MaxMaxKeyframes = 10
)

var flux3MaxAspectRatios = map[string]bool{
	"auto": true, "21:9": true, "2:1": true, "16:9": true, "4:3": true, "1:1": true, "3:4": true, "9:16": true,
}

// flux3MaxVideoInput selects the FLUX 3 Max endpoint from the input shape
// (timed keyframes, a start image with optional end image, or text only) and
// builds that endpoint's native request body. All three share one schema for
// duration, resolution, aspect ratio, reasoning level, audio and seed.
func flux3MaxVideoInput(req *model.VideoGenerationRequest) (string, map[string]any, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", nil, fmt.Errorf("prompt is required")
	}
	body := map[string]any{"prompt": prompt}

	duration := 5
	if d := strings.ToLower(strings.TrimSpace(string(req.Duration))); d != "" && d != "auto" {
		n, err := strconv.Atoi(d)
		if err != nil || n < 5 || n > 20 {
			return "", nil, fmt.Errorf("duration must be between 5 and 20 seconds")
		}
		duration = n
	}
	body["duration"] = duration

	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	switch resolution {
	case "":
		resolution = "720p"
	case "720p", "1080p":
	default:
		return "", nil, fmt.Errorf("resolution must be 720p or 1080p")
	}
	body["resolution"] = resolution

	aspect := strings.ToLower(strings.TrimSpace(req.AspectRatio))
	if aspect == "" {
		aspect = "auto"
	}
	if !flux3MaxAspectRatios[aspect] {
		return "", nil, fmt.Errorf("aspect_ratio must be one of auto, 21:9, 2:1, 16:9, 4:3, 1:1, 3:4, 9:16")
	}
	body["aspect_ratio"] = aspect

	switch level := strings.ToLower(strings.TrimSpace(req.ReasoningLevel)); level {
	case "":
	case "low", "high":
		body["reasoning_level"] = level
	default:
		return "", nil, fmt.Errorf("reasoning_level must be low or high")
	}
	if req.GenerateAudio != nil {
		body["generate_audio"] = *req.GenerateAudio
	}
	if req.Seed != nil {
		body["seed"] = *req.Seed
	}

	imageURL := strings.TrimSpace(req.ImageURL)
	if imageURL == "" && len(req.ImageURLs) > 0 {
		imageURL = strings.TrimSpace(req.ImageURLs[0])
	}
	if imageURL == "" {
		for _, item := range req.Content {
			if item.Type == "image_url" && item.ImageURL != nil && strings.TrimSpace(item.ImageURL.URL) != "" {
				imageURL = strings.TrimSpace(item.ImageURL.URL)
				break
			}
		}
	}
	endURL := strings.TrimSpace(req.EndImageURL)

	if len(req.Keyframes) > 0 {
		if imageURL != "" || endURL != "" {
			return "", nil, fmt.Errorf("use either keyframes or image_url/end_image_url, not both")
		}
		if len(req.Keyframes) > flux3MaxMaxKeyframes {
			return "", nil, fmt.Errorf("keyframes must contain at most %d images", flux3MaxMaxKeyframes)
		}
		frames := make([]map[string]any, 0, len(req.Keyframes))
		seen := map[int]bool{}
		for i, frame := range req.Keyframes {
			url := strings.TrimSpace(frame.ImageURL)
			if url == "" {
				return "", nil, fmt.Errorf("keyframes[%d].image_url is required", i)
			}
			if frame.FrameIndex < 0 || frame.FrameIndex > duration*flux3MaxFPS {
				return "", nil, fmt.Errorf("keyframes[%d].frame_index must be between 0 and %d for a %d second video", i, duration*flux3MaxFPS, duration)
			}
			if seen[frame.FrameIndex] {
				return "", nil, fmt.Errorf("keyframes frame_index values must be unique")
			}
			seen[frame.FrameIndex] = true
			frames = append(frames, map[string]any{"image_url": url, "frame_index": frame.FrameIndex})
		}
		body["keyframes"] = frames
		return flux3MaxApp + "/keyframes-to-video", body, nil
	}

	if imageURL != "" {
		body["image_url"] = imageURL
		if endURL != "" {
			body["end_image_url"] = endURL
		}
		return flux3MaxApp + "/image-to-video", body, nil
	}
	if endURL != "" {
		return "", nil, fmt.Errorf("end_image_url requires image_url")
	}
	return flux3MaxApp + "/text-to-video", body, nil
}
