package model

import (
	"testing"
	"time"
)

func TestNormalizeScopes(t *testing.T) {
	got, err := NormalizeScopes([]string{" images ", "chat.completions", "images"})
	if err != nil || len(got) != 2 || got[0] != ScopeChat || got[1] != ScopeImages {
		t.Fatalf("got %v %v", got, err)
	}
	if _, err := NormalizeScopes([]string{"images", "root"}); err == nil {
		t.Fatal("unknown scope accepted")
	}
}

func TestLegacyKeyScopes(t *testing.T) {
	k := &APIKey{}
	if !k.HasScope(ScopeChat) || !k.HasScope(ScopeAccountManage) || k.HasScope(ScopeKeysManage) || k.IsManagement() {
		t.Fatalf("legacy scopes %v", k.EffectiveScopes())
	}
}

func TestManagementKeyHasNoUsageScopes(t *testing.T) {
	k := &APIKey{Scopes: []string{ScopeKeysManage}}
	if !k.IsManagement() || k.HasScope(ScopeChat) || !k.HasScope(ScopeNone) {
		t.Fatal("management key scopes wrong")
	}
	if UsageScopes()[0] == ScopeKeysManage || len(AllScopes()) != len(UsageScopes())+1 {
		t.Fatal("catalog split wrong")
	}
}

func TestKeyLifecycleChecks(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Second), now.Add(time.Hour)
	if (&APIKey{RevokedAt: &future}).RevokedAsOf(now) || !(&APIKey{RevokedAt: &past}).RevokedAsOf(now) || !(&APIKey{Revoked: true}).RevokedAsOf(now) {
		t.Fatal("revocation window wrong")
	}
	if (&APIKey{ExpiresAt: &future}).ExpiredAsOf(now) || !(&APIKey{ExpiresAt: &past}).ExpiredAsOf(now) {
		t.Fatal("expiry wrong")
	}
}
