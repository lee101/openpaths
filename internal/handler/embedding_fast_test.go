package handler

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	gobedprov "github.com/openpaths/openpaths/internal/provider/gobed"
)

func sampleRaw() *model.RawEmbeddings {
	return &model.RawEmbeddings{N: 2, Dim: 4, Tokens: 7, Vecs: []float32{0.5, -0.25, 1e-7, 0, 1, -2, 0.125, 3}}
}

type wireResp struct {
	Object string `json:"object"`
	Data   []struct {
		Embedding json.RawMessage `json:"embedding"`
		Scale     float64         `json:"scale"`
		Index     int             `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		Prompt int `json:"prompt_tokens"`
		Total  int `json:"total_tokens"`
	} `json:"usage"`
}

func TestAppendEmbeddingJSONFloat(t *testing.T) {
	var r wireResp
	if err := json.Unmarshal(appendEmbeddingJSON(nil, sampleRaw(), `m"x`, ""), &r); err != nil {
		t.Fatal(err)
	}
	var v []float64
	json.Unmarshal(r.Data[1].Embedding, &v)
	if r.Model != `m"x` || r.Usage.Total != 7 || r.Data[1].Index != 1 || len(v) != 4 || v[3] != 3 {
		t.Fatalf("%+v %v", r, v)
	}
}

func TestAppendEmbeddingJSONBase64(t *testing.T) {
	var r wireResp
	json.Unmarshal(appendEmbeddingJSON(nil, sampleRaw(), "m", "base64"), &r)
	var s string
	json.Unmarshal(r.Data[0].Embedding, &s)
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != 16 {
		t.Fatalf("%v %d", err, len(b))
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(b[0:])); got != 0.5 {
		t.Fatalf("got %v", got)
	}
}

func TestAppendEmbeddingJSONInt8(t *testing.T) {
	var r wireResp
	if err := json.Unmarshal(appendEmbeddingJSON(nil, sampleRaw(), "m", "int8"), &r); err != nil {
		t.Fatal(err)
	}
	var s string
	json.Unmarshal(r.Data[1].Embedding, &s)
	b, _ := base64.StdEncoding.DecodeString(s)
	if len(b) != 4 || r.Data[1].Scale == 0 {
		t.Fatalf("len %d scale %v", len(b), r.Data[1].Scale)
	}
	for i, want := range []float32{1, -2, 0.125, 3} {
		got := float32(int8(b[i])) * float32(r.Data[1].Scale)
		if math.Abs(float64(got-want)) > 0.03 {
			t.Fatalf("dim %d got %v want %v", i, got, want)
		}
	}
	if r.Data[0].Index != 0 {
		t.Fatal("index")
	}
}

func TestValidEncodingFormat(t *testing.T) {
	if !validEncodingFormat("") || !validEncodingFormat("base64") || !validEncodingFormat("int8") || validEncodingFormat("x") {
		t.Fatal("validEncodingFormat")
	}
}

// HTTP-level: provider + serialization into a fasthttp response, per format.
func benchHTTP(b *testing.B, format string, n int) {
	_ = os.Chdir("../..")
	p, err := gobedprov.New()
	if err != nil {
		b.Skip(err)
	}
	in := make([]any, n)
	for i := range in {
		in[i] = "how do I reset the password for account number on the billing portal quickly"
	}
	b.ResetTimer()
	var bytes int64
	b.RunParallel(func(pb *testing.PB) {
		req := &model.EmbeddingRequest{Model: "m", Input: in, EncodingFormat: format}
		var ctx fasthttp.RequestCtx
		for pb.Next() {
			raw, err := p.EmbedRaw(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
			writeRawEmbeddings(&ctx, raw, "m", format)
			raw.Release()
			bytes += int64(len(ctx.Response.Body()))
		}
	})
	b.ReportMetric(float64(b.N*n)/b.Elapsed().Seconds(), "emb/s")
	b.ReportMetric(float64(bytes)/float64(b.N*n), "B/emb")
}

func BenchmarkHTTPFloat256(b *testing.B)  { benchHTTP(b, "", 256) }
func BenchmarkHTTPBase64256(b *testing.B) { benchHTTP(b, "base64", 256) }
func BenchmarkHTTPInt8256(b *testing.B)   { benchHTTP(b, "int8", 256) }

// Baseline: legacy Embed + encoding/json of float64 vectors.
func BenchmarkHTTPLegacyJSON256(b *testing.B) {
	_ = os.Chdir("../..")
	p, err := gobedprov.New()
	if err != nil {
		b.Skip(err)
	}
	in := make([]any, 256)
	for i := range in {
		in[i] = "how do I reset the password for account number on the billing portal quickly"
	}
	b.ResetTimer()
	var bytes int64
	b.RunParallel(func(pb *testing.PB) {
		req := &model.EmbeddingRequest{Model: "m", Input: in}
		for pb.Next() {
			resp, err := p.Embed(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
			d, _ := json.Marshal(resp)
			bytes += int64(len(d))
		}
	})
	b.ReportMetric(float64(b.N*256)/b.Elapsed().Seconds(), "emb/s")
	b.ReportMetric(float64(bytes)/float64(b.N*256), "B/emb")
}
