package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecisionRequestValidate(t *testing.T) {
	ok := `{"model":"decision-model-preview","state":{"ticket":"refund please"},"questions":{
		"refund":{"type":"noul","instructions":"Is this a refund request?","criteria":{"true":"asks for money back","false":"anything else"}},
		"queue":{"type":"choice","instructions":"Route it","criteria":{"billing":"payments","tech":null}},
		"urgency":{"type":"score","instructions":"How urgent?","criteria":["low","medium","high"]}}}`
	var req DecisionRequest
	if err := json.Unmarshal([]byte(ok), &req); err != nil {
		t.Fatal(err)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	for body, wantField := range map[string]string{
		`{"model":"m","questions":{"a":{"type":"noul","instructions":"x"}}}`:                                   "state",
		`{"model":"m","state":"s","questions":{}}`:                                                             "questions",
		`{"model":"m","state":"s","questions":{"a":{"type":"noul"}}}`:                                          "questions.a.instructions",
		`{"model":"m","state":"s","questions":{"a":{"type":"maybe","instructions":"x"}}}`:                      "questions.a.type",
		`{"model":"m","state":"s","questions":{"a":{"type":"choice","instructions":"x","criteria":{"y":"1"}}}}`: "questions.a.criteria",
		`{"model":"m","state":"s","questions":{"a":{"type":"score","instructions":"x","criteria":["one"]}}}`:    "questions.a.criteria",
		`{"model":"m","state":"s","questions":{"a":{"type":"noul","instructions":"x","criteria":{"y":"1"}}}}`:   "questions.a.criteria",
	} {
		var r DecisionRequest
		if err := json.Unmarshal([]byte(body), &r); err != nil {
			t.Fatal(err)
		}
		err := r.Validate()
		if err == nil || !strings.HasPrefix(err.Error(), wantField) {
			t.Errorf("%s: err %v, want field %s", body, err, wantField)
		}
	}
}
