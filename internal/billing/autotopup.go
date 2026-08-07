package billing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/stripe"
)

const (
	autoTopupSuccessDebounce = 60 * time.Second
	autoTopupFailureDebounce = 6 * time.Hour
)

type autoTopupUserStore interface {
	GetAutotopupInfo(ctx context.Context, userID string) (*model.User, int64, error)
	SetAutotopupLastAt(ctx context.Context, userID string) error
	ClaimAutotopup(ctx context.Context, userID string, debounce time.Duration) (bool, error)
}

type autoTopupChargeStore interface {
	LastChargeForUser(ctx context.Context, userID string) (*queries.AutotopupCharge, error)
	LogCharge(ctx context.Context, userID string, amountCents int64, amountUSD float64, stripePI, status, errMsg string) error
}

type autoTopupCharger interface {
	ChargeOffSession(customerID, paymentMethodID string, amountUSDCents int64, idempotencyKey string) (string, error)
}

type autoTopupDepositor interface {
	Deposit(ctx context.Context, userID string, amountCents int64, description string) error
}

type AutoTopupService struct {
	userQ    autoTopupUserStore
	creditQ  *queries.CreditQueries
	topupQ   autoTopupChargeStore
	stripe   autoTopupCharger
	engine   autoTopupDepositor
	guards   *queries.GuardQueries
	mu       sync.Mutex
	inFlight map[string]bool // prevent concurrent topups per user
}

// SetGuards wires the optional max top-up cap store (unlimited when unset).
func (s *AutoTopupService) SetGuards(g *queries.GuardQueries) { s.guards = g }

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

	// Short success debounce handles rapid balance checks after a successful charge.
	if user.AutotopupLastAt != nil && time.Since(*user.AutotopupLastAt) < autoTopupSuccessDebounce {
		return nil
	}
	if skip, err := s.recentFailureDebounced(ctx, userID, user.UpdatedAt); err != nil {
		return err
	} else if skip {
		return nil
	}

	// Convert our internal units (hundredths-of-a-cent) to USD cents for Stripe
	// e.g. 100000 internal = $10.00 = 1000 USD cents
	amountUSDCents := user.AutotopupAmountCents / 100
	if amountUSDCents < 50 { // Stripe minimum $0.50
		amountUSDCents = 50
	}
	// Billshock guard: never auto-charge above the user's max top-up cap.
	if s.guards != nil {
		if ok, cap := s.guards.TopupWithinCap(ctx, userID, amountUSDCents); !ok {
			log.Printf("autotopup skipped for %s: amount %d exceeds cap %d", userID, amountUSDCents, cap)
			return nil
		}
	}
	amountUSD := float64(amountUSDCents) / 100.0

	// Claim the attempt before touching Stripe. The in-flight map only guards
	// this process; the conditional update guards every instance at once, so a
	// balance check racing across replicas cannot charge twice.
	claimed, err := s.userQ.ClaimAutotopup(ctx, userID, autoTopupSuccessDebounce)
	if err != nil {
		return fmt.Errorf("claim: %w", err)
	}
	if !claimed {
		return nil
	}

	idempotencyKey := fmt.Sprintf("autotopup-%s-%d", userID, time.Now().Unix()/int64(autoTopupFailureDebounce.Seconds()))

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

func (s *AutoTopupService) recentFailureDebounced(ctx context.Context, userID string, userUpdatedAt time.Time) (bool, error) {
	if s.topupQ == nil {
		return false, nil
	}
	last, err := s.topupQ.LastChargeForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	if last == nil || last.Status != "failed" {
		return false, nil
	}
	if userUpdatedAt.After(last.CreatedAt) {
		return false, nil
	}
	return time.Since(last.CreatedAt) < autoTopupFailureDebounce, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
