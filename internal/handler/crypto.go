package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/crypto"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
)

type CryptoHandler struct {
	svc *crypto.Service
}

func NewCryptoHandler(svc *crypto.Service) *CryptoHandler {
	return &CryptoHandler{svc: svc}
}

func (h *CryptoHandler) HandleCreateCheckout(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req model.CryptoCheckoutRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	resp, err := h.svc.CreateCheckout(ctx, userID, &req)
	if err != nil {
		writeError(ctx, 400, "crypto_error", err.Error())
		return
	}
	writeJSON(ctx, 200, resp)
}

func (h *CryptoHandler) HandleGetCheckout(ctx *fasthttp.RequestCtx) {
	id, _ := ctx.UserValue("id").(string)
	intent, err := h.svc.GetCheckoutStatus(ctx, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "Checkout not found")
		return
	}
	writeJSON(ctx, 200, intent)
}

func (h *CryptoHandler) HandleCheckoutEvents(ctx *fasthttp.RequestCtx) {
	id, _ := ctx.UserValue("id").(string)
	intent, err := h.svc.GetCheckoutStatus(ctx, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "Checkout not found")
		return
	}

	ctx.Response.Header.Set("Content-Type", "text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")

	ch := h.svc.Subscribe(id)
	defer h.svc.Unsubscribe(id, ch)

	statusData, _ := json.Marshal(map[string]string{"status": string(intent.Status), "tx_sig": intent.TxSig})
	ctx.WriteString(fmt.Sprintf("event: status\ndata: %s\n\n", statusData))
	ctx.Response.ImmediateHeaderFlush = true

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	timeout := time.After(20 * time.Minute)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			ctx.WriteString(fmt.Sprintf("event: status\ndata: %s\n\n", msg))
			var statusMsg map[string]string
			json.Unmarshal([]byte(msg), &statusMsg)
			if statusMsg["status"] == "paid" || statusMsg["status"] == "expired" {
				return
			}
		case <-ticker.C:
			ctx.WriteString(": keepalive\n\n")
		case <-timeout:
			return
		}
	}
}

func (h *CryptoHandler) HandlePrices(ctx *fasthttp.RequestCtx) {
	prices := h.svc.Prices()
	writeJSON(ctx, 200, model.CryptoPricesResponse{
		SOL: model.CryptoPriceInfo{
			PriceUSD: prices.SOLPrice(),
		},
		CODEX: model.CryptoPriceInfo{
			PriceUSD: prices.CODEXPrice(),
			Mint:     prices.CODEXMint(),
		},
	})
}
