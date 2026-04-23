package billing

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	stripesvc "github.com/openpaths/openpaths/internal/stripe"
)

// Reconciler cross-checks Stripe-side checkout sessions against our
// stripe_deposits table and credits any paid session we haven't credited yet.
// It's the safety net behind the webhook: if the webhook is misconfigured,
// delayed, or drops an event, the next balance fetch still surfaces the credit.
type Reconciler struct {
	stripe   *stripesvc.Service
	userQ    *queries.UserQueries
	depositQ *queries.StripeDepositQueries
	minGap   time.Duration

	mu      sync.Mutex
	lastRun map[string]time.Time
}

func NewReconciler(
	stripeSvc *stripesvc.Service,
	userQ *queries.UserQueries,
	depositQ *queries.StripeDepositQueries,
	minGap time.Duration,
) *Reconciler {
	if minGap <= 0 {
		minGap = 30 * time.Second
	}
	return &Reconciler{
		stripe:   stripeSvc,
		userQ:    userQ,
		depositQ: depositQ,
		minGap:   minGap,
		lastRun:  make(map[string]time.Time),
	}
}

// ReconcileUser lists the user's recent Stripe checkout sessions and credits
// any paid ones not already recorded in stripe_deposits. Calls within minGap
// of a previous run for the same user are skipped to keep balance fetches fast.
// Returns the number of newly-credited sessions.
func (r *Reconciler) ReconcileUser(ctx context.Context, userID string) (int, error) {
	if r == nil || r.stripe == nil {
		return 0, nil
	}
	if !r.shouldRun(userID) {
		return 0, nil
	}

	user, err := r.userQ.GetByID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		r.markRan(userID)
		return 0, nil
	}

	sessions, err := r.stripe.ListCheckoutSessions(*user.StripeCustomerID, 20)
	if err != nil {
		return 0, err
	}

	credited := 0
	for _, sess := range sessions {
		if sess.PaymentStatus != "paid" {
			continue
		}
		if sess.AmountTotal <= 0 {
			continue
		}
		// Defense in depth: only credit sessions whose metadata claims this user.
		if metaUser := sess.Metadata["user_id"]; metaUser != "" && metaUser != userID {
			continue
		}
		ok, err := r.depositQ.CreditFromStripeSession(
			ctx, userID, sess.ID, sess.PaymentIntent, sess.AmountTotal, "reconcile",
		)
		if err != nil {
			log.Printf("reconcile: credit session %s for user %s: %v", sess.ID, userID, err)
			continue
		}
		if ok {
			credited++
			log.Printf("reconcile: credited user %s for session %s ($%.2f)",
				userID, sess.ID, float64(sess.AmountTotal)/100.0)
		}
	}

	r.markRan(userID)
	return credited, nil
}

func (r *Reconciler) shouldRun(userID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	last, ok := r.lastRun[userID]
	if !ok {
		return true
	}
	return time.Since(last) >= r.minGap
}

func (r *Reconciler) markRan(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRun[userID] = time.Now()
}
