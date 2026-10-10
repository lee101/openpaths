package gobed

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func chdirRoot() {
	if d := os.Getenv("GOBED_MODEL_DIR"); d != "" {
		_ = os.Chdir(d)
		return
	}
	_ = os.Chdir("../../..")
}

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
	req := &model.EmbeddingRequest{Model: "gobed", Input: benchInputs(256)}
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

func newTestProvider(t testing.TB) *GobedProvider {
	chdirRoot()
	p, err := New()
	if err != nil {
		t.Skip(err)
	}
	return p
}

func TestEmbedRawMatchesReference(t *testing.T) {
	p := newTestProvider(t)
	texts := []any{"hello world", "", "reset my password please", "a b c d e f g"}
	raw, err := p.EmbedRaw(context.Background(), &model.EmbeddingRequest{Input: texts})
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Release()
	if raw.N != 4 || raw.Dim != 512 {
		t.Fatalf("shape %d x %d", raw.N, raw.Dim)
	}
	for i, in := range texts {
		want, _ := p.model.Embed(in.(string))
		for j, w := range want {
			if d := math.Abs(float64(raw.Vec(i)[j] - w)); d > 1e-4 {
				t.Fatalf("text %d dim %d: got %v want %v", i, j, raw.Vec(i)[j], w)
			}
		}
	}
	if raw.Tokens == 0 {
		t.Fatal("tokens not counted")
	}
}

func TestEmbedDimensionsNormalized(t *testing.T) {
	p := newTestProvider(t)
	raw, err := p.EmbedRaw(context.Background(), &model.EmbeddingRequest{Input: "search query text", Dimensions: 128})
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Release()
	if raw.Dim != 128 || len(raw.Vecs) != 128 {
		t.Fatalf("dim %d len %d", raw.Dim, len(raw.Vecs))
	}
	var ss float64
	for _, x := range raw.Vecs {
		ss += float64(x) * float64(x)
	}
	if math.Abs(ss-1) > 1e-3 {
		t.Fatalf("norm^2 = %v", ss)
	}
}

func TestEmbedValidation(t *testing.T) {
	p := newTestProvider(t)
	for _, req := range []*model.EmbeddingRequest{
		{Input: "x", Dimensions: 513},
		{Input: "x", Dimensions: -1},
		{Input: "x", LongTextMode: "bogus"},
		{Input: 5},
		{Input: make([]string, MaxInputsPerRequest+1)},
	} {
		_, err := p.EmbedRaw(context.Background(), req)
		var pe *provider.ProviderError
		if !errors.As(err, &pe) || pe.StatusCode != 400 {
			t.Fatalf("%+v: err = %v", req, err)
		}
	}
}

func TestEmbedLegacyAndLarge(t *testing.T) {
	p := newTestProvider(t)
	in := benchInputs(300)
	resp, err := p.Embed(context.Background(), &model.EmbeddingRequest{Model: "m", Input: in})
	if err != nil || len(resp.Data) != 300 || resp.Data[299].Index != 299 || len(resp.Data[7].Embedding) != 512 {
		t.Fatalf("resp err=%v", err)
	}
	r2, _ := p.Embed(context.Background(), &model.EmbeddingRequest{Input: in[7], LongTextMode: "average_chunks"})
	for j := range r2.Data[0].Embedding {
		if math.Abs(r2.Data[0].Embedding[j]-resp.Data[7].Embedding[j]) > 1e-4 {
			t.Fatal("average_chunks differs for short text")
		}
	}
}

type stubOverflow struct{ calls int }

func (s *stubOverflow) Name() string { return "stub" }
func (s *stubOverflow) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	s.calls++
	return &model.EmbeddingResponse{Data: []model.EmbeddingData{{Embedding: []float64{1, 2}}}, Usage: model.EmbeddingUsage{TotalTokens: 3}}, nil
}

func TestSaturationBackpressureAndOverflow(t *testing.T) {
	p := newTestProvider(t)
	p.pool = newBatcher(1, 1, 0, 0)
	block := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	// occupy the single worker and the single queue slot
	p.pool.q <- job{run: func() { <-block }, done: &wg}
	p.pool.q <- job{run: func() {}, done: &wg}
	time.Sleep(20 * time.Millisecond)

	_, err := p.EmbedRaw(context.Background(), &model.EmbeddingRequest{Input: "x"})
	var pe *provider.ProviderError
	if !errors.As(err, &pe) || pe.StatusCode != 429 || pe.Retryable {
		t.Fatalf("want non-retryable 429, got %v", err)
	}

	ov := &stubOverflow{}
	p.SetOverflow(ov)
	raw, err := p.EmbedRaw(context.Background(), &model.EmbeddingRequest{Input: "x"})
	if err != nil || ov.calls != 1 || raw.N != 1 || raw.Tokens != 3 {
		t.Fatalf("overflow err=%v calls=%d", err, ov.calls)
	}
	close(block)
	wg.Wait()
}
