package model

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type VideoDuration string

func (d *VideoDuration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*d = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*d = VideoDuration(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*d = VideoDuration(strconv.Itoa(n))
		return nil
	}
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*d = VideoDuration(strconv.Itoa(int(f)))
		return nil
	}
	return fmt.Errorf("duration must be a string or number")
}

type VideoGenerationRequest struct {
	Model                 string             `json:"model"`
	Prompt                string             `json:"prompt"`
	Content               []VideoContentItem `json:"content,omitempty"`
	Input                 json.RawMessage    `json:"input,omitempty"`
	Async                 bool               `json:"async,omitempty"`
	Operation             string             `json:"operation,omitempty"`
	ImageURL              string             `json:"image_url,omitempty"`
	EndImageURL           string             `json:"end_image_url,omitempty"`
	ImageURLs             []string           `json:"image_urls,omitempty"`
	VideoURL              string             `json:"video_url,omitempty"`
	Video                 *VideoInput        `json:"video,omitempty"`
	VideoURLs             []string           `json:"video_urls,omitempty"`
	AudioURL              string             `json:"audio_url,omitempty"`
	AudioURLs             []string           `json:"audio_urls,omitempty"`
	SyncMode              string             `json:"sync_mode,omitempty"`
	AudioSetting          string             `json:"audio_setting,omitempty"`
	NumFrames             int                `json:"num_frames,omitempty"`
	FramesPerSecond       int                `json:"frames_per_second,omitempty"`
	Resolution            string             `json:"resolution,omitempty"`
	Duration              VideoDuration      `json:"duration,omitempty"`
	AspectRatio           string             `json:"aspect_ratio,omitempty"`
	Ratio                 string             `json:"ratio,omitempty"`
	OutputFormat          string             `json:"output_format,omitempty"`
	GenerateAudio         *bool              `json:"generate_audio,omitempty"`
	EnableThinking        *bool              `json:"enable_thinking,omitempty"`
	BitrateMode           string             `json:"bitrate_mode,omitempty"`
	EndUserID             string             `json:"end_user_id,omitempty"`
	NegativePrompt        string             `json:"negative_prompt,omitempty"`
	Seed                  *int               `json:"seed,omitempty"`
	NumInferenceSteps     int                `json:"num_inference_steps,omitempty"`
	GuidanceScale         *float64           `json:"guidance_scale,omitempty"`
	EnableSafetyChecker   *bool              `json:"enable_safety_checker,omitempty"`
	SafetyTolerance       int                `json:"safety_tolerance,omitempty"`
	UpscaleFactor         *float64           `json:"upscale_factor,omitempty"`
	Creativity            *int               `json:"creativity,omitempty"`
	PreviousInteractionID string             `json:"previous_interaction_id,omitempty"`
	GenerationConfig      json.RawMessage    `json:"generation_config,omitempty"`
	ResponseModalities    []string           `json:"response_modalities,omitempty"`
	ResponseFormat        json.RawMessage    `json:"response_format,omitempty"`
	Store                 *bool              `json:"store,omitempty"`
	Stream                *bool              `json:"stream,omitempty"`
	ControlType           string             `json:"control_type,omitempty"`
	ControlScale          *float64           `json:"control_scale,omitempty"`
	MaskVideoURL          string             `json:"mask_video_url,omitempty"`
	BackgroundColor       string             `json:"background_color,omitempty"`
}

// VideoContentItem is the multimodal content shape accepted by video models
// such as MiniMax-H3. URL-bearing items intentionally use an object so the
// request can be forwarded without losing the provider's native schema.
type VideoContentItem struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *VideoMediaURL `json:"image_url,omitempty"`
	VideoURL *VideoMediaURL `json:"video_url,omitempty"`
	AudioURL *VideoMediaURL `json:"audio_url,omitempty"`
	Role     string         `json:"role,omitempty"`
}

type VideoMediaURL struct {
	URL string `json:"url"`
}

type VideoInput struct {
	Type   string `json:"type,omitempty"`
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
}

type VideoGenerationResponse struct {
	VideoURL         string  `json:"video_url"`
	OriginalVideoURL string  `json:"original_video_url,omitempty"`
	OutputFormat     string  `json:"output_format,omitempty"`
	Bytes            int64   `json:"bytes,omitempty"`
	OriginalBytes    int64   `json:"original_bytes,omitempty"`
	Seed             *int    `json:"seed,omitempty"`
	BackendUsed      string  `json:"backend_used,omitempty"`
	CreditsCharged   float64 `json:"credits_charged,omitempty"`
	Model            string  `json:"model,omitempty"`
}

func (r VideoGenerationRequest) HasVideoInput() bool {
	if r.VideoURL != "" {
		return true
	}
	if r.Video != nil && (r.Video.URL != "" || r.Video.FileID != "") {
		return true
	}
	if len(r.VideoURLs) > 0 {
		return true
	}
	for _, item := range r.Content {
		if item.Type == "video_url" && item.VideoURL != nil && item.VideoURL.URL != "" {
			return true
		}
	}
	return false
}

func (r VideoGenerationRequest) InputImageCount() int {
	count := len(r.ImageURLs)
	if r.ImageURL != "" {
		count++
	}
	if r.EndImageURL != "" {
		count++
	}
	for _, item := range r.Content {
		if item.Type == "image_url" && item.ImageURL != nil && item.ImageURL.URL != "" {
			count++
		}
	}
	return count
}

func (r VideoGenerationRequest) TextPrompt() string {
	if r.Prompt != "" {
		return r.Prompt
	}
	for _, item := range r.Content {
		if item.Type == "text" && item.Text != "" {
			return item.Text
		}
	}
	return ""
}
