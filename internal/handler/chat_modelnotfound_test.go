package handler

import "testing"

func TestIsModelNotFoundError(t *testing.T) {
	cases := []struct {
		status int
		msg    string
		want   bool
	}{
		{400, "Model not found: foo", true},
		{400, `{"error":{"code":"model_not_found"}}`, true},
		{400, "The model does not exist", true},
		{400, "invalid schema", false},
		{404, "model not found", false},
		{500, "model not found", false},
	}
	for _, c := range cases {
		if got := isModelNotFoundError(c.status, c.msg); got != c.want {
			t.Errorf("%d %q: got %v want %v", c.status, c.msg, got, c.want)
		}
	}
}
