package middleware

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
)

const CtxKeyUserProviderKeys = "user_provider_keys"

func BYOKLoader(pkQ *queries.ProviderKeyQueries) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			userID, _ := ctx.UserValue(CtxKeyUserID).(string)
			if userID != "" {
				keys, err := pkQ.ListByUser(ctx, userID)
				if err == nil && len(keys) > 0 {
					m := make(map[string]*queries.UserProviderKey, len(keys))
					for _, k := range keys {
						m[k.Provider] = k
					}
					ctx.SetUserValue(CtxKeyUserProviderKeys, m)
				}
			}
			next(ctx)
		}
	}
}
