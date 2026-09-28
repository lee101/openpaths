package model

import "strings"

// Modality is the kind of request a model actually serves. It is derived from
// the catalog entry rather than declared, so a new model is classified the
// moment its id or pricing lands in config.yaml and every consumer agrees.
type Modality string

const (
	ModalityChat          Modality = "chat"
	ModalityEmbedding     Modality = "embedding"
	ModalityTranscription Modality = "transcription"
	ModalitySpeech        Modality = "speech"
	ModalityMusic         Modality = "music"
	ModalityVideo         Modality = "video"
	ModalityImage         Modality = "image"
	ModalityThreeD        Modality = "3d"
	ModalityForecasting   Modality = "forecasting"
)

// ModalityOf maps a model config to the domain that serves it. Order matters:
// id markers beat pricing hints so e.g. pocket-tts (billed per minute like
// STT) still classifies as speech.
func ModalityOf(cfg *ModelConfig) Modality {
	idl := strings.ToLower(cfg.ID)
	switch {
	case strings.Contains(idl, "embed"):
		return ModalityEmbedding
	case strings.Contains(idl, "whisper"), strings.Contains(idl, "stt"),
		strings.Contains(idl, "transcribe"), strings.Contains(idl, "transcription"):
		return ModalityTranscription
	case strings.Contains(idl, "tts"), strings.Contains(idl, "voice"),
		strings.Contains(idl, "realtime"), cfg.PricePer1MCharacters > 0,
		cfg.PricePerHour > 0:
		return ModalitySpeech
	case strings.Contains(idl, "music"), strings.Contains(idl, "sfx"):
		return ModalityMusic
	case strings.Contains(idl, "to-3d"), strings.Contains(idl, "3d"):
		return ModalityThreeD
	case cfg.PricePerVideo > 0, cfg.PricePerSecond > 0, cfg.PricePerSecondWithVideoInput > 0,
		len(cfg.PricePerSecondByResolution) > 0,
		strings.Contains(idl, "video"), strings.Contains(idl, "sora"), strings.Contains(idl, "veo"):
		return ModalityVideo
	case cfg.PricePerImage > 0, len(cfg.PricePerImageByResolution) > 0,
		cfg.PricePerMegapixel > 0, cfg.PriceFirstMegapixel > 0, cfg.PriceExtraMegapixel > 0,
		strings.Contains(idl, "image"):
		return ModalityImage
	case strings.Contains(idl, "chronos"), strings.Contains(idl, "forecast"):
		return ModalityForecasting
	default:
		return ModalityChat
	}
}

// ModelType is the OpenAI-compatible type a client filters on. "language" is
// the only type a chat completion request can be served by, so a picker that
// shows language models shows exactly the models it can actually drive.
func (m Modality) ModelType() string {
	if m == ModalityChat {
		return "language"
	}
	return string(m)
}

// IsChat reports whether the model can serve a chat completion.
func (m Modality) IsChat() bool { return m == ModalityChat }
