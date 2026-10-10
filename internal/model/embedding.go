package model

import (
	"math"
	"strconv"
)

type EmbeddingRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     int    `json:"dimensions,omitempty"`
	LongTextMode   string `json:"long_text_mode,omitempty"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`

	// F32 marks Embedding as float32-origin so it is serialized with float32 precision.
	F32 bool `json:"-"`
	// Base64 holds little-endian float32 bytes for encoding_format=base64.
	Base64 string `json:"-"`
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (d EmbeddingData) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 64+len(d.Embedding)*12+len(d.Base64))
	b = append(b, `{"object":`...)
	b = strconv.AppendQuote(b, d.Object)
	b = append(b, `,"embedding":`...)
	switch {
	case d.Base64 != "":
		b = append(b, '"')
		b = append(b, d.Base64...)
		b = append(b, '"')
	case len(d.Embedding) == 0:
		b = append(b, "[]"...)
	default:
		bits := 64
		if d.F32 {
			bits = 32
		}
		b = append(b, '[')
		for i, v := range d.Embedding {
			if i > 0 {
				b = append(b, ',')
			}
			if math.IsNaN(v) || math.IsInf(v, 0) {
				v = 0
			}
			b = strconv.AppendFloat(b, v, 'g', -1, bits)
		}
		b = append(b, ']')
	}
	b = append(b, `,"index":`...)
	b = strconv.AppendInt(b, int64(d.Index), 10)
	return append(b, '}'), nil
}

// RawEmbeddings is a contiguous float32 batch (N*Dim) produced by providers
// that can skip per-vector allocation and float64 conversion. Call Release
// once the response has been written.
type RawEmbeddings struct {
	N       int
	Dim     int
	Vecs    []float32
	Tokens  int
	release func()
}

func NewRawEmbeddings(n, dim int, vecs []float32, release func()) *RawEmbeddings {
	return &RawEmbeddings{N: n, Dim: dim, Vecs: vecs, release: release}
}

func (r *RawEmbeddings) Vec(i int) []float32 { return r.Vecs[i*r.Dim : (i+1)*r.Dim] }

func (r *RawEmbeddings) Release() {
	if r.release != nil {
		r.release()
		r.release = nil
	}
	r.Vecs = nil
}
