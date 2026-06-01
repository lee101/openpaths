package model

type Model3DGenerationRequest struct {
	Model                     string   `json:"model"`
	ImageURL                  string   `json:"image_url"`
	MeshURL                   string   `json:"mesh_url,omitempty"`
	Prompt                    string   `json:"prompt,omitempty"`
	Async                     bool     `json:"async,omitempty"`
	Seed                      *int     `json:"seed,omitempty"`
	Resolution                int      `json:"resolution,omitempty"`
	SSGuidanceStrength        *float64 `json:"ss_guidance_strength,omitempty"`
	SSGuidanceRescale         *float64 `json:"ss_guidance_rescale,omitempty"`
	SSSamplingSteps           int      `json:"ss_sampling_steps,omitempty"`
	SSRescaleT                *float64 `json:"ss_rescale_t,omitempty"`
	ShapeSlatGuidanceStrength *float64 `json:"shape_slat_guidance_strength,omitempty"`
	ShapeSlatGuidanceRescale  *float64 `json:"shape_slat_guidance_rescale,omitempty"`
	ShapeSlatSamplingSteps    int      `json:"shape_slat_sampling_steps,omitempty"`
	ShapeSlatRescaleT         *float64 `json:"shape_slat_rescale_t,omitempty"`
	TexSlatGuidanceStrength   *float64 `json:"tex_slat_guidance_strength,omitempty"`
	TexSlatGuidanceRescale    *float64 `json:"tex_slat_guidance_rescale,omitempty"`
	TexSlatSamplingSteps      int      `json:"tex_slat_sampling_steps,omitempty"`
	TexSlatRescaleT           *float64 `json:"tex_slat_rescale_t,omitempty"`
	MeshScale                 *float64 `json:"mesh_scale,omitempty"`
	MaxNumTokens              int      `json:"max_num_tokens,omitempty"`
	DecimationTarget          int      `json:"decimation_target,omitempty"`
	TextureSize               int      `json:"texture_size,omitempty"`
	Remesh                    *bool    `json:"remesh,omitempty"`

	// Meshy v6 image-to-3d options (ignored by Pixal3D).
	Topology        string `json:"topology,omitempty"`
	TargetPolycount int    `json:"target_polycount,omitempty"`
	SymmetryMode    string `json:"symmetry_mode,omitempty"`
	ShouldTexture   *bool  `json:"should_texture,omitempty"`
	EnablePBR       *bool  `json:"enable_pbr,omitempty"`
	TexturePrompt   string `json:"texture_prompt,omitempty"`
}

type Model3DGenerationResponse struct {
	ModelGLB       FileAsset       `json:"model_glb"`
	Seed           *int            `json:"seed,omitempty"`
	Billing        *Model3DBilling `json:"billing,omitempty"`
	BackendUsed    string          `json:"backend_used,omitempty"`
	CreditsCharged float64         `json:"credits_charged,omitempty"`
	Model          string          `json:"model,omitempty"`
}

type FileAsset struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
}

type Model3DBilling struct {
	TextureSize       int     `json:"texture_size"`
	ExternalCostUSD   float64 `json:"external_cost_usd"`
	ExternalCostCents int     `json:"external_cost_cents"`
}
