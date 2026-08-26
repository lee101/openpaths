package cron

import (
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func tc(name, args string) model.ToolCall {
	return model.ToolCall{ID: "t", Type: "function", Function: model.ToolCallFunc{Name: name, Arguments: args}}
}

func findCase(t *testing.T, id string) evalCase {
	t.Helper()
	for _, c := range evalCases() {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("case %q not found", id)
	return evalCase{}
}

func TestJSONNumberAnswer(t *testing.T) {
	g := jsonNumberAnswer(9)
	cases := map[string]float64{
		`{"answer": 9}`: 1,
		"The result is ```json\n{\"answer\": 9}\n```": 1,
		`{"answer": "9"}`: 1,
		`{"answer": 10}`:  0,
		"9":               0, // no JSON object at all
		"":                0,
	}
	for in, want := range cases {
		if got := g(in, nil); got != want {
			t.Errorf("jsonNumberAnswer(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizedAnswer(t *testing.T) {
	g := normalizedAnswer([]string{"o(nlogn)", "nlogn"})
	cases := map[string]float64{
		`{"answer": "O(n log n)"}`:        1,
		`The complexity is O(N LOG N).`:   1,
		`{"answer": "O(n^2)"}`:            0,
		"O(n log n) because of the sort.": 1, // suffix match after normalization
	}
	for in, want := range cases {
		if got := g(in, nil); got != want {
			t.Errorf("normalizedAnswer(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestGradeRegex(t *testing.T) {
	cases := map[string]float64{
		`{"answer": "c(a[rtb])"}`:         1,
		`{"answer": "(cat|car|cab)"}`:     1,
		`{"answer": "c(a[rtb])?"}`:        1, // RE2 optional group can't partially match, so ca/cart stay rejected
		`{"answer": "(cat|car|cab|cap)"}`: 0,
		`{"answer": "cat|car"}`:           0, // misses cab
		`{"answer": "c[("}`:               0, // invalid regex
		"no json here":                    0,
	}
	for in, want := range cases {
		if got := gradeRegex(in, nil); got != want {
			t.Errorf("gradeRegex(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestGradeBullets(t *testing.T) {
	good := "- Edge cuts round trips\n- latency drops near users\n- Less cloud spend overall"
	if got := gradeBullets(good, nil); got != 1 {
		t.Errorf("gradeBullets(good) = %v, want 1", got)
	}
	four := "- one\n- latency two\n- three\n- four"
	if got := gradeBullets(four, nil); got >= 1 {
		t.Errorf("gradeBullets(four bullets) = %v, want < 1", got)
	}
	wrongSpot := "- a b c d e\n- f g h i j\n- k l m n o"
	if got := gradeBullets(wrongSpot, nil); got >= 1 {
		t.Errorf("gradeBullets(no keyword) = %v, want < 1", got)
	}
	longWords := "- a b c d e f\n- latency here now ok\n- x y z w v"
	if got := gradeBullets(longWords, nil); got >= 1 {
		t.Errorf("gradeBullets(six-word bullet) = %v, want < 1", got)
	}
}

func TestExtractJSONObject(t *testing.T) {
	obj := extractJSONObject("Sure!\n```json\n{\"answer\": 6}\n```\nDone.")
	if obj == nil || obj["answer"] == nil {
		t.Fatalf("fenced object not extracted: %v", obj)
	}
	obj = extractJSONObject(`prefix {"answer": {"nested": "}tricky"}, "x": 1} suffix`)
	if obj == nil || obj["x"] == nil {
		t.Fatalf("balanced-brace scan failed: %v", obj)
	}
	if extractJSONObject("no braces") != nil {
		t.Error("expected nil for braceless input")
	}
}

func TestSVGBasicShapes(t *testing.T) {
	c := findCase(t, "svg-basic-shapes")
	good := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200"><rect width="100" height="100" fill="blue"/><circle cx="150" cy="150" r="30" fill="red"/></svg>`
	if got := c.grade(good, nil); got != 1 {
		t.Errorf("valid SVG scored %v, want 1", got)
	}
	wrongViewBox := strings.Replace(good, `viewBox="0 0 200 200"`, `viewBox="0 0 100 100"`, 1)
	if got := c.grade(wrongViewBox, nil); got >= 1 {
		t.Errorf("wrong viewBox scored %v, want < 1", got)
	}
	if got := c.grade("<div>not svg</div>", nil); got != 0 {
		t.Errorf("non-SVG scored %v, want 0", got)
	}
}

func TestSVGBarChartRatios(t *testing.T) {
	c := findCase(t, "svg-bar-chart")
	good := `<svg viewBox="0 0 300 220">
  <rect x="10" y="20" width="50" height="200" fill="#333"/>
  <rect x="80" y="88.75" width="50" height="112.5" fill="#666"/>
  <rect x="150" y="150" width="50" height="50" fill="#999"/>
</svg>`
	if got := c.grade(good, nil); got != 1 {
		t.Errorf("proportional bars scored %v, want 1", got)
	}
	bad := `<svg viewBox="0 0 300 220">
  <rect x="10" y="20" width="50" height="200" fill="#333"/>
  <rect x="80" y="20" width="50" height="199" fill="#666"/>
  <rect x="150" y="20" width="50" height="198" fill="#999"/>
</svg>`
	if got := c.grade(bad, nil); got >= 1 {
		t.Errorf("flat bars scored %v, want < 1", got)
	}
}

func TestSVGSceneAndWordmark(t *testing.T) {
	scene := findCase(t, "svg-mountain-scene")
	goodScene := `<svg viewBox="0 0 400 300"><circle cx="330" cy="60" r="30" fill="#f90"/><polygon points="20,280 120,80 220,280" fill="#456"/><polygon points="120,280 240,40 360,280" fill="#789"/><polygon points="250,280 320,140 390,280" fill="#123"/></svg>`
	if got := scene.grade(goodScene, nil); got != 1 {
		t.Errorf("valid scene scored %v, want 1 (checks: mountains/circle/fills)", got)
	}

	wordmark := findCase(t, "svg-wordmark")
	goodMark := `<svg viewBox="0 0 300 100"><text x="10" y="60" fill="#111">OPENPATHS</text><rect width="300" height="100" fill="#eee"/></svg>`
	if got := wordmark.grade(goodMark, nil); got != 1 {
		t.Errorf("valid wordmark scored %v, want 1", got)
	}
	tooManyColors := `<svg viewBox="0 0 300 100"><text x="10" y="60" fill="#111">OPENPATHS</text><rect width="30" height="100" fill="#222"/><rect x="40" width="30" height="100" fill="#333"/><rect x="80" width="30" height="100" fill="#444"/></svg>`
	if got := wordmark.grade(tooManyColors, nil); got >= 1 {
		t.Errorf("4-color wordmark scored %v, want < 1", got)
	}
}

func TestToolCallGraders(t *testing.T) {
	single := findCase(t, "tool-single-call")
	if got := single.grade("", []model.ToolCall{tc("get_weather", `{"city":"Tokyo","unit":"celsius"}`)}); got != 1 {
		t.Errorf("single correct call = %v, want 1", got)
	}
	if got := single.grade("", nil); got != 0 {
		t.Errorf("no call = %v, want 0", got)
	}
	if got := single.grade("", []model.ToolCall{tc("get_time", `{"timezone":"UTC"}`)}); got != 0.25 {
		t.Errorf("wrong tool call = %v, want 0.25", got)
	}
	if got := single.grade("", []model.ToolCall{tc("get_weather", `{}`)}); got != 0.5 {
		t.Errorf("right tool bad args = %v, want 0.5", got)
	}

	pair := findCase(t, "tool-two-calls")
	both := []model.ToolCall{tc("get_weather", `{"city":"Paris"}`), tc("get_time", `{"timezone":"Europe/Paris"}`)}
	if got := pair.grade("", both); got != 1 {
		t.Errorf("both calls = %v, want 1", got)
	}
	half := []model.ToolCall{tc("get_weather", `{"city":"Paris"}`)}
	if got := pair.grade("", half); got != 0.5 {
		t.Errorf("half calls = %v, want 0.5", got)
	}

	recovery := findCase(t, "tool-error-recovery")
	if len(recovery.Messages) == 0 {
		t.Fatal("recovery case must carry full message history")
	}
	if got := recovery.grade("", []model.ToolCall{tc("get_weather", `{"city":"New York"}`)}); got != 1 {
		t.Errorf("recovery retry = %v, want 1", got)
	}
	if got := recovery.grade("", []model.ToolCall{tc("get_weather", `{"city":"NYC"}`)}); got != 0 {
		t.Errorf("recovery repeat-failure = %v, want 0", got)
	}

	noTool := findCase(t, "no-tool-needed")
	if got := noTool.grade("408", nil); got != 1 {
		t.Errorf("no-tool direct answer = %v, want 1", got)
	}
	if got := noTool.grade("408", []model.ToolCall{tc("get_weather", `{"city":"Paris"}`)}); got != 0.5 {
		t.Errorf("no-tool with spurious call = %v, want 0.5", got)
	}
}

func TestEveryCaseHasGraderAndPrompt(t *testing.T) {
	seen := map[string]bool{}
	count := 0
	for _, c := range evalCases() {
		if c.grade == nil {
			t.Errorf("case %s has no grader", c.ID)
		}
		if len(c.Messages) == 0 && strings.TrimSpace(c.Prompt) == "" {
			t.Errorf("case %s has neither prompt nor messages", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("duplicate case id %s", c.ID)
		}
		seen[c.ID] = true
		count++
	}
	if count < 12 {
		t.Errorf("expected at least 12 cases, got %d", count)
	}
}

func TestTPS(t *testing.T) {
	start := time.Unix(0, 0)
	if got := tps(100, start, 3000, 500); got < 39 || got > 41 {
		t.Errorf("tps = %v, want ~40", got)
	}
	if got := tps(0, start, 1000, 100); got != 0 {
		t.Errorf("zero-token tps = %v, want 0", got)
	}
}
