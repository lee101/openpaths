package metrics

import (
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// Recorder provides a convenient API for recording request metrics.
type Recorder struct {
	collector *Collector
}

func NewRecorder(collector *Collector) *Recorder {
	return &Recorder{collector: collector}
}

// RecordSuccess records a successful API request.
func (r *Recorder) RecordSuccess(
	userID, apiKeyID, modelName, providerName string,
	tokensIn, tokensOut, latencyMs int,
	tps float32,
	costCents int64,
	streaming bool,
) {
	r.collector.Record(model.UsageLog{
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Model:      modelName,
		Provider:   providerName,
		TokensIn:   tokensIn,
		TokensOut:  tokensOut,
		LatencyMs:  latencyMs,
		TPS:        tps,
		CostCents:  costCents,
		StatusCode: 200,
		Streaming:  streaming,
		CreatedAt:  time.Now(),
	})
}

// RecordError records a failed API request.
func (r *Recorder) RecordError(
	userID, apiKeyID, modelName, providerName string,
	latencyMs, statusCode int,
	errMsg string,
	streaming bool,
) {
	r.collector.Record(model.UsageLog{
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Model:      modelName,
		Provider:   providerName,
		LatencyMs:  latencyMs,
		Error:      &errMsg,
		StatusCode: statusCode,
		Streaming:  streaming,
		CreatedAt:  time.Now(),
	})
}
