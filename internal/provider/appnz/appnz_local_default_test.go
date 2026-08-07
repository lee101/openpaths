package appnz

import "testing"

func TestDefaultBaseURLIsTheLocalFrontDoor(t *testing.T) {
	p := New("", "")
	if p.baseURL != "http://127.0.0.1:8791" {
		t.Fatalf("default baseURL = %q, want the omniserve-native front door; nothing listens on 9080", p.baseURL)
	}
}

func TestExplicitBaseURLWins(t *testing.T) {
	p := New("", "http://127.0.0.1:9999/")
	if p.baseURL != "http://127.0.0.1:9999" {
		t.Fatalf("baseURL = %q, want the configured value with the trailing slash trimmed", p.baseURL)
	}
}
