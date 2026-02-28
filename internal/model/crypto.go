package model

import "time"

type CryptoMethod string

const (
	CryptoMethodSOL   CryptoMethod = "sol"
	CryptoMethodCODEX CryptoMethod = "codex"
)

type CryptoCheckoutStatus string

const (
	CryptoStatusPending CryptoCheckoutStatus = "pending"
	CryptoStatusPaid    CryptoCheckoutStatus = "paid"
	CryptoStatusExpired CryptoCheckoutStatus = "expired"
)

type CryptoCheckoutIntent struct {
	ID             string               `json:"id"`
	UserID         string               `json:"user_id"`
	Method         CryptoMethod         `json:"method"`
	AmountUSD      float64              `json:"amount_usd"`
	AmountLamports uint64               `json:"amount_lamports"`
	AmountUI       string               `json:"amount_ui"`
	DepositIndex   int64                `json:"deposit_index"`
	DepositPubkey  string               `json:"deposit_pubkey"`
	Mint           string               `json:"mint,omitempty"`
	Status         CryptoCheckoutStatus `json:"status"`
	TxSig          string               `json:"tx_sig,omitempty"`
	CreditsCents   int64                `json:"credits_cents"`
	Swept          bool                 `json:"swept"`
	ExpiresAt      time.Time            `json:"expires_at"`
	HonorUntil     time.Time            `json:"honor_until"`
	CreatedAt      time.Time            `json:"created_at"`
	PaidAt         *time.Time           `json:"paid_at,omitempty"`
}

type CryptoCheckoutRequest struct {
	Method    CryptoMethod `json:"method"`
	AmountUSD float64      `json:"amount_usd"`
}

type CryptoCheckoutResponse struct {
	IntentID       string  `json:"intent_id"`
	DepositAddress string  `json:"deposit_address"`
	SolanaPayURL   string  `json:"solana_pay_url"`
	AmountUI       string  `json:"amount_ui"`
	AmountUSD      float64 `json:"amount_usd"`
	Method         string  `json:"method"`
	TokenMint      string  `json:"token_mint,omitempty"`
	ExpiresAt      string  `json:"expires_at"`
	CreditsCents   int64   `json:"credits_cents"`
}

type CryptoPricesResponse struct {
	SOL   CryptoPriceInfo `json:"sol"`
	CODEX CryptoPriceInfo `json:"codex"`
}

type CryptoPriceInfo struct {
	PriceUSD float64 `json:"price_usd"`
	Mint     string  `json:"mint,omitempty"`
}
