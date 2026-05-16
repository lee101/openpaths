package cron

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
)

type CodexRefresher struct {
	pkQ    *queries.ProviderKeyQueries
	client *http.Client
	stop   chan struct{}
}

func NewCodexRefresher(pkQ *queries.ProviderKeyQueries) *CodexRefresher {
	return &CodexRefresher{
		pkQ:    pkQ,
		client: &http.Client{Timeout: 30 * time.Second},
		stop:   make(chan struct{}),
	}
}

func (cr *CodexRefresher) Start() {
	go cr.loop()
	log.Printf("Codex token refresh cron started (every 3h)")
}

func (cr *CodexRefresher) Stop() {
	close(cr.stop)
}

func (cr *CodexRefresher) loop() {
	cr.run()
	ticker := time.NewTicker(3 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cr.run()
		case <-cr.stop:
			return
		}
	}
}

func (cr *CodexRefresher) run() {
	ctx := context.Background()
	keys, err := cr.pkQ.ListUsersWithProvider(ctx, "openai_codex")
	if err != nil {
		log.Printf("codex-refresh: list users: %v", err)
		return
	}
	if len(keys) == 0 {
		return
	}
	log.Printf("codex-refresh: refreshing tokens for %d users", len(keys))
	for _, k := range keys {
		cr.refreshOne(k)
	}
}

func (cr *CodexRefresher) refreshOne(k *queries.UserProviderKey) {
	token := k.APIKey
	if token == "" && k.AuthJSON != "" {
		var auth struct {
			Token        string `json:"token"`
			OpenAIAPIKey string `json:"OPENAI_API_KEY"`
			APIKey       string `json:"api_key"`
		}
		if json.Unmarshal([]byte(k.AuthJSON), &auth) == nil {
			switch {
			case auth.OpenAIAPIKey != "":
				token = auth.OpenAIAPIKey
			case auth.APIKey != "":
				token = auth.APIKey
			case auth.Token != "":
				token = auth.Token
			}
		}
	}
	if token == "" {
		return
	}

	body, _ := json.Marshal(map[string]any{
		"model": "codex-mini-latest",
		"messages": []map[string]string{
			{"role": "user", "content": "no"},
		},
		"max_completion_tokens": 1,
		"reasoning_effort":      "low",
	})

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := cr.client.Do(req)
	if err != nil {
		log.Printf("codex-refresh: user %s: request failed: %v", k.UserID[:8], err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == 200 {
		log.Printf("codex-refresh: user %s: ok", k.UserID[:8])
	} else {
		log.Printf("codex-refresh: user %s: status %d", k.UserID[:8], resp.StatusCode)
	}
}
