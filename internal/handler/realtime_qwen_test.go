package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestParseQwenRealtimeUsage(t *testing.T) {
	payload := []byte(`{"type":"response.done","response":{"id":"r1","usage":{"total_tokens":1035,"input_tokens":840,"output_tokens":195,"input_tokens_details":{"text_tokens":762,"audio_tokens":14,"video_tokens":64},"output_tokens_details":{"text_tokens":36,"audio_tokens":159}}}}`)
	id, usage, ok, err := parseRealtimeUsage(payload)
	if err != nil || !ok || id != "r1" {
		t.Fatalf("id %q ok %v err %v", id, ok, err)
	}
	if usage.TextInputTokens != 762 || usage.AudioInputTokens != 14 || usage.ImageInputTokens != 64 || usage.TextOutputTokens != 36 || usage.AudioOutputTokens != 159 {
		t.Fatalf("usage %+v", usage)
	}
}

func TestAlibabaRealtimeRouting(t *testing.T) {
	for id, want := range map[string]bool{"qwen3.8-omni-flash-realtime": true, "qwen-audio-3.1-realtime-plus": true, "qwen3.8-livetranslate-flash-realtime": true, "wan3.0-video": false} {
		if got := supportedRealtimeModel(&model.ModelConfig{Provider: "alibaba", ProviderModelID: id}); got != want {
			t.Errorf("%s: %v", id, got)
		}
	}
	u, err := makeAlibabaRealtimeURL("https://ws-x.ap-southeast-1.maas.aliyuncs.com", "qwen3.8-omni-flash-realtime")
	if err != nil || u != "wss://ws-x.ap-southeast-1.maas.aliyuncs.com/api-ws/v1/realtime?model=qwen3.8-omni-flash-realtime" {
		t.Fatalf("url %q err %v", u, err)
	}
}

func TestRealtimeCancelledWithoutUsageIsNotAnError(t *testing.T) {
	_, _, present, err := parseRealtimeUsage([]byte(`{"type":"response.done","response":{"id":"r2","status":"cancelled","status_details":{"reason":"turn_detected"}}}`))
	if err != nil || present {
		t.Fatalf("present %v err %v", present, err)
	}
	if _, _, _, err := parseRealtimeUsage([]byte(`{"type":"response.done","response":{"id":"r3","status":"completed"}}`)); err == nil {
		t.Fatal("completed responses must still report usage")
	}
}
