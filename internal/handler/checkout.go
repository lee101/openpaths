package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	priceID       string
	webhookSecret string
}

func NewCheckoutHandler(stripe *stripesvc.Service, userQ *queries.UserQueries, billing *billing.Engine, priceID, webhookSecret string) *CheckoutHandler {
	return &CheckoutHandler{
		stripe:        stripe,
		userQ:         userQ,
		billing:       billing,
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
	sig := string(ctx.Request.Header.Peek("Stripe-Signature"))

	evt, err := h.stripe.ConstructWebhookEvent(payload, sig, h.webhookSecret)
	if err != nil {
		log.Printf("webhook parse err: %v", err)
		writeError(ctx, 400, "webhook_error", "Invalid webhook payload")
		return
	}

	log.Printf("stripe webhook: %s", evt.Type)

	switch evt.Type {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(ctx, evt.Data.Object)
	default:
		log.Printf("unhandled webhook: %s", evt.Type)
	}

	ctx.SetStatusCode(200)
	ctx.SetBodyString(`{"received":true}`)
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

	userID := session.Metadata["user_id"]
	if userID == "" {
		log.Printf("webhook: no user_id in metadata for session %s", session.ID)
		return
	}

	// amount_total is in cents, convert to internal units (hundredths of a cent)
	// $25.00 -> 2500 cents -> 25000000 internal units
	internalAmount := session.AmountTotal * 100

	desc := fmt.Sprintf("Stripe checkout %s ($%.2f)", session.ID, float64(session.AmountTotal)/100.0)
	if err := h.billing.Deposit(ctx, userID, internalAmount, desc); err != nil {
		log.Printf("webhook: deposit failed for user %s: %v", userID, err)
		return
	}

	log.Printf("webhook: deposited %d units for user %s (session %s)", internalAmount, userID, session.ID)
}
