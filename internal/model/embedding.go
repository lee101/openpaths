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
	return d.AppendJSON(make([]byte, 0, 64+len(d.Embedding)*12+len(d.Base64))), nil
}

func (d EmbeddingData) AppendJSON(b []byte) []byte {
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
	return append(b, '}')
}

// AppendJSON serializes the response without the reflection and re-compaction
// that json.Marshal applies around custom marshalers.
func (r *EmbeddingResponse) AppendJSON(b []byte) []byte {
	b = append(b, `{"object":`...)
	b = strconv.AppendQuote(b, r.Object)
	b = append(b, `,"data":[`...)
	for i := range r.Data {
		if i > 0 {
			b = append(b, ',')
		}
		b = r.Data[i].AppendJSON(b)
	}
	b = append(b, `],"model":`...)
	b = strconv.AppendQuote(b, r.Model)
	b = append(b, `,"usage":{"prompt_tokens":`...)
	b = strconv.AppendInt(b, int64(r.Usage.PromptTokens), 10)
	b = append(b, `,"total_tokens":`...)
	b = strconv.AppendInt(b, int64(r.Usage.TotalTokens), 10)
	return append(b, `}}`...)
}
