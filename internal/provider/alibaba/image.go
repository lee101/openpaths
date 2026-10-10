package alibaba

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

const maxQwenImageInputs = 3

// ~1MP sizes per aspect ratio keep requests in the 1K billing tier.
var qwenImageAspectSizes = map[string]string{
	"1:1": "1024*1024", "16:9": "1376*768", "9:16": "768*1376", "4:3": "1184*888", "3:4": "888*1184",
	"3:2": "1248*832", "2:3": "832*1248", "21:9": "1568*672", "9:21": "672*1568",
}

type imageResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Output  struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Image string `json:"image"`
					Text  string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		OutputWidth     int    `json:"output_width"`
		OutputHeight    int    `json:"output_height"`
		OutputImageType string `json:"output_image_type"`
	} `json:"usage"`
}

// GenerateImage serves qwen-image-3.0 / -pro text-to-image and editing (up to 3 inputs).
func (p *Provider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	payload, err := qwenImagePayload(req)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: err.Error(), Retryable: false, Err: err}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+"/api/v1/services/aigc/multimodal-generation/generation", body, false)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	var out imageResponse
	_ = json.Unmarshal(responseBody, &out)
	if status != http.StatusOK {
		msg := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(out.Code+" "+out.Message), string(responseBody)))
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: msg, Retryable: status == 429 || status >= 500}
	}
	resp := &model.ImageGenerationResponse{Created: time.Now().Unix()}
	for _, choice := range out.Output.Choices {
		for _, item := range choice.Message.Content {
			if item.Image != "" {
				resp.Data = append(resp.Data, model.ImageData{URL: item.Image, Width: out.Usage.OutputWidth, Height: out.Usage.OutputHeight})
			}
		}
	}
	if len(resp.Data) == 0 {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "Qwen Image returned no images", Retryable: true}
	}
	// Bill the tier Alibaba actually delivered (qima_output_1k / qima_output_2k).
	switch {
	case strings.HasSuffix(out.Usage.OutputImageType, "_2k"):
		req.Resolution = "2k"
	case strings.HasSuffix(out.Usage.OutputImageType, "_1k"):
		req.Resolution = "1k"
	case out.Usage.OutputWidth*out.Usage.OutputHeight > 2250000:
		req.Resolution = "2k"
	default:
		req.Resolution = "1k"
	}
	return resp, nil
}

func qwenImagePayload(req *model.ImageGenerationRequest) (map[string]any, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	inputs := qwenImageInputs(req)
	if len(inputs) > maxQwenImageInputs {
		return nil, fmt.Errorf("Qwen Image accepts at most %d input images", maxQwenImageInputs)
	}
	content := make([]map[string]string, 0, len(inputs)+1)
	for _, url := range inputs {
		content = append(content, map[string]string{"image": url})
	}
	content = append(content, map[string]string{"text": prompt})
	n := req.N
	if n <= 0 {
		n = req.NumImages
	}
	params := map[string]any{"n": min(max(n, 1), 6), "watermark": false}
	if size, err := qwenImageSize(req, len(inputs) > 0); err != nil {
		return nil, err
	} else if size != "" {
		params["size"] = size
	}
	if req.Seed != nil {
		params["seed"] = *req.Seed
	}
	return map[string]any{
		"model":      req.Model,
		"input":      map[string]any{"messages": []map[string]any{{"role": "user", "content": content}}},
		"parameters": params,
	}, nil
}

func qwenImageSize(req *model.ImageGenerationRequest, edit bool) (string, error) {
	if raw := strings.TrimSpace(req.Size); raw != "" && !strings.EqualFold(raw, "auto") {
		size, ok := imgutil.ParseSize(strings.ReplaceAll(raw, "*", "x"))
		if !ok || size.W*size.H < 512*512 || size.W*size.H > 2560*2560 || size.W > 8*size.H || size.H > 8*size.W {
			return "", fmt.Errorf("size must be WxH between 512x512 and 2560x2560 total pixels, aspect at most 8:1")
		}
		return fmt.Sprintf("%d*%d", size.W, size.H), nil
	}
	if size, ok := qwenImageAspectSizes[strings.TrimSpace(req.AspectRatio)]; ok {
		return size, nil
	}
	if edit {
		return "", nil // keep the input aspect; billed at the delivered tier
	}
	return "1024*1024", nil
}

func qwenImageInputs(req *model.ImageGenerationRequest) []string {
	values := make([]string, 0, 3)
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
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
	return values
}
