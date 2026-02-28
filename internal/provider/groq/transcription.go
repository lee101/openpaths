package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

type GroqTranscriber struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewTranscriber(apiKey, baseURL string) *GroqTranscriber {
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai"
	}
	return &GroqTranscriber{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (t *GroqTranscriber) Name() string { return "groq" }

func (t *GroqTranscriber) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("file", req.Filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(req.File); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}

	mdl := req.Model
	if mdl == "" || mdl == "auto" {
		mdl = "whisper-large-v3-turbo"
	}
	w.WriteField("model", mdl)

	if req.Language != "" {
		w.WriteField("language", req.Language)
	}
	if req.Prompt != "" {
		w.WriteField("prompt", req.Prompt)
	}
	respFmt := req.Format
	if respFmt == "" {
		respFmt = "json"
	}
	w.WriteField("response_format", respFmt)
	w.Close()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+"/v1/audio/transcriptions", &buf)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "groq", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "groq", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "groq", StatusCode: 502, Message: "read response: " + err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "groq",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var result model.TranscriptionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &provider.ProviderError{
			Provider: "groq", StatusCode: 502, Message: "unmarshal: " + err.Error(), Retryable: false,
		}
	}

	return &result, nil
}
