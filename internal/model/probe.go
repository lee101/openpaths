package model

import "time"

type ModelProbeResult struct {
	Model            string    `json:"model"`
	Provider         string    `json:"provider"`
	LatencyMs        int       `json:"latency_ms"`
	OK               bool      `json:"ok"`
	StatusCode       int       `json:"status_code"`
	Error            *string   `json:"error,omitempty"`
	ResponsePreview  string    `json:"response_preview,omitempty"`
	ProbedAt         time.Time `json:"probed_at"`
}
