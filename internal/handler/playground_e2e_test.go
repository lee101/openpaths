//go:build manual

// Playground e2e test — verifies the streaming chat flow that the
// /playground UI relies on actually works end-to-end.
//
// Run manually:
//
//   go test -tags manual -run TestPlayground -v -timeout 2m ./internal/handler/
//
// Prerequisites:
//   - Server running (localhost:8080 by default, or set SERVER_URL)
//   - Valid API key set as OPENPATHS_API_KEY in .env or environment

package handler_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestPlaygroundStreamingChat simulates exactly what the Playground UI does:
// sends a streaming chat completion to a tiny model and verifies the full
// SSE response is parseable and contains content.
func TestPlaygroundStreamingChat(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	body, _ := json.Marshal(map[string]any{
		"model": "gemini-flash-lite",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "say no"},
		},
		"stream": true,
	})

	req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	start := time.Now()
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

	// Parse SSE exactly like the Playground frontend does
	scanner := bufio.NewScanner(resp.Body)
	var (
		fullContent string
		chunks      int
		gotDone     bool
		ttft        time.Duration
	)

	for scanner.Scan() {
		line := scanner.Text()
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
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if chunks == 0 {
				ttft = time.Since(start)
			}
			fullContent += chunk.Choices[0].Delta.Content
			chunks++
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("reading stream: %v", err)
	}

	t.Logf("TTFT: %v, chunks: %d, content: %q, done: %v", ttft, chunks, fullContent, gotDone)

	if chunks == 0 {
		t.Error("no stream chunks received")
	}
	if !gotDone {
		t.Error("no [DONE] marker received")
	}
	if fullContent == "" {
		t.Error("empty streamed content")
	}
	// The model should say "no" or something containing "no" given the prompt
	lower := strings.ToLower(fullContent)
	if !strings.Contains(lower, "no") {
		t.Logf("WARNING: expected 'no' in response to 'say no', got: %q", fullContent)
	}
}

// TestPlaygroundMultiModel simulates the multi-pane comparison feature:
// sends the same message to multiple models concurrently.
func TestPlaygroundMultiModel(t *testing.T) {
	apiKey := getAPIKey(t)
	serverURL := getServerURL()

	models := []string{"gemini-flash-lite", "mistral-small-latest"}

	type result struct {
		model   string
		content string
		err     error
	}

	results := make(chan result, len(models))

	for _, model := range models {
		go func(m string) {
			body, _ := json.Marshal(map[string]any{
				"model": m,
				"messages": []map[string]string{
					{"role": "user", "content": "Reply with exactly: hello"},
				},
				"max_tokens": 16,
			})

			req, _ := http.NewRequest("POST", serverURL+"/v1/chat/completions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results <- result{model: m, err: err}
				return
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				results <- result{model: m, err: &httpError{status: resp.StatusCode, body: string(respBody)}}
				return
			}

			var parsed struct {
				Choices []struct {
					Message struct{ Content string } `json:"message"`
				} `json:"choices"`
			}
			json.Unmarshal(respBody, &parsed)

			content := ""
			if len(parsed.Choices) > 0 {
				content = parsed.Choices[0].Message.Content
			}
			results <- result{model: m, content: content}
		}(model)
	}

	for range models {
		r := <-results
		if r.err != nil {
			t.Errorf("model %s: %v", r.model, r.err)
			continue
		}
		if r.content == "" {
			t.Errorf("model %s: empty response", r.model)
			continue
		}
		t.Logf("model=%s response=%q", r.model, r.content)
	}
}

type httpError struct {
	status int
	body   string
}

func (e *httpError) Error() string {
	return "HTTP " + http.StatusText(e.status) + ": " + e.body[:min(len(e.body), 200)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
