package middleware

import (
	"encoding/json"
	"os"

	"github.com/valyala/fasthttp"
)

const (
	SubscribeURLHeader       = "X-Subscribe-URL"
	SubscriptionRequiredCode = "subscription_required"
	defaultSubscribeURL      = "https://openpaths.io/pricing"
)

func SubscribeURL() string {
	if v := os.Getenv("OPENPATHS_SUBSCRIBE_URL"); v != "" {
		return v
	}
	return defaultSubscribeURL
}

var authCodes = map[string]bool{
	"missing_api_key": true,
	"invalid_api_key": true,
	"missing_auth":    true,
	"invalid_token":   true,
	"missing_token":   true,
}

// PaywallContract normalises paid GPU-route failures to the shared gateway
// contract: 401 stays for our own auth failures; any 402 (balance precheck,
// deduct failure, upstream 402) and upstream 401 (provider_error) become 402
// {"error":{"code":"subscription_required","message","subscribe_url"}} plus
// X-Subscribe-URL.
func PaywallContract(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		next(ctx)
		status := ctx.Response.StatusCode()
		if status != 401 && status != 402 {
			return
		}
		var body struct {
			Error map[string]any `json:"error"`
		}
		_ = json.Unmarshal(ctx.Response.Body(), &body)
		if body.Error == nil {
			body.Error = map[string]any{}
		}
		code, _ := body.Error["code"].(string)
		if status == 401 && (authCodes[code] || body.Error["type"] == "auth_error") {
			return
		}
		msg, _ := body.Error["message"].(string)
		if msg == "" || status == 401 {
			msg = "An active OpenPaths balance is required to run this model. Add credits to continue."
		}
		if code != "" && code != SubscriptionRequiredCode {
			body.Error["reason"] = code
		}
		url := SubscribeURL()
		body.Error["code"] = SubscriptionRequiredCode
		body.Error["type"] = "billing_error"
		body.Error["message"] = msg
		body.Error["subscribe_url"] = url
		out, _ := json.Marshal(body)
		ctx.SetStatusCode(402)
		ctx.SetContentType("application/json")
		ctx.Response.Header.Set(SubscribeURLHeader, url)
		ctx.SetBody(out)
	}
}
