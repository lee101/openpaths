package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConstructWebhookEvent_VerifiesSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_123"}}}`)
	secret := "whsec_test"
	header := signedHeader(payload, secret, time.Now())

	svc := NewService("sk_test")
	evt, err := svc.ConstructWebhookEvent(payload, header, secret)
	if err != nil {
		t.Fatalf("ConstructWebhookEvent() error = %v", err)
	}

	if evt.ID != "evt_123" {
		t.Fatalf("event ID = %q, want evt_123", evt.ID)
	}
	if evt.Type != "checkout.session.completed" {
		t.Fatalf("event type = %q", evt.Type)
	}

	var obj map[string]any
	if err := json.Unmarshal(evt.Data.Object, &obj); err != nil {
		t.Fatalf("unmarshal object: %v", err)
	}
	if obj["id"] != "cs_123" {
		t.Fatalf("object id = %v, want cs_123", obj["id"])
	}
}

func TestConstructWebhookEvent_RejectsBadSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_123"}}}`)
	svc := NewService("sk_test")

	_, err := svc.ConstructWebhookEvent(payload, signedHeader(payload, "whsec_other", time.Now()), "whsec_test")
	if err == nil {
		t.Fatal("expected signature verification error, got nil")
	}
}

func TestConstructWebhookEvent_RejectsOldTimestamp(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_123"}}}`)
	secret := "whsec_test"
	svc := NewService("sk_test")

	_, err := svc.ConstructWebhookEvent(payload, signedHeader(payload, secret, time.Now().Add(-10*time.Minute)), secret)
	if err == nil {
		t.Fatal("expected old timestamp error, got nil")
	}
}

func TestConstructWebhookEvent_RequiresSecret(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_123"}}}`)
	svc := NewService("sk_test")

	_, err := svc.ConstructWebhookEvent(payload, signedHeader(payload, "whsec_test", time.Now()), "")
	if err == nil {
		t.Fatal("expected missing secret error, got nil")
	}
}

func TestCreateCheckoutSession_SavesPaymentMethodForOffSessionUse(t *testing.T) {
	var gotSetupFutureUsage string
	var gotPaymentMethodType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" {
			t.Fatalf("path = %q, want /v1/checkout/sessions", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotSetupFutureUsage = r.Form.Get("payment_intent_data[setup_future_usage]")
		gotPaymentMethodType = r.Form.Get("payment_method_types[]")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_test","client_secret":"cs_secret_test"}`))
	}))
	defer srv.Close()

	svc := NewService("sk_test")
	svc.apiBaseURL = srv.URL

	secret, err := svc.CreateCheckoutSession("cus_test", "price_test", "https://example.test/return", 2500, map[string]string{"user_id": "user1"})
	if err != nil {
		t.Fatalf("CreateCheckoutSession() error = %v", err)
	}
	if secret != "cs_secret_test" {
		t.Fatalf("client secret = %q, want cs_secret_test", secret)
	}
	if gotSetupFutureUsage != "off_session" {
		t.Fatalf("setup_future_usage = %q, want off_session", gotSetupFutureUsage)
	}
	if gotPaymentMethodType != "card" {
		t.Fatalf("payment_method_types[] = %q, want card", gotPaymentMethodType)
	}
}

func TestRetrievePaymentIntent_ReturnsPaymentMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_intents/pi_test" {
			t.Fatalf("path = %q, want /v1/payment_intents/pi_test", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pi_test","payment_method":"pm_test"}`))
	}))
	defer srv.Close()

	svc := NewService("sk_test")
	svc.apiBaseURL = srv.URL

	pi, err := svc.RetrievePaymentIntent("pi_test")
	if err != nil {
		t.Fatalf("RetrievePaymentIntent() error = %v", err)
	}
	if pi.PaymentMethod != "pm_test" {
		t.Fatalf("payment method = %q, want pm_test", pi.PaymentMethod)
	}
}

func signedHeader(payload []byte, secret string, ts time.Time) string {
	signedPayload := fmt.Sprintf("%d.%s", ts.Unix(), payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}
