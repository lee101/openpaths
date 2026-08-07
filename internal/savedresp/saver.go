// Package savedresp persists opted-in users' generation inputs/outputs and makes
// them privately searchable (gobed semantic + pg_trgm). It mirrors the async,
// non-blocking design of internal/metrics so request handlers never block on it.
package savedresp

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	KindText  = "text"
	KindImage = "image"

	settingsTTL = 30 * time.Second
	queueSize   = 2048
)

type cachedSettings struct {
	text   bool
	images bool
	exp    time.Time
}

// Saver buffers saved-response entries, embeds their prompt, and writes them to
// the DB on a background goroutine. It also caches per-user opt-in flags so the
// hot request path can cheaply skip work when saving is disabled.
type Saver struct {
	q        *queries.SavedResponseQueries
	userQ    *queries.UserQueries
	embedder provider.EmbeddingProvider

	ch   chan *model.SavedResponse
	done chan struct{}

	mu       sync.RWMutex
	settings map[string]cachedSettings
}

func New(q *queries.SavedResponseQueries, userQ *queries.UserQueries, embedder provider.EmbeddingProvider) *Saver {
	return &Saver{
		q:        q,
		userQ:    userQ,
		embedder: embedder,
		ch:       make(chan *model.SavedResponse, queueSize),
		done:     make(chan struct{}),
		settings: make(map[string]cachedSettings),
	}
}

func (s *Saver) Start() {
	if s == nil {
		return
	}
	go func() {
		for {
			select {
			case entry := <-s.ch:
				s.persist(entry)
			case <-s.done:
				// Drain remaining queued entries before exiting.
				for {
					select {
					case entry := <-s.ch:
						s.persist(entry)
					default:
						return
					}
				}
			}
		}
	}()
}

func (s *Saver) Stop() {
	if s == nil {
		return
	}
	close(s.done)
}

// WantText reports whether the user has opted into saving text generations.
func (s *Saver) WantText(userID string) bool {
	if s == nil || userID == "" {
		return false
	}
	st := s.lookup(userID)
	return st.text
}

// WantImage reports whether the user has opted into saving image generations.
func (s *Saver) WantImage(userID string) bool {
	if s == nil || userID == "" {
		return false
	}
	st := s.lookup(userID)
	return st.images
}

// Save enqueues an entry for embedding + persistence. Non-blocking: drops if full.
func (s *Saver) Save(entry *model.SavedResponse) {
	if s == nil || entry == nil {
		return
	}
	select {
	case s.ch <- entry:
	default:
		log.Printf("savedresp: queue full, dropping %s entry", entry.Kind)
	}
}

// Invalidate clears a user's cached settings (call after they change the toggle).
func (s *Saver) Invalidate(userID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.settings, userID)
	s.mu.Unlock()
}

func (s *Saver) lookup(userID string) cachedSettings {
	now := time.Now()
	s.mu.RLock()
	st, ok := s.settings[userID]
	s.mu.RUnlock()
	if ok && now.Before(st.exp) {
		return st
	}

	st = cachedSettings{exp: now.Add(settingsTTL)}
	if s.userQ != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if u, err := s.userQ.GetByID(ctx, userID); err == nil && u != nil {
			st.text = u.SaveResponsesText
			st.images = u.SaveResponsesImages
		}
	}
	s.mu.Lock()
	s.settings[userID] = st
	s.mu.Unlock()
	return st
}

func (s *Saver) persist(entry *model.SavedResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if entry.Prompt != "" && s.embedder != nil {
		if vec, err := s.embed(ctx, entry.Prompt); err != nil {
			log.Printf("savedresp: embed failed (saving without vector): %v", err)
		} else {
			entry.Embedding = encodeVec(vec)
		}
	}
	if err := s.q.Insert(ctx, entry); err != nil {
		log.Printf("savedresp: insert failed: %v", err)
	}
}

func (s *Saver) embed(ctx context.Context, text string) ([]float64, error) {
	resp, err := s.embedder.Embed(ctx, &model.EmbeddingRequest{
		Model:        "gobed",
		Input:        text,
		LongTextMode: "truncate",
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errNoEmbedding
	}
	return resp.Data[0].Embedding, nil
}
