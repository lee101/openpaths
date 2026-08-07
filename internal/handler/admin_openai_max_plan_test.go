package handler

import (
	"testing"
	"time"
)

func resetAdminMaxPlan() {
	setAdminOpenAIMaxPlanCreds("", nil)
	openAIMaxPlanCircuits.Lock()
	openAIMaxPlanCircuits.items = map[string]credentialCircuit{}
	openAIMaxPlanCircuits.Unlock()
}

func TestLoadAdminOpenAIMaxPlanCredsTagsPlatform(t *testing.T) {
	resetAdminMaxPlan()
	raw := `{"auth_mode":"chatgpt","OPENAI_API_KEY":"sk-admin","tokens":{"refresh_token":"rt-1"}}`
	creds := loadAdminOpenAIMaxPlanCreds("admin-user", openAIMaxPlanProvider, raw)
	if len(creds) != 1 {
		t.Fatalf("len(creds) = %d, want 1", len(creds))
	}
	c := creds[0]
	if !c.Platform {
		t.Fatalf("credential not tagged Platform")
	}
	if c.OwnerUserID != "admin-user" {
		t.Fatalf("OwnerUserID = %q, want admin-user", c.OwnerUserID)
	}
	if c.RefreshToken != "rt-1" {
		t.Fatalf("RefreshToken = %q, want rt-1", c.RefreshToken)
	}
	if c.APIKey != "sk-admin" {
		t.Fatalf("APIKey = %q, want sk-admin", c.APIKey)
	}
}

func TestAdminOpenAIMaxPlanAttempts(t *testing.T) {
	resetAdminMaxPlan()
	if got := adminOpenAIMaxPlanAttempts(); got != nil {
		t.Fatalf("expected no attempts when unconfigured, got %d", len(got))
	}

	setAdminOpenAIMaxPlanCreds("admin-user",
		loadAdminOpenAIMaxPlanCreds("admin-user", openAIMaxPlanProvider,
			`{"OPENAI_API_KEY":"sk-admin","tokens":{"refresh_token":"rt-1"}}`))

	attempts := adminOpenAIMaxPlanAttempts()
	if len(attempts) != 1 {
		t.Fatalf("len(attempts) = %d, want 1", len(attempts))
	}
	if attempts[0].byok {
		t.Fatalf("admin platform attempt must not be byok (user is still billed)")
	}
	if attempts[0].cred == nil || !attempts[0].cred.Platform {
		t.Fatalf("attempt cred missing or not Platform: %+v", attempts[0].cred)
	}
}

func TestAdminOpenAIMaxPlanCircuitBreakerFilters(t *testing.T) {
	resetAdminMaxPlan()
	setAdminOpenAIMaxPlanCreds("admin-user",
		loadAdminOpenAIMaxPlanCreds("admin-user", openAIMaxPlanProvider,
			`{"OPENAI_API_KEY":"sk-admin"}`))

	id := adminOpenAIMaxPlanSnapshot()[0].ID
	markOpenAIMaxPlanCredentialFailure(id, nil)
	if got := adminOpenAIMaxPlanAttempts(); got != nil {
		t.Fatalf("circuit-broken credential should yield no attempts, got %d", len(got))
	}
	markOpenAIMaxPlanCredentialHealthy(id)
	if got := adminOpenAIMaxPlanAttempts(); len(got) != 1 {
		t.Fatalf("healthy credential should yield 1 attempt, got %d", len(got))
	}
}

func TestAdminOpenAIMaxPlanStatus(t *testing.T) {
	resetAdminMaxPlan()
	setAdminOpenAIMaxPlanCreds("admin-user",
		loadAdminOpenAIMaxPlanCreds("admin-user", openAIMaxPlanProvider,
			`{"OPENAI_API_KEY":"sk-admin"}`))
	uid, total, healthy := adminOpenAIMaxPlanStatus()
	if uid != "admin-user" || total != 1 || healthy != 1 {
		t.Fatalf("status = (%q, %d, %d), want (admin-user, 1, 1)", uid, total, healthy)
	}
	markOpenAIMaxPlanCredentialFailure(adminOpenAIMaxPlanSnapshot()[0].ID, nil)
	if _, _, healthy := adminOpenAIMaxPlanStatus(); healthy != 0 {
		t.Fatalf("healthy = %d after breaker trip, want 0", healthy)
	}
	_ = time.Now
}

func TestTriggerAdminOpenAIMaxPlanRefreshWhenDisabled(t *testing.T) {
	activeAdminMaxPlanRefresher.Lock()
	activeAdminMaxPlanRefresher.r = nil
	activeAdminMaxPlanRefresher.Unlock()
	if TriggerAdminOpenAIMaxPlanRefresh() {
		t.Fatalf("expected false when no refresher configured")
	}
	if ForceAdminOpenAIMaxPlanRefresh() {
		t.Fatalf("expected forced refresh to be false when no refresher is configured")
	}
	if AdminOpenAIMaxPlanEmail() != "" {
		t.Fatalf("expected empty email when disabled")
	}
}
