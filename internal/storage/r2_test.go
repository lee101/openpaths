package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestR2StoreUploadDefaultsOpenPathsPublicURL(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.Header.Get("Content-Type") != "image/png" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewR2Store(R2Config{
		Endpoint:  server.URL,
		Bucket:    "openpathsstatic",
		AccessKey: "test-access",
		SecretKey: "test-secret",
	})
	got, err := store.Upload(context.Background(), "sample.png", "image/png", strings.NewReader("png"))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if !strings.HasPrefix(got, "https://openpathsstatic.openpaths.io/uploads/") || !strings.HasSuffix(got, ".png") {
		t.Fatalf("url = %q", got)
	}
	if !strings.HasPrefix(gotPath, "/openpathsstatic/uploads/") || !strings.HasSuffix(gotPath, ".png") {
		t.Fatalf("upload path = %q", gotPath)
	}
}

func TestR2StoreUploadHonorsExplicitPublicURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := NewR2Store(R2Config{
		Endpoint:  server.URL,
		Bucket:    "openpathsstatic",
		AccessKey: "test-access",
		SecretKey: "test-secret",
		PublicURL: "https://cdn.example.com/",
	})
	got, err := store.Upload(context.Background(), "sample.jpg", "image/jpeg", strings.NewReader("jpg"))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if !strings.HasPrefix(got, "https://cdn.example.com/uploads/") || !strings.HasSuffix(got, ".jpg") {
		t.Fatalf("url = %q", got)
	}
}
