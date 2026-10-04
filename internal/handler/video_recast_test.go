package handler

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRecastMeasuresActualOutputDuration(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe unavailable")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg unavailable")
	}
	file := filepath.Join(t.TempDir(), "video.mp4")
	output, err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=black:s=32x32:r=20", "-t", "5.5", "-c:v", "libx264", file).CombinedOutput()
	if err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	seconds, err := probeRecastOutputDuration(context.Background(), server.URL+"/output.mp4")
	if err != nil || math.Abs(seconds-5.5) > .01 {
		t.Fatalf("seconds %v, err %v", seconds, err)
	}
}
