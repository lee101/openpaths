package audio

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const defaultAutoEmotionThreshold = 0.62

var supportedEmotionTags = map[string]struct{}{
	"admiration": {}, "adoration": {}, "aggression": {}, "agitation": {}, "amusement": {},
	"anger": {}, "annoyance": {}, "awe": {}, "confusion": {}, "curiosity": {},
	"determination": {}, "enthusiasm": {}, "excitement": {}, "frustration": {}, "hope": {},
	"interest": {}, "laughs": {}, "negative": {}, "nervousness": {}, "neutral": {},
	"positive": {}, "shouting": {}, "tension": {}, "whispers": {},
}

type AutoEmotion struct {
	embedder  provider.EmbeddingProvider
	threshold float64
	examples  map[string][]string
	entries   []emotionEntry
	ready     bool
	mu        sync.RWMutex
}

type emotionEntry struct {
	tag       string
	text      string
	embedding []float64
}

func NewAutoEmotion(embedder provider.EmbeddingProvider) *AutoEmotion {
	return &AutoEmotion{
		embedder:  embedder,
		threshold: defaultAutoEmotionThreshold,
		examples:  defaultEmotionExamples(),
	}
}

func (a *AutoEmotion) Init(ctx context.Context) error {
	if a == nil || a.embedder == nil {
		return fmt.Errorf("autoemotion: missing embedder")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	entries := make([]emotionEntry, 0, 512)
	texts := make([]string, 0, 512)
	for tag, examples := range a.examples {
		if tag != "" {
			if _, ok := supportedEmotionTags[tag]; !ok {
				return fmt.Errorf("autoemotion: unsupported tag %q", tag)
			}
		}
		for _, example := range examples {
			trimmed := strings.TrimSpace(example)
			if trimmed == "" {
				continue
			}
			entries = append(entries, emotionEntry{tag: tag, text: trimmed})
			texts = append(texts, trimmed)
		}
	}
	if len(entries) == 0 {
		return fmt.Errorf("autoemotion: no examples configured")
	}

	embeddings, err := a.embed(ctx, texts)
	if err != nil {
		return err
	}
	if len(embeddings) != len(entries) {
		return fmt.Errorf("autoemotion: embedding count mismatch got %d want %d", len(embeddings), len(entries))
	}
	for i := range entries {
		entries[i].embedding = embeddings[i]
	}

	a.entries = entries
	a.ready = true
	return nil
}

func (a *AutoEmotion) Markup(ctx context.Context, text string) (string, error) {
	if a == nil {
		return text, nil
	}
	if idx := transcriptBodyIndex(text); idx >= 0 {
		marked, err := a.Markup(ctx, text[idx:])
		if err != nil {
			return "", err
		}
		return text[:idx] + marked, nil
	}

	spans := splitSpeechSpans(text)
	if len(spans) == 0 {
		return text, nil
	}

	segments := make([]string, 0, len(spans))
	segmentIndexes := make([]int, 0, len(spans))
	for i, span := range spans {
		if !span.speech || hasLeadingAudioTag(span.text) {
			continue
		}
		cleaned := stripAudioTags(span.text)
		if cleaned == "" {
			continue
		}
		segments = append(segments, cleaned)
		segmentIndexes = append(segmentIndexes, i)
	}
	if len(segments) == 0 {
		return text, nil
	}

	embeddings, err := a.embed(ctx, segments)
	if err != nil {
		return "", err
	}
	for i, emb := range embeddings {
		tag := a.classify(emb)
		if tag == "" {
			continue
		}
		idx := segmentIndexes[i]
		spans[idx].text = prefixAfterWhitespace(spans[idx].text, "["+tag+"] ")
	}

	var b strings.Builder
	for _, span := range spans {
		b.WriteString(span.text)
	}
	return b.String(), nil
}

func transcriptBodyIndex(text string) int {
	lower := strings.ToLower(text)
	for _, marker := range []string{"## transcript:", "# transcript:", "transcript:"} {
		idx := strings.LastIndex(lower, marker)
		if idx < 0 {
			continue
		}
		body := idx + len(marker)
		for body < len(text) && (text[body] == ' ' || text[body] == '\t' || text[body] == '\r' || text[body] == '\n') {
			body++
		}
		if body < len(text) {
			return body
		}
	}
	return -1
}

func (a *AutoEmotion) classify(query []float64) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.ready || len(query) == 0 {
		return ""
	}

	bestTag := ""
	bestSim := -1.0
	for _, entry := range a.entries {
		sim := cosine(query, entry.embedding)
		if sim > bestSim {
			bestSim = sim
			bestTag = entry.tag
		}
	}
	if bestSim < a.threshold {
		return ""
	}
	return bestTag
}

func (a *AutoEmotion) embed(ctx context.Context, texts []string) ([][]float64, error) {
	resp, err := a.embedder.Embed(ctx, &model.EmbeddingRequest{Input: texts})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("autoemotion: got %d embeddings for %d texts", len(resp.Data), len(texts))
	}
	embeddings := make([][]float64, len(resp.Data))
	for _, data := range resp.Data {
		if data.Index < 0 || data.Index >= len(resp.Data) {
			return nil, fmt.Errorf("autoemotion: embedding index %d out of range", data.Index)
		}
		embeddings[data.Index] = data.Embedding
	}
	for i, emb := range embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("autoemotion: empty embedding at index %d", i)
		}
	}
	return embeddings, nil
}

type speechSpan struct {
	text   string
	speech bool
}

func splitSpeechSpans(text string) []speechSpan {
	var spans []speechSpan
	start := 0
	for i, r := range text {
		if r != '.' && r != '!' && r != '?' && r != '\n' {
			continue
		}
		end := i + len(string(r))
		for end < len(text) && (text[end] == '"' || text[end] == '\'' || text[end] == ')' || text[end] == ']') {
			end++
		}
		spans = append(spans, makeSpeechSpan(text[start:end]))
		start = end
	}
	if start < len(text) {
		spans = append(spans, makeSpeechSpan(text[start:]))
	}
	return spans
}

func makeSpeechSpan(text string) speechSpan {
	trimmed := strings.TrimSpace(text)
	return speechSpan{
		text:   text,
		speech: trimmed != "" && !strings.HasSuffix(trimmed, ":") && len(strings.Fields(trimmed)) > 1,
	}
}

var leadingAudioTagRE = regexp.MustCompile(`^\s*\[[A-Za-z ]+\]`)
var anyAudioTagRE = regexp.MustCompile(`\[[A-Za-z ]+\]`)

func hasLeadingAudioTag(text string) bool {
	return leadingAudioTagRE.MatchString(text)
}

func stripAudioTags(text string) string {
	return strings.TrimSpace(anyAudioTagRE.ReplaceAllString(text, " "))
}

func prefixAfterWhitespace(text, prefix string) string {
	leading := len(text) - len(strings.TrimLeft(text, " \t\r\n"))
	return text[:leading] + prefix + text[leading:]
}

func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func defaultEmotionExamples() map[string][]string {
	descriptors := map[string]string{
		"":              "plain narration, factual scene setting, neutral exposition, no performed emotion",
		"admiration":    "respectful praise for skill, courage, beauty, or achievement",
		"adoration":     "warm devotion, affection, cherishing, loving appreciation",
		"aggression":    "threatening confrontation, hostile challenge, forceful attack",
		"agitation":     "restless alarm, frantic movement, unsettled urgency",
		"amusement":     "playful delight, finding something funny, entertained reaction",
		"anger":         "furious protest, rage, outrage, explosive displeasure",
		"annoyance":     "irritated complaint, impatience, bothered dismissal",
		"awe":           "wonder, amazement, reverence before something vast or impossible",
		"confusion":     "uncertainty, not understanding, disoriented questioning",
		"curiosity":     "inquisitive interest, wanting to know more, investigative question",
		"determination": "resolve, courage, refusing to give up, pushing forward",
		"enthusiasm":    "energetic approval, eager participation, spirited support",
		"excitement":    "thrilled anticipation, high energy, breathless joy",
		"frustration":   "blocked effort, exasperation, repeated failure, strained patience",
		"hope":          "optimistic belief, fragile possibility, looking toward a better outcome",
		"interest":      "focused attention, engaged observation, thoughtful investment",
		"laughs":        "laughter, chuckling, joking, comic reaction",
		"negative":      "bleak judgment, sadness, rejection, bad outcome",
		"nervousness":   "anxious hesitation, fear, trembling uncertainty",
		"neutral":       "even tone, balanced statement, calm factual delivery",
		"positive":      "approval, relief, happy confidence, encouraging response",
		"shouting":      "raised voice, yelling, shouted warning, loud command",
		"tension":       "suspense, danger, pressure, ominous conflict",
		"whispers":      "quiet secret, hushed warning, barely audible speech",
	}
	templates := []string{
		"%s.",
		"The line carries %s.",
		"Say this with %s in the voice.",
		"A character speaks from %s.",
		"The feeling underneath is %s.",
		"This moment should sound like %s.",
		"The delivery suggests %s.",
		"The speaker is colored by %s.",
		"The sentence expresses %s.",
		"The emotional intent is %s.",
		"A short line full of %s.",
		"A dramatic phrase shaped by %s.",
		"The dialogue lands with %s.",
		"The performance direction is %s.",
		"The voice should imply %s.",
		"The scene beat is driven by %s.",
		"The response reveals %s.",
		"The subtext is %s.",
		"The actor should lean into %s.",
		"The listener should hear %s.",
	}
	examples := make(map[string][]string, len(descriptors))
	for tag, descriptor := range descriptors {
		values := make([]string, 0, len(templates))
		for _, tmpl := range templates {
			values = append(values, fmt.Sprintf(tmpl, descriptor))
		}
		examples[tag] = values
	}
	for tag, values := range realisticEmotionExamples() {
		examples[tag] = append(examples[tag], values...)
	}
	return examples
}

func realisticEmotionExamples() map[string][]string {
	return map[string][]string{
		"": {
			"Hi, how are you doing today?",
			"Hello, how are you?",
			"I'm a programming assistant.",
			"I am a programming assistant.",
			"What can I help you with today?",
			"Thanks for reaching out.",
			"Here is the information you requested.",
			"Let me know if you need anything else.",
			"I can help with coding questions.",
			"Please provide the file path.",
			"The answer depends on your setup.",
			"That endpoint returns JSON.",
			"This example uses the default settings.",
			"You can run the command again.",
			"Today is Tuesday.",
			"The build completed successfully.",
			"Your account is active.",
			"The request includes a model and input field.",
			"We should verify the response before deploying.",
			"This is a neutral assistant response.",
			"A dark crumbling dungeon with dripping water echoing in the distance.",
			"The rain started just after midnight.",
			"The door was made of old oak and iron.",
			"Sample context for a fantasy RPG scene.",
			"The package arrived on Tuesday morning.",
			"Speaker one stands near the northern gate.",
			"The hallway turns left after the stairs.",
			"This line is plain scene description.",
			"The room contains a table, two chairs, and a lamp.",
			"The file was saved to the archive.",
		},
		"admiration": {
			"You handled that beautifully.",
			"That was an incredible move.",
			"I have to respect how you pulled that off.",
			"You're genuinely good at this.",
			"That took real skill.",
			"Damn, that was impressive.",
		},
		"adoration": {
			"I love you more than anything.",
			"You're my whole world.",
			"I can't stop thinking about you.",
			"You're perfect to me.",
			"I adore every little thing about you.",
			"Come here, sweetheart.",
		},
		"aggression": {
			"Back off before I make you.",
			"Move or I'll move you myself.",
			"Touch that door and you're done.",
			"Say that again and see what happens.",
			"I will break through you if I have to.",
			"Get out of my way right now.",
		},
		"agitation": {
			"Where the hell are my keys?",
			"No no no, this can't be happening.",
			"Come on, come on, pick up the phone.",
			"I can't sit still right now.",
			"Everything is moving too fast.",
			"We have to go, we have to go now.",
		},
		"amusement": {
			"That's actually pretty funny.",
			"You're kidding me, right?",
			"Oh wow, that's ridiculous.",
			"I can't believe you said that.",
			"Bro, what was that?",
			"That is so dumb it's funny.",
		},
		"anger": {
			"I can't stand you right now.",
			"Get the hell out of here.",
			"Don't you ever talk to me like that again.",
			"I'm so pissed at you right now.",
			"You absolute idiot, what did you do?",
			"Shut up and leave me alone.",
		},
		"annoyance": {
			"Can you stop doing that?",
			"Bro, seriously, knock it off.",
			"Ugh, not this again.",
			"You're getting on my nerves.",
			"That's annoying as hell.",
			"Please just leave it alone.",
		},
		"awe": {
			"Look at the size of that thing.",
			"I've never seen anything like this.",
			"It's beautiful.",
			"Holy shit, the whole sky is on fire.",
			"This is impossible.",
			"We're standing inside a miracle.",
		},
		"confusion": {
			"What are you talking about?",
			"Wait, that makes no sense.",
			"I'm lost.",
			"How did we get here?",
			"Why would he do that?",
			"I don't understand what you mean.",
		},
		"curiosity": {
			"What's behind that door?",
			"Tell me more about that.",
			"How does this thing work?",
			"Why is the light still on?",
			"What happens if we press it?",
			"Okay, now I'm curious.",
		},
		"determination": {
			"I carry a message for the elder.",
			"Step aside or I will force my way through.",
			"I won't quit.",
			"We finish this tonight.",
			"I'm not backing down.",
			"No matter what happens, I keep moving.",
		},
		"enthusiasm": {
			"Let's go, this is going to be awesome.",
			"Hell yeah, I'm in.",
			"That sounds amazing.",
			"I've been waiting for this all week.",
			"Absolutely, let's do it.",
			"This is exactly what I wanted.",
		},
		"excitement": {
			"Oh my god, it's happening.",
			"We won, we actually won.",
			"I can't wait.",
			"This is huge.",
			"Let's fucking go.",
			"My heart is racing.",
		},
		"frustration": {
			"Why won't this stupid thing work?",
			"I've tried this three times already.",
			"Are you kidding me?",
			"This is such bullshit.",
			"I can't deal with this right now.",
			"Nothing is working.",
		},
		"hope": {
			"Maybe we still have a chance.",
			"It isn't over yet.",
			"If we hurry, we can still save him.",
			"Please tell me there's another way.",
			"I have to believe this can work.",
			"Tomorrow might be better.",
		},
		"interest": {
			"That's interesting.",
			"Keep going, I'm listening.",
			"I want to understand this.",
			"That detail matters.",
			"Show me how you found it.",
			"Okay, you've got my attention.",
		},
		"laughs": {
			"That's hilarious.",
			"You look like you lost a fight with a printer.",
			"No way, that's too funny.",
			"I can't stop laughing.",
			"Ha, nice one.",
			"Lmao, you really did that.",
		},
		"negative": {
			"This is bad.",
			"I don't like where this is going.",
			"That was a terrible idea.",
			"We're screwed.",
			"Everything about this feels wrong.",
			"This sucks.",
		},
		"nervousness": {
			"I don't know if I can do this.",
			"What if they see us?",
			"My hands are shaking.",
			"I think someone is following me.",
			"Please don't make me go first.",
			"Uh, guys, I don't like this.",
		},
		"neutral": {
			"The northern pass is sealed by order of the council.",
			"The meeting starts at nine.",
			"Your access code expires tomorrow.",
			"Turn left at the end of the hall.",
			"The report is on your desk.",
			"That is the current status.",
		},
		"positive": {
			"That's great news.",
			"We're going to be okay.",
			"Good job.",
			"I knew you could do it.",
			"This feels right.",
			"Thanks, I really needed that.",
		},
		"shouting": {
			"Halt, traveler!",
			"Now!",
			"I hate you, go away!",
			"Get out!",
			"Don't touch him!",
			"Run!",
			"Shut the fuck up!",
			"Leave me alone!",
			"Stop!",
			"Help!",
		},
		"tension": {
			"Something is wrong.",
			"Don't move.",
			"The shadow reached him first.",
			"The lock clicked behind us.",
			"He's standing right behind you.",
			"We don't have time for games.",
		},
		"whispers": {
			"Don't tell him what you're thinking.",
			"Don't tell him what your thinking.",
			"Keep your voice down.",
			"He's listening.",
			"The shadow is in the room.",
			"Meet me behind the chapel at midnight.",
			"Nobody can know we were here.",
			"Be quiet.",
		},
	}
}
