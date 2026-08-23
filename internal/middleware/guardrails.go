package middleware

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/guardrails"
	"github.com/openpaths/openpaths/internal/model"
)

const (
	CtxKeyGuardrailProviders        = "guardrail_providers"
	CtxKeyGuardrailBlockedProviders = "guardrail_blocked_providers"
	CtxKeyGuardrailEmailOnViolation = "guardrail_email_on_violation"
	CtxKeyGuardrailUserQueries      = "guardrail_user_queries"
)

type GuardrailDeps struct {
	Q     *queries.GuardrailQueries
	UserQ *queries.UserQueries
}

// GuardrailCheck enforces budget, model/provider access rules, and content filters.
func GuardrailCheck(deps GuardrailDeps) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if deps.Q == nil {
				next(ctx)
				return
			}
			userID, _ := ctx.UserValue(CtxKeyUserID).(string)
			if userID == "" {
				next(ctx)
				return
			}
			apiKey, _ := ctx.UserValue(CtxKeyAPIKey).(*model.APIKey)
			apiKeyID := ""
			if apiKey != nil {
				apiKeyID = apiKey.ID
			}

			policies, err := deps.Q.ResolveForRequest(ctx, userID, apiKeyID)
			if err != nil || len(policies) == 0 {
				next(ctx)
				return
			}
			for _, g := range policies {
				if g.EmailOnViolation {
					ctx.SetUserValue(CtxKeyGuardrailEmailOnViolation, true)
					ctx.SetUserValue(CtxKeyGuardrailUserQueries, deps.UserQ)
					break
				}
			}

			now := time.Now().UTC()
			for _, g := range policies {
				if g.LimitCents == nil || *g.LimitCents <= 0 || g.ResetInterval == nil || *g.ResetInterval == "" {
					continue
				}
				since := guardrails.PeriodStart(*g.ResetInterval, now)
				// Key assignment budgets are per-key; user-default budgets are per-user.
				perKey := false
				for _, a := range g.Assignments {
					if a.TargetType == "api_key" && a.TargetID == apiKeyID {
						perKey = true
						break
					}
				}
				// Assignments may not be loaded on Resolve — check via target resolution again.
				if !perKey && apiKeyID != "" {
					// Heuristic: if this policy was resolved via key assignment it is first when key has one.
					// ResolveForRequest order: key then user. Prefer per-key spend when limit is on a key-assigned policy.
					as, _ := deps.Q.ListAssignments(ctx, g.ID)
					g.Assignments = as
					for _, a := range as {
						if a.TargetType == "api_key" && a.TargetID == apiKeyID {
							perKey = true
							break
						}
					}
				}
				spend, err := deps.Q.SpendSince(ctx, userID, apiKeyID, since, perKey)
				if err != nil {
					log.Printf("guardrail spend check: %v", err)
					continue
				}
				if spend < *g.LimitCents {
					continue
				}
				detail, _ := json.Marshal(map[string]any{
					"limit_cents": *g.LimitCents,
					"spend_cents": spend,
					"interval":    *g.ResetInterval,
					"per_key":     perKey,
				})
				gid := g.ID
				if guardrails.HasAction(g.BudgetActions, guardrails.ActionEmail) {
					already, _ := deps.Q.HasBudgetEmailSince(ctx, g.ID, since)
					if !already {
						_ = deps.Q.InsertEvent(ctx, &queries.GuardrailEvent{
							GuardrailID: &gid, UserID: userID, APIKeyID: strPtr(apiKeyID),
							Stage: guardrails.StageBudget, Action: guardrails.ActionEmail, Detail: detail,
						})
						go notifyGuardrail(deps.UserQ, userID, "OpenPaths guardrail: credit limit reached",
							"Your guardrail \""+g.Name+"\" has reached its credit limit. Further requests may be blocked until the reset window.")
					}
				}
				if guardrails.HasAction(g.BudgetActions, guardrails.ActionBlock) || len(g.BudgetActions) == 0 {
					_ = deps.Q.InsertEvent(ctx, &queries.GuardrailEvent{
						GuardrailID: &gid, UserID: userID, APIKeyID: strPtr(apiKeyID),
						Stage: guardrails.StageBudget, Action: guardrails.ActionBlock, Detail: detail,
					})
					writeGuardrailError(ctx, "budget_exceeded", "Request blocked: guardrail credit limit reached", map[string]any{
						"guardrail_id": g.ID, "limit_cents": *g.LimitCents, "spend_cents": spend,
					})
					return
				}
			}

			modelID := requestModel(ctx)
			if modelID != "" {
				if hit, providers := guardrails.EvaluateAccess(policies, modelID); hit != nil {
					NotifyGuardrailAccessViolation(ctx, "OpenPaths guardrail: model/provider access blocked", hit.Message)
					gid := hit.Guardrail
					detail, _ := json.Marshal(hit)
					_ = deps.Q.InsertEvent(ctx, &queries.GuardrailEvent{
						GuardrailID: &gid, UserID: userID, APIKeyID: strPtr(apiKeyID),
						Stage: hit.Stage, Action: hit.Action, Detail: detail,
					})
					writeGuardrailError(ctx, "model_not_allowed", hit.Message, map[string]any{"guardrail_id": hit.Guardrail, "model": modelID})
					return
				} else {
					if len(providers) > 0 {
						ctx.SetUserValue(CtxKeyGuardrailProviders, providers)
					}
					if blocked := guardrails.BlockedProviders(policies); len(blocked) > 0 {
						ctx.SetUserValue(CtxKeyGuardrailBlockedProviders, blocked)
					}
				}
			} else {
				_, providers := guardrails.EvaluateAccess(policies, "")
				if len(providers) > 0 {
					ctx.SetUserValue(CtxKeyGuardrailProviders, providers)
				}
				if blocked := guardrails.BlockedProviders(policies); len(blocked) > 0 {
					ctx.SetUserValue(CtxKeyGuardrailBlockedProviders, blocked)
				}
			}

			body := ctx.PostBody()
			if len(body) == 0 {
				next(ctx)
				return
			}
			text := guardrails.ExtractText(body)
			content := guardrails.EvaluateContent(policies, text)
			for _, hit := range content.Hits {
				gid := hit.Guardrail
				detail, _ := json.Marshal(hit)
				_ = deps.Q.InsertEvent(ctx, &queries.GuardrailEvent{
					GuardrailID: &gid, UserID: userID, APIKeyID: strPtr(apiKeyID),
					Stage: hit.Stage, Action: hit.Action, Detail: detail,
				})
				if hit.Action == guardrails.ActionEmail {
					go notifyGuardrail(deps.UserQ, userID, "OpenPaths guardrail: "+hit.Stage,
						hit.Message+" (guardrail: "+hit.Name+")")
				}
			}
			if content.Blocked {
				msg := "Request blocked by guardrail"
				code := "content_blocked"
				meta := map[string]any{}
				for _, hit := range content.Hits {
					if hit.ShouldStop {
						msg = hit.Message
						code = hit.Stage
						meta["patterns"] = hit.Patterns
						meta["slug"] = hit.Slug
						meta["guardrail_id"] = hit.Guardrail
						break
					}
				}
				writeGuardrailError(ctx, code, msg, meta)
				return
			}
			if content.TextChanged {
				ctx.Request.SetBody(guardrails.ApplyPolicyRedactions(body, policies))
			}

			next(ctx)
		}
	}
}

func GuardrailProviders(ctx *fasthttp.RequestCtx) []string {
	v, _ := ctx.UserValue(CtxKeyGuardrailProviders).([]string)
	return v
}

func GuardrailBlockedProviders(ctx *fasthttp.RequestCtx) []string {
	v, _ := ctx.UserValue(CtxKeyGuardrailBlockedProviders).([]string)
	return v
}

// NotifyGuardrailAccessViolation sends an optional access-rule alert. The
// middleware stores the account's user-query handle only for requests that
// opted into these alerts, so normal requests do not perform email work.
func NotifyGuardrailAccessViolation(ctx *fasthttp.RequestCtx, subject, body string) {
	enabled, _ := ctx.UserValue(CtxKeyGuardrailEmailOnViolation).(bool)
	if !enabled {
		return
	}
	userID, _ := ctx.UserValue(CtxKeyUserID).(string)
	userQ, _ := ctx.UserValue(CtxKeyGuardrailUserQueries).(*queries.UserQueries)
	go notifyGuardrail(userQ, userID, subject, body)
}

func writeGuardrailError(ctx *fasthttp.RequestCtx, code, message string, meta map[string]any) {
	ctx.SetStatusCode(403)
	ctx.SetContentType("application/json")
	payload := map[string]any{
		"error": map[string]any{
			"message":  message,
			"type":     "guardrail_error",
			"code":     code,
			"metadata": meta,
		},
	}
	b, _ := json.Marshal(payload)
	ctx.SetBody(b)
}

func notifyGuardrail(userQ *queries.UserQueries, userID, subject, body string) {
	if userQ == nil {
		return
	}
	u, err := userQ.GetByID(context.Background(), userID)
	if err != nil || u == nil || strings.TrimSpace(u.Email) == "" {
		return
	}
	html := "<p>" + body + "</p><p style=\"color:#666;font-size:12px\">OpenPaths Guardrails</p>"
	if err := email.Send(u.Email, subject, html); err != nil {
		log.Printf("guardrail email: %v", err)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
