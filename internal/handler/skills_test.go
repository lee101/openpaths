package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/skillindex"
)

type stubIndex struct {
	ready   bool
	results []skillindex.Result
}

func (s stubIndex) Ready() bool               { return s.ready }
func (s stubIndex) Status() skillindex.Status { return skillindex.Status{Ready: s.ready} }
func (s stubIndex) Search(_ context.Context, _ string, _ model.SkillFilters, _ int) ([]skillindex.Result, error) {
	return s.results, nil
}
func (s stubIndex) Rebuild(_ context.Context) {}

type stubStore struct {
	skills   []model.Skill
	bySlug   map[string]model.Skill
	versions map[string]map[string]*model.SkillVersion // skillID -> version -> snapshot
	files    map[string]map[string]map[string]fileBlob // skillID -> version -> path -> blob
	created  []*model.Skill
	updated  []*model.Skill
	deleted  []string
}

type fileBlob struct {
	content []byte
	mime    string
}

func (s stubStore) List(_ context.Context, _ model.SkillFilters, _, _ int) ([]model.Skill, error) {
	return s.skills, nil
}
func (s stubStore) SearchILIKE(_ context.Context, _ string, _ model.SkillFilters, _ int) ([]model.Skill, error) {
	return s.skills, nil
}
func (s stubStore) GetBySlug(_ context.Context, slug string) (*model.Skill, error) {
	if sk, ok := s.bySlug[slug]; ok {
		c := sk
		return &c, nil
	}
	for _, sk := range s.skills {
		if sk.Slug == slug {
			c := sk
			return &c, nil
		}
	}
	return nil, errors.New("not found")
}
func (s stubStore) Count(_ context.Context) (int, error) { return len(s.skills), nil }
func (s stubStore) SourceCounts(_ context.Context) ([]queries.Facet, error) {
	return []queries.Facet{{Value: "hermes", Count: 1}}, nil
}
func (s stubStore) CategoryCounts(_ context.Context) ([]queries.Facet, error) {
	return []queries.Facet{}, nil
}
func (s stubStore) GetVersion(_ context.Context, skillID, version string) (*model.SkillVersion, error) {
	if v, ok := s.versions[skillID][version]; ok {
		c := *v
		return &c, nil
	}
	return nil, errors.New("not found")
}
func (s stubStore) ListVersions(_ context.Context, skillID string) ([]string, error) {
	out := []string{}
	for v := range s.versions[skillID] {
		out = append(out, v)
	}
	return out, nil
}
func (s stubStore) ListFiles(_ context.Context, skillID, version string) ([]model.SkillFile, error) {
	out := []model.SkillFile{}
	for p, b := range s.files[skillID][version] {
		out = append(out, model.SkillFile{Path: p, Mime: b.mime, Size: len(b.content)})
	}
	return out, nil
}
func (s stubStore) GetFile(_ context.Context, skillID, version, filePath string) ([]byte, string, error) {
	if b, ok := s.files[skillID][version][filePath]; ok {
		return b.content, b.mime, nil
	}
	return nil, "", errors.New("not found")
}
func (s *stubStore) CreateSkill(_ context.Context, sk *model.Skill) error {
	s.created = append(s.created, sk)
	if s.bySlug == nil {
		s.bySlug = map[string]model.Skill{}
	}
	s.bySlug[sk.Slug] = *sk
	return nil
}
func (s *stubStore) CreateVersion(_ context.Context, skillID string, v *model.SkillVersion) error {
	if s.versions == nil {
		s.versions = map[string]map[string]*model.SkillVersion{}
	}
	if s.versions[skillID] == nil {
		s.versions[skillID] = map[string]*model.SkillVersion{}
	}
	if _, ok := s.versions[skillID][v.Version]; ok {
		return errors.New("duplicate key value violates unique constraint")
	}
	s.versions[skillID][v.Version] = v
	return nil
}
func (s *stubStore) UpdateHead(_ context.Context, sk *model.Skill) error {
	s.updated = append(s.updated, sk)
	if s.bySlug == nil {
		s.bySlug = map[string]model.Skill{}
	}
	s.bySlug[sk.Slug] = *sk
	return nil
}
func (s *stubStore) DeleteSkill(_ context.Context, slug string) error {
	s.deleted = append(s.deleted, slug)
	delete(s.bySlug, slug)
	return nil
}
func (s *stubStore) UpsertFile(_ context.Context, skillID, version, filePath, mimeType string, content []byte) error {
	if s.files == nil {
		s.files = map[string]map[string]map[string]fileBlob{}
	}
	if s.files[skillID] == nil {
		s.files[skillID] = map[string]map[string]fileBlob{}
	}
	if s.files[skillID][version] == nil {
		s.files[skillID][version] = map[string]fileBlob{}
	}
	s.files[skillID][version][filePath] = fileBlob{content: content, mime: mimeType}
	return nil
}
func (s *stubStore) CopyVersionFiles(_ context.Context, skillID, fromVersion, toVersion string) error {
	if s.files == nil {
		return nil
	}
	if s.files[skillID] == nil {
		s.files[skillID] = map[string]map[string]fileBlob{}
	}
	if s.files[skillID][toVersion] == nil {
		s.files[skillID][toVersion] = map[string]fileBlob{}
	}
	for p, b := range s.files[skillID][fromVersion] {
		s.files[skillID][toVersion][p] = b
	}
	return nil
}

func newCtx(uri string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI(uri)
	return ctx
}

func newAuthedCtx(uri, userID string) *fasthttp.RequestCtx {
	ctx := newCtx(uri)
	ctx.SetUserValue(middleware.CtxKeyUserID, userID)
	return ctx
}

func decode(t *testing.T, ctx *fasthttp.RequestCtx) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &m); err != nil {
		t.Fatalf("decode: %v body=%s", err, ctx.Response.Body())
	}
	return m
}

func TestSkillsSearchSemantic(t *testing.T) {
	h := &SkillsHandler{
		index: stubIndex{ready: true, results: []skillindex.Result{{Skill: model.Skill{Slug: "hermes/dogfood"}, Score: 0.9}}},
		q:     &stubStore{},
	}
	ctx := newCtx("/v1/skills/search?q=qa&k=5")
	h.HandleSearch(ctx)
	m := decode(t, ctx)
	if m["semantic"] != true {
		t.Fatalf("expected semantic=true, got %v", m["semantic"])
	}
	if c, _ := m["count"].(float64); c != 1 {
		t.Fatalf("expected count=1, got %v", m["count"])
	}
}

func TestSkillsSearchFallback(t *testing.T) {
	// index not ready -> ILIKE fallback via the store.
	h := &SkillsHandler{
		index: stubIndex{ready: false},
		q:     &stubStore{skills: []model.Skill{{Slug: "hermes/docker", Name: "docker"}}},
	}
	ctx := newCtx("/v1/skills/search?q=docker")
	h.HandleSearch(ctx)
	m := decode(t, ctx)
	if m["semantic"] != false {
		t.Fatalf("expected semantic=false, got %v", m["semantic"])
	}
	if c, _ := m["count"].(float64); c != 1 {
		t.Fatalf("expected count=1, got %v", m["count"])
	}
}

func TestSkillsGetMarkdownAndNotFound(t *testing.T) {
	h := &SkillsHandler{q: &stubStore{bySlug: map[string]model.Skill{
		"hermes/dogfood": {ID: "hermes/dogfood", Slug: "hermes/dogfood", Body: "do qa", SetupPreamble: "## Setup\ngit clone x", CurrentVersion: "1.0.0"},
	}}}

	ok := newCtx("/v1/skills/hermes/dogfood")
	ok.SetUserValue("slug", "hermes/dogfood")
	h.HandleGet(ok)
	m := decode(t, ok)
	md, _ := m["markdown"].(string)
	if md == "" || md[:8] != "## Setup" {
		t.Fatalf("expected markdown with Setup preamble, got %q", md)
	}
	if _, ok := m["versions"]; !ok {
		t.Fatal("expected versions key in get response")
	}
	if _, ok := m["files"]; !ok {
		t.Fatal("expected files key in get response")
	}

	miss := newCtx("/v1/skills/nope")
	miss.SetUserValue("slug", "nope")
	h.HandleGet(miss)
	if miss.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", miss.Response.StatusCode())
	}
}

func TestSkillsGetPinnedVersion(t *testing.T) {
	store := &stubStore{
		bySlug: map[string]model.Skill{
			"mine": {ID: "mine", Slug: "mine", Name: "mine", Body: "head body", CurrentVersion: "1.0.1"},
		},
		versions: map[string]map[string]*model.SkillVersion{
			"mine": {
				"1.0.0": {SkillID: "mine", Version: "1.0.0", Body: "v1 body"},
				"1.0.1": {SkillID: "mine", Version: "1.0.1", Body: "head body"},
			},
		},
	}
	h := &SkillsHandler{q: store}

	pinned := newCtx("/v1/skills/mine?version=1.0.0")
	pinned.SetUserValue("slug", "mine")
	h.HandleGet(pinned)
	if pinned.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", pinned.Response.StatusCode(), pinned.Response.Body())
	}
	m := decode(t, pinned)
	sk, _ := m["skill"].(map[string]any)
	if sk["body"] != "v1 body" {
		t.Fatalf("expected pinned v1 body, got %v", sk["body"])
	}

	missing := newCtx("/v1/skills/mine?version=9.9.9")
	missing.SetUserValue("slug", "mine")
	h.HandleGet(missing)
	if missing.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404 for missing version, got %d", missing.Response.StatusCode())
	}
	if got := decode(t, missing)["error"]; got != "version not found" {
		t.Fatalf("expected version not found error, got %v", got)
	}
}

func TestSkillsUpdateOwnerOnly(t *testing.T) {
	store := &stubStore{
		bySlug: map[string]model.Skill{
			"mine": {ID: "mine", Slug: "mine", Name: "mine", Body: "b", OwnerID: "owner-1", CurrentVersion: "1.0.0"},
		},
	}
	h := &SkillsHandler{q: store}

	forbidden := newAuthedCtx("/v1/skills/mine", "owner-2")
	forbidden.SetUserValue("slug", "mine")
	forbidden.Request.Header.SetMethod("PUT")
	forbidden.Request.SetBody([]byte(`{"body":"new"}`))
	h.HandleUpdate(forbidden)
	if forbidden.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", forbidden.Response.StatusCode(), forbidden.Response.Body())
	}

	ok := newAuthedCtx("/v1/skills/mine", "owner-1")
	ok.SetUserValue("slug", "mine")
	ok.Request.Header.SetMethod("PUT")
	ok.Request.SetBody([]byte(`{"body":"new"}`))
	h.HandleUpdate(ok)
	if ok.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ok.Response.StatusCode(), ok.Response.Body())
	}
	m := decode(t, ok)
	sk, _ := m["skill"].(map[string]any)
	if sk["currentVersion"] != "1.0.1" {
		t.Fatalf("expected auto patch bump to 1.0.1, got %v", sk["currentVersion"])
	}
}

func TestSkillsCreateValidationAndConflict(t *testing.T) {
	store := &stubStore{bySlug: map[string]model.Skill{}}
	h := &SkillsHandler{q: store}

	anon := newCtx("/v1/skills")
	anon.Request.Header.SetMethod("POST")
	anon.Request.SetBody([]byte(`{"name":"x"}`))
	h.HandleCreate(anon)
	if anon.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", anon.Response.StatusCode())
	}

	tooLong := newAuthedCtx("/v1/skills", "u1")
	tooLong.Request.Header.SetMethod("POST")
	tooLong.Request.SetBody([]byte(`{"name":"way-too-long-` + repeat("n", 81) + `"}`))
	h.HandleCreate(tooLong)
	if tooLong.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("expected 400 for long name, got %d body=%s", tooLong.Response.StatusCode(), tooLong.Response.Body())
	}

	ok := newAuthedCtx("/v1/skills", "u1")
	ok.Request.Header.SetMethod("POST")
	ok.Request.SetBody([]byte(`{"name":"My Skill","description":"d","body":"b"}`))
	h.HandleCreate(ok)
	if ok.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", ok.Response.StatusCode(), ok.Response.Body())
	}

	dup := newAuthedCtx("/v1/skills", "u2")
	dup.Request.Header.SetMethod("POST")
	dup.Request.SetBody([]byte(`{"name":"My Skill","description":"d","body":"b"}`))
	h.HandleCreate(dup)
	if dup.Response.StatusCode() != fasthttp.StatusConflict {
		t.Fatalf("expected 409 for taken slug, got %d body=%s", dup.Response.StatusCode(), dup.Response.Body())
	}
}

func TestSkillsDeleteOwnerOnly(t *testing.T) {
	store := &stubStore{
		bySlug: map[string]model.Skill{
			"mine": {ID: "mine", Slug: "mine", Name: "mine", OwnerID: "owner-1", CurrentVersion: "1.0.0"},
		},
	}
	h := &SkillsHandler{q: store}

	forbidden := newAuthedCtx("/v1/skills/mine", "owner-2")
	forbidden.SetUserValue("slug", "mine")
	forbidden.Request.Header.SetMethod("DELETE")
	h.HandleDelete(forbidden)
	if forbidden.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", forbidden.Response.StatusCode())
	}

	ok := newAuthedCtx("/v1/skills/mine", "owner-1")
	ok.SetUserValue("slug", "mine")
	ok.Request.Header.SetMethod("DELETE")
	h.HandleDelete(ok)
	if ok.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ok.Response.StatusCode(), ok.Response.Body())
	}
}

func TestSkillsFilesAndRaw(t *testing.T) {
	store := &stubStore{
		bySlug: map[string]model.Skill{
			"mine": {ID: "mine", Slug: "mine", Name: "mine", CurrentVersion: "1.0.0"},
		},
		files: map[string]map[string]map[string]fileBlob{
			"mine": {"1.0.0": {"README.md": {content: []byte("# hi"), mime: "text/markdown"}}},
		},
	}
	h := &SkillsHandler{q: store}

	manifest := newCtx("/v1/skills/mine/files")
	manifest.SetUserValue("slug", "mine")
	h.HandleFiles(manifest)
	m := decode(t, manifest)
	if c, _ := m["count"].(float64); c != 1 {
		t.Fatalf("expected count=1, got %v", m["count"])
	}

	raw := newCtx("/v1/skills/mine/files/README.md")
	raw.SetUserValue("slug", "mine")
	raw.SetUserValue("filepath", "README.md")
	h.HandleFileRaw(raw)
	if raw.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d", raw.Response.StatusCode())
	}
	if string(raw.Response.Body()) != "# hi" {
		t.Fatalf("unexpected raw body %q", raw.Response.Body())
	}
	if ct := string(raw.Response.Header.ContentType()); ct != "text/markdown" {
		t.Fatalf("unexpected content type %q", ct)
	}

	missing := newCtx("/v1/skills/mine/files/nope.txt")
	missing.SetUserValue("slug", "mine")
	missing.SetUserValue("filepath", "nope.txt")
	h.HandleFileRaw(missing)
	if missing.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", missing.Response.StatusCode())
	}
}

func TestSkillsReindex(t *testing.T) {
	h := &SkillsHandler{index: stubIndex{}}
	ctx := newCtx("/v1/skills/reindex")
	h.HandleReindex(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusAccepted {
		t.Fatalf("expected 202, got %d", ctx.Response.StatusCode())
	}

	disabled := NewSkillsHandler(nil, nil)
	dctx := newCtx("/v1/skills/reindex")
	disabled.HandleReindex(dctx)
	if dctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", dctx.Response.StatusCode())
	}
}

func TestSkillsNilSafe(t *testing.T) {
	h := NewSkillsHandler(nil, nil) // both seams nil — must not panic
	ctx := newCtx("/v1/skills/search?q=x")
	h.HandleSearch(ctx)
	if decode(t, ctx)["count"].(float64) != 0 {
		t.Fatal("expected empty count")
	}
}

func repeat(s string, n int) string {
	out := ""
	for range n {
		out += s
	}
	return out
}

func TestSkillsGetEncodedSlug(t *testing.T) {
	h := &SkillsHandler{q: &stubStore{bySlug: map[string]model.Skill{
		"hermes/dogfood": {ID: "hermes/dogfood", Slug: "hermes/dogfood", Body: "do qa", CurrentVersion: "1.0.0"},
	}}}
	ctx := newCtx("/v1/skills/hermes%2Fdogfood")
	// fasthttp hands the router param back still-encoded; the handler must
	// unescape before the DB lookup.
	ctx.SetUserValue("slug", "hermes%2Fdogfood")
	h.HandleGet(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}
