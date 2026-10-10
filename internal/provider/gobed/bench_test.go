package gobed

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

var benchText = strings.Repeat("The quick brown fox jumps over the lazy dog while embedding retrieval benchmarks run on commodity hardware. ", 3)

func benchProvider(b *testing.B) *GobedProvider {
	if d := os.Getenv("GOBED_MODEL_DIR"); d != "" {
		if err := os.Chdir(d); err != nil {
			b.Skip(err)
		}
	}
	p, err := New()
	if err != nil {
		b.Skipf("model unavailable: %v", err)
	}
	return p
}

func BenchmarkEmbedSingle(b *testing.B) {
	p := benchProvider(b)
	req := &model.EmbeddingRequest{Model: "openpaths-embed", Input: benchText}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Embed(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmbedBatch64(b *testing.B) {
	p := benchProvider(b)
	in := make([]any, 64)
	for i := range in {
		in[i] = benchText
	}
	req := &model.EmbeddingRequest{Model: "openpaths-embed", Input: in}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.Embed(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmbedParallel(b *testing.B) {
	p := benchProvider(b)
	req := &model.EmbeddingRequest{Model: "openpaths-embed", Input: benchText}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := p.Embed(context.Background(), req); err != nil {
				b.Fatal(err)
			}
		}
	})
}
