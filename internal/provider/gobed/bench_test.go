package gobed

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func chdirRoot() { _ = os.Chdir("../../..") }

func benchInputs(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = fmt.Sprintf("how do I reset the password for account number %d on the billing portal quickly", i)
	}
	return out
}

func BenchmarkProviderEmbed256(b *testing.B) {
	chdirRoot()
	p, err := New()
	if err != nil {
		b.Skip(err)
	}
	in := benchInputs(256)
	req := &model.EmbeddingRequest{Model: "gobed", Input: in}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Embed(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.N*256)/b.Elapsed().Seconds(), "emb/s")
}

func BenchmarkProviderEmbed256Parallel(b *testing.B) {
	chdirRoot()
	p, err := New()
	if err != nil {
		b.Skip(err)
	}
	in := benchInputs(256)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := &model.EmbeddingRequest{Model: "gobed", Input: in}
		for pb.Next() {
			if _, err := p.Embed(context.Background(), req); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.ReportMetric(float64(b.N*256)/b.Elapsed().Seconds(), "emb/s")
}
