package modelaccess

import (
	"context"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

type fakeChecker map[string]bool

func (f fakeChecker) ModelAllowed(_ context.Context, _ string, id string) (bool, string) {
	if f[strings.ToLower(id)] {
		return false, "blocked " + id
	}
	return true, ""
}

func TestFilterCandidatesBlocksAliasTargetAndKeepsAllowedFallback(t *testing.T) {
	candidates := []router.RouteCandidate{
		{ModelCfg: &model.ModelConfig{ID: "restricted", ProviderModelID: "vendor/restricted"}},
		{ModelCfg: &model.ModelConfig{ID: "allowed", ProviderModelID: "vendor/allowed"}},
	}
	got, err := FilterCandidates(context.Background(), fakeChecker{"restricted": true}, "user", "friendly-alias", candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ModelCfg.ID != "allowed" {
		t.Fatalf("filtered candidates = %#v", got)
	}
}

func TestFilterCandidatesRejectsDeniedRequestedAndProviderIDs(t *testing.T) {
	candidate := []router.RouteCandidate{{ModelCfg: &model.ModelConfig{ID: "canonical", ProviderModelID: "vendor/model"}}}
	for _, denied := range []string{"alias", "vendor/model"} {
		if _, err := FilterCandidates(context.Background(), fakeChecker{denied: true}, "user", "alias", candidate); err == nil {
			t.Fatalf("expected %s denial", denied)
		}
	}
}
