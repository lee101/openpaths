package model

import (
	"encoding/json"
	"testing"
)

func TestEmbeddingResponseAppendJSONMatchesMarshal(t *testing.T) {
	r := &EmbeddingResponse{Object: "list", Model: "m\"x", Usage: EmbeddingUsage{PromptTokens: 3, TotalTokens: 3},
		Data: []EmbeddingData{
			{Object: "embedding", Embedding: []float64{0.5, -1.25e-7, 3}, Index: 0, F32: true},
			{Object: "embedding", Base64: "AAAA", Index: 1},
			{Object: "embedding", Embedding: []float64{0.1}, Index: 2},
		}}
	via, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(r.AppendJSON(nil)) != string(via) {
		t.Fatalf("mismatch\n%s\n%s", r.AppendJSON(nil), via)
	}
}

func BenchmarkMarshalEmbedding(b *testing.B) {
	v := make([]float64, 512)
	for i := range v {
		v[i] = float64(float32(i) * 0.00123)
	}
	r := &EmbeddingResponse{Object: "list", Data: []EmbeddingData{{Object: "embedding", Embedding: v, F32: true}}}
	b.Run("json.Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			json.Marshal(r)
		}
	})
	b.Run("AppendJSON", func(b *testing.B) {
		b.ReportAllocs()
		buf := make([]byte, 0, 8192)
		for i := 0; i < b.N; i++ {
			buf = r.AppendJSON(buf[:0])
		}
	})
}
