package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/model"
)

type DiscoveredModel struct {
	Provider      string   `json:"provider"`
	ModelID       string   `json:"model_id"`
	DisplayName   string   `json:"display_name,omitempty"`
	ModelType     string   `json:"model_type"`
	ContextLength int      `json:"context_length,omitempty"`
	InputPrice    float64  `json:"input_price_per_1m,omitempty"`
	OutputPrice   float64  `json:"output_price_per_1m,omitempty"`
	Features      []string `json:"features,omitempty"`
	InputMod      []string `json:"input_modalities,omitempty"`
	OutputMod     []string `json:"output_modalities,omitempty"`
	Endpoint      string   `json:"discovered_via"`
}

type ProbeResult struct {
	Provider    string
	Endpoint    string
	StatusCode  int
	Models      []DiscoveredModel
	Error       string
	Compatible  bool
	Format      string // "openai", "openrouter", "anthropic", "google", "unknown"
	RawResponse json.RawMessage
}

func main() {
	_ = godotenv.Load()

	configPath := flag.String("config", "config.yaml", "Path to config file")
	providerFilter := flag.String("provider", "", "Only probe this provider")
	verbose := flag.Bool("v", false, "Verbose output (show raw responses)")
	jsonOut := flag.Bool("json", false, "Output as JSON")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	ctx := context.Background()

	var results []ProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range cfg.Providers {
		if !p.Enabled || p.APIKey == "" {
			continue
		}
		if *providerFilter != "" && p.Name != *providerFilter {
			continue
		}

		wg.Add(1)
		go func(p model.ProviderConfig) {
			defer wg.Done()
			probeResults := probeProvider(ctx, client, p, *verbose)
			mu.Lock()
			results = append(results, probeResults...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Provider < results[j].Provider
	})

	if *jsonOut {
		outputJSON(results)
	} else {
		outputTable(results, *verbose, cfg.Models)
	}
}

func probeProvider(ctx context.Context, client *http.Client, p model.ProviderConfig, verbose bool) []ProbeResult {
	var results []ProbeResult

	// Determine which endpoints to probe based on provider
	endpoints := getProbeEndpoints(p)

	for _, ep := range endpoints {
		result := probeEndpoint(ctx, client, p, ep, verbose)
		results = append(results, result)
	}

	return results
}

type endpointSpec struct {
	URL     string
	Format  string
	Headers map[string]string
}

func getProbeEndpoints(p model.ProviderConfig) []endpointSpec {
	baseURL := strings.TrimRight(p.BaseURL, "/")
	bearerAuth := map[string]string{"Authorization": "Bearer " + p.APIKey}

	var endpoints []endpointSpec

	switch p.Name {
	case "anthropic":
		endpoints = append(endpoints, endpointSpec{
			URL:    baseURL + "/v1/models",
			Format: "anthropic",
			Headers: map[string]string{
				"x-api-key":         p.APIKey,
				"anthropic-version": "2023-06-01",
			},
		})
	case "google":
		endpoints = append(endpoints, endpointSpec{
			URL:    baseURL + "/v1beta/models?key=" + p.APIKey,
			Format: "google",
		})
	default:
		// Try OpenAI-compatible /v1/models
		endpoints = append(endpoints, endpointSpec{
			URL:     baseURL + "/v1/models",
			Format:  "openai",
			Headers: bearerAuth,
		})
		// Also try /models (some providers use this)
		if p.Name != "openai" {
			endpoints = append(endpoints, endpointSpec{
				URL:     baseURL + "/models",
				Format:  "openai",
				Headers: bearerAuth,
			})
		}
		// For OpenRouter specifically, also try their extended endpoint
		if p.Name == "openrouter" {
			endpoints = append(endpoints, endpointSpec{
				URL:     baseURL + "/v1/models",
				Format:  "openrouter",
				Headers: bearerAuth,
			})
		}
	}

	return endpoints
}

func probeEndpoint(ctx context.Context, client *http.Client, p model.ProviderConfig, ep endpointSpec, verbose bool) ProbeResult {
	result := ProbeResult{
		Provider: p.Name,
		Endpoint: ep.URL,
		Format:   ep.Format,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ep.URL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		return result
	}

	if verbose {
		result.RawResponse = json.RawMessage(body)
	}

	// Try to parse the response in various formats
	models := parseModels(p.Name, ep.Format, body)
	if len(models) > 0 {
		result.Compatible = true
		result.Models = models
	} else {
		result.Error = "response parsed but no models found"
	}

	return result
}

func parseModels(provName, format string, body []byte) []DiscoveredModel {
	switch format {
	case "openrouter":
		return parseOpenRouterModels(provName, body)
	case "anthropic":
		return parseAnthropicModels(body)
	case "google":
		return parseGoogleModels(body)
	default:
		return parseOpenAIModels(provName, body)
	}
}

func parseOpenAIModels(provName string, body []byte) []DiscoveredModel {
	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []DiscoveredModel
	for _, m := range result.Data {
		modType := classifyModelType(m.ID)
		models = append(models, DiscoveredModel{
			Provider:    provName,
			ModelID:     m.ID,
			ModelType:   modType,
			InputMod:    []string{"text"},
			OutputMod:   []string{"text"},
			Endpoint:    "openai-compat /v1/models",
		})
	}
	return models
}

func parseOpenRouterModels(provName string, body []byte) []DiscoveredModel {
	var result struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
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
			TopProvider struct {
				MaxCompletionTokens int  `json:"max_completion_tokens"`
				IsModerated         bool `json:"is_moderated"`
			} `json:"top_provider"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []DiscoveredModel
	for _, m := range result.Data {
		modType := "chat"
		inputMod := []string{"text"}
		outputMod := []string{"text"}
		modality := strings.ToLower(m.Architecture.Modality)

		if strings.Contains(modality, "image") {
			if strings.Contains(modality, "text") {
				inputMod = append(inputMod, "image")
			} else {
				modType = "image"
				outputMod = []string{"image"}
			}
		}

		promptPrice, _ := strconv.ParseFloat(m.Pricing.Prompt, 64)
		completionPrice, _ := strconv.ParseFloat(m.Pricing.Completion, 64)

		models = append(models, DiscoveredModel{
			Provider:      provName,
			ModelID:       m.ID,
			DisplayName:   m.Name,
			ModelType:     modType,
			ContextLength: m.ContextLength,
			InputPrice:    promptPrice * 1_000_000,
			OutputPrice:   completionPrice * 1_000_000,
			InputMod:      inputMod,
			OutputMod:     outputMod,
			Endpoint:      "openrouter /v1/models",
		})
	}
	return models
}

func parseAnthropicModels(body []byte) []DiscoveredModel {
	var result struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []DiscoveredModel
	for _, m := range result.Data {
		models = append(models, DiscoveredModel{
			Provider:      "anthropic",
			ModelID:       m.ID,
			DisplayName:   m.DisplayName,
			ModelType:     "chat",
			ContextLength: 200000,
			InputMod:      []string{"text", "image"},
			OutputMod:     []string{"text"},
			Features:      []string{"chat", "tools", "streaming", "vision"},
			Endpoint:      "anthropic /v1/models",
		})
	}
	return models
}

func parseGoogleModels(body []byte) []DiscoveredModel {
	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			Description                string   `json:"description"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			OutputTokenLimit           int      `json:"outputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []DiscoveredModel
	for _, m := range result.Models {
		modelID := m.Name
		if strings.HasPrefix(modelID, "models/") {
			modelID = strings.TrimPrefix(modelID, "models/")
		}

		modType := "chat"
		features := []string{}
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

		inputMod := []string{"text"}
		if strings.Contains(modelID, "gemini") && !strings.Contains(modelID, "embed") {
			inputMod = append(inputMod, "image")
			features = append(features, "vision")
		}

		models = append(models, DiscoveredModel{
			Provider:      "google",
			ModelID:       modelID,
			DisplayName:   m.DisplayName,
			ModelType:     modType,
			ContextLength: m.InputTokenLimit,
			Features:      features,
			InputMod:      inputMod,
			OutputMod:     []string{"text"},
			Endpoint:      "google /v1beta/models",
		})
	}
	return models
}

func classifyModelType(id string) string {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "embed"):
		return "embedding"
	case strings.Contains(lower, "tts") || strings.Contains(lower, "whisper") || strings.Contains(lower, "speech"):
		return "audio"
	case strings.Contains(lower, "dall-e") || strings.Contains(lower, "image") || strings.Contains(lower, "flux"):
		return "image"
	case strings.Contains(lower, "video") || strings.Contains(lower, "hailuo"):
		return "video"
	case strings.Contains(lower, "music"):
		return "music"
	case strings.Contains(lower, "moderation"):
		return "moderation"
	default:
		return "chat"
	}
}

func outputJSON(results []ProbeResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(results)
}

func outputTable(results []ProbeResult, verbose bool, configModels []model.ModelConfig) {
	// Build a set of already-configured model IDs
	configured := make(map[string]bool)
	configuredByProvider := make(map[string]map[string]bool)
	for _, m := range configModels {
		configured[m.ID] = true
		configured[m.ProviderModelID] = true
		key := m.Provider
		if configuredByProvider[key] == nil {
			configuredByProvider[key] = make(map[string]bool)
		}
		configuredByProvider[key][m.ProviderModelID] = true
	}

	fmt.Println("=" + strings.Repeat("=", 99))
	fmt.Printf("  PROVIDER DISCOVERY RESULTS\n")
	fmt.Println("=" + strings.Repeat("=", 99))
	fmt.Println()

	totalNew := 0

	for _, r := range results {
		icon := "x"
		if r.Compatible {
			icon := "OK"
			_ = icon
		}

		if r.Error != "" && !r.Compatible {
			fmt.Printf("  [FAIL] %s @ %s\n", r.Provider, r.Endpoint)
			fmt.Printf("         %s\n\n", r.Error)
			continue
		}

		if !r.Compatible {
			fmt.Printf("  [----] %s @ %s - no models found\n\n", r.Provider, r.Endpoint)
			continue
		}

		_ = icon

		// Count new vs existing
		newModels := 0
		existingModels := 0
		provModels := configuredByProvider[r.Provider]

		for _, m := range r.Models {
			if provModels != nil && provModels[m.ModelID] {
				existingModels++
			} else if configured[m.ModelID] {
				existingModels++
			} else {
				newModels++
			}
		}

		fmt.Printf("  [ OK ] %s @ %s\n", r.Provider, r.Endpoint)
		fmt.Printf("         Format: %s | Total: %d | Configured: %d | NEW: %d\n",
			r.Format, len(r.Models), existingModels, newModels)

		if newModels > 0 {
			fmt.Printf("\n         --- NEW/UNCONFIGURED MODELS ---\n")
			for _, m := range r.Models {
				isNew := true
				if provModels != nil && provModels[m.ModelID] {
					isNew = false
				} else if configured[m.ModelID] {
					isNew = false
				}
				if !isNew {
					continue
				}

				name := m.ModelID
				if m.DisplayName != "" {
					name = m.DisplayName + " (" + m.ModelID + ")"
				}
				extra := ""
				if m.ContextLength > 0 {
					extra += fmt.Sprintf(" ctx=%dk", m.ContextLength/1000)
				}
				if m.InputPrice > 0 {
					extra += fmt.Sprintf(" $%.2f/$%.2f per 1M", m.InputPrice, m.OutputPrice)
				}

				fmt.Printf("         [NEW] %-8s %s%s\n", m.ModelType, name, extra)
				totalNew++
			}
		}

		if verbose && len(r.Models) > 0 {
			fmt.Printf("\n         --- ALL MODELS ---\n")
			for _, m := range r.Models {
				status := "    "
				if provModels != nil && provModels[m.ModelID] {
					status = "HAVE"
				} else if configured[m.ModelID] {
					status = "HAVE"
				}
				fmt.Printf("         [%s] %-8s %s\n", status, m.ModelType, m.ModelID)
			}
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("  SUMMARY: %d new models found across all providers\n", totalNew)
	fmt.Println(strings.Repeat("-", 100))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
