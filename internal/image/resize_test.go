package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want Size
		ok   bool
	}{
		{"1024x768", Size{1024, 768}, true},
		{"512x512", Size{512, 512}, true},
		{"1920X1080", Size{1920, 1080}, true},
		{"bad", Size{}, false},
		{"0x100", Size{}, false},
		{"-1x100", Size{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseSize(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseSize(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestMatchSize(t *testing.T) {
	supported := []Size{
		{1024, 1024}, {1024, 768}, {768, 1024},
		{1024, 576}, {576, 1024}, {1280, 720}, {720, 1280},
	}

	tests := []struct {
		req  Size
		want Size
	}{
		{Size{1024, 1024}, Size{1024, 1024}},       // exact match
		{Size{1920, 1080}, Size{1280, 720}},         // 16:9 -> 16:9
		{Size{800, 600}, Size{1024, 768}},           // 4:3 -> 4:3
		{Size{1080, 1920}, Size{720, 1280}},         // ~9:16 -> closest 9:16 by pixel count
		{Size{500, 500}, Size{1024, 1024}},          // 1:1 -> 1:1
		{Size{1920, 1000}, Size{1280, 720}},         // 1.92 closest to 16:9=1.78
	}
	for _, tt := range tests {
		got := MatchSize(tt.req, supported)
		if got != tt.want {
			t.Errorf("MatchSize(%v) = %v; want %v", tt.req, got, tt.want)
		}
	}
}

func TestParseAspectRatio(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"16:9", 16.0 / 9.0, true},
		{"1:1", 1, true},
		{"19.5:9", 19.5 / 9.0, true},
		{" 9 : 19.5 ", 9.0 / 19.5, true},
		{"1024x768", 0, false},
		{"auto", 0, false},
		{"0:9", 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseAspectRatio(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("ParseAspectRatio(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestIsAspectRatioList(t *testing.T) {
	tests := []struct {
		in   []string
		want bool
	}{
		{[]string{"1:1", "16:9", "auto"}, true},
		{[]string{"1024x1024", "1360x768"}, false},
		{[]string{"auto", "16:9"}, true},
		{nil, false},
	}
	for _, tt := range tests {
		if got := IsAspectRatioList(tt.in); got != tt.want {
			t.Errorf("IsAspectRatioList(%v) = %v; want %v", tt.in, got, tt.want)
		}
	}
}

func TestMatchAspectRatio(t *testing.T) {
	// grok-imagine supported aspect ratios.
	supported := []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "19.5:9", "9:19.5", "auto"}
	tests := []struct {
		req  Size
		want string
	}{
		{Size{1360, 768}, "16:9"},  // 1.77 -> 16:9 (the reported 85:48 bug)
		{Size{768, 1360}, "9:16"},  // 0.56 -> 9:16
		{Size{1024, 1024}, "1:1"},  // 1.0  -> 1:1
		{Size{1024, 768}, "4:3"},   // 1.33 -> 4:3
		{Size{1200, 800}, "3:2"},   // 1.5  -> 3:2
		{Size{2000, 900}, "19.5:9"}, // 2.22 -> 19.5:9
	}
	for _, tt := range tests {
		got, ok := MatchAspectRatio(tt.req, supported)
		if !ok || got != tt.want {
			t.Errorf("MatchAspectRatio(%v) = %q, %v; want %q", tt.req, got, ok, tt.want)
		}
	}
}

func TestMatchAspectRatioNoRatios(t *testing.T) {
	if _, ok := MatchAspectRatio(Size{1024, 768}, []string{"auto", "1024x768"}); ok {
		t.Errorf("MatchAspectRatio with no ratio entries should return ok=false")
	}
}

func TestMatchSizeEmptySupported(t *testing.T) {
	got := MatchSize(Size{800, 600}, nil)
	if got != (Size{800, 600}) {
		t.Errorf("MatchSize with nil supported = %v; want {800 600}", got)
	}
}

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestResizeAndCrop(t *testing.T) {
	src := makePNG(1024, 576) // 16:9
	target := Size{1920, 1080}

	result, format, err := ResizeAndCrop(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" {
		t.Errorf("format = %q; want png", format)
	}

	decoded, _, err := image.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatal(err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != 1920 || bounds.Dy() != 1080 {
		t.Errorf("result size = %dx%d; want 1920x1080", bounds.Dx(), bounds.Dy())
	}
}

func TestResizeAndCropWithCrop(t *testing.T) {
	src := makePNG(1024, 768) // 4:3
	target := Size{1920, 1000}

	result, _, err := ResizeAndCrop(src, target)
	if err != nil {
		t.Fatal(err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatal(err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != 1920 || bounds.Dy() != 1000 {
		t.Errorf("result size = %dx%d; want 1920x1000", bounds.Dx(), bounds.Dy())
	}
}
