# Crypto Payment Support - Solana + CODEX

## Overview
Add Solana (SOL) and CODEX token payments to OpenPath. Adapted from codex-infinite's approach:
- HD wallet derivation for unique deposit addresses per payment
- Background payment polling via Solana RPC
- Automatic sweeping of funds to main wallet
- No discount on CODEX (market rate only, unlike codex-infinite's 20%)

## Architecture

### Payment Flow
1. User creates checkout intent via `POST /crypto/checkout`
2. Server derives unique deposit address (HD wallet from master seed)
3. User sends SOL or CODEX to deposit address
4. Background poller detects on-chain payment
5. Credits deposited to user's balance
6. Background sweeper moves funds from deposit address to main wallet

### Files to Create/Modify

#### New Files
- `internal/db/migrations/002_crypto.sql` - DB schema
- `internal/model/crypto.go` - Data models
- `internal/crypto/wallet.go` - HD wallet derivation, tx building
- `internal/crypto/rpc.go` - Solana RPC with retry/fallback
- `internal/crypto/price.go` - SOL/CODEX price feeds
- `internal/crypto/poller.go` - Payment detection + sweeper
- `internal/db/queries/crypto.go` - DB queries
- `internal/handler/crypto.go` - HTTP handlers

#### Modified Files
- `internal/config/config.go` - Add CryptoConfig
- `internal/server/server.go` - Wire crypto routes
- `cmd/openpaths/main.go` - Initialize crypto subsystem
- `go.mod` / `go.sum` - Add base58 dep

### Database Schema
```sql
-- Sequence for deposit address derivation
CREATE SEQUENCE crypto_deposit_index_seq;

-- Checkout intents
CREATE TABLE crypto_checkout_intents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    method          TEXT NOT NULL, -- 'sol' or 'codex'
    amount_usd      NUMERIC(12,2) NOT NULL,
    amount_lamports BIGINT NOT NULL,
    amount_ui       TEXT NOT NULL,
    deposit_index   BIGINT NOT NULL,
    deposit_pubkey  TEXT NOT NULL,
    mint            TEXT, -- NULL for SOL, token mint for CODEX
    status          TEXT NOT NULL DEFAULT 'pending', -- pending/paid/expired
    tx_sig          TEXT,
    credits_cents   BIGINT NOT NULL, -- credits to add on payment
    swept           BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at      TIMESTAMPTZ NOT NULL,
    honor_until     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at         TIMESTAMPTZ
);
```

### Config Addition
```yaml
crypto:
  enabled: true
  solana_wallet_pubkey: "${SOLANA_WALLET_PUBKEY}"
  hd_wallet_seed: "${HD_WALLET_SEED}"
  solana_rpc_url: "${SOLANA_RPC_URL}"
  helius_api_key: "${HELIUS_API_KEY}"
  codex_token_mint: "HAK9cX1jfYmcNpr6keTkLvxehGPWKELXSu7GH2ofBAGS"
  bags_api_key: "${BAGS_API_KEY}"
  min_topup_usd: 5
```

### API Endpoints (public chain, JWT optional for user context)
- `POST /crypto/checkout` - Create checkout intent
- `GET /crypto/checkout/{id}` - Get checkout status
- `GET /crypto/checkout/{id}/events` - SSE stream for status
- `GET /crypto/prices` - Current SOL/CODEX prices

### Key Differences from codex-infinite
- No CODEX discount (market rate)
- No subscription plans (credits-only model)
- Integrated with existing billing engine (credits in hundredths-of-a-cent)
- JWT auth for checkout creation (uses existing user system)
- Clean package separation (not monolith)
