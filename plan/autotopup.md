# Auto-Topup via Stripe

## Overview
When a user's credit balance falls below a configured threshold after a usage deduction, automatically charge their saved Stripe payment method and deposit credits. Stripe only (no crypto).

## Database Changes (migration 004)
Add columns to `users` table:
- `stripe_customer_id TEXT` - Stripe customer ID
- `stripe_payment_method_id TEXT` - default payment method for off-session charges
- `autotopup_enabled BOOLEAN DEFAULT FALSE`
- `autotopup_threshold_cents BIGINT DEFAULT 50000` (= $5.00 in our hundredths-of-a-cent unit)
- `autotopup_amount_cents BIGINT DEFAULT 100000` (= $10.00)
- `autotopup_last_at TIMESTAMPTZ` - prevent rapid-fire charges

Add `autotopup_charges` table for audit:
- `id UUID PK`
- `user_id UUID FK`
- `amount_cents BIGINT` (our units)
- `amount_usd NUMERIC(12,2)`
- `stripe_payment_intent_id TEXT`
- `status TEXT` (pending/succeeded/failed)
- `error TEXT`
- `created_at TIMESTAMPTZ`

## New Transaction Type
`TxTypeAutoTopup = "auto_topup"` in model/credit.go

## Stripe Service (`internal/stripe/stripe.go`)
Thin wrapper over Stripe API using `github.com/stripe/stripe-go/v82`:
- `CreateCustomer(email, name) -> customerID`
- `CreateSetupIntent(customerID) -> clientSecret` (for frontend to collect card)
- `ListPaymentMethods(customerID) -> []PaymentMethod`
- `ChargeOffSession(customerID, paymentMethodID, amountCents int64, idempotencyKey) -> paymentIntentID, error`

## Auto-Topup Service (`internal/billing/autotopup.go`)
- `AutoTopupService` struct holds: UserQueries, CreditQueries, StripeService, BillingEngine
- `CheckAndTopup(ctx, userID)` - called async (goroutine) after every deduction:
  1. Query user's autotopup settings + current balance in one query
  2. Guard: not enabled, or balance > threshold, or no stripe payment method -> return
  3. Guard: last topup was < 60 seconds ago -> return (prevents rapid-fire)
  4. Charge Stripe off-session
  5. On success: Deposit credits via billing engine, update autotopup_last_at, log to autotopup_charges
  6. On failure: log to autotopup_charges with error

## Integration Points

### Billing Engine
After every `Deduct*()` call, the handler goroutine-calls `autotopup.CheckAndTopup(ctx, userID)`.
Rather than modifying every handler, add a wrapper method on Engine: `DeductAndCheck()` that does deduct + async check.

### Config (`config.yaml`)
```yaml
stripe:
  secret_key: "${STRIPE_SECRET_KEY}"
  webhook_secret: "${STRIPE_WEBHOOK_SECRET}"
```

### API Endpoints (JWT-authed)
- `POST /account/stripe/setup` - creates Stripe customer (if needed) + SetupIntent, returns client_secret
- `POST /account/stripe/confirm` - after frontend confirms, saves payment_method_id to user
- `GET /account/stripe/payment-methods` - list saved cards
- `DELETE /account/stripe/payment-methods/{id}` - detach
- `POST /account/autotopup/settings` - update enabled/threshold/amount
- `GET /account/autotopup/settings` - get current settings

### Validation
- threshold: $1-$100 (10000-1000000 in our units)
- amount: $5-$500 (50000-5000000 in our units)
- must have saved payment method to enable

## Flow
1. User hits `POST /account/stripe/setup` -> gets clientSecret
2. Frontend uses Stripe.js to confirm SetupIntent with card details
3. User hits `POST /account/stripe/confirm` with payment_method_id -> saved
4. User hits `POST /account/autotopup/settings` with `{enabled: true, threshold_cents: 50000, amount_cents: 100000}`
5. During normal API usage, after each deduction, async goroutine checks if balance < threshold
6. If so, charges Stripe and deposits credits automatically
