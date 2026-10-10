package model

import "encoding/json"

type TranscriptionRequest struct {
	File     []byte `json:"-"`
	Filename string `json:"-"`
	Model    string `json:"model"`
	Language string `json:"language,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
	Format   string `json:"response_format,omitempty"`
}

type TranscriptionResponse struct {
	Text            string          `json:"text"`
	AudioDurationMS int64           `json:"audio_duration_ms,omitempty"`
	Turns           json.RawMessage `json:"turns,omitempty"`
	Segments        json.RawMessage `json:"segments,omitempty"`
	// Usage follows the OpenAI transcription shape; when present it drives billing.
	Usage *TranscriptionUsage `json:"usage,omitempty"`
}

type TranscriptionUsage struct {
	Type         string  `json:"type"` // "tokens" or "duration"
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	TotalTokens  int     `json:"total_tokens,omitempty"`
	Seconds      float64 `json:"seconds,omitempty"`
}
