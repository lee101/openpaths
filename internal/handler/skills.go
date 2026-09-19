package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"mime"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/skillindex"
)

// skillSlug reads the {slug} path param. Slugs contain slashes
// (hermes/docker), so clients send them %2F-encoded to keep the router's
// single-segment match; fasthttp hands back the still-encoded value, hence
// the unescape.
func skillSlug(ctx *fasthttp.RequestCtx) string {
	slug, _ := ctx.UserValue("slug").(string)
	slug = strings.TrimSpace(slug)
	if un, err := url.PathUnescape(slug); err == nil {
		slug = strings.TrimSpace(un)
	}
	return slug
}

// skillSearcher is the semantic index seam (satisfied by *skillindex.Service);
// kept as an interface so the handler is unit-testable with a stub.
type skillSearcher interface {
	Ready() bool
	Status() skillindex.Status
	Search(ctx context.Context, query string, f model.SkillFilters, limit int) ([]skillindex.Result, error)
	Rebuild(ctx context.Context)
}

// skillStore is the DB seam (satisfied by *queries.SkillQueries).
type skillStore interface {
	List(ctx context.Context, f model.SkillFilters, limit, offset int) ([]model.Skill, error)
	SearchILIKE(ctx context.Context, query string, f model.SkillFilters, limit int) ([]model.Skill, error)
	GetBySlug(ctx context.Context, slug string) (*model.Skill, error)
	Count(ctx context.Context) (int, error)
	SourceCounts(ctx context.Context) ([]queries.Facet, error)
	CategoryCounts(ctx context.Context) ([]queries.Facet, error)
	GetVersion(ctx context.Context, skillID, version string) (*model.SkillVersion, error)
	ListVersions(ctx context.Context, skillID string) ([]string, error)
	ListFiles(ctx context.Context, skillID, version string) ([]model.SkillFile, error)
	GetFile(ctx context.Context, skillID, version, filePath string) ([]byte, string, error)
	CreateSkill(ctx context.Context, s *model.Skill) error
	CreateVersion(ctx context.Context, skillID string, v *model.SkillVersion) error
	UpdateHead(ctx context.Context, s *model.Skill) error
	DeleteSkill(ctx context.Context, slug string) error
	UpsertFile(ctx context.Context, skillID, version, filePath, mimeType string, content []byte) error
	CopyVersionFiles(ctx context.Context, skillID, fromVersion, toVersion string) error
}

// SkillsHandler serves the agent-skill library: public reads (gobed semantic
// search with pg_trgm ILIKE fallback) plus owner-scoped writes (marketplace).
// Semantic search is powered by the gobed-backed skillindex; it degrades to
// pg_trgm ILIKE search whenever the index is unavailable so the endpoint
// always answers.
type SkillsHandler struct {
	index skillSearcher
	q     skillStore
}

func NewSkillsHandler(index *skillindex.Service, q *queries.SkillQueries) *SkillsHandler {
	h := &SkillsHandler{}
	if index != nil {
		h.index = index
	}
	if q != nil {
		h.q = q
	}
	return h
}

func skillFilters(ctx *fasthttp.RequestCtx) model.SkillFilters {
	qa := ctx.QueryArgs()
	return model.SkillFilters{
		Source:   strings.TrimSpace(string(qa.Peek("source"))),
		Category: strings.TrimSpace(string(qa.Peek("category"))),
	}
}

func skillAuthedUser(ctx *fasthttp.RequestCtx) string {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	return userID
}

// pinSkills overlays an immutable version snapshot onto each skill for
// ?version= reads. Skills without that version are dropped.
func (h *SkillsHandler) pinSkills(ctx context.Context, skills []model.Skill, version string) []model.Skill {
	if version == "" || h.q == nil {
		return skills
	}
	out := make([]model.Skill, 0, len(skills))
	for i := range skills {
		v, err := h.q.GetVersion(ctx, skills[i].ID, version)
		if err != nil || v == nil {
			continue
		}
		pinned := skills[i]
		pinned.Body = v.Body
		pinned.SetupScript = v.SetupScript
		pinned.SetupPrompt = v.SetupPrompt
		pinned.SkillPrompt = v.SkillPrompt
		pinned.CurrentVersion = v.Version
		out = append(out, pinned)
	}
	return out
}

// HandleList serves GET /v1/skills (browse + optional ?q= semantic search).
func (h *SkillsHandler) HandleList(ctx *fasthttp.RequestCtx) {
	if h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "count": 0, "skills": []any{}})
		return
	}
	f := skillFilters(ctx)
	limit := parseLimit(ctx, 200, 500)
	skills, err := h.q.List(ctx, f, limit, 0)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not list skills"})
		return
	}
	if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" {
		skills = h.pinSkills(ctx, skills, v)
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "count": len(skills), "skills": skills})
}

// HandleSearch serves GET /v1/skills/search?q=&k= — semantic when the index is
// ready, else pg_trgm ILIKE.
func (h *SkillsHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	query := strings.TrimSpace(string(ctx.QueryArgs().Peek("q")))
	f := skillFilters(ctx)
	k := parseLimit(ctx, 20, 50)
	if n, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("k"))); n > 0 && n <= 50 {
		k = n
	}
	if query == "" {
		WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "query": "", "semantic": false, "count": 0, "results": []any{}})
		return
	}

	if h.index != nil && h.index.Ready() {
		if results, err := h.index.Search(ctx, query, f, k); err == nil {
			if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" && h.q != nil {
				skills := make([]model.Skill, 0, len(results))
				for _, r := range results {
					skills = append(skills, r.Skill)
				}
				pinned := h.pinSkills(ctx, skills, v)
				byID := make(map[string]model.Skill, len(pinned))
				for _, s := range pinned {
					byID[s.ID] = s
				}
				kept := make([]skillindex.Result, 0, len(pinned))
				for _, r := range results {
					if s, ok := byID[r.Skill.ID]; ok {
						r.Skill = s
						kept = append(kept, r)
					}
				}
				results = kept
			}
			WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
				"object": "list", "query": query, "semantic": true, "count": len(results), "results": results,
			})
			return
		}
	}

	// Fallback: pg_trgm ILIKE.
	var results []model.Skill
	if h.q != nil {
		results, _ = h.q.SearchILIKE(ctx, query, f, k)
		if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" {
			results = h.pinSkills(ctx, results, v)
		}
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"object": "list", "query": query, "semantic": false, "count": len(results), "results": results,
	})
}

// HandleGet serves GET /v1/skills/{slug} — a single skill with its full body,
// a copy-paste-ready markdown payload (Setup preamble + body), the version
// list, and the file manifest (no bytes). ?version= pins an immutable snapshot;
// an unknown pin is 404 {error: 'version not found'}.
func (h *SkillsHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	slug := skillSlug(ctx)
	if slug == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" {
		snap, verr := h.q.GetVersion(ctx, skill.ID, v)
		if verr != nil || snap == nil {
			WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "version not found"})
			return
		}
		skill.Body = snap.Body
		skill.SetupScript = snap.SetupScript
		skill.SetupPrompt = snap.SetupPrompt
		skill.SkillPrompt = snap.SkillPrompt
		skill.CurrentVersion = snap.Version
	}
	versions, _ := h.q.ListVersions(ctx, skill.ID)
	if versions == nil {
		versions = []string{}
	}
	effective := skill.CurrentVersion
	if effective == "" {
		effective = "1.0.0"
	}
	files, _ := h.q.ListFiles(ctx, skill.ID, effective)
	if files == nil {
		files = []model.SkillFile{}
	}
	skill.Versions = versions
	skill.Files = files
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"object":   "skill",
		"skill":    skill,
		"markdown": skill.Markdown(),
		"versions": versions,
		"files":    files,
	})
}

// HandleFiles serves GET /v1/skills/{slug}/files — the file manifest (no
// bytes) for the head version, or ?version= for a pinned snapshot.
func (h *SkillsHandler) HandleFiles(ctx *fasthttp.RequestCtx) {
	slug := skillSlug(ctx)
	if slug == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	version := skill.CurrentVersion
	if version == "" {
		version = "1.0.0"
	}
	if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" {
		if snap, verr := h.q.GetVersion(ctx, skill.ID, v); verr != nil || snap == nil {
			WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "version not found"})
			return
		}
		version = v
	}
	files, _ := h.q.ListFiles(ctx, skill.ID, version)
	if files == nil {
		files = []model.SkillFile{}
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "count": len(files), "files": files})
}

// HandleFileRaw serves GET /v1/skills/{slug}/files/{path} — raw file bytes
func (h *SkillsHandler) HandleFileRaw(ctx *fasthttp.RequestCtx) {
	slug := skillSlug(ctx)
	filePath := skillFilePath(ctx)
	if slug == "" || filePath == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug and path required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	version := skill.CurrentVersion
	if version == "" {
		version = "1.0.0"
	}
	if v := model.NormalizeVersion(string(ctx.QueryArgs().Peek("version"))); v != "" {
		if snap, verr := h.q.GetVersion(ctx, skill.ID, v); verr != nil || snap == nil {
			WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "version not found"})
			return
		}
		version = v
	}
	content, mimeType, err := h.q.GetFile(ctx, skill.ID, version, filePath)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "file not found"})
		return
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType(mimeType)
	ctx.SetBody(content)
}

// skillFilePath reads the {filepath:*} (or legacy {path}) route param,
// unescaping %2F so nested paths survive the router as one segment.
func skillFilePath(ctx *fasthttp.RequestCtx) string {
	for _, key := range []string{"filepath", "path"} {
		if v, _ := ctx.UserValue(key).(string); strings.TrimSpace(v) != "" {
			v = strings.TrimSpace(v)
			if un, err := url.PathUnescape(v); err == nil {
				v = un
			}
			return v
		}
	}
	return ""
}

// HandleMeta serves GET /v1/skills/meta — sources, categories, total, index status.
func (h *SkillsHandler) HandleMeta(ctx *fasthttp.RequestCtx) {
	out := map[string]any{"object": "meta"}
	if h.q != nil {
		if total, err := h.q.Count(ctx); err == nil {
			out["total"] = total
		}
		if sources, err := h.q.SourceCounts(ctx); err == nil {
			out["sources"] = sources
		}
		if cats, err := h.q.CategoryCounts(ctx); err == nil {
			out["categories"] = cats
		}
	}
	if h.index != nil {
		out["index"] = h.index.Status()
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, out)
}

// HandleReindex rebuilds the semantic index from the DB. Admin-only (wrapped
// with adminH.RequireAdmin at route registration, like art reindex).
func (h *SkillsHandler) HandleReindex(ctx *fasthttp.RequestCtx) {
	if h.index == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{"error": "index disabled"})
		return
	}
	go h.index.Rebuild(context.Background())
	WriteJSONPublic(ctx, fasthttp.StatusAccepted, map[string]any{"status": "reindexing"})
}

// --- Owner-scoped writes (marketplace) ---

const (
	skillNameMax     = 80
	skillDescMax     = 300
	skillBodyMax     = 200_000
	skillTextFileMax = 200 * 1024
	skillBinFileMax  = 2 * 1024 * 1024
	skillFilesMax    = 100
)

type skillFileInput struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "" (text) or "base64" (binary)
}

type skillWriteRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Body        string            `json:"body"`
	SetupScript string            `json:"setupScript"`
	SetupPrompt string            `json:"setupPrompt"`
	SkillPrompt string            `json:"skillPrompt"`
	Version     string            `json:"version"`
	Files       *[]skillFileInput `json:"files"`
}

var skillSlugStripRe = regexp.MustCompile(`[^a-z0-9/_-]+`)

func slugifySkillName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = skillSlugStripRe.ReplaceAllString(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-/")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-/")
	}
	return s
}

func validateSkillWrite(req *skillWriteRequest) string {
	if len([]rune(req.Name)) == 0 {
		return "name is required"
	}
	if len([]rune(req.Name)) > skillNameMax {
		return "name must be at most 80 characters"
	}
	if len([]rune(req.Description)) > skillDescMax {
		return "description must be at most 300 characters"
	}
	if len([]rune(req.Body)) > skillBodyMax {
		return "body must be at most 200000 characters"
	}
	return ""
}

func isSkillConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") ||
		strings.Contains(msg, "already exists")
}

// decodeSkillFiles validates caps (≤100 files, text ≤200KB inline, binary
// ≤2MB) and returns path/content/mime triples.
func decodeSkillFiles(files []skillFileInput) ([]model.SkillFile, [][]byte, string) {
	if len(files) > skillFilesMax {
		return nil, nil, "at most 100 files per version"
	}
	seen := make(map[string]struct{}, len(files))
	meta := make([]model.SkillFile, 0, len(files))
	contents := make([][]byte, 0, len(files))
	for _, f := range files {
		p := path.Clean(strings.TrimSpace(f.Path))
		if p == "" || p == "." || p == "/" || strings.HasPrefix(p, "../") || strings.Contains(p, "../") || strings.HasPrefix(p, "/") {
			return nil, nil, "invalid file path: " + f.Path
		}
		if len(p) > 256 {
			return nil, nil, "file path too long: " + f.Path
		}
		if _, dup := seen[p]; dup {
			return nil, nil, "duplicate file path: " + p
		}
		seen[p] = struct{}{}
		var content []byte
		if strings.EqualFold(strings.TrimSpace(f.Encoding), "base64") {
			raw, err := base64.StdEncoding.DecodeString(f.Content)
			if err != nil {
				if raw, err2 := base64.URLEncoding.DecodeString(f.Content); err2 != nil {
					return nil, nil, "invalid base64 content for file: " + p
				} else {
					content = raw
				}
			} else {
				content = raw
			}
			if len(content) > skillBinFileMax {
				return nil, nil, "binary file too large (max 2MB): " + p
			}
		} else {
			content = []byte(f.Content)
			if len(content) > skillTextFileMax {
				return nil, nil, "text file too large (max 200KB inline): " + p
			}
		}
		mt := mime.TypeByExtension(path.Ext(p))
		if i := strings.Index(mt, ";"); i >= 0 {
			mt = strings.TrimSpace(mt[:i])
		}
		if mt == "" {
			mt = "application/octet-stream"
		}
		meta = append(meta, model.SkillFile{Path: p, Mime: mt, Size: len(content)})
		contents = append(contents, content)
	}
	return meta, contents, ""
}

// HandleCreate serves POST /v1/skills — creates a skill owned by the authed
// user at version 1.0.0.
func (h *SkillsHandler) HandleCreate(ctx *fasthttp.RequestCtx) {
	userID := skillAuthedUser(ctx)
	if userID == "" {
		WriteJSONPublic(ctx, fasthttp.StatusUnauthorized, map[string]any{"error": "authentication required"})
		return
	}
	if h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{"error": "skills store unavailable"})
		return
	}
	var req skillWriteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
		return
	}
	if msg := validateSkillWrite(&req); msg != "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": msg})
		return
	}
	slug := slugifySkillName(req.Name)
	if slug == "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "name must contain letters or numbers"})
		return
	}
	if existing, _ := h.q.GetBySlug(ctx, slug); existing != nil {
		WriteJSONPublic(ctx, fasthttp.StatusConflict, map[string]any{"error": "slug already taken"})
		return
	}
	var fileMeta []model.SkillFile
	var fileContents [][]byte
	if req.Files != nil {
		var msg string
		fileMeta, fileContents, msg = decodeSkillFiles(*req.Files)
		if msg != "" {
			WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": msg})
			return
		}
	}
	s := &model.Skill{
		ID:             slug,
		Slug:           slug,
		Name:           strings.TrimSpace(req.Name),
		Description:    req.Description,
		Body:           req.Body,
		OwnerID:        userID,
		CurrentVersion: "1.0.0",
		SetupScript:    req.SetupScript,
		SetupPrompt:    req.SetupPrompt,
		SkillPrompt:    req.SkillPrompt,
		Tags:           []string{},
	}
	if err := h.q.CreateSkill(ctx, s); err != nil {
		if isSkillConflict(err) {
			WriteJSONPublic(ctx, fasthttp.StatusConflict, map[string]any{"error": "slug already taken"})
			return
		}
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not create skill"})
		return
	}
	for i, fm := range fileMeta {
		if err := h.q.UpsertFile(ctx, s.ID, s.CurrentVersion, fm.Path, fm.Mime, fileContents[i]); err != nil {
			WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not store skill files"})
			return
		}
	}
	s.Files = fileMetaOrEmpty(fileMeta)
	s.Versions = []string{s.CurrentVersion}
	WriteJSONPublic(ctx, fasthttp.StatusCreated, map[string]any{"object": "skill", "skill": s})
}

// HandleUpdate serves PUT /v1/skills/{slug} — owner-only. Appends a new
// immutable version (explicit ?version? from the body, else an automatic
// patch bump) and moves the head pin.
func (h *SkillsHandler) HandleUpdate(ctx *fasthttp.RequestCtx) {
	userID := skillAuthedUser(ctx)
	if userID == "" {
		WriteJSONPublic(ctx, fasthttp.StatusUnauthorized, map[string]any{"error": "authentication required"})
		return
	}
	slug := skillSlug(ctx)
	if slug == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	if skill.OwnerID != userID {
		WriteJSONPublic(ctx, fasthttp.StatusForbidden, map[string]any{"error": "not your skill"})
		return
	}
	var req skillWriteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "invalid JSON: " + err.Error()})
		return
	}
	name := skill.Name
	if strings.TrimSpace(req.Name) != "" {
		name = strings.TrimSpace(req.Name)
	}
	description := skill.Description
	if req.Description != "" {
		description = req.Description
	}
	body := skill.Body
	if req.Body != "" {
		body = req.Body
	}
	setupScript := skill.SetupScript
	if req.SetupScript != "" {
		setupScript = req.SetupScript
	}
	setupPrompt := skill.SetupPrompt
	if req.SetupPrompt != "" {
		setupPrompt = req.SetupPrompt
	}
	skillPrompt := skill.SkillPrompt
	if req.SkillPrompt != "" {
		skillPrompt = req.SkillPrompt
	}
	probe := &skillWriteRequest{Name: name, Description: description, Body: body}
	if msg := validateSkillWrite(probe); msg != "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": msg})
		return
	}
	newVersion := model.NormalizeVersion(req.Version)
	if newVersion == "" {
		base := skill.CurrentVersion
		if base == "" {
			base = "1.0.0"
		}
		newVersion = model.BumpPatchVersion(base)
	} else if !model.IsValidVersion(newVersion) {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "version must look like '1.0.0'"})
		return
	}
	if existing, _ := h.q.GetVersion(ctx, skill.ID, newVersion); existing != nil {
		WriteJSONPublic(ctx, fasthttp.StatusConflict, map[string]any{"error": "version already exists"})
		return
	}
	var fileMeta []model.SkillFile
	var fileContents [][]byte
	if req.Files != nil {
		var msg string
		fileMeta, fileContents, msg = decodeSkillFiles(*req.Files)
		if msg != "" {
			WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": msg})
			return
		}
	}
	if err := h.q.CreateVersion(ctx, skill.ID, &model.SkillVersion{
		SkillID:     skill.ID,
		Version:     newVersion,
		Body:        body,
		SetupScript: setupScript,
		SetupPrompt: setupPrompt,
		SkillPrompt: skillPrompt,
	}); err != nil {
		if isSkillConflict(err) {
			WriteJSONPublic(ctx, fasthttp.StatusConflict, map[string]any{"error": "version already exists"})
			return
		}
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not create skill version"})
		return
	}
	for i, fm := range fileMeta {
		if err := h.q.UpsertFile(ctx, skill.ID, newVersion, fm.Path, fm.Mime, fileContents[i]); err != nil {
			WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not store skill files"})
			return
		}
	}
	if req.Files == nil {
		if err := h.q.CopyVersionFiles(ctx, skill.ID, skill.CurrentVersion, newVersion); err != nil {
			WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not carry over skill files"})
			return
		}
	}
	skill.Name = name
	skill.Description = description
	skill.Body = body
	skill.SetupScript = setupScript
	skill.SetupPrompt = setupPrompt
	skill.SkillPrompt = skillPrompt
	skill.CurrentVersion = newVersion
	if err := h.q.UpdateHead(ctx, skill); err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not update skill"})
		return
	}
	versions, _ := h.q.ListVersions(ctx, skill.ID)
	files, _ := h.q.ListFiles(ctx, skill.ID, newVersion)
	skill.Versions = versionsOrEmpty(versions)
	skill.Files = fileMetaOrEmpty(files)
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "skill", "skill": skill})
}

// HandleDelete serves DELETE /v1/skills/{slug} — owner-only.
func (h *SkillsHandler) HandleDelete(ctx *fasthttp.RequestCtx) {
	userID := skillAuthedUser(ctx)
	if userID == "" {
		WriteJSONPublic(ctx, fasthttp.StatusUnauthorized, map[string]any{"error": "authentication required"})
		return
	}
	slug := skillSlug(ctx)
	if slug == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	if skill.OwnerID != userID {
		WriteJSONPublic(ctx, fasthttp.StatusForbidden, map[string]any{"error": "not your skill"})
		return
	}
	if err := h.q.DeleteSkill(ctx, slug); err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not delete skill"})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"deleted": true, "slug": slug})
}

func fileMetaOrEmpty(in []model.SkillFile) []model.SkillFile {
	if in == nil {
		return []model.SkillFile{}
	}
	return in
}

func versionsOrEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
