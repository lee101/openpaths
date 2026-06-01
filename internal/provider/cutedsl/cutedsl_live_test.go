package cutedsl

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// These tests hit the real https://cutedsl.cc/api/service endpoint to pin down
// each service's response shape before deploy. Guarded behind RUN_LIVE_CUTEDSL=1.
//
//	RUN_LIVE_CUTEDSL=1 CUTEDSL_API_KEY=... go test ./internal/provider/cutedsl -run TestLiveCuteDSL -v -timeout 8m
func liveKey(t *testing.T) string {
	t.Helper()
	if os.Getenv("RUN_LIVE_CUTEDSL") != "1" {
		t.Skip("set RUN_LIVE_CUTEDSL=1 to run the live CuteDSL integration tests")
	}
	key := os.Getenv("CUTEDSL_API_KEY")
	if key == "" {
		t.Fatal("CUTEDSL_API_KEY is required for the live CuteDSL tests")
	}
	return key
}

func TestLiveCuteDSLImage(t *testing.T) {
	p := New(liveKey(t), "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	resp, err := p.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:  "z-image-turbo",
		Prompt: "a cute fairy in a glowing forest, soft light",
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if len(resp.Data) == 0 || (resp.Data[0].URL == "" && resp.Data[0].B64JSON == "") {
		t.Fatalf("no image returned: %#v", resp)
	}
	t.Logf("image url=%q b64len=%d", resp.Data[0].URL, len(resp.Data[0].B64JSON))
}

func TestLiveCuteDSLChronos2(t *testing.T) {
	p := New(liveKey(t), "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// A simple ramp so we can eyeball whether the forecast continues the trend.
	series := make([]float64, 0, 48)
	for i := 0; i < 48; i++ {
		series = append(series, float64(i)+float64(i%7))
	}

	resp, err := p.GenerateForecast(ctx, &model.ForecastingRequest{
		Model:            "chronos2",
		Context:          series,
		PredictionLength: 12,
		Quantiles:        []float64{0.1, 0.5, 0.9},
	})
	if err != nil {
		t.Fatalf("GenerateForecast() error = %v", err)
	}
	if len(resp.Forecast) == 0 {
		t.Fatalf("empty forecast: %#v", resp)
	}
	t.Logf("forecast (%d steps) = %v; quantiles=%v", len(resp.Forecast), resp.Forecast, resp.Quantiles)
}

// TestLiveCuteDSLOtherServices probes the remaining CuteDSL services so we can
// see their raw result envelopes and decide how to wire them as billable models.
// Failures here are logged, not fatal — the point is exploration.
func TestLiveCuteDSLOtherServices(t *testing.T) {
	p := New(liveKey(t), "")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	probes := []struct {
		name    string
		service string
		params  map[string]any
	}{
		{"flux-image", "flux-image", map[string]any{"prompt": "an astronaut riding a horse", "width": 1024, "height": 1024}},
		{"kokoro-tts", "kokoro-tts", map[string]any{"text": "Hello from CuteDSL", "voice": "default"}},
		{"speech-to-text", "speech-to-text", map[string]any{"audio_url": "https://cutedsl.cc/sample.wav"}},
		{"gemma4-chat", "gemma4-chat", map[string]any{"prompt": "Say hi in five words."}},
		{"image-caption", "image-caption", map[string]any{"image_url": "https://appstatic.app.nz/cutedsl/static/images/logo.webp"}},
	}

	for _, pr := range probes {
		t.Run(pr.name, func(t *testing.T) {
			env, err := p.callService(ctx, pr.service, pr.params)
			if err != nil {
				t.Logf("%s: error = %v", pr.service, err)
				return
			}
			pretty, _ := json.MarshalIndent(env, "", "  ")
			t.Logf("%s OK: %s", pr.service, string(pretty))
		})
	}
}
