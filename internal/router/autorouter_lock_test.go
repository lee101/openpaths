package router

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type gateEmbedder struct {
	gate    chan struct{}
	lastLen int
	mu      sync.Mutex
}

func (g *gateEmbedder) Name() string { return "gate" }

func (g *gateEmbedder) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	if g.gate != nil {
		select {
		case <-g.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	text, _ := req.Input.(string)
	g.mu.Lock()
	g.lastLen = len(text)
	g.mu.Unlock()
	return &model.EmbeddingResponse{Data: []model.EmbeddingData{{Embedding: []float64{1, 0, 0}}}}, nil
}

func TestResolveAutoDoesNotBlockWhileInitRuns(t *testing.T) {
	emb := &gateEmbedder{gate: make(chan struct{})}
	ar := NewAutoRouter(emb)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- ar.Init(ctx) }()
	time.Sleep(50 * time.Millisecond)
	res := make(chan error, 1)
	go func() {
		_, err := ar.ResolveAuto(context.Background(), "text", "hi")
		res <- err
	}()
	select {
	case err := <-res:
		if err == nil {
			t.Fatal("want not-initialized error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ResolveAuto blocked behind Init")
	}
	cancel()
	<-done
}

func TestResolveAutoClipsLongPrompt(t *testing.T) {
	emb := &gateEmbedder{}
	ar := NewAutoRouter(emb)
	if err := ar.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	long := make([]rune, 20000)
	for i := range long {
		long[i] = 'é'
	}
	if _, err := ar.ResolveAuto(context.Background(), "text", string(long)); err != nil {
		t.Fatal(err)
	}
	if emb.lastLen > maxAutoPromptBytes {
		t.Fatalf("embedded %d bytes", emb.lastLen)
	}
	if got := clipUTF8("aé", 2); got != "a" {
		t.Fatalf("clipUTF8 split rune: %q", got)
	}
}

func TestSetAutoRouterConcurrentWithResolve(t *testing.T) {
	r := newTestRouter(nil)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			r.SetAutoRouter(NewAutoRouter(&gateEmbedder{}))
		}()
		go func() {
			defer wg.Done()
			r.MaybeResolveAuto(context.Background(), "openpaths/auto", "text", "hi")
		}()
	}
	wg.Wait()
}

func TestMarkModelFailureSkipsRequestErrors(t *testing.T) {
	r := newTestRouter(nil)
	key := r.health.ModelProviderKey("p", "m")
	r.MarkModelFailure("p", "m", &provider.ProviderError{Provider: "p", StatusCode: 400})
	if !r.health.IsHealthy(key) {
		t.Fatal("400 must not mark unhealthy")
	}
	r.MarkModelFailure("p", "m", &provider.ProviderError{Provider: "p", StatusCode: 503, Retryable: true})
	if r.health.IsHealthy(key) {
		t.Fatal("503 must mark unhealthy")
	}
	r.MarkModelHealthy("p", "m")
	r.MarkModelFailure("p", "m", errors.New("boom"))
	if r.health.IsHealthy(key) {
		t.Fatal("generic error must mark unhealthy")
	}
}
