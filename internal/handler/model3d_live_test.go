package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLiveImageTo3DForLeePenkman(t *testing.T) {
	if os.Getenv("RUN_LIVE_IMAGE_TO_3D") != "1" {
		t.Skip("set RUN_LIVE_IMAGE_TO_3D=1 to run the live image-to-3D integration test")
	}

	apiKey := os.Getenv("OPENPATHS_LIVE_API_KEY")
	if apiKey == "" {
		t.Fatal("OPENPATHS_LIVE_API_KEY is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("OPENPATHS_TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Fatal("DATABASE_URL or OPENPATHS_TEST_DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	sum := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(sum[:])

	var email string
	err = pool.QueryRow(ctx, `
		SELECT u.email
		FROM api_keys ak
		JOIN users u ON u.id = ak.user_id
		WHERE ak.key_hash = $1 AND NOT ak.revoked AND NOT u.disabled
	`, keyHash).Scan(&email)
	if err != nil {
		t.Fatalf("live key is not valid in db: %v", err)
	}
	if strings.ToLower(email) != "leepenkman@gmail.com" {
		t.Fatalf("live key belongs to %q, want leepenkman@gmail.com", email)
	}

	baseURL := strings.TrimRight(os.Getenv("OPENPATHS_LIVE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "https://openpaths.io"
	}

	payload := map[string]any{
		"model":                        "pixal3d-image-to-3d",
		"image_url":                    DEFAULT_IMAGE_URL_FOR_TEST,
		"resolution":                   1024,
		"texture_size":                 1024,
		"remesh":                       true,
		"ss_guidance_strength":         7.5,
		"ss_guidance_rescale":          0.7,
		"ss_sampling_steps":            12,
		"ss_rescale_t":                 5,
		"shape_slat_guidance_strength": 7.5,
		"shape_slat_guidance_rescale":  0.5,
		"shape_slat_sampling_steps":    12,
		"shape_slat_rescale_t":         3,
		"tex_slat_guidance_strength":   1,
		"tex_slat_sampling_steps":      12,
		"tex_slat_rescale_t":           3,
		"mesh_scale":                   1,
		"max_num_tokens":               49152,
		"decimation_target":            200000,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/3d/generations", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post image-to-3d: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, buf.String())
	}

	var out struct {
		ModelGLB struct {
			URL string `json:"url"`
		} `json:"model_glb"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ModelGLB.URL == "" {
		t.Fatal("missing model_glb.url")
	}
}

const DEFAULT_IMAGE_URL_FOR_TEST = "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg"
