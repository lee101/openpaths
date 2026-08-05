// Package safefetch provides HTTP clients that cannot connect to private or
// special-purpose networks. It is intended for URLs supplied by API callers.
package safefetch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxRedirects = 10

// NewClient returns an HTTP client that validates every DNS answer and every
// redirect target, then dials the validated address directly to prevent DNS
// rebinding between validation and connection.
func NewClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Caller-supplied URLs must never escape through an environment-configured
	// forward proxy, which could resolve or fetch private destinations itself.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("unsafe URL address: %w", err)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("resolve %q: no addresses", host)
		}
		for _, resolved := range ips {
			if !PublicIP(resolved.IP) {
				return nil, fmt.Errorf("unsafe URL address %s", resolved.IP)
			}
		}
		var lastErr error
		for _, resolved := range ips {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return ValidateURL(req.URL)
		},
	}
}

// ValidateURL rejects non-HTTP URLs, credentials, and malformed hosts. Address
// safety is enforced after DNS resolution by the client's transport.
func ValidateURL(u *url.URL) error {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("URL must use http or https with a host")
	}
	if u.User != nil {
		return fmt.Errorf("URL credentials are not allowed")
	}
	if ip := net.ParseIP(strings.Trim(u.Hostname(), "[]")); ip != nil && !PublicIP(ip) {
		return fmt.Errorf("unsafe URL address %s", ip)
	}
	return nil
}

// PublicIP reports whether ip is globally routable and not part of a private,
// loopback, link-local, carrier-grade NAT, or benchmarking range.
func PublicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, raw := range []string{"100.64.0.0/10", "198.18.0.0/15"} {
		_, block, _ := net.ParseCIDR(raw)
		if block.Contains(ip) {
			return false
		}
	}
	return true
}
