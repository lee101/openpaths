package stripe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
