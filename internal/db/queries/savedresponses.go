package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type SavedResponseQueries struct {
	pool *pgxpool.Pool
}

func NewSavedResponseQueries(pool *pgxpool.Pool) *SavedResponseQueries {
	return &SavedResponseQueries{pool: pool}
}

const savedRespCols = `id, user_id, api_key_id, kind, model, provider, prompt, input, output,
	image_url, thumb_url, width, height, tokens_in, tokens_out, cost_cents, created_at`

func scanSavedResponse(row interface{ Scan(dest ...any) error }) (*model.SavedResponse, error) {
	var r model.SavedResponse
	err := row.Scan(
		&r.ID, &r.UserID, &r.APIKeyID, &r.Kind, &r.Model, &r.Provider, &r.Prompt, &r.Input, &r.Output,
		&r.ImageURL, &r.ThumbURL, &r.Width, &r.Height, &r.TokensIn, &r.TokensOut, &r.CostCents, &r.CreatedAt,
	)
	return &r, err
}

// Insert persists a saved response. The embedding may be nil (embedded later) or a
// 512×float32 little-endian byte slice.
func (q *SavedResponseQueries) Insert(ctx context.Context, r *model.SavedResponse) error {
	var embedding any
	if len(r.Embedding) > 0 {
		embedding = r.Embedding
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO saved_responses
		 (user_id, api_key_id, kind, model, provider, prompt, input, output,
		  image_url, thumb_url, width, height, tokens_in, tokens_out, cost_cents, embedding)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		r.UserID, r.APIKeyID, r.Kind, r.Model, r.Provider, r.Prompt, r.Input, r.Output,
		r.ImageURL, r.ThumbURL, r.Width, r.Height, r.TokensIn, r.TokensOut, r.CostCents, embedding,
	)
	if err != nil {
		return fmt.Errorf("insert saved response: %w", err)
	}
	return nil
}

// List returns a user's saved responses for a kind, newest first.
func (q *SavedResponseQueries) List(ctx context.Context, userID, kind string, limit, offset int) ([]model.SavedResponse, error) {
	limit, offset = clampPage(limit, offset)
	rows, err := q.pool.Query(ctx,
		`SELECT `+savedRespCols+` FROM saved_responses
		 WHERE user_id = $1 AND kind = $2
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, kind, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list saved responses: %w", err)
	}
	return collectSavedResponses(rows)
}

// CountByKind returns how many saved responses a user has of a given kind.
func (q *SavedResponseQueries) CountByKind(ctx context.Context, userID, kind string) (int, error) {
	var n int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM saved_responses WHERE user_id = $1 AND kind = $2`,
		userID, kind,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count saved responses: %w", err)
	}
	return n, nil
}

// Get returns a single saved response owned by the user.
func (q *SavedResponseQueries) Get(ctx context.Context, userID, id string) (*model.SavedResponse, error) {
	r, err := scanSavedResponse(q.pool.QueryRow(ctx,
		`SELECT `+savedRespCols+` FROM saved_responses WHERE user_id = $1 AND id = $2`,
		userID, id,
	))
	if err != nil {
		return nil, fmt.Errorf("get saved response: %w", err)
	}
	return r, nil
}

// SearchTrigram does exact/substring/fuzzy search over prompt + output.
func (q *SavedResponseQueries) SearchTrigram(ctx context.Context, userID, kind, query string, limit, offset int) ([]model.SavedResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return q.List(ctx, userID, kind, limit, offset)
	}
	limit, offset = clampPage(limit, offset)
	rows, err := q.pool.Query(ctx,
		`SELECT `+savedRespCols+` FROM saved_responses
		 WHERE user_id = $1 AND kind = $2
		   AND (prompt ILIKE '%' || $3 || '%' OR output ILIKE '%' || $3 || '%'
		        OR prompt % $3 OR output % $3)
		 ORDER BY GREATEST(similarity(prompt, $3), similarity(output, $3)) DESC, created_at DESC
		 LIMIT $4 OFFSET $5`,
		userID, kind, query, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search saved responses: %w", err)
	}
	return collectSavedResponses(rows)
}

// ListVectors loads all of a user's embedded responses of a kind for in-process
// semantic search. Rows without an embedding are skipped.
func (q *SavedResponseQueries) ListVectors(ctx context.Context, userID, kind string, cap int) ([]model.SavedResponse, error) {
	if cap <= 0 || cap > 100000 {
		cap = 50000
	}
	rows, err := q.pool.Query(ctx,
		`SELECT `+savedRespCols+`, embedding FROM saved_responses
		 WHERE user_id = $1 AND kind = $2 AND embedding IS NOT NULL
		 ORDER BY created_at DESC LIMIT $3`,
		userID, kind, cap,
	)
	if err != nil {
		return nil, fmt.Errorf("list saved response vectors: %w", err)
	}
	defer rows.Close()
	out := make([]model.SavedResponse, 0, 256)
	for rows.Next() {
		var r model.SavedResponse
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.APIKeyID, &r.Kind, &r.Model, &r.Provider, &r.Prompt, &r.Input, &r.Output,
			&r.ImageURL, &r.ThumbURL, &r.Width, &r.Height, &r.TokensIn, &r.TokensOut, &r.CostCents, &r.CreatedAt,
			&r.Embedding,
		); err != nil {
			return nil, fmt.Errorf("scan saved response vector: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func collectSavedResponses(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]model.SavedResponse, error) {
	defer rows.Close()
	out := make([]model.SavedResponse, 0, 48)
	for rows.Next() {
		r, err := scanSavedResponse(rows)
		if err != nil {
			return nil, fmt.Errorf("scan saved response: %w", err)
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func clampPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 200 {
		limit = 48
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
