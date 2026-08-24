package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type EvalQueries struct {
	pool *pgxpool.Pool
}

func NewEvalQueries(pool *pgxpool.Pool) *EvalQueries {
	return &EvalQueries{pool: pool}
}

func (q *EvalQueries) Upsert(ctx context.Context, r model.EvalResult) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO eval_results (suite, case_id, model, passed, score, ttft_ms, total_ms,
		                           prompt_tokens, completion_tokens, tokens_per_sec, cost_micro_usd,
		                           answer_preview, error, ran_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 ON CONFLICT (suite, case_id, model) DO UPDATE SET
		   passed            = EXCLUDED.passed,
		   score             = EXCLUDED.score,
		   ttft_ms           = EXCLUDED.ttft_ms,
		   total_ms          = EXCLUDED.total_ms,
		   prompt_tokens     = EXCLUDED.prompt_tokens,
		   completion_tokens = EXCLUDED.completion_tokens,
		   tokens_per_sec    = EXCLUDED.tokens_per_sec,
		   cost_micro_usd    = EXCLUDED.cost_micro_usd,
		   answer_preview    = EXCLUDED.answer_preview,
		   error             = EXCLUDED.error,
		   ran_at            = EXCLUDED.ran_at`,
		r.Suite, r.CaseID, r.Model, r.Passed, r.Score, r.TTFTMs, r.TotalMs,
		r.PromptTokens, r.CompletionTokens, r.TokensPerSec, r.CostMicroUSD,
		r.AnswerPreview, r.Error, r.RanAt,
	)
	if err != nil {
		return fmt.Errorf("upsert eval result: %w", err)
	}
	return nil
}

func (q *EvalQueries) List(ctx context.Context) ([]model.EvalResult, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT suite, case_id, model, passed, score, ttft_ms, total_ms,
		        prompt_tokens, completion_tokens, tokens_per_sec, cost_micro_usd,
		        answer_preview, error, ran_at
		 FROM eval_results
		 ORDER BY suite ASC, case_id ASC, model ASC`)
	if err != nil {
		return nil, fmt.Errorf("list eval results: %w", err)
	}
	defer rows.Close()

	var out []model.EvalResult
	for rows.Next() {
		var r model.EvalResult
		if err := rows.Scan(
			&r.Suite, &r.CaseID, &r.Model, &r.Passed, &r.Score, &r.TTFTMs, &r.TotalMs,
			&r.PromptTokens, &r.CompletionTokens, &r.TokensPerSec, &r.CostMicroUSD,
			&r.AnswerPreview, &r.Error, &r.RanAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval result: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *EvalQueries) LatestRunAt(ctx context.Context) (*time.Time, error) {
	var t *time.Time
	err := q.pool.QueryRow(ctx, `SELECT MAX(ran_at) FROM eval_results`).Scan(&t)
	if err != nil {
		return nil, fmt.Errorf("latest eval run time: %w", err)
	}
	return t, nil
}

// LatestUsage reconciles a streamed request's exact token counts and cost from
// usage_logs. Streaming clients do not always receive a usage frame (provider
// dependent), but the gateway records one row per request internally.
func (q *EvalQueries) LatestUsage(ctx context.Context, apiKeyID string, modelName string, since time.Time) (tokensIn, tokensOut int, latencyMs, ttftMs int, tps float64, costCents int64, found bool, err error) {
	row := q.pool.QueryRow(ctx,
		`SELECT tokens_in, tokens_out, latency_ms, COALESCE(ttft_ms, 0), COALESCE(tps, 0), cost_cents
		 FROM usage_logs
		 WHERE api_key_id = $1 AND model = $2 AND created_at >= $3 AND error IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		apiKeyID, modelName, since)
	var in, out, lat, ttft int
	var rate float64
	var cents int64
	if err = row.Scan(&in, &out, &lat, &ttft, &rate, &cents); err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, 0, 0, 0, 0, false, nil
		}
		err = fmt.Errorf("latest usage for eval: %w", err)
		return
	}
	tokensIn, tokensOut, latencyMs, ttftMs, tps, costCents = in, out, lat, ttft, rate, cents
	found = true
	return
}
