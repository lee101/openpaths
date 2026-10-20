package fal

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestFlux3MaxVideoRouting(t *testing.T) {
	cases := []struct {
		name string
		req  model.VideoGenerationRequest
		want string
	}{
		{"text", model.VideoGenerationRequest{Prompt: "p", Duration: "auto", Resolution: "HD"}, "blackforestlabs/flux-3-max/text-to-video"},
		{"image", model.VideoGenerationRequest{Prompt: "p", ImageURL: "https://x/a.png", EndImageURL: "https://x/b.png"}, "blackforestlabs/flux-3-max/image-to-video"},
		{"keyframes", model.VideoGenerationRequest{Prompt: "p", Duration: "10", ImageURLs: []string{"https://x/a.png", "https://x/b.png", "https://x/c.png"}}, "blackforestlabs/flux-3-max/keyframes-to-video"},
	}
	for _, c := range cases {
		falReq := map[string]any{"prompt": "p", "fps": 24, "image_urls": c.req.ImageURLs}
		got := applyFlux3MaxVideoRequest(&c.req, falReq)
		if got != c.want {
			t.Errorf("%s: endpoint %s want %s", c.name, got, c.want)
		}
		if _, ok := falReq["fps"]; ok {
			t.Errorf("%s: fps leaked", c.name)
		}
		if _, ok := falReq["image_urls"]; ok {
			t.Errorf("%s: image_urls leaked", c.name)
		}
		if falReq["duration"].(int) < 5 {
			t.Errorf("%s: duration", c.name)
		}
	}
	req := model.VideoGenerationRequest{Prompt: "p", Duration: "10", Resolution: "FHD", ImageURLs: []string{"a", "b", "c"}}
	falReq := map[string]any{}
	applyFlux3MaxVideoRequest(&req, falReq)
	kf := falReq["keyframes"].([]map[string]any)
	if falReq["resolution"] != "1080p" || kf[0]["frame_index"] != 0 || kf[1]["frame_index"] != 120 || kf[2]["frame_index"] != 240 {
		t.Errorf("bad payload %v", falReq)
	}
}
