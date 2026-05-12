package audio

import (
	"context"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type fakeEmotionEmbedder struct{}

func (f *fakeEmotionEmbedder) Name() string { return "fake-emotion" }

func (f *fakeEmotionEmbedder) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	var inputs []string
	switch v := req.Input.(type) {
	case []string:
		inputs = v
	case string:
		inputs = []string{v}
	default:
		inputs = []string{}
	}
	data := make([]model.EmbeddingData, 0, len(inputs))
	for i, input := range inputs {
		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Embedding: fakeEmotionVector(input),
			Index:     i,
		})
	}
	return &model.EmbeddingResponse{Object: "list", Data: data}, nil
}

func fakeEmotionVector(text string) []float64 {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "hate you") || strings.Contains(lower, "shut the fuck up") || strings.Contains(lower, "get out") || strings.Contains(lower, "run!") || strings.TrimSpace(lower) == "help!":
		return []float64{0, 0, 0, 1, 0, 0, 0}
	case strings.Contains(lower, "don't tell him") || strings.Contains(lower, "dont tell him") || strings.Contains(lower, "keep your voice down") || strings.Contains(lower, "be quiet"):
		return []float64{0, 0, 0, 0, 1, 0, 0}
	case strings.Contains(lower, "pissed") || strings.Contains(lower, "can't stand") || strings.Contains(lower, "hell out"):
		return []float64{0, 0, 0, 0, 0, 1, 0}
	case strings.Contains(lower, "what are you talking about") || strings.Contains(lower, "makes no sense"):
		return []float64{0, 0, 0, 0, 0, 0, 1}
	case strings.Contains(lower, "force") || strings.Contains(lower, "resolve") || strings.Contains(lower, "determined"):
		return []float64{1, 0, 0, 0, 0, 0, 0}
	case strings.Contains(lower, "hilarious") || strings.Contains(lower, "joking") || strings.Contains(lower, "laugh"):
		return []float64{0, 1, 0, 0, 0, 0, 0}
	default:
		return []float64{0, 0, 1, 0, 0, 0, 0}
	}
}

func newTestAutoEmotion(t *testing.T) *AutoEmotion {
	t.Helper()
	marker := &AutoEmotion{
		embedder:  &fakeEmotionEmbedder{},
		threshold: 0.99,
		examples: map[string][]string{
			"determination": {"resolve to force my way through"},
			"laughs":        {"joking about something hilarious"},
			"shouting":      {"I hate you, go away", "shut the fuck up", "get out"},
			"whispers":      {"don't tell him what you're thinking", "keep your voice down"},
			"anger":         {"I'm pissed and can't stand this", "get the hell out of here"},
			"confusion":     {"what are you talking about", "that makes no sense"},
			"":              {"plain factual narration", "hi how are you doing today", "im a programming assistant"},
		},
	}
	if err := marker.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	return marker
}

func TestDefaultEmotionExamplesCoverSupportedTags(t *testing.T) {
	examples := defaultEmotionExamples()
	for tag := range supportedEmotionTags {
		values := examples[tag]
		if len(values) < 20 {
			t.Fatalf("tag %q has %d examples, want at least 20", tag, len(values))
		}
	}
	if len(examples[""]) < 20 {
		t.Fatalf("none class has %d examples, want at least 20", len(examples[""]))
	}
}

func TestAutoEmotionMarkupCommonSenseExamples(t *testing.T) {
	marker := newTestAutoEmotion(t)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "hostile slang maps to shouting",
			in:   "i hate you go away",
			want: "[shouting] i hate you go away",
		},
		{
			name: "profanity command maps to shouting",
			in:   "shut the fuck up!",
			want: "[shouting] shut the fuck up!",
		},
		{
			name: "quiet secret maps to whispers",
			in:   "dont tell him what your thinking",
			want: "[whispers] dont tell him what your thinking",
		},
		{
			name: "punctuated quiet secret maps to whispers",
			in:   "Don't tell him what you're thinking.",
			want: "[whispers] Don't tell him what you're thinking.",
		},
		{
			name: "angry wording maps to anger",
			in:   "I'm pissed and I can't stand this.",
			want: "[anger] I'm pissed and I can't stand this.",
		},
		{
			name: "confused wording maps to confusion",
			in:   "What are you talking about?",
			want: "[confusion] What are you talking about?",
		},
		{
			name: "plain narration stays untagged",
			in:   "The corridor is empty.",
			want: "The corridor is empty.",
		},
		{
			name: "friendly greeting stays untagged",
			in:   "hi how are you doing today",
			want: "hi how are you doing today",
		},
		{
			name: "assistant identity stays untagged",
			in:   "im a programming assistant",
			want: "im a programming assistant",
		},
		{
			name: "punctuated assistant identity stays untagged",
			in:   "I'm a programming assistant.",
			want: "I'm a programming assistant.",
		},
		{
			name: "ordinary help question stays untagged",
			in:   "What can I help you with today?",
			want: "What can I help you with today?",
		},
		{
			name: "two neutral assistant sentences stay untagged",
			in:   "Hi, how are you doing today? I'm a programming assistant.",
			want: "Hi, how are you doing today? I'm a programming assistant.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marker.Markup(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("Markup() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Markup() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "[none]") {
				t.Fatalf("must not emit none tag: %q", got)
			}
		})
	}
}

func TestAutoEmotionMarkupPrefixesMatchingSentences(t *testing.T) {
	marker := newTestAutoEmotion(t)

	got, err := marker.Markup(context.Background(), "I will force my way through. The corridor is empty. That was hilarious!")
	if err != nil {
		t.Fatalf("Markup() error = %v", err)
	}

	if !strings.Contains(got, "[determination] I will force my way through.") {
		t.Fatalf("missing determination tag: %q", got)
	}
	if strings.Contains(got, "[none]") {
		t.Fatalf("must not emit none tag: %q", got)
	}
	if strings.Contains(got, "[neutral] The corridor is empty.") {
		t.Fatalf("none examples should leave neutral narration untagged: %q", got)
	}
	if !strings.Contains(got, "[laughs] That was hilarious!") {
		t.Fatalf("missing laughs tag: %q", got)
	}
}

func TestAutoEmotionMarkupPreservesManualTags(t *testing.T) {
	marker := newTestAutoEmotion(t)

	got, err := marker.Markup(context.Background(), "[whispers] I will force my way through. That was hilarious!")
	if err != nil {
		t.Fatalf("Markup() error = %v", err)
	}

	if strings.Contains(got, "[determination] [whispers]") || strings.Contains(got, "[determination] I will force") {
		t.Fatalf("manual leading tag should prevent auto prefix: %q", got)
	}
	if !strings.Contains(got, "[laughs] That was hilarious!") {
		t.Fatalf("expected second sentence to be tagged: %q", got)
	}
}

func TestAutoEmotionMarkupOnlyAppliesToTranscriptBody(t *testing.T) {
	marker := newTestAutoEmotion(t)

	input := "Read the following transcript.\n\n# Audio Profile\nFor Speaker 1: forceful guide.\n\n## Transcript:\nI will force my way through."
	got, err := marker.Markup(context.Background(), input)
	if err != nil {
		t.Fatalf("Markup() error = %v", err)
	}

	if strings.Contains(got, "For Speaker 1: [determination]") {
		t.Fatalf("profile text should not be tagged: %q", got)
	}
	if !strings.Contains(got, "## Transcript:\n[determination] I will force my way through.") {
		t.Fatalf("transcript text should be tagged: %q", got)
	}
}
