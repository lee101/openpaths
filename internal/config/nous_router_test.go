package config

import "testing"

func TestNousFreeVariantsNeverFallBackToPaid(t *testing.T) {
	models := loadAuditConfig(t)
	for _, id := range []string{"nous/meituan/longcat-2.5-preview:free", "nous/stepfun/step-5-preview:free", "nous/upstage/solar-mini4:free", "nous/poolside/laguna-s-2.1:free", "nous/poolside/laguna-xs-2.1:free", "nous/inclusionai/ling-3.0-flash-sante:free"} {
		m := models[id]
		if m == nil || m.Provider != "nous" || m.InputPricePer1M != 0 || m.OutputPricePer1M != 0 || m.PricePerRequest != 0 || len(m.FallbackModels) != 0 || len(m.FallbackProviders) != 0 {
			t.Errorf("invalid free Nous variant %s: %+v", id, m)
		}
	}
}
