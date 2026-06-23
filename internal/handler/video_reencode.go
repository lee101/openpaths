package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxVideoReencodeDownloadBytes = 256 << 20

type reencodedVideo struct {
	URL           string
	Bytes         int64
	OriginalBytes int64
}

func (h *VideoHandler) reencodeVideoWebM(ctx context.Context, sourceURL, modelID string) (*reencodedVideo, error) {
	if h.store == nil {
		return nil, fmt.Errorf("video webm reencode requires configured storage")
	}
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("video reencode source URL is empty")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg is not available on this host")
	}

	tmp, err := os.MkdirTemp("", "openpaths-video-webm-*")
	if err != nil {
		return nil, fmt.Errorf("allocate reencode workspace: %w", err)
	}
	defer os.RemoveAll(tmp)

	inputPath := filepath.Join(tmp, "source.mp4")
	originalBytes, err := downloadVideoForReencode(ctx, sourceURL, inputPath)
	if err != nil {
		return nil, err
	}

	outputPath := filepath.Join(tmp, "optimized.webm")
	ffctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ffctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-an",
		"-c:v", "libvpx-vp9",
		"-b:v", "0",
		"-crf", "36",
		"-deadline", "good",
		"-cpu-used", "4",
		"-row-mt", "1",
		"-pix_fmt", "yuv420p",
		outputPath,
	)
	stderr, err := cmd.CombinedOutput()
	if err != nil {
		tail := string(stderr)
		if len(tail) > 800 {
			tail = tail[len(tail)-800:]
		}
		return nil, fmt.Errorf("ffmpeg webm reencode failed: %s", strings.TrimSpace(tail))
	}

	f, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("open reencoded video: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat reencoded video: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("ffmpeg produced an empty webm")
	}
	safeModel := strings.NewReplacer("/", "-", " ", "-").Replace(modelID)
	url, err := h.store.Upload(ctx, safeModel+"-optimized.webm", "video/webm", f)
	if err != nil {
		return nil, fmt.Errorf("upload reencoded video: %w", err)
	}
	return &reencodedVideo{URL: url, Bytes: info.Size(), OriginalBytes: originalBytes}, nil
}

func downloadVideoForReencode(ctx context.Context, sourceURL, dest string) (int64, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return 0, fmt.Errorf("generated video URL must be http(s)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create video download request: %w", err)
	}
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("download generated video: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return 0, fmt.Errorf("download generated video failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("create video download file: %w", err)
	}
	defer out.Close()
	n, err := io.Copy(out, io.LimitReader(resp.Body, maxVideoReencodeDownloadBytes+1))
	if err != nil {
		return 0, fmt.Errorf("write video download: %w", err)
	}
	if n > maxVideoReencodeDownloadBytes {
		return 0, fmt.Errorf("generated video is too large to reencode")
	}
	return n, nil
}
