package alibaba

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

// Decide calls decision-model-preview, which speaks the TypeSafe /v1/systemone schema.
func (p *Provider) Decide(ctx context.Context, req *model.DecisionRequest) (*model.DecisionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+"/compatible-mode/v1/systemone", body, false)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	if status != http.StatusOK {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &e)
		msg := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(e.Code+" "+e.Message), string(responseBody)))
		if status == http.StatusBadRequest {
			status = http.StatusUnprocessableEntity
		}
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: msg, Retryable: status == 429 || status == 529 || status >= 500}
	}
	var out model.DecisionResponse
	if err := json.Unmarshal(responseBody, &out); err != nil || len(out.Answers) == 0 {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "decision model returned no answers", Retryable: true}
	}
	return &out, nil
}
