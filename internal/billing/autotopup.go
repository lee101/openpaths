package billing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/stripe"
)

type AutoTopupService struct {
	userQ      *queries.UserQueries
	creditQ    *queries.CreditQueries
	topupQ     *queries.AutotopupQueries
	stripe     *stripe.Service
	engine     *Engine
	mu         sync.Mutex
	inFlight   map[string]bool // prevent concurrent topups per user
}

func NewAutoTopupService(
	userQ *queries.UserQueries,
	creditQ *queries.CreditQueries,
	topupQ *queries.AutotopupQueries,
	stripeSvc *stripe.Service,
	engine *Engine,
) *AutoTopupService {
	return &AutoTopupService{
		userQ:    userQ,
		creditQ:  creditQ,
		topupQ:   topupQ,
		stripe:   stripeSvc,
		engine:   engine,
		inFlight: make(map[string]bool),
	}
}

func (s *AutoTopupService) CheckAndTopup(userID string) {
	if s == nil || s.stripe == nil {
		return
	}

	// Prevent concurrent topups for same user
	s.mu.Lock()
	if s.inFlight[userID] {
		s.mu.Unlock()
		return
	}
	s.inFlight[userID] = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.inFlight, userID)
			s.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.doTopup(ctx, userID); err != nil {
			log.Printf("autotopup: user=%s err=%v", userID, err)
		}
	}()
}

func (s *AutoTopupService) doTopup(ctx context.Context, userID string) error {
	user, balance, err := s.userQ.GetAutotopupInfo(ctx, userID)
	if err != nil {
		return fmt.Errorf("get info: %w", err)
	}

	if !user.AutotopupEnabled {
		return nil
	}
	if balance > user.AutotopupThresholdCents {
		return nil
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		return nil
	}
	if user.StripePaymentMethodID == nil || *user.StripePaymentMethodID == "" {
		return nil
	}
	if user.AutotopupAmountCents <= 0 {
		return nil
	}

	// Rate limit: no more than once per 60 seconds
	if user.AutotopupLastAt != nil && time.Since(*user.AutotopupLastAt) < 60*time.Second {
		return nil
	}

	// Convert our internal units (hundredths-of-a-cent) to USD cents for Stripe
	// e.g. 100000 internal = $10.00 = 1000 USD cents
	amountUSDCents := user.AutotopupAmountCents / 100
	if amountUSDCents < 50 { // Stripe minimum $0.50
		amountUSDCents = 50
	}
	amountUSD := float64(amountUSDCents) / 100.0

	idempotencyKey := fmt.Sprintf("autotopup-%s-%d", userID, time.Now().Unix()/60)

	piID, err := s.stripe.ChargeOffSession(
		*user.StripeCustomerID,
		*user.StripePaymentMethodID,
		amountUSDCents,
		idempotencyKey,
	)

	if err != nil {
		_ = s.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "failed", err.Error())
		return fmt.Errorf("stripe charge: %w", err)
	}

	// Deposit credits
	desc := fmt.Sprintf("Auto-topup $%.2f (Stripe: %s)", amountUSD, truncateStr(piID, 20))
	if err := s.engine.Deposit(ctx, userID, user.AutotopupAmountCents, desc); err != nil {
		_ = s.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "failed", "deposit failed: "+err.Error())
		return fmt.Errorf("deposit: %w", err)
	}

	_ = s.userQ.SetAutotopupLastAt(ctx, userID)
	_ = s.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "succeeded", "")

	log.Printf("autotopup: user=%s amount=$%.2f pi=%s", userID, amountUSD, piID)
	return nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
