package cron

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

// evalToolSet is shared by the agentic cases.
func evalToolSet() []model.Tool {
	return []model.Tool{
		{Type: "function", Function: &model.ToolFunction{
			Name:        "get_weather",
			Description: "Get current weather for a city.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]any{"type": "string", "description": "City name"},
					"unit": map[string]any{"type": "string", "enum": []string{"celsius", "fahrenheit"}},
				},
				"required": []string{"city"},
			},
		}},
		{Type: "function", Function: &model.ToolFunction{
			Name:        "get_time",
			Description: "Get the current time for an IANA timezone.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{"type": "string", "description": "IANA timezone, e.g. Europe/Paris"},
				},
				"required": []string{"timezone"},
			},
		}},
	}
}

type evalCase struct {
	ID       string
	Suite    model.EvalSuite
	System   string
	Prompt   string
	Messages []model.ChatMessage // full history override (tool-recovery case)
	Tools    []model.Tool
	MaxTok   int

	// grade returns 0..1. content is the assistant text (streaming cases);
	// toolCalls is populated for tool-calling (non-streaming) cases.
	grade func(content string, toolCalls []model.ToolCall) float64
}

func evalCases() []evalCase {
	cases := []evalCase{}

	// ---- Coding: deterministic single-answer questions --------------------
	cases = append(cases,
		evalCase{
			ID:    "closure-output",
			Suite: model.EvalSuiteCoding,
			Prompt: "What does this JavaScript program print to the console?\n" +
				"```js\nconst fns = [];\nfor (var i = 0; i < 3; i++) { fns.push(() => i * 2); }\nconsole.log(fns[0]());\n```\n" +
				`Reply with ONLY a JSON object: {"answer": <number>}`,
			MaxTok: 512,
			grade:  jsonNumberAnswer(6),
		},
		evalCase{
			ID:    "big-o-complexity",
			Suite: model.EvalSuiteCoding,
			Prompt: "What is the worst-case time complexity of this function in Big-O notation?\n" +
				"```js\nfunction f(a) {\n  let s = 0;\n  for (const x of a) { s += x; }\n  a.sort((p, q) => p - q);\n  return s;\n}\n```\n" +
				`Reply with ONLY a JSON object: {"answer": "O(...)"}`,
			MaxTok: 512,
			grade:  normalizedAnswer([]string{"o(nlogn)", "nlogn"}),
		},
		evalCase{
			ID:    "debug-line-number",
			Suite: model.EvalSuiteCoding,
			Prompt: "This function should return the sum 1..n but has exactly one bug. On which line number (1-indexed, counting from the first line shown) is the bug?\n" +
				"```\n1 function sumTo(n) {\n2   let total = 0;\n3   for (let i = 1; i <= n; i++);\n4     total += i;\n5   return total;\n6 }\n```\n" +
				`Reply with ONLY a JSON object: {"answer": <line number>}`,
			MaxTok: 512,
			grade:  jsonNumberAnswer(3),
		},
		evalCase{
			ID:    "sql-boundary-count",
			Suite: model.EvalSuiteCoding,
			Prompt: "Table users has one row per person with an age column. Rows have ages [25, 31, 42, 30, 18, 33]. " +
				"How many rows does `SELECT COUNT(*) FROM users WHERE age > 30;` return?\n" +
				`Reply with ONLY a JSON object: {"answer": <number>}`,
			MaxTok: 512,
			grade:  jsonNumberAnswer(3),
		},
		evalCase{
			ID:    "mod-arithmetic",
			Suite: model.EvalSuiteCoding,
			Prompt: "Compute 7^100 mod 13.\n" +
				`Reply with ONLY a JSON object: {"answer": <number>}`,
			MaxTok: 512,
			grade:  jsonNumberAnswer(9),
		},
		evalCase{
			ID:    "regex-build",
			Suite: model.EvalSuiteCoding,
			Prompt: "Write one regular expression that matches each of the strings cat, car, cab as complete strings and nothing else. " +
				"Do not use anchors or lookaround.\n" +
				`Reply with ONLY a JSON object: {"answer": "<regex>"}`,
			MaxTok: 512,
			grade:  gradeRegex,
		},

		// ---- Agentic: tool use + instruction following + recovery ------------
		evalCase{
			ID:     "tool-single-call",
			Suite:  model.EvalSuiteAgentic,
			Prompt: "What's the weather in Tokyo right now?",
			Tools:  evalToolSet(),
			MaxTok: 512,
			grade: func(_ string, calls []model.ToolCall) float64 {
				return bestCallScore(calls, "get_weather", func(args map[string]any) bool {
					return strings.Contains(strings.ToLower(argString(args["city"])), "tokyo")
				})
			},
		},
		evalCase{
			ID:     "tool-two-calls",
			Suite:  model.EvalSuiteAgentic,
			Prompt: "I need two things: the weather in Paris, and the current time in timezone Europe/Paris.",
			Tools:  evalToolSet(),
			MaxTok: 512,
			grade: func(_ string, calls []model.ToolCall) float64 {
				weather := hasCall(calls, "get_weather", func(args map[string]any) bool {
					return strings.Contains(strings.ToLower(argString(args["city"])), "paris")
				})
				timeOK := hasCall(calls, "get_time", func(args map[string]any) bool {
					return strings.EqualFold(strings.TrimSpace(argString(args["timezone"])), "Europe/Paris")
				})
				return (boolScore(weather) + boolScore(timeOK)) / 2
			},
		},
		evalCase{
			ID:     "no-tool-needed",
			Suite:  model.EvalSuiteAgentic,
			Prompt: "Using no tools at all: what is 17 * 24? Reply with just the number.",
			Tools:  evalToolSet(),
			MaxTok: 512,
			grade: func(content string, calls []model.ToolCall) float64 {
				noCall := boolScore(len(calls) == 0)
				answer := boolScore(extractIntAnswer(content) == 408)
				return (noCall + answer) / 2
			},
		},
		evalCase{
			ID:    "format-instruction",
			Suite: model.EvalSuiteAgentic,
			Prompt: "Reply with EXACTLY three bullet points (lines starting with \"- \"). Each bullet must be five words or fewer. " +
				"The second bullet must contain the word latency. Topic: why edge inference matters.",
			MaxTok: 512,
			grade:  gradeBullets,
		},
		evalCase{
			ID:    "tool-error-recovery",
			Suite: model.EvalSuiteAgentic,
			Messages: []model.ChatMessage{
				{Role: "user", Content: "What's the weather in NYC right now?"},
				{Role: "assistant", ToolCalls: []model.ToolCall{{
					ID: "call_1", Type: "function",
					Function: model.ToolCallFunc{Name: "get_weather", Arguments: `{"city": "NYC"}`},
				}}},
				{Role: "tool", ToolCallID: "call_1", Content: "ERROR: unknown city 'NYC'. Valid example cities: New York, London, Tokyo."},
				{Role: "user", Content: "The lookup failed. Please fix it and get the weather."},
			},
			Tools:  evalToolSet(),
			MaxTok: 512,
			grade: func(_ string, calls []model.ToolCall) float64 {
				if len(calls) == 0 {
					return 0
				}
				for _, c := range calls {
					args := parseArgs(c.Function.Arguments)
					city := strings.ToLower(argString(args["city"]))
					if c.Function.Name == "get_weather" && city != "" && city != "nyc" {
						return 1
					}
				}
				return 0
			},
		},

		// ---- Creative: constraint-checked SVG ---------------------------------
		evalCase{
			ID:    "svg-basic-shapes",
			Suite: model.EvalSuiteSVG,
			Prompt: "Generate a complete standalone SVG with viewBox exactly \"0 0 200 200\" containing exactly one blue rectangle and one red circle. " +
				"Reply with ONLY the SVG markup.",
			MaxTok: 2048,
			grade: func(content string, _ []model.ToolCall) float64 {
				doc, ok := parseSVG(content)
				if !ok {
					return 0
				}
				checks := []bool{
					doc.has("rect", func(e svgElement) bool { return strings.Contains(e.attr("fill"), "blue") }),
					doc.has("circle", func(e svgElement) bool { return strings.Contains(e.attr("fill"), "red") }),
					strings.TrimSpace(doc.rootAttr("viewBox")) == "0 0 200 200",
				}
				return fraction(checks)
			},
		},
		evalCase{
			ID:    "svg-bar-chart",
			Suite: model.EvalSuiteSVG,
			Prompt: "Generate a complete standalone SVG bar chart with three bars labelled A, B, C whose values are A=80, B=45, C=20. " +
				"Bar heights must be proportional to the values. Reply with ONLY the SVG markup.",
			MaxTok: 2048,
			grade: func(content string, _ []model.ToolCall) float64 {
				doc, ok := parseSVG(content)
				if !ok {
					return 0
				}
				rects := doc.all("rect")
				if len(rects) < 3 {
					return 0
				}
				heights := make([]float64, 0, len(rects))
				for _, r := range rects {
					if h := parseFloatAttr(r.attr("height")); h > 0 {
						heights = append(heights, h)
					}
				}
				sort.Float64s(heights)
				if len(heights) < 3 {
					return 0
				}
				n := len(heights)
				big, mid, small := heights[n-1], heights[n-2], heights[n-3]
				ratioMid := mid / big     // expect ~0.5625 (45/80)
				ratioSmall := small / big // expect 0.25 (20/80)
				checks := []bool{
					within(ratioMid, 45.0/80.0, 0.15),
					within(ratioSmall, 0.25, 0.15),
					big > mid,
				}
				return fraction(checks)
			},
		},
		evalCase{
			ID:    "svg-wordmark",
			Suite: model.EvalSuiteSVG,
			Prompt: "Generate a complete standalone SVG logo that shows the exact text OPENPATHS using a <text> element. Use at most three distinct fill colors overall. " +
				"Reply with ONLY the SVG markup.",
			MaxTok: 2048,
			grade: func(content string, _ []model.ToolCall) float64 {
				doc, ok := parseSVG(content)
				if !ok {
					return 0
				}
				hasText := false
				for _, t := range doc.all("text") {
					if strings.Contains(strings.ToUpper(t.content), "OPENPATHS") {
						hasText = true
					}
				}
				fills := map[string]bool{}
				for _, el := range doc.elements {
					f := el.attr("fill")
					if f != "" {
						fills[strings.ToLower(f)] = true
					}
				}
				checks := []bool{hasText, len(fills) >= 1, len(fills) <= 3}
				return fraction(checks)
			},
		},
		evalCase{
			ID:    "svg-mountain-scene",
			Suite: model.EvalSuiteSVG,
			Prompt: "Generate a complete standalone SVG of a mountain scene containing at least three triangle mountains (use <polygon> or <path>) and one circle sun, with at least two different fill colors. " +
				"Reply with ONLY the SVG markup.",
			MaxTok: 2048,
			grade: func(content string, _ []model.ToolCall) float64 {
				doc, ok := parseSVG(content)
				if !ok {
					return 0
				}
				mountains := len(doc.all("polygon"))
				for _, p := range doc.all("path") {
					if strings.Contains(p.attr("d"), "L") || strings.Contains(p.attr("d"), "l") {
						mountains++
					}
				}
				checks := []bool{
					mountains >= 3,
					len(doc.all("circle")) >= 1,
					countDistinctFills(doc) >= 2,
				}
				return fraction(checks)
			},
		},
	)
	return cases
}

// ---- grading helpers -------------------------------------------------------

func jsonNumberAnswer(want int) func(string, []model.ToolCall) float64 {
	return func(content string, _ []model.ToolCall) float64 {
		obj := extractJSONObject(content)
		if obj == nil {
			return 0
		}
		return boolScore(extractIntAnswer(fmt.Sprint(obj["answer"])) == want)
	}
}

func normalizedAnswer(accepted []string) func(string, []model.ToolCall) float64 {
	return func(content string, _ []model.ToolCall) float64 {
		got := normalizeForMatch(content)
		if obj := extractJSONObject(content); obj != nil {
			got = normalizeForMatch(fmt.Sprint(obj["answer"]))
		}
		for _, want := range accepted {
			if got == want || strings.Contains(got, want) {
				return 1
			}
		}
		return 0
	}
}

func gradeRegex(content string, _ []model.ToolCall) float64 {
	pattern := ""
	if obj := extractJSONObject(content); obj != nil {
		pattern = fmt.Sprint(obj["answer"])
	}
	pattern = strings.TrimSpace(strings.Trim(pattern, "`\"'"))
	if pattern == "" {
		return 0
	}
	reFull, err := regexp.Compile("^(?:" + pattern + ")$")
	if err != nil {
		return 0
	}
	for _, s := range []string{"cat", "car", "cab"} {
		if !reFull.MatchString(s) {
			return 0
		}
	}
	for _, s := range []string{"cap", "ca", "cart", "dog", "cats"} {
		if reFull.MatchString(s) {
			return 0
		}
	}
	return 1
}

func gradeBullets(content string, _ []model.ToolCall) float64 {
	var bullets []string
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		t = strings.TrimPrefix(t, "* ")
		t = strings.TrimPrefix(t, "- ")
		t = strings.TrimPrefix(t, "•")
		if t != strings.TrimSpace(line) || strings.HasPrefix(line, "•") {
			bullets = append(bullets, strings.TrimSpace(t))
		}
	}
	exactCount := boolScore(len(bullets) == 3)
	wordLimit := true
	if len(bullets) == 3 {
		for _, b := range bullets {
			if len(strings.Fields(b)) > 5 {
				wordLimit = false
			}
		}
	} else {
		wordLimit = false
	}
	hasLatency := false
	if len(bullets) >= 2 {
		hasLatency = strings.Contains(strings.ToLower(bullets[1]), "latency")
	}
	return (exactCount + boolScore(wordLimit) + boolScore(hasLatency)) / 3
}

func bestCallScore(calls []model.ToolCall, fn string, argsOK func(map[string]any) bool) float64 {
	if len(calls) == 0 {
		return 0
	}
	best := 0.25 // called *something*
	for _, c := range calls {
		switch {
		case c.Function.Name == fn && argsOK(parseArgs(c.Function.Arguments)):
			return 1
		case c.Function.Name == fn:
			best = math.Max(best, 0.5)
		}
	}
	return best
}

func hasCall(calls []model.ToolCall, fn string, argsOK func(map[string]any) bool) bool {
	for _, c := range calls {
		if c.Function.Name == fn && argsOK(parseArgs(c.Function.Arguments)) {
			return true
		}
	}
	return false
}

// ---- parsing helpers -------------------------------------------------------

// extractJSONObject pulls the first balanced {...} blob from a response,
// preferring fenced ```json blocks.
func extractJSONObject(s string) map[string]any {
	if m := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```").FindStringSubmatch(s); m != nil {
		var out map[string]any
		if json.Unmarshal([]byte(m[1]), &out) == nil {
			return out
		}
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return nil
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		switch {
		case c == '\\' && inStr:
			esc = true
		case c == '"':
			inStr = !inStr
		case inStr:
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				var out map[string]any
				if json.Unmarshal([]byte(s[start:i+1]), &out) == nil {
					return out
				}
				return nil
			}
		}
	}
	return nil
}

// extractIntAnswer finds the first integer in a short answer.
func extractIntAnswer(s string) int {
	m := regexp.MustCompile(`-?\d[\d,]*`).FindString(s)
	if m == "" {
		return math.MinInt
	}
	m = strings.ReplaceAll(m, ",", "")
	n, err := strconv.Atoi(m)
	if err != nil {
		return math.MinInt
	}
	return n
}

func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\'' || r == '`' || r == '.' {
			return -1
		}
		return r
	}, s)
	return s
}

func parseArgs(raw string) map[string]any {
	var out map[string]any
	if json.Unmarshal([]byte(raw), &out) == nil {
		return out
	}
	return map[string]any{}
}

func argString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func boolScore(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func fraction(checks []bool) float64 {
	total := float64(len(checks))
	if total == 0 {
		return 0
	}
	sum := 0.0
	for _, c := range checks {
		sum += boolScore(c)
	}
	return sum / total
}

func within(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol*math.Max(1, want)
}

// ---- SVG parsing -----------------------------------------------------------

type svgElement struct {
	tag      string
	attrs    []xml.Attr
	content  string
	children []*svgElement
}

func (e svgElement) attr(name string) string {
	for _, a := range e.attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

type svgDoc struct {
	root     *svgElement
	elements []*svgElement // flattened
}

func (d svgDoc) rootAttr(name string) string {
	if d.root == nil {
		return ""
	}
	return d.root.attr(name)
}

func (d svgDoc) all(tag string) []svgElement {
	var out []svgElement
	for _, el := range d.elements {
		if el.tag == tag {
			out = append(out, *el)
		}
	}
	return out
}

func (d svgDoc) has(tag string, pred func(svgElement) bool) bool {
	for _, el := range d.all(tag) {
		if pred(el) {
			return true
		}
	}
	return false
}

func parseSVG(s string) (svgDoc, bool) {
	start := strings.Index(s, "<svg")
	if start < 0 {
		return svgDoc{}, false
	}
	end := strings.LastIndex(s, "</svg>")
	if end < 0 {
		end = len(s)
	}
	frag := s[start:]
	if end+6 <= len(s) {
		frag = s[start : end+6]
	}
	root := &svgElement{}
	if err := xml.Unmarshal([]byte(frag), root); err != nil {
		return svgDoc{}, false
	}
	doc := svgDoc{root: root}
	var collect func(el *svgElement)
	collect = func(el *svgElement) {
		doc.elements = append(doc.elements, el)
		for _, c := range el.children {
			collect(c)
		}
	}
	collect(root)
	return doc, true
}

func (e *svgElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	e.tag = start.Name.Local
	e.attrs = start.Attr
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child := &svgElement{}
			if err := child.UnmarshalXML(d, t); err != nil {
				return err
			}
			e.children = append(e.children, child)
		case xml.CharData:
			e.content += string(t)
		case xml.EndElement:
			return nil
		}
	}
}

func countDistinctFills(d svgDoc) int {
	fills := map[string]bool{}
	for _, el := range d.elements {
		if f := el.attr("fill"); f != "" {
			fills[strings.ToLower(f)] = true
		}
	}
	return len(fills)
}

func parseFloatAttr(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}
