package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type CryptoQueries struct {
	pool *pgxpool.Pool
}

func NewCryptoQueries(pool *pgxpool.Pool) *CryptoQueries {
	return &CryptoQueries{pool: pool}
}

func (q *CryptoQueries) NextDepositIndex(ctx context.Context) (int64, error) {
	var idx int64
	err := q.pool.QueryRow(ctx, "SELECT nextval('crypto_deposit_index_seq')").Scan(&idx)
	return idx, err
}

func (q *CryptoQueries) CreateCheckoutIntent(ctx context.Context, intent *model.CryptoCheckoutIntent) error {
	return q.pool.QueryRow(ctx,
		`INSERT INTO crypto_checkout_intents
		 (user_id, method, amount_usd, amount_lamports, amount_ui, deposit_index, deposit_pubkey, mint, status, credits_cents, expires_at, honor_until)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at`,
		intent.UserID, intent.Method, intent.AmountUSD, int64(intent.AmountLamports),
		intent.AmountUI, intent.DepositIndex, intent.DepositPubkey, nilIfEmpty(intent.Mint),
		intent.Status, intent.CreditsCents, intent.ExpiresAt, intent.HonorUntil,
	).Scan(&intent.ID, &intent.CreatedAt)
}

func (q *CryptoQueries) GetCheckoutIntent(ctx context.Context, id string) (*model.CryptoCheckoutIntent, error) {
	var i model.CryptoCheckoutIntent
	var mint *string
	var txSig *string
	var lamports int64
	err := q.pool.QueryRow(ctx,
		`SELECT id, user_id, method, amount_usd, amount_lamports, amount_ui, deposit_index, deposit_pubkey,
		 mint, status, tx_sig, credits_cents, swept, expires_at, honor_until, created_at, paid_at
		 FROM crypto_checkout_intents WHERE id = $1`, id,
	).Scan(&i.ID, &i.UserID, &i.Method, &i.AmountUSD, &lamports, &i.AmountUI,
		&i.DepositIndex, &i.DepositPubkey, &mint, &i.Status, &txSig,
		&i.CreditsCents, &i.Swept, &i.ExpiresAt, &i.HonorUntil, &i.CreatedAt, &i.PaidAt)
	if err != nil {
		return nil, fmt.Errorf("get checkout: %w", err)
	}
	i.AmountLamports = uint64(lamports)
	if mint != nil {
		i.Mint = *mint
	}
	if txSig != nil {
		i.TxSig = *txSig
	}
	return &i, nil
}

func (q *CryptoQueries) ListPendingCheckouts(ctx context.Context) ([]*model.CryptoCheckoutIntent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, user_id, method, amount_usd, amount_lamports, amount_ui, deposit_index, deposit_pubkey,
		 mint, status, credits_cents, expires_at, honor_until, created_at
		 FROM crypto_checkout_intents WHERE status = 'pending' AND honor_until > now()
		 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckouts(rows)
}

func (q *CryptoQueries) ListUnsweptCheckouts(ctx context.Context) ([]*model.CryptoCheckoutIntent, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, user_id, method, amount_usd, amount_lamports, amount_ui, deposit_index, deposit_pubkey,
		 mint, status, credits_cents, expires_at, honor_until, created_at
		 FROM crypto_checkout_intents WHERE status = 'paid' AND NOT swept
		 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCheckouts(rows)
}

func (q *CryptoQueries) UpdateCheckoutStatus(ctx context.Context, id string, status model.CryptoCheckoutStatus, txSig string) error {
	var paidAt interface{}
	if status == model.CryptoStatusPaid {
		paidAt = time.Now()
	}
	_, err := q.pool.Exec(ctx,
		`UPDATE crypto_checkout_intents SET status = $1, tx_sig = $2, paid_at = $3 WHERE id = $4`,
		status, nilIfEmpty(txSig), paidAt, id)
	return err
}

func (q *CryptoQueries) MarkCheckoutSwept(ctx context.Context, id string) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE crypto_checkout_intents SET swept = TRUE WHERE id = $1`, id)
	return err
}

func (q *CryptoQueries) MarkExpiredCheckouts(ctx context.Context) (int64, error) {
	tag, err := q.pool.Exec(ctx,
		`UPDATE crypto_checkout_intents SET status = 'expired'
		 WHERE status = 'pending' AND honor_until < now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type checkoutRows interface {
	Next() bool
	Scan(dest ...interface{}) error
}

func scanCheckouts(rows checkoutRows) ([]*model.CryptoCheckoutIntent, error) {
	var intents []*model.CryptoCheckoutIntent
	for rows.Next() {
		var i model.CryptoCheckoutIntent
		var mint *string
		var lamports int64
		if err := rows.Scan(&i.ID, &i.UserID, &i.Method, &i.AmountUSD, &lamports, &i.AmountUI,
			&i.DepositIndex, &i.DepositPubkey, &mint, &i.Status, &i.CreditsCents,
			&i.ExpiresAt, &i.HonorUntil, &i.CreatedAt); err != nil {
			return nil, err
		}
		i.AmountLamports = uint64(lamports)
		if mint != nil {
			i.Mint = *mint
		}
		intents = append(intents, &i)
	}
	return intents, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
