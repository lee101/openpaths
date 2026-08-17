package config

import "testing"

func TestGPTTranscribeModel(t *testing.T) {
	models := loadAuditConfig(t)
	transcribe := models["gpt-transcribe"]
	if transcribe == nil {
		t.Fatal("gpt-transcribe missing from config.yaml")
	}
	if transcribe.Provider != "openai" || transcribe.ProviderModelID != "gpt-transcribe" {
		t.Fatalf("unexpected route: %+v", transcribe)
	}
	if transcribe.PricePerMinute != 0.0045 {
		t.Fatalf("price per minute = %v", transcribe.PricePerMinute)
	}
}
