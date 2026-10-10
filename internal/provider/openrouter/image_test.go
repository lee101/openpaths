package openrouter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestGenerateImageUsesOpenRouterUnifiedImageEndpoint(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("authorization = %q", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(model.ImageGenerationResponse{
			Data: []model.ImageData{{B64JSON: "aW1hZ2U="}},
		})
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "google/gemini-3.1-flash-image", Prompt: "fallback art", N: 1, Size: "1024x1024",
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if got["model"] != "google/gemini-3.1-flash-image" || len(resp.Data) != 1 {
		t.Fatalf("request = %#v, response = %#v", got, resp)
	}
}

func TestGenerateImageForwardsReferencesAspectAndAttribution(t *testing.T) {
	var got map[string]any
	var referer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer = r.Header.Get("HTTP-Referer")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(model.ImageGenerationResponse{Data: []model.ImageData{{B64JSON: "QQ=="}, {B64JSON: "Qg=="}}})
	}))
	defer srv.Close()
	resp, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "google/gemini-3.1-flash-image", Prompt: "edit", N: 2, AspectRatio: "16:9",
		ImageURL: "https://a.test/1.png", ReferenceImageURLs: []string{"data:image/png;base64,AAAA"},
	})
	if err != nil || len(resp.Data) != 2 {
		t.Fatalf("resp=%#v err=%v", resp, err)
	}
	refs, _ := got["input_references"].([]any)
	if got["aspect_ratio"] != "16:9" || got["n"] != float64(2) || len(refs) != 2 || referer == "" {
		t.Fatalf("req=%#v referer=%q", got, referer)
	}
	first := refs[0].(map[string]any)["image_url"].(map[string]any)["url"]
	if first != "https://a.test/1.png" {
		t.Fatalf("refs=%#v", refs)
	}
}

func TestGenerateImageErrorsAreBoundedAndTyped(t *testing.T) {
	status := http.StatusTooManyRequests
	body := strings.Repeat("x", 10000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := New("k", srv.URL)
	_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	var pe *provider.ProviderError
	if !errors.As(err, &pe) || pe.StatusCode != 429 || !pe.Retryable || len(pe.Message) > maxErrorMessageBytes {
		t.Fatalf("err=%#v", err)
	}
	status = http.StatusBadRequest
	_, err = p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	if !errors.As(err, &pe) || pe.Retryable {
		t.Fatalf("err=%#v", err)
	}
}

func TestGenerateImageEmptyResponseIsRetryableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	_, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	var pe *provider.ProviderError
	if !errors.As(err, &pe) || !pe.Retryable {
		t.Fatalf("err=%#v", err)
	}
}
