package discovery

import "testing"

func TestOpenAICompatModelsURLDoesNotDuplicateVersion(t *testing.T) {
	for _, tc := range []struct {
		base, want string
	}{
		{"https://api.meta.ai/v1", "https://api.meta.ai/v1/models"},
		{"https://api.meta.ai/v1/", "https://api.meta.ai/v1/models"},
		{"https://api.openai.com", "https://api.openai.com/v1/models"},
	} {
		if got := openAICompatModelsURL(tc.base); got != tc.want {
			t.Errorf("openAICompatModelsURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}
