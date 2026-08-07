package queries

import "testing"

func TestClassifyProduct(t *testing.T) {
	cases := map[string]string{
		// chat / text
		"claude-opus-4-8":      "chat",
		"gpt-5.5":              "chat",
		"openpaths/auto":       "chat",
		"openpaths/auto-code":  "chat",
		"deepseek-reasoner":    "chat",
		"gemini-2.5-pro":       "chat",
		"qwen3-coder":          "chat",
		"minimax-m2.7":         "chat",
		"openpaths/auto-vision": "chat",
		// image
		"flux-schnell":          "image",
		"openpaths/auto-image":  "image",
		"grok-imagine-image":    "image",
		"stable-diffusion-3":    "image",
		"glm-image":             "image",
		"zimage":                "image",
		"hidream-o1-image-dev":  "image",
		"ra1":                   "image",
		"smart-resize":          "image",
		"fal-ai/flux-2-pro/outpaint": "image",
		// video (must beat image even when name contains "image")
		"seedance-2.0-image-to-video":       "video",
		"alibaba/happy-horse/image-to-video": "video",
		"ltx-2.3-image-to-video":            "video",
		"hailuo-2.3":                        "video",
		"wan":                               "video",
		"ra2v":                              "video",
		// 3d (must beat image)
		"pixal3d-image-to-3d":  "3d",
		"meshy-v6-image-to-3d": "3d",
		"tripo-p1-image-to-3d": "3d",
		"trellis-2-retexture":  "3d",
		"meshy-rigging":        "3d",
		"text-to-3d":           "3d",
		// embedding
		"openpaths-embed":         "embedding",
		"gemini-embedding-001":    "embedding",
		"text-embedding":          "embedding",
		// speech / tts
		"gemini-3.1-flash-tts-preview": "speech",
		"xai-tts":                      "speech",
		"grok-voice-think-fast-1.0":    "speech",
		// transcription
		"xai-stt":      "transcription",
		"whisper-1":    "transcription",
		// music
		"lyria-3-pro-preview": "music",
	}
	for in, want := range cases {
		if got := ClassifyProduct(in); got != want {
			t.Errorf("ClassifyProduct(%q) = %q, want %q", in, got, want)
		}
	}
}
