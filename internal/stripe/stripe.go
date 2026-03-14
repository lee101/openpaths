package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	secretKey string
	client    *http.Client
}

func NewService(secretKey string) *Service {
	return &Service{
		secretKey: secretKey,
		client:    &http.Client{},
	}
}

func (s *Service) CreateCustomer(email, name string) (string, error) {
	vals := url.Values{
		"email": {email},
		"name":  {name},
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := s.post("/v1/customers", vals, &resp); err != nil {
		return "", fmt.Errorf("create customer: %w", err)
	}
	return resp.ID, nil
}

func (s *Service) CreateSetupIntent(customerID string) (string, string, error) {
	vals := url.Values{
		"customer":             {customerID},
		"payment_method_types[]": {"card"},
		"usage":                {"off_session"},
	}
	var resp struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := s.post("/v1/setup_intents", vals, &resp); err != nil {
		return "", "", fmt.Errorf("create setup intent: %w", err)
	}
	return resp.ID, resp.ClientSecret, nil
}

type PaymentMethod struct {
	ID   string `json:"id"`
	Card *struct {
		Brand    string `json:"brand"`
		Last4    string `json:"last4"`
		ExpMonth int    `json:"exp_month"`
		ExpYear  int    `json:"exp_year"`
	} `json:"card"`
}

func (s *Service) ListPaymentMethods(customerID string) ([]PaymentMethod, error) {
	var resp struct {
		Data []PaymentMethod `json:"data"`
	}
	if err := s.get(fmt.Sprintf("/v1/payment_methods?customer=%s&type=card", customerID), &resp); err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}
	return resp.Data, nil
}

func (s *Service) DetachPaymentMethod(paymentMethodID string) error {
	var resp struct{}
	return s.post(fmt.Sprintf("/v1/payment_methods/%s/detach", paymentMethodID), nil, &resp)
}

// CreateCheckoutSession creates an embedded Stripe Checkout session.
// quantity is how many units of the price to buy (e.g. 2500 for $25 at $0.01/unit).
func (s *Service) CreateCheckoutSession(customerID, priceID, returnURL string, quantity int64, metadata map[string]string) (string, error) {
	vals := url.Values{
		"customer":   {customerID},
		"mode":       {"payment"},
		"ui_mode":    {"embedded"},
		"return_url": {returnURL},
		"line_items[0][price]":    {priceID},
		"line_items[0][quantity]": {fmt.Sprintf("%d", quantity)},
	}
	i := 0
	for k, v := range metadata {
		vals.Set(fmt.Sprintf("metadata[%s]", k), v)
		i++
	}
	var resp struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := s.post("/v1/checkout/sessions", vals, &resp); err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return resp.ClientSecret, nil
}

// RetrieveCheckoutSession gets a checkout session by ID.
func (s *Service) RetrieveCheckoutSession(sessionID string) (*CheckoutSession, error) {
	var resp CheckoutSession
	if err := s.get("/v1/checkout/sessions/"+sessionID, &resp); err != nil {
		return nil, fmt.Errorf("retrieve checkout session: %w", err)
	}
	return &resp, nil
}

type CheckoutSession struct {
	ID            string            `json:"id"`
	Customer      string            `json:"customer"`
	PaymentStatus string            `json:"payment_status"`
	AmountTotal   int64             `json:"amount_total"`
	Metadata      map[string]string `json:"metadata"`
	Status        string            `json:"status"`
}

// ConstructWebhookEvent verifies and parses a Stripe webhook payload.
func (s *Service) ConstructWebhookEvent(payload []byte, sigHeader, webhookSecret string) (*WebhookEvent, error) {
	if webhookSecret == "" {
		return nil, errors.New("stripe webhook secret is not configured")
	}
	if err := verifyWebhookSignature(payload, sigHeader, webhookSecret, 5*time.Minute); err != nil {
		return nil, err
	}

	var evt WebhookEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("parse webhook: %w", err)
	}
	return &evt, nil
}

type WebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// ChargeOffSession creates and confirms a PaymentIntent off-session.
// amountUSDCents is in real USD cents (e.g. 1000 = $10.00).
func (s *Service) ChargeOffSession(customerID, paymentMethodID string, amountUSDCents int64, idempotencyKey string) (string, error) {
	vals := url.Values{
		"amount":                    {fmt.Sprintf("%d", amountUSDCents)},
		"currency":                  {"usd"},
		"customer":                  {customerID},
		"payment_method":            {paymentMethodID},
		"confirm":                   {"true"},
		"off_session":               {"true"},
		"error_on_requires_action":  {"true"},
		"description":               {"OpenPath auto-topup"},
		"metadata[type]":            {"auto_topup"},
	}
	var resp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	headers := map[string]string{}
	if idempotencyKey != "" {
		headers["Idempotency-Key"] = idempotencyKey
	}
	if err := s.postWithHeaders("/v1/payment_intents", vals, &resp, headers); err != nil {
		return "", fmt.Errorf("charge off-session: %w", err)
	}
	if resp.Status != "succeeded" {
		msg := resp.Status
		if resp.Error != nil {
			msg = resp.Error.Message
		}
		return resp.ID, fmt.Errorf("payment not succeeded: %s", msg)
	}
	return resp.ID, nil
}

func (s *Service) post(path string, vals url.Values, out any) error {
	return s.postWithHeaders(path, vals, out, nil)
}

func (s *Service) postWithHeaders(path string, vals url.Values, out any, headers map[string]string) error {
	var body io.Reader
	if vals != nil {
		body = strings.NewReader(vals.Encode())
	}
	req, err := http.NewRequest("POST", "https://api.stripe.com"+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func (s *Service) get(path string, out any) error {
	req, err := http.NewRequest("GET", "https://api.stripe.com"+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.secretKey, "")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("stripe %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func verifyWebhookSignature(payload []byte, sigHeader, webhookSecret string, tolerance time.Duration) error {
	timestamp, signatures, err := parseSignatureHeader(sigHeader)
	if err != nil {
		return err
	}
	if tolerance > 0 && time.Since(timestamp) > tolerance {
		return errors.New("stripe webhook signature timestamp is too old")
	}

	signedPayload := fmt.Sprintf("%d.%s", timestamp.Unix(), payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := mac.Sum(nil)

	for _, sig := range signatures {
		decoded, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare(decoded, expected) == 1 {
			return nil
		}
	}

	return errors.New("stripe webhook signature verification failed")
}

func parseSignatureHeader(sigHeader string) (time.Time, []string, error) {
	if sigHeader == "" {
		return time.Time{}, nil, errors.New("missing Stripe-Signature header")
	}

	var (
		timestamp time.Time
		signatures []string
	)

	for _, part := range strings.Split(sigHeader, ",") {
		part = strings.TrimSpace(part)
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}

		switch key {
		case "t":
			secs, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return time.Time{}, nil, fmt.Errorf("invalid stripe signature timestamp: %w", err)
			}
			timestamp = time.Unix(secs, 0)
		case "v1":
			signatures = append(signatures, value)
		}
	}

	if timestamp.IsZero() {
		return time.Time{}, nil, errors.New("missing stripe signature timestamp")
	}
	if len(signatures) == 0 {
		return time.Time{}, nil, errors.New("missing stripe v1 signature")
	}

	return timestamp, signatures, nil
}
