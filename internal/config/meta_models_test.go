package config

import "testing"

func TestMetaModels(t *testing.T) {
	models := loadAuditConfig(t)

	spark := models["muse-spark-1.3"]
	if spark == nil || spark.Provider != "meta" || spark.ProviderModelID != "muse-spark-1.3" {
		t.Fatalf("Muse Spark route = %+v", spark)
	}
	if spark.InputPricePer1M != 1.25 || spark.InputCacheHitPricePer1M != 0.15 || spark.OutputPricePer1M != 4.25 {
		t.Fatalf("Muse Spark pricing = %v/%v/%v", spark.InputPricePer1M, spark.InputCacheHitPricePer1M, spark.OutputPricePer1M)
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
