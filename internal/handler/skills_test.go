package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
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

type stubStore struct {
	skills []model.Skill
	bySlug map[string]model.Skill
}

func (s stubStore) List(_ context.Context, _ model.SkillFilters, _, _ int) ([]model.Skill, error) {
	return s.skills, nil
}
func (s stubStore) SearchILIKE(_ context.Context, _ string, _ model.SkillFilters, _ int) ([]model.Skill, error) {
	return s.skills, nil
}
func (s stubStore) GetBySlug(_ context.Context, slug string) (*model.Skill, error) {
	if sk, ok := s.bySlug[slug]; ok {
		return &sk, nil
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

func newCtx(uri string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI(uri)
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
		q:     stubStore{},
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
		q:     stubStore{skills: []model.Skill{{Slug: "hermes/docker", Name: "docker"}}},
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
	h := &SkillsHandler{q: stubStore{bySlug: map[string]model.Skill{
		"hermes/dogfood": {Slug: "hermes/dogfood", Body: "do qa", SetupPreamble: "## Setup\ngit clone x"},
	}}}

	ok := newCtx("/v1/skills/hermes/dogfood")
	ok.SetUserValue("slug", "hermes/dogfood")
	h.HandleGet(ok)
	m := decode(t, ok)
	md, _ := m["markdown"].(string)
	if md == "" || md[:8] != "## Setup" {
		t.Fatalf("expected markdown with Setup preamble, got %q", md)
	}

	miss := newCtx("/v1/skills/nope")
	miss.SetUserValue("slug", "nope")
	h.HandleGet(miss)
	if miss.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", miss.Response.StatusCode())
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
