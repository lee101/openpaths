package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/valyala/fasthttp"
)

var activeResearchJobs int64

func ActiveResearchJobs() int64 { return atomic.LoadInt64(&activeResearchJobs) }

type ResearchHandler struct {
	jobQ   *queries.ResearchJobQueries
	engine *researchEngine
}

func NewResearchHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, jobQ *queries.ResearchJobQueries, exa, papers SearchProviderConfig) *ResearchHandler {
	return &ResearchHandler{
		jobQ: jobQ,
		engine: &researchEngine{
			router:       r,
			billing:      b,
			recorder:     rec,
			search:       newResearchSearchClient(exa, papers),
			jobQ:         jobQ,
			plannerModel: defaultResearchModel,
			extractModel: defaultResearchModel,
			verifyModel:  defaultResearchModel,
			synthModel:   defaultResearchModel,
		},
	}
}

type researchRequest struct {
	Query       string `json:"query"`
	Depth       string `json:"depth"`
	Model       string `json:"model"`
	VerifyModel string `json:"verify_model"`
	SynthModel  string `json:"synth_model"`
	Premium     bool   `json:"premium"`
}

func (h *ResearchHandler) HandleCreate(ctx *fasthttp.RequestCtx) {
	if h.jobQ == nil {
		writeError(ctx, 503, "unavailable", "deep research requires a database")
		return
	}
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req researchRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		writeError(ctx, 400, "invalid_request", "query is required")
		return
	}
	if len(req.Query) > 4000 {
		writeError(ctx, 400, "invalid_request", "query too long")
		return
	}
	depth := req.Depth
	switch depth {
	case "", "standard":
		depth = "standard"
	case "quick", "deep":
	default:
		writeError(ctx, 400, "invalid_request", "depth must be quick, standard, or deep")
		return
	}

	plannerModel := defaultResearchModel
	if strings.TrimSpace(req.Model) != "" {
		plannerModel = strings.TrimSpace(req.Model)
	}

	job, err := h.jobQ.Create(ctx, &queries.ResearchJob{
		ID:       newResearchJobID(),
		UserID:   userID,
		APIKeyID: apiKeyID,
		Query:    req.Query,
		Depth:    depth,
		Model:    plannerModel,
		Status:   "queued",
	})
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to create research job")
		return
	}

	eng := h.engineFor(req, plannerModel)
	go runResearch(eng, job.ID, userID, req.Query, depth)

	writeJSON(ctx, 202, researchJobPayload(job))
}

// engineFor returns an engine configured for this request's model overrides.
func (h *ResearchHandler) engineFor(req researchRequest, plannerModel string) *researchEngine {
	e := *h.engine // shallow copy; search/router/billing shared
	e.plannerModel = plannerModel
	e.extractModel = plannerModel
	e.verifyModel = plannerModel
	e.synthModel = plannerModel
	if req.Premium {
		e.verifyModel = premiumResearchModel
		e.synthModel = premiumResearchModel
	}
	if m := strings.TrimSpace(req.VerifyModel); m != "" {
		e.verifyModel = m
	}
	if m := strings.TrimSpace(req.SynthModel); m != "" {
		e.synthModel = m
	}
	return &e
}

const premiumResearchModel = "claude-sonnet-5"

func runResearch(e *researchEngine, jobID, userID, query, depth string) {
	atomic.AddInt64(&activeResearchJobs, 1)
	defer atomic.AddInt64(&activeResearchJobs, -1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	e.run(ctx, jobID, userID, query, depth)
}

func (h *ResearchHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	if h.jobQ == nil {
		writeError(ctx, 503, "unavailable", "deep research requires a database")
		return
	}
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id is required")
		return
	}
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	job, err := h.jobQ.GetByID(ctx, jobID)
	if err != nil || job.UserID != userID {
		writeError(ctx, 404, "not_found", "research job not found")
		return
	}
	writeJSON(ctx, 200, researchJobPayload(job))
}

func (h *ResearchHandler) HandleList(ctx *fasthttp.RequestCtx) {
	if h.jobQ == nil {
		writeError(ctx, 503, "unavailable", "deep research requires a database")
		return
	}
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	jobs, err := h.jobQ.ListByUser(ctx, userID, 50)
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to list research jobs")
		return
	}
	out := make([]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, researchJobSummary(j))
	}
	writeJSON(ctx, 200, map[string]any{"jobs": out})
}

func researchJobPayload(j *queries.ResearchJob) map[string]any {
	p := researchJobSummary(j)
	p["steps"] = rawOrEmpty(j.Steps, "[]")
	p["report"] = j.Report
	p["citations"] = rawOrEmpty(j.Citations, "[]")
	return p
}

func researchJobSummary(j *queries.ResearchJob) map[string]any {
	p := map[string]any{
		"id":         j.ID,
		"object":     "research.job",
		"query":      j.Query,
		"depth":      j.Depth,
		"model":      j.Model,
		"status":     j.Status,
		"cost_units": j.CostUnits,
		"created_at": j.CreatedAt.Unix(),
		"updated_at": j.UpdatedAt.Unix(),
	}
	if j.ErrorType != nil {
		p["error_type"] = *j.ErrorType
	}
	if j.ErrorMessage != nil {
		p["error"] = *j.ErrorMessage
	}
	if j.FinishedAt != nil {
		p["finished_at"] = j.FinishedAt.Unix()
	}
	return p
}

func rawOrEmpty(r json.RawMessage, fallback string) json.RawMessage {
	if len(r) == 0 {
		return json.RawMessage(fallback)
	}
	return r
}

func newResearchJobID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "research_" + hex.EncodeToString(b)
}
