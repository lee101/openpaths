package model

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ScopeNone          = ""
	ScopeChat          = "chat.completions"
	ScopeModels        = "models:read"
	ScopeImages        = "images"
	ScopeVideo         = "video"
	ScopeAudio         = "audio"
	ScopeEmbeddings    = "embeddings"
	ScopeSearch        = "search"
	Scope3D            = "3d"
	ScopeFiles         = "files"
	ScopeFineTuning    = "fine_tuning"
	ScopeRealtime      = "realtime"
	ScopeMCP           = "mcp"
	ScopeArtifacts     = "artifacts"
	ScopeAgents        = "agents"
	ScopeUsageRead     = "usage:read"
	ScopeBillingRead   = "billing:read"
	ScopeAccountManage = "account:manage"
	ScopeKeysManage    = "keys:manage"
)

type ScopeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var scopeCatalog = []ScopeInfo{
	{ScopeChat, "Chat completions, Anthropic messages, structured extract/decide and forecasts"},
	{ScopeModels, "List and describe models"},
	{ScopeImages, "Image generation and edits"},
	{ScopeVideo, "Video generation, edits and extensions"},
	{ScopeAudio, "Speech, transcription and music generation"},
	{ScopeEmbeddings, "Embeddings and decisions"},
	{ScopeSearch, "Web search and deep research"},
	{Scope3D, "3D generation and rigging"},
	{ScopeFiles, "File upload and listing"},
	{ScopeFineTuning, "Fine-tuning jobs"},
	{ScopeRealtime, "Realtime WebSocket sessions"},
	{ScopeMCP, "MCP server (each tool also needs its own scope)"},
	{ScopeArtifacts, "Create, search and manage artifacts"},
	{ScopeAgents, "Create and run agents"},
	{ScopeUsageRead, "Read usage and spend statistics"},
	{ScopeBillingRead, "Read balance and transactions"},
	{ScopeAccountManage, "Account settings: guardrails, provider keys, orgs, skills, payments"},
	{ScopeKeysManage, "Management key: list, create, rotate and revoke API keys"},
}

func ScopeCatalog() []ScopeInfo {
	return append([]ScopeInfo(nil), scopeCatalog...)
}

func UsageScopes() []string {
	out := make([]string, 0, len(scopeCatalog))
	for _, s := range scopeCatalog {
		if s.Name != ScopeKeysManage {
			out = append(out, s.Name)
		}
	}
	return out
}

func AllScopes() []string {
	return append(UsageScopes(), ScopeKeysManage)
}

func ValidScope(s string) bool {
	for _, c := range scopeCatalog {
		if c.Name == s {
			return true
		}
	}
	return false
}

func NormalizeScopes(in []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if !ValidScope(s) {
			return nil, fmt.Errorf("unknown scope %q", s)
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (k *APIKey) EffectiveScopes() []string {
	if k.Scopes == nil {
		return UsageScopes()
	}
	return k.Scopes
}

func (k *APIKey) HasScope(scope string) bool {
	if scope == ScopeNone {
		return true
	}
	for _, s := range k.EffectiveScopes() {
		if s == scope {
			return true
		}
	}
	return false
}

func (k *APIKey) IsManagement() bool {
	return k.HasScope(ScopeKeysManage)
}

func (k *APIKey) RevokedAsOf(now time.Time) bool {
	return k.Revoked || (k.RevokedAt != nil && !k.RevokedAt.After(now))
}

func (k *APIKey) ExpiredAsOf(now time.Time) bool {
	return k.ExpiresAt != nil && !k.ExpiresAt.After(now)
}
