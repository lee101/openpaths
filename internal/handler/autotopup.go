package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/stripe"
)

type AutotopupHandler struct {
	userQ  *queries.UserQueries
	stripe *stripe.Service
}

func NewAutotopupHandler(userQ *queries.UserQueries, stripeSvc *stripe.Service) *AutotopupHandler {
	return &AutotopupHandler{userQ: userQ, stripe: stripeSvc}
}

// HandleStripeSetup creates a Stripe customer (if needed) and SetupIntent for saving a card.
func (h *AutotopupHandler) HandleStripeSetup(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

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

	_, clientSecret, err := h.stripe.CreateSetupIntent(customerID)
	if err != nil {
		writeError(ctx, 500, "stripe_error", "Failed to create setup intent")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"client_secret": clientSecret,
		"customer_id":   customerID,
	})
}

type stripeConfirmRequest struct {
	PaymentMethodID string `json:"payment_method_id"`
}

// HandleStripeConfirm saves the payment method ID after frontend confirms the SetupIntent.
func (h *AutotopupHandler) HandleStripeConfirm(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req stripeConfirmRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil || req.PaymentMethodID == "" {
		writeError(ctx, 400, "invalid_request", "payment_method_id required")
		return
	}

	if err := h.userQ.SetStripePaymentMethod(ctx, userID, req.PaymentMethodID); err != nil {
		writeError(ctx, 500, "server_error", "Failed to save payment method")
		return
	}

	writeJSON(ctx, 200, map[string]any{"message": "Payment method saved"})
}

// HandleListPaymentMethods returns the user's saved Stripe payment methods.
func (h *AutotopupHandler) HandleListPaymentMethods(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user")
		return
	}

	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		writeJSON(ctx, 200, map[string]any{"payment_methods": []any{}})
		return
	}

	methods, err := h.stripe.ListPaymentMethods(*user.StripeCustomerID)
	if err != nil {
		writeError(ctx, 500, "stripe_error", "Failed to list payment methods")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"payment_methods":           methods,
		"default_payment_method_id": user.StripePaymentMethodID,
	})
}

// HandleDeletePaymentMethod detaches a payment method.
func (h *AutotopupHandler) HandleDeletePaymentMethod(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	pmID, _ := ctx.UserValue("id").(string)

	if err := h.stripe.DetachPaymentMethod(pmID); err != nil {
		writeError(ctx, 500, "stripe_error", "Failed to detach payment method")
		return
	}

	// If this was the default, clear it
	user, _ := h.userQ.GetByID(ctx, userID)
	if user != nil && user.StripePaymentMethodID != nil && *user.StripePaymentMethodID == pmID {
		_ = h.userQ.ClearStripePaymentMethod(ctx, userID)
	}

	writeJSON(ctx, 200, map[string]any{"message": "Payment method removed"})
}

type autotopupSettingsRequest struct {
	Enabled        bool  `json:"enabled"`
	ThresholdCents int64 `json:"threshold_cents"`
	AmountCents    int64 `json:"amount_cents"`
}

// HandleUpdateAutotopupSettings updates auto-topup preferences.
func (h *AutotopupHandler) HandleUpdateAutotopupSettings(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req autotopupSettingsRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	if req.Enabled {
		// Validate ranges: threshold $1-$100, amount $5-$500 (in hundredths-of-a-cent)
		if req.ThresholdCents < 10000 || req.ThresholdCents > 1000000 {
			writeError(ctx, 400, "invalid_request", "threshold must be $1-$100")
			return
		}
		if req.AmountCents < 50000 || req.AmountCents > 5000000 {
			writeError(ctx, 400, "invalid_request", "amount must be $5-$500")
			return
		}

		user, err := h.userQ.GetByID(ctx, userID)
		if err != nil {
			writeError(ctx, 500, "server_error", "Failed to get user")
			return
		}
		if user.StripePaymentMethodID == nil || *user.StripePaymentMethodID == "" {
			writeError(ctx, 400, "invalid_request", "Add a payment method first")
			return
		}
	}

	if err := h.userQ.UpdateAutotopupSettings(ctx, userID, req.Enabled, req.ThresholdCents, req.AmountCents); err != nil {
		writeError(ctx, 500, "server_error", "Failed to update settings")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"enabled":        req.Enabled,
		"threshold_cents": req.ThresholdCents,
		"amount_cents":   req.AmountCents,
	})
}

// HandleGetAutotopupSettings returns current auto-topup settings.
func (h *AutotopupHandler) HandleGetAutotopupSettings(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"enabled":           user.AutotopupEnabled,
		"threshold_cents":   user.AutotopupThresholdCents,
		"threshold_usd":     float64(user.AutotopupThresholdCents) / 10000.0,
		"amount_cents":      user.AutotopupAmountCents,
		"amount_usd":        float64(user.AutotopupAmountCents) / 10000.0,
		"has_payment_method": user.StripePaymentMethodID != nil && *user.StripePaymentMethodID != "",
	})
}
