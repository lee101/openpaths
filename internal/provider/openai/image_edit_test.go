package openai

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateImageEditUploadsURLInputAsMultipart(t *testing.T) {
	originalDownloader := downloadImageInput
	downloadImageInput = func(context.Context, string) ([]byte, string, string, error) {
		return []byte("fake image bytes"), "image/png", "source.png", nil
	}
	defer func() { downloadImageInput = originalDownloader }()

	var form *multipart.Reader
	var fields = map[string]string{}
	var imageBytes []byte
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("parse content type: %v", err)
			return
		}
		form = multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := form.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("read multipart form: %v", err)
				return
			}
			data, _ := io.ReadAll(part)
			if part.FormName() == "image" {
				imageBytes = data
			} else {
				fields[part.FormName()] = string(data)
			}
		}
		_ = json.NewEncoder(w).Encode(model.ImageGenerationResponse{Data: []model.ImageData{{B64JSON: "abc"}}})
	}))
	defer api.Close()

	provider := New("test-key", api.URL)
	result, err := provider.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "gpt-image-2", Prompt: "make it a painting", Size: "1536x1024", N: 1,
		ImageURL: "https://example.com/source.png",
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if fields["model"] != "gpt-image-2" || fields["prompt"] != "make it a painting" || fields["size"] != "1536x1024" {
		t.Fatalf("form fields = %#v", fields)
	}
	if string(imageBytes) != "fake image bytes" || len(result.Data) != 1 || result.Data[0].B64JSON != "abc" {
		t.Fatalf("image/result = %q / %#v", imageBytes, result)
	}
}

func TestImageInputURLsCollectsPrimaryAndReferenceInputs(t *testing.T) {
	urls := imageInputURLs(&model.ImageGenerationRequest{
		ImageURL:           "https://example.com/one.png",
		Images:             []model.ImageInput{{URL: "https://example.com/two.png"}},
		ReferenceImageURLs: []string{"https://example.com/three.png"},
	})
	if len(urls) != 3 || urls[0] != "https://example.com/one.png" || urls[2] != "https://example.com/three.png" {
		t.Fatalf("urls = %#v", urls)
	}
}
