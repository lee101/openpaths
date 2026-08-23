package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	stripesvc "github.com/openpaths/openpaths/internal/stripe"
)

type CheckoutHandler struct {
	stripe        *stripesvc.Service
	userQ         *queries.UserQueries
	billing       *billing.Engine
	depositQ      *queries.StripeDepositQueries
	priceID       string
	webhookSecret string
	guardQ        *queries.GuardQueries
}

// SetGuards wires the optional max top-up cap store (unlimited when unset).
func (h *CheckoutHandler) SetGuards(g *queries.GuardQueries) { h.guardQ = g }

func NewCheckoutHandler(stripe *stripesvc.Service, userQ *queries.UserQueries, billing *billing.Engine, depositQ *queries.StripeDepositQueries, priceID, webhookSecret string) *CheckoutHandler {
	return &CheckoutHandler{
		stripe:        stripe,
		userQ:         userQ,
		billing:       billing,
		depositQ:      depositQ,
		priceID:       priceID,
		webhookSecret: webhookSecret,
	}
}

type createCheckoutRequest struct {
	AmountUSD int `json:"amount_usd"`
}

// HandleCreateCheckout creates an embedded Stripe Checkout session.
// POST /account/stripe/checkout
func (h *CheckoutHandler) HandleCreateCheckout(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req createCheckoutRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if req.AmountUSD < 1 || req.AmountUSD > 500 {
		writeError(ctx, 400, "invalid_request", "amount_usd must be 1-500")
		return
	}
	if h.guardQ != nil {
		if ok, cap := h.guardQ.TopupWithinCap(ctx, userID, int64(req.AmountUSD)*100); !ok {
			writeError(ctx, 400, "topup_cap_exceeded",
				fmt.Sprintf("top-up exceeds your max top-up cap of $%.2f", float64(cap)/100))
			return
		}
	}

	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user")
		return
	}

	customerID := ""
	if user.StripeCustomerID != nil && *user.StripeCustomerID != "" {
		customerID = *user.StripeCustomerID
	} else {
		cid, err := h.stripe.CreateCustomer(user.Email, user.Name)
		if err != nil {
			writeError(ctx, 500, "stripe_error", "Failed to create Stripe customer")
			return
		}
		customerID = cid
		if err := h.userQ.SetStripeCustomerID(ctx, userID, customerID); err != nil {
			writeError(ctx, 500, "server_error", "Failed to save customer ID")
			return
		}
	}

	// price is $0.01 per unit, so $25 = 2500 quantity
	quantity := int64(req.AmountUSD) * 100

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://openpaths.io"
	}
	returnURL := appURL + "/account?payment=success&session_id={CHECKOUT_SESSION_ID}"

	metadata := map[string]string{
		"user_id":    userID,
		"amount_usd": fmt.Sprintf("%d", req.AmountUSD),
		"type":       "credits_purchase",
	}

	clientSecret, err := h.stripe.CreateCheckoutSession(customerID, h.priceID, returnURL, quantity, metadata)
	if err != nil {
		log.Printf("stripe checkout err: %v", err)
		writeError(ctx, 500, "stripe_error", "Failed to create checkout session")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"client_secret": clientSecret,
	})
}

// HandleStripeConfig returns the publishable key for frontend init.
// GET /account/stripe/config
func (h *CheckoutHandler) HandleStripeConfig(ctx *fasthttp.RequestCtx) {
	pk := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	writeJSON(ctx, 200, map[string]any{
		"publishable_key": pk,
	})
}

// HandleWebhook processes Stripe webhook events.
// POST /stripe/webhooks
func (h *CheckoutHandler) HandleWebhook(ctx *fasthttp.RequestCtx) {
	payload := ctx.PostBody()
	sig := stripeSignatureHeader(ctx)

	evt, err := h.stripe.ConstructWebhookEvent(payload, sig, h.webhookSecret)
	if err != nil {
		log.Printf("webhook parse err: %v (signature_present=%t payload_bytes=%d)", err, sig != "", len(payload))
		writeError(ctx, 400, "webhook_error", "Invalid webhook payload")
		return
	}

	log.Printf("stripe webhook: %s", evt.Type)

	switch evt.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(ctx, evt.Data.Object)
	case "charge.refunded":
		h.handleChargeRefunded(ctx, evt.ID, evt.Data.Object)
	default:
		log.Printf("unhandled webhook: %s", evt.Type)
	}

	ctx.SetStatusCode(200)
	ctx.SetBodyString(`{"received":true}`)
}

func stripeSignatureHeader(ctx *fasthttp.RequestCtx) string {
	if sig := string(ctx.Request.Header.Peek("Stripe-Signature")); sig != "" {
		return sig
	}

	var sig string
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		if sig == "" && strings.EqualFold(string(key), "Stripe-Signature") {
			sig = string(value)
		}
	})
	return sig
}

func (h *CheckoutHandler) handleCheckoutCompleted(ctx *fasthttp.RequestCtx, raw json.RawMessage) {
	var session stripesvc.CheckoutSession
	if err := json.Unmarshal(raw, &session); err != nil {
		log.Printf("webhook: failed to parse session: %v", err)
		return
	}

	if session.PaymentStatus != "paid" {
		log.Printf("webhook: session %s not paid (status=%s)", session.ID, session.PaymentStatus)
		return
	}
	if session.Status != "" && session.Status != "complete" {
		log.Printf("webhook: session %s not complete (status=%s)", session.ID, session.Status)
		return
	}
	if session.Metadata["type"] != "credits_purchase" {
		log.Printf("webhook: session %s is not a credits purchase", session.ID)
		return
	}

	userID := session.Metadata["user_id"]
	if userID == "" {
		log.Printf("webhook: no user_id in metadata for session %s", session.ID)
		return
	}
	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		log.Printf("webhook: failed to load user %s for session %s: %v", userID, session.ID, err)
		return
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" || session.Customer != *user.StripeCustomerID {
		log.Printf("webhook: session %s customer does not match user %s", session.ID, userID)
		return
	}

	credited, err := h.depositQ.CreditFromStripeSession(
		ctx, userID, session.ID, session.PaymentIntent, session.AmountTotal, "webhook",
	)
	if err != nil {
		log.Printf("webhook: credit failed for user %s session %s: %v", userID, session.ID, err)
		return
	}
	if credited {
		log.Printf("webhook: credited user %s for session %s ($%.2f)",
			userID, session.ID, float64(session.AmountTotal)/100.0)
	} else {
		log.Printf("webhook: session %s already credited, no-op", session.ID)
	}

	h.saveCheckoutPaymentMethod(ctx, userID, session.PaymentIntent)
}

func (h *CheckoutHandler) saveCheckoutPaymentMethod(ctx *fasthttp.RequestCtx, userID, paymentIntentID string) {
	if paymentIntentID == "" {
		return
	}
	pi, err := h.stripe.RetrievePaymentIntent(paymentIntentID)
	if err != nil {
		log.Printf("webhook: payment method lookup failed for pi %s: %v", paymentIntentID, err)
		return
	}
	if pi.PaymentMethod == "" {
		return
	}
	if err := h.userQ.SetStripePaymentMethod(ctx, userID, pi.PaymentMethod); err != nil {
		log.Printf("webhook: failed to save payment method for user %s: %v", userID, err)
	}
}

func (h *CheckoutHandler) handleChargeRefunded(ctx *fasthttp.RequestCtx, eventID string, raw json.RawMessage) {
	var charge stripesvc.Charge
	if err := json.Unmarshal(raw, &charge); err != nil {
		log.Printf("webhook: failed to parse charge: %v", err)
		return
	}
	if charge.PaymentIntent == "" {
		log.Printf("webhook: charge %s has no payment_intent, skipping", charge.ID)
		return
	}
	if charge.AmountRefunded <= 0 {
		return
	}

	deducted, err := h.depositQ.RefundByPaymentIntent(ctx, charge.PaymentIntent, charge.AmountRefunded, eventID)
	if err != nil {
		log.Printf("webhook: refund failed for charge %s (pi %s): %v", charge.ID, charge.PaymentIntent, err)
		return
	}
	if deducted > 0 {
		log.Printf("webhook: clawed back %d internal units for charge %s (pi %s, cumulative refunded=%d cents)",
			deducted, charge.ID, charge.PaymentIntent, charge.AmountRefunded)
	} else {
		log.Printf("webhook: refund for charge %s already applied or no deposit match, no-op", charge.ID)
	}
}
