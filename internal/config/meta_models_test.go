package config

import "testing"

func TestMetaModels(t *testing.T) {
	t.Skip("pending config: muse-spark-1.3 remap to muse-spark-1.3-contributor; prod WIP test ahead of prod config")
	models := loadAuditConfig(t)

	for _, id := range []string{"muse-spark-1.3", "muse-spark", "meta-muse-spark", "meta-chat"} {
		got := models[id]
		if got == nil || got.ID != "muse-spark-1.3-contributor" || got.ProviderModelID != "muse-spark-1.3-contributor" {
			t.Fatalf("%s resolves to %+v, want muse-spark-1.3-contributor", id, got)
		}
	}

	contrib := models["muse-spark-1.3-contributor"]
	if contrib == nil || contrib.Provider != "meta" || contrib.ProviderModelID != "muse-spark-1.3-contributor" {
		t.Fatalf("Muse Spark Contributor route = %+v", contrib)
	}
	if contrib.InputPricePer1M != 0.10 || contrib.InputCacheHitPricePer1M != 0.002 || contrib.OutputPricePer1M != 0.20 {
		t.Fatalf("Muse Spark Contributor pricing = %v/%v/%v", contrib.InputPricePer1M, contrib.InputCacheHitPricePer1M, contrib.OutputPricePer1M)
	}
	if got := models["muse-spark-contributor"]; got == nil || got.ID != "muse-spark-1.3-contributor" {
		t.Fatalf("muse-spark-contributor alias resolves to %v, want muse-spark-1.3-contributor", got)
	}

	image := models["muse-image-1.0"]
	if image == nil || image.Provider != "meta" || image.PricePerImage != 0.01 {
		t.Fatalf("Muse Image config = %+v", image)
	}

	voice := models["muse-voice-transcribe-1.0"]
	if voice == nil || voice.Provider != "meta" || voice.PricePerHour != 0.18 {
		t.Fatalf("Muse Voice config = %+v", voice)
	}
}

func TestJevRouter(t *testing.T) {
	t.Skip("pending config: typesafe/jev-router; prod WIP test ahead of prod config")
	models := loadAuditConfig(t)
	for _, id := range []string{"typesafe/jev-router", "jev-router", "jev", "or/jev-router"} {
		got := models[id]
		if got == nil || got.ID != "typesafe/jev-router" || got.Provider != "openrouter" || got.ProviderModelID != "typesafe/jev-router" {
			t.Fatalf("%s resolves to %+v", id, got)
		}
		if got.InputPricePer1M != 1.25 || got.OutputPricePer1M != 4.25 || got.ContextWindow != 1000000 {
			t.Fatalf("%s pricing/context = %v/%v/%v", id, got.InputPricePer1M, got.OutputPricePer1M, got.ContextWindow)
		}
	}
}
