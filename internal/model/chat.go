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
	ResponseFormat      *ResponseFormat `json:"response_format,omitempty"`
	Tools               []Tool          `json:"tools,omitempty"`
	ToolChoice          any             `json:"tool_choice,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
	Thinking            *ThinkingConfig `json:"thinking,omitempty"`
	ChatTemplateKwargs  map[string]any  `json:"chat_template_kwargs,omitempty"`

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
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
	Type         string `json:"type,omitempty"`
	BudgetTokens *int   `json:"budget_tokens,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
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
}

type ModelPricing struct {
	InputPer1M              float64 `json:"input_per_1m_tokens,omitempty"`
	OutputPer1M             float64 `json:"output_per_1m_tokens,omitempty"`
	PerRequest              float64 `json:"per_request,omitempty"`
	PerImage                float64 `json:"per_image,omitempty"`
	PerMegapixel            float64 `json:"per_megapixel,omitempty"`
	FirstMegapixel          float64 `json:"first_megapixel,omitempty"`
	ExtraMegapixel          float64 `json:"extra_megapixel,omitempty"`
	PerInputImage           float64 `json:"per_input_image,omitempty"`
	PerVideo                float64 `json:"per_video,omitempty"`
	PerSecond               float64 `json:"per_second,omitempty"`
	PerSecondWithVideoInput float64 `json:"per_second_with_video_input,omitempty"`
}

type ModelCapabilities struct {
	Streaming bool `json:"streaming"`
	Tools     bool `json:"tools"`
	Vision    bool `json:"vision"`
}
