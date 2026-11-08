package anthropic

import (
	"net/http"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/promptcache"
)

// maxCacheBreakpoints is Anthropic's per-request cache_control limit.
const maxCacheBreakpoints = 4

// normalizeCacheControl validates a client-supplied cache_control. Only
// {type:"ephemeral"} with an optional ttl of "5m" or "1h" is accepted; anything
// else returns nil so it is dropped rather than rejected upstream.
func normalizeCacheControl(v any) *anthropicCacheControl {
	var typ, ttl string
	switch c := v.(type) {
	case nil:
		return nil
	case *model.CacheControl:
		if c == nil {
			return nil
		}
		typ, ttl = c.Type, c.TTL
	case *anthropicCacheControl:
		if c == nil {
			return nil
		}
		typ, ttl = c.Type, c.TTL
	case map[string]any:
		typ, _ = c["type"].(string)
		if raw, present := c["ttl"]; present && raw != nil {
			str, ok := raw.(string)
			if !ok {
				return nil
			}
			ttl = str
		}
	default:
		return nil
	}
	if typ != "ephemeral" || (ttl != "" && ttl != "5m" && ttl != "1h") {
		return nil
	}
	return &anthropicCacheControl{Type: typ, TTL: ttl}
}

// sanitizeBlockCacheControl validates cache_control on a passthrough content
// block. The caller's map is shared across fallback candidates, so it is copied
// before being changed.
func sanitizeBlockCacheControl(block map[string]any) map[string]any {
	raw, has := block["cache_control"]
	if !has {
		return block
	}
	out := make(map[string]any, len(block))
	for k, v := range block {
		out[k] = v
	}
	if cc := normalizeCacheControl(raw); cc != nil {
		out["cache_control"] = cc
	} else {
		delete(out, "cache_control")
	}
	return out
}

func contentParts(content any) []any {
	switch c := content.(type) {
	case []any:
		return c
	case []map[string]any:
		parts := make([]any, 0, len(c))
		for _, m := range c {
			parts = append(parts, m)
		}
		return parts
	}
	return nil
}

// buildSystem merges every system message, in order, into the top-level system
// field. A lone string message stays a plain string.
func buildSystem(msgs []model.ChatMessage) any {
	if len(msgs) == 0 {
		return nil
	}
	if len(msgs) == 1 {
		if s, ok := msgs[0].Content.(string); ok {
			return s
		}
	}
	var blocks []anthropicTextBlock
	for _, msg := range msgs {
		if s, ok := msg.Content.(string); ok {
			if s != "" {
				blocks = append(blocks, anthropicTextBlock{Type: "text", Text: s})
			}
			continue
		}
		for _, part := range contentParts(msg.Content) {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := m["type"].(string); t != "text" {
				continue
			}
			text, ok := m["text"].(string)
			if !ok || text == "" {
				continue
			}
			blocks = append(blocks, anthropicTextBlock{Type: "text", Text: text, CacheControl: normalizeCacheControl(m["cache_control"])})
		}
	}
	if len(blocks) == 0 {
		return nil
	}
	return blocks
}

type breakpointRef struct {
	get func() *anthropicCacheControl
	set func(*anthropicCacheControl)
}

// breakpoints lists every block that can carry cache_control, in Anthropic's
// cache order: tools, then system, then messages.
func breakpoints(r *anthropicRequest) []breakpointRef {
	var refs []breakpointRef
	for i := range r.Tools {
		t := &r.Tools[i]
		refs = append(refs, breakpointRef{
			get: func() *anthropicCacheControl { return t.CacheControl },
			set: func(cc *anthropicCacheControl) { t.CacheControl = cc },
		})
	}
	if sys, ok := r.System.([]anthropicTextBlock); ok {
		for i := range sys {
			b := &sys[i]
			refs = append(refs, textRef(b))
		}
	}
	for mi := range r.Messages {
		switch c := r.Messages[mi].Content.(type) {
		case []anthropicTextBlock:
			for i := range c {
				refs = append(refs, textRef(&c[i]))
			}
		case []any:
			for i := range c {
				switch b := c[i].(type) {
				case anthropicTextBlock:
					idx := i
					refs = append(refs, breakpointRef{
						get: func() *anthropicCacheControl { return c[idx].(anthropicTextBlock).CacheControl },
						set: func(cc *anthropicCacheControl) {
							tb := c[idx].(anthropicTextBlock)
							tb.CacheControl = cc
							c[idx] = tb
						},
					})
				case map[string]any:
					m := b
					refs = append(refs, breakpointRef{
						get: func() *anthropicCacheControl {
							cc, _ := m["cache_control"].(*anthropicCacheControl)
							return cc
						},
						set: func(cc *anthropicCacheControl) {
							if cc == nil {
								delete(m, "cache_control")
							} else {
								m["cache_control"] = cc
							}
						},
					})
				}
			}
		}
	}
	return refs
}

func textRef(b *anthropicTextBlock) breakpointRef {
	return breakpointRef{
		get: func() *anthropicCacheControl { return b.CacheControl },
		set: func(cc *anthropicCacheControl) { b.CacheControl = cc },
	}
}

// capBreakpoints enforces Anthropic's four-breakpoint limit by dropping the
// earliest breakpoints in cache order.
func capBreakpoints(r *anthropicRequest) {
	refs := breakpoints(r)
	n := 0
	for _, ref := range refs {
		if ref.get() != nil {
			n++
		}
	}
	for _, ref := range refs {
		if n <= maxCacheBreakpoints {
			return
		}
		if ref.get() != nil {
			ref.set(nil)
			n--
		}
	}
}

func hasBreakpoints(r *anthropicRequest) bool {
	for _, ref := range breakpoints(r) {
		if ref.get() != nil {
			return true
		}
	}
	return false
}

// applyCacheControl records the request's prefix timing and sets the cheapest
// cache_control TTL for it, as decided by the optimizer. Client-supplied
// breakpoints always win and suppress automatic ones. No-op when no optimizer
// is attached.
func (p *AnthropicProvider) applyCacheControl(anthReq *anthropicRequest) {
	if p.optimizer == nil || hasBreakpoints(anthReq) {
		return
	}
	// Anthropic only caches prefixes above a minimum size (~1024 tokens for
	// Opus/Sonnet). Below that, cache_control is ignored upstream, so skip it
	// rather than track a prefix that can never produce a cache write/hit.
	if cacheablePrefixChars(anthReq) < minCacheablePrefixChars {
		return
	}
	key := prefixKey(anthReq)
	p.optimizer.Observe(key, anthReq.Model, time.Now())
	switch p.optimizer.Decide(key) {
	case promptcache.TTL5m:
		autoBreakpoints(anthReq, "5m")
	case promptcache.TTL1h:
		autoBreakpoints(anthReq, "1h")
	}
}

// autoBreakpoints covers the static head (last tool, last system block) and
// adds a rolling breakpoint on the last user message so multi-turn history is
// reused. Anthropic's cache order is tools, system, messages.
func autoBreakpoints(r *anthropicRequest, ttl string) {
	newCC := func() *anthropicCacheControl { return &anthropicCacheControl{Type: "ephemeral", TTL: ttl} }
	if n := len(r.Tools); n > 0 {
		r.Tools[n-1].CacheControl = newCC()
	}
	switch sys := r.System.(type) {
	case string:
		if sys != "" {
			r.System = []anthropicTextBlock{{Type: "text", Text: sys, CacheControl: newCC()}}
		}
	case []anthropicTextBlock:
		if n := len(sys); n > 0 && sys[n-1].Text != "" {
			sys[n-1].CacheControl = newCC()
		}
	}
	attachRolling(r, newCC)
}

func attachRolling(r *anthropicRequest, newCC func() *anthropicCacheControl) {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role != "user" {
			continue
		}
		msg := &r.Messages[i]
		switch c := msg.Content.(type) {
		case string:
			if c != "" {
				msg.Content = []anthropicTextBlock{{Type: "text", Text: c, CacheControl: newCC()}}
			}
		case []anthropicTextBlock:
			if n := len(c); n > 0 && c[n-1].Text != "" {
				c[n-1].CacheControl = newCC()
			}
		case []any:
			n := len(c)
			if n == 0 {
				return
			}
			switch b := c[n-1].(type) {
			case anthropicTextBlock:
				if b.Text != "" {
					b.CacheControl = newCC()
					c[n-1] = b
				}
			case map[string]any:
				switch t, _ := b["type"].(string); t {
				case "text", "image", "document", "tool_result", "tool_use":
					out := make(map[string]any, len(b)+1)
					for k, v := range b {
						out[k] = v
					}
					out["cache_control"] = newCC()
					c[n-1] = out
				}
			}
		}
		return
	}
}

// setCacheBeta adds the beta header required for 1h cache TTLs.
func setCacheBeta(httpReq *http.Request, anthReq *anthropicRequest) {
	if requestCacheControlTTL(anthReq) == "1h" {
		httpReq.Header.Set("anthropic-beta", extendedCacheTTLBeta)
	}
}

// requestCacheControlTTL returns "1h" when any breakpoint asks for it, else the
// first explicit TTL, else "".
func requestCacheControlTTL(anthReq *anthropicRequest) string {
	first := ""
	for _, ref := range breakpoints(anthReq) {
		cc := ref.get()
		if cc == nil || cc.TTL == "" {
			continue
		}
		if cc.TTL == "1h" {
			return "1h"
		}
		if first == "" {
			first = cc.TTL
		}
	}
	return first
}
