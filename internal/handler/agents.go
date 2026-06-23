package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/agent"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
)

// AgentsHandler serves the user-built agents feature: CRUD, data sources, and runs.
type AgentsHandler struct {
	q      *queries.AgentQueries
	engine *agent.Engine
}

func NewAgentsHandler(q *queries.AgentQueries, engine *agent.Engine) *AgentsHandler {
	return &AgentsHandler{q: q, engine: engine}
}

func agentUserID(ctx *fasthttp.RequestCtx) string {
	id, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	return id
}

// HandlePresets serves GET /v1/agents/presets and the tool catalogue.
func (h *AgentsHandler) HandlePresets(ctx *fasthttp.RequestCtx) {
	WriteJSONPublic(ctx, 200, map[string]any{
		"object":  "list",
		"presets": agent.Presets,
		"tools":   model.BuiltinTools,
	})
}

// HandleList serves GET /v1/agents.
func (h *AgentsHandler) HandleList(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	if uid == "" {
		writeError(ctx, 401, "unauthorized", "login required")
		return
	}
	agents, err := h.q.ListByUser(ctx, uid)
	if err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "count": len(agents), "agents": agents})
}

// HandleCreate serves POST /v1/agents.
func (h *AgentsHandler) HandleCreate(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	if uid == "" {
		writeError(ctx, 401, "unauthorized", "login required")
		return
	}
	var req model.CreateAgentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(ctx, 400, "invalid_request", "name required")
		return
	}
	if req.Preset != "" {
		if p := agent.PresetByKey(req.Preset); p != nil {
			if req.SystemPrompt == "" {
				req.SystemPrompt = p.SystemPrompt
			}
			if req.Model == "" {
				req.Model = p.Model
			}
			if len(req.Config.Tools) == 0 {
				req.Config = p.Config
			}
		}
	}
	a, err := h.q.Create(ctx, uid, req)
	if err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, a)
}

// HandleGet serves GET /v1/agents/{id} including data sources.
func (h *AgentsHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.Get(ctx, uid, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	a.Sources, _ = h.q.ListSources(ctx, a.ID)
	writeJSON(ctx, 200, a)
}

// HandleUpdate serves PATCH /v1/agents/{id}.
func (h *AgentsHandler) HandleUpdate(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	var req model.UpdateAgentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	a, err := h.q.Update(ctx, uid, id, req)
	if err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, a)
}

// HandleDelete serves DELETE /v1/agents/{id}.
func (h *AgentsHandler) HandleDelete(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	if err := h.q.Delete(ctx, uid, id); err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"deleted": true})
}

// HandleListSources serves GET /v1/agents/{id}/sources.
func (h *AgentsHandler) HandleListSources(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	if _, err := h.q.Get(ctx, uid, id); err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	sources, _ := h.q.ListSources(ctx, id)
	writeJSON(ctx, 200, map[string]any{"object": "list", "sources": sources})
}

// HandleCreateSource serves POST /v1/agents/{id}/sources for text/database/url sources.
// Body: {kind, name, content?, dsn?, driver?, url?}
func (h *AgentsHandler) HandleCreateSource(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.Get(ctx, uid, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	var req struct {
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		Content string `json:"content"`
		DSN     string `json:"dsn"`
		Driver  string `json:"driver"`
		URL     string `json:"url"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	if req.Kind == "" {
		req.Kind = "document"
	}
	meta := map[string]any{}
	switch req.Kind {
	case "database":
		if req.DSN == "" {
			writeError(ctx, 400, "invalid_request", "dsn required for database source")
			return
		}
		meta["dsn"] = req.DSN
		meta["driver"] = req.Driver
	case "url":
		meta["url"] = req.URL
	}
	src, err := h.q.CreateSource(ctx, uid, id, req.Kind, defaultName(req.Name, req.Kind), meta)
	if err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	if req.Kind == "document" && strings.TrimSpace(req.Content) != "" {
		go h.ingest(a.ID, uid, src.ID, src.Name, req.Content)
		_ = h.q.SetSourceStatus(ctx, src.ID, "ingesting", 0)
		src.Status = "ingesting"
	}
	writeJSON(ctx, 200, src)
}

// HandleUploadSource serves POST /v1/agents/{id}/sources/upload (multipart file).
func (h *AgentsHandler) HandleUploadSource(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.Get(ctx, uid, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "invalid multipart: "+err.Error())
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		writeError(ctx, 400, "invalid_request", "file field required")
		return
	}
	fh := files[0]
	f, err := fh.Open()
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 25<<20))
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	md, err := agent.ToMarkdown(fh.Filename, fh.Header.Get("Content-Type"), data)
	if err != nil {
		writeError(ctx, 400, "convert_error", err.Error())
		return
	}
	src, err := h.q.CreateSource(ctx, uid, id, "document", fh.Filename, nil)
	if err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	_ = h.q.SetSourceStatus(ctx, src.ID, "ingesting", 0)
	go h.ingest(a.ID, uid, src.ID, fh.Filename, md)
	src.Status = "ingesting"
	writeJSON(ctx, 200, src)
}

func (h *AgentsHandler) ingest(agentID, userID, sourceID, title, markdown string) {
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	n, err := h.engine.Ingest(bg, userID, agentID, sourceID, title, markdown)
	status := "ready"
	if err != nil {
		status = "error"
	}
	_ = h.q.SetSourceStatus(bg, sourceID, status, n)
}

// HandleDeleteSource serves DELETE /v1/agents/{id}/sources/{sid}.
func (h *AgentsHandler) HandleDeleteSource(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	sid, _ := ctx.UserValue("sid").(string)
	if err := h.q.DeleteSource(ctx, uid, sid); err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"deleted": true})
}

// HandleSearch serves GET /v1/agents/{id}/search?q= (RAG preview).
func (h *AgentsHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	if _, err := h.q.Get(ctx, uid, id); err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	q := strings.TrimSpace(string(ctx.QueryArgs().Peek("q")))
	docs, _ := h.engine.SearchDocs(ctx, id, q, parseLimit(ctx, 8, 20))
	writeJSON(ctx, 200, map[string]any{"object": "list", "query": q, "results": docs})
}

// HandleRuns serves GET /v1/agents/{id}/runs.
func (h *AgentsHandler) HandleRuns(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	if _, err := h.q.Get(ctx, uid, id); err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	runs, _ := h.q.ListRuns(ctx, id, parseLimit(ctx, 20, 100))
	writeJSON(ctx, 200, map[string]any{"object": "list", "runs": runs})
}

// HandleRun serves POST /v1/agents/{id}/run (synchronous).
func (h *AgentsHandler) HandleRun(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.Get(ctx, uid, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	var req model.RunAgentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	run, err := h.engine.Run(ctx, uid, a, req, nil)
	if err != nil {
		writeError(ctx, 500, "run_error", err.Error())
		return
	}
	writeJSON(ctx, 200, run)
}

// HandleRunStream serves POST /v1/agents/{id}/run/stream (SSE step trace).
func (h *AgentsHandler) HandleRunStream(ctx *fasthttp.RequestCtx) {
	uid := agentUserID(ctx)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.Get(ctx, uid, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "agent not found")
		return
	}
	var req model.RunAgentRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	bg := context.Background()
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		emit := func(s model.AgentRunStep) {
			writeSSE(w, "step", s)
			w.Flush()
		}
		run, err := h.engine.Run(bg, uid, a, req, emit)
		if err != nil {
			writeSSE(w, "error", map[string]string{"error": err.Error()})
		} else {
			writeSSE(w, "done", run)
		}
		w.Flush()
	})
}

func defaultName(name, kind string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return kind + " source"
}
