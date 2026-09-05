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

	image := models["muse-image-1.0"]
	if image == nil || image.Provider != "meta" || image.PricePerImage != 0.01 {
		t.Fatalf("Muse Image config = %+v", image)
	}

	voice := models["muse-voice-transcribe-1.0"]
	if voice == nil || voice.Provider != "meta" || voice.PricePerHour != 0.18 {
		t.Fatalf("Muse Voice config = %+v", voice)
	}
}
