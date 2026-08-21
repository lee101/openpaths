package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/safefetch"
)

type openaiImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

func (p *OpenAIProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	inputURLs := imageInputURLs(req)
	if p.providerName == "openai" && len(inputURLs) > 0 {
		return p.editImage(ctx, req, inputURLs)
	}

	out := openaiImageRequest{
		Model:          req.Model,
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
	}

	// GPT Image models return b64_json by default and do not accept response_format.
	// dall-e-3 is now served by the same images backend and also rejects
	// response_format ("Unknown parameter: 'response_format'"); only legacy
	// dall-e-2 still honours it. The handler rehosts URL responses regardless,
	// so dropping the param just lets the provider use its default format.
	m := strings.ToLower(req.Model)
	isGPTImage := strings.HasPrefix(m, "gpt-image")
	rejectsResponseFormat := isGPTImage || strings.HasPrefix(m, "dall-e-3")
	if rejectsResponseFormat {
		out.ResponseFormat = ""
	}
	if isGPTImage {
		// GPT Image models don't support "style"; they use quality only.
		out.Style = ""
	}
	if !rejectsResponseFormat && out.ResponseFormat == "" {
		out.ResponseFormat = "b64_json"
	}

	body, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "openai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var result model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// editImage adapts OpenPaths' URL-based image input contract to OpenAI's
// multipart image-edit endpoint. OpenAI expects uploaded image bytes here;
// the URL is fetched server-side after the request has passed OpenPaths'
// normal public-image validation.
func (p *OpenAIProvider) editImage(ctx context.Context, req *model.ImageGenerationRequest, inputURLs []string) (*model.ImageGenerationResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for index, imageURL := range inputURLs {
		data, contentType, filename, err := downloadImageInput(ctx, imageURL)
		if err != nil {
			return nil, &provider.ProviderError{Provider: p.providerName, StatusCode: http.StatusBadRequest, Message: err.Error(), Retryable: false, Err: err}
		}
		field := "image"
		if index > 0 {
			field = "image[]"
		}
		part, err := writer.CreateFormFile(field, filename)
		if err != nil {
			return nil, fmt.Errorf("create image form part: %w", err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("write image form part: %w", err)
		}
		_ = contentType // CreateFormFile sets the filename; OpenAI detects the image bytes.
	}
	fields := map[string]string{
		"model":  req.Model,
		"prompt": req.Prompt,
	}
	if req.N > 0 {
		fields["n"] = fmt.Sprintf("%d", req.N)
	}
	if req.Size != "" {
		fields["size"] = req.Size
	}
	if req.Quality != "" {
		fields["quality"] = req.Quality
	}
	if req.ResponseFormat != "" {
		fields["response_format"] = req.ResponseFormat
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("write image edit field %s: %w", key, err)
		}
	}
	formContentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close image edit form: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/images/edits", &body)
	if err != nil {
		return nil, fmt.Errorf("create image edit request: %w", err)
	}
	httpReq.Header.Set("Content-Type", formContentType)
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: p.providerName, StatusCode: http.StatusBadGateway, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read image edit response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &provider.ProviderError{Provider: p.providerName, StatusCode: resp.StatusCode, Message: string(respBody), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}
	var result model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal image edit response: %w", err)
	}
	return &result, nil
}

func imageInputURLs(req *model.ImageGenerationRequest) []string {
	urls := make([]string, 0, 8)
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			urls = append(urls, value)
		}
	}
	add(req.ImageURL)
	if req.Image != nil {
		add(req.Image.URL)
	}
	for _, image := range req.Images {
		add(image.URL)
	}
	for _, value := range req.ImageURLs {
		add(value)
	}
	for _, value := range req.ReferenceImageURLs {
		add(value)
	}
	return urls
}

func fetchImageInput(ctx context.Context, imageURL string) ([]byte, string, string, error) {
	u, err := url.Parse(imageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, "", "", fmt.Errorf("image input must be an absolute http(s) URL")
	}
	if err := safefetch.ValidateURL(u); err != nil {
		return nil, "", "", fmt.Errorf("unsafe image input: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("create image input request: %w", err)
	}
	resp, err := imageFetchClient.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("download image input: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", "", fmt.Errorf("download image input failed: HTTP %d", resp.StatusCode)
	}
	const maxImageBytes = 20 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, "", "", fmt.Errorf("read image input: %w", err)
	}
	if len(data) == 0 || len(data) > maxImageBytes {
		return nil, "", "", fmt.Errorf("image input must be between 1 byte and 20 MB")
	}
	contentType := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	if !strings.HasPrefix(contentType, "image/") {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", "", fmt.Errorf("image input did not return an image")
	}
	filename := filepath.Base(u.Path)
	if filename == "." || filename == "/" || filename == "" || !strings.Contains(filename, ".") {
		filename = "input." + strings.TrimPrefix(strings.TrimPrefix(contentType, "image/"), "x-")
	}
	return data, contentType, filename, nil
}

var imageFetchClient = safefetch.NewClient(2 * time.Minute)
var downloadImageInput = fetchImageInput
