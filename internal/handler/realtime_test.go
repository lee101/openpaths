package handler

import (
	"testing"
)

func TestParseRealtimeUsage(t *testing.T) {
	payload := []byte(`{
  "type":"response.done",
  "response":{"id":"resp_1","usage":{
    "input_tokens":3100,"output_tokens":3500,
    "input_token_details":{"text_tokens":1000,"audio_tokens":2000,"image_tokens":100,"cached_tokens":700,
      "cached_tokens_details":{"text_tokens":400,"audio_tokens":250,"image_tokens":50}},
    "output_token_details":{"text_tokens":500,"audio_tokens":3000}
  }}
}`)
	id, usage, completed, err := parseRealtimeUsage(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !completed || id != "resp_1" {
		t.Fatalf("completed = %v, id = %q", completed, id)
	}
	if usage.TextInputTokens != 1000 || usage.CachedTextInputTokens != 400 ||
		usage.AudioInputTokens != 2000 || usage.CachedAudioInputTokens != 250 ||
		usage.ImageInputTokens != 100 || usage.CachedImageInputTokens != 50 ||
		usage.TextOutputTokens != 500 || usage.AudioOutputTokens != 3000 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestParseRealtimeUsageFallbacks(t *testing.T) {
	payload := []byte(`{"type":"response.done","response":{"id":"resp_2","usage":{"input_tokens":100,"output_tokens":20,"input_token_details":{"cached_tokens":30},"output_token_details":{}}}}`)
	_, usage, completed, err := parseRealtimeUsage(payload)
	if err != nil || !completed {
		t.Fatalf("completed = %v, err = %v", completed, err)
	}
	if usage.TextInputTokens != 100 || usage.CachedTextInputTokens != 30 || usage.TextOutputTokens != 20 {
		t.Fatalf("unexpected fallback usage: %+v", usage)
	}
}

func TestParseRealtimeUsageRejectsMissingUsage(t *testing.T) {
	_, _, _, err := parseRealtimeUsage([]byte(`{"type":"response.done","response":{"id":"resp_3"}}`))
	if err == nil {
		t.Fatal("expected missing usage to fail")
	}
}

func TestMakeRealtimeURL(t *testing.T) {
	got, err := makeRealtimeURL("https://api.openai.com", "gpt-realtime-2.1-mini")
	if err != nil {
		t.Fatal(err)
	}
	want := "wss://api.openai.com/v1/realtime?model=gpt-realtime-2.1-mini"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}
