package billing

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

// --- fakes ---

type fakeUserQ struct {
	mu           sync.Mutex
	user         *model.User
	bal          int64
	err          error
	lastAtCalled bool
	claims       int
	claimErr     error
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

func (f *fakeUserQ) ClaimAutotopup(_ context.Context, _ string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return false, f.claimErr
	}
	f.claims++
	return f.claims == 1, nil
}

type fakeTopupQ struct {
	mu         sync.Mutex
	charges    []loggedCharge
	lastCharge *queries.AutotopupCharge
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

func (f *fakeTopupQ) LastChargeForUser(_ context.Context, _ string) (*queries.AutotopupCharge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastCharge, nil
}

type fakeStripe struct {
	mu      sync.Mutex
	charges []chargeCall
	err     error
}

type chargeCall struct {
	customerID     string
	pmID           string
	amount         int64
	idempotencyKey string
}

func (f *fakeStripe) ChargeOffSession(customerID, pmID string, amount int64, idempotencyKey string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.charges = append(f.charges, chargeCall{customerID, pmID, amount, idempotencyKey})
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

func newTestableTopup(user *model.User, balance int64) *AutoTopupService {
	return &AutoTopupService{
		userQ:    &fakeUserQ{user: user, bal: balance},
		topupQ:   &fakeTopupQ{},
		stripe:   &fakeStripe{},
		engine:   &fakeEngine{},
		inFlight: make(map[string]bool),
	}
}

func fakeUsers(s *AutoTopupService) *fakeUserQ     { return s.userQ.(*fakeUserQ) }
func fakeTopups(s *AutoTopupService) *fakeTopupQ   { return s.topupQ.(*fakeTopupQ) }
func fakeCharges(s *AutoTopupService) *fakeStripe  { return s.stripe.(*fakeStripe) }
func fakeDeposits(s *AutoTopupService) *fakeEngine { return s.engine.(*fakeEngine) }

// --- tests ---

func strPtr(s string) *string { return &s }

func TestAutoTopup_ChargesWhenBelowThreshold(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,  // $5
		AutotopupAmountCents:    100000, // $10
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 30000) // $3 balance (below $5 threshold)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fakeCharges(svc).charges) != 1 {
		t.Fatalf("expected 1 stripe charge, got %d", len(fakeCharges(svc).charges))
	}

	ch := fakeCharges(svc).charges[0]
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

	if len(fakeDeposits(svc).deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(fakeDeposits(svc).deposits))
	}
	if fakeDeposits(svc).deposits[0].amount != 100000 {
		t.Errorf("deposit amount = %d, want 100000", fakeDeposits(svc).deposits[0].amount)
	}

	if len(fakeTopups(svc).charges) != 1 || fakeTopups(svc).charges[0].status != "succeeded" {
		t.Errorf("expected succeeded charge log")
	}
	if !fakeUsers(svc).lastAtCalled {
		t.Error("expected SetAutotopupLastAt to be called")
	}
}

func TestAutoTopup_SkipsWhenAboveThreshold(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 60000) // $6 balance (above $5 threshold)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
		t.Errorf("expected no charges, got %d", len(fakeCharges(svc).charges))
	}
}

func TestAutoTopup_SkipsWhenDisabled(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        false,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
		t.Errorf("expected no charges when disabled")
	}
}

func TestAutoTopup_SkipsWithoutPaymentMethod(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   nil,
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
		t.Errorf("expected no charges without payment method")
	}
}

func TestAutoTopup_SkipsWithoutCustomer(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        nil,
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
		t.Errorf("expected no charges without customer")
	}
}

func TestAutoTopup_RateLimits60Seconds(t *testing.T) {
	recent := time.Now().Add(-30 * time.Second)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
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
	if len(fakeCharges(svc).charges) != 0 {
		t.Errorf("expected no charges within 60s rate limit")
	}
}

func TestAutoTopup_AllowsAfterRateLimitExpires(t *testing.T) {
	old := time.Now().Add(-90 * time.Second)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
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
	if len(fakeCharges(svc).charges) != 1 {
		t.Errorf("expected 1 charge after rate limit expired, got %d", len(fakeCharges(svc).charges))
	}
}

func TestAutoTopup_MinimumStripeCharge(t *testing.T) {
	// Amount is 4000 internal (= $0.40 = 40 USD cents, below Stripe $0.50 minimum)
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    4000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 1 {
		t.Fatalf("expected 1 charge, got %d", len(fakeCharges(svc).charges))
	}
	if fakeCharges(svc).charges[0].amount != 50 {
		t.Errorf("amount = %d, want 50 (Stripe minimum)", fakeCharges(svc).charges[0].amount)
	}
}

func TestAutoTopup_StripeErrorLogsFailure(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	fakeCharges(svc).err = errors.New("card_declined")

	err := svc.doTopup(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error on stripe failure")
	}
	if len(fakeTopups(svc).charges) != 1 {
		t.Fatalf("expected 1 charge log, got %d", len(fakeTopups(svc).charges))
	}
	if fakeTopups(svc).charges[0].status != "failed" {
		t.Errorf("status = %q, want failed", fakeTopups(svc).charges[0].status)
	}
	if len(fakeDeposits(svc).deposits) != 0 {
		t.Error("no deposit should occur on stripe failure")
	}
}

func TestAutoTopup_DebouncesRecentFailedAttempt(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	fakeTopups(svc).lastCharge = &queries.AutotopupCharge{
		Status:    "failed",
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
		t.Fatalf("expected recent failed attempt to debounce charge, got %d", len(fakeCharges(svc).charges))
	}
}

func TestAutoTopup_RetriesAfterFailedAttemptDebounceExpires(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	fakeTopups(svc).lastCharge = &queries.AutotopupCharge{
		Status:    "failed",
		CreatedAt: time.Now().Add(-7 * time.Hour),
	}

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 1 {
		t.Fatalf("expected retry after debounce expires, got %d charges", len(fakeCharges(svc).charges))
	}
	if fakeCharges(svc).charges[0].idempotencyKey == "" {
		t.Fatal("expected idempotency key")
	}
}

func TestAutoTopup_RetriesAfterUserUpdatesBillingAfterFailure(t *testing.T) {
	failedAt := time.Now().Add(-30 * time.Minute)
	svc := newTestableTopup(&model.User{
		UpdatedAt:               failedAt.Add(5 * time.Minute),
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	fakeTopups(svc).lastCharge = &queries.AutotopupCharge{
		Status:    "failed",
		CreatedAt: failedAt,
	}

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 1 {
		t.Fatalf("expected retry after user billing update, got %d charges", len(fakeCharges(svc).charges))
	}
}

func TestAutoTopup_DepositErrorLogsFailure(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)
	fakeDeposits(svc).err = errors.New("db error")

	err := svc.doTopup(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error on deposit failure")
	}
	if len(fakeCharges(svc).charges) != 1 {
		t.Fatal("stripe should still have been called")
	}
	if len(fakeTopups(svc).charges) != 1 || fakeTopups(svc).charges[0].status != "failed" {
		t.Error("expected failed charge log on deposit error")
	}
}

func TestAutoTopup_SkipsZeroAmount(t *testing.T) {
	svc := newTestableTopup(&model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    0,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}, 10000)

	err := svc.doTopup(context.Background(), "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fakeCharges(svc).charges) != 0 {
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
				AutotopupEnabled:        true,
				AutotopupThresholdCents: 50000,
				AutotopupAmountCents:    tt.amountInternal,
				StripeCustomerID:        strPtr("cus_123"),
				StripePaymentMethodID:   strPtr("pm_123"),
			}, 10000)

			err := svc.doTopup(context.Background(), "user1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fakeCharges(svc).charges[0].amount != tt.wantUSDCents {
				t.Errorf("stripe amount = %d USD cents, want %d", fakeCharges(svc).charges[0].amount, tt.wantUSDCents)
			}
			if fakeTopups(svc).charges[0].amountUSD != tt.wantUSD {
				t.Errorf("logged USD = %.2f, want %.2f", fakeTopups(svc).charges[0].amountUSD, tt.wantUSD)
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

func autotopupTestUser() *model.User {
	return &model.User{
		AutotopupEnabled:        true,
		AutotopupThresholdCents: 50000,
		AutotopupAmountCents:    100000,
		StripeCustomerID:        strPtr("cus_123"),
		StripePaymentMethodID:   strPtr("pm_123"),
	}
}

func TestAutoTopup_ClaimLostMeansNoCharge(t *testing.T) {
	s := newTestableTopup(autotopupTestUser(), 30000)
	fakeUsers(s).claims = 1

	if err := s.doTopup(context.Background(), "user1"); err != nil {
		t.Fatalf("doTopup: %v", err)
	}
	if got := len(fakeCharges(s).charges); got != 0 {
		t.Fatalf("charges = %d, want 0 when another instance holds the claim", got)
	}
	if len(fakeDeposits(s).deposits) != 0 {
		t.Fatal("credits must not be deposited without a charge")
	}
}

func TestAutoTopup_ConcurrentCallsChargeOnce(t *testing.T) {
	s := newTestableTopup(autotopupTestUser(), 30000)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.doTopup(context.Background(), "user1")
		}()
	}
	wg.Wait()

	if got := len(fakeCharges(s).charges); got != 1 {
		t.Fatalf("charges = %d, want exactly 1", got)
	}
}
