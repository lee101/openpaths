package model

type SpeechRequest struct {
	Model         string               `json:"model"`
	Input         string               `json:"input"`
	Text          string               `json:"text,omitempty"`
	Voice         string               `json:"voice,omitempty"`
	VoiceID       string               `json:"voice_id,omitempty"`
	Language      string               `json:"language,omitempty"`
	Speed         float64              `json:"speed,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	AutoEmotion   bool                 `json:"auto_emotion,omitempty"`
	SpeakerVoices []SpeechSpeakerVoice `json:"speaker_voices,omitempty"`
}

type SpeechSpeakerVoice struct {
	Speaker string `json:"speaker"`
	Voice   string `json:"voice"`
}

type SpeechResponse struct {
	Audio        string `json:"audio,omitempty"`
	AudioURL     string `json:"audio_url,omitempty"`
	Format       string `json:"format"`
	DurationMs   int    `json:"duration_ms,omitempty"`
	Characters   int    `json:"characters,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
}
