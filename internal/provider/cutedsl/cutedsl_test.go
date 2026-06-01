package cutedsl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateImageEnvelope(t *testing.T) {
	var gotAuth, gotService string
	var gotWidth, gotHeight float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotService, _ = body["service"].(string)
		gotWidth, _ = body["width"].(float64)
		gotHeight, _ = body["height"].(float64)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"image_url":"https://cdn.cutedsl.cc/out/abc.webp"},"credits_used":1000,"usd_equivalent":0.04}`))
	}))
	defer srv.Close()

	p := New("sk-test", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:  "z-image-turbo",
		Prompt: "a cute fairy in a forest",
		Size:   "1024x768",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q, want Bearer sk-test", gotAuth)
	}
	if gotService != "z-image-turbo" {
		t.Errorf("service = %q, want z-image-turbo", gotService)
	}
	if gotWidth != 1024 || gotHeight != 768 {
		t.Errorf("size = %vx%v, want 1024x768", gotWidth, gotHeight)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://cdn.cutedsl.cc/out/abc.webp" {
		t.Fatalf("unexpected image data: %#v", resp.Data)
	}
}

func TestGenerateForecastEnvelope(t *testing.T) {
	var gotService string
	var gotPredLen float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotService, _ = body["service"].(string)
		gotPredLen, _ = body["prediction_length"].(float64)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"forecast":[14.1,14.4,13.9],"quantiles":{"0.1":[13.0,13.1,12.8],"0.9":[15.2,15.6,15.0]}},"credits_used":500,"usd_equivalent":0.20}`))
	}))
	defer srv.Close()

	p := New("sk-test", srv.URL)
	resp, err := p.GenerateForecast(context.Background(), &model.ForecastingRequest{
		Model:            "chronos2",
		Context:          []float64{12.0, 13.5, 11.2, 14.0},
		PredictionLength: 3,
		Quantiles:        []float64{0.1, 0.9},
	})
	if err != nil {
		t.Fatalf("GenerateForecast() error = %v", err)
	}
	if gotService != "chronos2" {
		t.Errorf("service = %q, want chronos2", gotService)
	}
	if gotPredLen != 3 {
		t.Errorf("prediction_length = %v, want 3", gotPredLen)
	}
	if len(resp.Forecast) != 3 || resp.Forecast[0] != 14.1 {
		t.Fatalf("unexpected forecast: %#v", resp.Forecast)
	}
	if len(resp.Quantiles["0.9"]) != 3 {
		t.Fatalf("unexpected quantiles: %#v", resp.Quantiles)
	}
}

func TestImagesFromResultVariants(t *testing.T) {
	cases := map[string]string{
		"bare url":     `"https://x/y.png"`,
		"url field":    `{"url":"https://x/y.png"}`,
		"images array": `{"images":[{"url":"https://x/y.png"}]}`,
		"data uri":     `{"image":"data:image/png;base64,QUJD"}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			imgs := imagesFromResult(json.RawMessage(raw))
			if len(imgs) == 0 {
				t.Fatalf("no image parsed from %s", raw)
			}
		})
	}
}

func TestForecastFromResultVariants(t *testing.T) {
	cases := []string{
		`[1.0,2.0,3.0]`,
		`{"forecast":[1.0,2.0,3.0]}`,
		`{"mean":[1.0,2.0,3.0]}`,
		`{"predictions":[[1.0,2.0,3.0]]}`,
	}
	for _, raw := range cases {
		vals, _ := forecastFromResult(json.RawMessage(raw))
		if len(vals) != 3 {
			t.Fatalf("forecast parse failed for %s: %#v", raw, vals)
		}
	}
}

func TestErrorStatusMapsToProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(402)
		_, _ = w.Write([]byte(`{"error":"Insufficient $CUTEDSL credits"}`))
	}))
	defer srv.Close()

	p := New("sk-test", srv.URL)
	_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "z-image-turbo", Prompt: "x"})
	if err == nil {
		t.Fatal("expected error on 402")
	}
}
