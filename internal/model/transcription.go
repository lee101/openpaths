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
}
