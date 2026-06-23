package localwhisper

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestTranscribeWithWhisperCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is unix-only")
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "whisper")
	body := `#!/bin/sh
out=""
input=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output_dir)
      shift
      out="$1"
      ;;
    --model|--output_format|--language|--initial_prompt)
      shift
      ;;
    *)
      input="$1"
      ;;
  esac
  shift
done
base=$(basename "$input")
base=${base%.*}
mkdir -p "$out"
printf "hello local whisper\n" > "$out/$base.txt"
`
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}

	p := New(Config{Bin: script, Model: "base", Timeout: time.Second})
	resp, err := p.Transcribe(context.Background(), &model.TranscriptionRequest{
		File:     []byte("fake audio"),
		Filename: "clip.webm",
		Model:    "local-whisper",
		Language: "en",
		Prompt:   "proper nouns",
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if resp.Text != "hello local whisper" {
		t.Fatalf("Text = %q", resp.Text)
	}
}
