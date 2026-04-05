package queries

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserProviderKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Provider  string    `json:"provider"`
	APIKey    string    `json:"api_key,omitempty"`
	AuthJSON  string    `json:"auth_json,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProviderKeyQueries struct {
	pool *pgxpool.Pool
}

func NewProviderKeyQueries(pool *pgxpool.Pool) *ProviderKeyQueries {
	return &ProviderKeyQueries{pool: pool}
}

func (q *ProviderKeyQueries) Upsert(ctx context.Context, userID, provider, apiKey, authJSON string) error {
	_, err := q.pool.Exec(ctx, `
		INSERT INTO user_provider_keys (user_id, provider, api_key, auth_json)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET api_key = $3, auth_json = $4, updated_at = now()
	`, userID, provider, apiKey, authJSON)
	return err
}

func (q *ProviderKeyQueries) ListByUser(ctx context.Context, userID string) ([]*UserProviderKey, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, provider, api_key, auth_json, created_at, updated_at
		FROM user_provider_keys WHERE user_id = $1 ORDER BY provider
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*UserProviderKey
	for rows.Next() {
		k := &UserProviderKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Provider, &k.APIKey, &k.AuthJSON, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (q *ProviderKeyQueries) GetByUserAndProvider(ctx context.Context, userID, provider string) (*UserProviderKey, error) {
	k := &UserProviderKey{}
	err := q.pool.QueryRow(ctx, `
		SELECT id, user_id, provider, api_key, auth_json, created_at, updated_at
		FROM user_provider_keys WHERE user_id = $1 AND provider = $2
	`, userID, provider).Scan(&k.ID, &k.UserID, &k.Provider, &k.APIKey, &k.AuthJSON, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (q *ProviderKeyQueries) Delete(ctx context.Context, userID, provider string) error {
	_, err := q.pool.Exec(ctx, `
		DELETE FROM user_provider_keys WHERE user_id = $1 AND provider = $2
	`, userID, provider)
	return err
}

func (q *ProviderKeyQueries) ListUsersWithProvider(ctx context.Context, provider string) ([]*UserProviderKey, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id, user_id, provider, api_key, auth_json, created_at, updated_at
		FROM user_provider_keys WHERE provider = $1 AND (api_key != '' OR auth_json != '')
	`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*UserProviderKey
	for rows.Next() {
		k := &UserProviderKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.Provider, &k.APIKey, &k.AuthJSON, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}
