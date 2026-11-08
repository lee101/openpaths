package image

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestToWebP(t *testing.T) {
	if cwebp() == "" {
		t.Skip("cwebp not installed")
	}
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			img.Set(x, y, color.RGBA{uint8(x / 2), uint8(y / 2), uint8((x + y) / 4), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, ok := ToWebP(context.Background(), buf.Bytes())
	if !ok {
		t.Fatal("expected conversion")
	}
	if string(out[:4]) != "RIFF" || string(out[8:12]) != "WEBP" {
		t.Fatalf("not webp: %q", out[:12])
	}
	if len(out) >= buf.Len() {
		t.Fatalf("webp %d not smaller than png %d", len(out), buf.Len())
	}
	if _, ok := ToWebP(context.Background(), out); ok {
		t.Fatal("webp input must not be reconverted")
	}
	if _, ok := ToWebP(context.Background(), []byte("junk")); ok {
		t.Fatal("junk must not convert")
	}
}

func TestWantsWebP(t *testing.T) {
	for in, want := range map[string]bool{"": true, "webp": true, "png": false, "jpeg": false} {
		if WantsWebP(in) != want {
			t.Fatalf("%q", in)
		}
	}
}
