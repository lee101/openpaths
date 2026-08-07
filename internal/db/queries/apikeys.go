package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type APIKeyQueries struct {
	pool *pgxpool.Pool
}

func NewAPIKeyQueries(pool *pgxpool.Pool) *APIKeyQueries {
	return &APIKeyQueries{pool: pool}
}

func (q *APIKeyQueries) Create(ctx context.Context, userID, keyHash, keyPrefix, name string) (*model.APIKey, error) {
	var k model.APIKey
	err := q.pool.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked, rate_limit_rpm`,
		userID, keyHash, keyPrefix, name,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt, &k.LastUsedAt, &k.Revoked, &k.RateLimitRPM)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &k, nil
}

// SetRateLimit updates the per-minute rate limit for an API key. Used for the
// internal model-probe key, which fans out across all chat models and would
// otherwise hit the default 60 rpm limit.
func (q *APIKeyQueries) SetRateLimit(ctx context.Context, id string, rpm int) error {
	_, err := q.pool.Exec(ctx,
		"UPDATE api_keys SET rate_limit_rpm = $1 WHERE id = $2",
		rpm, id,
	)
	if err != nil {
		return fmt.Errorf("set api key rate limit: %w", err)
	}
	return nil
}

func (q *APIKeyQueries) ValidateKey(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var k model.APIKey
	err := q.pool.QueryRow(ctx,
		`SELECT ak.id, ak.user_id, ak.key_hash, ak.key_prefix, ak.name, ak.created_at,
		        ak.last_used_at, ak.revoked, ak.rate_limit_rpm
		 FROM api_keys ak
		 JOIN users u ON u.id = ak.user_id
		 WHERE ak.key_hash = $1 AND NOT ak.revoked AND NOT u.disabled`,
		keyHash,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt,
		&k.LastUsedAt, &k.Revoked, &k.RateLimitRPM)
	if err != nil {
		return nil, fmt.Errorf("validate api key: %w", err)
	}

	// Update last_used_at in the background (fire and forget)
	go func() {
		q.pool.Exec(context.Background(),
			"UPDATE api_keys SET last_used_at = now() WHERE id = $1", k.ID)
	}()

	return &k, nil
}

func (q *APIKeyQueries) ListByUser(ctx context.Context, userID string) ([]model.APIKey, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, user_id, key_prefix, name, created_at, last_used_at, revoked, rate_limit_rpm
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyPrefix, &k.Name, &k.CreatedAt,
			&k.LastUsedAt, &k.Revoked, &k.RateLimitRPM); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (q *APIKeyQueries) GetFirstByUser(ctx context.Context, userID string) (*model.APIKey, error) {
	var k model.APIKey
	err := q.pool.QueryRow(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked, rate_limit_rpm
		 FROM api_keys WHERE user_id = $1 AND NOT revoked ORDER BY created_at ASC LIMIT 1`,
		userID,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt, &k.LastUsedAt, &k.Revoked, &k.RateLimitRPM)
	if err != nil {
		return nil, fmt.Errorf("get first api key: %w", err)
	}
	return &k, nil
}

func (q *APIKeyQueries) Revoke(ctx context.Context, id, userID string) error {
	tag, err := q.pool.Exec(ctx,
		"UPDATE api_keys SET revoked = TRUE WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}
