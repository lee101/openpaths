package handler

import (
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/guardrails"
	"github.com/openpaths/openpaths/internal/middleware"
)

type GuardrailHandler struct {
	q      *queries.GuardrailQueries
	apiKey *queries.APIKeyQueries
}

func NewGuardrailHandler(q *queries.GuardrailQueries, apiKeyQ *queries.APIKeyQueries) *GuardrailHandler {
	return &GuardrailHandler{q: q, apiKey: apiKeyQ}
}

func (h *GuardrailHandler) HandleList(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	list, err := h.q.ListByUser(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list guardrails")
		return
	}
	if list == nil {
		list = []*queries.Guardrail{}
	}
	writeJSON(ctx, 200, map[string]any{"guardrails": list})
}

func (h *GuardrailHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	g, err := h.q.Get(ctx, id, userID)
	if err != nil {
		writeError(ctx, 404, "not_found", "Guardrail not found")
		return
	}
	writeJSON(ctx, 200, g)
}

type guardrailBody struct {
	Name             string          `json:"name"`
	LimitCents       *int64          `json:"limit_cents"`
	ResetInterval    *string         `json:"reset_interval"`
	BudgetActions    []string        `json:"budget_actions"`
	AllowedModels    []string        `json:"allowed_models"`
	AllowedProviders []string        `json:"allowed_providers"`
	PromptInjection  json.RawMessage `json:"prompt_injection"`
	SensitiveInfo    json.RawMessage `json:"sensitive_info"`
	CustomFilters    json.RawMessage `json:"custom_filters"`
}

func (h *GuardrailHandler) HandleCreate(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	var req guardrailBody
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if err := validateGuardrailBody(&req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	g := &queries.Guardrail{
		UserID: userID, Name: req.Name,
		LimitCents: req.LimitCents, ResetInterval: req.ResetInterval,
		BudgetActions: req.BudgetActions, AllowedModels: req.AllowedModels, AllowedProviders: req.AllowedProviders,
		PromptInjection: req.PromptInjection, SensitiveInfo: req.SensitiveInfo, CustomFilters: req.CustomFilters,
	}
	out, err := h.q.Create(ctx, g)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to create guardrail")
		return
	}
	writeJSON(ctx, 201, out)
}

func (h *GuardrailHandler) HandleUpdate(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	var req guardrailBody
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if err := validateGuardrailBody(&req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	g := &queries.Guardrail{
		ID: id, UserID: userID, Name: req.Name,
		LimitCents: req.LimitCents, ResetInterval: req.ResetInterval,
		BudgetActions: req.BudgetActions, AllowedModels: req.AllowedModels, AllowedProviders: req.AllowedProviders,
		PromptInjection: req.PromptInjection, SensitiveInfo: req.SensitiveInfo, CustomFilters: req.CustomFilters,
	}
	out, err := h.q.Update(ctx, g)
	if err != nil {
		writeError(ctx, 404, "not_found", "Guardrail not found")
		return
	}
	as, _ := h.q.ListAssignments(ctx, out.ID)
	out.Assignments = as
	writeJSON(ctx, 200, out)
}

func (h *GuardrailHandler) HandleDelete(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	if err := h.q.Delete(ctx, id, userID); err != nil {
		writeError(ctx, 404, "not_found", "Guardrail not found")
		return
	}
	writeJSON(ctx, 200, map[string]any{"ok": true})
}

func (h *GuardrailHandler) HandleAssignments(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	var req struct {
		APIKeyIDs   []string `json:"api_key_ids"`
		UserDefault bool     `json:"user_default"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if err := h.q.ReplaceAssignments(ctx, id, userID, req.APIKeyIDs, req.UserDefault); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	g, err := h.q.Get(ctx, id, userID)
	if err != nil {
		writeError(ctx, 404, "not_found", "Guardrail not found")
		return
	}
	writeJSON(ctx, 200, g)
}

func (h *GuardrailHandler) HandleEvents(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	events, err := h.q.ListEvents(ctx, userID, 50)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list events")
		return
	}
	writeJSON(ctx, 200, map[string]any{"events": events})
}

func validateGuardrailBody(req *guardrailBody) error {
	if req.ResetInterval != nil {
		v := strings.ToLower(strings.TrimSpace(*req.ResetInterval))
		if v == "" {
			req.ResetInterval = nil
		} else if v != "daily" && v != "weekly" && v != "monthly" {
			return errString("reset_interval must be daily, weekly, or monthly")
		} else {
			req.ResetInterval = &v
		}
	}
	if req.LimitCents != nil && *req.LimitCents < 0 {
		return errString("limit_cents must be >= 0")
	}
	for _, a := range req.BudgetActions {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != guardrails.ActionBlock && a != guardrails.ActionEmail {
			return errString("budget_actions may only include block, email")
		}
	}
	pi := guardrails.ParsePromptInjection(req.PromptInjection)
	for _, p := range pi.Patterns {
		if err := guardrails.ValidateRegex(p); err != nil {
			return errString("invalid prompt injection pattern: " + err.Error())
		}
	}
	si := guardrails.ParseSensitiveInfo(req.SensitiveInfo)
	for _, f := range si.Filters {
		switch f.Slug {
		case "email", "phone", "ssn", "credit-card", "ip-address":
		default:
			return errString("unknown sensitive info slug: " + f.Slug)
		}
		switch f.Action {
		case guardrails.ActionBlock, guardrails.ActionRedact, guardrails.ActionEmail:
		default:
			return errString("sensitive info action must be block, redact, or email")
		}
	}
	for _, cf := range guardrails.ParseCustomFilters(req.CustomFilters) {
		if err := guardrails.ValidateRegex(cf.Pattern); err != nil {
			return errString("invalid custom filter: " + err.Error())
		}
		switch cf.Action {
		case guardrails.ActionBlock, guardrails.ActionRedact, guardrails.ActionEmail, "":
		default:
			return errString("custom filter action must be block, redact, or email")
		}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }
