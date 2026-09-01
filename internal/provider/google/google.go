package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/safefetch"
	"google.golang.org/genai"
)

type GoogleProvider struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	imageClient *http.Client
	cacheMgr    *GeminiCacheManager
}

func New(apiKey, baseURL string) *GoogleProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &GoogleProvider{
		apiKey:      apiKey,
		baseURL:     strings.TrimRight(baseURL, "/"),
		client:      &http.Client{Timeout: 5 * time.Minute},
		imageClient: safefetch.NewClient(30 * time.Second),
	}
}

// SetCacheManager attaches a Gemini explicit-cache manager. When nil or
// disabled, requests are sent unchanged (relying on free implicit caching).
func (p *GoogleProvider) SetCacheManager(m *GeminiCacheManager) {
	p.cacheMgr = m
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) GenerateMusic(ctx context.Context, req *model.MusicGenerationRequest) (*model.MusicGenerationResponse, error) {
	prompt := strings.TrimSpace(req.Lyrics)
	if prompt == "" {
		prompt = strings.TrimSpace(req.Prompt)
	}
	if prompt == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: 400, Message: "lyrics or prompt is required", Retryable: false}
	}

	baseURL := p.baseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  p.apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: baseURL,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
	}

	audio, mimeType, _, _, err := collectAudioFromGenerateContentStream(ctx, client, req.Model, prompt, config)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: 502, Message: "no audio returned", Retryable: true}
	}
	audio, mimeType, format, err := encodeLyriaOutput(ctx, audio, mimeType, req.OutputFormat)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: false, Err: err}
	}

	return &model.MusicGenerationResponse{
		Data: &model.MusicData{
			Status:   2,
			Audio:    base64.StdEncoding.EncodeToString(audio),
			Format:   format,
			MimeType: mimeType,
		},
		ExtraInfo: &model.MusicExtraInfo{
			Size: len(audio),
		},
		AnalysisInfo: map[string]string{
			"mime_type": mimeType,
		},
		BaseResp: &model.MusicBaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
	}, nil
}

func encodeLyriaOutput(ctx context.Context, audio []byte, mimeType, requested string) ([]byte, string, string, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "" || requested == "url" {
		return audio, mimeType, audioFormatFromMime(mimeType), nil
	}
	if requested == "mp3" && (strings.Contains(strings.ToLower(mimeType), "mpeg") || strings.Contains(strings.ToLower(mimeType), "mp3")) {
		return audio, "audio/mpeg", "mp3", nil
	}

	var args []string
	var outputMIME string
	switch requested {
	case "opus", "ogg":
		args = []string{"-c:a", "libopus", "-b:a", "192k", "-vbr", "on", "-application", "audio", "-f", "opus"}
		outputMIME = "audio/ogg;codecs=opus"
		requested = "opus"
	case "wav":
		args = []string{"-c:a", "pcm_s16le", "-f", "wav"}
		outputMIME = "audio/wav"
	case "mp3":
		args = []string{"-c:a", "libmp3lame", "-b:a", "256k", "-f", "mp3"}
		outputMIME = "audio/mpeg"
	default:
		return nil, "", "", fmt.Errorf("unsupported Lyria output format %q (use opus, mp3, or wav)", requested)
	}

	ffctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	commandArgs := append([]string{"-hide_banner", "-loglevel", "error", "-i", "pipe:0"}, args...)
	commandArgs = append(commandArgs, "pipe:1")
	cmd := exec.CommandContext(ffctx, "ffmpeg", commandArgs...)
	cmd.Stdin = bytes.NewReader(audio)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if len(message) > 300 {
			message = message[len(message)-300:]
		}
		return nil, "", "", fmt.Errorf("encode Lyria audio as %s: %w: %s", requested, err, message)
	}
	if stdout.Len() == 0 {
		return nil, "", "", fmt.Errorf("encode Lyria audio as %s: ffmpeg produced no output", requested)
	}
	return stdout.Bytes(), outputMIME, requested, nil
}

func (p *GoogleProvider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	text := strings.TrimSpace(req.Input)
	if text == "" {
		text = strings.TrimSpace(req.Text)
	}
	if text == "" {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: 400, Message: "input is required", Retryable: false}
	}

	baseURL := p.baseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  p.apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: baseURL,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig:       buildGeminiSpeechConfig(req),
	}
	if req.Temperature != nil {
		temp := float32(*req.Temperature)
		config.Temperature = &temp
	}

	audio, mimeType, inputTokens, outputTokens, err := collectAudioFromGenerateContentStream(ctx, client, req.Model, text, config)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, &provider.ProviderError{Provider: "google", StatusCode: 502, Message: "no audio returned", Retryable: true}
	}
	audio, mimeType = browserReadySpeechAudio(audio, mimeType)

	return &model.SpeechResponse{
		Audio:        base64.StdEncoding.EncodeToString(audio),
		Format:       audioFormatFromMime(mimeType),
		Characters:   len(text),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	}, nil
}

// Gemini TTS returns headerless 24 kHz, 16-bit mono PCM. Browsers cannot play
// those bytes directly, so expose a real WAV from OpenPaths' speech endpoint.
func browserReadySpeechAudio(audio []byte, mimeType string) ([]byte, string) {
	lower := strings.ToLower(mimeType)
	if !strings.Contains(lower, "pcm") && !strings.Contains(lower, "l16") {
		return audio, mimeType
	}

	const (
		sampleRate    = uint32(24000)
		channels      = uint16(1)
		bitsPerSample = uint16(16)
	)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample/8)
	blockAlign := channels * (bitsPerSample / 8)
	wav := make([]byte, 44+len(audio))
	copy(wav[0:4], "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(len(wav)-8))
	copy(wav[8:12], "WAVE")
	copy(wav[12:16], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1)
	binary.LittleEndian.PutUint16(wav[22:24], channels)
	binary.LittleEndian.PutUint32(wav[24:28], sampleRate)
	binary.LittleEndian.PutUint32(wav[28:32], byteRate)
	binary.LittleEndian.PutUint16(wav[32:34], blockAlign)
	binary.LittleEndian.PutUint16(wav[34:36], bitsPerSample)
	copy(wav[36:40], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(len(audio)))
	copy(wav[44:], audio)
	return wav, "audio/wav"
}

func buildGeminiSpeechConfig(req *model.SpeechRequest) *genai.SpeechConfig {
	cfg := &genai.SpeechConfig{}
	if req.Language != "" {
		cfg.LanguageCode = req.Language
	}
	if len(req.SpeakerVoices) > 0 {
		speakers := make([]*genai.SpeakerVoiceConfig, 0, len(req.SpeakerVoices))
		for _, speaker := range req.SpeakerVoices {
			if strings.TrimSpace(speaker.Speaker) == "" {
				continue
			}
			speakers = append(speakers, &genai.SpeakerVoiceConfig{
				Speaker: speaker.Speaker,
				VoiceConfig: &genai.VoiceConfig{
					PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: normalizeGeminiTTSVoice(speaker.Voice)},
				},
			})
		}
		if len(speakers) > 0 {
			cfg.MultiSpeakerVoiceConfig = &genai.MultiSpeakerVoiceConfig{SpeakerVoiceConfigs: speakers}
			return cfg
		}
	}

	voice := req.Voice
	if voice == "" {
		voice = req.VoiceID
	}
	cfg.VoiceConfig = &genai.VoiceConfig{
		PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: normalizeGeminiTTSVoice(voice)},
	}
	return cfg
}

func normalizeGeminiTTSVoice(voice string) string {
	switch strings.ToLower(strings.TrimSpace(voice)) {
	case "achernar":
		return "Achernar"
	case "achird":
		return "Achird"
	case "algenib":
		return "Algenib"
	case "algieba":
		return "Algieba"
	case "alnilam":
		return "Alnilam"
	case "aoede":
		return "Aoede"
	case "autonoe":
		return "Autonoe"
	case "callirrhoe":
		return "Callirrhoe"
	case "charon":
		return "Charon"
	case "despina":
		return "Despina"
	case "enceladus":
		return "Enceladus"
	case "erinome":
		return "Erinome"
	case "fenrir":
		return "Fenrir"
	case "gacrux":
		return "Gacrux"
	case "iapetus":
		return "Iapetus"
	case "kore":
		return "Kore"
	case "laomedeia":
		return "Laomedeia"
	case "leda":
		return "Leda"
	case "orus":
		return "Orus"
	case "puck":
		return "Puck"
	case "pulcherrima":
		return "Pulcherrima"
	case "rasalgethi":
		return "Rasalgethi"
	case "sadachbia":
		return "Sadachbia"
	case "sadaltager":
		return "Sadaltager"
	case "schedar":
		return "Schedar"
	case "sulafat":
		return "Sulafat"
	case "umbriel":
		return "Umbriel"
	case "vindemiatrix":
		return "Vindemiatrix"
	case "zephyr":
		return "Zephyr"
	case "zubenelgenubi":
		return "Zubenelgenubi"
	default:
		return "Puck"
	}
}

func audioFormatFromMime(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "wav"):
		return "wav"
	case strings.Contains(mimeType, "ogg"):
		return "ogg"
	case strings.Contains(mimeType, "flac"):
		return "flac"
	case strings.Contains(mimeType, "mp4"):
		return "mp4"
	case strings.Contains(mimeType, "webm"):
		return "webm"
	case strings.Contains(mimeType, "mpeg"), strings.Contains(mimeType, "mp3"):
		return "mp3"
	default:
		return "wav"
	}
}

func collectAudioFromGenerateContentStream(ctx context.Context, client *genai.Client, modelName, prompt string, config *genai.GenerateContentConfig) ([]byte, string, int, int, error) {
	var audio []byte
	var mimeType string
	var inputTokens int
	var outputTokens int

	for chunk, err := range client.Models.GenerateContentStream(ctx, modelName, genai.Text(prompt), config) {
		if err != nil {
			return nil, "", 0, 0, googleProviderError(err)
		}
		if chunk.UsageMetadata != nil {
			inputTokens = int(chunk.UsageMetadata.PromptTokenCount)
			outputTokens = int(chunk.UsageMetadata.CandidatesTokenCount)
		}
		for _, cand := range chunk.Candidates {
			if cand == nil || cand.Content == nil {
				continue
			}
			for _, part := range cand.Content.Parts {
				if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
					continue
				}
				if mimeType == "" {
					mimeType = part.InlineData.MIMEType
				}
				audio = append(audio, part.InlineData.Data...)
			}
		}
	}

	return audio, mimeType, inputTokens, outputTokens, nil
}

func googleProviderError(err error) error {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return &provider.ProviderError{
			Provider:   "google",
			StatusCode: apiErr.Code,
			Message:    apiErr.Message,
			Retryable:  apiErr.Code >= 500 || apiErr.Code == 429,
			Err:        err,
		}
	}
	return &provider.ProviderError{Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
}

// Gemini API types
type geminiRequest struct {
	Contents          []geminiContent      `json:"contents"`
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationCfg `json:"generationConfig,omitempty"`
	Tools             []geminiToolDecl     `json:"tools,omitempty"`
	// CachedContent references a pre-created explicit cache (e.g.
	// "cachedContents/abc"). When set, SystemInstruction and Tools live in the
	// cache and must be omitted from the request.
	CachedContent string `json:"cachedContent,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string          `json:"text,omitempty"`
	InlineData   *geminiBlob     `json:"inlineData,omitempty"`
	FileData     *geminiFileData `json:"fileData,omitempty"`
	FunctionCall *geminiFuncCall `json:"functionCall,omitempty"`
	FunctionResp *geminiFuncResp `json:"functionResponse,omitempty"`
}

type geminiBlob struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data"`
}

type geminiFileData struct {
	MimeType string `json:"mimeType,omitempty"`
	FileURI  string `json:"fileUri"`
}

type geminiFuncCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

type geminiFuncResp struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

type geminiGenerationCfg struct {
	Temperature     *float64           `json:"temperature,omitempty"`
	TopP            *float64           `json:"topP,omitempty"`
	MaxOutputTokens *int               `json:"maxOutputTokens,omitempty"`
	StopSequences   []string           `json:"stopSequences,omitempty"`
	ThinkingConfig  *geminiThinkingCfg `json:"thinkingConfig,omitempty"`
}

type geminiThinkingCfg struct {
	ThinkingBudget *int   `json:"thinkingBudget,omitempty"`
	ThinkingLevel  string `json:"thinkingLevel,omitempty"`
}

type geminiToolDecl struct {
	FunctionDeclarations []geminiFuncDecl `json:"functionDeclarations"`
}

type geminiFuncDecl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata"`
}

type geminiEmbedRequest struct {
	Model                string             `json:"model,omitempty"`
	Content              geminiEmbedContent `json:"content"`
	TaskType             string             `json:"taskType,omitempty"`
	OutputDimensionality *int               `json:"outputDimensionality,omitempty"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedPart struct {
	Text string `json:"text,omitempty"`
}

type geminiEmbedResponse struct {
	Embedding geminiContentEmbedding `json:"embedding"`
}

type geminiBatchEmbedRequest struct {
	Requests []geminiEmbedRequest `json:"requests"`
}

type geminiBatchEmbedResponse struct {
	Embeddings []geminiContentEmbedding `json:"embeddings"`
}

type geminiContentEmbedding struct {
	Values []float64 `json:"values"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsage struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
}

func (p *GoogleProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	gemReq := translateRequest(req)

	// Explicit context cache (default-off). When the manager decides to use a
	// cache, the stable prefix (systemInstruction + tools) is omitted from the
	// request and referenced via cachedContent instead.
	var cacheKey string
	if p.cacheMgr != nil {
		if name, key := p.cacheMgr.Ensure(ctx, req.Model, gemReq); name != "" {
			cacheKey = key
			gemReq.CachedContent = name
			gemReq.SystemInstruction = nil
			gemReq.Tools = nil
		}
	}

	respBody, status, err := p.sendGenerate(ctx, req.Model, gemReq)
	if err != nil {
		return nil, err
	}
	// A stale or forbidden cache: drop it and retry once with the full prefix so
	// the request never hard-fails because of our caching.
	if status != 200 && cacheKey != "" && (status == 403 || status == 404) {
		p.cacheMgr.Invalidate(cacheKey)
		respBody, status, err = p.sendGenerate(ctx, req.Model, translateRequest(req))
		if err != nil {
			return nil, err
		}
	}
	if status != 200 {
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: status,
			Message:    string(respBody),
			Retryable:  status >= 500 || status == 429,
		}
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return translateResponse(&gemResp, req.Model), nil
}

// sendGenerate performs one generateContent POST and returns the raw body and
// HTTP status (or a transport-level ProviderError).
func (p *GoogleProvider) sendGenerate(ctx context.Context, modelName string, gemReq *geminiRequest) ([]byte, int, error) {
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, modelName, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, 0, &provider.ProviderError{
			Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

func (p *GoogleProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	inputs, err := normalizeEmbeddingInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "google", StatusCode: 400, Message: err.Error(),
		}
	}

	if len(inputs) == 0 {
		return &model.EmbeddingResponse{
			Object: "list",
			Data:   []model.EmbeddingData{},
			Model:  req.Model,
			Usage:  model.EmbeddingUsage{},
		}, nil
	}

	requests := make([]geminiEmbedRequest, 0, len(inputs))
	totalTokens := 0
	for _, text := range inputs {
		requests = append(requests, buildGeminiEmbedRequest(req.Model, text, req.Dimensions))
		totalTokens += len(text) / 4
	}

	body, err := json.Marshal(geminiBatchEmbedRequest{Requests: requests})
	if err != nil {
		return nil, fmt.Errorf("marshal batch embed request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:batchEmbedContents?key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var batchResp geminiBatchEmbedResponse
	if err := json.Unmarshal(respBody, &batchResp); err != nil {
		return nil, fmt.Errorf("unmarshal embed response: %w", err)
	}
	if len(batchResp.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("embed response count mismatch: got %d embeddings for %d inputs", len(batchResp.Embeddings), len(inputs))
	}

	data := make([]model.EmbeddingData, 0, len(batchResp.Embeddings))
	for i, emb := range batchResp.Embeddings {
		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Embedding: emb.Values,
			Index:     i,
		})
	}

	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: model.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (p *GoogleProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	gemReq := translateRequest(req)

	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	ch := make(chan provider.StreamEvent, 64)
	chatID := "chatcmpl-" + uuid.New().String()[:8]
	createdTime := time.Now().Unix()

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		var lastUsage *model.UsageInfo

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var gemResp geminiResponse
			if err := json.Unmarshal([]byte(data), &gemResp); err != nil {
				continue
			}

			if gemResp.UsageMetadata != nil {
				lastUsage = translateUsage(gemResp.UsageMetadata)
			}

			for _, cand := range gemResp.Candidates {
				for _, part := range cand.Content.Parts {
					if part.Text != "" {
						chunk := &model.ChatCompletionChunk{
							ID:      chatID,
							Object:  "chat.completion.chunk",
							Created: createdTime,
							Model:   req.Model,
							Choices: []model.ChatChoice{{
								Index: 0,
								Delta: &model.ChatMessage{Content: part.Text},
							}},
						}
						ch <- provider.StreamEvent{Chunk: chunk}
					}
				}

				if cand.FinishReason != "" && cand.FinishReason != "FINISH_REASON_UNSPECIFIED" {
					reason := mapFinishReason(cand.FinishReason)
					chunk := &model.ChatCompletionChunk{
						ID:      chatID,
						Object:  "chat.completion.chunk",
						Created: createdTime,
						Model:   req.Model,
						Choices: []model.ChatChoice{{
							Index:        0,
							Delta:        &model.ChatMessage{},
							FinishReason: &reason,
						}},
					}
					ch <- provider.StreamEvent{Chunk: chunk}
				}
			}
		}

		ch <- provider.StreamEvent{Done: true, Usage: lastUsage}
	}()

	return ch, nil
}

func (p *GoogleProvider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", p.baseURL, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}

func buildGeminiEmbedRequest(modelName, text string, dimensions int) geminiEmbedRequest {
	req := geminiEmbedRequest{
		Model: modelName,
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
	}
	if dimensions > 0 {
		req.OutputDimensionality = &dimensions
	}
	return req
}

func normalizeEmbeddingInput(input any) ([]string, error) {
	switch v := input.(type) {
	case string:
		return []string{v}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("input array must contain strings")
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return v, nil
	default:
		return nil, fmt.Errorf("input must be a string or array of strings")
	}
}

func translateRequest(req *model.ChatCompletionRequest) *geminiRequest {
	gemReq := &geminiRequest{
		GenerationConfig: &geminiGenerationCfg{
			Temperature:   req.Temperature,
			TopP:          req.TopP,
			StopSequences: req.Stop,
		},
	}

	if req.MaxTokens != nil {
		gemReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		gemReq.GenerationConfig.MaxOutputTokens = req.MaxCompletionTokens
	}

	if req.ReasoningEffort != "" {
		if level := reasoningToThinkingLevel(req.Model, req.ReasoningEffort); level != "" {
			gemReq.GenerationConfig.ThinkingConfig = &geminiThinkingCfg{ThinkingLevel: level}
		} else {
			budget := reasoningToBudget(req.ReasoningEffort, gemReq.GenerationConfig.MaxOutputTokens)
			gemReq.GenerationConfig.ThinkingConfig = &geminiThinkingCfg{ThinkingBudget: &budget}
		}
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if s, ok := msg.Content.(string); ok {
				gemReq.SystemInstruction = &geminiContent{
					Parts: []geminiPart{{Text: s}},
				}
			}
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []geminiPart
		switch c := msg.Content.(type) {
		case string:
			parts = []geminiPart{{Text: c}}
		case []any:
			parts = translateGeminiParts(c)
		case []map[string]any:
			rawParts := make([]any, 0, len(c))
			for _, part := range c {
				rawParts = append(rawParts, part)
			}
			parts = translateGeminiParts(rawParts)
		default:
			// For complex content, serialize to string
			b, _ := json.Marshal(c)
			parts = []geminiPart{{Text: string(b)}}
		}

		gemReq.Contents = append(gemReq.Contents, geminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	// Cross-provider prefill: Gemini continues from a trailing "model" turn,
	// matching Anthropic's assistant-prefill behavior.
	if req.Prefill != "" {
		n := len(gemReq.Contents)
		if n == 0 || gemReq.Contents[n-1].Role != "model" {
			gemReq.Contents = append(gemReq.Contents, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: req.Prefill}},
			})
		}
	}

	// Translate tools
	if len(req.Tools) > 0 {
		var funcDecls []geminiFuncDecl
		for _, tool := range req.Tools {
			if tool.Function == nil {
				continue
			}
			funcDecls = append(funcDecls, geminiFuncDecl{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		if len(funcDecls) > 0 {
			gemReq.Tools = []geminiToolDecl{{FunctionDeclarations: funcDecls}}
		}
	}

	return gemReq
}

func translateGeminiParts(rawParts []any) []geminiPart {
	parts := make([]geminiPart, 0, len(rawParts))
	for _, raw := range rawParts {
		block, ok := raw.(map[string]any)
		if !ok {
			b, _ := json.Marshal(raw)
			parts = append(parts, geminiPart{Text: string(b)})
			continue
		}
		switch blockType, _ := block["type"].(string); blockType {
		case "text":
			if text, ok := block["text"].(string); ok {
				parts = append(parts, geminiPart{Text: text})
				continue
			}
		case "image_url":
			if part, ok := geminiPartFromOpenAIImageURL(block); ok {
				parts = append(parts, part)
				continue
			}
		}
		b, _ := json.Marshal(block)
		parts = append(parts, geminiPart{Text: string(b)})
	}
	return parts
}

func geminiPartFromOpenAIImageURL(block map[string]any) (geminiPart, bool) {
	var rawURL string
	switch v := block["image_url"].(type) {
	case string:
		rawURL = v
	case map[string]any:
		rawURL, _ = v["url"].(string)
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return geminiPart{}, false
	}
	if strings.HasPrefix(rawURL, "data:") {
		mediaType, data, ok := parseGeminiDataURL(rawURL)
		if !ok {
			return geminiPart{}, false
		}
		return geminiPart{InlineData: &geminiBlob{MimeType: mediaType, Data: data}}, true
	}
	return geminiPart{FileData: &geminiFileData{FileURI: rawURL}}, true
}

func parseGeminiDataURL(rawURL string) (string, string, bool) {
	header, data, ok := strings.Cut(rawURL, ",")
	if !ok || !strings.Contains(header, ";base64") {
		return "", "", false
	}
	mediaType := strings.TrimPrefix(strings.TrimSuffix(header, ";base64"), "data:")
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	return mediaType, data, true
}

func translateResponse(resp *geminiResponse, requestModel string) *model.ChatCompletionResponse {
	var textContent string
	var toolCalls []model.ToolCall
	finishReason := "stop"

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				textContent += part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, model.ToolCall{
					ID:   "call_" + uuid.New().String()[:8],
					Type: "function",
					Function: model.ToolCallFunc{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		finishReason = mapFinishReason(cand.FinishReason)
	}

	message := &model.ChatMessage{
		Role:    "assistant",
		Content: textContent,
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	var usage *model.UsageInfo
	if resp.UsageMetadata != nil {
		usage = translateUsage(resp.UsageMetadata)
	}

	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []model.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: &finishReason,
		}},
		Usage: usage,
	}
}

func reasoningToBudget(effort string, maxOutputTokens *int) int {
	clamp := func(budget int) int {
		if maxOutputTokens != nil && *maxOutputTokens >= 0 && budget > *maxOutputTokens {
			return *maxOutputTokens
		}
		return budget
	}

	switch effort {
	case "none":
		return 0
	case "low":
		return clamp(1024)
	case "medium":
		return clamp(8192)
	case "high":
		return clamp(32768)
	default:
		return 0
	}
}

func reasoningToThinkingLevel(modelID, effort string) string {
	if !strings.Contains(strings.ToLower(modelID), "gemini-3.5-flash") {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "low":
		return "LOW"
	case "medium", "auto":
		return "MEDIUM"
	case "high":
		return "HIGH"
	default:
		return ""
	}
}

func translateUsage(usage *geminiUsage) *model.UsageInfo {
	if usage == nil {
		return nil
	}

	completionTokens := usage.CandidatesTokenCount + usage.ThoughtsTokenCount
	totalTokens := usage.TotalTokenCount
	if totalTokens == 0 {
		totalTokens = usage.PromptTokenCount + completionTokens
	}

	return &model.UsageInfo{
		PromptTokens:     usage.PromptTokenCount, // already includes cached tokens
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		// Informational only: Gemini 2.5 implicit caching reports the cached
		// subset here. PromptTokenCount already includes it, so this does not
		// affect billing — the implicit-cache discount accrues to us.
		CacheReadTokens: usage.CachedContentTokenCount,
	}
}

func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return "stop"
	}
}
