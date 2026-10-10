package handler

import (
	"encoding/json"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

func newModelsHandlerForTest() *ModelsHandler {
	models := []model.ModelConfig{
		{ID: "b-model", Provider: "p", ProviderModelID: "b"},
		{ID: "a-model", Provider: "p", ProviderModelID: "a"},
	}
	return NewModelsHandler(router.New(provider.NewRegistry(), models))
}

func TestHandleListModelsCachedAndSorted(t *testing.T) {
	h := newModelsHandlerForTest()
	call := func() []byte {
		var ctx fasthttp.RequestCtx
		h.HandleListModels(&ctx)
		if ctx.Response.StatusCode() != 200 {
			t.Fatalf("status %d", ctx.Response.StatusCode())
		}
		return append([]byte(nil), ctx.Response.Body()...)
	}
	first, second := call(), call()
	if string(first) != string(second) {
		t.Fatal("cached body differs")
	}
	var out struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(first, &out); err != nil {
		t.Fatal(err)
	}
	if out.Object != "list" || len(out.Data) != 2 || out.Data[0].ID != "a-model" {
		t.Fatalf("unexpected list: %s", first)
	}
}

func BenchmarkHandleListModels(b *testing.B) {
	h := newModelsHandlerForTest()
	var ctx fasthttp.RequestCtx
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h.HandleListModels(&ctx)
	}
}

func TestOpenRouterModelsCachedAndSorted(t *testing.T) {
	models := []model.ModelConfig{
		{ID: "zz-chat", Provider: "p", ProviderModelID: "z", InputPricePer1M: 1, OutputPricePer1M: 2, SupportsStreaming: true},
		{ID: "aa-chat", Provider: "p", ProviderModelID: "a", InputPricePer1M: 1, OutputPricePer1M: 2, SupportsStreaming: true},
	}
	h := NewOpenRouterProviderHandler(router.New(provider.NewRegistry(), models))
	var c1, c2 fasthttp.RequestCtx
	h.HandleListModels(&c1)
	h.HandleListModels(&c2)
	if c1.Response.StatusCode() != 200 || string(c1.Response.Body()) != string(c2.Response.Body()) {
		t.Fatal("not cached or failed")
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(c1.Response.Body(), &out); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(out.Data); i++ {
		if out.Data[i-1].ID > out.Data[i].ID {
			t.Fatalf("unsorted: %v", out.Data)
		}
	}
}
