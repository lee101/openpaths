package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/valyala/fasthttp"
)

// artifactSearchCostUnits is $1 per 1000 searches. Units are hundredths-of-a-cent ($1 = 10000).
const artifactSearchCostUnits = int64(10)

type ArtifactHandler struct {
	q       *queries.ArtifactQueries
	billing *billing.Engine
}

func NewArtifactHandler(q *queries.ArtifactQueries, b *billing.Engine) *ArtifactHandler {
	return &ArtifactHandler{q: q, billing: b}
}

type artifactFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type artifactRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Files       []artifactFile `json:"files"`
	Entry       string         `json:"entry"`
	Visibility  string         `json:"visibility"`
	Tags        []string       `json:"tags"`
	ForkOf      string         `json:"fork_of"`
}

func validVisibility(v string) bool {
	return v == "private" || v == "public" || v == "unlisted"
}

// --- Dashboard (owner-scoped) ---

func (h *ArtifactHandler) HandleListMine(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	limit := atoiDefault(string(ctx.QueryArgs().Peek("limit")), 50)
	items, err := h.q.ListByUser(ctx, userID, limit)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "artifacts": artifactSummaries(items)})
}

func (h *ArtifactHandler) HandleCreate(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	var req artifactRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if !validVisibility(req.Visibility) {
		writeError(ctx, 400, "invalid_request", "visibility must be private, public, or unlisted")
		return
	}
	a := &queries.Artifact{
		ID:          newArtifactID(),
		UserID:      userID,
		Slug:        makeArtifactSlug(req.Title),
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		ImageURL:    req.ImageURL,
		Files:       marshalFiles(req.Files),
		Entry:       req.Entry,
		Visibility:  req.Visibility,
		Tags:        req.Tags,
	}
	if req.ForkOf != "" {
		a.ForkOf = &req.ForkOf
	}
	created, err := h.q.Create(ctx, a)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 201, artifactPayload(created))
}

func (h *ArtifactHandler) HandleGetMine(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	a, err := h.q.GetByID(ctx, id)
	if err != nil || a == nil {
		writeError(ctx, 404, "not_found", "Artifact not found")
		return
	}
	if a.UserID != userID {
		writeError(ctx, 403, "forbidden", "Not your artifact")
		return
	}
	writeJSON(ctx, 200, artifactPayload(a))
}

func (h *ArtifactHandler) HandleUpdate(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	var req artifactRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Visibility == "" {
		req.Visibility = "private"
	}
	if !validVisibility(req.Visibility) {
		writeError(ctx, 400, "invalid_request", "visibility must be private, public, or unlisted")
		return
	}
	a := &queries.Artifact{
		ID:          id,
		UserID:      userID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		ImageURL:    req.ImageURL,
		Files:       marshalFiles(req.Files),
		Entry:       req.Entry,
		Visibility:  req.Visibility,
		Tags:        req.Tags,
	}
	updated, err := h.q.Update(ctx, a)
	if err != nil {
		writeError(ctx, 404, "not_found", "Artifact not found or not yours")
		return
	}
	writeJSON(ctx, 200, artifactPayload(updated))
}

func (h *ArtifactHandler) HandleDelete(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	id, _ := ctx.UserValue("id").(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	if err := h.q.Delete(ctx, id, userID); err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"deleted": true, "id": id})
}

// --- Public API ---

// HandleGet returns a single artifact and its files. Free. Public/unlisted only (private hidden).
func (h *ArtifactHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	id, _ := ctx.UserValue("id").(string)
	a := h.lookup(ctx, id)
	if a == nil {
		writeError(ctx, 404, "not_found", "Artifact not found")
		return
	}
	if a.Visibility == "private" {
		userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
		if a.UserID != userID {
			writeError(ctx, 404, "not_found", "Artifact not found")
			return
		}
	}
	h.q.IncrementViews(ctx, a.ID)
	writeJSON(ctx, 200, artifactPayload(a))
}

func (h *ArtifactHandler) HandleListPublic(ctx *fasthttp.RequestCtx) {
	limit := atoiDefault(string(ctx.QueryArgs().Peek("limit")), 48)
	offset := atoiDefault(string(ctx.QueryArgs().Peek("offset")), 0)
	items, err := h.q.ListPublic(ctx, limit, offset)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "artifacts": artifactSummaries(items)})
}

// HandleSearch is billed at $1 per 1000 searches.
func (h *ArtifactHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	if err := h.billing.PreCheckFixed(ctx, userID, artifactSearchCostUnits); err != nil {
		writeError(ctx, 402, "insufficient_balance", "Insufficient credits for artifact search")
		return
	}
	term := string(ctx.QueryArgs().Peek("q"))
	limit := atoiDefault(string(ctx.QueryArgs().Peek("limit")), 24)
	// Public-only search via the API key surface.
	items, err := h.q.Search(ctx, term, "", limit)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	if err := h.billing.DeductFixed(ctx, userID, "artifact-search", artifactSearchCostUnits, "Artifact search", ""); err != nil {
		writeError(ctx, 402, "insufficient_balance", "Insufficient credits for artifact search")
		return
	}
	writeJSON(ctx, 200, map[string]any{
		"object":     "list",
		"query":      term,
		"cost_units": artifactSearchCostUnits,
		"artifacts":  artifactSummaries(items),
	})
}

// HandleSearchMine searches the authenticated user's own + public artifacts (dashboard, free).
func (h *ArtifactHandler) HandleSearchMine(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		writeError(ctx, 401, "unauthorized", "Authentication required")
		return
	}
	term := string(ctx.QueryArgs().Peek("q"))
	limit := atoiDefault(string(ctx.QueryArgs().Peek("limit")), 24)
	items, err := h.q.Search(ctx, term, userID, limit)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "query": term, "artifacts": artifactSummaries(items)})
}

func (h *ArtifactHandler) lookup(ctx *fasthttp.RequestCtx, id string) *queries.Artifact {
	if id == "" {
		return nil
	}
	if a, err := h.q.GetByID(ctx, id); err == nil && a != nil {
		return a
	}
	if a, err := h.q.GetBySlug(ctx, id); err == nil && a != nil {
		return a
	}
	return nil
}

// --- payloads ---

func artifactPayload(a *queries.Artifact) map[string]any {
	p := artifactSummary(a)
	p["files"] = rawOrEmpty(a.Files, "[]")
	return p
}

func artifactSummary(a *queries.Artifact) map[string]any {
	tags := a.Tags
	if tags == nil {
		tags = []string{}
	}
	p := map[string]any{
		"id":          a.ID,
		"object":      "artifact",
		"slug":        a.Slug,
		"user_id":     a.UserID,
		"title":       a.Title,
		"description": a.Description,
		"image_url":   a.ImageURL,
		"entry":       a.Entry,
		"visibility":  a.Visibility,
		"tags":        tags,
		"view_count":  a.ViewCount,
		"created_at":  a.CreatedAt.Unix(),
		"updated_at":  a.UpdatedAt.Unix(),
	}
	if a.ForkOf != nil {
		p["fork_of"] = *a.ForkOf
	}
	if a.PublishedAt != nil {
		p["published_at"] = a.PublishedAt.Unix()
	}
	return p
}

func artifactSummaries(items []*queries.Artifact) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		out = append(out, artifactSummary(a))
	}
	return out
}

// --- helpers ---

func marshalFiles(files []artifactFile) json.RawMessage {
	if len(files) == 0 {
		return json.RawMessage("[]")
	}
	data, err := json.Marshal(files)
	if err != nil {
		return json.RawMessage("[]")
	}
	return data
}

func newArtifactID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "artifact_" + hex.EncodeToString(b)
}

var slugStripRe = regexp.MustCompile(`[^a-z0-9]+`)

func makeArtifactSlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugStripRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "artifact"
	}
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return s + "-" + hex.EncodeToString(b)
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
