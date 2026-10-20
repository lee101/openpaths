package handler

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Measure the generated file so billing never trusts a caller-supplied duration.
func probeRecastOutputDuration(ctx context.Context, videoURL string) (float64, error) {
	dir, err := os.MkdirTemp("", "openpaths-recast-duration-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "output.mp4")
	if _, err := downloadVideoForReencode(ctx, videoURL, path); err != nil {
		return 0, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(probeCtx, "ffprobe", "-v", "error", "-protocol_whitelist", "file", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		return 0, fmt.Errorf("could not measure recast output duration: %w", err)
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > 31 {
		return 0, fmt.Errorf("invalid recast output duration")
	}
	return seconds, nil
}
