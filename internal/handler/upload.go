package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"mime"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/storage"
)

type UploadHandler struct {
	store storage.Store
}

func NewUploadHandler(store storage.Store) *UploadHandler {
	return &UploadHandler{store: store}
}

func (h *UploadHandler) HandleUpload(ctx *fasthttp.RequestCtx) {
	ct := string(ctx.Request.Header.ContentType())

	if strings.HasPrefix(ct, "multipart/") {
		h.handleMultipart(ctx)
	} else if strings.HasPrefix(ct, "application/json") {
		h.handleJSON(ctx)
	} else {
		writeError(ctx, 400, "invalid_request", "content-type must be multipart/form-data or application/json")
	}
}

func (h *UploadHandler) handleMultipart(ctx *fasthttp.RequestCtx) {
	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "invalid multipart: "+err.Error())
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		writeError(ctx, 400, "invalid_request", "file field required")
		return
	}

	fh := files[0]
	f, err := fh.Open()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "open file: "+err.Error())
		return
	}
	defer f.Close()

	contentType := fh.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fh.Filename))
	}

	url, err := h.store.Upload(ctx, fh.Filename, contentType, f)
	if err != nil {
		log.Printf("upload: %v", err)
		writeError(ctx, 500, "upload_error", "upload failed")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"url":       url,
		"image_url": map[string]string{"url": url},
	})
}

type jsonUploadReq struct {
	Data     string `json:"data"`
	Filename string `json:"filename"`
}

func (h *UploadHandler) handleJSON(ctx *fasthttp.RequestCtx) {
	var req jsonUploadReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "invalid JSON: "+err.Error())
		return
	}
	if req.Data == "" {
		writeError(ctx, 400, "invalid_request", "data field required (base64)")
		return
	}

	raw := req.Data
	if idx := strings.Index(raw, ","); idx != -1 {
		raw = raw[idx+1:]
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			writeError(ctx, 400, "invalid_request", "invalid base64")
			return
		}
	}

	filename := req.Filename
	if filename == "" {
		filename = "upload.png"
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = detectContentType(decoded)
	}

	url, err := h.store.Upload(ctx, filename, contentType, bytes.NewReader(decoded))
	if err != nil {
		log.Printf("upload: %v", err)
		writeError(ctx, 500, "upload_error", "upload failed")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"url":       url,
		"image_url": map[string]string{"url": url},
	})
}

func detectContentType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "image/png"
	}
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	if string(data[:4]) == "RIFF" && len(data) > 11 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	if string(data[:3]) == "GIF" {
		return "image/gif"
	}
	return "application/octet-stream"
}
