package handler

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/storage"
)

func pngBytes(w, h int) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)))
	return buf.Bytes()
}

func TestDownloadImageDimensionsBlocksPrivateAddresses(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(pngBytes(4, 3))
	}))
	defer srv.Close()
	if w, h := downloadImageDimensions(context.Background(), srv.URL); w != 0 || h != 0 {
		t.Fatalf("got %dx%d", w, h)
	}
	if hits.Load() != 0 {
		t.Fatal("private address was fetched")
	}
}

func TestRehostImageURLRejectsOversize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(make([]byte, maxImageDownloadBytes+10))
	}))
	defer srv.Close()
	store, err := storage.NewLocalStore(t.TempDir(), "https://assets.test")
	if err != nil {
		t.Fatal(err)
	}
	h := &ImageHandler{store: store}
	if _, _, _, err := h.rehostImageURL(context.Background(), srv.URL, 0); err == nil {
		t.Fatal("want oversize error")
	}
	resp := &model.ImageGenerationResponse{Data: []model.ImageData{{URL: srv.URL}}}
	h.rehostImageURLs(context.Background(), resp)
	if resp.Data[0].URL != srv.URL {
		t.Fatalf("url = %q, want upstream kept", resp.Data[0].URL)
	}
}

func TestRehostImageURLStoresWithDimensions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(8, 5))
	}))
	defer srv.Close()
	store, _ := storage.NewLocalStore(t.TempDir(), "https://assets.test")
	h := &ImageHandler{store: store}
	u, w, hh, err := h.rehostImageURL(context.Background(), srv.URL, 1)
	if err != nil || w != 8 || hh != 5 || u == srv.URL {
		t.Fatalf("u=%q %dx%d err=%v", u, w, hh, err)
	}
}
