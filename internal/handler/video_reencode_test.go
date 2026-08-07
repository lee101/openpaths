package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/storage"
)

func TestWantsVideoWebM(t *testing.T) {
	for _, value := range []string{"webm", "WEBM", "video/webm", " video/webm "} {
		if !wantsVideoWebM(value) {
			t.Fatalf("wantsVideoWebM(%q) = false", value)
		}
	}
	if wantsVideoWebM("mp4") {
		t.Fatal("wantsVideoWebM(mp4) = true")
	}
}

func TestReencodeVideoWebMUploadsResult(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=size=160x90:rate=12", "-t", "1", "-pix_fmt", "yuv420p", source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create source video: %v\n%s", err, out)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, source)
	}))
	defer srv.Close()

	store, err := storage.NewLocalStore(filepath.Join(tmp, "uploads"), "http://assets.test")
	if err != nil {
		t.Fatal(err)
	}
	h := &VideoHandler{store: store}
	got, err := h.reencodeVideoWebM(context.Background(), srv.URL+"/source.mp4", "grok-imagine-video")
	if err != nil {
		t.Fatal(err)
	}
	if got.URL == "" || !strings.HasSuffix(got.URL, ".webm") {
		t.Fatalf("url = %q, want webm URL", got.URL)
	}
	if got.Bytes <= 0 || got.OriginalBytes <= 0 {
		t.Fatalf("sizes = %+v, want positive sizes", got)
	}
	uploaded := filepath.Join(tmp, "uploads", filepath.Base(got.URL))
	if _, err := os.Stat(uploaded); err != nil {
		t.Fatalf("uploaded file missing: %v", err)
	}
}
