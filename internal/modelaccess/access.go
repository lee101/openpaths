// Package modelaccess centralizes Model IAM checks at the point where a
// requested model has been expanded into concrete router candidates.
package modelaccess

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/openpaths/openpaths/internal/router"
)

// Checker is implemented by the Model IAM query store.
type Checker interface {
	ModelAllowed(context.Context, string, string) (bool, string)
}

// DeniedError identifies a policy denial without coupling callers to HTTP.
type DeniedError struct{ Reason string }

func (e *DeniedError) Error() string {
	if strings.TrimSpace(e.Reason) == "" {
		return "model is blocked by an access policy"
	}
	return e.Reason
}

// CheckIDs requires every supplied model identifier to be allowed. Callers use
// this for direct provider invocations that do not produce router candidates.
func CheckIDs(ctx context.Context, checker Checker, userID string, modelIDs ...string) error {
	if checkerIsNil(checker) || strings.TrimSpace(userID) == "" {
		return nil
	}
	seen := make(map[string]struct{}, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		key := strings.ToLower(modelID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if ok, reason := checker.ModelAllowed(ctx, userID, modelID); !ok {
			return &DeniedError{Reason: reason}
		}
	}
	return nil
}

func checkerIsNil(checker Checker) bool {
	if checker == nil {
		return true
	}
	v := reflect.ValueOf(checker)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// FilterCandidates enforces policy over the caller-visible ID and every
// resolved candidate ID. Denied fallbacks are removed; if none remain, the
// policy error is returned and no provider can be invoked.
func FilterCandidates(ctx context.Context, checker Checker, userID, requestedModel string, candidates []router.RouteCandidate) ([]router.RouteCandidate, error) {
	if err := CheckIDs(ctx, checker, userID, requestedModel); err != nil {
		return nil, err
	}
	if checkerIsNil(checker) || strings.TrimSpace(userID) == "" {
		return candidates, nil
	}
	allowed := make([]router.RouteCandidate, 0, len(candidates))
	var denied error
	for _, candidate := range candidates {
		if candidate.ModelCfg == nil {
			denied = &DeniedError{Reason: "resolved model has no configuration"}
			continue
		}
		err := CheckIDs(ctx, checker, userID, candidate.ModelCfg.ID, candidate.ModelCfg.ProviderModelID)
		if err != nil {
			denied = err
			continue
		}
		allowed = append(allowed, candidate)
	}
	if len(allowed) == 0 && denied != nil {
		return nil, denied
	}
	if len(allowed) == 0 && len(candidates) > 0 {
		return nil, fmt.Errorf("no usable model candidates")
	}
	return allowed, nil
}
