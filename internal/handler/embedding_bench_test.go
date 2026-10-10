package handler

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	gobedprov "github.com/openpaths/openpaths/internal/provider/gobed"
	"github.com/openpaths/openpaths/internal/router"
)

func BenchmarkEmbeddingHandlerE2E(b *testing.B) {
	dir := os.Getenv("GOBED_MODEL_DIR")
	if dir == "" {
		b.Skip("GOBED_MODEL_DIR not set")
	}
	if err := os.Chdir(dir); err != nil {
		b.Skip(err)
	}
	gp, err := gobedprov.New()
	if err != nil {
		b.Skip(err)
	}
	models := []model.ModelConfig{{ID: "openpaths-embed", Provider: "gobed", InputPricePer1M: 0.001, SubUnitBilling: true}}
	eng := billing.NewEngine(billing.NewPricingTable(models), nil)
	coll := metrics.NewCollector(nil, time.Hour)
	h := NewEmbeddingHandler(router.New(provider.NewRegistry(), models), eng, metrics.NewRecorder(coll), []provider.EmbeddingProvider{gp})

	text := strings.Repeat("The quick brown fox jumps over the lazy dog while embedding retrieval benchmarks run on commodity hardware. ", 3)
	for _, tc := range []struct {
		name string
		body string
		toks int
	}{
		{"single", `{"model":"openpaths-embed","input":"` + text + `"}`, 70},
		{"batch32", `{"model":"openpaths-embed","input":[` + strings.TrimSuffix(strings.Repeat(`"`+text+`",`, 32), ",") + `]}`, 70 * 32},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				ctx := &fasthttp.RequestCtx{}
				for pb.Next() {
					ctx.Request.Reset()
					ctx.Response.Reset()
					ctx.SetUserValue(middleware.CtxKeyUserID, "u1")
					ctx.Request.SetBodyString(tc.body)
					h.HandleEmbedding(ctx)
					if ctx.Response.StatusCode() != 200 {
						b.Fatalf("status %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
					}
				}
			})
			b.ReportMetric(float64(tc.toks)*float64(b.N)/b.Elapsed().Seconds()/1e6, "Mtok/s")
		})
	}
}
