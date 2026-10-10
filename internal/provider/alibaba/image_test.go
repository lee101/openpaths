package alibaba

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestQwenImagePayload(t *testing.T) {
	payload, err := qwenImagePayload(&model.ImageGenerationRequest{Model: "qwen-image-3.0", Prompt: "a fox", AspectRatio: "16:9", N: 9})
	if err != nil {
		t.Fatal(err)
	}
	params := payload["parameters"].(map[string]any)
	if params["size"] != "1376*768" || params["n"] != 6 || params["watermark"] != false {
		t.Fatalf("params %#v", params)
	}
	edit, err := qwenImagePayload(&model.ImageGenerationRequest{Model: "qwen-image-3.0", Prompt: "make it blue", ImageURL: "https://x/a.png", ImageURLs: []string{"https://x/b.png"}})
	if err != nil {
		t.Fatal(err)
	}
	content := edit["input"].(map[string]any)["messages"].([]map[string]any)[0]["content"].([]map[string]string)
	if len(content) != 3 || content[0]["image"] != "https://x/a.png" || content[2]["text"] != "make it blue" {
		t.Fatalf("content %#v", content)
	}
	if _, ok := edit["parameters"].(map[string]any)["size"]; ok {
		t.Fatal("edits without a size keep the input aspect")
	}
	for _, bad := range []*model.ImageGenerationRequest{
		{Model: "qwen-image-3.0"},
		{Model: "qwen-image-3.0", Prompt: "x", ImageURLs: []string{"1", "2", "3", "4"}},
		{Model: "qwen-image-3.0", Prompt: "x", Size: "100x100"},
	} {
		if _, err := qwenImagePayload(bad); err == nil {
			t.Fatalf("expected error for %+v", bad)
		}
	}
	sized, _ := qwenImagePayload(&model.ImageGenerationRequest{Model: "m", Prompt: "x", Size: "1536x1024"})
	if sized["parameters"].(map[string]any)["size"] != "1536*1024" {
		t.Fatal("WxH must become W*H")
	}
}

func TestGenerateImageBillsDeliveredTier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/services/aigc/multimodal-generation/generation" || r.Header.Get("X-DashScope-Async") != "" {
			t.Errorf("path %s async %q", r.URL.Path, r.Header.Get("X-DashScope-Async"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"qwen-image-3.0-pro"`) {
			t.Errorf("body %s", body)
		}
		io.WriteString(w, `{"output":{"choices":[{"message":{"content":[{"type":"image","image":"https://oss/a.png"}]}}]},"usage":{"output_width":2720,"output_height":1536,"output_image_type":"qima_output_2k"}}`)
	}))
	defer srv.Close()
	req := &model.ImageGenerationRequest{Model: "qwen-image-3.0-pro", Prompt: "x", ImageURL: "https://x/a.png"}
	resp, err := New("key", srv.URL).GenerateImage(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://oss/a.png" || resp.Data[0].Width != 2720 || req.Resolution != "2k" {
		b, _ := json.Marshal(resp)
		t.Fatalf("resp %s resolution %q", b, req.Resolution)
	}
}
