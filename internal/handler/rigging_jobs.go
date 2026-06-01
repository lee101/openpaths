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

const riggingJobTTL = 24 * time.Hour

type riggingExecutionResult struct {
	Response     *model.MeshRiggingResponse
	StatusCode   int
	ErrorType    string
	ErrorMessage string
}

type riggingJob struct {
	ID         string
	Key        string
	Status     model3DJobStatus
	Request    model.MeshRiggingRequest
	Result     *model.MeshRiggingResponse
	StatusCode int
	ErrorType  string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Done       chan struct{}
}

type riggingJobCache struct {
	mu     sync.Mutex
	byID   map[string]*riggingJob
	byHash map[string]string
}

func newRiggingJobCache() *riggingJobCache {
	return &riggingJobCache{
		byID:   make(map[string]*riggingJob),
		byHash: make(map[string]string),
	}
}

func (c *riggingJobCache) getOrCreate(req model.MeshRiggingRequest) (*riggingJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(time.Now())
	key := riggingRequestCacheKey(req)
	if id, ok := c.byHash[key]; ok {
		if job := c.byID[id]; job != nil {
			return cloneRiggingJob(job), true
		}
	}

	now := time.Now().UTC()
	job := &riggingJob{
		ID:        newRiggingJobID(),
		Key:       key,
		Status:    model3DJobQueued,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
		Done:      make(chan struct{}),
	}
	c.byID[job.ID] = job
	c.byHash[key] = job.ID
	return cloneRiggingJob(job), false
}

func (c *riggingJobCache) get(id string) (*riggingJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return cloneRiggingJob(job), true
}

func (c *riggingJobCache) markRunning(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if job := c.byID[id]; job != nil {
		job.Status = model3DJobRunning
		job.UpdatedAt = time.Now().UTC()
	}
}

func (c *riggingJobCache) complete(id string, result riggingExecutionResult) {
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

func (c *riggingJobCache) wait(id string, timeout time.Duration) (*riggingJob, bool) {
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

func (c *riggingJobCache) pruneLocked(now time.Time) {
	for id, job := range c.byID {
		if now.Sub(job.UpdatedAt) <= riggingJobTTL {
			continue
		}
		delete(c.byID, id)
		if c.byHash[job.Key] == id {
			delete(c.byHash, job.Key)
		}
	}
}

func cloneRiggingJob(job *riggingJob) *riggingJob {
	cp := *job
	return &cp
}

func riggingRequestCacheKey(req model.MeshRiggingRequest) string {
	req.Async = false
	body, _ := json.Marshal(req)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func newRiggingJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "rigjob_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("rigjob_%d", time.Now().UnixNano())
}

func riggingJobHTTPStatus(job *riggingJob) int {
	if job.StatusCode >= 400 {
		return job.StatusCode
	}
	return 502
}

func riggingJobPayload(job *riggingJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "3d.rigging.job",
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
		payload["status_code"] = riggingJobHTTPStatus(job)
	}
	return payload
}
