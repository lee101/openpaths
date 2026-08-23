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

// FilterRouteCandidates applies provider allow and deny rules to router
// candidates. Deny rules win when a provider appears in both lists.
func FilterRouteCandidates(candidates []router.RouteCandidate, allowed, blocked []string) ([]router.RouteCandidate, error) {
	if len(allowed) == 0 && len(blocked) == 0 {
		return candidates, nil
	}
	out := make([]router.RouteCandidate, 0, len(candidates))
	for _, c := range candidates {
		provider := routeCand{c}.ProviderName()
		if !ProviderBlocked(blocked, provider) && ProviderAllowed(allowed, provider) {
			out = append(out, c)
		}
	}
	if len(out) == 0 && len(candidates) > 0 {
		return nil, fmt.Errorf("no providers allowed by guardrail")
	}
	return out, nil
}
