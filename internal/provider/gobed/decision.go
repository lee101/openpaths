package gobed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/lee101/gobed/pkg/ann/simd"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	decisionChunk     = 512
	decisionMaxChunks = 8
	decisionMaxTokens = decisionChunk * decisionMaxChunks
	decisionMaxBytes  = 64 << 10
	// softmax temperature over cosine similarities
	decisionTemp = 0.05
	// noul without criteria: sigmoid((cos-center)/scale)
	noulCenter = 0.20
	noulScale  = 0.05
)

type decisionQuestion struct {
	Type         string          `json:"type"`
	Instructions json.RawMessage `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria"`
}

// Decide answers TypeSafe /v1/systemone questions by cosine similarity between the
// state and each option, softmaxed into probabilities. The state is embedded as the
// mean of up to 8 x 512-token chunk embeddings (4096-token context).
func (p *GobedProvider) Decide(ctx context.Context, req *model.DecisionRequest) (*model.DecisionResponse, error) {
	sc := scratchPool.Get().(*scratch)
	defer scratchPool.Put(sc)
	state, tokens := p.embedLong(sc, jsonText(req.State))
	if tokens == 0 {
		return nil, &provider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: "state has no embeddable text"}
	}
	keys := make([]string, 0, len(req.Questions))
	for k := range req.Questions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	answers := make(map[string]any, len(keys))
	for _, key := range keys {
		var q decisionQuestion
		if err := json.Unmarshal(req.Questions[key], &q); err != nil {
			return nil, &provider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: fmt.Sprintf("questions.%s must be an object", key)}
		}
		ans, n, err := p.answer(sc, state, q)
		if err != nil {
			return nil, &provider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: fmt.Sprintf("questions.%s: %v", key, err)}
		}
		tokens += n
		answers[key] = ans
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return nil, err
	}
	return &model.DecisionResponse{Model: req.Model, Answers: raw, Usage: model.DecisionUsage{InputTokens: tokens}}, nil
}

func (p *GobedProvider) answer(sc *scratch, state []float32, q decisionQuestion) (map[string]any, int, error) {
	instr := jsonText(q.Instructions)
	switch q.Type {
	case "noul":
		var crit map[string]json.RawMessage
		_ = json.Unmarshal(q.Criteria, &crit)
		yes, no := jsonText(crit["true"]), jsonText(crit["false"])
		// static embeddings ignore negation, so a "not X" false side carries no signal
		if no == "" || negated(no) {
			v, n := p.embedLong(sc, strings.TrimSpace(instr+" "+yes))
			prob := 1 / (1 + math.Exp(-(cosine(state, v)-noulCenter)/noulScale))
			return map[string]any{"type": "noul", "noul": round2(prob)}, n, nil
		}
		if yes == "" {
			yes = instr
		}
		probs, n := p.optionProbs(sc, state, []string{yes, no})
		return map[string]any{"type": "noul", "noul": round2(probs[0])}, n, nil
	case "choice":
		var crit map[string]json.RawMessage
		if err := json.Unmarshal(q.Criteria, &crit); err != nil || len(crit) < 2 {
			return nil, 0, fmt.Errorf("criteria must map at least two options to descriptions")
		}
		opts := make([]string, 0, len(crit))
		for k := range crit {
			opts = append(opts, k)
		}
		sort.Strings(opts)
		texts := make([]string, len(opts))
		for i, k := range opts {
			texts[i] = optionText(k, jsonText(crit[k]))
		}
		probs, n := p.optionProbs(sc, state, texts)
		out := make(map[string]float64, len(opts))
		best := 0
		for i, k := range opts {
			out[k] = round2(probs[i])
			if probs[i] > probs[best] {
				best = i
			}
		}
		return map[string]any{"type": "choice", "choice": opts[best], "probabilities": out, "confidence": round2(confidence(probs))}, n, nil
	case "score":
		var crit []json.RawMessage
		if err := json.Unmarshal(q.Criteria, &crit); err != nil || len(crit) < 2 {
			return nil, 0, fmt.Errorf("criteria must be an array of 2 to 10 levels")
		}
		texts := make([]string, len(crit))
		legend := make(map[string]string, len(crit))
		for i, c := range crit {
			texts[i] = jsonText(c)
			legend[fmt.Sprint(i)] = texts[i]
		}
		probs, n := p.optionProbs(sc, state, texts)
		out := make(map[string]float64, len(crit))
		score := 0.0
		for i, pr := range probs {
			out[fmt.Sprint(i)] = round2(pr)
			score += float64(i) * pr
		}
		return map[string]any{"type": "score", "score": round2(score), "legend": legend, "probabilities": out, "confidence": round2(confidence(probs))}, n, nil
	}
	return nil, 0, fmt.Errorf("type must be noul, choice or score")
}

func (p *GobedProvider) optionProbs(sc *scratch, state []float32, texts []string) ([]float64, int) {
	logits := make([]float64, len(texts))
	tokens := 0
	for i, t := range texts {
		v, n := p.embedLong(sc, t)
		tokens += n
		logits[i] = cosine(state, v) / decisionTemp
	}
	return softmax(logits), tokens
}

// embedLong returns a unit vector: the mean of up to 8 chunk embeddings of 512 tokens.
func (p *GobedProvider) embedLong(sc *scratch, text string) ([]float32, int) {
	out := make([]float32, dim)
	if strings.TrimSpace(text) == "" {
		return out, 0
	}
	if len(text) > decisionMaxBytes {
		text = strings.ToValidUTF8(text[:decisionMaxBytes], "")
	}
	toks := p.model.TokenizeInto(text, sc.tokens[:0], decisionMaxTokens)
	acc := sc.acc[:]
	chunks := 0
	for lo := 0; lo < len(toks); lo += decisionChunk {
		hi := min(lo+decisionChunk, len(toks))
		valid := p.model.EmbedTokensIntoSIMD(toks[lo:hi], acc)
		if valid == 0 {
			continue
		}
		normalize(acc)
		for i, x := range acc {
			out[i] += x
		}
		chunks++
	}
	if chunks > 0 {
		normalize(out)
	}
	return out, len(toks)
}

func normalize(v []float32) {
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	if n > 0 {
		simd.ScaleF32(v, float32(1/math.Sqrt(n)))
	}
}

func cosine(a, b []float32) float64 {
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

func softmax(logits []float64) []float64 {
	mx := math.Inf(-1)
	for _, l := range logits {
		mx = math.Max(mx, l)
	}
	out := make([]float64, len(logits))
	var sum float64
	for i, l := range logits {
		out[i] = math.Exp(l - mx)
		sum += out[i]
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

// confidence is 1 minus normalized entropy.
func confidence(probs []float64) float64 {
	if len(probs) < 2 {
		return 1
	}
	var h float64
	for _, p := range probs {
		if p > 0 {
			h -= p * math.Log(p)
		}
	}
	return 1 - h/math.Log(float64(len(probs)))
}

var negations = map[string]bool{"no": true, "not": true, "none": true, "never": true, "without": true, "nothing": true, "neither": true, "nor": true, "cannot": true}

func negated(text string) bool {
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r == '\'')
	}) {
		if negations[w] || strings.HasSuffix(w, "n't") {
			return true
		}
	}
	return false
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func optionText(key, desc string) string {
	k := strings.NewReplacer("_", " ", "-", " ").Replace(key)
	if desc == "" {
		return k
	}
	return k + ": " + desc
}

// jsonText flattens a JSON value into plain text: strings verbatim, objects as "key: value" lines.
func jsonText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	var b strings.Builder
	flatten(&b, "", v)
	return strings.TrimSpace(b.String())
}

func flatten(b *strings.Builder, prefix string, v any) {
	switch t := v.(type) {
	case nil:
	case string:
		if prefix != "" {
			b.WriteString(prefix + ": ")
		}
		b.WriteString(t + "\n")
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			flatten(b, strings.TrimSpace(prefix+" "+k), t[k])
		}
	case []any:
		for _, x := range t {
			flatten(b, prefix, x)
		}
	default:
		if prefix != "" {
			b.WriteString(prefix + ": ")
		}
		b.WriteString(fmt.Sprint(t) + "\n")
	}
}
