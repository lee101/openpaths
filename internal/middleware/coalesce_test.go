package middleware

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

func TestRequestCoalescerSharesPendingGeneration(t *testing.T) {
	var calls int32
	started := make(chan struct{})
	release := make(chan struct{})
	handler := NewRequestCoalescer(time.Hour)(func(ctx *fasthttp.RequestCtx) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(started)
			<-release
		}
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"ok":true}`)
	})

	var first fasthttp.RequestCtx
	first.Request.Header.SetMethod("POST")
	first.Request.SetRequestURI("/v1/images/generations")
	first.Request.SetBodyString(`{"model":"x","prompt":"same"}`)
	first.SetUserValue(CtxKeyUserID, "user-1")
	first.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})

	var second fasthttp.RequestCtx
	second.Request.Header.SetMethod("POST")
	second.Request.SetRequestURI("/v1/images/generations")
	second.Request.SetBodyString(`{"model":"x","prompt":"same"}`)
	second.SetUserValue(CtxKeyUserID, "user-1")
	second.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		handler(&first)
	}()
	<-started
	go func() {
		defer wg.Done()
		handler(&second)
	}()
	time.Sleep(25 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
	if string(second.Response.Body()) != `{"ok":true}` {
		t.Fatalf("second body = %s", second.Response.Body())
	}
}

func TestRequestCoalescerDoesNotCacheErrors(t *testing.T) {
	var calls int32
	handler := NewRequestCoalescer(time.Hour)(func(ctx *fasthttp.RequestCtx) {
		if atomic.AddInt32(&calls, 1) == 1 {
			ctx.SetStatusCode(502)
			ctx.SetBodyString(`{"error":true}`)
			return
		}
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"ok":true}`)
	})

	for i := 0; i < 2; i++ {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/3d/generations")
		ctx.Request.SetBodyString(`{"model":"x"}`)
		ctx.SetUserValue(CtxKeyUserID, "user-1")
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})
		handler(&ctx)
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("handler calls = %d, want 2", got)
	}
}

func TestRequestCoalescerIsScopedByUser(t *testing.T) {
	var calls int32
	handler := NewRequestCoalescer(time.Hour)(func(ctx *fasthttp.RequestCtx) {
		atomic.AddInt32(&calls, 1)
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"ok":true}`)
	})

	for _, userID := range []string{"user-1", "user-2"} {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/images/generations")
		ctx.Request.SetBodyString(`{"model":"x","prompt":"same"}`)
		ctx.SetUserValue(CtxKeyUserID, userID)
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-" + userID})
		handler(&ctx)
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("handler calls = %d, want 2", got)
	}
}

func TestRequestCoalescerRefreshesExpiredEntry(t *testing.T) {
	var calls int32
	handler := NewRequestCoalescer(time.Nanosecond)(func(ctx *fasthttp.RequestCtx) {
		atomic.AddInt32(&calls, 1)
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"ok":true}`)
	})

	for i := 0; i < 2; i++ {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/images/generations")
		ctx.Request.SetBodyString(`{"model":"x","prompt":"same"}`)
		ctx.SetUserValue(CtxKeyUserID, "user-1")
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})
		handler(&ctx)
		time.Sleep(time.Millisecond)
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("handler calls = %d, want 2", got)
	}
}

func TestRequestCoalescerCoversOtherExpensiveModelRoutes(t *testing.T) {
	for _, path := range []string{"/v1/audio/transcriptions", "/v1/stt", "/v1/embeddings"} {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI(path)
		if !coalesceable(&ctx) {
			t.Fatalf("%s should be coalesced", path)
		}
	}
}

func BenchmarkRequestCoalescerUniqueWithLargeCache(b *testing.B) {
	const existingEntries = 10000
	handler := NewRequestCoalescer(time.Hour)(func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(`{"ok":true}`)
	})

	for i := 0; i < existingEntries; i++ {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/images/generations")
		ctx.Request.SetBodyString(fmt.Sprintf(`{"model":"x","prompt":"seed-%d"}`, i))
		ctx.SetUserValue(CtxKeyUserID, "user-1")
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})
		handler(&ctx)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/images/generations")
		ctx.Request.SetBodyString(fmt.Sprintf(`{"model":"x","prompt":"bench-%d"}`, i))
		ctx.SetUserValue(CtxKeyUserID, "user-1")
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-1"})
		handler(&ctx)
	}
}

func BenchmarkRequestCoalescerSkipsChat(b *testing.B) {
	handler := NewRequestCoalescer(time.Hour)(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(200)
	})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetRequestURI("/v1/chat/completions")
		handler(&ctx)
	}
}
