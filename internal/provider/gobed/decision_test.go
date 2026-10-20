package gobed

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type decisionCase struct {
	state, questions string
	want             map[string]string
}

var decisionCases = []decisionCase{
	{`"I was charged twice for my subscription this month, please refund the duplicate payment"`,
		`{"queue":{"type":"choice","instructions":"Which team should handle this ticket?","criteria":{"billing":"payments, invoices, refunds, charges","engineering":"bugs, crashes, errors, outages","sales":"pricing questions, demos, upgrades"}},
		  "refund":{"type":"noul","instructions":"Is the customer asking for a refund?","criteria":{"true":"customer asks for money back, a refund","false":"customer does not want money back"}},
		  "urgency":{"type":"score","instructions":"How urgent is it?","criteria":["low priority, no rush, whenever","medium, needs attention this week","high, urgent, critical, immediately"]}}`,
		map[string]string{"queue": "billing", "refund": "true"}},
	{`"The app crashes with a null pointer exception every time I open settings"`,
		`{"queue":{"type":"choice","instructions":"Which team?","criteria":{"billing":"payments, invoices, refunds, charges","engineering":"bugs, crashes, errors, outages","sales":"pricing questions, demos, upgrades"}}}`,
		map[string]string{"queue": "engineering"}},
	{`"Can I get a demo of the enterprise plan for my team of 200?"`,
		`{"queue":{"type":"choice","instructions":"Which team?","criteria":{"billing":"payments, invoices, refunds, charges","engineering":"bugs, crashes, errors, outages","sales":"pricing questions, demos, upgrades"}}}`,
		map[string]string{"queue": "sales"}},
	{`{"review":"Absolutely terrible food, cold and late, never ordering again","stars":1}`,
		`{"sentiment":{"type":"choice","instructions":"Sentiment","criteria":{"positive":"happy, great, love, excellent","negative":"bad, terrible, angry, awful","neutral":"okay, fine, average"}},
		  "satisfaction":{"type":"score","instructions":"Satisfaction","criteria":["very unhappy, terrible, awful","neutral, okay","very happy, delighted, excellent"]}}`,
		map[string]string{"sentiment": "negative", "satisfaction": "0"}},
	{`"URGENT: production database is down, all customers affected, fix immediately"`,
		`{"urgency":{"type":"score","instructions":"Urgency","criteria":["low priority, no rush, whenever","medium, needs attention this week","high, urgent, critical, immediately"]}}`,
		map[string]string{"urgency": "2"}},
}

func TestDecideGobed(t *testing.T) {
	p := testProvider(t)
	for _, c := range decisionCases {
		req := &model.DecisionRequest{Model: "gobed-decision", State: json.RawMessage(c.state)}
		if err := json.Unmarshal([]byte(c.questions), &req.Questions); err != nil {
			t.Fatal(err)
		}
		resp, err := p.Decide(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s -> %s tokens=%d", c.state[:min(40, len(c.state))], resp.Answers, resp.Usage.InputTokens)
		var answers map[string]struct {
			Choice string  `json:"choice"`
			Noul   float64 `json:"noul"`
			Score  float64 `json:"score"`
		}
		_ = json.Unmarshal(resp.Answers, &answers)
		for k, want := range c.want {
			a := answers[k]
			switch {
			case a.Choice != "":
				if a.Choice != want {
					t.Errorf("%s: choice %s want %s", k, a.Choice, want)
				}
			case want == "true" || want == "false":
				if (a.Noul >= 0.5) != (want == "true") {
					t.Errorf("%s: noul %v want %s", k, a.Noul, want)
				}
			default:
				lvl := map[string]float64{"0": 0, "1": 1, "2": 2}[want]
				if d := a.Score - lvl; d > 0.75 || d < -0.75 {
					t.Errorf("%s: score %v want ~%s", k, a.Score, want)
				}
			}
		}
	}
}

func TestDecideLongStateUsesEightChunks(t *testing.T) {
	p := testProvider(t)
	long := strings.Repeat("the server keeps crashing with errors and outages. ", 2000)
	req := &model.DecisionRequest{Model: "m", State: json.RawMessage(`"` + long + `"`), Questions: map[string]json.RawMessage{
		"q": json.RawMessage(`{"type":"choice","instructions":"team","criteria":{"billing":"refunds","engineering":"crashes"}}`),
	}}
	resp, err := p.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.InputTokens < decisionMaxTokens || resp.Usage.InputTokens > decisionMaxTokens+50 {
		t.Fatalf("tokens %d", resp.Usage.InputTokens)
	}
	if !strings.Contains(string(resp.Answers), `"choice":"engineering"`) {
		t.Fatalf("answers %s", resp.Answers)
	}
}

func TestJSONTextFlattens(t *testing.T) {
	got := jsonText(json.RawMessage(`{"user":{"name":"Ann","msgs":["hi","refund"]},"n":2}`))
	if got != "n: 2\nuser msgs: hi\nuser msgs: refund\nuser name: Ann" {
		t.Fatalf("%q", got)
	}
}

func TestSoftmaxConfidence(t *testing.T) {
	p := softmax([]float64{0, 0})
	if p[0] != 0.5 || confidence(p) != 0 {
		t.Fatal(p)
	}
	if c := confidence([]float64{1, 0}); c != 1 {
		t.Fatal(c)
	}
}

func TestDecideNoulNegatedFalseCriterion(t *testing.T) {
	p := testProvider(t)
	q := `{"type":"noul","instructions":"Is the customer asking for money back?","criteria":{"true":"asks for a refund, money back","false":"no refund requested"}}`
	for state, yes := range map[string]bool{
		`{"message":"I was charged twice for order #4412. Please refund the duplicate charge."}`: true,
		`{"message":"The app crashes when I open settings"}`:                                       false,
	} {
		resp, err := p.Decide(context.Background(), &model.DecisionRequest{Model: "m", State: json.RawMessage(state), Questions: map[string]json.RawMessage{"q": json.RawMessage(q)}})
		if err != nil {
			t.Fatal(err)
		}
		var a map[string]struct{ Noul float64 }
		_ = json.Unmarshal(resp.Answers, &a)
		if (a["q"].Noul >= 0.5) != yes {
			t.Errorf("%s -> %v", state, a["q"].Noul)
		}
	}
	if !negated("customer doesn't want money back") || !negated("no refund") || negated("a normal personal message") {
		t.Fatal("negated")
	}
}
