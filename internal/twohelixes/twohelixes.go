// Package twohelixes sends server-side funnel events (purchases) to twoHelixes analytics.
package twohelixes

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	collectURL      = "https://twohelixes.com/v1/collect"
	defaultWriteKey = "thw_ybeyFj_kDt_tt07RSO3CyLexIPoCb9xk"
)

var client = &http.Client{Timeout: 5 * time.Second}

func writeKey() string {
	if k, ok := os.LookupEnv("TWOHELIXES_WRITE_KEY"); ok {
		return k
	}
	return defaultWriteKey
}

func event(userID, eventID, name string, props map[string]any) map[string]any {
	return map[string]any{
		"event":     name,
		"client_id": userID,
		"user_id":   userID,
		"event_id":  eventID,
		"ts":        time.Now().UnixMilli(),
		"props":     props,
	}
}

// Track posts one event in the background; eventID makes retries idempotent.
func Track(userID, eventID, name string, props map[string]any) {
	key := writeKey()
	if key == "" || userID == "" || testing.Testing() {
		return
	}
	body, err := json.Marshal(map[string]any{"site_id": key, "events": []map[string]any{event(userID, eventID, name, props)}})
	if err != nil {
		return
	}
	go func() {
		resp, err := client.Post(collectURL, "text/plain", bytes.NewReader(body))
		if err != nil {
			log.Printf("twohelixes: %s %s: %v", name, eventID, err)
			return
		}
		resp.Body.Close()
	}()
}

// Purchase records confirmed revenue in major units.
func Purchase(userID, transactionID, plan, surface, currency string, revenue float64) {
	if currency == "" {
		currency = "USD"
	}
	props := map[string]any{"plan": plan, "revenue": revenue, "currency": currency, "transaction_id": transactionID}
	if surface != "" {
		props["surface"] = surface
	}
	Track(userID, transactionID, "purchase", props)
}
