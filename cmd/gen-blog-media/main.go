// Command gen-blog-media generates the real images and video used by the
// image/video model tips blog posts. It reuses the actual OpenPaths provider
// clients (fal, xai, netwrck) so the output matches what the API serves, and
// pays through provider keys directly — no op- key / billing path.
//
// It chains generations: base stills are produced on fal (which returns a
// hosted fal.media URL), and that URL is fed straight into outpaint, image
// edit, and image-to-video so nothing needs self-hosting mid-run.
//
// Images are written as WebP q85. Video is encoded to AV1 WebM (primary) with
// an h264 mp4 fallback and a WebP poster.
//
// Usage:
//
//	go run ./cmd/gen-blog-media          # everything
//	go run ./cmd/gen-blog-media outpaint # only steps whose name contains "outpaint"
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider/fal"
	"github.com/openpaths/openpaths/internal/provider/xai"
)

const (
	imgDir      = "public/static/blog/image-tips"
	vidDir      = "public/static/blog/video-tips"
	webpQuality = 85
)

var (
	falP *fal.FalProvider
	xaiP *xai.XAIProvider
	only map[string]bool
)

// ---- prompts -------------------------------------------------------------

const (
	promptBare = "a fox"

	promptStructured = "A cinematic wildlife photograph of a red fox sitting on a " +
		"moss-covered rock in a misty pine forest at golden hour, shot on an 85mm " +
		"f/1.4 lens, shallow depth of field, warm rim light, volumetric fog, " +
		"ultra-detailed fur, sharp focus, photorealistic"

	promptStreet = "A lone figure holding a glowing umbrella walking down a neon-lit " +
		"rainy Tokyo alley at night, reflections on wet pavement, dense signage, " +
		"cinematic color grade, moody atmosphere, photorealistic"

	promptOutpaintBase = "A astronaut in a white suit standing on a red alien desert, " +
		"centered full-body composition, dramatic sunset sky, cinematic, photorealistic"

	promptOutpaint = "Extend the alien desert landscape outward with rolling dunes, " +
		"distant jagged mountains, and two pale moons low in the sunset sky"

	promptEditBase = "A studio product photograph of a single emerald-green glass " +
		"perfume bottle on a polished marble surface, soft diffused lighting, clean " +
		"neutral background, photorealistic"

	promptEdit = "Replace the perfume bottle with a rugged black camera lens standing " +
		"upright, keep the same marble surface, lighting and background"

	promptVideoBase = "A cinematic wide aerial shot of turquoise ocean waves crashing " +
		"against dark rocky cliffs at sunset, golden backlight, sea spray, drifting " +
		"clouds, photorealistic"

	promptVideoMotion = "Slow cinematic aerial push-in, ocean waves rolling and " +
		"crashing against the cliffs, sea spray rising, clouds drifting across the " +
		"golden sky, gentle continuous camera motion"
)

func want(name string) bool { return len(only) == 0 || only[name] }

func main() {
	_ = godotenv.Overload()
	falP = fal.New(falKey())
	xaiP = xai.New(os.Getenv("XAI_API_KEY"), "https://api.x.ai")

	only = map[string]bool{}
	for _, a := range os.Args[1:] {
		only[a] = true
	}
	must(os.MkdirAll(imgDir, 0o755))
	must(os.MkdirAll(vidDir, 0o755))

	// --- prompt anatomy: bare vs structured -----------------------------
	if want("prompt") {
		genImage("prompt-bare", "fal-ai/flux/dev", promptBare, "1024x1024", imgDir)
		genImage("prompt-structured", "fal-ai/flux/dev", promptStructured, "1024x1024", imgDir)
	}

	// --- aspect ratio: same scene, two framings -------------------------
	if want("aspect") {
		genImage("aspect-portrait", "fal-ai/flux/dev", promptStreet, "832x1216", imgDir)
		genImage("aspect-wide", "fal-ai/flux/dev", promptStreet, "1216x832", imgDir)
	}

	// --- outpainting: base -> expanded ----------------------------------
	if want("outpaint") {
		if url := genImage("outpaint-before", "fal-ai/flux/dev", promptOutpaintBase, "832x1216", imgDir); url != "" {
			genOutpaint("outpaint-after", url, promptOutpaint)
		}
	}

	// --- image edit: base -> edited -------------------------------------
	if want("edit") {
		if url := genImage("edit-before", "fal-ai/flux/dev", promptEditBase, "1024x1024", imgDir); url != "" {
			genEdit("edit-after", url, promptEdit)
		}
	}

	// --- image-to-video: base -> clip -----------------------------------
	if want("video") {
		url := genImage("coast-still", "fal-ai/flux/dev", promptVideoBase, "1216x832", vidDir)
		if url != "" {
			genVideo("coast", url, promptVideoMotion)
		}
	}

	fmt.Println("done")
}

// genImage runs a plain text-to-image generation, writes <slug>.webp into dir,
// and returns the provider-hosted source URL (for chaining) when available.
func genImage(slug, modelID, prompt, size, dir string) string {
	fmt.Printf("→ image %-18s (%s) ... ", slug, modelID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	resp, err := falP.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model: modelID, Prompt: prompt, N: 1, Size: size, ResponseFormat: "url",
	})
	url, data, err := firstImage(ctx, resp, err)
	if err != nil {
		fmt.Println("FAILED:", err)
		return ""
	}
	if err := writeWebP(filepath.Join(dir, slug+".webp"), data); err != nil {
		fmt.Println("FAILED webp:", err)
		return url
	}
	fmt.Println("ok")
	return url
}

func genOutpaint(slug, imageURL, prompt string) {
	fmt.Printf("→ outpaint %-15s ... ", slug)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	expand := 512
	resp, err := falP.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:        "fal-ai/flux-2-pro/outpaint",
		Prompt:       prompt,
		ImageURL:     imageURL,
		ExpandLeft:   &expand,
		ExpandRight:  &expand,
		OutputFormat: "png",
	})
	_, data, err := firstImage(ctx, resp, err)
	if err != nil {
		fmt.Println("FAILED:", err)
		return
	}
	if err := writeWebP(filepath.Join(imgDir, slug+".webp"), data); err != nil {
		fmt.Println("FAILED webp:", err)
		return
	}
	fmt.Println("ok")
}

func genEdit(slug, imageURL, prompt string) {
	fmt.Printf("→ edit %-19s ... ", slug)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	resp, err := falP.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:              "fal-ai/hidream-o1-image/edit",
		Prompt:             prompt,
		ReferenceImageURLs: []string{imageURL},
		ImageSize:          "square_hd",
		OutputFormat:       "png",
	})
	_, data, err := firstImage(ctx, resp, err)
	if err != nil {
		fmt.Println("FAILED:", err)
		return
	}
	if err := writeWebP(filepath.Join(imgDir, slug+".webp"), data); err != nil {
		fmt.Println("FAILED webp:", err)
		return
	}
	fmt.Println("ok")
}

func genVideo(slug, imageURL, prompt string) {
	fmt.Printf("→ video %-18s ... ", slug)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	resp, err := xaiP.GenerateVideo(ctx, &model.VideoGenerationRequest{
		Model:      "grok-imagine-video",
		Prompt:     prompt,
		ImageURL:   imageURL,
		Duration:   model.VideoDuration("6"),
		Resolution: "720p",
	})
	if err != nil {
		fmt.Println("FAILED:", err)
		return
	}
	if resp == nil || resp.VideoURL == "" {
		fmt.Println("FAILED: no video url")
		return
	}
	raw, err := download(ctx, resp.VideoURL)
	if err != nil {
		fmt.Println("FAILED download:", err)
		return
	}
	src := filepath.Join(vidDir, slug+"-src.mp4")
	if err := os.WriteFile(src, raw, 0o644); err != nil {
		fmt.Println("FAILED write src:", err)
		return
	}
	if err := encodeVideo(ctx, src, filepath.Join(vidDir, slug)); err != nil {
		fmt.Println("FAILED encode:", err)
		return
	}
	_ = os.Remove(src)
	fmt.Println("ok")
}

// ---- helpers -------------------------------------------------------------

// firstImage extracts the first image from a provider response as both its
// source URL (when present) and decoded bytes (downloading the URL if needed).
func firstImage(ctx context.Context, resp *model.ImageGenerationResponse, err error) (string, []byte, error) {
	if err != nil {
		return "", nil, err
	}
	if resp == nil || len(resp.Data) == 0 {
		return "", nil, fmt.Errorf("no image data")
	}
	img := resp.Data[0]
	switch {
	case img.URL != "":
		data, err := download(ctx, img.URL)
		return img.URL, data, err
	case img.B64JSON != "":
		data, err := base64.StdEncoding.DecodeString(img.B64JSON)
		return "", data, err
	default:
		return "", nil, fmt.Errorf("response had neither url nor b64_json")
	}
}

func writeWebP(path string, data []byte) error {
	cmd := exec.Command("cwebp", "-q", fmt.Sprintf("%d", webpQuality), "-quiet", "-o", path, "--", "-")
	cmd.Stdin = bytes.NewReader(data)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, errBuf.String())
	}
	return nil
}

// encodeVideo writes <base>.webm (AV1), <base>.mp4 (h264 fallback) and
// <base>-poster.webp from a source mp4. Audio is dropped: the blog clips
// autoplay muted on loop, so the audio track is dead weight.
func encodeVideo(ctx context.Context, src, base string) error {
	// AV1 WebM — best compression-per-quality for a web background loop.
	if err := run(ctx, "ffmpeg", "-y", "-i", src,
		"-c:v", "libsvtav1", "-crf", "30", "-preset", "5",
		"-pix_fmt", "yuv420p", "-an", base+".webm"); err != nil {
		return fmt.Errorf("av1: %w", err)
	}
	// h264 mp4 fallback for browsers without AV1.
	if err := run(ctx, "ffmpeg", "-y", "-i", src,
		"-c:v", "libx264", "-crf", "23", "-preset", "slow",
		"-pix_fmt", "yuv420p", "-movflags", "+faststart", "-an", base+".mp4"); err != nil {
		return fmt.Errorf("h264: %w", err)
	}
	// Poster: first frame -> PNG -> WebP q85.
	png, err := output(ctx, "ffmpeg", "-y", "-i", src,
		"-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-")
	if err != nil {
		return fmt.Errorf("poster frame: %w", err)
	}
	if err := writeWebP(base+"-poster.webp", png); err != nil {
		return fmt.Errorf("poster webp: %w", err)
	}
	return nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := errBuf.String()
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

func output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}

func download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 3 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func falKey() string {
	if k := strings.TrimSpace(os.Getenv("FAL_API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("FAL_KEY"))
}

func must(err error) {
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
}
