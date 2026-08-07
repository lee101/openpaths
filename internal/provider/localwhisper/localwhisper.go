package localwhisper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	defaultBin     = "whisper"
	defaultModel   = "turbo"
	defaultTimeout = 10 * time.Minute
)

type Config struct {
	Bin     string
	Model   string
	Timeout time.Duration
}

type Transcriber struct {
	bin     string
	model   string
	timeout time.Duration
}

func New(cfg Config) *Transcriber {
	bin := strings.TrimSpace(cfg.Bin)
	if bin == "" {
		bin = defaultBin
	}
	modelName := strings.TrimSpace(cfg.Model)
	if modelName == "" {
		modelName = defaultModel
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Transcriber{bin: bin, model: modelName, timeout: timeout}
}

func NewFromEnv() *Transcriber {
	timeout := defaultTimeout
	if raw := strings.TrimSpace(os.Getenv("LOCAL_WHISPER_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	return New(Config{
		Bin:     os.Getenv("LOCAL_WHISPER_BIN"),
		Model:   os.Getenv("LOCAL_WHISPER_MODEL"),
		Timeout: timeout,
	})
}

func (t *Transcriber) Name() string { return "local-whisper" }

func (t *Transcriber) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	if len(req.File) == 0 {
		return nil, providerError(400, "file is required", false, nil)
	}
	bin, err := exec.LookPath(t.bin)
	if err != nil {
		return nil, providerError(503, fmt.Sprintf("%q not found; install Whisper or set LOCAL_WHISPER_BIN", t.bin), false, err)
	}

	tmpDir, err := os.MkdirTemp("", "openpaths-local-whisper-*")
	if err != nil {
		return nil, providerError(500, "create temp dir: "+err.Error(), false, err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, safeAudioFilename(req.Filename))
	if err := os.WriteFile(inputPath, req.File, 0600); err != nil {
		return nil, providerError(500, "write temp audio: "+err.Error(), false, err)
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" || modelName == "auto" || modelName == "local-whisper" {
		modelName = t.model
	}

	runCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	args := []string{
		inputPath,
		"--model", modelName,
		"--output_format", "txt",
		"--output_dir", tmpDir,
	}
	if req.Language != "" {
		args = append(args, "--language", req.Language)
	}
	if req.Prompt != "" {
		args = append(args, "--initial_prompt", req.Prompt)
	}

	var combined bytes.Buffer
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, providerError(504, "local whisper timed out", true, runCtx.Err())
		}
		msg := strings.TrimSpace(combined.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, providerError(502, "local whisper failed: "+msg, true, err)
	}

	outputPath := filepath.Join(tmpDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+".txt")
	text, err := os.ReadFile(outputPath)
	if err != nil {
		msg := strings.TrimSpace(combined.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, providerError(502, "read local whisper output: "+msg, true, err)
	}

	return &model.TranscriptionResponse{Text: strings.TrimSpace(string(text))}, nil
}

func providerError(status int, message string, retryable bool, err error) *provider.ProviderError {
	return &provider.ProviderError{
		Provider:   "local-whisper",
		StatusCode: status,
		Message:    message,
		Retryable:  retryable,
		Err:        err,
	}
}

func safeAudioFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "audio.webm"
	}
	ext := filepath.Ext(base)
	if ext == "" {
		base += ".webm"
	}
	return base
}
