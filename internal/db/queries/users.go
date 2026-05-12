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

const userCols = `id, email, password_hash, name, created_at, updated_at, disabled,
	stripe_customer_id, stripe_payment_method_id,
	autotopup_enabled, autotopup_threshold_cents, autotopup_amount_cents, autotopup_last_at`

func scanUser(row interface{ Scan(dest ...any) error }) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt, &u.Disabled,
		&u.StripeCustomerID, &u.StripePaymentMethodID,
		&u.AutotopupEnabled, &u.AutotopupThresholdCents, &u.AutotopupAmountCents, &u.AutotopupLastAt,
	)
	return &u, err
}

func (q *UserQueries) Create(ctx context.Context, email, passwordHash, name string) (*model.User, error) {
	u, err := scanUser(q.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
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
