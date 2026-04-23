package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpaths/openpaths/internal/model"
)

type StripeDepositQueries struct {
	pool *pgxpool.Pool
}

func NewStripeDepositQueries(pool *pgxpool.Pool) *StripeDepositQueries {
	return &StripeDepositQueries{pool: pool}
}

// CreditFromStripeSession idempotently credits a user for a paid Stripe checkout session.
// Returns credited=true only the first time a given session_id is seen. Subsequent calls
// (webhook replay, reconcile re-run) are no-ops and return credited=false.
//
// amountTotalCents is Stripe's amount_total in cents (e.g. 2500 for $25). We convert to
// the internal hundredths-of-a-cent unit by multiplying by 100 to match the credits pricing.
func (q *StripeDepositQueries) CreditFromStripeSession(
	ctx context.Context,
	userID, sessionID, paymentIntentID string,
	amountTotalCents int64,
	source string,
) (credited bool, err error) {
	if userID == "" || sessionID == "" {
		return false, fmt.Errorf("user_id and session_id required")
	}
	if amountTotalCents <= 0 {
		return false, fmt.Errorf("amount_total_cents must be positive")
	}

	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var piArg any
	if paymentIntentID != "" {
		piArg = paymentIntentID
	}

	var gotSession string
	err = tx.QueryRow(ctx, `
		INSERT INTO stripe_deposits (session_id, user_id, payment_intent_id, amount_total_cents, source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO NOTHING
		RETURNING session_id
	`, sessionID, userID, piArg, amountTotalCents, source).Scan(&gotSession)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, tx.Commit(ctx)
	}
	if err != nil {
		return false, fmt.Errorf("insert stripe_deposit: %w", err)
	}

	internalAmount := amountTotalCents * 100
	desc := fmt.Sprintf("Stripe checkout %s ($%.2f)", sessionID, float64(amountTotalCents)/100.0)

	var newBalance int64
	err = tx.QueryRow(ctx, `
		UPDATE credit_balances SET balance_cents = balance_cents + $1, updated_at = now()
		WHERE user_id = $2 RETURNING balance_cents
	`, internalAmount, userID).Scan(&newBalance)
	if err != nil {
		return false, fmt.Errorf("update balance: %w", err)
	}

	var txID string
	err = tx.QueryRow(ctx, `
		INSERT INTO credit_transactions (user_id, amount_cents, balance_after, tx_type, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, userID, internalAmount, newBalance, model.TxTypeDeposit, desc).Scan(&txID)
	if err != nil {
		return false, fmt.Errorf("insert credit_transaction: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE stripe_deposits SET credit_tx_id = $1 WHERE session_id = $2
	`, txID, sessionID)
	if err != nil {
		return false, fmt.Errorf("link credit_tx_id: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

// RefundByPaymentIntent claws back credits for a refunded Stripe charge.
// cumulativeRefundedCents is Stripe's charge.amount_refunded (always the
// total refunded across all refunds on the charge so far). We deduct only
// the delta against what we've previously clawed back, so repeated events
// for the same cumulative amount are no-ops — and partial refunds stack
// cleanly.
//
// Balance may go negative (user already spent part of their deposit); that's
// correct behaviour for a refund.
//
// Returns the number of internal units (hundredths of a cent) newly deducted.
// Returns 0 if there's no matching stripe_deposits row or no new delta.
func (q *StripeDepositQueries) RefundByPaymentIntent(
	ctx context.Context,
	paymentIntentID string,
	cumulativeRefundedCents int64,
	eventID string,
) (int64, error) {
	if paymentIntentID == "" {
		return 0, fmt.Errorf("payment_intent_id required")
	}
	if cumulativeRefundedCents < 0 {
		return 0, fmt.Errorf("cumulative refund cannot be negative")
	}

	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var sessionID, userID string
	var alreadyRefunded, amountTotal int64
	err = tx.QueryRow(ctx, `
		SELECT session_id, user_id, amount_total_cents, refunded_cents
		FROM stripe_deposits
		WHERE payment_intent_id = $1
		FOR UPDATE
	`, paymentIntentID).Scan(&sessionID, &userID, &amountTotal, &alreadyRefunded)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, tx.Commit(ctx)
	}
	if err != nil {
		return 0, fmt.Errorf("lookup deposit: %w", err)
	}

	// Cap to what was actually deposited — Stripe shouldn't ever refund more
	// than the charge total, but defensively clamp.
	newTotal := cumulativeRefundedCents
	if newTotal > amountTotal {
		newTotal = amountTotal
	}
	deltaCents := newTotal - alreadyRefunded
	if deltaCents <= 0 {
		return 0, tx.Commit(ctx)
	}

	internalAmount := deltaCents * 100

	var newBalance int64
	err = tx.QueryRow(ctx, `
		UPDATE credit_balances SET balance_cents = balance_cents - $1, updated_at = now()
		WHERE user_id = $2 RETURNING balance_cents
	`, internalAmount, userID).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("deduct balance: %w", err)
	}

	desc := fmt.Sprintf("Stripe refund on %s ($%.2f)", sessionID, float64(deltaCents)/100.0)
	if eventID != "" {
		desc = fmt.Sprintf("%s [event %s]", desc, eventID)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credit_transactions (user_id, amount_cents, balance_after, tx_type, description)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, -internalAmount, newBalance, model.TxTypeRefund, desc)
	if err != nil {
		return 0, fmt.Errorf("insert refund tx: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE stripe_deposits SET refunded_cents = $1 WHERE session_id = $2
	`, newTotal, sessionID)
	if err != nil {
		return 0, fmt.Errorf("update refunded_cents: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return internalAmount, nil
}
