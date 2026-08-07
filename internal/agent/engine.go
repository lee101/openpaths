package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/modelaccess"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/storage"
)

var errNoEmbedding = errors.New("no embedding returned")

const defaultModel = "claude-sonnet-latest"

// Engine executes agents: a tool-calling loop over the chat router with per-step
// billing, plus document ingest/RAG.
type Engine struct {
	router      *router.Router
	billing     *billing.Engine
	embedder    provider.EmbeddingProvider
	q           *queries.AgentQueries
	store       storage.Store
	computerURL string // optional computer-use sandbox endpoint
	exaKey      string // optional Exa key enabling the web_search tool
	accessQ     modelaccess.Checker
}

func NewEngine(r *router.Router, b *billing.Engine, emb provider.EmbeddingProvider, q *queries.AgentQueries, store storage.Store, computerURL string) *Engine {
	return &Engine{router: r, billing: b, embedder: emb, q: q, store: store, computerURL: computerURL}
}

// SetExaKey enables the web_search tool with the platform Exa API key.
func (e *Engine) SetExaKey(key string) { e.exaKey = strings.TrimSpace(key) }

// SetAccess wires the optional Model IAM policy store (default-open when unset).
func (e *Engine) SetAccess(checker modelaccess.Checker) { e.accessQ = checker }

func (e *Engine) resolveAuthorizedCandidates(ctx context.Context, userID, modelID string) ([]router.RouteCandidate, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" || modelID == "auto" {
		modelID = defaultModel
	}
	candidates, err := e.router.ResolveForRequest(modelID, "")
	if err != nil || len(candidates) == 0 {
		if candidates, err = e.router.ResolveWithRetries(defaultModel); err != nil {
			return nil, fmt.Errorf("resolve agent model: %w", err)
		}
	}
	return modelaccess.FilterCandidates(ctx, e.accessQ, userID, modelID, candidates)
}

// AuthorizeModel lets HTTP handlers reject a denied run before starting a
// streaming response. Run repeats the check at its provider boundary.
func (e *Engine) AuthorizeModel(ctx context.Context, userID, modelID string) error {
	_, err := e.resolveAuthorizedCandidates(ctx, userID, modelID)
	return err
}

// Run executes the agent against the request. emit, if non-nil, receives each
// step as it happens (for SSE streaming).
func (e *Engine) Run(ctx context.Context, userID string, ag *model.Agent, req model.RunAgentRequest, emit func(model.AgentRunStep)) (*model.AgentRun, error) {
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = ag.Config.MaxSteps
	}
	if maxSteps <= 0 || maxSteps > 20 {
		maxSteps = 8
	}

	modelID := strings.TrimSpace(ag.Model)
	if modelID == "" || modelID == "auto" {
		modelID = defaultModel
	}
	candidates, err := e.resolveAuthorizedCandidates(ctx, userID, modelID)
	if err != nil {
		return nil, err
	}

	tools, toolDefs := e.buildTools(userID, ag)

	messages := make([]model.ChatMessage, 0, len(req.Messages)+2)
	if sp := strings.TrimSpace(ag.SystemPrompt); sp != "" {
		messages = append(messages, model.ChatMessage{Role: "system", Content: sp})
	}
	messages = append(messages, req.Messages...)
	if strings.TrimSpace(req.Input) != "" {
		messages = append(messages, model.ChatMessage{Role: "user", Content: req.Input})
	}

	// Agent runs are not behind the BalanceCheck middleware (accountChain has no
	// balance or rate-limit stage) and each step is billed only after it has
	// been served, so without this gate a zero-balance user could burn up to
	// maxSteps full model calls before anything stopped them.
	if e.billing != nil {
		if err := e.billing.PreCheck(ctx, userID, modelID, 1000); err != nil {
			return nil, fmt.Errorf("insufficient credits: %w", err)
		}
	}

	run := &model.AgentRun{AgentID: ag.ID, UserID: userID, Input: req.Input, Status: "completed"}
	record := func(s model.AgentRunStep) {
		run.Steps = append(run.Steps, s)
		if emit != nil {
			emit(s)
		}
	}

	for step := 0; step < maxSteps; step++ {
		// Re-check between steps: a long multi-step run can exhaust a balance
		// that was healthy when the run started.
		if step > 0 && e.billing != nil {
			if err := e.billing.PreCheck(ctx, userID, modelID, 1000); err != nil {
				run.Status = "error"
				record(model.AgentRunStep{Type: "error", Output: "run stopped: insufficient credits"})
				break
			}
		}

		chatReq := &model.ChatCompletionRequest{Model: modelID, Messages: messages, Temperature: ag.Config.Temperature}
		if len(toolDefs) > 0 {
			chatReq.Tools = toolDefs
		}

		start := time.Now()
		resp, perr := callFirstHealthy(ctx, candidates, chatReq)
		if perr != nil {
			run.Status = "error"
			record(model.AgentRunStep{Type: "error", Output: perr.Error(), Elapsed: time.Since(start).Milliseconds()})
			break
		}
		if resp.Usage != nil && e.billing != nil {
			cost, _ := e.billing.Deduct(ctx, userID, resp.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "", "")
			run.CostCents += cost
		}
		if len(resp.Choices) == 0 {
			break
		}
		msg := resp.Choices[0].Message
		if msg == nil {
			break
		}
		messages = append(messages, *msg)

		if len(msg.ToolCalls) == 0 {
			final := contentText(msg.Content)
			run.Output = final
			record(model.AgentRunStep{Type: "final", Model: resp.Model, Output: final, Elapsed: time.Since(start).Milliseconds()})
			break
		}

		record(model.AgentRunStep{Type: "model", Model: resp.Model, Output: contentText(msg.Content), Elapsed: time.Since(start).Milliseconds()})

		for _, tc := range msg.ToolCalls {
			ts := time.Now()
			out := e.execTool(ctx, userID, ag, tools, tc)
			messages = append(messages, model.ChatMessage{Role: "tool", ToolCallID: tc.ID, Name: tc.Function.Name, Content: out})
			record(model.AgentRunStep{Type: "tool", Tool: tc.Function.Name, Input: tc.Function.Arguments, Output: out, Elapsed: time.Since(ts).Milliseconds()})
		}
	}

	if run.Output == "" && run.Status == "completed" {
		run.Output = "(agent stopped without a final answer; raise max steps)"
	}
	if e.q != nil {
		if id, err := e.q.InsertRun(ctx, *run); err == nil {
			run.ID = id
		}
	}
	return run, nil
}

func (e *Engine) execTool(ctx context.Context, userID string, ag *model.Agent, tools map[string]toolImpl, tc model.ToolCall) string {
	t, ok := tools[tc.Function.Name]
	if !ok {
		return "error: unknown tool " + tc.Function.Name
	}
	var args map[string]any
	if s := strings.TrimSpace(tc.Function.Arguments); s != "" {
		_ = json.Unmarshal([]byte(s), &args)
	}
	out, err := t.run(ctx, userID, ag, args)
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

func callFirstHealthy(ctx context.Context, candidates []router.RouteCandidate, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	var lastErr error
	for _, c := range candidates {
		resp, err := c.Provider.ChatCompletion(ctx, req)
		if err == nil {
			if resp.Model == "" {
				resp.Model = c.ModelCfg.ID
			}
			return resp, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no candidates")
	}
	return nil, lastErr
}

func contentText(c any) string {
	switch v := c.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]any); ok {
				if t, _ := m["text"].(string); t != "" {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		return ""
	}
}
