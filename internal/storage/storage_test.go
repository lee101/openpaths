package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeExt(t *testing.T) {
	for in, want := range map[string]string{
		"a.png":                  ".png",
		"a":                      ".bin",
		"a.":                     ".bin",
		"a.?acl":                 ".bin",
		"a.%2f%2e%2e%2fx":        ".bin",
		"a.p\\ng":                ".bin",
		"a.p\x00ng":              ".bin",
		"../../etc/passwd":       ".bin",
		"x.png/../../etc/passwd": ".bin",
	} {
		if got := safeExt(in); got != want {
			t.Errorf("safeExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalStoreUploadStaysInDir(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStore(dir, "http://x")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Upload(context.Background(), "a.%2f..%2f..%2fpwn", "", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimPrefix(u, "http://x/uploads/")
	if !strings.HasSuffix(name, ".bin") || strings.ContainsAny(name, "/%") {
		t.Fatalf("stored name %q", name)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
}
