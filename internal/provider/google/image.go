package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const maxGeminiImagesPerRequest = 4

func (p *GoogleProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusUnauthorized, Message: "missing GEMINI_API_KEY", Retryable: false}
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadRequest, Message: "prompt is required", Retryable: false}
	}
	payload, err := p.buildGeminiImageRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal google image request: %w", err)
	}

	n := req.N
	if n <= 0 {
		n = 1
	}
	if n > maxGeminiImagesPerRequest {
		n = maxGeminiImagesPerRequest
	}
	results := make([]model.ImageData, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = p.generateOneGeminiImage(ctx, body)
		}(i)
	}
	wg.Wait()

	out := &model.ImageGenerationResponse{Created: time.Now().Unix()}
	var firstErr error
	for i := range results {
		if errs[i] != nil {
			if firstErr == nil {
				firstErr = errs[i]
			}
			continue
		}
		out.Data = append(out.Data, results[i])
	}
	if len(out.Data) == 0 {
		return nil, firstErr
	}
	return out, nil
}

func (p *GoogleProvider) generateOneGeminiImage(ctx context.Context, body []byte) (model.ImageData, error) {
	endpoint := p.baseURL + "/v1beta/interactions?key=" + url.QueryEscape(p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return model.ImageData{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return model.ImageData{}, googleProviderError(err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return model.ImageData{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.ImageData{}, &provider.ProviderError{
			Provider:   "google",
			StatusCode: resp.StatusCode,
			Message:    googleErrorMessage(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		}
	}
	data, text := parseGeminiInteractionImage(respBody)
	if data == "" {
		msg := "google interaction response did not include an image"
		if text != "" {
			msg += ": " + truncateRunes(text, 200)
		}
		return model.ImageData{}, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadGateway, Message: msg, Retryable: true}
	}
	return model.ImageData{B64JSON: data, RevisedPrompt: text}, nil
}

func (p *GoogleProvider) buildGeminiImageRequest(ctx context.Context, req *model.ImageGenerationRequest) (map[string]any, error) {
	var input any = req.Prompt
	if urls := geminiImageInputURLs(req); len(urls) > 0 {
		parts := make([]any, 0, len(urls)+1)
		parts = append(parts, map[string]any{"type": "text", "text": req.Prompt})
		for _, u := range urls {
			part, err := p.geminiImageInputPartAny(ctx, u)
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		}
		input = parts
	}
	imageConfig := map[string]any{"image_size": "1K"}
	if ar := geminiAspectRatio(req); ar != "" {
		imageConfig["aspect_ratio"] = ar
	}
	return map[string]any{
		"model": normalizeGeminiImageModel(req.Model),
		"input": input,
		"generation_config": map[string]any{
			"temperature":       1,
			"max_output_tokens": geminiOmniDefaultMaxOutputTokens,
			"top_p":             0.95,
			"thinking_level":    "minimal",
			"image_config":      imageConfig,
		},
		"response_modalities": []string{"image", "text"},
		"store":               false,
	}, nil
}

func (p *GoogleProvider) geminiImageInputPartAny(ctx context.Context, ref string) (map[string]any, error) {
	if strings.HasPrefix(ref, "data:") {
		mediaType, data, ok := parseGeminiDataURL(ref)
		if !ok {
			return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadRequest, Message: "invalid image data URL", Retryable: false}
		}
		return map[string]any{"type": "image", "data": data, "mime_type": mediaType}, nil
	}
	return p.geminiImageInputPart(ctx, ref)
}

func geminiImageInputURLs(req *model.ImageGenerationRequest) []string {
	var out []string
	add := func(v string) {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	add(req.ImageURL)
	if req.Image != nil {
		add(req.Image.URL)
	}
	for _, img := range req.Images {
		add(img.URL)
	}
	for _, v := range req.ImageURLs {
		add(v)
	}
	for _, v := range req.ReferenceImageURLs {
		add(v)
	}
	return out
}

var geminiAspectRatios = []string{"1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9"}

func geminiAspectRatio(req *model.ImageGenerationRequest) string {
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		return ar
	}
	size, ok := imgutil.ParseSize(strings.TrimSpace(req.Size))
	if !ok {
		return ""
	}
	ar, _ := imgutil.MatchAspectRatio(size, geminiAspectRatios)
	return ar
}

func normalizeGeminiImageModel(name string) string {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "models/") {
		name = "models/" + name
	}
	return name
}

func parseGeminiInteractionImage(body []byte) (data, text string) {
	var doc struct {
		Steps []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Data string `json:"data"`
			} `json:"content"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", ""
	}
	var texts []string
	for _, step := range doc.Steps {
		if step.Type != "model_output" {
			continue
		}
		for _, c := range step.Content {
			switch c.Type {
			case "image":
				if data == "" {
					data = c.Data
				}
			case "text":
				if t := strings.TrimSpace(c.Text); t != "" {
					texts = append(texts, t)
				}
			}
		}
	}
	return data, strings.Join(texts, "\n")
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
