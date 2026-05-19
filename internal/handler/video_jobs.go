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

const videoJobTTL = 24 * time.Hour

type videoJobStatus string

const (
	videoJobQueued    videoJobStatus = "queued"
	videoJobRunning   videoJobStatus = "running"
	videoJobCompleted videoJobStatus = "completed"
	videoJobFailed    videoJobStatus = "failed"
)

type videoExecutionResult struct {
	Response     *model.VideoGenerationResponse
	StatusCode   int
	ErrorType    string
	ErrorMessage string
}

type videoJob struct {
	ID         string
	Key        string
	Status     videoJobStatus
	Request    model.VideoGenerationRequest
	Result     *model.VideoGenerationResponse
	StatusCode int
	ErrorType  string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Done       chan struct{}
}

type videoJobCache struct {
	mu     sync.Mutex
	byID   map[string]*videoJob
	byHash map[string]string
}

func newVideoJobCache() *videoJobCache {
	return &videoJobCache{
		byID:   make(map[string]*videoJob),
		byHash: make(map[string]string),
	}
}

func (c *videoJobCache) getOrCreate(req model.VideoGenerationRequest) (*videoJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneLocked(time.Now())
	key := videoRequestCacheKey(req)
	if id, ok := c.byHash[key]; ok {
		if job := c.byID[id]; job != nil {
			return cloneVideoJob(job), true
		}
	}

	job := &videoJob{
		ID:        newVideoJobID(),
		Key:       key,
		Status:    videoJobQueued,
		Request:   req,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Done:      make(chan struct{}),
	}
	c.byID[job.ID] = job
	c.byHash[key] = job.ID
	return cloneVideoJob(job), false
}

func (c *videoJobCache) get(id string) (*videoJob, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return cloneVideoJob(job), true
}

func (c *videoJobCache) markRunning(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if job := c.byID[id]; job != nil {
		job.Status = videoJobRunning
		job.UpdatedAt = time.Now().UTC()
	}
}

func (c *videoJobCache) complete(id string, result videoExecutionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job := c.byID[id]
	if job == nil {
		return
	}
	job.UpdatedAt = time.Now().UTC()
	if result.Response != nil && result.StatusCode < 400 {
		job.Status = videoJobCompleted
		job.Result = result.Response
	} else {
		job.Status = videoJobFailed
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

func (c *videoJobCache) wait(id string, timeout time.Duration) (*videoJob, bool) {
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

func (c *videoJobCache) pruneLocked(now time.Time) {
	for id, job := range c.byID {
		if now.Sub(job.UpdatedAt) <= videoJobTTL {
			continue
		}
		delete(c.byID, id)
		if c.byHash[job.Key] == id {
			delete(c.byHash, job.Key)
		}
	}
}

func cloneVideoJob(job *videoJob) *videoJob {
	cp := *job
	return &cp
}

func videoRequestCacheKey(req model.VideoGenerationRequest) string {
	req.Async = false
	body, _ := json.Marshal(req)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func newVideoJobID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "vidjob_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("vidjob_%d", time.Now().UnixNano())
}

func videoJobPayload(job *videoJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "video.generation.job",
		"status":     job.Status,
		"cached":     cached,
		"created_at": job.CreatedAt.Format(time.RFC3339),
		"updated_at": job.UpdatedAt.Format(time.RFC3339),
	}
	if job.Status == videoJobCompleted && job.Result != nil {
		payload["result"] = job.Result
	}
	if job.Status == videoJobFailed {
		payload["error"] = map[string]string{
			"type":    job.ErrorType,
			"message": job.Error,
		}
		payload["status_code"] = job.StatusCode
	}
	return payload
}

func durableVideoJobPayload(job *queries.VideoJob, cached bool) map[string]any {
	payload := map[string]any{
		"id":         job.ID,
		"object":     "video.generation.job",
		"status":     job.Status,
		"cached":     cached,
		"created_at": job.CreatedAt.Format(time.RFC3339),
		"updated_at": job.UpdatedAt.Format(time.RFC3339),
	}
	if job.Status == string(videoJobCompleted) {
		if resp, err := durableVideoJobResult(job); err == nil {
			payload["result"] = resp
		}
	}
	if job.Status == string(videoJobFailed) {
		payload["error"] = map[string]string{
			"type":    stringPtrValue(job.ErrorType, "provider_error"),
			"message": stringPtrValue(job.ErrorMessage, "video generation failed"),
		}
		payload["status_code"] = 502
	}
	return payload
}

func durableVideoJobResult(job *queries.VideoJob) (*model.VideoGenerationResponse, error) {
	var resp model.VideoGenerationResponse
	if err := json.Unmarshal(job.ResultJSON, &resp); err != nil {
		return nil, err
	}
	if resp.VideoURL == "" {
		return nil, fmt.Errorf("missing video_url")
	}
	return &resp, nil
}

func stringPtrValue(s *string, fallback string) string {
	if s != nil && *s != "" {
		return *s
	}
	return fallback
}
