package handler

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
)

// Platform-wide OpenAI max-plan credential.
//
// An admin signs in once with their ChatGPT/Codex max plan through the normal
// Account OAuth device flow (stored under the openai_codex provider on their
// user row). The email of that admin is configured via ADMIN_OPENAI_MAX_PLAN_EMAIL.
// From then on the backend uses that credential for ALL OpenAI traffic: it is
// tried after a request user's own OpenAI keys and before the platform env key,
// with the same circuit breaker as customer max plans and a proactive refresh
// every 4h (the derived API key is short lived and is rotated via the stored
// refresh token). On failure it circuit-breaks and the router falls through to
// the configured platform OPENAI_API_KEY.

var adminOpenAIMaxPlan = struct {
	sync.RWMutex
	userID string
	creds  []*openAIMaxPlanCredential
}{}

func setAdminOpenAIMaxPlanCreds(userID string, creds []*openAIMaxPlanCredential) {
	adminOpenAIMaxPlan.Lock()
	adminOpenAIMaxPlan.userID = userID
	adminOpenAIMaxPlan.creds = creds
	adminOpenAIMaxPlan.Unlock()
}

func adminOpenAIMaxPlanSnapshot() []*openAIMaxPlanCredential {
	adminOpenAIMaxPlan.RLock()
	creds := adminOpenAIMaxPlan.creds
	adminOpenAIMaxPlan.RUnlock()
	return creds
}

// adminOpenAIMaxPlanAttempts returns provider attempts for the shared admin
// credential, filtered to those not currently circuit-broken.
func adminOpenAIMaxPlanAttempts() []selectedProvider {
	creds := adminOpenAIMaxPlanSnapshot()
	if len(creds) == 0 {
		return nil
	}
	now := time.Now()
	healthy := make([]*openAIMaxPlanCredential, 0, len(creds))
	for _, c := range creds {
		if c != nil && c.APIKey != "" && isOpenAIMaxPlanCredentialHealthy(c.ID, now) {
			healthy = append(healthy, c)
		}
	}
	if len(healthy) == 0 {
		return nil
	}
	shuffleCredentials(healthy)
	attempts := make([]selectedProvider, 0, len(healthy))
	for _, c := range healthy {
		attempts = append(attempts, selectedProvider{
			provider: makeUserProvider("openai", c.APIKey),
			byok:     false, // platform credential: still bill the requesting user
			cred:     c,
		})
	}
	return attempts
}

// loadAdminOpenAIMaxPlanCreds parses an auth_json blob into credentials tagged
// as the shared platform credential owned by userID.
func loadAdminOpenAIMaxPlanCreds(userID, providerName, authJSON string) []*openAIMaxPlanCredential {
	creds := parseOpenAIMaxPlanCredentials(userID, providerName, authJSON)
	for _, c := range creds {
		c.Platform = true
		c.OwnerUserID = userID
	}
	return creds
}

// handleAdminOpenAIMaxPlanFailure reacts to a failed request on the shared
// credential: on an auth error it attempts a token refresh (persisted to the
// admin's row and reloaded into the cache) and clears the breaker on success,
// otherwise it trips the breaker so the router falls back to the env key.
func (h *ChatHandler) handleAdminOpenAIMaxPlanFailure(
	ctx context.Context,
	cred *openAIMaxPlanCredential,
	err error,
	allowRefresh bool,
) *openAIMaxPlanCredential {
	if cred == nil {
		return nil
	}
	if !isCredentialAuthError(err) || h.providerKeyQ == nil {
		markOpenAIMaxPlanCredentialFailure(cred.ID, err)
		return nil
	}
	if allowRefresh {
		if refreshed, rerr := tryRefreshAdminOpenAIMaxPlan(ctx, h.providerKeyQ, cred); refreshed != nil {
			markOpenAIMaxPlanCredentialHealthy(refreshed.ID)
			return refreshed
		} else if rerr != nil {
			log.Printf("admin openai max plan refresh failed for %s: %v", cred.Label, rerr)
		}
	}
	markOpenAIMaxPlanCredentialFailure(cred.ID, err)
	return nil
}

func tryRefreshAdminOpenAIMaxPlan(ctx context.Context, pkQ *queries.ProviderKeyQueries, cred *openAIMaxPlanCredential) (*openAIMaxPlanCredential, error) {
	if cred.RefreshToken == "" || cred.AuthJSON == "" || cred.OwnerUserID == "" || cred.ProviderName == "" {
		return nil, nil
	}
	refreshed, _, err := refreshStoredOpenAIMaxPlanAuthJSON(
		ctx, pkQ, cred.OwnerUserID, cred.ProviderName, cred.AuthJSON, true,
	)
	if err != nil {
		return nil, err
	}
	if refreshed.APIKey == "" || refreshed.AuthJSON == "" {
		return nil, nil
	}
	creds := loadAdminOpenAIMaxPlanCreds(cred.OwnerUserID, cred.ProviderName, refreshed.AuthJSON)
	setAdminOpenAIMaxPlanCreds(cred.OwnerUserID, creds)
	if len(creds) == 0 {
		return nil, nil
	}
	return creds[0], nil
}

// AdminMaxPlanRefresher keeps the shared admin credential loaded and fresh.
type AdminMaxPlanRefresher struct {
	pkQ   *queries.ProviderKeyQueries
	userQ *queries.UserQueries
	email string
	stop  chan struct{}
}

func NewAdminMaxPlanRefresher(pkQ *queries.ProviderKeyQueries, userQ *queries.UserQueries, email string) *AdminMaxPlanRefresher {
	return &AdminMaxPlanRefresher{
		pkQ:   pkQ,
		userQ: userQ,
		email: strings.TrimSpace(email),
		stop:  make(chan struct{}),
	}
}

// activeAdminMaxPlanRefresher lets admin HTTP handlers trigger a refresh.
var activeAdminMaxPlanRefresher struct {
	sync.RWMutex
	r *AdminMaxPlanRefresher
}

func (m *AdminMaxPlanRefresher) Start() {
	if m.email == "" || m.pkQ == nil || m.userQ == nil {
		return
	}
	activeAdminMaxPlanRefresher.Lock()
	activeAdminMaxPlanRefresher.r = m
	activeAdminMaxPlanRefresher.Unlock()
	go m.loop()
	log.Printf("Admin OpenAI max plan enabled for %s (refresh every 4h)", m.email)
}

// TriggerAdminOpenAIMaxPlanRefresh reloads the shared credential immediately
// and rotates it only when it is near expiry. Returns false if not enabled.
func TriggerAdminOpenAIMaxPlanRefresh() bool {
	activeAdminMaxPlanRefresher.RLock()
	r := activeAdminMaxPlanRefresher.r
	activeAdminMaxPlanRefresher.RUnlock()
	if r == nil {
		return false
	}
	go r.refresh(false)
	return true
}

// ForceAdminOpenAIMaxPlanRefresh is the explicit admin action: it reloads and
// rotates now even if the access token is not yet near expiry.
func ForceAdminOpenAIMaxPlanRefresh() bool {
	activeAdminMaxPlanRefresher.RLock()
	r := activeAdminMaxPlanRefresher.r
	activeAdminMaxPlanRefresher.RUnlock()
	if r == nil {
		return false
	}
	go r.refresh(true)
	return true
}

// AdminOpenAIMaxPlanEmail returns the configured admin email, or "".
func AdminOpenAIMaxPlanEmail() string {
	activeAdminMaxPlanRefresher.RLock()
	r := activeAdminMaxPlanRefresher.r
	activeAdminMaxPlanRefresher.RUnlock()
	if r == nil {
		return ""
	}
	return r.email
}

// AdminOpenAIMaxPlanStatus is the exported view for the admin dashboard.
func AdminOpenAIMaxPlanStatus() (email, userID string, total, healthy int) {
	userID, total, healthy = adminOpenAIMaxPlanStatus()
	return AdminOpenAIMaxPlanEmail(), userID, total, healthy
}

// AdminOpenAIMaxPlanAuthInfo reports how the shared credential was obtained.
// A pasted API key stores auth_mode "apikey" and carries no refresh token, so
// it silently rots when the key is revoked; only a ChatGPT/Codex OAuth sign-in
// ("chatgpt") can be rotated by the 4h refresher. Surfacing this in /admin is
// what distinguishes "refresh is broken" from "the credential is a dead key".
func AdminOpenAIMaxPlanAuthInfo() (authMode string, refreshable bool) {
	for _, c := range adminOpenAIMaxPlanSnapshot() {
		if c == nil {
			continue
		}
		if c.RefreshToken != "" {
			refreshable = true
		}
		if authMode == "" && c.AuthJSON != "" {
			var parsed struct {
				AuthMode string `json:"auth_mode"`
			}
			if json.Unmarshal([]byte(c.AuthJSON), &parsed) == nil {
				authMode = parsed.AuthMode
			}
		}
	}
	return authMode, refreshable
}

func (m *AdminMaxPlanRefresher) Stop() {
	if m.email == "" {
		return
	}
	close(m.stop)
}

func (m *AdminMaxPlanRefresher) loop() {
	m.refresh(false)
	ticker := time.NewTicker(4 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.refresh(false)
		case <-m.stop:
			return
		}
	}
}

func (m *AdminMaxPlanRefresher) refresh(force bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	user, err := m.userQ.GetByEmail(ctx, m.email)
	if err != nil || user == nil {
		log.Printf("admin max plan: user %s not found: %v", m.email, err)
		return
	}
	pk, err := m.pkQ.GetByUserAndProvider(ctx, user.ID, openAIMaxPlanProvider)
	if err != nil || pk == nil || strings.TrimSpace(pk.AuthJSON) == "" {
		log.Printf("admin max plan: no %s sign-in for %s (have the admin sign in with OpenAI on Account)", openAIMaxPlanProvider, m.email)
		return
	}

	creds := loadAdminOpenAIMaxPlanCreds(user.ID, openAIMaxPlanProvider, pk.AuthJSON)
	refreshable := false
	for _, cred := range creds {
		if cred.RefreshToken != "" {
			refreshable = true
			break
		}
	}
	if refreshable {
		if refreshed, changed, rerr := refreshStoredOpenAIMaxPlanAuthJSON(
			ctx, m.pkQ, user.ID, openAIMaxPlanProvider, pk.AuthJSON, force,
		); rerr == nil && refreshed != nil && refreshed.AuthJSON != "" {
			creds = loadAdminOpenAIMaxPlanCreds(user.ID, openAIMaxPlanProvider, refreshed.AuthJSON)
			if changed {
				log.Printf("admin max plan: refreshed credential for %s", m.email)
			}
		} else if rerr != nil {
			log.Printf("admin max plan: proactive refresh failed for %s: %v", m.email, rerr)
		}
	}

	setAdminOpenAIMaxPlanCreds(user.ID, creds)
	// Give freshly loaded credentials a clean breaker so a rotated key is retried.
	for _, c := range creds {
		markOpenAIMaxPlanCredentialHealthy(c.ID)
	}
	log.Printf("admin max plan: loaded %d credential(s) for %s", len(creds), m.email)
}

// adminOpenAIMaxPlanStatus reports current state for the admin dashboard.
func adminOpenAIMaxPlanStatus() (userID string, total, healthy int) {
	creds := adminOpenAIMaxPlanSnapshot()
	now := time.Now()
	for _, c := range creds {
		if c == nil || c.APIKey == "" {
			continue
		}
		total++
		if isOpenAIMaxPlanCredentialHealthy(c.ID, now) {
			healthy++
		}
	}
	adminOpenAIMaxPlan.RLock()
	userID = adminOpenAIMaxPlan.userID
	adminOpenAIMaxPlan.RUnlock()
	return userID, total, healthy
}
