package handler

import (
	"fmt"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/middleware"
)

type UnsubscribeHandler struct {
	suppressionQ *queries.SuppressionQueries
	userQ        *queries.UserQueries
	secret       string
}

func NewUnsubscribeHandler(suppressionQ *queries.SuppressionQueries, userQ *queries.UserQueries, secret string) *UnsubscribeHandler {
	return &UnsubscribeHandler{suppressionQ: suppressionQ, userQ: userQ, secret: secret}
}

// HandleAccountUnsubscribe handles POST /account/unsubscribe for logged-in
// users (legacy email links point at /account?unsubscribe=true).
func (h *UnsubscribeHandler) HandleAccountUnsubscribe(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	u, err := h.userQ.GetByID(ctx, userID)
	if err != nil || u == nil {
		writeError(ctx, 404, "not_found", "User not found")
		return
	}
	if err := h.suppressionQ.Add(ctx, u.Email, "unsubscribe"); err != nil {
		writeError(ctx, 500, "server_error", "Failed to unsubscribe")
		return
	}
	writeJSON(ctx, 200, map[string]any{"unsubscribed": true, "email": u.Email})
}

// HandleUnsubscribe handles GET/POST /unsubscribe?e=<email>&t=<token>.
// Token is an HMAC of the email so no login is required; POST supports
// RFC 8058 one-click unsubscribe.
func (h *UnsubscribeHandler) HandleUnsubscribe(ctx *fasthttp.RequestCtx) {
	args := ctx.QueryArgs()
	addr := string(args.Peek("e"))
	token := string(args.Peek("t"))

	if addr == "" || token == "" || !email.VerifyUnsubscribeToken(h.secret, addr, token) {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetContentType("text/html; charset=utf-8")
		ctx.SetBodyString(unsubPage("Invalid unsubscribe link", "This unsubscribe link is invalid or expired. Contact support@openpaths.io and we'll remove you manually."))
		return
	}

	if err := h.suppressionQ.Add(ctx, addr, "unsubscribe"); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentType("text/html; charset=utf-8")
		ctx.SetBodyString(unsubPage("Something went wrong", "We couldn't process your unsubscribe. Contact support@openpaths.io and we'll remove you manually."))
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetBodyString(unsubPage("You're unsubscribed", fmt.Sprintf("%s will no longer receive emails from OpenPaths.", addr)))
}

func unsubPage(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s — OpenPaths</title></head>
<body style="margin:0; background:#0a0a0a; font-family:'Segoe UI',sans-serif; color:#e0e0e0;">
<div style="max-width:480px; margin:80px auto; padding:40px; background:#111; border:1px solid #222; border-radius:12px; text-align:center;">
<h1 style="font-size:22px; margin:0 0 12px;">%s</h1>
<p style="color:#888; font-size:15px; line-height:1.6; margin:0;">%s</p>
<p style="margin:24px 0 0;"><a href="https://openpaths.io" style="color:#f5f5f5;">openpaths.io</a></p>
</div></body></html>`, title, title, body)
}
