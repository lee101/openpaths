package config

import "testing"

func TestNousMissingNoFreeRoute(t *testing.T) {
	models := loadAuditConfig(t)
	free := models["openpaths/free"]
	if free == nil || free.ID != "missingno" || free.Provider != "nous" || free.ProviderModelID != "stealth/missingno" {
		t.Fatalf("unexpected free route: %+v", free)
	}
	if free.InputPricePer1M != 0 || free.OutputPricePer1M != 0 || free.PricePerRequest != 0 || len(free.FallbackModels) != 0 || len(free.FallbackProviders) != 0 {
		t.Fatalf("free route must never silently incur paid fallback charges: %+v", free)
	}
	for _, id := range []string{"nous/meituan/longcat-2.5-preview:free", "nous/stepfun/step-5-preview:free", "nous/upstage/solar-mini4:free", "nous/poolside/laguna-s-2.1:free", "nous/poolside/laguna-xs-2.1:free", "nous/inclusionai/ling-3.0-flash-sante:free"} {
		m := models[id]
		if m == nil || m.Provider != "nous" || m.InputPricePer1M != 0 || m.OutputPricePer1M != 0 || m.PricePerRequest != 0 || len(m.FallbackModels) != 0 || len(m.FallbackProviders) != 0 {
			t.Errorf("invalid free Nous variant %s: %+v", id, m)
		}
	}
}
