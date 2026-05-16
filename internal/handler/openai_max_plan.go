package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/valyala/fasthttp"
)

const openAIMaxPlanProvider = "openai_codex"

type selectedProvider struct {
	provider provider.Provider
	byok     bool
	cred     *openAIMaxPlanCredential
}

type openAIMaxPlanCredential struct {
	ID     string
	Label  string
	APIKey string
}

type credentialCircuit struct {
	failures int
	until    time.Time
}

var openAIMaxPlanCircuits = struct {
	sync.Mutex
	items map[string]credentialCircuit
}{items: make(map[string]credentialCircuit)}

func getProviderAttempts(ctx *fasthttp.RequestCtx, userID, providerName string) []selectedProvider {
	if providerName != "openai" {
		if p, ok := getBYOKProvider(ctx, providerName); ok {
			return []selectedProvider{{provider: p, byok: true}}
		}
		return nil
	}

	keys := getUserProviderKeys(ctx)
	if keys == nil {
		return nil
	}

	var attempts []selectedProvider
	for _, cred := range healthyOpenAIMaxPlanCredentials(userID, keys) {
		attempts = append(attempts, selectedProvider{
			provider: makeUserProvider("openai", cred.APIKey),
			byok:     true,
			cred:     cred,
		})
	}

	if uk, ok := keys["openai"]; ok && strings.TrimSpace(uk.APIKey) != "" {
		attempts = append(attempts, selectedProvider{
			provider: makeUserProvider("openai", strings.TrimSpace(uk.APIKey)),
			byok:     true,
		})
	}

	return attempts
}

func healthyOpenAIMaxPlanCredentials(userID string, keys map[string]*queries.UserProviderKey) []*openAIMaxPlanCredential {
	var creds []*openAIMaxPlanCredential
	for _, providerName := range []string{openAIMaxPlanProvider, "openai"} {
		if uk, ok := keys[providerName]; ok && strings.TrimSpace(uk.AuthJSON) != "" {
			creds = append(creds, parseOpenAIMaxPlanCredentials(userID, providerName, uk.AuthJSON)...)
		}
	}
	if len(creds) == 0 {
		return nil
	}

	now := time.Now()
	filtered := creds[:0]
	for _, cred := range creds {
		if isOpenAIMaxPlanCredentialHealthy(cred.ID, now) {
			filtered = append(filtered, cred)
		}
	}
	shuffleCredentials(filtered)
	return filtered
}

func parseOpenAIMaxPlanCredentials(userID, providerName, raw string) []*openAIMaxPlanCredential {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var root any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil
	}

	var creds []*openAIMaxPlanCredential
	var walk func(any, string)
	walk = func(v any, path string) {
		switch x := v.(type) {
		case map[string]any:
			if key := openAIKeyFromAuthObject(x); key != "" {
				creds = append(creds, newOpenAIMaxPlanCredential(userID, providerName, path, key))
				return
			}
			for _, field := range []string{"credentials", "accounts", "auths", "items", "keys", "OPENAI_API_KEYS"} {
				if child, ok := x[field]; ok {
					walk(child, path+"."+field)
				}
			}
		case []any:
			for i, child := range x {
				walk(child, fmt.Sprintf("%s.%d", path, i))
			}
		case string:
			if strings.HasPrefix(strings.TrimSpace(x), "{") {
				walk(json.RawMessage([]byte(x)), path)
				return
			}
			if strings.HasPrefix(x, "sk-") {
				creds = append(creds, newOpenAIMaxPlanCredential(userID, providerName, path, strings.TrimSpace(x)))
			}
		case json.RawMessage:
			var nested any
			if json.Unmarshal(x, &nested) == nil {
				walk(nested, path)
			}
		}
	}
	walk(root, providerName)

	return dedupeOpenAIMaxPlanCredentials(creds)
}

func openAIKeyFromAuthObject(m map[string]any) string {
	for _, field := range []string{"OPENAI_API_KEY", "api_key", "openai_api_key", "token"} {
		if raw, ok := m[field]; ok {
			if s, ok := raw.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func newOpenAIMaxPlanCredential(userID, providerName, path, apiKey string) *openAIMaxPlanCredential {
	sum := sha256.Sum256([]byte(userID + ":" + providerName + ":" + apiKey))
	id := hex.EncodeToString(sum[:])[:16]
	return &openAIMaxPlanCredential{
		ID:     id,
		Label:  providerName + "/" + id[:8] + " (" + path + ")",
		APIKey: strings.TrimSpace(apiKey),
	}
}

func dedupeOpenAIMaxPlanCredentials(creds []*openAIMaxPlanCredential) []*openAIMaxPlanCredential {
	seen := make(map[string]bool, len(creds))
	out := creds[:0]
	for _, cred := range creds {
		if cred == nil || cred.APIKey == "" || seen[cred.ID] {
			continue
		}
		seen[cred.ID] = true
		out = append(out, cred)
	}
	return out
}

func isOpenAIMaxPlanCredentialHealthy(id string, now time.Time) bool {
	openAIMaxPlanCircuits.Lock()
	defer openAIMaxPlanCircuits.Unlock()
	cb, ok := openAIMaxPlanCircuits.items[id]
	if !ok {
		return true
	}
	if now.After(cb.until) {
		delete(openAIMaxPlanCircuits.items, id)
		return true
	}
	return false
}

func markOpenAIMaxPlanCredentialFailure(id string, err error) {
	if id == "" {
		return
	}
	openAIMaxPlanCircuits.Lock()
	defer openAIMaxPlanCircuits.Unlock()
	cb := openAIMaxPlanCircuits.items[id]
	cb.failures++
	cooldown := time.Duration(cb.failures) * 5 * time.Minute
	if isCredentialAuthError(err) {
		cooldown = time.Duration(cb.failures) * 30 * time.Minute
	}
	if cooldown > 24*time.Hour {
		cooldown = 24 * time.Hour
	}
	cb.until = time.Now().Add(cooldown)
	openAIMaxPlanCircuits.items[id] = cb
}

func markOpenAIMaxPlanCredentialHealthy(id string) {
	if id == "" {
		return
	}
	openAIMaxPlanCircuits.Lock()
	defer openAIMaxPlanCircuits.Unlock()
	delete(openAIMaxPlanCircuits.items, id)
}

func shuffleCredentials(creds []*openAIMaxPlanCredential) {
	for i := len(creds) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue
		}
		j := int(n.Int64())
		creds[i], creds[j] = creds[j], creds[i]
	}
}

func (h *ChatHandler) handleOpenAIMaxPlanCredentialFailure(ctx context.Context, userID string, cred *openAIMaxPlanCredential, err error) {
	if cred == nil {
		return
	}
	markOpenAIMaxPlanCredentialFailure(cred.ID, err)
	if !isCredentialAuthError(err) || h.userQ == nil || h.providerKeyQ == nil {
		return
	}
	shouldSend, alertErr := h.providerKeyQ.ShouldSendCredentialAlert(ctx, userID, openAIMaxPlanProvider, cred.ID)
	if alertErr != nil {
		log.Printf("openai max plan credential alert throttle failed: %v", alertErr)
		return
	}
	if !shouldSend {
		return
	}
	user, userErr := h.userQ.GetByID(ctx, userID)
	if userErr != nil || user == nil || strings.TrimSpace(user.Email) == "" {
		if userErr != nil {
			log.Printf("openai max plan credential alert user lookup failed: %v", userErr)
		}
		return
	}
	go sendOpenAIMaxPlanCredentialEmail(user.Email, cred.Label, err)
}

func sendOpenAIMaxPlanCredentialEmail(toEmail, label string, err error) {
	msg := html.EscapeString(userFacingCredentialFailure(err))
	body := fmt.Sprintf(`<p>One of your OpenAI max plan credentials in OpenPaths is not working.</p>
<p>Credential: <strong>%s</strong></p>
<p>OpenPaths skipped it and tried another available credential or fallback key. Please update your OpenAI Codex <code>auth.json</code> in Account so future requests keep working.</p>
<p>Error: %s</p>`, html.EscapeString(label), msg)
	if sendErr := email.Send(toEmail, "OpenPaths OpenAI max plan credential needs attention", body); sendErr != nil {
		log.Printf("openai max plan credential alert email failed for %s: %v", toEmail, sendErr)
	}
}

func isCredentialAuthError(err error) bool {
	if err == nil {
		return false
	}
	var pe *provider.ProviderError
	if errors.As(err, &pe) {
		if pe.StatusCode == 401 || pe.StatusCode == 403 {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{"invalid api key", "incorrect api key", "unauthorized", "forbidden", "authentication", "auth", "credentials"} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func userFacingCredentialFailure(err error) string {
	if err == nil {
		return "credential request failed"
	}
	var pe *provider.ProviderError
	if errors.As(err, &pe) {
		if pe.StatusCode == 401 || pe.StatusCode == 403 {
			return fmt.Sprintf("OpenAI rejected the credential with HTTP %d.", pe.StatusCode)
		}
		return fmt.Sprintf("OpenAI returned HTTP %d.", pe.StatusCode)
	}
	return "OpenAI rejected the credential."
}
