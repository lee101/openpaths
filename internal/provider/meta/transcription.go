package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const defaultTranscriptionModel = "muse-voice-transcribe-1.0"

type transcriptionRequest struct {
	Model         string `json:"model"`
	AudioEncoding string `json:"audioEncoding"`
	Language      string `json:"language,omitempty"`
	Prompt        string `json:"prompt,omitempty"`
}

type transcriptionResponse struct {
	Transcript      string          `json:"transcript"`
	AudioDurationMS int64           `json:"audioDurationMs"`
	Turns           json.RawMessage `json:"turns"`
}

func (p *Provider) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	encoding, contentType, err := audioEncoding(req.Filename, req.File)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "meta", StatusCode: http.StatusBadRequest, Message: err.Error(), Retryable: false, Err: err,
		}
	}

	modelID := req.Model
	if modelID == "" || modelID == "auto" {
		modelID = defaultTranscriptionModel
	}
	requestJSON, err := json.Marshal(transcriptionRequest{
		Model: modelID, AudioEncoding: encoding, Language: req.Language, Prompt: req.Prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal Meta transcription request: %w", err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	requestHeaders := make(textproto.MIMEHeader)
	requestHeaders.Set("Content-Disposition", `form-data; name="request"`)
	requestHeaders.Set("Content-Type", "application/json")
	requestPart, err := w.CreatePart(requestHeaders)
	if err != nil {
		return nil, fmt.Errorf("create Meta request part: %w", err)
	}
	if _, err := requestPart.Write(requestJSON); err != nil {
		return nil, fmt.Errorf("write Meta request part: %w", err)
	}

	audioHeaders := make(textproto.MIMEHeader)
	audioHeaders.Set("Content-Disposition", fmt.Sprintf(`form-data; name="audio"; filename="%s"`, escapeFilename(req.Filename)))
	audioHeaders.Set("Content-Type", contentType)
	audioPart, err := w.CreatePart(audioHeaders)
	if err != nil {
		return nil, fmt.Errorf("create Meta audio part: %w", err)
	}
	if _, err := audioPart.Write(req.File); err != nil {
		return nil, fmt.Errorf("write Meta audio part: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close Meta transcription form: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL+"/asr/transcribe", &body)
	if err != nil {
		return nil, fmt.Errorf("create Meta transcription request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "meta", StatusCode: http.StatusBadGateway, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Meta transcription response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &provider.ProviderError{
			Provider: "meta", StatusCode: resp.StatusCode, Message: string(respBody),
			Retryable: resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		}
	}

	var result transcriptionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &provider.ProviderError{
			Provider: "meta", StatusCode: http.StatusBadGateway, Message: "unmarshal: " + err.Error(), Retryable: false, Err: err,
		}
	}
	if strings.TrimSpace(result.Transcript) == "" {
		return nil, &provider.ProviderError{
			Provider: "meta", StatusCode: http.StatusBadGateway, Message: "Meta returned an empty transcript", Retryable: true,
		}
	}

	return &model.TranscriptionResponse{
		Text:            result.Transcript,
		AudioDurationMS: result.AudioDurationMS,
		Turns:           result.Turns,
	}, nil
}

func audioEncoding(filename string, data []byte) (encoding, contentType string, err error) {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return "WAV", "audio/wav", nil
	}
	if len(data) >= 4 && string(data[:4]) == "fLaC" {
		return "FLAC", "audio/flac", nil
	}
	if len(data) >= 4 && string(data[:4]) == "OggS" {
		return "OGG", "audio/ogg", nil
	}
	if len(data) >= 3 && string(data[:3]) == "ID3" || len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0 {
		return "MP3", "audio/mpeg", nil
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
		return "WEBM", "audio/webm", nil
	}
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		return "M4A", "audio/mp4", nil
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav", ".wave":
		return "WAV", "audio/wav", nil
	case ".flac":
		return "FLAC", "audio/flac", nil
	case ".ogg", ".oga", ".opus":
		return "OGG", "audio/ogg", nil
	case ".mp3":
		return "MP3", "audio/mpeg", nil
	case ".webm":
		return "WEBM", "audio/webm", nil
	case ".m4a", ".mp4":
		return "M4A", "audio/mp4", nil
	default:
		return "", "", fmt.Errorf("unsupported audio format for Meta transcription")
	}
}

func escapeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, `\`, "_")
	filename = strings.ReplaceAll(filename, `"`, "_")
	if filename == "." || filename == "" {
		return "audio.wav"
	}
	return filename
}
