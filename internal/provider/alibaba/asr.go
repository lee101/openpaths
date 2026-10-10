package alibaba

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

const maxSyncASRBytes = 10 << 20

// Transcribe routes the three Qwen ASR protocols: async file transcription,
// WebSocket "message" recognition, and the sync multimodal-generation call.
func (p *Provider) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	if len(req.File) == 0 {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "file is required", Retryable: false}
	}
	switch {
	case strings.Contains(req.Model, "filetrans"):
		return p.transcribeFile(ctx, req)
	case strings.Contains(req.Model, "asr-flash-message"):
		return p.transcribeMessage(ctx, req)
	default:
		return p.transcribeSync(ctx, req)
	}
}

func audioFormat(filename string, data []byte) (string, string) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	switch ext {
	case "wav", "mp3", "m4a", "aac", "flac", "ogg", "opus", "webm", "amr", "mp4", "mov", "mkv", "wma":
	default:
		ext = ""
	}
	if ext == "" {
		switch ct := http.DetectContentType(data); {
		case strings.Contains(ct, "wav"):
			ext = "wav"
		case strings.Contains(ct, "mpeg"):
			ext = "mp3"
		case strings.Contains(ct, "ogg"):
			ext = "ogg"
		case strings.Contains(ct, "webm"):
			ext = "webm"
		default:
			ext = "wav"
		}
	}
	mime := map[string]string{"mp3": "audio/mpeg", "m4a": "audio/mp4", "mp4": "video/mp4", "mov": "video/quicktime", "mkv": "video/x-matroska", "webm": "audio/webm", "ogg": "audio/ogg", "opus": "audio/ogg"}[ext]
	if mime == "" {
		mime = "audio/" + ext
	}
	return ext, mime
}

func dataURI(req *model.TranscriptionRequest) (string, string) {
	format, mime := audioFormat(req.Filename, req.File)
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(req.File), format
}

func languageHints(lang string) []string {
	if lang = strings.TrimSpace(lang); lang == "" || strings.EqualFold(lang, "auto") {
		return nil
	}
	return []string{strings.ToLower(lang)}
}

func (p *Provider) transcribeSync(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	if len(req.File) > maxSyncASRBytes {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 413, Message: "qwen-audio-3.0-asr-flash accepts files up to 10MB and 5 minutes; use qwen-audio-3.1-asr-flash-filetrans for longer audio", Retryable: false}
	}
	uri, format := dataURI(req)
	params := map[string]any{"format": format}
	if hints := languageHints(req.Language); hints != nil {
		params["language_hints"] = hints
	}
	payload := map[string]any{
		"model":      req.Model,
		"input":      map[string]any{"messages": []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_audio", "input_audio": map[string]string{"data": uri}}}}}},
		"parameters": params,
	}
	body, _ := json.Marshal(payload)
	status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+"/api/v1/services/aigc/multimodal-generation/generation", body, false)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	var out struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Output  struct {
			Text     string `json:"text"`
			Sentence struct {
				Text string `json:"text"`
			} `json:"sentence"`
		} `json:"output"`
		Usage struct {
			Duration float64 `json:"duration"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(responseBody, &out)
	if status != http.StatusOK {
		return nil, asrError(p.Name(), status, out.Code, out.Message, responseBody)
	}
	text := strings.TrimSpace(firstNonEmpty(out.Output.Text, out.Output.Sentence.Text))
	return &model.TranscriptionResponse{Text: text, AudioDurationMS: int64(out.Usage.Duration * 1000), Usage: &model.TranscriptionUsage{Type: "duration", Seconds: out.Usage.Duration}}, nil
}

type asrTokenUsage struct {
	Duration     float64 `json:"duration"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
}

type fileTransTask struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Usage   *asrTokenUsage `json:"usage"`
	Output  struct {
		TaskID           string `json:"task_id"`
		TaskStatus       string `json:"task_status"`
		Code             string `json:"code"`
		Message          string `json:"message"`
		TranscriptionURL string `json:"transcription_url"`
		Output           struct {
			TranscriptionURL string `json:"transcription_url"`
		} `json:"output"`
		Results []struct {
			TranscriptionURL string `json:"transcription_url"`
			Code             string `json:"code"`
			Message          string `json:"message"`
		} `json:"results"`
		Usage *asrTokenUsage `json:"usage"`
	} `json:"output"`
}

func (p *Provider) transcribeFile(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	uri, _ := dataURI(req)
	params := map[string]any{"channel_id": []int{0}}
	if hints := languageHints(req.Language); hints != nil {
		params["language_hints"] = hints
	}
	body, _ := json.Marshal(map[string]any{"model": req.Model, "input": map[string]any{"file_url": uri}, "parameters": params})
	status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+"/api/v1/services/audio/asr/transcription", body)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	var task fileTransTask
	_ = json.Unmarshal(responseBody, &task)
	if status != http.StatusOK || task.Output.TaskID == "" {
		return nil, asrError(p.Name(), status, task.Code, task.Message, responseBody)
	}
	taskURL := p.baseURL + "/api/v1/tasks/" + task.Output.TaskID
	interval := min(p.pollInterval, time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
		status, responseBody, err = p.do(ctx, http.MethodGet, taskURL, nil)
		if err != nil || status == 429 || status >= 500 {
			continue
		}
		task = fileTransTask{}
		if err := json.Unmarshal(responseBody, &task); err != nil {
			return nil, fmt.Errorf("decode ASR task: %w", err)
		}
		switch strings.ToUpper(task.Output.TaskStatus) {
		case "SUCCEEDED":
			return p.fetchTranscript(ctx, task)
		case "FAILED", "CANCELED", "UNKNOWN":
			code, msg := task.Output.Code, task.Output.Message
			for _, r := range task.Output.Results {
				code, msg = firstNonEmpty(code, r.Code), firstNonEmpty(msg, r.Message)
			}
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: strings.TrimSpace("ASR task failed: " + code + " " + msg), Retryable: false}
		}
	}
}

func (p *Provider) fetchTranscript(ctx context.Context, task fileTransTask) (*model.TranscriptionResponse, error) {
	link := firstNonEmpty(task.Output.TranscriptionURL, task.Output.Output.TranscriptionURL)
	for _, r := range task.Output.Results {
		link = firstNonEmpty(link, r.TranscriptionURL)
	}
	if link == "" {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "ASR task succeeded without a transcription URL", Retryable: true}
	}
	if u, err := url.Parse(link); err != nil || u.Scheme != "https" {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "ASR returned an invalid transcription URL", Retryable: false}
	}
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	res, err := p.client.Do(getReq)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil || res.StatusCode != http.StatusOK {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: fmt.Sprintf("download transcript: HTTP %d", res.StatusCode), Retryable: true}
	}
	var doc struct {
		Properties struct {
			OriginalDurationMS int64 `json:"original_duration_in_milliseconds"`
		} `json:"properties"`
		Transcripts []struct {
			Text      string          `json:"text"`
			Sentences json.RawMessage `json:"sentences"`
		} `json:"transcripts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode transcript: %w", err)
	}
	resp := &model.TranscriptionResponse{AudioDurationMS: doc.Properties.OriginalDurationMS}
	parts := make([]string, 0, len(doc.Transcripts))
	for _, t := range doc.Transcripts {
		parts = append(parts, strings.TrimSpace(t.Text))
	}
	resp.Text = strings.TrimSpace(strings.Join(parts, "\n"))
	if len(doc.Transcripts) == 1 {
		resp.Segments = doc.Transcripts[0].Sentences
	}
	u := task.Usage
	if u == nil {
		u = task.Output.Usage
	}
	if u != nil {
		resp.Usage = &model.TranscriptionUsage{Type: "tokens", InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens, Seconds: u.Duration}
	}
	return resp, nil
}

// transcribeMessage streams 16 kHz PCM to the duplex ASR WebSocket at ~10x realtime.
func (p *Provider) transcribeMessage(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	pcm, err := toPCM16k(ctx, req.File)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "could not decode audio: " + err.Error(), Retryable: false}
	}
	wsURL, err := websocketURL(p.baseURL, "/api-ws/v1/inference")
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+p.apiKey)
	dialer := gws.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, res, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		status := 502
		if res != nil {
			status = res.StatusCode
			res.Body.Close()
		}
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: "ASR websocket: " + err.Error(), Retryable: status >= 500 || status == 429}
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	}
	taskID := uuid.NewString()
	header := func(action string) map[string]any {
		return map[string]any{"action": action, "task_id": taskID, "streaming": "duplex"}
	}
	params := map[string]any{"format": "pcm", "sample_rate": 16000}
	if hints := languageHints(req.Language); hints != nil {
		params["language_hints"] = hints
	}
	if err := conn.WriteJSON(map[string]any{"header": header("run-task"), "payload": map[string]any{
		"task_group": "audio", "task": "asr", "function": "recognition", "model": req.Model, "parameters": params, "input": map[string]any{},
	}}); err != nil {
		return nil, err
	}
	type event struct {
		Header struct {
			Event        string `json:"event"`
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		} `json:"header"`
		Payload struct {
			Output struct {
				Sentence struct {
					Text        string `json:"text"`
					SentenceEnd bool   `json:"sentence_end"`
				} `json:"sentence"`
			} `json:"output"`
			Usage *struct {
				Duration     float64 `json:"duration"`
				InputTokens  int     `json:"input_tokens"`
				OutputTokens int     `json:"output_tokens"`
				TotalTokens  int     `json:"total_tokens"`
			} `json:"usage"`
		} `json:"payload"`
	}
	readEvent := func() (event, error) {
		var ev event
		err := conn.ReadJSON(&ev)
		return ev, err
	}
	for {
		ev, err := readEvent()
		if err != nil {
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "ASR websocket: " + err.Error(), Retryable: true}
		}
		if ev.Header.Event == "task-failed" {
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: strings.TrimSpace(ev.Header.ErrorCode + " " + ev.Header.ErrorMessage), Retryable: false}
		}
		if ev.Header.Event == "task-started" {
			break
		}
	}
	sendErr := make(chan error, 1)
	go func() {
		for i := 0; i < len(pcm); i += 3200 {
			if err := conn.WriteMessage(gws.BinaryMessage, pcm[i:min(i+3200, len(pcm))]); err != nil {
				sendErr <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		sendErr <- conn.WriteJSON(map[string]any{"header": header("finish-task"), "payload": map[string]any{"input": map[string]any{}}})
	}()
	var sentences []string
	resp := &model.TranscriptionResponse{AudioDurationMS: int64(math.Round(float64(len(pcm)) / 32))}
	for {
		ev, err := readEvent()
		if err != nil {
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "ASR websocket: " + err.Error(), Retryable: true}
		}
		switch ev.Header.Event {
		case "result-generated":
			if s := ev.Payload.Output.Sentence; s.SentenceEnd && strings.TrimSpace(s.Text) != "" {
				sentences = append(sentences, strings.TrimSpace(s.Text))
			}
			if u := ev.Payload.Usage; u != nil {
				resp.Usage = &model.TranscriptionUsage{Type: "tokens", InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens, Seconds: u.Duration}
			}
		case "task-finished":
			if err := <-sendErr; err != nil {
				return nil, err
			}
			resp.Text = strings.Join(sentences, " ")
			return resp, nil
		case "task-failed":
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: strings.TrimSpace(ev.Header.ErrorCode + " " + ev.Header.ErrorMessage), Retryable: false}
		}
	}
}

// toPCM16k decodes from a temp file because MP4/M4A need a seekable input.
func toPCM16k(ctx context.Context, data []byte) ([]byte, error) {
	f, err := os.CreateTemp("", "openpaths-asr-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-protocol_whitelist", "file", "-i", f.Name(), "-f", "s16le", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", "pipe:1")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("no audio samples")
	}
	return out.Bytes(), nil
}

func websocketURL(baseURL, path string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid provider URL")
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	return u.String(), nil
}

func asrError(provider string, status int, code, message string, body []byte) error {
	msg := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(code+" "+message), string(body)))
	return &baseprovider.ProviderError{Provider: provider, StatusCode: status, Message: msg, Retryable: status == 429 || status >= 500}
}
