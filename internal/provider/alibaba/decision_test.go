package alibaba

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

func TestDecideForwardsSystemOneSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/systemone" || r.Header.Get("X-DashScope-Async") != "" {
			t.Errorf("path %s async %q", r.URL.Path, r.Header.Get("X-DashScope-Async"))
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]json.RawMessage
		_ = json.Unmarshal(body, &got)
		if string(got["model"]) != `"decision-model-preview"` || string(got["state"]) != `"hello"` || !strings.Contains(string(got["questions"]), `"noul"`) {
			t.Errorf("body %s", body)
		}
		io.WriteString(w, `{"model":"decision-model-preview","request_id":"r","answers":{"q":{"type":"noul","noul":0.9}},"usage":{"input_tokens":12},"latency_ms":300}`)
	}))
	defer srv.Close()
	req := &model.DecisionRequest{Model: "decision-model-preview", State: json.RawMessage(`"hello"`), Questions: map[string]json.RawMessage{"q": json.RawMessage(`{"type":"noul","instructions":"yes?"}`)}}
	resp, err := New("key", srv.URL).Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.InputTokens != 12 || !strings.Contains(string(resp.Answers), `"noul":0.9`) {
		t.Fatalf("resp %+v", resp)
	}
}

func TestDecideMapsValidationErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"code":"InvalidParameter","message":"questions.q.criteria is required"}`)
	}))
	defer srv.Close()
	_, err := New("key", srv.URL).Decide(context.Background(), &model.DecisionRequest{Model: "m", State: json.RawMessage(`"s"`)})
	pe, ok := err.(*baseprovider.ProviderError)
	if !ok || pe.StatusCode != 422 || pe.Retryable || !strings.Contains(pe.Message, "criteria") {
		t.Fatalf("err %#v", err)
	}
}
