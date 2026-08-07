package model

import "time"

// SavedResponse is one persisted text or image generation (input + output) for a
// user who has opted into response saving. Privately searchable via gobed semantic
// embeddings + pg_trgm.
type SavedResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	APIKeyID  string    `json:"api_key_id,omitempty"`
	Kind      string    `json:"kind"` // "text" | "image"
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	Prompt    string    `json:"prompt"`
	Input     string    `json:"input,omitempty"`
	Output    string    `json:"output,omitempty"`
	ImageURL  string    `json:"image_url,omitempty"`
	ThumbURL  string    `json:"thumb_url,omitempty"`
	Width     int       `json:"width,omitempty"`
	Height    int       `json:"height,omitempty"`
	TokensIn  int       `json:"tokens_in,omitempty"`
	TokensOut int       `json:"tokens_out,omitempty"`
	CostCents int64     `json:"cost_cents,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Embedding holds the raw 512×float32 little-endian vector when loaded for search.
	// Never serialised to clients.
	Embedding []byte  `json:"-"`
	Score     float64 `json:"score,omitempty"`
}
