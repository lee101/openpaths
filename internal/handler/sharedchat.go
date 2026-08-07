package handler

import (
	"crypto/rand"
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

const (
	maxSharedChatBody     = 400 << 10
	maxSharedChatMessages = 200
)

// SharedChatHandler serves share-a-chat: publish a transcript, read it publicly.
type SharedChatHandler struct {
	q *queries.SharedChatQueries
}

func NewSharedChatHandler(q *queries.SharedChatQueries) *SharedChatHandler {
	return &SharedChatHandler{q: q}
}

type sharedChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// HandleShare serves POST /v1/chats/share.
func (h *SharedChatHandler) HandleShare(ctx *fasthttp.RequestCtx) {
	if h.q == nil {
		writeError(ctx, 503, "unavailable", "chat sharing is not configured")
		return
	}
	uid, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if uid == "" {
		writeError(ctx, 401, "unauthorized", "login required")
		return
	}
	body := ctx.PostBody()
	if len(body) > maxSharedChatBody {
		writeError(ctx, 413, "payload_too_large", "shared chat exceeds the 400KB limit")
		return
	}
	var req struct {
		Title        string              `json:"title"`
		Model        string              `json:"model"`
		SystemPrompt string              `json:"system_prompt"`
		Messages     []sharedChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	if len(req.Messages) == 0 {
		writeError(ctx, 400, "invalid_request", "messages required")
		return
	}
	if len(req.Messages) > maxSharedChatMessages {
		writeError(ctx, 400, "invalid_request", "too many messages (max 200)")
		return
	}
	for _, m := range req.Messages {
		switch m.Role {
		case "system", "user", "assistant":
		default:
			writeError(ctx, 400, "invalid_request", "message roles must be system, user or assistant")
			return
		}
	}
	msgs, err := json.Marshal(req.Messages)
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	slug := shareSlug(req.Title)
	if _, err := h.q.Insert(ctx, slug, strings.TrimSpace(req.Title), strings.TrimSpace(req.Model), req.SystemPrompt, msgs, uid); err != nil {
		writeError(ctx, 500, "db_error", err.Error())
		return
	}
	writeJSON(ctx, 200, map[string]any{"slug": slug, "url": "https://openpaths.io/chat/" + slug})
}

// HandleGetShared serves GET /v1/chats/shared?slug= (public).
func (h *SharedChatHandler) HandleGetShared(ctx *fasthttp.RequestCtx) {
	if h.q == nil {
		writeError(ctx, 503, "unavailable", "chat sharing is not configured")
		return
	}
	slug := strings.TrimSpace(string(ctx.QueryArgs().Peek("slug")))
	if slug == "" {
		writeError(ctx, 400, "invalid_request", "slug is required")
		return
	}
	c, err := h.q.GetBySlug(ctx, slug)
	if err != nil {
		writeError(ctx, 404, "not_found", "shared chat not found")
		return
	}
	WriteJSONPublic(ctx, 200, map[string]any{
		"slug":          c.Slug,
		"title":         c.Title,
		"model":         c.Model,
		"system_prompt": c.SystemPrompt,
		"messages":      c.Messages,
		"views":         c.Views,
		"created_at":    c.CreatedAt,
	})
}

func shareSlug(title string) string {
	base := slugifyShareTitle(title)
	if base != "" {
		return base + "-" + randBase36(8)
	}
	return "chat-" + randBase36(8)
}

func slugifyShareTitle(title string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
		if b.Len() >= 40 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

func randBase36(n int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}
