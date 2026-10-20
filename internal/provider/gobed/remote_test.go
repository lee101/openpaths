package gobed

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestRemoteDecodesBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "no", 401)
			return
		}
		var req model.EmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.EncodingFormat != "base64" {
			http.Error(w, "fmt", 400)
			return
		}
		b := make([]byte, 8)
		binary.LittleEndian.PutUint32(b, math.Float32bits(1.5))
		binary.LittleEndian.PutUint32(b[4:], math.Float32bits(-2))
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"embedding": base64.StdEncoding.EncodeToString(b), "index": 0}}, "usage": map[string]int{"total_tokens": 3}})
	}))
	defer srv.Close()
	r := NewRemote(srv.URL, "k")
	resp, err := r.Embed(context.Background(), &model.EmbeddingRequest{Model: "openpaths-embed", Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Embedding[0] != 1.5 || resp.Data[0].Embedding[1] != -2 || resp.Usage.TotalTokens != 3 {
		t.Fatalf("%+v", resp)
	}
	if _, err := NewRemote(srv.URL, "bad").Embed(context.Background(), &model.EmbeddingRequest{Input: "x"}); err == nil {
		t.Fatal("want auth error")
	}
}
