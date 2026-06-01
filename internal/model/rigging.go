package model

// MeshRiggingRequest is the request for 3D mesh auto-rigging (3D-to-3D). The
// input is a public URL to a humanoid GLB mesh; the output is a rigged GLB/FBX,
// optionally with a walk/run animation preset applied.
type MeshRiggingRequest struct {
	Model               string   `json:"model"`
	ModelURL            string   `json:"model_url"`
	HeightMeters        *float64 `json:"height_meters,omitempty"`
	EnableAnimation     bool     `json:"enable_animation,omitempty"`
	AnimationActionID   *int     `json:"animation_action_id,omitempty"`
	EnableSafetyChecker *bool    `json:"enable_safety_checker,omitempty"`
	Async               bool     `json:"async,omitempty"`
}

// MeshRiggingResponse carries the rigged assets returned by the provider.
type MeshRiggingResponse struct {
	RiggedCharacterGLB FileAsset           `json:"rigged_character_glb"`
	RiggedCharacterFBX *FileAsset          `json:"rigged_character_fbx,omitempty"`
	BasicAnimations    *BasicAnimations    `json:"basic_animations,omitempty"`
	AnimationGLB       *FileAsset          `json:"animation_glb,omitempty"`
	AnimationFBX       *FileAsset          `json:"animation_fbx,omitempty"`
	RigTaskID          string              `json:"rig_task_id,omitempty"`
	Billing            *MeshRiggingBilling `json:"billing,omitempty"`
	BackendUsed        string              `json:"backend_used,omitempty"`
	CreditsCharged     float64             `json:"credits_charged,omitempty"`
	Model              string              `json:"model,omitempty"`
}

// BasicAnimations holds the default walk/run animations Meshy always returns.
type BasicAnimations struct {
	WalkingGLB         *FileAsset `json:"walking_glb,omitempty"`
	WalkingFBX         *FileAsset `json:"walking_fbx,omitempty"`
	WalkingArmatureGLB *FileAsset `json:"walking_armature_glb,omitempty"`
	RunningGLB         *FileAsset `json:"running_glb,omitempty"`
	RunningFBX         *FileAsset `json:"running_fbx,omitempty"`
	RunningArmatureGLB *FileAsset `json:"running_armature_glb,omitempty"`
}

type MeshRiggingBilling struct {
	Animation         bool    `json:"animation"`
	ExternalCostUSD   float64 `json:"external_cost_usd"`
	ExternalCostCents int     `json:"external_cost_cents"`
}
