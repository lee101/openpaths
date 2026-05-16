package handler

import "testing"

func TestParseOpenAIMaxPlanCredentials(t *testing.T) {
	raw := `{
		"credentials": [
			{"auth_mode": "chatgpt", "OPENAI_API_KEY": "sk-one"},
			{"api_key": "sk-two"},
			{"token": "sk-three"},
			"sk-four"
		]
	}`

	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 4 {
		t.Fatalf("len(creds) = %d, want 4", len(creds))
	}
	got := map[string]bool{}
	for _, cred := range creds {
		got[cred.APIKey] = true
		if cred.ID == "" || cred.Label == "" {
			t.Fatalf("credential missing stable metadata: %+v", cred)
		}
	}
	for _, want := range []string{"sk-one", "sk-two", "sk-three", "sk-four"} {
		if !got[want] {
			t.Fatalf("missing credential %q in %+v", want, got)
		}
	}
}

func TestParseOpenAIMaxPlanCredentialsDedupes(t *testing.T) {
	raw := `[{"OPENAI_API_KEY":"sk-one"},{"OPENAI_API_KEY":"sk-one"}]`
	creds := parseOpenAIMaxPlanCredentials("user-1", openAIMaxPlanProvider, raw)
	if len(creds) != 1 {
		t.Fatalf("len(creds) = %d, want 1", len(creds))
	}
}
