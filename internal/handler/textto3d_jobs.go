package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

const textTo3DJobTTL = 24 * time.Hour

type textTo3DExecutionResult struct {
	Response     *model.TextTo3DGenerationResponse
	StatusCode   int
	ErrorType    string
	ErrorMessage string
}

type textTo3DJob struct {
	ID         string
	Key        string
	Status     model3DJobStatus
	Request    model.TextTo3DGenerationRequest
	Result     *model.TextTo3DGenerationResponse
	StatusCode int
	ErrorType  string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Done       chan struct{}
}

type textTo3DJobCache struct {
	mu     sync.Mutex
	byID   map[string]*textTo3DJob
	byHash map[string]string
}

func newTextTo3DJobCache() *textTo3DJobCache {
	return &textTo3DJobCache{
		byID:   make(map[string]*textTo3DJob),
		byHash: make(map[string]string),
	}
}

func (c *textTo3DJobCache) getOrCreate(req model.TextTo3DGenerationRequest) (*textTo3DJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(time.Now())
	key := textTo3DRequestCacheKey(req)
	if id, ok := c.byHash[key]; ok {
		if job := c.byID[id]; job != nil {
			return cloneTextTo3DJob(job), true
		}
	}

	now := time.Now().UTC()
	job := &textTo3DJob{
		ID:        newTextTo3DJobID(),
		Key:       key,
		Status:    model3DJobQueued,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
		Done:      make(chan struct{}),
	}
	c.byID[job.ID] = job
	c.byHash[key] = job.ID
	return cloneTextTo3DJob(job), false
}

func (c *textTo3DJobCache) get(id string) (*textTo3DJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return cloneTextTo3DJob(job), true
}

func (c *textTo3DJobCache) markRunning(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if job := c.byID[id]; job != nil {
		job.Status = model3DJobRunning
		job.UpdatedAt = time.Now().UTC()
	}
}

func (c *textTo3DJobCache) complete(id string, result textTo3DExecutionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job := c.byID[id]
	if job == nil {
		return
	}
	job.UpdatedAt = time.Now().UTC()
	if result.Response != nil && result.StatusCode < 400 {
		job.Status = model3DJobCompleted
		job.Result = result.Response
	} else {
		job.Status = model3DJobFailed
		job.StatusCode = result.StatusCode
		job.ErrorType = result.ErrorType
		if job.ErrorType == "" {
			job.ErrorType = "provider_error"
		}
		job.Error = result.ErrorMessage
		if c.byHash[job.Key] == id {
			delete(c.byHash, job.Key)
		}
	}
	close(job.Done)
}

func (c *textTo3DJobCache) wait(id string, timeout time.Duration) (*textTo3DJob, bool) {
	c.mu.Lock()
	job := c.byID[id]
	if job == nil {
		c.mu.Unlock()
		return nil, false
	}
	done := job.Done
	c.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return c.get(id)
	case <-timer.C:
		job, ok := c.get(id)
		return job, ok
	}
}

func (c *textTo3DJobCache) pruneLocked(now time.Time) {
	for id, job := range c.byID {
		if now.Sub(job.UpdatedAt) <= textTo3DJobTTL {
			continue
		}
		delete(c.byID, id)
		if c.byHash[job.Key] == id {
			delete(c.byHash, job.Key)
		}
	}
}

func cloneTextTo3DJob(job *textTo3DJob) *textTo3DJob {
	cp := *job
	return &cp
}

func textTo3DRequestCacheKey(req model.TextTo3DGenerationRequest) string {
	req.Async = false
	body, _ := json.Marshal(req)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func newTextTo3DJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "t3djob_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("t3djob_%d", time.Now().UnixNano())
}

func textTo3DJobHTTPStatus(job *textTo3DJob) int {
	if job.StatusCode >= 400 {
		return job.StatusCode
	}
	return 502
}

func textTo3DJobPayload(job *textTo3DJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "3d.text_generation.job",
		"status":     job.Status,
		"cached":     cached,
		"created_at": job.CreatedAt.Format(time.RFC3339),
		"updated_at": job.UpdatedAt.Format(time.RFC3339),
	}
	if job.Status == model3DJobCompleted && job.Result != nil {
		payload["result"] = job.Result
	}
	if job.Status == model3DJobFailed {
		payload["error"] = map[string]string{
			"type":    job.ErrorType,
			"message": job.Error,
		}
		payload["status_code"] = textTo3DJobHTTPStatus(job)
	}
	return payload
}
