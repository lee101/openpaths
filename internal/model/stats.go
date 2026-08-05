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
	AvgTTFTMs      float64 `json:"avg_ttft_ms"`
	AvgTPS         float64 `json:"avg_tps"`
	ErrorRate      float64 `json:"error_rate"`
	TotalCostCents int64   `json:"total_cost_cents"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type ModelDailyUsagePoint struct {
	Date     time.Time `json:"date"`
	Model    string    `json:"model"`
	Provider string    `json:"provider"`
	Requests int64     `json:"requests"`
}

type StatsResponse struct {
	Period    string            `json:"period"`
	Models    []ModelStats      `json:"models"`
	Latency   []TimeSeriesPoint `json:"latency,omitempty"`
	TPS       []TimeSeriesPoint `json:"tps,omitempty"`
	Requests  []TimeSeriesPoint `json:"requests,omitempty"`
	ErrorRate []TimeSeriesPoint `json:"error_rate,omitempty"`
}

type UsageBreakdown struct {
	Task           string  `json:"task"`
	Provider       string  `json:"provider"`
	Model          string  `json:"model"`
	TotalRequests  int64   `json:"total_requests"`
	TotalTokensIn  int64   `json:"total_tokens_in"`
	TotalTokensOut int64   `json:"total_tokens_out"`
	TotalCostCents int64   `json:"total_cost_cents"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	AvgTTFTMs      float64 `json:"avg_ttft_ms"`
	AvgTPS         float64 `json:"avg_tps"`
	ErrorRate      float64 `json:"error_rate"`
}

// APIKeySpend is spend/usage aggregated per API key.
type APIKeySpend struct {
	APIKeyID       string `json:"api_key_id"`
	KeyPrefix      string `json:"key_prefix"`
	KeyName        string `json:"key_name"`
	TotalRequests  int64  `json:"total_requests"`
	TotalCostCents int64  `json:"total_cost_cents"`
}

// ProviderSpend is spend/usage aggregated per provider.
type ProviderSpend struct {
	Provider       string `json:"provider"`
	TotalRequests  int64  `json:"total_requests"`
	TotalCostCents int64  `json:"total_cost_cents"`
}

// ProductSpend is spend/usage aggregated per product (capability/modality:
// chat, image, video, music, speech, transcription, embedding, 3d).
type ProductSpend struct {
	Product        string `json:"product"`
	TotalRequests  int64  `json:"total_requests"`
	TotalTokensIn  int64  `json:"total_tokens_in"`
	TotalTokensOut int64  `json:"total_tokens_out"`
	TotalCostCents int64  `json:"total_cost_cents"`
}

// DailyActivityPoint is one day's usage, used for the contribution heatmap.
type DailyActivityPoint struct {
	Date           string `json:"date"` // YYYY-MM-DD
	TotalRequests  int64  `json:"total_requests"`
	TotalCostCents int64  `json:"total_cost_cents"`
}

// AdminUserSpend is the per-user rollup shown in the admin spend dashboard.
type AdminUserSpend struct {
	UserID                string     `json:"user_id"`
	Email                 string     `json:"email"`
	Name                  string     `json:"name"`
	CreatedAt             time.Time  `json:"created_at"`
	LastRequestAt         *time.Time `json:"last_request_at,omitempty"`
	Disabled              bool       `json:"disabled"`
	IsAdmin               bool       `json:"is_admin"`
	BalanceCents          int64      `json:"balance_cents"`
	StripeGrossCents      int64      `json:"stripe_gross_cents"`
	StripeRefundedCents   int64      `json:"stripe_refunded_cents"`
	StripeNetCents        int64      `json:"stripe_net_cents"`
	APIRequests           int64      `json:"api_requests"`
	APISpendCents         int64      `json:"api_spend_cents"`
	ProviderBaseCostCents int64      `json:"provider_base_cost_cents"`
	ProviderEstimated     bool       `json:"provider_estimated"`
}
