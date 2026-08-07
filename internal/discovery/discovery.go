package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
	case "cerebras":
		return s.discoverOpenAICompat(ctx, p, "cerebras")
	case "nvidia":
		return s.discoverOpenAICompat(ctx, p, "nvidia")
	case "fireworks":
		return s.discoverOpenAICompat(ctx, p, "fireworks")
	case "anthropic":
		return s.discoverAnthropic(ctx, p)
	case "google":
		return s.discoverGoogle(ctx, p)
	case "nous":
		return s.discoverNous(ctx, p)
	case "minimax":
		return s.discoverMiniMax(ctx, p)
	case "zai":
		return s.discoverZAI(ctx, p)
	default:
		return 0, nil
	}
}

// --- Mistral ---

type mistralModel struct {
	ID            string   `json:"id"`
	Object        string   `json:"object"`
	Created       int64    `json:"created"`
	OwnedBy       string   `json:"owned_by"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	MaxContextLen int      `json:"max_context_length"`
	Aliases       []string `json:"aliases"`
	DefaultTemp   *float64 `json:"default_model_temperature"`
	Type          string   `json:"TYPE"`
	Deprecation   any      `json:"deprecation"`
	Capabilities  struct {
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

// --- OpenAI / OpenAI-compatible ---

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
		// some providers (together) return a bare array
		if err2 := json.Unmarshal(body, &result.Data); err2 != nil {
			return 0, err
		}
	}

	count := 0
	for _, m := range result.Data {
		modType := "chat"
		if strings.Contains(m.ID, "embed") {
			modType = "embedding"
		} else if strings.Contains(m.ID, "tts") {
			modType = "tts"
		} else if strings.Contains(m.ID, "whisper") || strings.Contains(m.ID, "transcribe") {
			modType = "stt"
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

// --- OpenRouter (enriched) ---

type openRouterModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Created       int64  `json:"created"`
	ContextLength int    `json:"context_length"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
		Image      string `json:"image"`
		Request    string `json:"request"`
	} `json:"pricing"`
	Architecture struct {
		Modality     string `json:"modality"`
		Tokenizer    string `json:"tokenizer"`
		InstructType string `json:"instruct_type"`
	} `json:"architecture"`
	TopProvider struct {
		MaxCompletionTokens int  `json:"max_completion_tokens"`
		IsModerated         bool `json:"is_moderated"`
	} `json:"top_provider"`
	PerRequestLimits any `json:"per_request_limits"`
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
		inputMod := []string{"text"}
		outputMod := []string{"text"}

		if strings.Contains(modality, "image") {
			if strings.Contains(modality, "text") {
				// text+image -> text (vision model)
				inputMod = append(inputMod, "image")
			} else {
				modType = "image"
				outputMod = []string{"image"}
			}
		}

		features := []string{"chat"}
		if m.TopProvider.MaxCompletionTokens > 0 {
			features = append(features, "streaming")
		}

		// Parse pricing to store as per-1M-token prices
		inputPrice := parseTokenPrice(m.Pricing.Prompt)
		outputPrice := parseTokenPrice(m.Pricing.Completion)
		imagePrice := parseTokenPrice(m.Pricing.Image)

		featJSON, _ := json.Marshal(features)
		inputModJSON, _ := json.Marshal(inputMod)
		outputModJSON, _ := json.Marshal(outputMod)

		meta := &queries.ModelMetadata{
			Provider:         "openrouter",
			ModelID:          m.ID,
			DisplayName:      m.Name,
			Organization:     org,
			ModelType:        modType,
			ContextLength:    m.ContextLength,
			InputPrice:       inputPrice * 1_000_000, // store as per-1M
			OutputPrice:      outputPrice * 1_000_000,
			PricePerImage:    imagePrice,
			Features:         featJSON,
			InputModalities:  inputModJSON,
			OutputModalities: outputModJSON,
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

// --- Anthropic ---

type anthropicModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	Type        string `json:"type"`
}

func (s *Service) discoverAnthropic(ctx context.Context, p model.ProviderConfig) (int, error) {
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
		Data []anthropicModel `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Data {
		features := []string{"chat", "tools", "streaming"}
		inputMod := []string{"text", "image"}
		outputMod := []string{"text"}

		featJSON, _ := json.Marshal(features)
		inputModJSON, _ := json.Marshal(inputMod)
		outputModJSON, _ := json.Marshal(outputMod)
		rawJSON, _ := json.Marshal(m)

		meta := &queries.ModelMetadata{
			Provider:         "anthropic",
			ModelID:          m.ID,
			DisplayName:      m.DisplayName,
			Organization:     "anthropic",
			ModelType:        "chat",
			ContextLength:    200000, // Anthropic models list doesn't always expose this
			Features:         featJSON,
			InputModalities:  inputModJSON,
			OutputModalities: outputModJSON,
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery anthropic upsert %s: %v", m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

// --- Google (Gemini) ---

type geminiModel struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	InputTokenLimit            int      `json:"inputTokenLimit"`
	OutputTokenLimit           int      `json:"outputTokenLimit"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

func (s *Service) discoverGoogle(ctx context.Context, p model.ProviderConfig) (int, error) {
	url := fmt.Sprintf("%s/v1beta/models?key=%s",
		strings.TrimRight(p.BaseURL, "/"), p.APIKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

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
		Models []geminiModel `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Models {
		// Extract model ID from "models/gemini-2.5-pro" format
		modelID := m.Name
		if strings.HasPrefix(modelID, "models/") {
			modelID = strings.TrimPrefix(modelID, "models/")
		}

		modType := "chat"
		features := []string{}
		inputMod := []string{"text"}
		outputMod := []string{"text"}

		for _, method := range m.SupportedGenerationMethods {
			switch method {
			case "generateContent":
				features = append(features, "chat")
			case "streamGenerateContent":
				features = append(features, "streaming")
			case "embedContent":
				modType = "embedding"
			}
		}

		// Gemini models generally support vision
		if strings.Contains(modelID, "gemini") && !strings.Contains(modelID, "embed") {
			inputMod = append(inputMod, "image")
			features = append(features, "vision")
		}

		featJSON, _ := json.Marshal(features)
		inputModJSON, _ := json.Marshal(inputMod)
		outputModJSON, _ := json.Marshal(outputMod)
		rawJSON, _ := json.Marshal(m)

		meta := &queries.ModelMetadata{
			Provider:         "google",
			ModelID:          modelID,
			DisplayName:      m.DisplayName,
			Organization:     "google",
			ModelType:        modType,
			ContextLength:    m.InputTokenLimit,
			Features:         featJSON,
			InputModalities:  inputModJSON,
			OutputModalities: outputModJSON,
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery google upsert %s: %v", modelID, err)
			continue
		}
		count++
	}
	return count, nil
}

// --- Nous Research (OpenAI-compatible) ---

func (s *Service) discoverNous(ctx context.Context, p model.ProviderConfig) (int, error) {
	return s.discoverOpenAICompat(ctx, p, "nous")
}

// --- MiniMax ---

type minimaxModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func (s *Service) discoverMiniMax(ctx context.Context, p model.ProviderConfig) (int, error) {
	// MiniMax uses OpenAI-compatible format at their API
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("minimax models endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// MiniMax might not have a standard models endpoint; skip gracefully
		log.Printf("discovery minimax: models endpoint returned %d, skipping", resp.StatusCode)
		return 0, nil
	}

	var result struct {
		Data []minimaxModel `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range result.Data {
		modType := "chat"
		if strings.Contains(strings.ToLower(m.ID), "hailuo") || strings.Contains(strings.ToLower(m.ID), "video") {
			modType = "video"
		} else if strings.Contains(strings.ToLower(m.ID), "music") {
			modType = "music"
		} else if strings.Contains(strings.ToLower(m.ID), "speech") {
			modType = "tts"
		}

		rawJSON, _ := json.Marshal(m)
		meta := &queries.ModelMetadata{
			Provider:         "minimax",
			ModelID:          m.ID,
			Organization:     "minimax",
			ModelType:        modType,
			Features:         json.RawMessage(`["chat","streaming"]`),
			InputModalities:  json.RawMessage(`["text"]`),
			OutputModalities: json.RawMessage(`["text"]`),
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery minimax upsert %s: %v", m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

// --- Z.AI (ChatGLM) ---

func (s *Service) discoverZAI(ctx context.Context, p model.ProviderConfig) (int, error) {
	// Z.AI uses OpenAI-compatible format
	url := strings.TrimRight(p.BaseURL, "/") + "/v1/models"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("zai models endpoint: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		// Z.AI may not have a standard endpoint; skip gracefully
		log.Printf("discovery zai: models endpoint returned %d, skipping", resp.StatusCode)
		return 0, nil
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
		inputMod := []string{"text"}
		outputMod := []string{"text"}

		if strings.Contains(m.ID, "image") {
			modType = "image"
			outputMod = []string{"image"}
		} else if strings.Contains(m.ID, "video") {
			modType = "video"
			outputMod = []string{"video"}
		} else if strings.Contains(m.ID, "4.6v") || strings.Contains(m.ID, "vision") {
			inputMod = append(inputMod, "image")
		}

		inputModJSON, _ := json.Marshal(inputMod)
		outputModJSON, _ := json.Marshal(outputMod)
		rawJSON, _ := json.Marshal(m)

		meta := &queries.ModelMetadata{
			Provider:         "zai",
			ModelID:          m.ID,
			Organization:     "zhipu",
			ModelType:        modType,
			Features:         json.RawMessage(`["chat","tools","streaming"]`),
			InputModalities:  inputModJSON,
			OutputModalities: outputModJSON,
			RawMetadata:      rawJSON,
		}
		if err := s.modelQ.Upsert(ctx, meta); err != nil {
			log.Printf("discovery zai upsert %s: %v", m.ID, err)
			continue
		}
		count++
	}
	return count, nil
}

// --- Together ---

func (s *Service) discoverTogether(ctx context.Context, p model.ProviderConfig) (int, error) {
	return s.discoverOpenAICompat(ctx, p, "together")
}

// --- Helpers ---

func parseTokenPrice(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	// sentinel/absurd prices (openrouter/auto reports -1) overflow the numeric column
	if f < 0 || f > 1 {
		return 0
	}
	return f
}
