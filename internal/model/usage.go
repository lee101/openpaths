package model

import "time"

type UsageLog struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	APIKeyID      string    `json:"api_key_id"`
	Model         string    `json:"model"`
	Provider      string    `json:"provider"`
	TokensIn      int       `json:"tokens_in"`
	TokensOut     int       `json:"tokens_out"`
	LatencyMs     int       `json:"latency_ms"`
	TTFTMs        int       `json:"ttft_ms"`
	TPS           float32   `json:"tps"`
	CostCents     int64     `json:"cost_cents"`
	Error         *string   `json:"error,omitempty"`
	StatusCode    int       `json:"status_code"`
	Streaming     bool      `json:"streaming"`
	AppID         string    `json:"app_id,omitempty"`
	AppURL        string    `json:"app_url,omitempty"`
	AppTitle      string    `json:"app_title,omitempty"`
	AppCategories []string  `json:"app_categories,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
