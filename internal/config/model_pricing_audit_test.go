package config

import (
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func loadAuditConfig(t *testing.T) map[string]*model.ModelConfig {
	t.Helper()
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*model.ModelConfig{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		byName[m.ID] = m
		for _, a := range m.Aliases {
			if _, clash := byName[a]; !clash {
				byName[a] = m
			}
		}
	}
	return byName
}

// TestAuditedTokenPrices pins the prices that were wrong because another
// provider's rate card had been pasted in: deepseek-v4-pro carried Fireworks'
// $1.74/$3.48, and fireworks/gpt-oss-120b carried GLM's $0.90 flat. Each value
// here comes from the serving provider's own published rate card.
func TestAuditedTokenPrices(t *testing.T) {
	byName := loadAuditConfig(t)
	want := map[string]struct{ in, cache, out float64 }{
		// api-docs.deepseek.com/quick_start/pricing
		"deepseek-v4-pro":   {0.435, 0.003625, 0.87},
		"deepseek-v4-flash": {0.14, 0.0028, 0.28},
		// docs.fireworks.ai/serverless/pricing
		"fireworks/deepseek-v4-pro": {1.74, 0.145, 3.48},
		"fireworks/gpt-oss-120b":    {0.15, 0.015, 0.60},
		"fireworks/glm-5.2":         {1.40, 0.14, 4.40},
		"fireworks/kimi-k2.6":       {0.95, 0.16, 4.00},
		// api.x.ai/v1/language-models
		"grok-4.5": {2.00, 0.30, 6.00},
		"grok-4.3": {1.25, 0.20, 2.50},
		// openrouter.ai/api/v1/models (50%-off promotion through August 27, 2026)
		"gemini-3.7-flash": {0.375, 0.0375, 1.875},
		// ai.google.dev/gemini-api/docs/pricing
		"gemini-3.6-flash":      {0.75, 0, 3.75},
		"gemini-3.5-flash-lite": {0.30, 0, 2.50},
		// developers.openai.com/api/docs/pricing
		"gpt-5.5-pro": {30.00, 0, 180.00},
		// openrouter.ai/api/v1/models
		"glm-5.3-flash":    {0.075, 0.015, 0.25},
		"glm-5.3":          {1.40, 0.26, 4.40},
		"or/glm-5.3":       {1.40, 0.26, 4.40},
		"or/gpt-5.6-sol":   {5.00, 0.50, 30.00},
		"or/gpt-5.6-terra": {1.00, 0.10, 6.00},
		"or/gpt-5.6-luna":  {0.10, 0.01, 0.60},
		"or/qwen3.8-max":   {2.00, 0.25, 6.00},
		"or/gpt-5-codex":   {1.75, 0.175, 14.00},
	}
	for id, w := range want {
		m, ok := byName[id]
		if !ok {
			t.Errorf("%s missing from config.yaml", id)
			continue
		}
		if m.InputPricePer1M != w.in || m.OutputPricePer1M != w.out {
			t.Errorf("%s = %v/%v, want %v/%v", id, m.InputPricePer1M, m.OutputPricePer1M, w.in, w.out)
		}
		if w.cache > 0 && m.InputCacheHitPricePer1M != w.cache {
			t.Errorf("%s cache-hit = %v, want %v", id, m.InputCacheHitPricePer1M, w.cache)
		}
	}
}

func TestRetiredOxAliasesResolveToPaidGLM53Flash(t *testing.T) {
	byName := loadAuditConfig(t)
	for _, id := range []string{
		"openpaths/stealth/ox-alpha",
		"stealth/ox-alpha",
		"openpaths/ox-alpha",
		"ox-alpha",
	} {
		m := byName[id]
		if m == nil {
			t.Errorf("compatibility alias %s is missing", id)
			continue
		}
		if m.ID != "glm-5.3-flash" || m.ProviderModelID != "z-ai/glm-5.3-flash" {
			t.Errorf("%s resolves to %s/%s, want paid GLM-5.3 Flash", id, m.ID, m.ProviderModelID)
		}
		if m.InputPricePer1M <= 0 || m.OutputPricePer1M <= 0 {
			t.Errorf("%s still has free pricing %v/%v", id, m.InputPricePer1M, m.OutputPricePer1M)
		}
	}
}

// TestCacheHitRatesBelowInput catches the direction error: billing only charges
// the cache rate when it is set, so a cache rate at or above the input rate is
// either a copy of the wrong column or a lost discount.
func TestCacheHitRatesBelowInput(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range loadAuditConfig(t) {
		if seen[m.ID] || m.InputCacheHitPricePer1M == 0 {
			continue
		}
		seen[m.ID] = true
		if m.InputPricePer1M <= 0 {
			t.Errorf("%s prices a cache hit but has no input rate", m.ID)
			continue
		}
		if m.InputCacheHitPricePer1M >= m.InputPricePer1M {
			t.Errorf("%s cache-hit rate %v is not below the input rate %v",
				m.ID, m.InputCacheHitPricePer1M, m.InputPricePer1M)
		}
	}
}

// TestFallbacksNotRepeated catches a fallback listed twice in one chain, which
// silently wastes a candidate slot. Two catalogue ids pointing at the same
// upstream model is deliberate here (aliased mirrors act as each other's
// fallbacks) and the router expands one level only, so loops are harmless.
func TestFallbacksNotRepeated(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range loadAuditConfig(t) {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		listed := map[string]bool{}
		for _, fb := range m.FallbackModels {
			if listed[fb] {
				t.Errorf("%s lists fallback %q twice", m.ID, fb)
			}
			listed[fb] = true
		}
	}
}

// TestLongContextTiersAreHigherAndSane guards the unit and direction errors a
// tiered rate card invites: a long rate below the base rate silently discounts
// big prompts, and a per-1K figure pasted into a per-1M field shows up as an
// absurd multiple.
func TestLongContextTiersAreHigherAndSane(t *testing.T) {
	byName := loadAuditConfig(t)
	seen := map[string]bool{}
	for _, m := range byName {
		if seen[m.ID] || m.LongContextThreshold == 0 {
			continue
		}
		seen[m.ID] = true
		if m.InputPricePer1MLong == 0 && m.OutputPricePer1MLong == 0 {
			t.Errorf("%s sets long_context_threshold but no long rates", m.ID)
		}
		for _, p := range []struct {
			name       string
			base, long float64
		}{
			{"input", m.InputPricePer1M, m.InputPricePer1MLong},
			{"cache", m.InputCacheHitPricePer1M, m.InputCacheHitPricePer1MLong},
			{"output", m.OutputPricePer1M, m.OutputPricePer1MLong},
		} {
			if p.long == 0 {
				continue
			}
			if p.long < p.base {
				t.Errorf("%s long %s rate %v is below the base rate %v", m.ID, p.name, p.long, p.base)
			}
			if p.base > 0 && p.long > p.base*10 {
				t.Errorf("%s long %s rate %v is %.0fx the base rate — unit error?", m.ID, p.name, p.long, p.long/p.base)
			}
		}
	}
}

// TestXAILongContextTiersConfigured keeps the >200K doubling xAI publishes on
// /v1/language-models attached to every billed Grok text model. Without it a
// long prompt bills at half its real cost.
func TestXAILongContextTiersConfigured(t *testing.T) {
	byName := loadAuditConfig(t)
	for _, id := range []string{
		"grok-4.5", "grok-4.3", "grok-3-mini",
		"grok-4.20-0309-reasoning", "grok-4.20-0309-non-reasoning",
		"grok-4.20-multi-agent-0309", "grok-build-0.1",
	} {
		m, ok := byName[id]
		if !ok {
			t.Errorf("%s missing from config.yaml", id)
			continue
		}
		if m.LongContextThreshold != 200000 {
			t.Errorf("%s long_context_threshold = %d, want 200000", id, m.LongContextThreshold)
		}
		if m.InputPricePer1MLong != m.InputPricePer1M*2 {
			t.Errorf("%s long input = %v, want 2x %v", id, m.InputPricePer1MLong, m.InputPricePer1M)
		}
		if m.OutputPricePer1MLong != m.OutputPricePer1M*2 {
			t.Errorf("%s long output = %v, want 2x %v", id, m.OutputPricePer1MLong, m.OutputPricePer1M)
		}
	}
}

func TestXAIMediaAndVoicePricing(t *testing.T) {
	byName := loadAuditConfig(t)

	for id, hourly := range map[string]float64{
		"grok-voice-think-fast-1.0": 3.00,
		"grok-voice-think-fast-2.0": 4.80,
		"grok-voice-latest":         4.80,
	} {
		if m := byName[id]; m == nil || m.PricePerHour != hourly {
			t.Errorf("%s hourly price = %v, want %v", id, func() float64 {
				if m == nil {
					return 0
				}
				return m.PricePerHour
			}(), hourly)
		}
	}

	quality := byName["grok-imagine-image-quality"]
	if quality == nil || quality.PricePerImageByResolution["1k"] != 0.05 || quality.PricePerImageByResolution["2k"] != 0.07 || quality.PricePerInputImage != 0.01 {
		t.Errorf("grok-imagine-image-quality resolution/input pricing is not the published xAI rate")
	}
	v15 := byName["grok-imagine-video-1.5"]
	if v15 == nil || v15.PricePerSecondByResolution["480p"] != 0.08 || v15.PricePerSecondByResolution["720p"] != 0.14 || v15.PricePerSecondByResolution["1080p"] != 0.25 {
		t.Errorf("grok-imagine-video-1.5 resolution pricing is not the published xAI rate")
	}
	if stt := byName["xai-stt"]; stt == nil || stt.PricePerHour != 0.10 {
		t.Errorf("xai-stt REST price = %v, want 0.10/hour", func() float64 {
			if stt == nil {
				return 0
			}
			return stt.PricePerHour
		}())
	}
	if tts := byName["xai-tts"]; tts == nil || tts.PricePer1MCharacters != 15.00 || tts.InputPricePer1M != 0 {
		t.Errorf("xai-tts must bill $15 per 1M characters, not language-model tokens")
	}
}

// TestFallbackModelsResolve catches typos in the fallback chains, including the
// ones added so retired upstream ids (dall-e-3, mixtral-8x7b-32768,
// claude-opus-4-20250514) degrade to a live model instead of 404ing.
func TestFallbackModelsResolve(t *testing.T) {
	byName := loadAuditConfig(t)
	seen := map[string]bool{}
	for _, m := range byName {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		for _, fb := range m.FallbackModels {
			if _, ok := byName[fb]; !ok {
				t.Errorf("%s falls back to unknown model %q", m.ID, fb)
			}
			if fb == m.ID {
				t.Errorf("%s falls back to itself", m.ID)
			}
		}
	}
}

// TestRetiredProviderModelIDsRepointed keeps ids whose upstream model was
// withdrawn pointed at something the provider still serves. Each of these
// returned a not-found from the provider during the 2026-07 catalogue audit.
func TestRetiredProviderModelIDsRepointed(t *testing.T) {
	byName := loadAuditConfig(t)
	for id, wantNot := range map[string]string{
		"grok-3-mini":    "grok-3-mini",
		"gpt-5.6":        "gpt-5.6",
		"or/gpt-5-codex": "openai/gpt-5-codex",
	} {
		m, ok := byName[id]
		if !ok {
			t.Errorf("%s missing from config.yaml", id)
			continue
		}
		if m.ProviderModelID == wantNot {
			t.Errorf("%s still routes to retired upstream id %q", id, wantNot)
		}
	}
	for _, removed := range []string{"or/nemotron-embed", "or/lfm-thinking", "or/lfm-instruct"} {
		if _, exists := byName[removed]; exists {
			t.Errorf("withdrawn OpenRouter model %q is still advertised", removed)
		}
	}
	if trinity := byName["trinity-mini"]; trinity == nil || trinity.ProviderModelID != "arcee-ai/trinity-large-thinking" {
		t.Errorf("retired trinity-mini alias does not resolve to the live Trinity route")
	}
	// The :free OpenRouter variants below were withdrawn; a paid slug is the
	// only working route, so a zero price would mean serving paid tokens free.
	for _, id := range []string{"or/stepfun-flash", "or/solar-pro-3", "or/arcee-trinity"} {
		m, ok := byName[id]
		if !ok {
			t.Errorf("%s missing from config.yaml", id)
			continue
		}
		if len(m.ProviderModelID) > 5 && m.ProviderModelID[len(m.ProviderModelID)-5:] == ":free" {
			t.Errorf("%s still routes to a withdrawn :free slug %q", id, m.ProviderModelID)
		}
		if m.InputPricePer1M == 0 || m.OutputPricePer1M == 0 {
			t.Errorf("%s routes to a paid slug but bills %v/%v", id, m.InputPricePer1M, m.OutputPricePer1M)
		}
	}
}
