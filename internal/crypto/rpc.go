package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RPCEndpoint struct {
	Name            string
	URL             string
	RateLimitRPS    float64
	lastRequestTime time.Time
	failures        int
	backoffUntil    time.Time
}

type RPCClient struct {
	endpoints []*RPCEndpoint
	mu        sync.Mutex
}

func NewRPCClient(endpoints []*RPCEndpoint) *RPCClient {
	return &RPCClient{endpoints: endpoints}
}

func (c *RPCClient) Call(method string, params interface{}, timeout time.Duration) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		for _, ep := range c.endpoints {
			if time.Now().Before(ep.backoffUntil) {
				continue
			}
			minInterval := time.Duration(float64(time.Second) / ep.RateLimitRPS)
			if time.Since(ep.lastRequestTime) < minInterval {
				time.Sleep(minInterval - time.Since(ep.lastRequestTime))
			}
			ep.lastRequestTime = time.Now()

			body, err := c.doRequest(ep.URL, method, params, timeout)
			if err != nil {
				if isRateLimit(err) {
					ep.failures++
					backoff := time.Duration(1<<ep.failures) * time.Second
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
					ep.backoffUntil = time.Now().Add(backoff)
				}
				lastErr = err
				continue
			}
			ep.failures = 0
			ep.backoffUntil = time.Time{}
			return body, nil
		}
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(1<<attempt) * 500 * time.Millisecond)
		}
	}
	return nil, fmt.Errorf("all RPC endpoints failed after %d attempts: %v", maxRetries, lastErr)
}

func (c *RPCClient) doRequest(rpcURL, method string, params interface{}, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", rpcURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("429 rate limited")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpcResp); err == nil && rpcResp.Error != nil {
		if rpcResp.Error.Code == 429 || strings.Contains(strings.ToLower(rpcResp.Error.Message), "rate") {
			return nil, fmt.Errorf("429 rate limited: %s", rpcResp.Error.Message)
		}
	}

	return body, nil
}

func isRateLimit(err error) bool {
	s := err.Error()
	return strings.Contains(s, "429") || strings.Contains(strings.ToLower(s), "rate")
}
