package seedream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

// GenerateVideo submits a ModelArk contents/generations task (Seedance) and
// polls it to completion. Content items pass through in ModelArk's native
// shape (text, image_url first_frame/last_frame/reference_image,
// video_url reference_video, audio_url reference_audio).
func (p *Provider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	payload := map[string]any{
		"model":     req.Model,
		"content":   videoContent(req),
		"watermark": false,
	}
	ratio := req.Ratio
	if ratio == "" {
		ratio = req.AspectRatio
	}
	if ratio != "" {
		payload["ratio"] = ratio
	}
	if req.Resolution != "" {
		payload["resolution"] = req.Resolution
	}
	if d := strings.TrimSpace(string(req.Duration)); d != "" && d != "auto" {
		n, err := strconv.Atoi(d)
		if err != nil {
			return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: 400, Message: "duration must be an integer number of seconds", Err: err}
		}
		payload["duration"] = n
	}
	if req.GenerateAudio != nil {
		payload["generate_audio"] = *req.GenerateAudio
	}
	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}
	body, _ := json.Marshal(payload)

	var task struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Content struct {
			VideoURL string `json:"video_url"`
		} `json:"content"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := p.do(ctx, http.MethodPost, p.baseURL+"/contents/generations/tasks", body, &task); err != nil {
		return nil, err
	}
	if task.ID == "" {
		return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: 502, Message: "ModelArk returned no task id", Retryable: true}
	}
	for task.Status != "succeeded" {
		switch task.Status {
		case "failed", "cancelled", "expired":
			msg := "ModelArk video task " + task.Status
			if task.Error != nil && task.Error.Message != "" {
				msg = task.Error.Message
			}
			return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: 502, Message: msg}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.pollInterval):
		}
		if err := p.do(ctx, http.MethodGet, p.baseURL+"/contents/generations/tasks/"+task.ID, nil, &task); err != nil {
			return nil, err
		}
	}
	if task.Content.VideoURL == "" {
		return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: 502, Message: "ModelArk completed without a video URL", Retryable: true}
	}
	return &model.VideoGenerationResponse{VideoURL: task.Content.VideoURL, OutputFormat: "mp4", Model: req.Model, BackendUsed: "seedream"}, nil
}

func videoContent(req *model.VideoGenerationRequest) []map[string]any {
	var out []map[string]any
	media := func(typ, url, role string) {
		if url == "" {
			return
		}
		item := map[string]any{"type": typ, typ: map[string]string{"url": url}}
		if role != "" {
			item["role"] = role
		}
		out = append(out, item)
	}
	if len(req.Content) > 0 {
		for _, c := range req.Content {
			switch {
			case c.Type == "text":
				out = append(out, map[string]any{"type": "text", "text": c.Text})
			case c.ImageURL != nil:
				media("image_url", c.ImageURL.URL, c.Role)
			case c.VideoURL != nil:
				media("video_url", c.VideoURL.URL, c.Role)
			case c.AudioURL != nil:
				media("audio_url", c.AudioURL.URL, c.Role)
			}
		}
		return out
	}
	if t := req.TextPrompt(); t != "" {
		out = append(out, map[string]any{"type": "text", "text": t})
	}
	images := append([]string{}, req.ImageURLs...)
	if req.ImageURL != "" {
		images = append([]string{req.ImageURL}, images...)
	}
	switch {
	case len(req.ReferenceImageURLs) > 0 || req.VideoURL != "" || len(req.VideoURLs) > 0 || req.AudioURL != "" || len(req.AudioURLs) > 0:
		for _, u := range append(images, req.ReferenceImageURLs...) {
			media("image_url", u, "reference_image")
		}
	case len(images) > 0:
		media("image_url", images[0], "first_frame")
		if req.EndImageURL != "" {
			media("image_url", req.EndImageURL, "last_frame")
		} else if len(images) > 1 {
			media("image_url", images[1], "last_frame")
		}
	}
	vids := append([]string{}, req.VideoURLs...)
	if req.VideoURL != "" {
		vids = append([]string{req.VideoURL}, vids...)
	} else if req.Video != nil && req.Video.URL != "" {
		vids = append([]string{req.Video.URL}, vids...)
	}
	for _, u := range vids {
		media("video_url", u, "reference_video")
	}
	auds := append([]string{}, req.AudioURLs...)
	if req.AudioURL != "" {
		auds = append([]string{req.AudioURL}, auds...)
	}
	for _, u := range auds {
		media("audio_url", u, "reference_audio")
	}
	return out
}

func (p *Provider) do(ctx context.Context, method, url string, body []byte, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return &baseprovider.ProviderError{Provider: "seedream", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return &baseprovider.ProviderError{Provider: "seedream", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw)),
			Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}
