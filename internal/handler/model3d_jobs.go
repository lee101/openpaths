package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

const model3DJobTTL = 24 * time.Hour

type model3DJobStatus string

const (
	model3DJobQueued    model3DJobStatus = "queued"
	model3DJobRunning   model3DJobStatus = "running"
	model3DJobCompleted model3DJobStatus = "completed"
	model3DJobFailed    model3DJobStatus = "failed"
)

type model3DExecutionResult struct {
	Response     *model.Model3DGenerationResponse
	StatusCode   int
	ErrorType    string
	ErrorMessage string
}

type model3DJob struct {
	ID         string
	Key        string
	Status     model3DJobStatus
	Request    model.Model3DGenerationRequest
	Result     *model.Model3DGenerationResponse
	StatusCode int
	ErrorType  string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Done       chan struct{}
}

type model3DJobCache struct {
	mu     sync.Mutex
	byID   map[string]*model3DJob
	byHash map[string]string
}

func newModel3DJobCache() *model3DJobCache {
	return &model3DJobCache{
		byID:   make(map[string]*model3DJob),
		byHash: make(map[string]string),
	}
}

func (c *model3DJobCache) getOrCreate(req model.Model3DGenerationRequest) (*model3DJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(time.Now())
	key := model3DRequestCacheKey(req)
	if id, ok := c.byHash[key]; ok {
		if job := c.byID[id]; job != nil {
			return cloneModel3DJob(job), true
		}
	}

	now := time.Now().UTC()
	job := &model3DJob{
		ID:        newModel3DJobID(),
		Key:       key,
		Status:    model3DJobQueued,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
		Done:      make(chan struct{}),
	}
	c.byID[job.ID] = job
	c.byHash[key] = job.ID
	return cloneModel3DJob(job), false
}

func (c *model3DJobCache) get(id string) (*model3DJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return cloneModel3DJob(job), true
}

func (c *model3DJobCache) markRunning(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if job := c.byID[id]; job != nil {
		job.Status = model3DJobRunning
		job.UpdatedAt = time.Now().UTC()
	}
}

func (c *model3DJobCache) complete(id string, result model3DExecutionResult) {
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

func (c *model3DJobCache) wait(id string, timeout time.Duration) (*model3DJob, bool) {
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

func (c *model3DJobCache) pruneLocked(now time.Time) {
	for id, job := range c.byID {
		if now.Sub(job.UpdatedAt) <= model3DJobTTL {
			continue
		}
		delete(c.byID, id)
		if c.byHash[job.Key] == id {
			delete(c.byHash, job.Key)
		}
	}
}

func cloneModel3DJob(job *model3DJob) *model3DJob {
	cp := *job
	return &cp
}

func model3DRequestCacheKey(req model.Model3DGenerationRequest) string {
	req.Async = false
	body, _ := json.Marshal(req)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func newModel3DJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "3djob_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("3djob_%d", time.Now().UnixNano())
}

func model3DJobHTTPStatus(job *model3DJob) int {
	if job.StatusCode >= 400 {
		return job.StatusCode
	}
	return 502
}

func model3DJobPayload(job *model3DJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "3d.generation.job",
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
		payload["status_code"] = model3DJobHTTPStatus(job)
	}
	return payload
}

func durableModel3DJobPayload(job *queries.Model3DJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "3d.generation.job",
		"status":     job.Status,
		"cached":     cached,
		"created_at": job.CreatedAt.Format(time.RFC3339),
		"updated_at": job.UpdatedAt.Format(time.RFC3339),
	}
	if job.Status == string(model3DJobCompleted) {
		if resp, err := durableModel3DJobResult(job); err == nil {
			payload["result"] = resp
		}
	}
	if job.Status == string(model3DJobFailed) {
		payload["error"] = map[string]string{
			"type":    stringPtrValue(job.ErrorType, "provider_error"),
			"message": stringPtrValue(job.ErrorMessage, "3D generation failed"),
		}
		payload["status_code"] = 502
	}
	return payload
}

func durableModel3DJobResult(job *queries.Model3DJob) (*model.Model3DGenerationResponse, error) {
	var resp model.Model3DGenerationResponse
	if err := json.Unmarshal(job.ResultJSON, &resp); err != nil {
		return nil, err
	}
	if resp.ModelGLB.URL == "" {
		return nil, fmt.Errorf("missing model_glb.url")
	}
	return &resp, nil
}
