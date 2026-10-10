package gobed

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func testProvider(t testing.TB) *GobedProvider {
	if d := os.Getenv("GOBED_MODEL_DIR"); d != "" {
		if err := os.Chdir(d); err != nil {
			t.Skip(err)
		}
	}
	p, err := New()
	if err != nil {
		t.Skipf("model unavailable: %v", err)
	}
	return p
}

func randASCII(r *rand.Rand) string {
	words := []string{"the", "Quick", "brown", "fox", "unaffable", "embedding", "xyzzyqq", "COVID-19", "it's", "snake_case", "3.14159", "tokenization", "zzzzzzzzzzzz", "a", "I", "naïve"}
	var sb strings.Builder
	n := r.Intn(80)
	for i := 0; i < n; i++ {
		switch r.Intn(6) {
		case 0:
			sb.WriteByte(byte(33 + r.Intn(94)))
		case 1:
			sb.WriteString(words[r.Intn(len(words))])
		case 2:
			b := make([]byte, 1+r.Intn(12))
			for j := range b {
				b[j] = byte('a' + r.Intn(26))
			}
			sb.Write(b)
		default:
			sb.WriteString(words[r.Intn(len(words)-1)])
		}
		sb.WriteString([]string{" ", "  ", "\n", "\t", ""}[r.Intn(5)])
	}
	return sb.String()
}

func TestTokenizeIntoMatchesSimpleTokenize(t *testing.T) {
	p := testProvider(t)
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 5000; i++ {
		s := randASCII(r)
		want := p.model.SimpleTokenize(s)
		got := p.model.TokenizeInto(s, nil, 0)
		if !slices.Equal(want, got) {
			t.Fatalf("mismatch for %q", s)
		}
		lim := p.model.TokenizeInto(s, nil, gobedChunkSize)
		w := want
		if len(w) > gobedChunkSize {
			w = w[:gobedChunkSize]
		}
		if !slices.Equal(w, lim) {
			t.Fatalf("limit mismatch for %q", s)
		}
	}
}

func TestEmbedParity(t *testing.T) {
	p := testProvider(t)
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 500; i++ {
		s := randASCII(r)
		if i%50 == 0 {
			s = strings.Repeat(s+" ", 20)
		}
		for _, mode := range []string{"truncate", "average_chunks"} {
			resp, err := p.Embed(context.Background(), &model.EmbeddingRequest{Input: s, LongTextMode: mode})
			if err != nil {
				t.Fatal(err)
			}
			toks := p.model.SimpleTokenize(s)
			if mode == "truncate" && len(toks) > gobedChunkSize {
				toks = toks[:gobedChunkSize]
			}
			ref, _ := p.model.EmbedTokens(toks)
			if resp.Usage.TotalTokens != len(toks) {
				t.Fatalf("tokens %d != %d", resp.Usage.TotalTokens, len(toks))
			}
			for j, v := range resp.Data[0].Embedding {
				if math.Abs(v-float64(ref[j])) > 1e-5 {
					t.Fatalf("emb[%d] %v != %v mode=%s", j, v, ref[j], mode)
				}
			}
		}
	}
}

func TestDimensionsAndBase64(t *testing.T) {
	p := testProvider(t)
	full, _ := p.Embed(context.Background(), &model.EmbeddingRequest{Input: "hello world", Dimensions: 128})
	if len(full.Data[0].Embedding) != 128 {
		t.Fatal("dims")
	}
	var n float64
	for _, v := range full.Data[0].Embedding {
		n += v * v
	}
	if math.Abs(n-1) > 1e-4 {
		t.Fatalf("norm %v", n)
	}
	b, _ := p.Embed(context.Background(), &model.EmbeddingRequest{Input: []any{"a", "b"}, EncodingFormat: "base64"})
	raw, err := base64.StdEncoding.DecodeString(b.Data[1].Base64)
	if err != nil || len(raw) != 512*4 {
		t.Fatal("b64", err, len(raw))
	}
	_ = binary.LittleEndian.Uint32(raw)
	js, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	var back model.EmbeddingResponse
	if err := json.Unmarshal(js, &back); err != nil || len(back.Data[0].Embedding) != 128 {
		t.Fatal("roundtrip", err)
	}
	for i, v := range back.Data[0].Embedding {
		if float32(v) != float32(full.Data[0].Embedding[i]) {
			t.Fatal("precision")
		}
	}
	jb, _ := json.Marshal(b)
	if !strings.Contains(string(jb), `"embedding":"`) {
		t.Fatal("b64 json")
	}
}
