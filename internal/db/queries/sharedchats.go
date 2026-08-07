package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SharedChatQueries is the data layer for shared chat transcripts (shared_chats).
type SharedChatQueries struct {
	pool *pgxpool.Pool
}

func NewSharedChatQueries(pool *pgxpool.Pool) *SharedChatQueries {
	return &SharedChatQueries{pool: pool}
}

// SharedChat is a public read-only chat transcript.
type SharedChat struct {
	ID           string          `json:"id"`
	Slug         string          `json:"slug"`
	Title        string          `json:"title"`
	Model        string          `json:"model"`
	SystemPrompt string          `json:"system_prompt"`
	Messages     json.RawMessage `json:"messages"`
	UserID       string          `json:"user_id,omitempty"`
	Views        int             `json:"views"`
	CreatedAt    string          `json:"created_at"`
}

const sharedChatColumns = `id, slug, title, model, system_prompt, messages, COALESCE(user_id, ''), views, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanSharedChat(row pgx.Row) (*SharedChat, error) {
	var c SharedChat
	err := row.Scan(&c.ID, &c.Slug, &c.Title, &c.Model, &c.SystemPrompt, &c.Messages, &c.UserID, &c.Views, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Insert stores a shared chat and returns its id.
func (q *SharedChatQueries) Insert(ctx context.Context, slug, title, modelID, systemPrompt string, messages []byte, userID string) (string, error) {
	var id string
	err := q.pool.QueryRow(ctx,
		`INSERT INTO shared_chats (slug, title, model, system_prompt, messages, user_id)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		 RETURNING id`,
		slug, title, modelID, systemPrompt, messages, userID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert shared chat: %w", err)
	}
	return id, nil
}

// GetBySlug fetches a shared chat and atomically increments its view count.
func (q *SharedChatQueries) GetBySlug(ctx context.Context, slug string) (*SharedChat, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, pgx.ErrNoRows
	}
	return scanSharedChat(q.pool.QueryRow(ctx,
		fmt.Sprintf(`UPDATE shared_chats SET views = views + 1 WHERE slug = $1 RETURNING %s`, sharedChatColumns), slug))
}

// ListByUser returns a user's shared chats, newest first.
func (q *SharedChatQueries) ListByUser(ctx context.Context, userID string, limit int) ([]SharedChat, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM shared_chats WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, sharedChatColumns),
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list shared chats: %w", err)
	}
	defer rows.Close()
	out := make([]SharedChat, 0, limit)
	for rows.Next() {
		c, err := scanSharedChat(rows)
		if err != nil {
			return nil, fmt.Errorf("scan shared chat: %w", err)
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}
