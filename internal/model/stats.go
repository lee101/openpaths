package model

import "time"

type ModelStats struct {
	Model          string  `json:"model"`
	Provider       string  `json:"provider"`
	TotalRequests  int64   `json:"total_requests"`
	TotalTokensIn  int64   `json:"total_tokens_in"`
	TotalTokensOut int64   `json:"total_tokens_out"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P50LatencyMs   float64 `json:"p50_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	AvgTPS         float64 `json:"avg_tps"`
	ErrorRate      float64 `json:"error_rate"`
	TotalCostCents int64   `json:"total_cost_cents"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type StatsResponse struct {
	Period    string            `json:"period"`
	Models    []ModelStats      `json:"models"`
	Latency   []TimeSeriesPoint `json:"latency,omitempty"`
	TPS       []TimeSeriesPoint `json:"tps,omitempty"`
	Requests  []TimeSeriesPoint `json:"requests,omitempty"`
	ErrorRate []TimeSeriesPoint `json:"error_rate,omitempty"`
}
