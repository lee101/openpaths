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
	r.RecordSuccessWithApp(userID, apiKeyID, modelName, providerName, tokensIn, tokensOut, latencyMs, tps, costCents, streaming, "", "", "", nil)
}

func (r *Recorder) RecordSuccessWithApp(
	userID, apiKeyID, modelName, providerName string,
	tokensIn, tokensOut, latencyMs int,
	tps float32,
	costCents int64,
	streaming bool,
	appID, appURL, appTitle string,
	appCategories []string,
) {
	r.RecordSuccessWithAppTimings(userID, apiKeyID, modelName, providerName, tokensIn, tokensOut, latencyMs, 0, tps, costCents, streaming, appID, appURL, appTitle, appCategories)
}

func (r *Recorder) RecordSuccessWithAppTimings(
	userID, apiKeyID, modelName, providerName string,
	tokensIn, tokensOut, latencyMs, ttftMs int,
	tps float32,
	costCents int64,
	streaming bool,
	appID, appURL, appTitle string,
	appCategories []string,
) {
	r.collector.Record(model.UsageLog{
		UserID:        userID,
		APIKeyID:      apiKeyID,
		Model:         modelName,
		Provider:      providerName,
		TokensIn:      tokensIn,
		TokensOut:     tokensOut,
		LatencyMs:     latencyMs,
		TTFTMs:        ttftMs,
		TPS:           tps,
		CostCents:     costCents,
		StatusCode:    200,
		Streaming:     streaming,
		AppID:         appID,
		AppURL:        appURL,
		AppTitle:      appTitle,
		AppCategories: appCategories,
		CreatedAt:     time.Now(),
	})
}

// RecordError records a failed API request.
func (r *Recorder) RecordError(
	userID, apiKeyID, modelName, providerName string,
	latencyMs, statusCode int,
	errMsg string,
	streaming bool,
) {
	r.RecordErrorWithApp(userID, apiKeyID, modelName, providerName, latencyMs, statusCode, errMsg, streaming, "", "", "", nil)
}

func (r *Recorder) RecordErrorWithApp(
	userID, apiKeyID, modelName, providerName string,
	latencyMs, statusCode int,
	errMsg string,
	streaming bool,
	appID, appURL, appTitle string,
	appCategories []string,
) {
	r.collector.Record(model.UsageLog{
		UserID:        userID,
		APIKeyID:      apiKeyID,
		Model:         modelName,
		Provider:      providerName,
		LatencyMs:     latencyMs,
		Error:         &errMsg,
		StatusCode:    statusCode,
		Streaming:     streaming,
		AppID:         appID,
		AppURL:        appURL,
		AppTitle:      appTitle,
		AppCategories: appCategories,
		CreatedAt:     time.Now(),
	})
}
