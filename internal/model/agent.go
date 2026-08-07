package model

import "time"

// Agent is a user-built agent: a model + system prompt + a set of granted tools
// and connected data sources, executed via a tool-calling loop.
type Agent struct {
	ID           string      `json:"id"`
	UserID       string      `json:"user_id,omitempty"`
	Slug         string      `json:"slug,omitempty"`
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	SystemPrompt string      `json:"system_prompt,omitempty"`
	Model        string      `json:"model"`
	Config       AgentConfig `json:"config"`
	Preset       string      `json:"preset,omitempty"`
	CreatedAt    time.Time   `json:"created_at,omitempty"`
	UpdatedAt    time.Time   `json:"updated_at,omitempty"`

	Sources []AgentDataSource `json:"sources,omitempty"`
}

// AgentConfig holds the behavioural knobs persisted as jsonb.
type AgentConfig struct {
	Tools       []string `json:"tools"`               // enabled built-in tool names
	MaxSteps    int      `json:"max_steps,omitempty"` // tool-loop bound (default 8)
	Temperature *float64 `json:"temperature,omitempty"`
	// ImageModel/VideoModel/EmbedModel let presets pin a sub-model for a tool.
	ImageModel string `json:"image_model,omitempty"`
	VideoModel string `json:"video_model,omitempty"`
}

// AgentDataSource is a connected corpus or database the agent can query.
type AgentDataSource struct {
	ID         string         `json:"id"`
	AgentID    string         `json:"agent_id"`
	UserID     string         `json:"user_id,omitempty"`
	Kind       string         `json:"kind"` // document | database | url
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Meta       map[string]any `json:"meta,omitempty"`
	ChunkCount int            `json:"chunk_count"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
}

// AgentDocument is one embedded chunk of a document data source.
type AgentDocument struct {
	ID           string    `json:"id"`
	DataSourceID string    `json:"data_source_id"`
	AgentID      string    `json:"agent_id"`
	UserID       string    `json:"user_id,omitempty"`
	Title        string    `json:"title,omitempty"`
	ChunkIndex   int       `json:"chunk_index"`
	Content      string    `json:"content"`
	Embedding    []byte    `json:"-"`
	Score        float64   `json:"score,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// AgentRun records one execution of an agent.
type AgentRun struct {
	ID        string         `json:"id"`
	AgentID   string         `json:"agent_id"`
	UserID    string         `json:"user_id,omitempty"`
	Input     string         `json:"input"`
	Output    string         `json:"output"`
	Steps     []AgentRunStep `json:"steps"`
	Status    string         `json:"status"`
	CostCents int64          `json:"cost_cents"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

// AgentRunStep is one entry in a run trace: either a model turn or a tool call.
type AgentRunStep struct {
	Type    string `json:"type"` // "model" | "tool" | "final" | "error"
	Tool    string `json:"tool,omitempty"`
	Input   string `json:"input,omitempty"` // tool args / model text
	Output  string `json:"output,omitempty"`
	Model   string `json:"model,omitempty"`
	Elapsed int64  `json:"elapsed_ms,omitempty"`
}

// CreateAgentRequest / UpdateAgentRequest are the write payloads.
type CreateAgentRequest struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	SystemPrompt string      `json:"system_prompt"`
	Model        string      `json:"model"`
	Config       AgentConfig `json:"config"`
	Preset       string      `json:"preset,omitempty"`
}

type UpdateAgentRequest struct {
	Name         *string      `json:"name,omitempty"`
	Description  *string      `json:"description,omitempty"`
	SystemPrompt *string      `json:"system_prompt,omitempty"`
	Model        *string      `json:"model,omitempty"`
	Config       *AgentConfig `json:"config,omitempty"`
}

// RunAgentRequest is the run payload.
type RunAgentRequest struct {
	Input    string        `json:"input"`
	Messages []ChatMessage `json:"messages,omitempty"` // optional prior turns
	MaxSteps int           `json:"max_steps,omitempty"`
}

// AgentToolSpec describes a built-in tool for the UI / discovery.
type AgentToolSpec struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	Description  string `json:"description"`
	Experimental bool   `json:"experimental,omitempty"`
}

// BuiltinTools is the catalogue of grantable tools.
var BuiltinTools = []AgentToolSpec{
	{Name: "search_documents", Label: "Document search", Description: "Semantic + keyword RAG over the agent's connected documents."},
	{Name: "query_database", Label: "Database query", Description: "Read-only SELECT against a connected SQL database."},
	{Name: "generate_image", Label: "Generate image", Description: "Text-to-image with any OpenPaths image model."},
	{Name: "generate_video", Label: "Generate video", Description: "Text-to-video or image-to-video (Minimax/xAI)."},
	{Name: "call_model", Label: "Call another model", Description: "Invoke any other OpenPaths chat model for a sub-task."},
	{Name: "web_search", Label: "Web search", Description: "Search the live web (Exa); returns titles, URLs, and snippets."},
	{Name: "computer_use", Label: "Computer use", Description: "Drive a sandboxed desktop (Anthropic computer-use beta).", Experimental: true},
}

// AgentPreset is a ready-made agent template surfaced in the gallery.
type AgentPreset struct {
	Key            string      `json:"key"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	SystemPrompt   string      `json:"system_prompt"`
	Model          string      `json:"model"`
	Config         AgentConfig `json:"config"`
	ExamplePrompts []string    `json:"example_prompts,omitempty"`
}
