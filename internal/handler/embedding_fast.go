package handler

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"math"
	"strconv"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

// rawEmbedder is implemented by providers that return a contiguous float32
// batch so the handler can serialize without float64/reflection overhead.
type rawEmbedder interface {
	EmbedRaw(ctx context.Context, req *model.EmbeddingRequest) (*model.RawEmbeddings, error)
}

func validEncodingFormat(f string) bool {
	return f == "" || f == "float" || f == "base64" || f == "int8"
}

// appendEmbeddingJSON writes an OpenAI-compatible list response. float is the
// default; base64 is float32 little-endian; int8 is base64 of signed bytes with
// a per-vector "scale" (value ~= int8 * scale).
func appendEmbeddingJSON(b []byte, raw *model.RawEmbeddings, modelName, format string) []byte {
	b = append(b, `{"object":"list","data":[`...)
	var f32 []byte
	var q []byte
	for i := 0; i < raw.N; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		v := raw.Vec(i)
		b = append(b, `{"object":"embedding","embedding":`...)
		switch format {
		case "base64":
			f32 = f32[:0]
			for _, x := range v {
				f32 = binary.LittleEndian.AppendUint32(f32, math.Float32bits(x))
			}
			b = append(b, '"')
			b = base64.StdEncoding.AppendEncode(b, f32)
			b = append(b, '"')
		case "int8":
			var mx float32
			for _, x := range v {
				if a := float32(math.Abs(float64(x))); a > mx {
					mx = a
				}
			}
			q = q[:0]
			var inv float32
			if mx > 0 {
				inv = 127 / mx
			}
			for _, x := range v {
				q = append(q, byte(int8(math.RoundToEven(float64(x*inv)))))
			}
			b = append(b, '"')
			b = base64.StdEncoding.AppendEncode(b, q)
			b = append(b, `","scale":`...)
			b = strconv.AppendFloat(b, float64(mx/127), 'g', -1, 32)
			b = append(b, `,"index":`...)
			b = strconv.AppendInt(b, int64(i), 10)
			b = append(b, '}')
			continue
		default:
			b = append(b, '[')
			for j, x := range v {
				if j > 0 {
					b = append(b, ',')
				}
				b = strconv.AppendFloat(b, float64(x), 'g', -1, 32)
			}
			b = append(b, ']')
		}
		b = append(b, `,"index":`...)
		b = strconv.AppendInt(b, int64(i), 10)
		b = append(b, '}')
	}
	b = append(b, `],"model":`...)
	b = strconv.AppendQuote(b, modelName)
	b = append(b, `,"usage":{"prompt_tokens":`...)
	b = strconv.AppendInt(b, int64(raw.Tokens), 10)
	b = append(b, `,"total_tokens":`...)
	b = strconv.AppendInt(b, int64(raw.Tokens), 10)
	b = append(b, `}}`...)
	return b
}

func writeRawEmbeddings(ctx *fasthttp.RequestCtx, raw *model.RawEmbeddings, modelName, format string) {
	per := raw.Dim*12 + 48
	switch format {
	case "base64":
		per = raw.Dim*16/3 + 96
	case "int8":
		per = raw.Dim/3*4 + 80
	}
	body := appendEmbeddingJSON(make([]byte, 0, raw.N*per+128), raw, modelName, format)
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json")
	ctx.Response.SetBodyRaw(body)
}
