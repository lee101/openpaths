package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	WebPQuality    = 85
	webpMinSavings = 0.10
	webpMaxInput   = 40 << 20
)

var (
	cwebpOnce sync.Once
	cwebpPath string
)

func cwebp() string {
	cwebpOnce.Do(func() {
		if p, err := exec.LookPath("cwebp"); err == nil {
			cwebpPath = p
		}
	})
	return cwebpPath
}

func convertible(data []byte) bool {
	return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) || bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF})
}

func WantsWebP(outputFormat string) bool {
	switch outputFormat {
	case "", "webp":
		return true
	}
	return false
}

func ToWebP(ctx context.Context, data []byte) ([]byte, bool) {
	bin := cwebp()
	if bin == "" || len(data) == 0 || len(data) > webpMaxInput || !convertible(data) {
		return nil, false
	}
	dir, err := os.MkdirTemp("", "webp-")
	if err != nil {
		return nil, false
	}
	defer os.RemoveAll(dir)
	in, out := filepath.Join(dir, "in"), filepath.Join(dir, "out.webp")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, false
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "-quiet", "-q", strconv.Itoa(WebPQuality), "-m", "4", "-alpha_q", "100", "-metadata", "none", in, "-o", out)
	if err := cmd.Run(); err != nil {
		return nil, false
	}
	res, err := os.ReadFile(out)
	if err != nil || float64(len(res)) > float64(len(data))*(1-webpMinSavings) {
		return nil, false
	}
	return res, true
}

func ToWebPB64(ctx context.Context, b64 string) (string, bool) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", false
	}
	out, ok := ToWebP(ctx, data)
	if !ok {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(out), true
}
