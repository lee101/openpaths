package model

import "time"

type AIApp struct {
	ID           string    `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Description  string    `json:"description"`
	FaviconURL   string    `json:"favicon_url"`
	Categories   []string  `json:"categories"`
	Source       string    `json:"source"`
	ExternalPath string    `json:"external_path"`
	TotalTokens  int64     `json:"total_tokens"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

type AppUsageStats struct {
	AppID         string          `json:"app_id"`
	Slug          string          `json:"slug"`
	Name          string          `json:"name"`
	URL           string          `json:"url"`
	Description   string          `json:"description"`
	FaviconURL    string          `json:"favicon_url"`
	Categories    []string        `json:"categories"`
	Source        string          `json:"source"`
	TotalRequests int64           `json:"total_requests"`
	TotalTokens   int64           `json:"total_tokens"`
	Models        []AppModelUsage `json:"models"`
}

type AppModelUsage struct {
	Model       string `json:"model"`
	Provider    string `json:"provider"`
	Requests    int64  `json:"requests"`
	TokensIn    int64  `json:"tokens_in"`
	TokensOut   int64  `json:"tokens_out"`
	TotalTokens int64  `json:"total_tokens"`
	Source      string `json:"source"`
}
