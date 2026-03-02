package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

type Service struct {
	providers []model.ProviderConfig
	modelQ    *queries.ModelMetadataQueries
	client    *http.Client
}

func New(providers []model.ProviderConfig, modelQ *queries.ModelMetadataQueries) *Service {
	return &Service{
		providers: providers,
		modelQ:    modelQ,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Service) DiscoverAll(ctx context.Context) (int, error) {
	total := 0
	for _, p := range s.providers {
		if !p.Enabled || p.APIKey == "" {
			continue
		}
		n, err := s.discoverProvider(ctx, p)
		if err != nil {
			log.Printf("discovery %s: %v", p.Name, err)
			continue
		}
		total += n
		log.Printf("discovery %s: indexed %d models", p.Name, n)
	}
	return total, nil
}

func (s *Service) discoverProvider(ctx context.Context, p model.ProviderConfig) (int, error) {
	switch p.Name {
	case "mistral":
		return s.discoverMistral(ctx, p)
	case "openai":
		return s.discoverOpenAI(ctx, p)
	case "openrouter":
		return s.discoverOpenRouter(ctx, p)
	case "together":
		return s.discoverTogether(ctx, p)
	case "groq":
		return s.discoverOpenAICompat(ctx, p, "groq")
	case "xai":
		return s.discoverOpenAICompat(ctx, p, "xai")
	case "deepseek":
		return s.discoverOpenAICompat(ctx, p, "deepseek")
	default:
		return 0, nil
	}
}

type mistralModel struct {
	ID              string   `json:"id"`
	Object          string   `json:"object"`
	Created         int64    `json:"created"`
	OwnedBy         string   `json:"owned_by"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	MaxContextLen   int      `json:"max_context_length"`
	Aliases         []string `json:"aliases"`
	DefaultTemp     *float64 `json:"default_model_temperature"`
	Type            string   `json:"TYPE"`
	Deprecation     any      `json:"deprecation"`
	Capabilities    struct {
		Chat           bool `json:"completion_chat"`
		FIM            bool `json:"completion_fim"`
		FunctionCall   bool `json:"function_calling"`
		FineTuning     bool `json:"fine_tuning"`
		Vision         bool `json:"vision"`
		Classification bool `json:"classification"`
	} `json:"capabilities"`
}

func (s *Service) discoverMistral(ctx context.Context, p model.ProviderConfig) (int, error) {
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []mistralModel `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Data {
		if m.Deprecation != nil {
			depStr := fmt.Sprintf("%v", m.Deprecation)
			if depStr != "" && depStr != "<nil>" && depStr != "None" {
				continue
			}
		}

		features := []string{}
		modType := "chat"
		if m.Capabilities.Chat {
			features = append(features, "chat")
		}
		if m.Capabilities.FIM {
			features = append(features, "fim")
		}
		if m.Capabilities.FunctionCall {
			features = append(features, "tools")
		}
		if m.Capabilities.Vision {
			features = append(features, "vision")
		}
		if m.Capabilities.FineTuning {
			features = append(features, "fine_tuning")
		}

		if strings.Contains(m.ID, "embed") {
			modType = "embedding"
		} else if strings.Contains(m.ID, "moderation") {
			modType = "moderation"
		} else if strings.Contains(m.ID, "ocr") {
			modType = "ocr"
		} else if strings.Contains(m.ID, "voxtral") {
			modType = "voice"
		}

		featJSON, _ := json.Marshal(features)
		rawJSON, _ := json.Marshal(m)

		inputMod := []string{"text"}
		outputMod := []string{"text"}
		if m.Capabilities.Vision {
			inputMod = append(inputMod, "image")
		}

		inputModJSON, _ := json.Marshal(inputMod)
		outputModJSON, _ := json.Marshal(outputMod)

		meta := &queries.ModelMetadata{
			Provider:         "mistral",
			ModelID:          m.ID,
			DisplayName:      m.Name,
			Organization:     m.OwnedBy,
			ModelType:        modType,
			ContextLength:    m.MaxContextLen,
			Features:         featJSON,
			InputModalities:  inputModJSON,
			OutputModalities: outputModJSON,
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery mistral upsert %s: %v", m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

type openaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func (s *Service) discoverOpenAI(ctx context.Context, p model.ProviderConfig) (int, error) {
	return s.discoverOpenAICompat(ctx, p, "openai")
}

func (s *Service) discoverOpenAICompat(ctx context.Context, p model.ProviderConfig, provName string) (int, error) {
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var result struct {
		Data []openaiModel `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Data {
		modType := "chat"
		if strings.Contains(m.ID, "embed") {
			modType = "embedding"
		} else if strings.Contains(m.ID, "tts") || strings.Contains(m.ID, "whisper") {
			modType = "audio"
		} else if strings.Contains(m.ID, "dall-e") || strings.Contains(m.ID, "image") {
			modType = "image"
		}

		rawJSON, _ := json.Marshal(m)
		meta := &queries.ModelMetadata{
			Provider:         provName,
			ModelID:          m.ID,
			Organization:     m.OwnedBy,
			ModelType:        modType,
			Features:         json.RawMessage(`["chat"]`),
			InputModalities:  json.RawMessage(`["text"]`),
			OutputModalities: json.RawMessage(`["text"]`),
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery %s upsert %s: %v", provName, m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

type openRouterModel struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	ContextLength int     `json:"context_length"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
		Image      string `json:"image"`
	} `json:"pricing"`
	Architecture struct {
		Modality     string `json:"modality"`
		Tokenizer    string `json:"tokenizer"`
		InstructType string `json:"instruct_type"`
	} `json:"architecture"`
}

func (s *Service) discoverOpenRouter(ctx context.Context, p model.ProviderConfig) (int, error) {
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}

	var result struct {
		Data []openRouterModel `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Data {
		rawJSON, _ := json.Marshal(m)
		parts := strings.SplitN(m.ID, "/", 2)
		org := ""
		if len(parts) == 2 {
			org = parts[0]
		}

		modType := "chat"
		modality := strings.ToLower(m.Architecture.Modality)
		if strings.Contains(modality, "image") {
			modType = "image"
		}

		meta := &queries.ModelMetadata{
			Provider:         "openrouter",
			ModelID:          m.ID,
			DisplayName:      m.Name,
			Organization:     org,
			ModelType:        modType,
			ContextLength:    m.ContextLength,
			Features:         json.RawMessage(`["chat"]`),
			InputModalities:  json.RawMessage(`["text"]`),
			OutputModalities: json.RawMessage(`["text"]`),
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery openrouter upsert %s: %v", m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

func (s *Service) discoverTogether(ctx context.Context, p model.ProviderConfig) (int, error) {
	return s.discoverOpenAICompat(ctx, p, "together")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
