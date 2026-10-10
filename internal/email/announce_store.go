package email

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgAnnounceStore struct {
	db *pgxpool.Pool
}

func (s *pgAnnounceStore) SyncCatalog(ctx context.Context, ids []string) ([]string, bool, error) {
	if ids == nil {
		ids = []string{}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('model_catalog_seen'))`); err != nil {
		return nil, false, err
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM model_catalog_seen`).Scan(&n); err != nil {
		return nil, false, err
	}
	if n == 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO model_catalog_seen (model_id, announced_at) SELECT DISTINCT unnest($1::text[]), now() ON CONFLICT DO NOTHING`, ids); err != nil {
			return nil, false, err
		}
		return nil, len(ids) > 0, tx.Commit(ctx)
	}
	rows, err := tx.Query(ctx,
		`INSERT INTO model_catalog_seen (model_id) SELECT DISTINCT unnest($1::text[]) ON CONFLICT DO NOTHING RETURNING model_id`, ids)
	if err != nil {
		return nil, false, err
	}
	var added []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, false, err
		}
		added = append(added, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return added, false, tx.Commit(ctx)
}

func (s *pgAnnounceStore) PendingCatalog(ctx context.Context) ([]CatalogEntry, error) {
	rows, err := s.db.Query(ctx, `SELECT model_id, first_seen_at FROM model_catalog_seen WHERE announced_at IS NULL ORDER BY first_seen_at, model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CatalogEntry{}
	for rows.Next() {
		var e CatalogEntry
		if err := rows.Scan(&e.ModelID, &e.FirstSeenAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *pgAnnounceStore) MarkAnnounced(ctx context.Context, modelID string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO model_catalog_seen (model_id, announced_at) VALUES ($1, now())
		 ON CONFLICT (model_id) DO UPDATE SET announced_at = now()`, modelID)
	return err
}

func (s *pgAnnounceStore) Users(ctx context.Context, modelID string) ([]announceUser, error) {
	rows, err := s.db.Query(ctx, `
		SELECT u.id::text, u.email, u.disabled,
		       EXISTS(SELECT 1 FROM email_suppressions s WHERE s.email = LOWER(u.email)),
		       COALESCE(a.status, '')
		FROM users u
		LEFT JOIN model_announcements a ON a.model_id = $1 AND a.user_id = u.id
		ORDER BY u.created_at, u.id`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []announceUser
	for rows.Next() {
		var u announceUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Disabled, &u.Suppressed, &u.Status); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *pgAnnounceStore) Reserve(ctx context.Context, modelID, userID, email string, retryPending bool) (bool, error) {
	var one int
	err := s.db.QueryRow(ctx, `
		INSERT INTO model_announcements (model_id, user_id, email, status) VALUES ($1, $2::uuid, $3, 'pending')
		ON CONFLICT (model_id, user_id) DO UPDATE
		SET status = 'pending', error = '', email = EXCLUDED.email, updated_at = now()
		WHERE model_announcements.status = 'error' OR ($4 AND model_announcements.status = 'pending')
		RETURNING 1`, modelID, userID, email, retryPending).Scan(&one)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *pgAnnounceStore) Mark(ctx context.Context, modelID, userID, status, errMsg string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE model_announcements SET status = $3, error = $4, updated_at = now() WHERE model_id = $1 AND user_id = $2::uuid`,
		modelID, userID, status, errMsg)
	return err
}

func (s *pgAnnounceStore) Status(ctx context.Context, modelID string) (AnnounceStatusRow, error) {
	row := AnnounceStatusRow{Counts: map[string]int{}}
	rows, err := s.db.Query(ctx, `SELECT status, count(*) FROM model_announcements WHERE model_id = $1 GROUP BY status`, modelID)
	if err != nil {
		return row, err
	}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return row, err
		}
		row.Counts[st] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return row, err
	}
	var at *time.Time
	if err := s.db.QueryRow(ctx, `SELECT announced_at FROM model_catalog_seen WHERE model_id = $1`, modelID).Scan(&at); err != nil && err != pgx.ErrNoRows {
		return row, err
	}
	row.AnnouncedAt = at
	erows, err := s.db.Query(ctx,
		`SELECT email || ': ' || error FROM model_announcements WHERE model_id = $1 AND status = 'error' ORDER BY updated_at DESC LIMIT 5`, modelID)
	if err != nil {
		return row, err
	}
	defer erows.Close()
	for erows.Next() {
		var e string
		if err := erows.Scan(&e); err != nil {
			return row, err
		}
		row.RecentErrors = append(row.RecentErrors, e)
	}
	return row, erows.Err()
}

func (s *pgAnnounceStore) Lock(ctx context.Context, modelID string) (func(), bool, error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended('announce:' || $1, 0))`, modelID).Scan(&ok); err != nil || !ok {
		conn.Release()
		return nil, false, err
	}
	return func() {
		bg := context.Background()
		if _, err := conn.Exec(bg, `SELECT pg_advisory_unlock(hashtextextended('announce:' || $1, 0))`, modelID); err != nil {
			conn.Conn().Close(bg)
		}
		conn.Release()
	}, true, nil
}
