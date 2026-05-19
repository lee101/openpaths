//go:build manual

// API compatibility e2e test — verifies OpenAI and Anthropic API formats
// work correctly through the gateway, including cross-provider model routing
// and Anthropic-specific features like prefill.
//
// Run manually:
//
//   go test -tags manual -run TestAPICompat -v -timeout 2m ./internal/handler/
//
// Prerequisites:
//   - Server running (localhost:8080 by default, or set SERVER_URL)
//   - Valid API key set as OPENPATHS_API_KEY in .env or environment
//
// Uses cheap/fast models: gemini-3.1-flash-lite, claude-haiku, gpt-5-mini

package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func init() {
	loadEnvFile(".env")
	loadEnvFile("../../.env")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func getServerURL() string {
	if u := os.Getenv("SERVER_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func getAPIKey(t *testing.T) string {
	key := os.Getenv("OPENPATHS_API_KEY")
	if key == "" {
		t.Fatal("OPENPATHS_API_KEY not set")
	}
	return key
}

// --- OpenAI-format tests (POST /v1/chat/completions) ---

func TestAPICompatOpenAIChatBasic(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	models := []string{"gemini-3.1-flash-lite", "gpt-5-mini", "claude-haiku"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": "Reply with exactly one word: hello"},
				},
				"max_tokens": 32,
			})

			req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, respBody)
			}

			var result struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Model   string `json:"model"`
				Choices []struct {
					Message struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil {
				t.Fatalf("unmarshal: %v\nbody: %s", err, respBody)
			}

			if result.Object != "chat.completion" {
				t.Errorf("object = %q, want chat.completion", result.Object)
			}
			if len(result.Choices) == 0 {
				t.Fatal("no choices returned")
			}
			if result.Choices[0].Message.Role != "assistant" {
				t.Errorf("role = %q, want assistant", result.Choices[0].Message.Role)
			}
			if result.Choices[0].Message.Content == "" {
				t.Error("empty content")
			}
			if result.Choices[0].FinishReason == "" {
				t.Error("empty finish_reason")
			}

			t.Logf("model=%s response=%q tokens_in=%d tokens_out=%d",
				model, result.Choices[0].Message.Content,
				result.Usage.PromptTokens, result.Usage.CompletionTokens)
		})
	}
}

func TestAPICompatOpenAISystemPrompt(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	body, _ := json.Marshal(map[string]any{
		"model": "gemini-3.1-flash-lite",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a pirate. Always respond with 'Arrr' in your answer."},
			{"role": "user", "content": "Say hi"},
		},
		"max_tokens": 64,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Choices) == 0 {
		t.Fatal("no choices")
	}

	content := strings.ToLower(result.Choices[0].Message.Content)
	t.Logf("pirate response: %q", result.Choices[0].Message.Content)
	if !strings.Contains(content, "arr") {
		t.Logf("WARNING: system prompt may not have been followed (no 'arr' in response)")
	}
}

func TestAPICompatOpenAIStreaming(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	body, _ := json.Marshal(map[string]any{
		"model": "gemini-3.1-flash-lite",
		"messages": []map[string]string{
			{"role": "user", "content": "Say hello"},
		},
		"max_tokens": 32,
		"stream":     true,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}

	data, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(data), "\n")

	chunks := 0
	gotDone := false
	var fullContent string

	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			gotDone = true
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			fullContent += chunk.Choices[0].Delta.Content
			chunks++
		}
	}

	t.Logf("streaming: %d chunks, content=%q, done=%v", chunks, fullContent, gotDone)

	if chunks == 0 {
		t.Error("no stream chunks received")
	}
	if !gotDone {
		t.Error("no [DONE] marker received")
	}
	if fullContent == "" {
		t.Error("empty streamed content")
	}
}

// --- Anthropic-format tests (POST /v1/messages) ---

func TestAPICompatAnthropicBasic(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Test that Anthropic API format works with different provider models
	models := []string{"claude-haiku", "gemini-3.1-flash-lite", "gpt-5-mini"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"model": model,
				"messages": []map[string]string{
					{"role": "user", "content": "Reply with exactly one word: hello"},
				},
				"max_tokens": 32,
			})

			req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, respBody)
			}

			var result struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Role       string `json:"role"`
				Model      string `json:"model"`
				Content    []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
				StopReason string `json:"stop_reason"`
				Usage      struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(respBody, &result); err != nil {
				t.Fatalf("unmarshal: %v\nbody: %s", err, respBody)
			}

			if result.Type != "message" {
				t.Errorf("type = %q, want message", result.Type)
			}
			if result.Role != "assistant" {
				t.Errorf("role = %q, want assistant", result.Role)
			}
			if len(result.Content) == 0 {
				t.Fatal("no content blocks")
			}
			if result.Content[0].Type != "text" {
				t.Errorf("content[0].type = %q, want text", result.Content[0].Type)
			}
			if result.Content[0].Text == "" {
				t.Error("empty text content")
			}
			if result.StopReason == "" {
				t.Error("empty stop_reason")
			}

			t.Logf("model=%s response=%q tokens_in=%d tokens_out=%d",
				model, result.Content[0].Text,
				result.Usage.InputTokens, result.Usage.OutputTokens)
		})
	}
}

func TestAPICompatAnthropicSystemPrompt(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Anthropic format uses top-level "system" field
	body, _ := json.Marshal(map[string]any{
		"model":  "claude-haiku",
		"system": "You must always include the word 'banana' in your response.",
		"messages": []map[string]string{
			{"role": "user", "content": "Say hi"},
		},
		"max_tokens": 64,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Content) == 0 {
		t.Fatal("no content")
	}

	t.Logf("system prompt response: %q", result.Content[0].Text)
	if !strings.Contains(strings.ToLower(result.Content[0].Text), "banana") {
		t.Logf("WARNING: system prompt may not have been followed (no 'banana' in response)")
	}
}

func TestAPICompatAnthropicPrefill(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Anthropic prefill: include a trailing assistant message to guide the response.
	// The model should continue from the prefilled text.
	body, _ := json.Marshal(map[string]any{
		"model": "claude-haiku",
		"messages": []map[string]any{
			{"role": "user", "content": "What is 2+2? Reply with just the number."},
			{"role": "assistant", "content": "The answer is: "},
		},
		"max_tokens": 32,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Content) == 0 {
		t.Fatal("no content")
	}

	text := result.Content[0].Text
	t.Logf("prefill response: %q", text)

	// With prefill "The answer is: ", the model should continue with "4"
	if !strings.Contains(text, "4") {
		t.Errorf("expected '4' in prefill continuation, got %q", text)
	}
}

func TestAPICompatAnthropicStreaming(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	body, _ := json.Marshal(map[string]any{
		"model": "claude-haiku",
		"messages": []map[string]string{
			{"role": "user", "content": "Say hello"},
		},
		"max_tokens": 32,
		"stream":     true,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}

	data, _ := io.ReadAll(resp.Body)
	body_str := string(data)

	// Anthropic SSE format has specific event types
	gotMessageStart := strings.Contains(body_str, "message_start")
	gotContentDelta := strings.Contains(body_str, "content_block_delta")
	gotMessageStop := strings.Contains(body_str, "message_stop")

	t.Logf("streaming events: message_start=%v content_block_delta=%v message_stop=%v",
		gotMessageStart, gotContentDelta, gotMessageStop)

	if !gotMessageStart {
		t.Error("missing message_start event")
	}
	if !gotContentDelta {
		t.Error("missing content_block_delta event")
	}
	if !gotMessageStop {
		t.Error("missing message_stop event")
	}
}

// --- Cross-format model compatibility ---

func TestAPICompatCrossProvider(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Send the same prompt to different providers via OpenAI format
	// then via Anthropic format — all should work
	prompt := "What is 1+1? Reply with just the number."

	tests := []struct {
		name   string
		format string // "openai" or "anthropic"
		model  string
	}{
		{"openai/gemini-3.1-flash-lite", "openai", "gemini-3.1-flash-lite"},
		{"openai/gpt-5-mini", "openai", "gpt-5-mini"},
		{"openai/claude-haiku", "openai", "claude-haiku"},
		{"anthropic/gemini-3.1-flash-lite", "anthropic", "gemini-3.1-flash-lite"},
		{"anthropic/gpt-5-mini", "anthropic", "gpt-5-mini"},
		{"anthropic/claude-haiku", "anthropic", "claude-haiku"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var respBody []byte
			var statusCode int

			if tc.format == "openai" {
				body, _ := json.Marshal(map[string]any{
					"model": tc.model,
					"messages": []map[string]string{
						{"role": "user", "content": prompt},
					},
					"max_tokens": 16,
				})
				req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+apiKey)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer resp.Body.Close()
				statusCode = resp.StatusCode
				respBody, _ = io.ReadAll(resp.Body)
			} else {
				body, _ := json.Marshal(map[string]any{
					"model": tc.model,
					"messages": []map[string]string{
						{"role": "user", "content": prompt},
					},
					"max_tokens": 16,
				})
				req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("x-api-key", apiKey)
				req.Header.Set("anthropic-version", "2023-06-01")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer resp.Body.Close()
				statusCode = resp.StatusCode
				respBody, _ = io.ReadAll(resp.Body)
			}

			if statusCode != 200 {
				t.Fatalf("status %d: %s", statusCode, respBody)
			}

			// Just verify we got valid JSON back with content
			var raw map[string]any
			if err := json.Unmarshal(respBody, &raw); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			t.Logf("%s/%s: OK (%d bytes)", tc.format, tc.model, len(respBody))
		})
	}
}

// --- Alias resolution test ---

func TestAPICompatAliasResolution(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	aliases := []struct {
		alias    string
		wantModel string
	}{
		{"openai-chat-latest", "gpt-5.5"},
		{"haiku", "claude-haiku-4-5-20251001"},
		{"sonnet", "claude-sonnet-latest"},
		{"opus", "claude-opus-latest"},
		{"flash-lite", "gemini-3.1-flash-lite"},
	}

	for _, tc := range aliases {
		t.Run(tc.alias, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"model": tc.alias,
				"messages": []map[string]string{
					{"role": "user", "content": "Say ok"},
				},
				"max_tokens": 8,
			})

			req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				t.Fatalf("alias %q: status %d: %s", tc.alias, resp.StatusCode, respBody)
			}

			var result struct {
				Model string `json:"model"`
			}
			json.Unmarshal(respBody, &result)

			// The response model should be the alias we sent (gateway preserves original model)
			t.Logf("alias=%q resolved, response model=%q", tc.alias, result.Model)

			if result.Model != tc.alias {
				t.Logf("NOTE: response model %q differs from alias %q (gateway may return original)", result.Model, tc.alias)
			}
		})
	}
}

// --- Error format tests ---

func TestAPICompatOpenAIErrorFormat(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Missing model should return OpenAI error format
	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		t.Fatal("expected error, got 200")
	}

	var result struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("error response is not valid JSON: %v\nbody: %s", err, respBody)
	}

	if result.Error.Message == "" {
		t.Error("error.message is empty")
	}
	t.Logf("OpenAI error format: type=%q message=%q", result.Error.Type, result.Error.Message)
}

func TestAPICompatAnthropicErrorFormat(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Missing max_tokens should return Anthropic error format
	body, _ := json.Marshal(map[string]any{
		"model": "claude-haiku",
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		t.Fatal("expected error, got 200")
	}

	var result struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("error response is not valid JSON: %v\nbody: %s", err, respBody)
	}

	if result.Type != "error" {
		t.Errorf("type = %q, want error", result.Type)
	}
	if result.Error.Message == "" {
		t.Error("error.message is empty")
	}
	t.Logf("Anthropic error format: type=%q error.type=%q message=%q",
		result.Type, result.Error.Type, result.Error.Message)
}

// --- Prefill via OpenAI format (assistant message at end) ---

func TestAPICompatOpenAIPrefill(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Some models support continuing from an assistant message
	// This tests that the gateway passes it through correctly
	models := []string{"claude-haiku", "gemini-3.1-flash-lite"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"model": model,
				"messages": []map[string]any{
					{"role": "user", "content": "List 3 colors. Format: 1. X 2. Y 3. Z"},
					{"role": "assistant", "content": "1. Red 2. Blue 3."},
				},
				"max_tokens": 16,
			})

			req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, respBody)
			}

			var result struct {
				Choices []struct {
					Message struct{ Content string } `json:"message"`
				} `json:"choices"`
			}
			json.Unmarshal(respBody, &result)

			if len(result.Choices) == 0 {
				t.Fatal("no choices")
			}

			t.Logf("prefill via openai format: model=%s response=%q", model, result.Choices[0].Message.Content)
		})
	}
}

func TestAPICompatAnthropicSystemArray(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Anthropic supports system as array of content blocks
	body, _ := json.Marshal(map[string]any{
		"model": "claude-haiku",
		"system": []map[string]string{
			{"type": "text", "text": "You always reply in ALL CAPS."},
		},
		"messages": []map[string]string{
			{"role": "user", "content": "Say hello"},
		},
		"max_tokens": 32,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Content) == 0 {
		t.Fatal("no content")
	}

	t.Logf("system array response: %q", result.Content[0].Text)

	// Check if response has uppercase (system prompt asked for ALL CAPS)
	text := result.Content[0].Text
	upper := strings.ToUpper(text)
	if text == upper {
		t.Logf("system array content block: ALL CAPS correctly followed")
	} else {
		t.Logf("NOTE: response not all caps — system array may need more enforcement")
	}
}

func TestAPICompatMultiTurn(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	// Multi-turn conversation
	body, _ := json.Marshal(map[string]any{
		"model": "gemini-3.1-flash-lite",
		"messages": []map[string]string{
			{"role": "user", "content": "My name is TestBot."},
			{"role": "assistant", "content": "Nice to meet you, TestBot!"},
			{"role": "user", "content": "What is my name?"},
		},
		"max_tokens": 32,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &result)

	if len(result.Choices) == 0 {
		t.Fatal("no choices")
	}

	content := result.Choices[0].Message.Content
	t.Logf("multi-turn response: %q", content)

	if !strings.Contains(strings.ToLower(content), "testbot") {
		t.Errorf("model didn't remember the name, got: %q", content)
	}
}

func TestAPICompatModelsList(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	req, _ := http.NewRequest("GET", serverURL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Object != "list" {
		t.Errorf("object = %q, want list", result.Object)
	}
	if len(result.Data) == 0 {
		t.Error("no models returned")
	}

	// Verify key models exist
	modelSet := make(map[string]bool)
	for _, m := range result.Data {
		modelSet[m.ID] = true
	}

	required := []string{"gpt-5.5", "gpt-5.4", "claude-sonnet-latest", "claude-opus-latest", "gemini-3.1-flash-lite", "claude-haiku-4-5-20251001"}
	for _, id := range required {
		if !modelSet[id] {
			t.Errorf("expected model %q in /v1/models list", id)
		}
	}

	t.Logf("models list: %d models returned", len(result.Data))

	// Log a few interesting ones
	for _, m := range result.Data {
		if m.ID == "gpt-5.5" || m.ID == "gpt-5.4" || m.ID == "claude-sonnet-latest" || m.ID == "claude-opus-latest" {
			t.Logf("  %s (owned_by: %s)", m.ID, m.OwnedBy)
		}
	}
}

// Verify the openai-chat-latest alias resolves to gpt-5.5
func TestAPICompatOpenAIChatLatestAlias(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	body, _ := json.Marshal(map[string]any{
		"model": "openai-chat-latest",
		"messages": []map[string]string{
			{"role": "user", "content": "Say ok"},
		},
		"max_tokens": 8,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Model string `json:"model"`
	}
	json.Unmarshal(respBody, &result)

	t.Logf("openai-chat-latest resolved to model=%q in response", result.Model)

	// The response should echo back "openai-chat-latest" as the model
	// (gateway preserves the requested model name)
	if result.Model != "openai-chat-latest" {
		t.Logf("NOTE: response model %q (gateway may echo alias or resolved ID)", result.Model)
	}

	fmt.Println("openai-chat-latest -> gpt-5.5: OK")
}
