package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Store interface {
	Upload(ctx context.Context, filename string, contentType string, r io.Reader) (url string, err error)
}

type LocalStore struct {
	dir     string
	baseURL string
}

func NewLocalStore(dir, baseURL string) (*LocalStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	log.Printf("storage: local dir=%s base=%s", dir, baseURL)
	return &LocalStore{dir: dir, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

var safeExtRe = regexp.MustCompile(`^\.[A-Za-z0-9]{1,16}$`)

func safeExt(filename string) string {
	ext := filepath.Ext(filename)
	if !safeExtRe.MatchString(ext) {
		return ".bin"
	}
	return ext
}

func (s *LocalStore) Upload(_ context.Context, filename string, _ string, r io.Reader) (string, error) {
	ext := safeExt(filename)
	id := generateID()
	stored := id + ext

	f, err := os.Create(filepath.Join(s.dir, stored))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return s.baseURL + "/uploads/" + stored, nil
}

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	ts := fmt.Sprintf("%x", time.Now().UnixMilli())
	return ts + hex.EncodeToString(b)
}
