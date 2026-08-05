package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
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
	provider     provider.Provider
	byok         bool
	cred         *openAIMaxPlanCredential
	afterRefresh bool
}

type openAIMaxPlanCredential struct {
	ID           string
	Label        string
	APIKey       string
	ProviderName string
	AuthJSON     string
	RefreshToken string
	// Platform marks the shared admin max-plan credential used for all OpenAI
	// traffic. OwnerUserID is the user the credential is stored under (the
	// admin), used to persist refreshes to the right row rather than the
	// requesting user.
	Platform    bool
	OwnerUserID string
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

	var attempts []selectedProvider
	if keys := getUserProviderKeys(ctx); keys != nil {
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
	}

	// Shared admin max-plan credential: used for everyone, after a request
	// user's own OpenAI keys and before the platform env-key fallback that
	// chat.go appends. byok is false so the requesting user is still billed.
	attempts = append(attempts, adminOpenAIMaxPlanAttempts()...)

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
				cred := newOpenAIMaxPlanCredential(userID, providerName, path, key)
				cred.Label = safeOpenAIMaxPlanCredentialLabel(providerName, path, x)
				cred.AuthJSON = raw
				cred.RefreshToken = refreshTokenFromAuthObject(x)
				creds = append(creds, cred)
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

func refreshTokenFromAuthObject(m map[string]any) string {
	if token := stringField(m, "refresh_token"); token != "" {
		return token
	}
	if tokens, ok := m["tokens"].(map[string]any); ok {
		return stringField(tokens, "refresh_token")
	}
	return ""
}

func newOpenAIMaxPlanCredential(userID, providerName, path, apiKey string) *openAIMaxPlanCredential {
	sum := sha256.Sum256([]byte(userID + ":" + providerName + ":" + apiKey))
	id := hex.EncodeToString(sum[:])[:16]
	return &openAIMaxPlanCredential{
		ID:           id,
		Label:        safeOpenAIMaxPlanCredentialLabel(providerName, path, nil),
		APIKey:       strings.TrimSpace(apiKey),
		ProviderName: providerName,
	}
}

func safeOpenAIMaxPlanCredentialLabel(providerName, path string, m map[string]any) string {
	if m != nil {
		if email := stringField(m, "email"); email != "" {
			return "OpenAI Codex sign-in for " + email
		}
		if tokens, ok := m["tokens"].(map[string]any); ok {
			if idToken, ok := tokens["id_token"].(map[string]any); ok {
				if email := stringField(idToken, "email"); email != "" {
					return "OpenAI Codex sign-in for " + email
				}
			}
		}
	}
	if providerName == openAIMaxPlanProvider {
		return "OpenAI Codex sign-in"
	}
	return "OpenAI credential at " + path
}

func stringField(m map[string]any, field string) string {
	if raw, ok := m[field]; ok {
		if s, ok := raw.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
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

func (h *ChatHandler) handleOpenAIMaxPlanCredentialFailure(
	ctx context.Context,
	userID string,
	cred *openAIMaxPlanCredential,
	err error,
	allowRefresh bool,
) *openAIMaxPlanCredential {
	if cred == nil {
		return nil
	}
	if cred.Platform {
		return h.handleAdminOpenAIMaxPlanFailure(ctx, cred, err, allowRefresh)
	}
	if !isCredentialAuthError(err) || h.userQ == nil || h.providerKeyQ == nil {
		markOpenAIMaxPlanCredentialFailure(cred.ID, err)
		return nil
	}
	if allowRefresh {
		if refreshed, refreshErr := h.tryRefreshOpenAIMaxPlanCredential(ctx, userID, cred); refreshed {
			markOpenAIMaxPlanCredentialHealthy(cred.ID)
			return cred
		} else if refreshErr != nil {
			log.Printf("openai max plan credential refresh failed for %s: %v", cred.Label, refreshErr)
		}
	}
	markOpenAIMaxPlanCredentialFailure(cred.ID, err)
	shouldSend, alertErr := h.providerKeyQ.ShouldSendCredentialAlert(ctx, userID, openAIMaxPlanProvider, cred.ID)
	if alertErr != nil {
		log.Printf("openai max plan credential alert throttle failed: %v", alertErr)
		return nil
	}
	if !shouldSend {
		return nil
	}
	user, userErr := h.userQ.GetByID(ctx, userID)
	if userErr != nil || user == nil || strings.TrimSpace(user.Email) == "" {
		if userErr != nil {
			log.Printf("openai max plan credential alert user lookup failed: %v", userErr)
		}
		return nil
	}
	go sendOpenAIMaxPlanCredentialEmail(user.Email, cred.Label, err)
	return nil
}

func (h *ChatHandler) tryRefreshOpenAIMaxPlanCredential(ctx context.Context, userID string, cred *openAIMaxPlanCredential) (bool, error) {
	if cred.RefreshToken == "" || cred.AuthJSON == "" || cred.ProviderName == "" {
		return false, nil
	}
	refreshed, _, err := refreshStoredOpenAIMaxPlanAuthJSON(
		ctx, h.providerKeyQ, userID, cred.ProviderName, cred.AuthJSON, true,
	)
	if err != nil {
		return false, err
	}
	if refreshed.APIKey == "" || refreshed.AuthJSON == "" {
		return false, nil
	}
	cred.APIKey = refreshed.APIKey
	cred.ID = newOpenAIMaxPlanCredential(userID, cred.ProviderName, "", refreshed.APIKey).ID
	cred.AuthJSON = refreshed.AuthJSON
	cred.RefreshToken = refreshed.RefreshToken
	return true, nil
}

func sendOpenAIMaxPlanCredentialEmail(toEmail, label string, err error) {
	msg := html.EscapeString(userFacingCredentialFailure(err))
	body := fmt.Sprintf(`<p>One of your OpenAI max plan sign-ins in OpenPaths is not working.</p>
<p>Sign-in: <strong>%s</strong></p>
<p>OpenPaths noticed this when OpenAI rejected a request using this sign-in. We tried refreshing it with the stored refresh token, then skipped it and tried another available sign-in or fallback key.</p>
<p>Please sign in with OpenAI again from Account so future requests keep working.</p>
<p>Error: %s</p>`, html.EscapeString(label), msg)
	if sendErr := email.Send(toEmail, "OpenPaths OpenAI max plan credential needs attention", body); sendErr != nil {
		log.Printf("openai max plan credential alert email failed for %s: %v", toEmail, sendErr)
	}
}

type refreshedOpenAIMaxPlanAuth struct {
	APIKey       string
	RefreshToken string
	AuthJSON     string
}

const (
	openAIAccessTokenRefreshWindow = 10 * time.Minute
	openAIUnknownTokenMaxAge       = 3 * time.Hour
)

// RefreshStoredOpenAICodexCredentialIfNeeded proactively rotates a stored
// Codex sign-in when its access token is near expiry. It is exported for the
// refresh scheduler; request-time 401 recovery uses the same locked path with
// force=true below.
func RefreshStoredOpenAICodexCredentialIfNeeded(
	ctx context.Context,
	pkQ *queries.ProviderKeyQueries,
	key *queries.UserProviderKey,
) (bool, error) {
	if pkQ == nil || key == nil || key.Provider != openAIMaxPlanProvider || strings.TrimSpace(key.AuthJSON) == "" {
		return false, nil
	}
	if !openAIAuthNeedsRefresh(key.AuthJSON, time.Now()) {
		return false, nil
	}
	_, changed, err := refreshStoredOpenAIMaxPlanAuthJSON(
		ctx, pkQ, key.UserID, key.Provider, key.AuthJSON, false,
	)
	return changed, err
}

// refreshStoredOpenAIMaxPlanAuthJSON locks the database row before calling the
// rotating-token endpoint. If another replica already updated the observed
// auth JSON, its newer result is reused instead of consuming the old refresh
// token a second time.
func refreshStoredOpenAIMaxPlanAuthJSON(
	ctx context.Context,
	pkQ *queries.ProviderKeyQueries,
	userID, providerName, observedAuthJSON string,
	force bool,
) (*refreshedOpenAIMaxPlanAuth, bool, error) {
	if pkQ == nil {
		return nil, false, fmt.Errorf("provider key store is unavailable")
	}
	var result *refreshedOpenAIMaxPlanAuth
	current, changed, err := pkQ.UpdateAuthJSONLocked(ctx, userID, providerName, func(current string) (string, error) {
		// A concurrent refresher won the race. Reuse what it persisted.
		if observedAuthJSON != "" && current != observedAuthJSON {
			var snapshotErr error
			result, snapshotErr = openAIMaxPlanAuthSnapshot(current)
			return current, snapshotErr
		}
		if !force && !openAIAuthNeedsRefresh(current, time.Now()) {
			var snapshotErr error
			result, snapshotErr = openAIMaxPlanAuthSnapshot(current)
			return current, snapshotErr
		}
		snapshot, snapshotErr := openAIMaxPlanAuthSnapshot(current)
		if snapshotErr != nil {
			return "", snapshotErr
		}
		result, snapshotErr = refreshOpenAIMaxPlanAuthJSON(ctx, current, snapshot.RefreshToken)
		if snapshotErr != nil {
			return "", snapshotErr
		}
		return result.AuthJSON, nil
	})
	if err != nil {
		return nil, false, err
	}
	if result == nil {
		result, err = openAIMaxPlanAuthSnapshot(current)
		if err != nil {
			return nil, false, err
		}
	}
	return result, changed, nil
}

func openAIMaxPlanAuthSnapshot(raw string) (*refreshedOpenAIMaxPlanAuth, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil, err
	}
	apiKey := openAIKeyFromAuthObject(root)
	refreshToken := refreshTokenFromAuthObject(root)
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI sign-in is missing its derived API key")
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("OpenAI sign-in is missing its refresh token")
	}
	return &refreshedOpenAIMaxPlanAuth{
		APIKey:       apiKey,
		RefreshToken: refreshToken,
		AuthJSON:     raw,
	}, nil
}

func openAIAuthNeedsRefresh(raw string, now time.Time) bool {
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil || refreshTokenFromAuthObject(root) == "" {
		return false
	}
	if tokens, ok := root["tokens"].(map[string]any); ok {
		if expiresAt, ok := jwtExpiration(stringField(tokens, "access_token")); ok {
			return !expiresAt.After(now.Add(openAIAccessTokenRefreshWindow))
		}
	}
	if lastRefresh := stringField(root, "last_refresh"); lastRefresh != "" {
		if refreshedAt, err := time.Parse(time.RFC3339, lastRefresh); err == nil {
			return !refreshedAt.After(now.Add(-openAIUnknownTokenMaxAge))
		}
	}
	// Older stored sign-ins did not record token expiry or last_refresh. Rotate
	// them once so they are upgraded to the current format.
	return true
}

func jwtExpiration(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		ExpiresAt int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.ExpiresAt <= 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.ExpiresAt, 0), true
}

func refreshOpenAIMaxPlanAuthJSON(ctx context.Context, rawAuthJSON, refreshToken string) (*refreshedOpenAIMaxPlanAuth, error) {
	tokenResp, err := requestOpenAIAuthRefresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	idToken := strings.TrimSpace(tokenResp.IDToken)
	if idToken == "" {
		idToken = idTokenFromAuthJSON(rawAuthJSON)
	}
	if idToken == "" {
		return nil, fmt.Errorf("refresh response did not include id_token")
	}
	apiKey, err := exchangeOpenAIIDTokenForAPIKey(ctx, idToken)
	if err != nil {
		return nil, err
	}
	nextRefreshToken := strings.TrimSpace(tokenResp.RefreshToken)
	if nextRefreshToken == "" {
		nextRefreshToken = refreshToken
	}
	updated, err := updateOpenAIMaxPlanAuthJSON(rawAuthJSON, apiKey, tokenResp.IDToken, tokenResp.AccessToken, tokenResp.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &refreshedOpenAIMaxPlanAuth{APIKey: apiKey, RefreshToken: nextRefreshToken, AuthJSON: updated}, nil
}

type openAIRefreshTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func requestOpenAIAuthRefresh(ctx context.Context, refreshToken string) (*openAIRefreshTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", openAIOAuthClientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIOAuthTokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := openAIOAuthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("refresh token exchange returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed openAIRefreshTokenResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func exchangeOpenAIIDTokenForAPIKey(ctx context.Context, idToken string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("client_id", openAIOAuthClientID)
	form.Set("requested_token", "openai-api-key")
	form.Set("subject_token", idToken)
	form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:id_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIOAuthTokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := openAIOAuthHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("api key token exchange returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return "", fmt.Errorf("api key token exchange did not return access_token")
	}
	return strings.TrimSpace(parsed.AccessToken), nil
}

func idTokenFromAuthJSON(raw string) string {
	var root map[string]any
	if json.Unmarshal([]byte(raw), &root) != nil {
		return ""
	}
	tokens, _ := root["tokens"].(map[string]any)
	if tokens == nil {
		return ""
	}
	if token := stringField(tokens, "id_token"); token != "" {
		return token
	}
	if idToken, ok := tokens["id_token"].(map[string]any); ok {
		return stringField(idToken, "raw_jwt")
	}
	return ""
}

func updateOpenAIMaxPlanAuthJSON(rawAuthJSON, apiKey, idToken, accessToken, refreshToken string) (string, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(rawAuthJSON), &root); err != nil {
		return "", err
	}
	root["auth_mode"] = "chatgpt"
	root["OPENAI_API_KEY"] = apiKey
	tokens, _ := root["tokens"].(map[string]any)
	if tokens == nil {
		tokens = make(map[string]any)
		root["tokens"] = tokens
	}
	if strings.TrimSpace(idToken) != "" {
		tokens["id_token"] = idToken
	}
	if strings.TrimSpace(accessToken) != "" {
		tokens["access_token"] = accessToken
	}
	if strings.TrimSpace(refreshToken) != "" {
		tokens["refresh_token"] = refreshToken
	}
	root["last_refresh"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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
