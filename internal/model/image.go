package model

import "encoding/json"

type ImageGenerationRequest struct {
	Grounding           *bool        `json:"grounding,omitempty"`
	Model               string       `json:"model"`
	Prompt              string       `json:"prompt"`
	N                   int          `json:"n,omitempty"`
	NumImages           int          `json:"num_images,omitempty"`
	Size                string       `json:"size,omitempty"`
	Quality             string       `json:"quality,omitempty"`
	Style               string       `json:"style,omitempty"`
	ResponseFormat      string       `json:"response_format,omitempty"`
	OutputFormat        string       `json:"output_format,omitempty"`
	ImageSize           any          `json:"image_size,omitempty"`
	Image               *ImageInput  `json:"image,omitempty"`
	Images              []ImageInput `json:"images,omitempty"`
	ImageURL            string       `json:"image_url,omitempty"`
	ImageURLs           []string     `json:"image_urls,omitempty"`
	ReferenceImageURLs  []string     `json:"reference_image_urls,omitempty"`
	AspectRatio         string       `json:"aspect_ratio,omitempty"`
	ExpandTop           *int         `json:"expand_top,omitempty"`
	ExpandBottom        *int         `json:"expand_bottom,omitempty"`
	ExpandLeft          *int         `json:"expand_left,omitempty"`
	ExpandRight         *int         `json:"expand_right,omitempty"`
	ZoomOutPercentage   *float64     `json:"zoom_out_percentage,omitempty"`
	AutoCrop            *bool        `json:"auto_crop,omitempty"`
	Seed                *int         `json:"seed,omitempty"`
	NumInferenceSteps   int          `json:"num_inference_steps,omitempty"`
	GuidanceScale       *float64     `json:"guidance_scale,omitempty"`
	EnableSafetyChecker *bool        `json:"enable_safety_checker,omitempty"`
	KeepOriginalAspect  *bool        `json:"keep_original_aspect,omitempty"`
	TargetSizes         []string     `json:"target_sizes,omitempty"`
	NumImagesPerSize    int          `json:"num_images_per_size,omitempty"`
	Resolution          string       `json:"resolution,omitempty"`
	SafetyTolerance     string       `json:"safety_tolerance,omitempty"`
	DisablePUP          *bool        `json:"disable_pup,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
}

type ImageInput struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Accept native BFL URL/base64 images and numeric safety tolerance alongside
// the shared object-based image schema used by other providers.
func (r *ImageGenerationRequest) UnmarshalJSON(data []byte) error {
	type plain ImageGenerationRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	images := fields["images"]
	delete(fields, "images")
	if raw := fields["safety_tolerance"]; len(raw) > 0 && raw[0] != '"' && string(raw) != "null" {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return err
		}
		fields["safety_tolerance"], _ = json.Marshal(n.String())
	}
	rest, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var decoded plain
	if err := json.Unmarshal(rest, &decoded); err != nil {
		return err
	}
	if len(images) > 0 && string(images) != "null" {
		var entries []json.RawMessage
		if images[0] == '"' {
			entries = []json.RawMessage{images}
		} else if err := json.Unmarshal(images, &entries); err != nil {
			return err
		}
		for _, entry := range entries {
			var image ImageInput
			if len(entry) > 0 && entry[0] == '"' {
				if err := json.Unmarshal(entry, &image.URL); err != nil {
					return err
				}
			} else if err := json.Unmarshal(entry, &image); err != nil {
				return err
			}
			decoded.Images = append(decoded.Images, image)
		}
	}
	*r = ImageGenerationRequest(decoded)
	return nil
}
