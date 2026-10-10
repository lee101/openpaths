package alibaba

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// WAN_LIVE=1 go test ./internal/provider/alibaba -run TestLiveWan3 -v -timeout 40m
func TestLiveWan3(t *testing.T) {
	if os.Getenv("WAN_LIVE") != "1" {
		t.Skip("set WAN_LIVE=1 to hit Alibaba Model Studio")
	}
	baseURL := os.Getenv("ALIBABA_BASE_URL")
	if baseURL == "" {
		baseURL = "https://ws-rvo01jxeg9yitdn7.ap-southeast-1.maas.aliyuncs.com"
	}
	p := New(os.Getenv("DASHSCOPE_API_KEY"), baseURL)
	for name, req := range map[string]*model.VideoGenerationRequest{
		"text": {Model: "wan3.0-video", Prompt: "A kitten running across a rooftop under the moonlight, city neon lights flickering in the distance, cinematic quality, smooth camera movement.", Resolution: "720p", AspectRatio: "16:9", Duration: "5"},
		"first_frame": {Model: "wan3.0-video", Prompt: "Slow cinematic push-in as the perfume bottle rotates once on the pedestal, soft studio highlights sweeping across the glass, gentle particles drifting behind, no added text.", ImageURL: "https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg", Resolution: "480p", Duration: "5"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			start := time.Now()
			resp, err := p.GenerateVideo(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s url=%s seconds=%.1f resolution=%s took=%s", name, resp.VideoURL, resp.DurationSeconds, req.Resolution, time.Since(start).Round(time.Second))
		})
	}
}

// WAN_LIVE=1 go test ./internal/provider/alibaba -run TestLiveDecision -v
func TestLiveDecision(t *testing.T) {
	if os.Getenv("WAN_LIVE") != "1" {
		t.Skip("set WAN_LIVE=1 to hit Alibaba Model Studio")
	}
	p := New(os.Getenv("DASHSCOPE_API_KEY"), "https://ws-rvo01jxeg9yitdn7.ap-southeast-1.maas.aliyuncs.com")
	req := &model.DecisionRequest{
		Model: "decision-model-preview",
		State: json.RawMessage(`{"channel":"email","message":"I was charged twice for order #4412 and the app keeps crashing when I open my receipts. Please fix this today."}`),
		Questions: map[string]json.RawMessage{
			"refund":  json.RawMessage(`{"type":"noul","instructions":"Is the customer asking for money back?"}`),
			"queue":   json.RawMessage(`{"type":"choice","instructions":"Which team should own this ticket?","criteria":{"billing":"payments, refunds, invoices","engineering":"bugs and crashes","sales":"new purchases"}}`),
			"urgency": json.RawMessage(`{"type":"score","instructions":"How urgent is this ticket?","criteria":["can wait a week","handle this week","handle today"]}`),
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	resp, err := p.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("model=%s usage=%+v answers=%s", resp.Model, resp.Usage, resp.Answers)
}
