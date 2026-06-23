package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type UserQueries struct {
	pool *pgxpool.Pool
}

func NewUserQueries(pool *pgxpool.Pool) *UserQueries {
	return &UserQueries{pool: pool}
}

const userCols = `id, email, password_hash, name, created_at, updated_at, disabled, is_admin,
	stripe_customer_id, stripe_payment_method_id,
	autotopup_enabled, autotopup_threshold_cents, autotopup_amount_cents, autotopup_last_at,
	save_responses_text, save_responses_images`

func scanUser(row interface{ Scan(dest ...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt, &u.Disabled, &u.IsAdmin,
		&u.StripeCustomerID, &u.StripePaymentMethodID,
		&u.AutotopupEnabled, &u.AutotopupThresholdCents, &u.AutotopupAmountCents, &u.AutotopupLastAt,
		&u.SaveResponsesText, &u.SaveResponsesImages,
	)
	return &u, err
}

// SetResponseSaving updates the per-user response-saving opt-in toggles.
func (q *UserQueries) SetResponseSaving(ctx context.Context, userID string, text, images bool) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET save_responses_text = $1, save_responses_images = $2, updated_at = now()
		 WHERE id = $3`,
		text, images, userID,
	)
	return err
}

func (q *UserQueries) Create(ctx context.Context, email, passwordHash, name string) (*model.User, error) {
	u, err := scanUser(q.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, is_admin)
		 VALUES ($1, $2, $3, lower($1) = 'leepenkman@gmail.com')
		 RETURNING `+userCols,
		email, passwordHash, name,
	))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (q *UserQueries) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := scanUser(q.pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, email,
	))
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (q *UserQueries) GetByID(ctx context.Context, id string) (*model.User, error) {
	u, err := scanUser(q.pool.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (q *UserQueries) SetStripeCustomerID(ctx context.Context, userID, customerID string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET stripe_customer_id = $1, updated_at = now() WHERE id = $2`,
		customerID, userID,
	)
	return err
}

func (q *UserQueries) SetStripePaymentMethod(ctx context.Context, userID, paymentMethodID string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET stripe_payment_method_id = $1, updated_at = now() WHERE id = $2`,
		paymentMethodID, userID,
	)
	return err
}

func (q *UserQueries) ClearStripePaymentMethod(ctx context.Context, userID string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET stripe_payment_method_id = NULL, autotopup_enabled = FALSE, updated_at = now() WHERE id = $1`,
		userID,
	)
	return err
}

func (q *UserQueries) UpdateAutotopupSettings(ctx context.Context, userID string, enabled bool, thresholdCents, amountCents int64) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET autotopup_enabled = $1, autotopup_threshold_cents = $2, autotopup_amount_cents = $3, updated_at = now()
		 WHERE id = $4`,
		enabled, thresholdCents, amountCents, userID,
	)
	return err
}

func (q *UserQueries) SetAutotopupLastAt(ctx context.Context, userID string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET autotopup_last_at = now() WHERE id = $1`, userID,
	)
	return err
}

type AutotopupInfo struct {
	Enabled          bool
	ThresholdCents   int64
	AmountCents      int64
	StripeCustomerID *string
	PaymentMethodID  *string
	LastAt           *interface{}
	BalanceCents     int64
}

func (q *UserQueries) GetAutotopupInfo(ctx context.Context, userID string) (*model.User, int64, error) {
	var u model.User
	var balance int64
	err := q.pool.QueryRow(ctx,
		`SELECT u.id, u.updated_at, u.stripe_customer_id, u.stripe_payment_method_id,
		        u.autotopup_enabled, u.autotopup_threshold_cents, u.autotopup_amount_cents, u.autotopup_last_at,
		        COALESCE(cb.balance_cents, 0)
		 FROM users u
		 LEFT JOIN credit_balances cb ON cb.user_id = u.id
		 WHERE u.id = $1`,
		userID,
	).Scan(
		&u.ID, &u.UpdatedAt, &u.StripeCustomerID, &u.StripePaymentMethodID,
		&u.AutotopupEnabled, &u.AutotopupThresholdCents, &u.AutotopupAmountCents, &u.AutotopupLastAt,
		&balance,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get autotopup info: %w", err)
	}
	return &u, balance, nil
}

func (q *UserQueries) ListAdminUserSpend(ctx context.Context) ([]model.AdminUserSpend, error) {
	rows, err := q.pool.Query(ctx, `
		WITH stripe AS (
			SELECT
				user_id,
				COALESCE(SUM(amount_total_cents), 0)::bigint AS gross_cents,
				COALESCE(SUM(refunded_cents), 0)::bigint AS refunded_cents
			FROM stripe_deposits
			GROUP BY user_id
		),
		usage AS (
			SELECT
				user_id,
				COUNT(*)::bigint AS requests,
				COALESCE(SUM(cost_cents), 0)::bigint AS api_spend_cents,
				MAX(created_at) AS last_request_at
			FROM usage_logs
			GROUP BY user_id
		)
		SELECT
			u.id,
			u.email,
			u.name,
			u.created_at,
			usage.last_request_at,
			u.disabled,
			u.is_admin,
			COALESCE(cb.balance_cents, 0)::bigint,
			COALESCE(stripe.gross_cents, 0)::bigint,
			COALESCE(stripe.refunded_cents, 0)::bigint,
			(COALESCE(stripe.gross_cents, 0) - COALESCE(stripe.refunded_cents, 0))::bigint,
			COALESCE(usage.requests, 0)::bigint,
			COALESCE(usage.api_spend_cents, 0)::bigint,
			COALESCE(usage.api_spend_cents, 0)::bigint,
			TRUE
		FROM users u
		LEFT JOIN credit_balances cb ON cb.user_id = u.id
		LEFT JOIN stripe ON stripe.user_id = u.id
		LEFT JOIN usage ON usage.user_id = u.id
		ORDER BY COALESCE(usage.api_spend_cents, 0) DESC, u.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list admin user spend: %w", err)
	}
	defer rows.Close()

	var users []model.AdminUserSpend
	for rows.Next() {
		var u model.AdminUserSpend
		if err := rows.Scan(
			&u.UserID,
			&u.Email,
			&u.Name,
			&u.CreatedAt,
			&u.LastRequestAt,
			&u.Disabled,
			&u.IsAdmin,
			&u.BalanceCents,
			&u.StripeGrossCents,
			&u.StripeRefundedCents,
			&u.StripeNetCents,
			&u.APIRequests,
			&u.APISpendCents,
			&u.ProviderBaseCostCents,
			&u.ProviderEstimated,
		); err != nil {
			return nil, fmt.Errorf("scan admin user spend: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin user spend: %w", err)
	}
	return users, nil
}
