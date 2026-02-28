package model

type VideoGenerationRequest struct {
	Model               string   `json:"model"`
	Prompt              string   `json:"prompt"`
	ImageURL            string   `json:"image_url,omitempty"`
	NumFrames           int      `json:"num_frames,omitempty"`
	FramesPerSecond     int      `json:"frames_per_second,omitempty"`
	Resolution          string   `json:"resolution,omitempty"`
	AspectRatio         string   `json:"aspect_ratio,omitempty"`
	NegativePrompt      string   `json:"negative_prompt,omitempty"`
	Seed                *int     `json:"seed,omitempty"`
	NumInferenceSteps   int      `json:"num_inference_steps,omitempty"`
	GuidanceScale       *float64 `json:"guidance_scale,omitempty"`
	EnableSafetyChecker *bool    `json:"enable_safety_checker,omitempty"`
}

type VideoGenerationResponse struct {
	VideoURL       string  `json:"video_url"`
	Seed           *int    `json:"seed,omitempty"`
	BackendUsed    string  `json:"backend_used,omitempty"`
	CreditsCharged float64 `json:"credits_charged,omitempty"`
	Model          string  `json:"model,omitempty"`
}
