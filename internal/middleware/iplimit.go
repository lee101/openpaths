package middleware

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// IPRateLimit throttles unauthenticated endpoints (register/login) by client IP
// and by /24 (or /48) network block. RateLimit() only covers requests that
// already carry an API key, so without this a bot can mint accounts and keys
// at will before any key-scoped limit applies.

const ipRLShards = 32

type ipWindow struct {
	count   int
	resetAt time.Time
}

type ipCounters struct {
	minute ipWindow
	hour   ipWindow
	day    ipWindow
	seen   time.Time
}

type ipLimiter struct {
	shards [ipRLShards]struct {
		mu   sync.Mutex
		keys map[string]*ipCounters
	}
}

func newIPLimiter() *ipLimiter {
	l := &ipLimiter{}
	for i := range l.shards {
		l.shards[i].keys = make(map[string]*ipCounters)
	}
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			cutoff := time.Now().Add(-24 * time.Hour)
			for i := range l.shards {
				s := &l.shards[i]
				s.mu.Lock()
				for k, c := range s.keys {
					if c.seen.Before(cutoff) {
						delete(s.keys, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
	return l
}

func ipWindowAllow(w *ipWindow, limit int, now time.Time, d time.Duration) (bool, time.Duration) {
	if limit <= 0 {
		return false, d
	}
	if now.After(w.resetAt) {
		w.count = 0
		w.resetAt = now.Add(d)
	}
	if w.count >= limit {
		return false, time.Until(w.resetAt)
	}
	return true, 0
}

func (l *ipLimiter) allow(key string, perMin, perHour, perDay int) (bool, time.Duration) {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	s := &l.shards[h%ipRLShards]
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	c, ok := s.keys[key]
	if !ok {
		c = &ipCounters{}
		s.keys[key] = c
	}
	c.seen = now

	if ok, retry := ipWindowAllow(&c.minute, perMin, now, time.Minute); !ok {
		return false, retry
	}
	if ok, retry := ipWindowAllow(&c.hour, perHour, now, time.Hour); !ok {
		return false, retry
	}
	if ok, retry := ipWindowAllow(&c.day, perDay, now, 24*time.Hour); !ok {
		return false, retry
	}
	c.minute.count++
	c.hour.count++
	c.day.count++
	return true, 0
}

func envInt(name string, def int) int {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			return v
		}
	}
	return def
}

// clientIPFor resolves the caller IP. Forwarded headers are only trusted when
// we sit behind a proxy we control; otherwise a rotating X-Forwarded-For resets
// the budget on every request.
func clientIPFor(ctx *fasthttp.RequestCtx) string {
	if proxyHeadersTrusted(ctx.RemoteIP()) {
		if v := strings.TrimSpace(string(ctx.Request.Header.Peek("CF-Connecting-IP"))); validForwardedIP(v) {
			return v
		}
		if v := strings.TrimSpace(string(ctx.Request.Header.Peek("X-Real-IP"))); v != "" {
			if validForwardedIP(v) {
				return v
			}
		}
		if xff := string(ctx.Request.Header.Peek("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			if v := strings.TrimSpace(parts[0]); validForwardedIP(v) {
				return v
			}
		}
	}
	return ctx.RemoteIP().String()
}

func validForwardedIP(raw string) bool {
	return net.ParseIP(strings.TrimSpace(raw)) != nil
}

// proxyHeadersTrusted requires both an explicit opt-in and a trusted immediate
// peer. TRUSTED_PROXY_CIDRS is comma-separated; no list means no proxy is
// trusted. This is deliberately secure-by-default for direct/open-source use.
func proxyHeadersTrusted(peer net.IP) bool {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("TRUST_PROXY_HEADERS")), "true") || peer == nil {
		return false
	}
	for _, raw := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if ip := net.ParseIP(raw); ip != nil && ip.Equal(peer) {
			return true
		}
		if _, block, err := net.ParseCIDR(raw); err == nil && block.Contains(peer) {
			return true
		}
	}
	return false
}

func networkKeyFor(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	return ip.Mask(net.CIDRMask(48, 128)).String() + "/48"
}

var authIPLimiter = newIPLimiter()

// IPRateLimit returns a middleware throttling a public endpoint by IP and
// network. name namespaces the counters so login and register budgets are
// independent.
func IPRateLimit(name string, perMin, perHour, perDay int) Middleware {
	netMin, netHour, netDay := perMin*4, perHour*4, perDay*4
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if os.Getenv("DISABLE_IP_RATE_LIMIT") == "true" {
				next(ctx)
				return
			}
			ip := clientIPFor(ctx)
			if ok, retry := authIPLimiter.allow(name+":net:"+networkKeyFor(ip), netMin, netHour, netDay); !ok {
				rejectIP(ctx, retry)
				return
			}
			if ok, retry := authIPLimiter.allow(name+":ip:"+ip, perMin, perHour, perDay); !ok {
				rejectIP(ctx, retry)
				return
			}
			next(ctx)
		}
	}
}

func rejectIP(ctx *fasthttp.RequestCtx, retry time.Duration) {
	ctx.SetStatusCode(429)
	ctx.SetContentType("application/json")
	if retry > 0 {
		ctx.Response.Header.Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
	}
	ctx.SetBodyString(`{"error":{"message":"Too many requests from this IP. Please try again later.","type":"rate_limit_error","code":"rate_limit_exceeded"}}`)
}

// RegisterIPLimits / LoginIPLimits are the tuned defaults, overridable by env.
func RegisterIPLimits() (int, int, int) {
	return envInt("REGISTER_PER_MIN", 2), envInt("REGISTER_PER_HOUR", 5), envInt("REGISTER_PER_DAY", 10)
}

func LoginIPLimits() (int, int, int) {
	return envInt("LOGIN_PER_MIN", 10), envInt("LOGIN_PER_HOUR", 60), envInt("LOGIN_PER_DAY", 300)
}
