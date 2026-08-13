package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceEnvPreservesUnrelatedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("A=one\nTEST_API_KEY=old\nB=two words\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceEnv(path, "TEST_API_KEY", "op-new"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "A=one\nTEST_API_KEY=op-new\nB=two words\n"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestReplaceEnvAppendsMissingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("A=one\n"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := replaceEnv(path, "TEST_API_KEY", "op-new"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(got), "TEST_API_KEY=op-new\n") {
		t.Fatalf("missing appended key: %q", got)
	}
}
