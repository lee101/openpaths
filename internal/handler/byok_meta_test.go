package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/provider"
)

func TestMakeUserProviderMeta(t *testing.T) {
	p := makeUserProvider("meta", "meta-test-key")
	if p == nil || p.Name() != "meta" {
		t.Fatalf("Meta BYOK provider = %#v", p)
	}
	if _, ok := p.(provider.ImageProvider); !ok {
		t.Fatal("Meta BYOK provider does not support images")
	}
	if _, ok := p.(provider.TranscriptionProvider); !ok {
		t.Fatal("Meta BYOK provider does not support transcription")
	}
}
