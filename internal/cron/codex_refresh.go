package cron

import (
	"context"
	"log"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/handler"
)

type CodexRefresher struct {
	pkQ  *queries.ProviderKeyQueries
	stop chan struct{}
}

func NewCodexRefresher(pkQ *queries.ProviderKeyQueries) *CodexRefresher {
	return &CodexRefresher{
		pkQ:  pkQ,
		stop: make(chan struct{}),
	}
}

func (cr *CodexRefresher) Start() {
	go cr.loop()
	log.Printf("Codex token refresh cron started (expiry check every 5m)")
}

func (cr *CodexRefresher) Stop() {
	close(cr.stop)
}

func (cr *CodexRefresher) loop() {
	cr.run()
	ticker := time.NewTicker(5 * time.Minute)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	keys, err := cr.pkQ.ListUsersWithProvider(ctx, "openai_codex")
	if err != nil {
		log.Printf("codex-refresh: list users: %v", err)
		return
	}
	if len(keys) == 0 {
		return
	}
	refreshed := 0
	for _, k := range keys {
		changed, refreshErr := handler.RefreshStoredOpenAICodexCredentialIfNeeded(ctx, cr.pkQ, k)
		if refreshErr != nil {
			log.Printf("codex-refresh: user %s: %v", shortUserID(k.UserID), refreshErr)
			continue
		}
		if changed {
			refreshed++
			log.Printf("codex-refresh: user %s: rotated OAuth tokens", shortUserID(k.UserID))
		}
	}
	if refreshed > 0 {
		log.Printf("codex-refresh: rotated %d of %d sign-in(s)", refreshed, len(keys))
	}
	// Reload this replica's shared Max-plan cache. Its own expiry check prevents
	// a second rotation if the all-user pass just refreshed the admin row.
	handler.TriggerAdminOpenAIMaxPlanRefresh()
}

func shortUserID(userID string) string {
	if len(userID) <= 8 {
		return userID
	}
	return userID[:8]
}
