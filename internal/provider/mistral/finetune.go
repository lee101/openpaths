package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func (p *MistralProvider) UploadFineTuneFile(ctx context.Context, filename string, data []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return "", fmt.Errorf("write data: %w", err)
	}
	if err := w.WriteField("purpose", "fine-tune"); err != nil {
		return "", fmt.Errorf("write purpose: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/files", &buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", &provider.ProviderError{Provider: "mistral", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", &provider.ProviderError{Provider: "mistral", StatusCode: resp.StatusCode, Message: string(body), Retryable: resp.StatusCode >= 500}
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}
	return result.ID, nil
}

func (p *MistralProvider) CreateFineTuneJob(ctx context.Context, req *model.FineTuneJobRequest, provTrainFileID string, provValFileID string) (*model.ProviderFineTuneJob, error) {
	payload := map[string]any{
		"model":          req.Model,
		"training_files": []map[string]any{{"file_id": provTrainFileID, "weight": 1}},
	}
	if provValFileID != "" {
		payload["validation_files"] = []string{provValFileID}
	}
	if req.Hyperparameters != nil {
		hp := map[string]any{}
		if req.Hyperparameters.NEpochs != nil {
			hp["training_steps"] = *req.Hyperparameters.NEpochs * 100
		}
		if req.Hyperparameters.LearningRate != nil {
			hp["learning_rate"] = *req.Hyperparameters.LearningRate
		}
		if len(hp) > 0 {
			payload["hyperparameters"] = hp
		}
	}
	if req.Suffix != "" {
		payload["suffix"] = req.Suffix
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/fine_tuning/jobs", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "mistral", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{Provider: "mistral", StatusCode: resp.StatusCode, Message: string(respBody), Retryable: resp.StatusCode >= 500}
	}

	return parseMistralJob(respBody)
}

func (p *MistralProvider) GetFineTuneJob(ctx context.Context, providerJobID string) (*model.ProviderFineTuneJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v1/fine_tuning/jobs/"+providerJobID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "mistral", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{Provider: "mistral", StatusCode: resp.StatusCode, Message: string(body), Retryable: resp.StatusCode >= 500}
	}

	return parseMistralJob(body)
}

func (p *MistralProvider) CancelFineTuneJob(ctx context.Context, providerJobID string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/fine_tuning/jobs/"+providerJobID+"/cancel", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &provider.ProviderError{Provider: "mistral", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return &provider.ProviderError{Provider: "mistral", StatusCode: resp.StatusCode, Message: string(body)}
	}
	return nil
}

func parseMistralJob(data []byte) (*model.ProviderFineTuneJob, error) {
	var raw struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		FineTunedModel string `json:"fine_tuned_model"`
		TrainedTokens  int64  `json:"trained_tokens"`
		Error          any    `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	job := &model.ProviderFineTuneJob{
		ProviderJobID:  raw.ID,
		Status:         mapMistralStatus(raw.Status),
		FineTunedModel: raw.FineTunedModel,
		TrainedTokens:  raw.TrainedTokens,
	}
	if raw.Error != nil {
		if errStr, ok := raw.Error.(string); ok {
			job.Error = errStr
		} else if errMap, ok := raw.Error.(map[string]any); ok {
			if msg, ok := errMap["message"].(string); ok {
				job.Error = msg
			}
		}
	}
	return job, nil
}

func mapMistralStatus(s string) string {
	switch s {
	case "QUEUED":
		return "queued"
	case "STARTED", "RUNNING":
		return "running"
	case "SUCCESS":
		return "succeeded"
	case "FAILED", "FAILED_VALIDATION":
		return "failed"
	case "CANCELLED":
		return "cancelled"
	default:
		return s
	}
}
