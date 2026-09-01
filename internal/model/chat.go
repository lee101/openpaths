package model

// ChatCompletionRequest mirrors the OpenAI chat completion request format.
type ChatCompletionRequest struct {
	Model               string          `json:"model"`
	Messages            []ChatMessage   `json:"messages"`
	Temperature         *float64        `json:"temperature,omitempty"`
	TopP                *float64        `json:"top_p,omitempty"`
	N                   *int            `json:"n,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	StreamOptions       *StreamOptions  `json:"stream_options,omitempty"`
	Stop                []string        `json:"stop,omitempty"`
	MaxTokens           *int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int            `json:"max_completion_tokens,omitempty"`
	PresencePenalty     *float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float64        `json:"frequency_penalty,omitempty"`
	User                string          `json:"user,omitempty"`
	SafetyIdentifier    string          `json:"safety_identifier,omitempty"`
	ResponseFormat      *ResponseFormat `json:"response_format,omitempty"`
	Tools               []Tool          `json:"tools,omitempty"`
	ToolChoice          any             `json:"tool_choice,omitempty"`
	ToolStream          bool            `json:"tool_stream,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
	Thinking            *ThinkingConfig `json:"thinking,omitempty"`
	ChatTemplateKwargs  map[string]any  `json:"chat_template_kwargs,omitempty"`
	Fusion              *FusionConfig   `json:"fusion,omitempty"`

	// Prefill is a non-standard cross-provider hint: the assistant response
	// should start with this exact string. Anthropic and Google Gemini support
	// this natively (trailing assistant turn); OpenAI does not and ignores it.
	// Prefer setting this field over manually appending a trailing assistant
	// message so providers can translate it correctly.
	Prefill string `json:"prefill,omitempty"`

	// TaskTier is a non-standard cross-provider hint for routers: "easy" for
	// cheap/fast paths (nano, flash-lite), "hard" for reasoning-heavy work
	// (Opus, flagship). Providers and model selectors MAY use this as input;
	// the direct provider HTTP layer ignores it.
	TaskTier string `json:"task_tier,omitempty"`

	// RoutingStrategy controls how OpenPaths orders healthy candidates after the
	// model/auto router has built a fallback chain. Supported values are:
	// "price" (default, cheapest blended token rate first), "config" (catalogue
	// order), and "fastest" (currently catalogue order unless a caller supplies
	// an explicit low-latency route such as openpaths/auto-fast).
	RoutingStrategy string `json:"routing_strategy,omitempty"`

	// PromptCacheKey is OpenAI's optional cache-routing hint: requests sharing a
	// key are routed together to raise the prompt-cache hit rate. Only the real
	// OpenAI provider sets/forwards it (other OpenAI-compatible upstreams may
	// reject unknown fields), so it is stripped for non-OpenAI endpoints.
	PromptCacheKey string `json:"prompt_cache_key,omitempty"`
}

type ChatMessage struct {
	Role             string     `json:"role"`
	Content          any        `json:"content"`
	Name             string     `json:"name,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Reasoning        string     `json:"reasoning,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

type ChatCompletionResponse struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"`
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	Choices           []ChatChoice `json:"choices"`
	Usage             *UsageInfo   `json:"usage,omitempty"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
}

type ChatChoice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason"`
}

type UsageInfo struct {
	PromptTokens          int                  `json:"prompt_tokens"`
	CompletionTokens      int                  `json:"completion_tokens"`
	TotalTokens           int                  `json:"total_tokens"`
	PromptCacheHitTokens  int                  `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens int                  `json:"prompt_cache_miss_tokens,omitempty"`
	PromptTokensDetails   *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`

	// CacheReadTokens / CacheWriteTokens are informational telemetry of upstream
	// prompt-cache activity (cache hits and writes respectively). They are NOT
	// read by CachedPromptTokens and do NOT affect customer billing — they exist
	// so the cache optimizers can measure realized savings. Keeping them out of
	// billing means the cache discount accrues to us, not the customer.
	CacheReadTokens  int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

func (u *UsageInfo) CachedPromptTokens() int {
	if u == nil {
		return 0
	}
	if u.PromptCacheHitTokens > 0 {
		return u.PromptCacheHitTokens
	}
	if u.PromptTokensDetails != nil {
		return u.PromptTokensDetails.CachedTokens
	}
	return 0
}

type ChatCompletionChunk struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"`
	Created           int64        `json:"created"`
	Model             string       `json:"model"`
	Choices           []ChatChoice `json:"choices"`
	Usage             *UsageInfo   `json:"usage,omitempty"`
	SystemFingerprint string       `json:"system_fingerprint,omitempty"`
}

type ResponseFormat struct {
	Type       string      `json:"type"`
	JsonSchema *JsonSchema `json:"json_schema,omitempty"`
}

type JsonSchema struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Schema      any    `json:"schema,omitempty"`
	Strict      *bool  `json:"strict,omitempty"`
}

type ThinkingConfig struct {
	Type          string `json:"type,omitempty"`
	BudgetTokens  *int   `json:"budget_tokens,omitempty"`
	ClearThinking *bool  `json:"clear_thinking,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type Tool struct {
	Type       string        `json:"type"`
	Function   *ToolFunction `json:"function,omitempty"`
	Parameters any           `json:"parameters,omitempty"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type FusionConfig struct {
	Type                string   `json:"type,omitempty"`
	AnalysisModels      []string `json:"analysis_models,omitempty"`
	Model               string   `json:"model,omitempty"`
	Prompt              string   `json:"prompt,omitempty"`
	MaxPanelModels      int      `json:"max_panel_models,omitempty"`
	MaxCompletionTokens int      `json:"max_completion_tokens,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ErrorResponse is the OpenAI-compatible error format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Code    *string `json:"code,omitempty"`
}

// ModelInfo represents a model in the /v1/models response.
type ModelInfo struct {
	ID              string             `json:"id"`
	Object          string             `json:"object"`
	Created         int64              `json:"created"`
	OwnedBy         string             `json:"owned_by"`
	Pricing         *ModelPricing      `json:"pricing,omitempty"`
	Capabilities    *ModelCapabilities `json:"capabilities,omitempty"`
	ContextWindow   int                `json:"context_window,omitempty"`
	MaxOutputTokens int                `json:"max_output_tokens,omitempty"`
	Aliases         []string           `json:"aliases,omitempty"`
	SupportedSizes  []string           `json:"supported_sizes,omitempty"`
	// Deprecated ids still answer, but on a substitute upstream named in
	// DeprecatedNote. Surfaced so callers can migrate rather than discover it
	// from a changed response.
	Deprecated     bool   `json:"deprecated,omitempty"`
	DeprecatedNote string `json:"deprecated_note,omitempty"`
}

type ModelPricing struct {
	InputPer1M              float64            `json:"input_per_1m_tokens,omitempty"`
	InputCacheHitPer1M      float64            `json:"input_cache_hit_per_1m_tokens,omitempty"`
	OutputPer1M             float64            `json:"output_per_1m_tokens,omitempty"`
	AudioInputPer1M         float64            `json:"audio_input_per_1m_tokens,omitempty"`
	AudioInputCacheHitPer1M float64            `json:"audio_input_cache_hit_per_1m_tokens,omitempty"`
	AudioOutputPer1M        float64            `json:"audio_output_per_1m_tokens,omitempty"`
	ImageInputPer1M         float64            `json:"image_input_per_1m_tokens,omitempty"`
	ImageInputCacheHitPer1M float64            `json:"image_input_cache_hit_per_1m_tokens,omitempty"`
	Per1MCharacters         float64            `json:"per_1m_characters,omitempty"`
	LongContextThreshold    int                `json:"long_context_threshold,omitempty"`
	InputPer1MLong          float64            `json:"input_per_1m_tokens_long,omitempty"`
	InputCacheHitPer1MLong  float64            `json:"input_cache_hit_per_1m_tokens_long,omitempty"`
	OutputPer1MLong         float64            `json:"output_per_1m_tokens_long,omitempty"`
	PerRequest              float64            `json:"per_request,omitempty"`
	PerImage                float64            `json:"per_image,omitempty"`
	PerImageByResolution    map[string]float64 `json:"per_image_by_resolution,omitempty"`
	PerMegapixel            float64            `json:"per_megapixel,omitempty"`
	FirstMegapixel          float64            `json:"first_megapixel,omitempty"`
	ExtraMegapixel          float64            `json:"extra_megapixel,omitempty"`
	PerInputImage           float64            `json:"per_input_image,omitempty"`
	PerVideo                float64            `json:"per_video,omitempty"`
	PerSecond               float64            `json:"per_second,omitempty"`
	PerSecondByResolution   map[string]float64 `json:"per_second_by_resolution,omitempty"`
	PerSecondWithVideoInput float64            `json:"per_second_with_video_input,omitempty"`
	PerInputVideoSecond     float64            `json:"per_input_video_second,omitempty"`
	PerMinute               float64            `json:"per_minute,omitempty"`
	PerHour                 float64            `json:"per_hour,omitempty"`
}

type ModelCapabilities struct {
	Streaming bool `json:"streaming"`
	Tools     bool `json:"tools"`
	Vision    bool `json:"vision"`
}
