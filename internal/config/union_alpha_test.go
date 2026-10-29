package config

import "testing"

// Union Alpha's free preview ended. The stealth/union-alpha ids are kept as
// compatibility aliases on paid DeepSeek V4 Flash so old code 404s never.
func TestRetiredUnionAlphaAliasesResolveToPaidDeepSeekFlash(t *testing.T) {
	models := loadAuditConfig(t)
	for _, id := range []string{
		"stealth/union-alpha",
		"union-alpha",
		"or/union-alpha",
		"openrouter/stealth/union-alpha",
		"openpaths/stealth/union-alpha",
		"openpaths/union-alpha",
		"openpaths-union-alpha",
	} {
		m := models[id]
		if m == nil {
			t.Errorf("compatibility alias %s is missing", id)
			continue
		}
		if m.ID != "deepseek-v4-flash" || m.Provider != "seedream" || m.ProviderModelID != "deepseek-v4-1-flash-261017" {
			t.Errorf("%s resolves to %s/%s, want paid ModelArk deepseek-v4-1-flash", id, m.ID, m.ProviderModelID)
		}
		if m.InputPricePer1M <= 0 || m.OutputPricePer1M <= 0 {
			t.Errorf("%s still has free pricing %v/%v", id, m.InputPricePer1M, m.OutputPricePer1M)
		}
	}
}
