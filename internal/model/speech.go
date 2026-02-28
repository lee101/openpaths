package model

type SpeechRequest struct {
	Model string  `json:"model"`
	Input string  `json:"input"`
	Voice string  `json:"voice,omitempty"`
	Speed float64 `json:"speed,omitempty"`
}

type SpeechResponse struct {
	Audio      string `json:"audio,omitempty"`
	AudioURL   string `json:"audio_url,omitempty"`
	Format     string `json:"format"`
	DurationMs int    `json:"duration_ms,omitempty"`
	Characters int    `json:"characters,omitempty"`
}
