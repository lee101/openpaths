package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/safefetch"
)

const geminiOmniDefaultMaxOutputTokens = 65536

func (p *GoogleProvider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusUnauthorized, Message: "missing GEMINI_API_KEY", Retryable: false}
	}

	interactionReq, err := p.buildGeminiOmniInteractionRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(interactionReq)
	if err != nil {
		return nil, fmt.Errorf("marshal google interaction request: %w", err)
	}
	endpoint := p.baseURL + "/v1beta/interactions?key=" + url.QueryEscape(p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, googleProviderError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: resp.StatusCode,
			Message:    googleErrorMessage(body),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		}
	}

	videoURL := findGeminiInteractionVideoURL(body)
	if videoURL == "" {
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: http.StatusBadGateway,
			Message:    "google interaction response did not include a video URL",
			Retryable:  true,
		}
	}
	if isGeminiInteractionDownloadURL(videoURL) {
		dataURL, err := p.downloadGeminiInteractionVideo(ctx, videoURL)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "google",
				StatusCode: http.StatusBadGateway,
				Message:    "download google interaction video: " + err.Error(),
				Retryable:  true,
			}
		}
		videoURL = dataURL
	}
	return &model.VideoGenerationResponse{
		VideoURL:     videoURL,
		OutputFormat: outputFormatFromVideoURL(videoURL),
	}, nil
}

func (p *GoogleProvider) buildGeminiOmniInteractionRequest(ctx context.Context, req *model.VideoGenerationRequest) (map[string]any, error) {
	input, err := p.geminiInteractionInput(ctx, req)
	if err != nil {
		return nil, err
	}
	store := true
	stream := false
	payload := map[string]any{
		"model":               normalizeGeminiOmniModel(req.Model),
		"input":               input,
		"generation_config":   geminiInteractionGenerationConfig(req, googleInputHasImages(input)),
		"response_modalities": geminiResponseModalities(req),
		"response_format":     geminiInteractionResponseFormat(req),
		"store":               store,
		"stream":              stream,
	}
	if req.PreviousInteractionID != "" {
		payload["previous_interaction_id"] = req.PreviousInteractionID
	}
	if req.Store != nil {
		payload["store"] = *req.Store
	}
	if req.Stream != nil {
		payload["stream"] = *req.Stream
	}
	return payload, nil
}

func normalizeGeminiOmniModel(modelName string) string {
	return strings.TrimSpace(modelName)
}

func (p *GoogleProvider) geminiInteractionInput(ctx context.Context, req *model.VideoGenerationRequest) (any, error) {
	if len(req.Input) > 0 {
		return json.RawMessage(req.Input), nil
	}
	imageURLs := geminiInputImageURLs(req)
	if len(imageURLs) == 0 {
		return req.Prompt, nil
	}
	parts := make([]any, 0, len(imageURLs)+1)
	for _, imageURL := range imageURLs {
		part, err := p.geminiImageInputPart(ctx, imageURL)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if strings.TrimSpace(req.Prompt) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": req.Prompt})
	}
	return parts, nil
}

func geminiInputImageURLs(req *model.VideoGenerationRequest) []string {
	var out []string
	if strings.TrimSpace(req.ImageURL) != "" {
		out = append(out, strings.TrimSpace(req.ImageURL))
	}
	for _, imageURL := range req.ImageURLs {
		if strings.TrimSpace(imageURL) != "" {
			out = append(out, strings.TrimSpace(imageURL))
		}
	}
	return out
}

func (p *GoogleProvider) geminiImageInputPart(ctx context.Context, imageURL string) (map[string]any, error) {
	u, err := url.Parse(imageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadRequest, Message: "image_url must be http(s)", Retryable: false}
	}
	if err := safefetch.ValidateURL(u); err != nil {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadRequest, Message: "unsafe image_url: " + err.Error(), Retryable: false}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", "openpaths-google-video/1.0")
	resp, err := p.imageClient.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadGateway, Message: "download image_url: " + err.Error(), Retryable: true}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: resp.StatusCode, Message: fmt.Sprintf("download image_url failed: HTTP %d", resp.StatusCode), Retryable: resp.StatusCode >= 500}
	}
	contentType := strings.ToLower(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(contentType, "image/") {
		contentType = http.DetectContentType(body)
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: http.StatusBadRequest, Message: "image_url did not return an image", Retryable: false}
	}
	return map[string]any{
		"type":      "image",
		"data":      base64.StdEncoding.EncodeToString(body),
		"mime_type": contentType,
	}, nil
}

func googleInputHasImages(input any) bool {
	parts, ok := input.([]any)
	if !ok {
		return false
	}
	for _, part := range parts {
		m, ok := part.(map[string]any)
		if ok && strings.EqualFold(fmt.Sprint(m["type"]), "image") {
			return true
		}
	}
	return false
}

func geminiInteractionGenerationConfig(req *model.VideoGenerationRequest, hasImageInput bool) any {
	if len(req.GenerationConfig) > 0 {
		return json.RawMessage(req.GenerationConfig)
	}
	task := "unspecified"
	if hasImageInput {
		task = "image_to_video"
	}
	return map[string]any{
		"max_output_tokens": geminiOmniDefaultMaxOutputTokens,
		"thinking_level":    "high",
		"video_config": map[string]any{
			"task": task,
		},
	}
}

func geminiResponseModalities(req *model.VideoGenerationRequest) []string {
	if len(req.ResponseModalities) > 0 {
		return req.ResponseModalities
	}
	return []string{"video"}
}

func geminiInteractionResponseFormat(req *model.VideoGenerationRequest) any {
	if len(req.ResponseFormat) > 0 {
		return json.RawMessage(req.ResponseFormat)
	}
	return map[string]any{
		"type":     "video",
		"duration": geminiDurationString(string(req.Duration)),
		"delivery": "uri",
	}
}

func geminiDurationString(raw string) string {
	d := strings.TrimSpace(raw)
	if d == "" || strings.EqualFold(d, "auto") {
		return "10s"
	}
	if strings.HasSuffix(strings.ToLower(d), "s") {
		return d
	}
	return d + "s"
}

func findGeminiInteractionVideoURL(body []byte) string {
	var root any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return ""
	}
	return walkGeminiInteractionVideo(root, "")
}

func walkGeminiInteractionVideo(v any, key string) string {
	switch x := v.(type) {
	case map[string]any:
		if url := videoDataURLFromMap(x, key); url != "" {
			return url
		}
		for _, preferred := range []string{"video_url", "videoUrl", "output_video_url", "outputVideoUrl", "download_url", "downloadUrl", "url", "uri"} {
			if s, ok := stringValue(x[preferred]); ok && isRenderableMediaURL(s) {
				return s
			}
		}
		for k, val := range x {
			if url := walkGeminiInteractionVideo(val, strings.ToLower(k)); url != "" {
				return url
			}
		}
	case []any:
		for _, val := range x {
			if url := walkGeminiInteractionVideo(val, key); url != "" {
				return url
			}
		}
	case string:
		if isRenderableMediaURL(x) && (strings.Contains(key, "video") || strings.Contains(key, "url") || strings.Contains(key, "uri")) {
			return x
		}
	}
	return ""
}

func videoDataURLFromMap(m map[string]any, key string) string {
	mimeType := firstString(m, "mime_type", "mimeType", "mime")
	if mimeType == "" {
		mimeType = "video/mp4"
	}
	typeHint := strings.ToLower(firstString(m, "type", "media_type", "mediaType"))
	isVideo := strings.Contains(key, "video") || strings.Contains(typeHint, "video") || strings.Contains(mimeType, "video/")
	if !isVideo {
		return ""
	}
	data := firstString(m, "data", "bytes", "base64", "base64_data", "base64Data", "bytes_base64_encoded", "bytesBase64Encoded")
	if data == "" {
		return ""
	}
	if strings.HasPrefix(data, "data:") {
		return data
	}
	return "data:" + mimeType + ";base64," + data
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := stringValue(m[key]); ok {
			return s
		}
	}
	return ""
}

func stringValue(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	return s, s != ""
}

func isRenderableMediaURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "data:video/")
}

func isGeminiInteractionDownloadURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return strings.Contains(u.Path, "/v1beta/files/") && strings.Contains(u.Path, ":download")
}

func (p *GoogleProvider) downloadGeminiInteractionVideo(ctx context.Context, raw string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("x-goog-api-key", p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "video/") {
		contentType = "video/mp4"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(body), nil
}

func googleErrorMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	msg := strings.TrimSpace(string(body))
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}
	if msg == "" {
		return "google interaction request failed"
	}
	return msg
}

func outputFormatFromVideoURL(videoURL string) string {
	lower := strings.ToLower(strings.Split(videoURL, "?")[0])
	for _, ext := range []string{"webm", "mov", "m4v", "mp4"} {
		if strings.HasSuffix(lower, "."+ext) {
			return ext
		}
	}
	if strings.HasPrefix(lower, "data:video/webm") {
		return "webm"
	}
	return "mp4"
}
