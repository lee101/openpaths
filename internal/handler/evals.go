package handler

import (
	"sort"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

// EvalSweeper is the runner surface the evals endpoints need. Declared here as
// an interface because internal/cron already imports this package.
type EvalSweeper interface {
	RunAsync() bool
	Running() bool
}

type EvalsHandler struct {
	evalQ  *queries.EvalQueries
	runner EvalSweeper
}

func NewEvalsHandler(evalQ *queries.EvalQueries, runner EvalSweeper) *EvalsHandler {
	return &EvalsHandler{evalQ: evalQ, runner: runner}
}

// HandleResults serves GET /v1/evals/results — the latest live-eval snapshot
// rendered on /evals. Public by design: it is marketing data.
func (h *EvalsHandler) HandleResults(ctx *fasthttp.RequestCtx) {
	if h.evalQ == nil {
		writeError(ctx, 503, "unavailable", "Live evals are not configured")
		return
	}
	rows, err := h.evalQ.List(ctx)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to load eval results")
		return
	}
	ranAt, _ := h.evalQ.LatestRunAt(ctx)
	writeJSON(ctx, 200, buildEvalSnapshot(rows, ranAt))
}

// HandleRun triggers one sweep. Admin-gated at the route level.
func (h *EvalsHandler) HandleRun(ctx *fasthttp.RequestCtx) {
	if h.runner == nil {
		writeError(ctx, 503, "unavailable", "Live evals are not configured")
		return
	}
	if !h.runner.RunAsync() {
		writeJSON(ctx, 409, map[string]any{"started": false, "detail": "sweep already running"})
		return
	}
	writeJSON(ctx, 202, map[string]any{"started": true})
}

// HandleStatus serves GET /v1/evals/status.
func (h *EvalsHandler) HandleStatus(ctx *fasthttp.RequestCtx) {
	running := h.runner != nil && h.runner.Running()
	writeJSON(ctx, 200, map[string]any{"running": running})
}

type evalSuiteAgg struct {
	AvgScore            float64 `json:"avg_score"`
	PassRate            float64 `json:"pass_rate"`
	Cases               int     `json:"cases"`
	MedianTTFTMs        int     `json:"median_ttft_ms"`
	AvgTPS              float64 `json:"avg_tps"`
	CostPerCaseMicroUSD int64   `json:"cost_per_case_micro_usd"`
}

type evalPerModel struct {
	Model   string                  `json:"model"`
	BySuite map[string]evalSuiteAgg `json:"by_suite"`
	Overall evalSuiteAgg            `json:"overall"`
}

type evalCaseEntry struct {
	Suite   string                      `json:"suite"`
	CaseID  string                      `json:"case_id"`
	Results map[string]model.EvalResult `json:"results"`
}

const evalAutoModel = "openpaths/auto"

func buildEvalSnapshot(rows []model.EvalResult, ranAt *time.Time) map[string]any {
	modelAgg := map[string]*evalPerModel{}
	cases := map[string]*evalCaseEntry{}
	caseOrder := []string{}

	for _, r := range rows {
		pm := modelAgg[r.Model]
		if pm == nil {
			pm = &evalPerModel{Model: r.Model, BySuite: map[string]evalSuiteAgg{}}
			modelAgg[r.Model] = pm
		}
		key := r.Suite + "/" + r.CaseID
		cs := cases[key]
		if cs == nil {
			cs = &evalCaseEntry{Suite: r.Suite, CaseID: r.CaseID, Results: map[string]model.EvalResult{}}
			cases[key] = cs
			caseOrder = append(caseOrder, key)
		}
		cs.Results[r.Model] = r
	}

	type accum struct {
		n       int
		sum     float64
		passed  int
		ttfts   []int
		tpsSum  float64
		costSum int64
	}
	acc := map[string]map[string]*accum{}
	for _, r := range rows {
		bucket, ok := acc[r.Model]
		if !ok {
			bucket = map[string]*accum{}
			acc[r.Model] = bucket
		}
		for _, scope := range [...]string{r.Suite, "__overall__"} {
			a := bucket[scope]
			if a == nil {
				a = &accum{}
				bucket[scope] = a
			}
			a.n++
			a.sum += r.Score
			a.ttfts = append(a.ttfts, r.TTFTMs)
			a.tpsSum += r.TokensPerSec
			a.costSum += r.CostMicroUSD
			if r.Passed {
				a.passed++
			}
		}
	}

	flush := func(bucket map[string]*accum, scope string) evalSuiteAgg {
		a := bucket[scope]
		if a == nil {
			return evalSuiteAgg{}
		}
		sort.Ints(a.ttfts)
		return evalSuiteAgg{
			AvgScore:            evalRound(a.sum/float64(a.n), 4),
			PassRate:            evalRound(float64(a.passed)/float64(a.n), 4),
			Cases:               a.n,
			MedianTTFTMs:        a.ttfts[len(a.ttfts)/2],
			AvgTPS:              evalRound(a.tpsSum/float64(a.n), 1),
			CostPerCaseMicroUSD: a.costSum / int64(max(a.n, 1)),
		}
	}

	outModels := make([]evalPerModel, 0, len(modelAgg))
	for id, pm := range modelAgg {
		bucket := acc[id]
		for _, suite := range []string{string(model.EvalSuiteCoding), string(model.EvalSuiteAgentic), string(model.EvalSuiteSVG)} {
			pm.BySuite[suite] = flush(bucket, suite)
		}
		pm.Overall = flush(bucket, "__overall__")
		outModels = append(outModels, *pm)
	}
	sort.Slice(outModels, func(i, j int) bool { return outModels[i].Model < outModels[j].Model })

	outCases := make([]evalCaseEntry, 0, len(caseOrder))
	for _, key := range caseOrder {
		outCases = append(outCases, *cases[key])
	}

	snapshot := map[string]any{
		"models":       outModels,
		"cases":        outCases,
		"auto_vs_best": computeAutoVsBest(outModels),
	}
	if ranAt != nil {
		snapshot["ran_at"] = *ranAt
	} else {
		snapshot["ran_at"] = nil
	}
	return snapshot
}

func computeAutoVsBest(models []evalPerModel) map[string]any {
	var auto *evalPerModel
	for i := range models {
		if models[i].Model == evalAutoModel {
			auto = &models[i]
		}
	}

	scoreFor := func(m *evalPerModel, scope string) (float64, int64, bool) {
		if scope == "__overall__" {
			return m.Overall.AvgScore, m.Overall.CostPerCaseMicroUSD, true
		}
		s, ok := m.BySuite[scope]
		if !ok || s.Cases == 0 {
			return 0, 0, false
		}
		return s.AvgScore, s.CostPerCaseMicroUSD, true
	}

	out := map[string]any{}
	for _, scope := range []string{"coding", "agentic", "creative", "__overall__"} {
		key := scope
		var best *evalPerModel
		bestScore := -1.0
		bestCost := int64(0)
		for i := range models {
			m := &models[i]
			if m.Model == evalAutoModel {
				continue
			}
			score, cost, ok := scoreFor(m, scope)
			if !ok {
				continue
			}
			if score > bestScore {
				bestScore, bestCost, best = score, cost, m
			}
		}
		entry := map[string]any{
			"best_model": nil,
			"best_score": nil,
			"auto_score": nil,
		}
		if best != nil {
			entry["best_model"] = best.Model
			entry["best_score"] = evalRound(bestScore, 4)
			entry["best_cost_per_case_micro_usd"] = bestCost
		}
		if auto != nil {
			if score, cost, ok := scoreFor(auto, scope); ok {
				entry["auto_score"] = evalRound(score, 4)
				entry["auto_cost_per_case_micro_usd"] = cost
			}
		}
		out[key] = entry
	}
	return out
}

func evalRound(v float64, places int) float64 {
	scale := 1.0
	for i := 0; i < places; i++ {
		scale *= 10
	}
	return float64(int(v*scale+0.5)) / scale
}
