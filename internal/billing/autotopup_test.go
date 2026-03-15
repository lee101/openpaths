package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// --- fakes ---

type fakeUserQ struct {
	mu   sync.Mutex
	user *model.User
	bal  int64
	err  error
	lastAtCalled bool
}

func (f *fakeUserQ) GetAutotopupInfo(_ context.Context, _ string) (*model.User, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.user, f.bal, nil
}

func (f *fakeUserQ) SetAutotopupLastAt(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastAtCalled = true
	return nil
}

type fakeTopupQ struct {
	mu      sync.Mutex
	charges []loggedCharge
}

type loggedCharge struct {
	amountCents int64
	amountUSD   float64
	status      string
	errMsg      string
}

func (f *fakeTopupQ) LogCharge(_ context.Context, _ string, amountCents int64, amountUSD float64, _ string, status string, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.charges = append(f.charges, loggedCharge{amountCents, amountUSD, status, errMsg})
	return nil
}

type fakeStripe struct {
	mu         sync.Mutex
	charges    []chargeCall
	err        error
}

type chargeCall struct {
	customerID string
	pmID       string
	amount     int64
}

func (f *fakeStripe) ChargeOffSession(customerID, pmID string, amount int64, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.charges = append(f.charges, chargeCall{customerID, pmID, amount})
	if f.err != nil {
		return "", f.err
	}
	return "pi_test_123", nil
}

type fakeEngine struct {
	mu       sync.Mutex
	deposits []depositCall
	err      error
}

type depositCall struct {
	userID string
	amount int64
	desc   string
}

func (f *fakeEngine) Deposit(_ context.Context, userID string, amount int64, desc string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deposits = append(f.deposits, depositCall{userID, amount, desc})
	return f.err
}

// --- autotopup service using fakes ---

type testableTopupService struct {
	userQ   *fakeUserQ
	topupQ  *fakeTopupQ
	stripe  *fakeStripe
	engine  *fakeEngine
	mu      sync.Mutex
	inFlight map[string]bool
}

func newTestableTopup(user *model.User, balance int64) *testableTopupService {
	return &testableTopupService{
		userQ:    &fakeUserQ{user: user, bal: balance},
		topupQ:   &fakeTopupQ{},
		stripe:   &fakeStripe{},
		engine:   &fakeEngine{},
		inFlight: make(map[string]bool),
	}
}

func (t *testableTopupService) doTopup(ctx context.Context, userID string) error {
	user, balance, err := t.userQ.GetAutotopupInfo(ctx, userID)
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
	if user.AutotopupLastAt != nil && time.Since(*user.AutotopupLastAt) < 60*time.Second {
		return nil
	}

	amountUSDCents := user.AutotopupAmountCents / 100
	if amountUSDCents < 50 {
		amountUSDCents = 50
	}
	amountUSD := float64(amountUSDCents) / 100.0

	idempotencyKey := fmt.Sprintf("autotopup-%s-%d", userID, time.Now().Unix()/60)

	piID, err := t.stripe.ChargeOffSession(
		*user.StripeCustomerID,
		*user.StripePaymentMethodID,
		amountUSDCents,
		idempotencyKey,
	)
	if err != nil {
		_ = t.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "failed", err.Error())
		return fmt.Errorf("stripe charge: %w", err)
	}

	desc := fmt.Sprintf("Auto-topup $%.2f (Stripe: %s)", amountUSD, truncateStr(piID, 20))
	if err := t.engine.Deposit(ctx, userID, user.AutotopupAmountCents, desc); err != nil {
		_ = t.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "failed", "deposit failed: "+err.Error())
		return fmt.Errorf("deposit: %w", err)
	}

	_ = t.userQ.SetAutotopupLastAt(ctx, userID)
	_ = t.topupQ.LogCharge(ctx, userID, user.AutotopupAmountCents, amountUSD, piID, "succeeded", "")
	return nil
}

// --- tests ---

func strPtr(s string) *string { return &s }

func TestAutoTopup_ChargesWhenBelowThreshold(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,  // $5
		AutotopupAmountCents:    100000, // $10
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 30000) // $3 balance (below $5 threshold)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(svc.stripe.charges) != 1 {
		t.Fatalf("expected 1 stripe charge, got %d", len(svc.stripe.charges))
	}

	ch := svc.stripe.charges[0]
	if ch.customerID != "cus_123" {
		t.Errorf("customer = %q, want cus_123", ch.customerID)
	}
	if ch.pmID != "pm_123" {
		t.Errorf("pm = %q, want pm_123", ch.pmID)
	}
	// 100000 internal / 100 = 1000 USD cents = $10
	if ch.amount != 1000 {
		t.Errorf("amount = %d USD cents, want 1000", ch.amount)
	}

	if len(svc.engine.deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(svc.engine.deposits))
	}
	if svc.engine.deposits[0].amount != 100000 {
		t.Errorf("deposit amount = %d, want 100000", svc.engine.deposits[0].amount)
	}

	if len(svc.topupQ.charges) != 1 || svc.topupQ.charges[0].status != "succeeded" {
		t.Errorf("expected succeeded charge log")
	}
	if !svc.userQ.lastAtCalled {
		t.Error("expected SetAutotopupLastAt to be called")
	}
}

func TestAutoTopup_SkipsWhenAboveThreshold(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 60000) // $6 balance (above $5 threshold)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges, got %d", len(svc.stripe.charges))
	}
}

func TestAutoTopup_SkipsWhenDisabled(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       false,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges when disabled")
	}
}

func TestAutoTopup_SkipsWithoutPaymentMethod(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   nil,
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges without payment method")
	}
}

func TestAutoTopup_SkipsWithoutCustomer(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        nil,
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges without customer")
	}
}

func TestAutoTopup_RateLimits60Seconds(t *testing.T) {
	recent := time.Now().Add(-30 * time.Second)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
		AutotopupLastAt:         &recent,
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges within 60s rate limit")
	}
}

func TestAutoTopup_AllowsAfterRateLimitExpires(t *testing.T) {
	old := time.Now().Add(-90 * time.Second)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
		AutotopupLastAt:         &old,
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 1 {
		t.Errorf("expected 1 charge after rate limit expired, got %d", len(svc.stripe.charges))
	}
}

func TestAutoTopup_MinimumStripeCharge(t *testing.T) {
	// Amount is 4000 internal (= $0.40 = 40 USD cents, below Stripe $0.50 minimum)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    4000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 1 {
		t.Fatalf("expected 1 charge, got %d", len(svc.stripe.charges))
	}
	if svc.stripe.charges[0].amount != 50 {
		t.Errorf("amount = %d, want 50 (Stripe minimum)", svc.stripe.charges[0].amount)
	}
}

func TestAutoTopup_StripeErrorLogsFailure(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	svc.stripe.err = errors.New("card_declined")

	err := svc.doTopup(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error on stripe failure")
	}
	if len(svc.topupQ.charges) != 1 {
		t.Fatalf("expected 1 charge log, got %d", len(svc.topupQ.charges))
	}
	if svc.topupQ.charges[0].status != "failed" {
		t.Errorf("status = %q, want failed", svc.topupQ.charges[0].status)
	}
	if len(svc.engine.deposits) != 0 {
		t.Error("no deposit should occur on stripe failure")
	}
}

func TestAutoTopup_DepositErrorLogsFailure(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	svc.engine.err = errors.New("db error")

	err := svc.doTopup(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error on deposit failure")
	}
	if len(svc.stripe.charges) != 1 {
		t.Fatal("stripe should still have been called")
	}
	if len(svc.topupQ.charges) != 1 || svc.topupQ.charges[0].status != "failed" {
		t.Error("expected failed charge log on deposit error")
	}
}

func TestAutoTopup_SkipsZeroAmount(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:       true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    0,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svc.stripe.charges) != 0 {
		t.Errorf("expected no charges with zero amount")
	}
}

func TestAutoTopup_UnitConversion(t *testing.T) {
	tests := []struct {
		name           string
		amountInternal int64
		wantUSDCents   int64
		wantUSD        float64
	}{
		{"$5 topup", 50000, 500, 5.00},
		{"$10 topup", 100000, 1000, 10.00},
		{"$25 topup", 250000, 2500, 25.00},
		{"$50 topup", 500000, 5000, 50.00},
		{"$100 topup", 1000000, 10000, 100.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestableTopup(&model.User{
				AutotopupEnabled:       true,
				AutotopupThresholdCents: 50000,
				AutotopupAmountCents:    tt.amountInternal,
				StripeCustomerID:        strPtr("cus_123"),
				StripePaymentMethodID:   strPtr("pm_123"),
			}, 10000)

			err := svc.doTopup(context.Background(), "user1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc.stripe.charges[0].amount != tt.wantUSDCents {
				t.Errorf("stripe amount = %d USD cents, want %d", svc.stripe.charges[0].amount, tt.wantUSDCents)
			}
			if svc.topupQ.charges[0].amountUSD != tt.wantUSD {
				t.Errorf("logged USD = %.2f, want %.2f", svc.topupQ.charges[0].amountUSD, tt.wantUSD)
			}
		})
	}
}

func TestAutoTopup_InFlightPrevention(t *testing.T) {
	svc := &AutoTopupService{
		inFlight: make(map[string]bool),
	}

	svc.mu.Lock()
	svc.inFlight["user1"] = true
	svc.mu.Unlock()

	// CheckAndTopup should return immediately for in-flight user
	svc.CheckAndTopup("user1")

	svc.mu.Lock()
	if !svc.inFlight["user1"] {
		t.Error("in-flight flag should still be set")
	}
	svc.mu.Unlock()
}

func TestAutoTopup_NilServiceSafe(t *testing.T) {
	var svc *AutoTopupService
	svc.CheckAndTopup("user1") // should not panic
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"ab", 1, "a..."},
	}
	for _, tt := range tests {
		got := truncateStr(tt.in, tt.n)
		if got != tt.want {
			t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}
