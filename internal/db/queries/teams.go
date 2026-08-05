package queries

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errInviteInvalid = errors.New("invalid invite")
	errInviteUsed    = errors.New("invite already used")
	errInviteExpired = errors.New("invite expired")
	errInviteEmail   = errors.New("invite belongs to a different email address")
)

// InviteErrInvalid/Used/Expired expose invite failures for handler messaging.
func InviteErrInvalid() error { return errInviteInvalid }
func InviteErrUsed() error    { return errInviteUsed }
func InviteErrExpired() error { return errInviteExpired }

// TeamQueries backs teams (orgs), membership, and email invites.
type TeamQueries struct {
	pool *pgxpool.Pool
}

func NewTeamQueries(pool *pgxpool.Pool) *TeamQueries {
	return &TeamQueries{pool: pool}
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role,omitempty"`
}

type TeamMember struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (q *TeamQueries) Create(ctx context.Context, name, slug, createdBy string) (Team, error) {
	var t Team
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return t, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx,
		`INSERT INTO teams (name, slug, created_by) VALUES ($1, $2, $3) RETURNING id::text, name, slug`,
		name, slug, createdBy).Scan(&t.ID, &t.Name, &t.Slug); err != nil {
		return t, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, 'owner')`,
		t.ID, createdBy); err != nil {
		return t, err
	}
	t.Role = "owner"
	return t, tx.Commit(ctx)
}

func (q *TeamQueries) BySlug(ctx context.Context, slug string) (Team, error) {
	var t Team
	err := q.pool.QueryRow(ctx, "SELECT id::text, name, slug FROM teams WHERE slug = $1", slug).Scan(&t.ID, &t.Name, &t.Slug)
	return t, err
}

func (q *TeamQueries) ListForUser(ctx context.Context, userID string) ([]Team, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT t.id::text, t.name, t.slug, m.role FROM teams t
		 JOIN team_members m ON m.team_id = t.id WHERE m.user_id = $1 ORDER BY t.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Role); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (q *TeamQueries) Role(ctx context.Context, userID, teamID string) string {
	var role string
	if q.pool.QueryRow(ctx, "SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2", teamID, userID).Scan(&role) != nil {
		return ""
	}
	return role
}

func (q *TeamQueries) Members(ctx context.Context, teamID string) ([]TeamMember, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT m.user_id::text, u.email, m.role, m.created_at FROM team_members m
		 JOIN users u ON u.id = m.user_id WHERE m.team_id = $1 ORDER BY m.created_at`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TeamMember{}
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q *TeamQueries) AddMember(ctx context.Context, teamID, userID, role string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (team_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		teamID, userID, role)
	return err
}

func (q *TeamQueries) RemoveMember(ctx context.Context, teamID, userID string) error {
	_, err := q.pool.Exec(ctx, "DELETE FROM team_members WHERE team_id = $1 AND user_id = $2", teamID, userID)
	return err
}

func (q *TeamQueries) CreateInvite(ctx context.Context, teamID, email, role, tokenHash, invitedBy string, expiresAt time.Time) (string, error) {
	var id string
	err := q.pool.QueryRow(ctx,
		`INSERT INTO team_invites (team_id, email, role, token_hash, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		teamID, email, role, tokenHash, invitedBy, expiresAt).Scan(&id)
	return id, err
}

type TeamInvite struct {
	ID       string
	Email    string
	Role     string
	Expires  time.Time
	Accepted *time.Time
}

func validateInvite(inv TeamInvite, accountEmail string, now time.Time) error {
	if inv.Accepted != nil {
		return errInviteUsed
	}
	if now.After(inv.Expires) {
		return errInviteExpired
	}
	if !strings.EqualFold(strings.TrimSpace(inv.Email), strings.TrimSpace(accountEmail)) {
		return errInviteEmail
	}
	return nil
}

// AcceptInvite locks and validates a single-use invite, adds the invited user,
// and consumes the token in one transaction. A failed membership write rolls
// back the token consumption, while the row lock serializes racing acceptors.
func (q *TeamQueries) AcceptInvite(ctx context.Context, teamID, tokenHash, userID, accountEmail string) (string, error) {
	var inv TeamInvite
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx,
		`SELECT id::text, email, role, expires_at, accepted_at
		 FROM team_invites WHERE team_id = $1 AND token_hash = $2 FOR UPDATE`,
		teamID, tokenHash).Scan(&inv.ID, &inv.Email, &inv.Role, &inv.Expires, &inv.Accepted)
	if err != nil {
		return "", errInviteInvalid
	}
	if err := validateInvite(inv, accountEmail, time.Now()); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (team_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		teamID, userID, inv.Role); err != nil {
		return "", err
	}
	tag, err := tx.Exec(ctx, "UPDATE team_invites SET accepted_at = now() WHERE id = $1 AND accepted_at IS NULL", inv.ID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", errInviteUsed
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return inv.Role, nil
}
