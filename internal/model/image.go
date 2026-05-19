package model

type ImageGenerationRequest struct {
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
