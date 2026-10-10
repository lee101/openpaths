package alibaba

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"

	"github.com/openpaths/openpaths/internal/model"
)

func wavBytes() []byte {
	b := []byte("RIFF\x24\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x80\x3e\x00\x00\x00\x7d\x00\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	return b
}

func TestTranscribeSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/api/v1/services/aigc/multimodal-generation/generation" || !strings.Contains(string(body), `"format":"wav"`) || !strings.Contains(string(body), "data:audio/wav;base64,") || !strings.Contains(string(body), `"language_hints":["en"]`) {
			t.Errorf("path %s body %.300s", r.URL.Path, body)
		}
		io.WriteString(w, `{"output":{"text":"hello world "},"usage":{"duration":11}}`)
	}))
	defer srv.Close()
	resp, err := New("key", srv.URL).Transcribe(context.Background(), &model.TranscriptionRequest{Model: "qwen-audio-3.0-asr-flash", File: wavBytes(), Filename: "a.wav", Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hello world" || resp.Usage == nil || resp.Usage.Type != "duration" || resp.Usage.Seconds != 11 {
		t.Fatalf("resp %+v usage %+v", resp, resp.Usage)
	}
}

func TestTranscribeFileTrans(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/services/audio/asr/transcription":
			if r.Header.Get("X-DashScope-Async") != "enable" {
				t.Error("filetrans must be async")
			}
			io.WriteString(w, `{"output":{"task_id":"t1","task_status":"PENDING"}}`)
		case "/api/v1/tasks/t1":
			io.WriteString(w, `{"output":{"task_id":"t1","task_status":"SUCCEEDED","transcription_url":"`+srv.URL+`/result.json"},"usage":{"duration":9,"input_tokens":194,"output_tokens":28,"total_tokens":222}}`)
		case "/result.json":
			io.WriteString(w, `{"properties":{"original_duration_in_milliseconds":10580},"transcripts":[{"text":"Hello there.","sentences":[{"begin_time":0,"end_time":900,"text":"Hello there."}]}]}`)
		}
	}))
	defer srv.Close()
	p := New("key", srv.URL)
	p.client = srv.Client()
	p.pollInterval = time.Millisecond
	resp, err := p.Transcribe(context.Background(), &model.TranscriptionRequest{Model: "qwen-audio-3.1-asr-flash-filetrans", File: wavBytes(), Filename: "a.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello there." || resp.AudioDurationMS != 10580 || resp.Usage.InputTokens != 194 || resp.Usage.OutputTokens != 28 || !strings.Contains(string(resp.Segments), "begin_time") {
		t.Fatalf("resp %+v usage %+v", resp, resp.Usage)
	}
}

func TestTranscribeMessageWebSocket(t *testing.T) {
	upgrader := gws.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api-ws/v1/inference" || r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("path %s auth %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var run map[string]any
		_ = conn.ReadJSON(&run)
		if run["payload"].(map[string]any)["model"] != "qwen-audio-3.1-asr-flash-message" {
			t.Errorf("run %v", run)
		}
		_ = conn.WriteJSON(map[string]any{"header": map[string]any{"event": "task-started"}})
		var audioBytes int
		for {
			kind, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind == gws.BinaryMessage {
				audioBytes += len(data)
				continue
			}
			var msg map[string]map[string]any
			_ = json.Unmarshal(data, &msg)
			if msg["header"]["action"] == "finish-task" {
				break
			}
		}
		if audioBytes == 0 {
			t.Error("no audio streamed")
		}
		_ = conn.WriteJSON(map[string]any{"header": map[string]any{"event": "result-generated"}, "payload": map[string]any{"output": map[string]any{"sentence": map[string]any{"text": "First.", "sentence_end": true}}}})
		_ = conn.WriteJSON(map[string]any{"header": map[string]any{"event": "result-generated"}, "payload": map[string]any{"output": map[string]any{"sentence": map[string]any{"text": "Second.", "sentence_end": true}}, "usage": map[string]any{"duration": 2, "input_tokens": 40, "output_tokens": 5, "total_tokens": 45}}})
		_ = conn.WriteJSON(map[string]any{"header": map[string]any{"event": "task-finished"}})
	}))
	defer srv.Close()
	pcmWav := append(wavBytes()[:40], []byte{0x00, 0x7d, 0x00, 0x00}...)
	pcmWav = append(pcmWav, make([]byte, 32000)...)
	resp, err := New("key", srv.URL).Transcribe(context.Background(), &model.TranscriptionRequest{Model: "qwen-audio-3.1-asr-flash-message", File: pcmWav, Filename: "a.wav"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "First. Second." || resp.Usage.InputTokens != 40 {
		t.Fatalf("resp %+v", resp)
	}
}
