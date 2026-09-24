package handler

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

func TestParseMultipartImageRequest(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range map[string]string{"model": "ra2-edit", "prompt": "make it blue", "size": "512x512", "n": "1", "seed": "7"} {
		_ = w.WriteField(k, v)
	}
	part, _ := w.CreateFormFile("image", "a.png")
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	_, _ = part.Write(png)
	_ = w.Close()
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType(w.FormDataContentType())
	ctx.Request.SetBody(body.Bytes())
	var req model.ImageGenerationRequest
	if err := parseMultipartImageRequest(ctx, &req); err != nil {
		t.Fatal(err)
	}
	if req.Model != "ra2-edit" || req.Prompt != "make it blue" || req.Size != "512x512" || req.N != 1 || req.Seed == nil || *req.Seed != 7 {
		t.Fatalf("fields: %+v", req)
	}
	if req.Image == nil || !strings.HasPrefix(req.Image.URL, "data:image/png;base64,") || len(req.Images) != 1 {
		t.Fatalf("image: %+v", req.Image)
	}
}
