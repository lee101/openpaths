package queries

import (
	"context"
	"errors"
	"fmt"
	"github.com/openpaths/openpaths/internal/ttlcache"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

var (
	ErrKeyNotFound = errors.New("api key not found")
	ErrKeyRevoked  = errors.New("api key revoked")
	ErrKeyExpired  = errors.New("api key expired")
)

const apiKeyColumns = `id, user_id, key_hash, key_prefix, name, created_at, last_used_at, revoked, rate_limit_rpm,
	scopes, expires_at, revoked_at, rotated_from::text`

func scanAPIKey(row rowScanner) (*model.APIKey, error) {
	var k model.APIKey
	err := row.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt, &k.LastUsedAt,
		&k.Revoked, &k.RateLimitRPM, &k.Scopes, &k.ExpiresAt, &k.RevokedAt, &k.RotatedFrom)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func checkKeyActive(k *model.APIKey, now time.Time) error {
	if k.RevokedAsOf(now) {
		return ErrKeyRevoked
	}
	if k.ExpiredAsOf(now) {
		return ErrKeyExpired
	}
	return nil
}

type APIKeyQueries struct {
	pool     *pgxpool.Pool
	valid    *ttlcache.Cache[string, *model.APIKey]
	lastUsed *ttlcache.Cache[string, struct{}]
}

func NewAPIKeyQueries(pool *pgxpool.Pool) *APIKeyQueries {
	return &APIKeyQueries{
		pool:     pool,
		valid:    ttlcache.New[string, *model.APIKey](10*time.Second, 100000),
		lastUsed: ttlcache.New[string, struct{}](time.Minute, 100000),
	}
}

func (q *APIKeyQueries) forgetKeys() {
	if q.valid != nil {
		q.valid.Clear()
	}
}

func (q *APIKeyQueries) Create(ctx context.Context, userID, keyHash, keyPrefix, name string) (*model.APIKey, error) {
	return q.CreateScoped(ctx, userID, keyHash, keyPrefix, name, nil, nil)
}

func (q *APIKeyQueries) CreateScoped(ctx context.Context, userID, keyHash, keyPrefix, name string, scopes []string, expiresAt *time.Time) (*model.APIKey, error) {
	k, err := scanAPIKey(q.pool.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, scopes, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+apiKeyColumns,
		userID, keyHash, keyPrefix, name, scopes, expiresAt,
	))
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return k, nil
}

// SetRateLimit updates the per-minute rate limit for an API key. Used for the
// internal model-probe key, which fans out across all chat models and would
// otherwise hit the default 60 rpm limit.
func (q *APIKeyQueries) SetRateLimit(ctx context.Context, id string, rpm int) error {
	defer q.forgetKeys()
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
	if q.valid == nil {
		return q.validateKey(ctx, keyHash)
	}
	if k, ok := q.valid.Get(keyHash); ok {
		if err := checkKeyActive(k, time.Now()); err != nil {
			return nil, err
		}
		c := *k
		return &c, nil
	}
	k, err := q.validateKey(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	stored := *k
	q.valid.Set(keyHash, &stored)
	return k, nil
}

func (q *APIKeyQueries) validateKey(ctx context.Context, keyHash string) (*model.APIKey, error) {
	k, err := scanAPIKey(q.pool.QueryRow(ctx,
		`SELECT ak.id, ak.user_id, ak.key_hash, ak.key_prefix, ak.name, ak.created_at, ak.last_used_at,
		        ak.revoked, ak.rate_limit_rpm, ak.scopes, ak.expires_at, ak.revoked_at, ak.rotated_from::text
		 FROM api_keys ak
		 JOIN users u ON u.id = ak.user_id
		 WHERE ak.key_hash = $1 AND NOT u.disabled`,
		keyHash,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("validate api key: %w", err)
	}
	if err := checkKeyActive(k, time.Now()); err != nil {
		return nil, err
	}

	if q.lastUsed != nil {
		if _, ok := q.lastUsed.Get(k.ID); ok {
			return k, nil
		}
		q.lastUsed.Set(k.ID, struct{}{})
	}
	go func() {
		q.pool.Exec(context.Background(),
			"UPDATE api_keys SET last_used_at = now() WHERE id = $1", k.ID)
	}()

	return k, nil
}

func (q *APIKeyQueries) ListByUser(ctx context.Context, userID string) ([]model.APIKey, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT `+apiKeyColumns+`
		 FROM api_keys
		 WHERE user_id = $1 AND NOT revoked AND (revoked_at IS NULL OR revoked_at > now())
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, *k)
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
	defer q.forgetKeys()
	tag, err := q.pool.Exec(ctx,
		"UPDATE api_keys SET revoked = TRUE, revoked_at = now() WHERE id = $1 AND user_id = $2 AND NOT revoked",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrKeyNotFound
	}
	return nil
}

func (q *APIKeyQueries) RevokeMany(ctx context.Context, ids []string, userID string) error {
	defer q.forgetKeys()
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin revoke api keys: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, id := range ids {
		if _, err := tx.Exec(ctx, "UPDATE api_keys SET revoked = TRUE, revoked_at = now() WHERE id = $1 AND user_id = $2 AND NOT revoked", id, userID); err != nil {
			return fmt.Errorf("revoke api keys: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (q *APIKeyQueries) Rotate(ctx context.Context, id, userID, newHash, newPrefix string, grace time.Duration) (newKey, oldKey *model.APIKey, err error) {
	defer q.forgetKeys()
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin rotate api key: %w", err)
	}
	defer tx.Rollback(ctx)

	oldKey, err = scanAPIKey(tx.QueryRow(ctx,
		`SELECT `+apiKeyColumns+` FROM api_keys WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load api key: %w", err)
	}
	if oldKey.Revoked || oldKey.RevokedAt != nil {
		return nil, nil, ErrKeyRevoked
	}
	if oldKey.ExpiredAsOf(time.Now()) {
		return nil, nil, ErrKeyExpired
	}

	newKey, err = scanAPIKey(tx.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, scopes, expires_at, rate_limit_rpm, rotated_from)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+apiKeyColumns,
		userID, newHash, newPrefix, oldKey.Name, oldKey.Scopes, oldKey.ExpiresAt, oldKey.RateLimitRPM, oldKey.ID))
	if err != nil {
		return nil, nil, fmt.Errorf("create rotated api key: %w", err)
	}

	if grace <= 0 {
		err = tx.QueryRow(ctx,
			`UPDATE api_keys SET revoked = TRUE, revoked_at = now() WHERE id = $1 RETURNING revoked_at`, oldKey.ID).Scan(&oldKey.RevokedAt)
		oldKey.Revoked = true
	} else {
		err = tx.QueryRow(ctx,
			`UPDATE api_keys SET revoked_at = now() + make_interval(secs => $2) WHERE id = $1 RETURNING revoked_at`,
			oldKey.ID, grace.Seconds()).Scan(&oldKey.RevokedAt)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("retire api key: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO guardrail_assignments (guardrail_id, target_type, target_id)
		 SELECT guardrail_id, target_type, $2 FROM guardrail_assignments WHERE target_type = 'api_key' AND target_id = $1
		 ON CONFLICT DO NOTHING`, oldKey.ID, newKey.ID); err != nil {
		return nil, nil, fmt.Errorf("copy guardrail assignments: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit rotate api key: %w", err)
	}
	return newKey, oldKey, nil
}
