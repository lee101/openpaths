package model

type ImageGenerationRequest struct {
	Model               string       `json:"model"`
	Prompt              string       `json:"prompt"`
	N                   int          `json:"n,omitempty"`
	Size                string       `json:"size,omitempty"`
	Quality             string       `json:"quality,omitempty"`
	Style               string       `json:"style,omitempty"`
	ResponseFormat      string       `json:"response_format,omitempty"`
	OutputFormat        string       `json:"output_format,omitempty"`
	Image               *ImageInput  `json:"image,omitempty"`
	Images              []ImageInput `json:"images,omitempty"`
	ImageURL            string       `json:"image_url,omitempty"`
	ImageURLs           []string     `json:"image_urls,omitempty"`
	ReferenceImageURLs  []string     `json:"reference_image_urls,omitempty"`
	AspectRatio         string       `json:"aspect_ratio,omitempty"`
	Seed                *int         `json:"seed,omitempty"`
	NumInferenceSteps   int          `json:"num_inference_steps,omitempty"`
	GuidanceScale       *float64     `json:"guidance_scale,omitempty"`
	EnableSafetyChecker *bool        `json:"enable_safety_checker,omitempty"`
	KeepOriginalAspect  *bool        `json:"keep_original_aspect,omitempty"`
	TargetSizes         []string     `json:"target_sizes,omitempty"`
	NumImagesPerSize    int          `json:"num_images_per_size,omitempty"`
	Resolution          string       `json:"resolution,omitempty"`
	SafetyTolerance     string       `json:"safety_tolerance,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageInput struct {
	Type string `json:"type,omitempty"`
	URL  string `json:"url,omitempty"`
}
