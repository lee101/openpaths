package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/gobed"
)

func main() {
	p, err := gobed.New()
	if err != nil {
		log.Fatal(err)
	}
	key := os.Getenv("EMBED_WORKER_KEY")
	if key == "" {
		log.Fatal("EMBED_WORKER_KEY required")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8000"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") != key {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		var req model.EmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		raw, err := p.EmbedRaw(ctx, &req)
		if err != nil {
			code := http.StatusInternalServerError
			if pe, ok := err.(*provider.ProviderError); ok {
				code = pe.StatusCode
			}
			if code == http.StatusTooManyRequests {
				w.Header().Set("Retry-After", "1")
			}
			http.Error(w, err.Error(), code)
			return
		}
		defer raw.Release()
		type item struct {
			Object    string `json:"object"`
			Embedding string `json:"embedding"`
			Index     int    `json:"index"`
		}
		out := struct {
			Object string               `json:"object"`
			Data   []item               `json:"data"`
			Model  string               `json:"model"`
			Usage  model.EmbeddingUsage `json:"usage"`
		}{Object: "list", Model: req.Model, Usage: model.EmbeddingUsage{PromptTokens: raw.Tokens, TotalTokens: raw.Tokens}}
		buf := make([]byte, raw.Dim*4)
		for i := 0; i < raw.N; i++ {
			for j, f := range raw.Vec(i) {
				binary.LittleEndian.PutUint32(buf[j*4:], math.Float32bits(f))
			}
			out.Data = append(out.Data, item{"embedding", base64.StdEncoding.EncodeToString(buf), i})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&out)
	})
	log.Printf("embedworker listening on %s", addr)
	log.Fatal((&http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}).ListenAndServe())
}
