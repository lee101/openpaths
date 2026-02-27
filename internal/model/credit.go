package model

import "time"

type CreditBalance struct {
	UserID       string    `json:"user_id"`
	BalanceCents int64     `json:"balance_cents"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	AmountCents  int64     `json:"amount_cents"`
	BalanceAfter int64     `json:"balance_after"`
	TxType       string    `json:"tx_type"`
	Description  string    `json:"description"`
	ReferenceID  *string   `json:"reference_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	TxTypeDeposit        = "deposit"
	TxTypeUsageDeduction = "usage_deduction"
	TxTypeRefund         = "refund"
	TxTypeAdjustment     = "adjustment"
)
