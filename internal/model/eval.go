package model

import "time"

// EvalSuite is one benchmark category rendered on /evals.
type EvalSuite string

const (
	EvalSuiteCoding  EvalSuite = "coding"
	EvalSuiteAgentic EvalSuite = "agentic"
	EvalSuiteSVG     EvalSuite = "creative"
)

// EvalResult is the outcome of one eval case for one model. Rows are keyed by
// (suite, case_id, model) and overwritten each sweep, so the table always holds
// the latest snapshot.
type EvalResult struct {
	Suite            string    `json:"suite"`
	CaseID           string    `json:"case_id"`
	Model            string    `json:"model"`
	Passed           bool      `json:"passed"`
	Score            float64   `json:"score"` // 0..1
	TTFTMs           int       `json:"ttft_ms"`
	TotalMs          int       `json:"total_ms"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TokensPerSec     float64   `json:"tokens_per_sec"`
	CostMicroUSD     int64     `json:"cost_micro_usd"`
	AnswerPreview    string    `json:"answer_preview"`
	Error            *string   `json:"error,omitempty"`
	RanAt            time.Time `json:"ran_at"`
}
