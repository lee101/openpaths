package google

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestBrowserReadySpeechAudioWrapsPCMAsWAV(t *testing.T) {
	pcm := []byte{0x00, 0x00, 0xff, 0x7f}
	got, mimeType := browserReadySpeechAudio(pcm, "audio/L16;codec=pcm;rate=24000")
	if mimeType != "audio/wav" {
		t.Fatalf("mime type = %q, want audio/wav", mimeType)
	}
	if string(got[:4]) != "RIFF" || string(got[8:12]) != "WAVE" {
		t.Fatalf("missing WAV header: %q / %q", got[:4], got[8:12])
	}
	if rate := binary.LittleEndian.Uint32(got[24:28]); rate != 24000 {
		t.Fatalf("sample rate = %d, want 24000", rate)
	}
	if size := binary.LittleEndian.Uint32(got[40:44]); size != uint32(len(pcm)) {
		t.Fatalf("data size = %d, want %d", size, len(pcm))
	}
	if string(got[44:]) != string(pcm) {
		t.Fatal("PCM payload changed")
	}
}

func TestBrowserReadySpeechAudioLeavesEncodedAudioAlone(t *testing.T) {
	mp3 := []byte("encoded")
	got, mimeType := browserReadySpeechAudio(mp3, "audio/mpeg")
	if mimeType != "audio/mpeg" || string(got) != string(mp3) {
		t.Fatalf("encoded audio changed: mime=%q data=%q", mimeType, got)
	}
}

func TestEncodeLyriaOutputAsOpus(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	pcm := make([]byte, 2400)
	wav, _ := browserReadySpeechAudio(pcm, "audio/pcm")
	got, mimeType, format, err := encodeLyriaOutput(context.Background(), wav, "audio/wav", "opus")
	if err != nil {
		t.Fatalf("encodeLyriaOutput() error = %v", err)
	}
	if mimeType != "audio/ogg;codecs=opus" || format != "opus" {
		t.Fatalf("mime=%q format=%q, want Opus", mimeType, format)
	}
	if len(got) < 4 || string(got[:4]) != "OggS" {
		t.Fatalf("missing Ogg container header: %q", got[:min(4, len(got))])
	}
}

func TestTranslateRequest_PrefillAppendsModelTurn(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "gemini-3-pro-preview",
		Messages: []model.ChatMessage{{Role: "user", Content: "Return JSON"}},
		Prefill:  "{",
	}

	gemReq := translateRequest(req)
	if len(gemReq.Contents) != 2 {
		t.Fatalf("got %d contents, want 2 (user + model prefill)", len(gemReq.Contents))
	}
	last := gemReq.Contents[len(gemReq.Contents)-1]
	if last.Role != "model" {
		t.Errorf("last role = %q, want model", last.Role)
	}
	if len(last.Parts) != 1 || last.Parts[0].Text != "{" {
		t.Errorf("last parts = %+v, want text={", last.Parts)
	}
}

func TestTranslateRequest_ClampsThinkingBudgetToMaxOutput(t *testing.T) {
	maxTokens := 128
	req := &model.ChatCompletionRequest{
		Model:           "gemini-2.5-flash",
		ReasoningEffort: "high",
		MaxTokens:       &maxTokens,
		Messages:        []model.ChatMessage{{Role: "user", Content: "Solve this carefully."}},
	}

	gemReq := translateRequest(req)
	if gemReq.GenerationConfig == nil || gemReq.GenerationConfig.ThinkingConfig == nil || gemReq.GenerationConfig.ThinkingConfig.ThinkingBudget == nil {
		t.Fatal("expected thinking config to be set")
	}
	if got := *gemReq.GenerationConfig.ThinkingConfig.ThinkingBudget; got != 128 {
		t.Fatalf("thinking budget = %d, want %d", got, 128)
	}
}

func TestTranslateRequest_Gemini35UsesThinkingLevel(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:           "gemini-3.5-flash",
		ReasoningEffort: "medium",
		Messages:        []model.ChatMessage{{Role: "user", Content: "Solve this carefully."}},
	}

	gemReq := translateRequest(req)
	if gemReq.GenerationConfig == nil || gemReq.GenerationConfig.ThinkingConfig == nil {
		t.Fatal("expected thinking config to be set")
	}
	if got := gemReq.GenerationConfig.ThinkingConfig.ThinkingLevel; got != "MEDIUM" {
		t.Fatalf("thinking level = %q, want MEDIUM", got)
	}
	if gemReq.GenerationConfig.ThinkingConfig.ThinkingBudget != nil {
		t.Fatalf("thinking budget should be omitted for Gemini 3.5 Flash")
	}
}

func TestTranslateRequest_MapsOpenAIImageURLBlocks(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model: "gemini-2.5-flash",
		Messages: []model.ChatMessage{{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "Describe this image."},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/cat.png"}},
			},
		}},
	}

	gemReq := translateRequest(req)
	if len(gemReq.Contents) != 1 {
		t.Fatalf("got %d contents, want 1", len(gemReq.Contents))
	}
	parts := gemReq.Contents[0].Parts
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	if parts[0].Text != "Describe this image." {
		t.Fatalf("text part = %q, want prompt text", parts[0].Text)
	}
	if parts[1].FileData == nil || parts[1].FileData.FileURI != "https://example.com/cat.png" {
		t.Fatalf("image part = %#v, want fileData URI", parts[1])
	}
}

func TestTranslateUsage_CountsThoughtTokensAsBillableOutput(t *testing.T) {
	usage := translateUsage(&geminiUsage{
		PromptTokenCount:     100,
		CandidatesTokenCount: 40,
		ThoughtsTokenCount:   60,
		TotalTokenCount:      200,
	})

	if usage.PromptTokens != 100 {
		t.Fatalf("PromptTokens = %d, want %d", usage.PromptTokens, 100)
	}
	if usage.CompletionTokens != 100 {
		t.Fatalf("CompletionTokens = %d, want %d", usage.CompletionTokens, 100)
	}
	if usage.TotalTokens != 200 {
		t.Fatalf("TotalTokens = %d, want %d", usage.TotalTokens, 200)
	}
}

func TestEmbed_BatchEmbeddings(t *testing.T) {
	var captured geminiBatchEmbedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/models/gemini-embedding-001:batchEmbedContents" {
			t.Fatalf("path = %s, want batch embed endpoint", r.URL.Path)
		}
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Fatalf("api key = %q, want test-key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"embeddings": []map[string]any{
				{"values": []float64{0.1, 0.2}},
				{"values": []float64{0.3, 0.4}},
			},
		})
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	dims := 768
	resp, err := p.Embed(context.Background(), &model.EmbeddingRequest{
		Model:      "gemini-embedding-001",
		Input:      []string{"hello", "world"},
		Dimensions: dims,
	})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(captured.Requests) != 2 {
		t.Fatalf("captured %d requests, want 2", len(captured.Requests))
	}
	if captured.Requests[0].Model != "gemini-embedding-001" {
		t.Fatalf("request model = %q", captured.Requests[0].Model)
	}
	if captured.Requests[0].OutputDimensionality == nil || *captured.Requests[0].OutputDimensionality != dims {
		t.Fatalf("output dimensionality = %v, want %d", captured.Requests[0].OutputDimensionality, dims)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("got %d embeddings, want 2", len(resp.Data))
	}
	if resp.Usage.TotalTokens == 0 {
		t.Fatal("expected non-zero token accounting")
	}
}
