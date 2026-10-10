package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

func TestOpenAICompatModelsURLDoesNotDuplicateVersion(t *testing.T) {
	for _, tc := range []struct {
		base, want string
	}{
		{"https://api.meta.ai/v1", "https://api.meta.ai/v1/models"},
		{"https://api.meta.ai/v1/", "https://api.meta.ai/v1/models"},
		{"https://api.openai.com", "https://api.openai.com/v1/models"},
	} {
		if got := openAICompatModelsURL(tc.base); got != tc.want {
			t.Errorf("openAICompatModelsURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}

type fakeMeta struct {
	rows map[string]bool
}

func (f *fakeMeta) Upsert(ctx context.Context, m *queries.ModelMetadata) (bool, error) {
	k := m.Provider + "|" + m.ModelID
	inserted := !f.rows[k]
	f.rows[k] = true
	return inserted, nil
}

func (f *fakeMeta) CountByProvider(ctx context.Context, provider string) (int, error) {
	n := 0
	for k := range f.rows {
		if strings.HasPrefix(k, provider+"|") {
			n++
		}
	}
	return n, nil
}

func TestDiscoverAllSurfacesOnlyNewModelsAfterBootstrap(t *testing.T) {
	ids := []string{"m1", "m2"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data []map[string]string
		for _, id := range ids {
			data = append(data, map[string]string{"id": id})
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	s := New([]model.ProviderConfig{{Name: "groq", Enabled: true, APIKey: "k", BaseURL: srv.URL}}, nil)
	s.modelQ = &fakeMeta{rows: map[string]bool{}}

	if n, _ := s.DiscoverAll(context.Background()); n != 2 {
		t.Fatalf("indexed %d", n)
	}
	if got := s.NewlyDiscovered(); len(got) != 0 {
		t.Fatalf("bootstrap run flagged %v", got)
	}

	ids = append(ids, "m3")
	s.DiscoverAll(context.Background())
	s.DiscoverAll(context.Background())
	got := s.NewlyDiscovered()
	if len(got) != 1 || got[0].Provider != "groq" || got[0].ModelID != "m3" {
		t.Fatalf("got %+v", got)
	}
}
