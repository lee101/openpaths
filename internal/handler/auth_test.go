package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestActiveAPIKeyCount(t *testing.T) {
	keys := []model.APIKey{
		{Revoked: false},
		{Revoked: true},
		{Revoked: false},
	}
	if got := activeAPIKeyCount(keys); got != 2 {
		t.Fatalf("activeAPIKeyCount() = %d, want 2", got)
	}
}
