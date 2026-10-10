package gobed

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type RemoteProvider struct {
	url    string
	key    string
	client *http.Client
}

func NewRemote(baseURL, key string) *RemoteProvider {
	return &RemoteProvider{
		url:    strings.TrimRight(baseURL, "/") + "/v1/embeddings",
		key:    key,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *RemoteProvider) Name() string { return "gobed-overflow" }

func (r *RemoteProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	body := *req
	body.EncodingFormat = "base64"
	buf, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	hr.Header.Set("Content-Type", "application/json")
	if r.key != "" {
		hr.Header.Set("Authorization", "Bearer "+r.key)
	}
	resp, err := r.client.Do(hr)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "gobed-overflow", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "gobed-overflow", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &provider.ProviderError{Provider: "gobed-overflow", StatusCode: resp.StatusCode,
			Message: fmt.Sprintf("overflow backend: %s", truncate(string(raw), 200)), Retryable: resp.StatusCode >= 500}
	}
	var wire struct {
		Data []struct {
			Embedding string `json:"embedding"`
			Index     int    `json:"index"`
		} `json:"data"`
		Usage model.EmbeddingUsage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, &provider.ProviderError{Provider: "gobed-overflow", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	out := &model.EmbeddingResponse{Object: "list", Model: req.Model, Usage: wire.Usage}
	for _, d := range wire.Data {
		b, err := base64.StdEncoding.DecodeString(d.Embedding)
		if err != nil || len(b)%4 != 0 {
			return nil, &provider.ProviderError{Provider: "gobed-overflow", StatusCode: 502, Message: "bad base64 embedding", Retryable: true}
		}
		v := make([]float64, len(b)/4)
		for i := range v {
			v[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:])))
		}
		out.Data = append(out.Data, model.EmbeddingData{Object: "embedding", Embedding: v, Index: d.Index})
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
