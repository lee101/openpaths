package guardrails

import (
	"fmt"

	"github.com/openpaths/openpaths/internal/router"
)

type routeCand struct{ c router.RouteCandidate }

func (r routeCand) ProviderName() string {
	if r.c.ModelCfg != nil && r.c.ModelCfg.Provider != "" {
		return r.c.ModelCfg.Provider
	}
	if r.c.Provider != nil {
		return r.c.Provider.Name()
	}
	return ""
}

// FilterRouteCandidates applies provider allowlist to router candidates.
func FilterRouteCandidates(candidates []router.RouteCandidate, allowed []string) ([]router.RouteCandidate, error) {
	if len(allowed) == 0 {
		return candidates, nil
	}
	out := make([]router.RouteCandidate, 0, len(candidates))
	for _, c := range candidates {
		if ProviderAllowed(allowed, routeCand{c}.ProviderName()) {
			out = append(out, c)
		}
	}
	if len(out) == 0 && len(candidates) > 0 {
		return nil, fmt.Errorf("no providers allowed by guardrail")
	}
	return out, nil
}
