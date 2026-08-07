package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/middleware"
)

// OrgHandler exposes Model IAM rules, billing guards, and team/org management.
// All controls are opt-in and open by default.
type OrgHandler struct {
	accessQ *queries.AccessQueries
	guardQ  *queries.GuardQueries
	teamQ   *queries.TeamQueries
	userQ   *queries.UserQueries
}

func NewOrgHandler(accessQ *queries.AccessQueries, guardQ *queries.GuardQueries, teamQ *queries.TeamQueries, userQ *queries.UserQueries) *OrgHandler {
	return &OrgHandler{accessQ: accessQ, guardQ: guardQ, teamQ: teamQ, userQ: userQ}
}

func (h *OrgHandler) userID(ctx *fasthttp.RequestCtx) string {
	id, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	return id
}

func (h *OrgHandler) isGlobalAdmin(ctx *fasthttp.RequestCtx, userID string) bool {
	u, err := h.userQ.GetByID(ctx, userID)
	return err == nil && u != nil && u.IsAdmin
}

func (h *OrgHandler) canAdminTeam(ctx *fasthttp.RequestCtx, userID, teamID string) bool {
	if h.isGlobalAdmin(ctx, userID) {
		return true
	}
	switch h.teamQ.Role(ctx, userID, teamID) {
	case "owner", "admin":
		return true
	}
	return false
}

func appBaseURL() string {
	if u := os.Getenv("APP_URL"); u != "" {
		return u
	}
	return "https://openpaths.io"
}

// ---- Model IAM rules: /account/model-rules ----

func (h *OrgHandler) HandleModelRules(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required")
		return
	}
	authorize := func(scope, scopeID string) bool {
		switch scope {
		case "user":
			return scopeID == userID || h.isGlobalAdmin(ctx, userID)
		case "team":
			return h.canAdminTeam(ctx, userID, scopeID)
		}
		return false
	}

	switch string(ctx.Method()) {
	case "GET":
		scope := string(ctx.QueryArgs().Peek("scope"))
		scopeID := string(ctx.QueryArgs().Peek("scopeId"))
		if scope == "" {
			scope = "user"
		}
		if scope == "user" && scopeID == "" {
			scopeID = userID
		}
		if !authorize(scope, scopeID) {
			writeError(ctx, 403, "forbidden", "Not authorized for this scope")
			return
		}
		rules, err := h.accessQ.ListRules(ctx, scope, scopeID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Could not load rules")
			return
		}
		writeJSON(ctx, 200, map[string]any{"rules": rules})

	case "POST":
		var req struct{ Scope, ScopeID, ModelGlob, Effect string }
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, 400, "invalid_request", "invalid JSON")
			return
		}
		req.Scope = strings.TrimSpace(req.Scope)
		req.ScopeID = strings.TrimSpace(req.ScopeID)
		req.ModelGlob = strings.TrimSpace(req.ModelGlob)
		req.Effect = strings.ToLower(strings.TrimSpace(req.Effect))
		if req.Scope == "" {
			req.Scope = "user"
		}
		if req.Scope == "user" && req.ScopeID == "" {
			req.ScopeID = userID
		}
		if req.Scope != "user" && req.Scope != "team" {
			writeError(ctx, 400, "invalid_request", "scope must be 'user' or 'team'")
			return
		}
		if req.Effect != "allow" && req.Effect != "deny" {
			writeError(ctx, 400, "invalid_request", "effect must be 'allow' or 'deny'")
			return
		}
		if req.ModelGlob == "" {
			writeError(ctx, 400, "invalid_request", "modelGlob required")
			return
		}
		if !authorize(req.Scope, req.ScopeID) {
			writeError(ctx, 403, "forbidden", "Not authorized for this scope")
			return
		}
		id, err := h.accessQ.AddRule(ctx, req.Scope, req.ScopeID, req.ModelGlob, req.Effect, userID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Could not save rule")
			return
		}
		writeJSON(ctx, 200, map[string]any{"id": id})

	case "DELETE":
		id := string(ctx.QueryArgs().Peek("id"))
		if id == "" {
			writeError(ctx, 400, "invalid_request", "id required")
			return
		}
		scope, scopeID, err := h.accessQ.RuleScope(ctx, id)
		if err != nil {
			writeError(ctx, 404, "not_found", "rule not found")
			return
		}
		if !authorize(scope, scopeID) {
			writeError(ctx, 403, "forbidden", "Not authorized for this scope")
			return
		}
		if err := h.accessQ.DeleteRule(ctx, id); err != nil {
			writeError(ctx, 500, "server_error", "Could not delete rule")
			return
		}
		writeJSON(ctx, 200, map[string]any{"success": true})

	default:
		writeError(ctx, 405, "method_not_allowed", "Method not allowed")
	}
}

// ---- Billing guards: /account/billing-guards ----

func (h *OrgHandler) HandleBillingGuards(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required")
		return
	}
	switch string(ctx.Method()) {
	case "GET":
		if team := string(ctx.QueryArgs().Peek("teamId")); team != "" {
			if !h.canAdminTeam(ctx, userID, team) {
				writeError(ctx, 403, "forbidden", "Not authorized for this team")
				return
			}
		}
		g, err := h.guardQ.GetUserGuards(ctx, userID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Could not load settings")
			return
		}
		writeJSON(ctx, 200, map[string]any{
			"alertEnabled":        g.AlertEnabled,
			"alertThresholdCents": g.AlertThresholdCents,
			"maxTopupCapCents":    g.MaxTopupCapCents,
			"effectiveTopupCap":   h.guardQ.EffectiveTopupCapCents(ctx, userID),
		})

	case "POST":
		var req struct {
			Scope               string `json:"scope"`
			TeamID              string `json:"teamId"`
			AlertEnabled        bool   `json:"alertEnabled"`
			AlertThresholdCents int64  `json:"alertThresholdCents"`
			MaxTopupCapCents    int64  `json:"maxTopupCapCents"`
		}
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, 400, "invalid_request", "invalid JSON")
			return
		}
		if req.AlertThresholdCents < 0 || req.MaxTopupCapCents < 0 {
			writeError(ctx, 400, "invalid_request", "values must be >= 0")
			return
		}
		if req.Scope == "team" {
			if !h.canAdminTeam(ctx, userID, req.TeamID) {
				writeError(ctx, 403, "forbidden", "Not authorized for this team")
				return
			}
			if err := h.guardQ.SetTeamGuards(ctx, req.TeamID, req.AlertThresholdCents, req.MaxTopupCapCents); err != nil {
				writeError(ctx, 500, "server_error", "Could not save team settings")
				return
			}
			writeJSON(ctx, 200, map[string]any{"success": true})
			return
		}
		if err := h.guardQ.SetUserGuards(ctx, userID, queries.UserGuards{
			AlertEnabled:        req.AlertEnabled,
			AlertThresholdCents: req.AlertThresholdCents,
			MaxTopupCapCents:    req.MaxTopupCapCents,
		}); err != nil {
			writeError(ctx, 500, "server_error", "Could not save settings")
			return
		}
		writeJSON(ctx, 200, map[string]any{"success": true})

	default:
		writeError(ctx, 405, "method_not_allowed", "Method not allowed")
	}
}

// ---- Teams/orgs: /account/orgs[...] ----

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func (h *OrgHandler) HandleOrgs(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required")
		return
	}
	switch string(ctx.Method()) {
	case "GET":
		teams, err := h.teamQ.ListForUser(ctx, userID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Could not load orgs")
			return
		}
		writeJSON(ctx, 200, map[string]any{"orgs": teams})
	case "POST":
		var req struct{ Name, Slug string }
		_ = json.Unmarshal(ctx.PostBody(), &req)
		name := strings.TrimSpace(req.Name)
		slug := slugify(req.Slug)
		if slug == "" {
			slug = slugify(name)
		}
		if name == "" || slug == "" {
			writeError(ctx, 400, "invalid_request", "org name required")
			return
		}
		t, err := h.teamQ.Create(ctx, name, slug, userID)
		if err != nil {
			writeError(ctx, 400, "invalid_request", "Could not create org (slug may be taken)")
			return
		}
		writeJSON(ctx, 200, map[string]any{"org": t})
	default:
		writeError(ctx, 405, "method_not_allowed", "Method not allowed")
	}
}

func (h *OrgHandler) HandleOrgInvite(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required")
		return
	}
	slug, _ := ctx.UserValue("slug").(string)
	team, err := h.teamQ.BySlug(ctx, slug)
	if err != nil {
		writeError(ctx, 404, "not_found", "org not found")
		return
	}
	if !h.canAdminTeam(ctx, userID, team.ID) {
		writeError(ctx, 403, "forbidden", "Only org owners/admins can invite")
		return
	}
	var req struct{ Email, Role string }
	_ = json.Unmarshal(ctx.PostBody(), &req)
	addr := strings.ToLower(strings.TrimSpace(req.Email))
	if addr == "" || !strings.Contains(addr, "@") {
		writeError(ctx, 400, "invalid_request", "valid email required")
		return
	}
	role := validRole(req.Role)
	token := randomToken()
	if _, err := h.teamQ.CreateInvite(ctx, team.ID, addr, role, hashTok(token), userID, time.Now().Add(14*24*time.Hour)); err != nil {
		writeError(ctx, 500, "server_error", "Could not create invite")
		return
	}
	joinURL := fmt.Sprintf("%s/orgs/%s/join?token=%s", appBaseURL(), team.Slug, token)
	go func(to, name, link string) {
		html := fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px">
<h2>You've been added to %s</h2>
<p>You've been invited to join the <strong>%s</strong> organization on OpenPaths.</p>
<p><a href="%s" style="display:inline-block;padding:10px 16px;background:#111;color:#fff;border-radius:6px;text-decoration:none">Accept invitation</a></p>
<p style="color:#888;font-size:12px">Sign up with this email first if you don't have an account. Expires in 14 days.</p>
</div>`, name, name, link)
		if err := email.Send(to, fmt.Sprintf("You've been added to %s on OpenPaths", name), html); err != nil {
			log.Printf("team-invite email failed to=%s: %v", to, err)
		}
	}(addr, team.Name, joinURL)
	writeJSON(ctx, 200, map[string]any{"success": true})
}

func (h *OrgHandler) HandleOrgJoin(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required to accept the invite")
		return
	}
	slug, _ := ctx.UserValue("slug").(string)
	team, err := h.teamQ.BySlug(ctx, slug)
	if err != nil {
		writeError(ctx, 404, "not_found", "org not found")
		return
	}
	token := strings.TrimSpace(string(ctx.QueryArgs().Peek("token")))
	if token == "" {
		writeError(ctx, 400, "invalid_request", "invite token required")
		return
	}
	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil || user == nil {
		writeError(ctx, 401, "unauthorized", "Account could not be loaded")
		return
	}
	role, err := h.teamQ.AcceptInvite(ctx, team.ID, hashTok(token), userID, user.Email)
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"org": map[string]any{"id": team.ID, "name": team.Name, "slug": team.Slug, "role": role}})
}

func (h *OrgHandler) HandleOrgMembers(ctx *fasthttp.RequestCtx) {
	userID := h.userID(ctx)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Sign in required")
		return
	}
	slug, _ := ctx.UserValue("slug").(string)
	team, err := h.teamQ.BySlug(ctx, slug)
	if err != nil {
		writeError(ctx, 404, "not_found", "org not found")
		return
	}
	switch string(ctx.Method()) {
	case "GET":
		if h.teamQ.Role(ctx, userID, team.ID) == "" && !h.isGlobalAdmin(ctx, userID) {
			writeError(ctx, 403, "forbidden", "Not a member of this org")
			return
		}
		members, err := h.teamQ.Members(ctx, team.ID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Could not load members")
			return
		}
		writeJSON(ctx, 200, map[string]any{"members": members})
	case "DELETE":
		if !h.canAdminTeam(ctx, userID, team.ID) {
			writeError(ctx, 403, "forbidden", "Only org owners/admins can remove members")
			return
		}
		memberID, _ := ctx.UserValue("user_id").(string)
		if memberID == "" {
			writeError(ctx, 400, "invalid_request", "member id required")
			return
		}
		if err := h.teamQ.RemoveMember(ctx, team.ID, memberID); err != nil {
			writeError(ctx, 500, "server_error", "Could not remove member")
			return
		}
		writeJSON(ctx, 200, map[string]any{"success": true})
	default:
		writeError(ctx, 405, "method_not_allowed", "Method not allowed")
	}
}

func validRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "admin", "member", "viewer":
		return strings.ToLower(strings.TrimSpace(role))
	}
	return "member"
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashTok(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
