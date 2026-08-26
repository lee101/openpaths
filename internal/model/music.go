package model

type MusicGenerationRequest struct {
	Model        string        `json:"model"`
	Prompt       string        `json:"prompt,omitempty"`
	Lyrics       string        `json:"lyrics"`
	Stream       bool          `json:"stream,omitempty"`
	Duration     int           `json:"duration,omitempty"`
	OutputFormat string        `json:"output_format,omitempty"`
	AudioSetting *AudioSetting `json:"audio_setting,omitempty"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
}

type MusicGenerationResponse struct {
	Data         *MusicData         `json:"data,omitempty"`
	TraceID      string             `json:"trace_id,omitempty"`
	ExtraInfo    *MusicExtraInfo    `json:"extra_info,omitempty"`
	AnalysisInfo any                `json:"analysis_info,omitempty"`
	BaseResp     *MusicBaseResponse `json:"base_resp,omitempty"`
}

type MusicData struct {
	Status int    `json:"status"`
	Audio  string `json:"audio,omitempty"`
}

type MusicExtraInfo struct {
	Duration   int `json:"music_duration,omitempty"`
	SampleRate int `json:"music_sample_rate,omitempty"`
	Channel    int `json:"music_channel,omitempty"`
	Bitrate    int `json:"bitrate,omitempty"`
	Size       int `json:"music_size,omitempty"`
}

type MusicBaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}
