package agent

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type toolRunFunc func(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error)

type toolImpl struct {
	def model.Tool
	run toolRunFunc
}

func fnTool(name, desc string, params map[string]any, run toolRunFunc) toolImpl {
	return toolImpl{
		def: model.Tool{Type: "function", Function: &model.ToolFunction{Name: name, Description: desc, Parameters: params}},
		run: run,
	}
}

func obj(props map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

// buildTools returns the enabled tool implementations and their chat definitions.
func (e *Engine) buildTools(userID string, ag *model.Agent) (map[string]toolImpl, []model.Tool) {
	all := map[string]toolImpl{
		"search_documents": fnTool("search_documents",
			"Search the agent's connected documents for relevant passages.",
			obj(map[string]any{"query": strProp("what to look for")}, "query"),
			e.toolSearchDocuments),
		"query_database": fnTool("query_database",
			"Run a read-only SQL SELECT against a connected database. Returns rows as text.",
			obj(map[string]any{
				"sql":    strProp("a single read-only SELECT statement"),
				"source": strProp("optional data source name to target"),
			}, "sql"),
			e.toolQueryDatabase),
		"generate_image": fnTool("generate_image",
			"Generate an image from a text prompt. Returns the image URL.",
			obj(map[string]any{
				"prompt": strProp("the image description"),
				"size":   strProp("optional, e.g. 1024x1024"),
			}, "prompt"),
			e.toolGenerateImage),
		"generate_video": fnTool("generate_video",
			"Generate a video from a text prompt and optional source image URL (image-to-video). Returns a job id or URL.",
			obj(map[string]any{
				"prompt":    strProp("the video description"),
				"image_url": strProp("optional source image to animate"),
			}, "prompt"),
			e.toolGenerateVideo),
		"call_model": fnTool("call_model",
			"Ask another OpenPaths chat model to complete a sub-task. Returns its answer.",
			obj(map[string]any{
				"model":  strProp("model id, e.g. gpt-4o or claude-opus-4-8"),
				"prompt": strProp("the sub-task prompt"),
			}, "model", "prompt"),
			e.toolCallModel),
		"computer_use": fnTool("computer_use",
			"Perform an action on a sandboxed desktop (screenshot, click, type).",
			obj(map[string]any{"action": strProp("action description")}, "action"),
			e.toolComputerUse),
	}

	enabled := map[string]toolImpl{}
	var defs []model.Tool
	for _, name := range ag.Config.Tools {
		if t, ok := all[name]; ok {
			enabled[name] = t
			defs = append(defs, t.def)
		}
	}
	return enabled, defs
}

func (e *Engine) toolSearchDocuments(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	query := strArg(args, "query")
	if query == "" {
		return "", fmt.Errorf("query required")
	}
	docs, err := e.SearchDocs(ctx, ag.ID, query, 6)
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "No relevant passages found in connected documents.", nil
	}
	var b strings.Builder
	for i, d := range docs {
		fmt.Fprintf(&b, "[%d] %s (chunk %d)\n%s\n\n", i+1, d.Title, d.ChunkIndex, d.Content)
	}
	return b.String(), nil
}

func (e *Engine) toolQueryDatabase(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	q := strArg(args, "sql")
	if !isReadOnlySQL(q) {
		return "", fmt.Errorf("only single read-only SELECT statements are allowed")
	}
	sources, err := e.q.ListSources(ctx, ag.ID)
	if err != nil {
		return "", err
	}
	want := strArg(args, "source")
	var dsn, driver string
	for _, s := range sources {
		if s.Kind != "database" {
			continue
		}
		if want != "" && !strings.EqualFold(s.Name, want) {
			continue
		}
		dsn, _ = s.Meta["dsn"].(string)
		driver, _ = s.Meta["driver"].(string)
		break
	}
	if dsn == "" {
		return "", fmt.Errorf("no connected database source found")
	}
	if driver == "" {
		driver = "pgx"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return "", fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	rows, err := db.QueryContext(cctx, q)
	if err != nil {
		return "", fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	return rowsToText(rows, 100)
}

func (e *Engine) toolGenerateImage(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	prompt := strArg(args, "prompt")
	if prompt == "" {
		return "", fmt.Errorf("prompt required")
	}
	modelID := firstNonEmpty(ag.Config.ImageModel, "zimage")
	cands, err := e.router.ResolveWithRetries(modelID)
	if err != nil {
		return "", fmt.Errorf("resolve image model: %w", err)
	}
	for _, c := range cands {
		ip, ok := c.Provider.(provider.ImageProvider)
		if !ok {
			continue
		}
		req := &model.ImageGenerationRequest{Model: c.ModelCfg.ID, Prompt: prompt, N: 1, Size: strArg(args, "size")}
		resp, err := ip.GenerateImage(ctx, req)
		if err != nil {
			continue
		}
		if e.billing != nil {
			cost, _ := e.billing.DeductImageWithInputsAndSize(ctx, userID, c.ModelCfg.ID, 1, 0, strArg(args, "size"), "")
			_ = cost
		}
		if len(resp.Data) > 0 {
			d := resp.Data[0]
			url := d.URL
			if url == "" && d.B64JSON != "" && e.store != nil {
				if u, err := e.uploadB64(ctx, d.B64JSON); err == nil {
					url = u
				}
			}
			return fmt.Sprintf("Generated image: %s", url), nil
		}
	}
	return "", fmt.Errorf("no image provider produced a result for %q", modelID)
}

func (e *Engine) toolGenerateVideo(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	prompt := strArg(args, "prompt")
	if prompt == "" {
		return "", fmt.Errorf("prompt required")
	}
	imageURL := strArg(args, "image_url")
	modelID := ag.Config.VideoModel
	if modelID == "" {
		if imageURL != "" {
			modelID = "seedance-2.0-image-to-video"
		} else {
			modelID = "seedance-2.0-fast-text-to-video"
		}
	}
	cands, err := e.router.ResolveWithRetries(modelID)
	if err != nil {
		return "", fmt.Errorf("resolve video model: %w", err)
	}
	for _, c := range cands {
		vp, ok := c.Provider.(provider.VideoProvider)
		if !ok {
			continue
		}
		req := &model.VideoGenerationRequest{Model: c.ModelCfg.ID, Prompt: prompt, ImageURL: imageURL, Operation: "generation"}
		resp, err := vp.GenerateVideo(ctx, req)
		if err != nil {
			continue
		}
		if resp.VideoURL != "" {
			return "Generated video: " + resp.VideoURL, nil
		}
		return "Video generation accepted (no URL returned synchronously).", nil
	}
	return "", fmt.Errorf("no video provider produced a result for %q", modelID)
}

func (e *Engine) toolCallModel(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	modelID := strArg(args, "model")
	prompt := strArg(args, "prompt")
	if modelID == "" || prompt == "" {
		return "", fmt.Errorf("model and prompt required")
	}
	cands, err := e.router.ResolveWithRetries(modelID)
	if err != nil {
		return "", err
	}
	resp, err := callFirstHealthy(ctx, cands, &model.ChatCompletionRequest{
		Model: modelID, Messages: []model.ChatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	if e.billing != nil && resp.Usage != nil {
		cost, _ := e.billing.Deduct(ctx, userID, resp.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "", "")
		_ = cost
	}
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		return contentText(resp.Choices[0].Message.Content), nil
	}
	return "", fmt.Errorf("empty response from %s", modelID)
}

func (e *Engine) toolComputerUse(ctx context.Context, userID string, ag *model.Agent, args map[string]any) (string, error) {
	if e.computerURL == "" {
		return "Computer use is not configured. Set OPENPATHS_COMPUTER_USE_URL to a sandbox endpoint to enable screenshot/click/type actions.", nil
	}
	return "", fmt.Errorf("computer-use sandbox dispatch not yet implemented for endpoint %s", e.computerURL)
}

func (e *Engine) uploadB64(ctx context.Context, b64 string) (string, error) {
	data, err := decodeB64(b64)
	if err != nil {
		return "", err
	}
	return e.store.Upload(ctx, fmt.Sprintf("agent-image-%d.png", time.Now().UnixNano()), "image/png", bytes.NewReader(data))
}
