package model

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
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
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
