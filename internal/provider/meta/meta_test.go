package meta

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestNormalizeBaseURLs(t *testing.T) {
	for _, tc := range []struct {
		input, root, api string
	}{
		{"", "https://api.meta.ai", "https://api.meta.ai/v1"},
		{"https://api.meta.ai/v1/", "https://api.meta.ai", "https://api.meta.ai/v1"},
		{"https://example.test", "https://example.test", "https://example.test/v1"},
	} {
		root, api := normalizeBaseURLs(tc.input)
		if root != tc.root || api != tc.api {
			t.Errorf("normalizeBaseURLs(%q) = %q, %q; want %q, %q", tc.input, root, api, tc.root, tc.api)
		}
	}
}

func TestProviderUsesNormalizedChatAndImagePaths(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		if got := r.Header.Get("Authorization"); got != "Bearer meta-test-key" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/chat/completions":
			_, _ = io.WriteString(w, `{"id":"chatcmpl-meta","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`)
		case "/v1/images/generations":
			_, _ = io.WriteString(w, `{"created":1,"data":[{"b64_json":"aGk="}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := New("meta-test-key", server.URL+"/v1")
	chat, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "muse-spark-1.3", Messages: []model.ChatMessage{{Role: "user", Content: "Say hi"}},
	})
	if err != nil || len(chat.Choices) != 1 {
		t.Fatalf("ChatCompletion = %#v, %v", chat, err)
	}
	image, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "muse-image-1.0", Prompt: "hi", N: 1,
	})
	if err != nil || len(image.Data) != 1 {
		t.Fatalf("GenerateImage = %#v, %v", image, err)
	}
	for _, path := range []string{"/v1/chat/completions", "/v1/images/generations"} {
		if !seen[path] {
			t.Errorf("request path %q was not called; seen %#v", path, seen)
		}
	}
}

func TestTranscribeUsesMetaMultipartContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/asr/transcribe" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		parts := map[string][]byte{}
		contentTypes := map[string]string{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			parts[part.FormName()], _ = io.ReadAll(part)
			contentTypes[part.FormName()] = part.Header.Get("Content-Type")
		}
		var request transcriptionRequest
		if err := json.Unmarshal(parts["request"], &request); err != nil {
			t.Fatalf("request part = %q: %v", parts["request"], err)
		}
		if request.Model != defaultTranscriptionModel || request.AudioEncoding != "WAV" || request.Language != "en" || request.Prompt != "OpenPaths" {
			t.Errorf("request = %#v", request)
		}
		if contentTypes["request"] != "application/json" || contentTypes["audio"] != "audio/wav" {
			t.Errorf("content types = %#v", contentTypes)
		}
		if string(parts["audio"]) != "RIFFxxxxWAVEpayload" {
			t.Errorf("audio part = %q", parts["audio"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"transcript":"Hi. Nothing else.","audioDurationMs":2160,"turns":[{"speaker":"A","startMs":0,"endMs":2160}]}`)
	}))
	defer server.Close()

	p := New("meta-test-key", server.URL+"/v1")
	resp, err := p.Transcribe(context.Background(), &model.TranscriptionRequest{
		File: []byte("RIFFxxxxWAVEpayload"), Filename: `../bad\"name.wav`, Model: defaultTranscriptionModel,
		Language: "en", Prompt: "OpenPaths",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hi. Nothing else." || resp.AudioDurationMS != 2160 || len(resp.Turns) == 0 {
		t.Fatalf("response = %#v", resp)
	}
}

func TestAudioEncodingRejectsUnknownData(t *testing.T) {
	if _, _, err := audioEncoding("recording.bin", []byte("not audio")); err == nil {
		t.Fatal("expected unsupported format error")
	}
}
