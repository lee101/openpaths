package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Size struct {
	W, H int
}

func (s Size) Aspect() float64 {
	if s.H == 0 {
		return 0
	}
	return float64(s.W) / float64(s.H)
}

func ParseSize(s string) (Size, bool) {
	parts := strings.SplitN(strings.ToLower(s), "x", 2)
	if len(parts) != 2 {
		return Size{}, false
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return Size{}, false
	}
	return Size{w, h}, true
}

func FormatSize(s Size) string {
	return fmt.Sprintf("%dx%d", s.W, s.H)
}

func ParseSizes(sizes []string) []Size {
	out := make([]Size, 0, len(sizes))
	for _, s := range sizes {
		if sz, ok := ParseSize(s); ok {
			out = append(out, sz)
		}
	}
	return out
}

// MatchSize finds the supported size with the closest aspect ratio to requested.
// Returns requested directly if it's already in the supported list.
func MatchSize(requested Size, supported []Size) Size {
	if len(supported) == 0 {
		return requested
	}
	for _, s := range supported {
		if s.W == requested.W && s.H == requested.H {
			return requested
		}
	}
	reqAspect := requested.Aspect()
	reqPixels := requested.W * requested.H
	best := supported[0]
	bestDiff := math.Abs(best.Aspect() - reqAspect)
	bestPixelDiff := abs(best.W*best.H - reqPixels)
	for _, s := range supported[1:] {
		diff := math.Abs(s.Aspect() - reqAspect)
		if diff < bestDiff || (diff == bestDiff && abs(s.W*s.H-reqPixels) < bestPixelDiff) {
			bestDiff = diff
			bestPixelDiff = abs(s.W*s.H - reqPixels)
			best = s
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func NeedsResize(requested, providerSize Size) bool {
	return requested.W != providerSize.W || requested.H != providerSize.H
}

// ResizeAndCrop scales up with nearest-neighbor to cover target, then center-crops.
func ResizeAndCrop(imgData []byte, target Size) ([]byte, string, error) {
	src, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}

	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	scaleX := float64(target.W) / float64(srcW)
	scaleY := float64(target.H) / float64(srcH)
	scale := math.Max(scaleX, scaleY)

	scaledW := int(math.Ceil(float64(srcW) * scale))
	scaledH := int(math.Ceil(float64(srcH) * scale))

	scaled := nearestNeighborScale(src, scaledW, scaledH)

	cropX := (scaledW - target.W) / 2
	cropY := (scaledH - target.H) / 2
	cropped := image.NewRGBA(image.Rect(0, 0, target.W, target.H))
	draw.Draw(cropped, cropped.Bounds(), scaled, image.Pt(cropX, cropY), draw.Src)

	var buf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&buf, cropped)
	default:
		err = jpeg.Encode(&buf, cropped, &jpeg.Options{Quality: 95})
		format = "jpeg"
	}
	if err != nil {
		return nil, "", fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), format, nil
}

func nearestNeighborScale(src image.Image, w, h int) image.Image {
	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		srcY := srcBounds.Min.Y + y*srcH/h
		for x := 0; x < w; x++ {
			srcX := srcBounds.Min.X + x*srcW/w
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func FetchImage(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ProcessImageURL fetches, resizes/crops, returns b64.
func ProcessImageURL(imageURL string, target Size) (string, error) {
	data, err := FetchImage(imageURL)
	if err != nil {
		return "", err
	}
	resized, _, err := ResizeAndCrop(data, target)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(resized), nil
}

// ProcessB64 decodes b64, resizes/crops, returns new b64.
func ProcessB64(b64Data string, target Size) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", err
	}
	resized, _, err := ResizeAndCrop(data, target)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(resized), nil
}
