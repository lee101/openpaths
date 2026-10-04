package model

import (
	"encoding/json"
	"testing"
)

func TestImageRequestAcceptsNativeAndSharedImages(t *testing.T) {
	for _, input := range []string{
		`{"model":"flux-3-image","prompt":"edit","images":"https://x/a.png","safety_tolerance":2}`,
		`{"model":"flux-3-image","prompt":"edit","images":["https://x/a.png"],"safety_tolerance":"2"}`,
		`{"model":"other","prompt":"edit","images":[{"url":"https://x/a.png","type":"image_url"}],"safety_tolerance":"2"}`,
	} {
		var req ImageGenerationRequest
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		if req.Prompt != "edit" || len(req.Images) != 1 || req.Images[0].URL != "https://x/a.png" || req.SafetyTolerance != "2" {
			t.Fatalf("decoded %#v", req)
		}
	}
}
