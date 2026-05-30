package model

// TextTo3DGenerationRequest drives the two-step text/image -> 3D pipeline:
// an image is generated from the prompt via the auto image route, then turned
// into a textured GLB by the 3D provider. If ImageURL is provided the image
// step is skipped and it behaves like image-to-3D.
type TextTo3DGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	ImageModel     string `json:"image_model,omitempty"`
	ImageURL       string `json:"image_url,omitempty"`
	Size           string `json:"size,omitempty"`
	Async          bool   `json:"async,omitempty"`
	Seed           *int   `json:"seed,omitempty"`
	Resolution     int    `json:"resolution,omitempty"`
	TextureSize    int    `json:"texture_size,omitempty"`
	Remesh         *bool  `json:"remesh,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type TextTo3DGenerationResponse struct {
	ModelGLB       FileAsset          `json:"model_glb"`
	Image          *TextTo3DImageInfo `json:"image,omitempty"`
	Prompt         string             `json:"prompt,omitempty"`
	Seed           *int               `json:"seed,omitempty"`
	Billing        *TextTo3DBilling   `json:"billing,omitempty"`
	BackendUsed    string             `json:"backend_used,omitempty"`
	CreditsCharged float64            `json:"credits_charged,omitempty"`
	Model          string             `json:"model,omitempty"`
}

type TextTo3DImageInfo struct {
	URL   string `json:"url,omitempty"`
	Model string `json:"model,omitempty"`
}

// TextTo3DBilling reports both legs of the combined charge in USD.
type TextTo3DBilling struct {
	TextureSize    int     `json:"texture_size"`
	ImageModel     string  `json:"image_model,omitempty"`
	ImageCostUSD   float64 `json:"image_cost_usd"`
	Model3DCostUSD float64 `json:"model_3d_cost_usd"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
}
