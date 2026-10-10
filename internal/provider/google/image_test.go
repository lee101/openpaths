package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestGenerateImagePostsInteractionAndParsesImage(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/interactions" || r.URL.Query().Get("key") != "k" {
			t.Fatalf("bad request %s", r.URL.String())
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"steps":[{"type":"thought"},{"type":"model_output","content":[{"type":"text","text":"done"},{"type":"image","data":"QUJD","mime_type":"image/jpeg"}]}]}`))
	}))
	defer srv.Close()
	p := New("k", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:       "gemini-3.1-flash-lite-image",
		Prompt:      "a fort",
		AspectRatio: "16:9",
		Resolution:  "4k",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].B64JSON != "QUJD" || resp.Data[0].RevisedPrompt != "done" {
		t.Fatalf("resp = %#v", resp)
	}
	if got["model"] != "models/gemini-3.1-flash-lite-image" || got["input"] != "a fort" {
		t.Fatalf("req = %#v", got)
	}
	cfg := got["generation_config"].(map[string]any)
	ic := cfg["image_config"].(map[string]any)
	if cfg["thinking_level"] != "minimal" || ic["image_size"] != "1K" || ic["aspect_ratio"] != "16:9" {
		t.Fatalf("generation_config = %#v", cfg)
	}
}

func TestGenerateImageEditSendsDataURLInput(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"steps":[{"type":"model_output","content":[{"type":"image","data":"QUJD"}]}]}`))
	}))
	defer srv.Close()
	_, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "models/gemini-nano-banana-2.1", Prompt: "polish", Images: []model.ImageInput{{URL: "data:image/png;base64,AAAA"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := got["input"].([]any)
	img := parts[1].(map[string]any)
	if len(parts) != 2 || img["type"] != "image" || img["mime_type"] != "image/png" || img["data"] != "AAAA" || got["model"] != "models/gemini-nano-banana-2.1" {
		t.Fatalf("input = %#v", got)
	}
}

func TestGenerateImageNoImageIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"steps":[{"type":"model_output","content":[{"type":"text","text":"blocked"}]}]}`))
	}))
	defer srv.Close()
	_, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	if err == nil {
		t.Fatal("want error")
	}
}

func imageOK(data string) string {
	return `{"steps":[{"type":"model_output","content":[{"type":"image","data":"` + data + `"}]}]}`
}

func TestGenerateImageMultipleImagesFansOut(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(imageOK("QUJD")))
	}))
	defer srv.Close()
	for _, tc := range []struct{ n, want int }{{0, 1}, {3, 3}, {9, maxGeminiImagesPerRequest}} {
		calls.Store(0)
		resp, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x", N: tc.n})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Data) != tc.want || int(calls.Load()) != tc.want {
			t.Fatalf("n=%d: data=%d calls=%d want %d", tc.n, len(resp.Data), calls.Load(), tc.want)
		}
	}
}

func TestGenerateImagePartialFailureReturnsSuccesses(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 2 {
			http.Error(w, `{"error":{"message":"boom"}}`, http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(imageOK("QUJD")))
	}))
	defer srv.Close()
	resp, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x", N: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("data = %d, want 2", len(resp.Data))
	}
}

func TestGenerateImageAllFailReturnsProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"quota"}}`, http.StatusTooManyRequests)
	}))
	defer srv.Close()
	_, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x", N: 2})
	var pe *provider.ProviderError
	if !errors.As(err, &pe) || pe.StatusCode != http.StatusTooManyRequests || !pe.Retryable {
		t.Fatalf("err = %#v", err)
	}
}

func TestGenerateImageRejectsMissingPromptAndKey(t *testing.T) {
	var pe *provider.ProviderError
	_, err := New("k", "http://127.0.0.1:1").GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: " "})
	if !errors.As(err, &pe) || pe.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %#v", err)
	}
	_, err = New("", "http://127.0.0.1:1").GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	if !errors.As(err, &pe) || pe.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %#v", err)
	}
}

func TestGenerateImageReferenceImagesOrderedAfterPrompt(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(imageOK("QUJD")))
	}))
	defer srv.Close()
	_, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "m", Prompt: "blend",
		ImageURL:           "data:image/png;base64,AAAA",
		Images:             []model.ImageInput{{URL: "data:image/jpeg;base64,BBBB"}},
		ReferenceImageURLs: []string{"data:image/webp;base64,CCCC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	parts := got["input"].([]any)
	if len(parts) != 4 || parts[0].(map[string]any)["text"] != "blend" {
		t.Fatalf("input = %#v", parts)
	}
	want := []string{"AAAA", "BBBB", "CCCC"}
	for i, w := range want {
		if parts[i+1].(map[string]any)["data"] != w {
			t.Fatalf("part %d = %#v", i+1, parts[i+1])
		}
	}
}

func TestGenerateImageInvalidDataURLIs400(t *testing.T) {
	var pe *provider.ProviderError
	_, err := New("k", "http://127.0.0.1:1").GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x", ImageURL: "data:bogus"})
	if !errors.As(err, &pe) || pe.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %#v", err)
	}
}

func TestGeminiAspectRatioFromSize(t *testing.T) {
	for _, tc := range []struct{ size, aspect, want string }{
		{"1024x1024", "", "1:1"},
		{"1792x1024", "", "16:9"},
		{"1024x1792", "", "9:16"},
		{"1536x1024", "", "3:2"},
		{"1024x1536", "", "2:3"},
		{"1024x1792", "4:3", "4:3"},
		{"auto", "", ""},
		{"", "", ""},
	} {
		got := geminiAspectRatio(&model.ImageGenerationRequest{Size: tc.size, AspectRatio: tc.aspect})
		if got != tc.want {
			t.Errorf("size=%q aspect=%q got %q want %q", tc.size, tc.aspect, got, tc.want)
		}
	}
}

func TestGenerateImageSizeMapsToAspectRatio(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(imageOK("QUJD")))
	}))
	defer srv.Close()
	if _, err := New("k", srv.URL).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x", Size: "1792x1024"}); err != nil {
		t.Fatal(err)
	}
	ic := got["generation_config"].(map[string]any)["image_config"].(map[string]any)
	if ic["aspect_ratio"] != "16:9" {
		t.Fatalf("image_config = %#v", ic)
	}
}

func TestGenerateImageTransportErrorRedactsAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	_, err := New("sekrit-key", url).GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "m", Prompt: "x"})
	var pe *provider.ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %#v", err)
	}
	if strings.Contains(pe.Message, "sekrit-key") || strings.Contains(pe.Error(), "sekrit-key") {
		t.Fatalf("message leaks key: %q", pe.Message)
	}
}
