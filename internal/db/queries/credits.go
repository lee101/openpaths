package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type CreditQueries struct {
	pool *pgxpool.Pool
}

func NewCreditQueries(pool *pgxpool.Pool) *CreditQueries {
	return &CreditQueries{pool: pool}
}

func (q *CreditQueries) InitBalance(ctx context.Context, userID string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO credit_balances (user_id, balance_cents) VALUES ($1, 0)
		 ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	return err
}

func (q *CreditQueries) GetBalance(ctx context.Context, userID string) (int64, error) {
	var balance int64
	err := q.pool.QueryRow(ctx,
		"SELECT balance_cents FROM credit_balances WHERE user_id = $1",
		userID,
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

// Deposit adds credits and records a transaction atomically.
func (q *CreditQueries) Deposit(ctx context.Context, userID string, amountCents int64, description string) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var newBalance int64
	err = tx.QueryRow(ctx,
		`UPDATE credit_balances SET balance_cents = balance_cents + $1, updated_at = now()
		 WHERE user_id = $2 RETURNING balance_cents`,
		amountCents, userID,
	).Scan(&newBalance)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO credit_transactions (user_id, amount_cents, balance_after, tx_type, description)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, amountCents, newBalance, model.TxTypeDeposit, description,
	)
	if err != nil {
		return fmt.Errorf("record transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// DeductWithTransaction atomically deducts credits using SELECT FOR UPDATE.
func (q *CreditQueries) DeductWithTransaction(ctx context.Context, userID string, amountCents int64, txType, description string, referenceID *string) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the row and check balance
	var currentBalance int64
	err = tx.QueryRow(ctx,
		"SELECT balance_cents FROM credit_balances WHERE user_id = $1 FOR UPDATE",
		userID,
	).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("lock balance: %w", err)
	}

	if currentBalance < amountCents {
		return fmt.Errorf("insufficient balance: have %d, need %d", currentBalance, amountCents)
	}

	newBalance := currentBalance - amountCents
	_, err = tx.Exec(ctx,
		"UPDATE credit_balances SET balance_cents = $1, updated_at = now() WHERE user_id = $2",
		newBalance, userID,
	)
	if err != nil {
		return fmt.Errorf("deduct balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO credit_transactions (user_id, amount_cents, balance_after, tx_type, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, -amountCents, newBalance, txType, description, referenceID,
	)
	if err != nil {
		return fmt.Errorf("record deduction: %w", err)
	}

	return tx.Commit(ctx)
}

func (q *CreditQueries) GetTransactions(ctx context.Context, userID string, limit, offset int) ([]model.CreditTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx,
		`SELECT id, user_id, amount_cents, balance_after, tx_type, description, reference_id, created_at
		 FROM credit_transactions WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}
	defer rows.Close()

	var txns []model.CreditTransaction
	for rows.Next() {
		var t model.CreditTransaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.AmountCents, &t.BalanceAfter,
			&t.TxType, &t.Description, &t.ReferenceID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, nil
}
