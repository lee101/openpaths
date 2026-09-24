package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

const multipartImageMaxBytes = 20 << 20

// parseMultipartImageRequest accepts the OpenAI SDK's multipart /v1/images/edits
// shape: image files become data: URLs (masks are not forwarded).
func parseMultipartImageRequest(ctx *fasthttp.RequestCtx, req *model.ImageGenerationRequest) error {
	form, err := ctx.MultipartForm()
	if err != nil {
		return err
	}
	value := func(name string) string {
		if v := form.Value[name]; len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	req.Model = value("model")
	req.Prompt = value("prompt")
	req.Size = value("size")
	req.Quality = value("quality")
	req.ResponseFormat = value("response_format")
	req.OutputFormat = value("output_format")
	if n, err := strconv.Atoi(value("n")); err == nil {
		req.N = n
	}
	if seed, err := strconv.Atoi(value("seed")); err == nil {
		req.Seed = &seed
	}
	for _, name := range []string{"image", "image[]", "image_file"} {
		for _, file := range form.File[name] {
			url, err := multipartDataURL(file)
			if err != nil {
				return err
			}
			req.Images = append(req.Images, model.ImageInput{Type: "image_url", URL: url})
		}
	}
	if url := value("image_url"); url != "" {
		req.Images = append(req.Images, model.ImageInput{Type: "image_url", URL: url})
	}
	if len(req.Images) > 0 {
		first := req.Images[0]
		req.Image = &first
	}
	return nil
}

func multipartDataURL(file *multipart.FileHeader) (string, error) {
	if file.Size > multipartImageMaxBytes {
		return "", fmt.Errorf("%s exceeds %d MiB", file.Filename, multipartImageMaxBytes>>20)
	}
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, multipartImageMaxBytes+1))
	if err != nil {
		return "", err
	}
	mediaType := http.DetectContentType(data)
	if !strings.HasPrefix(mediaType, "image/") {
		mediaType = file.Header.Get("Content-Type")
	}
	if !strings.HasPrefix(mediaType, "image/") {
		return "", fmt.Errorf("%s is not an image", file.Filename)
	}
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
