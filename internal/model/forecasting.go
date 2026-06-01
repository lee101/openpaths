package model

// ForecastingRequest is a time-series forecasting request (e.g. chronos2).
//
// Context/Series are interchangeable names for the historical observations;
// PredictionLength/Horizon are interchangeable names for how many future steps
// to predict. The handler normalises the aliases before dispatch so callers can
// use whichever naming their client library prefers.
type ForecastingRequest struct {
	Model            string    `json:"model"`
	Context          []float64 `json:"context,omitempty"`
	Series           []float64 `json:"series,omitempty"`
	PredictionLength int       `json:"prediction_length,omitempty"`
	Horizon          int       `json:"horizon,omitempty"`
	Quantiles        []float64 `json:"quantiles,omitempty"`
	Freq             string    `json:"freq,omitempty"`
}

// History returns the historical series, preferring Context then Series.
func (r *ForecastingRequest) History() []float64 {
	if len(r.Context) > 0 {
		return r.Context
	}
	return r.Series
}

// Steps returns the requested forecast horizon, preferring PredictionLength
// then Horizon, defaulting to 24 when neither is set.
func (r *ForecastingRequest) Steps() int {
	if r.PredictionLength > 0 {
		return r.PredictionLength
	}
	if r.Horizon > 0 {
		return r.Horizon
	}
	return 24
}

// ForecastingResponse is the normalised forecast result returned to callers.
type ForecastingResponse struct {
	Created   int64                `json:"created"`
	Model     string               `json:"model"`
	Forecast  []float64            `json:"forecast"`
	Quantiles map[string][]float64 `json:"quantiles,omitempty"`
}
