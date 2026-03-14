package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func signedHeader(payload []byte, secret string, ts time.Time) string {
	signedPayload := fmt.Sprintf("%d.%s", ts.Unix(), payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return fmt.Sprintf("t=%d,v1=%s", ts.Unix(), hex.EncodeToString(mac.Sum(nil)))
}
