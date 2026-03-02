package handler

import (
	"bytes"
	gocontext "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/storage"
)

type FineTuneHandler struct {
	ftQ       *queries.FineTuneQueries
	providers map[string]provider.FineTuneProvider
	store     storage.Store
}

func NewFineTuneHandler(ftQ *queries.FineTuneQueries, providers map[string]provider.FineTuneProvider, store storage.Store) *FineTuneHandler {
	return &FineTuneHandler{ftQ: ftQ, providers: providers, store: store}
}

func (h *FineTuneHandler) HandleUploadFile(ctx *fasthttp.RequestCtx) {
	userID := getUserID(ctx)
	if userID == "" {
		writeError(ctx, 401, "auth_error", "unauthorized")
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "multipart form required")
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		writeError(ctx, 400, "invalid_request", "file field required")
		return
	}

	purpose := "fine-tune"
	if v := form.Value["purpose"]; len(v) > 0 {
		purpose = v[0]
	}

	fh := files[0]
	f, err := fh.Open()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "open file: "+err.Error())
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		writeError(ctx, 400, "invalid_request", "read file: "+err.Error())
		return
	}

	storageURL, err := h.store.Upload(ctx, fh.Filename, "application/jsonl", bytes.NewReader(data))
	if err != nil {
		storageURL = ""
	}

	ftFile := &queries.FineTuneFile{
		UserID:   userID,
		Filename: fh.Filename,
		Purpose:  purpose,
		Bytes:    int64(len(data)),
		StorageURL: storageURL,
	}
	if err := h.ftQ.CreateFile(ctx, ftFile); err != nil {
		log.Printf("finetune create file: %v", err)
		writeError(ctx, 500, "server_error", "failed to create file record")
		return
	}

	writeJSON(ctx, 200, model.FineTuneFile{
		ID:        ftFile.ID,
		Object:    "file",
		Filename:  ftFile.Filename,
		Purpose:   ftFile.Purpose,
		Bytes:     ftFile.Bytes,
		CreatedAt: ftFile.CreatedAt.Unix(),
	})
}

func (h *FineTuneHandler) HandleListFiles(ctx *fasthttp.RequestCtx) {
	userID := getUserID(ctx)
	files, err := h.ftQ.ListFilesByUser(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to list files")
		return
	}
	var data []model.FineTuneFile
	for _, f := range files {
		data = append(data, model.FineTuneFile{
			ID:        f.ID,
			Object:    "file",
			Filename:  f.Filename,
			Purpose:   f.Purpose,
			Bytes:     f.Bytes,
			CreatedAt: f.CreatedAt.Unix(),
		})
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "data": data})
}

func (h *FineTuneHandler) HandleCreateJob(ctx *fasthttp.RequestCtx) {
	userID := getUserID(ctx)
	if userID == "" {
		writeError(ctx, 401, "auth_error", "unauthorized")
		return
	}

	var req model.FineTuneJobRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model required")
		return
	}
	if req.TrainingFile == "" {
		writeError(ctx, 400, "invalid_request", "training_file required")
		return
	}

	prov := h.resolveProvider(req.Model)
	if prov == nil {
		writeError(ctx, 400, "invalid_request", "no fine-tuning provider for model: "+req.Model)
		return
	}

	hpJSON, _ := json.Marshal(req.Hyperparameters)

	job := &queries.FineTuneJob{
		UserID:          userID,
		Provider:        prov.Name(),
		Model:           req.Model,
		Status:          "pending",
		TrainingFile:    req.TrainingFile,
		ValidationFile:  nilIfEmpty(req.ValidationFile),
		Hyperparameters: hpJSON,
	}
	if err := h.ftQ.CreateJob(ctx, job); err != nil {
		log.Printf("finetune create job: %v", err)
		writeError(ctx, 500, "server_error", "failed to create job")
		return
	}

	h.ftQ.AddEvent(ctx, &queries.FineTuneEvent{
		JobID:   job.ID,
		Type:    "message",
		Level:   "info",
		Message: "Job created, uploading training data to " + prov.Name(),
		Data:    json.RawMessage("{}"),
	})

	go h.processJob(job.ID, prov, &req)

	writeJSON(ctx, 200, jobToAPI(job))
}

func (h *FineTuneHandler) HandleListJobs(ctx *fasthttp.RequestCtx) {
	userID := getUserID(ctx)
	jobs, err := h.ftQ.ListJobsByUser(ctx, userID, 20)
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to list jobs")
		return
	}
	var data []model.FineTuneJob
	for _, j := range jobs {
		data = append(data, *jobToAPI(j))
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "data": data, "has_more": false})
}

func (h *FineTuneHandler) HandleGetJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id required")
		return
	}

	job, err := h.ftQ.GetJob(ctx, jobID)
	if err != nil {
		writeError(ctx, 404, "not_found", "job not found")
		return
	}

	if job.Status == "running" || job.Status == "queued" || job.Status == "pending" {
		h.syncJobStatus(ctx, job)
	}

	writeJSON(ctx, 200, jobToAPI(job))
}

func (h *FineTuneHandler) HandleCancelJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	job, err := h.ftQ.GetJob(ctx, jobID)
	if err != nil {
		writeError(ctx, 404, "not_found", "job not found")
		return
	}

	prov, ok := h.providers[job.Provider]
	if !ok {
		writeError(ctx, 400, "invalid_request", "provider not available")
		return
	}

	if job.ProviderJobID != "" {
		if err := prov.CancelFineTuneJob(ctx, job.ProviderJobID); err != nil {
			log.Printf("finetune cancel %s: %v", job.ProviderJobID, err)
		}
	}

	cancelled := "cancelled"
	now := time.Now()
	h.ftQ.UpdateJobStatus(ctx, jobID, cancelled, nil, 0, nil, &now)
	job.Status = cancelled
	writeJSON(ctx, 200, jobToAPI(job))
}

func (h *FineTuneHandler) HandleListEvents(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	events, err := h.ftQ.ListEvents(ctx, jobID)
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to list events")
		return
	}
	var data []model.FineTuneEvent
	for _, e := range events {
		data = append(data, model.FineTuneEvent{
			ID:        e.ID,
			Object:    "fine_tuning.event",
			Type:      e.Type,
			Level:     e.Level,
			Message:   e.Message,
			CreatedAt: e.CreatedAt.Unix(),
		})
	}
	writeJSON(ctx, 200, map[string]any{"object": "list", "data": data})
}

func (h *FineTuneHandler) fetchFileData(ctx gocontext.Context, storageURL string) ([]byte, error) {
	if storageURL == "" {
		return nil, fmt.Errorf("no storage URL")
	}
	resp, err := http.Get(storageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (h *FineTuneHandler) processJob(jobID string, prov provider.FineTuneProvider, req *model.FineTuneJobRequest) {
	ctx := gocontext.Background()

	file, err := h.ftQ.GetFile(ctx, req.TrainingFile)
	if err != nil {
		h.failJob(ctx, jobID, "training file not found: "+err.Error())
		return
	}

	fileData, err := h.fetchFileData(ctx, file.StorageURL)
	if err != nil {
		h.failJob(ctx, jobID, "fetch training file: "+err.Error())
		return
	}

	provFileID, err := prov.UploadFineTuneFile(ctx, file.Filename, fileData)
	if err != nil {
		h.failJob(ctx, jobID, "upload to provider failed: "+err.Error())
		return
	}

	h.ftQ.AddEvent(ctx, &queries.FineTuneEvent{
		JobID: jobID, Type: "message", Level: "info",
		Message: "Training file uploaded to provider (id=" + provFileID + ")",
		Data:    json.RawMessage("{}"),
	})

	var valFileID string
	if req.ValidationFile != "" {
		valFile, err := h.ftQ.GetFile(ctx, req.ValidationFile)
		if err == nil {
			if valData, err := h.fetchFileData(ctx, valFile.StorageURL); err == nil {
				valFileID, _ = prov.UploadFineTuneFile(ctx, valFile.Filename, valData)
			}
		}
	}

	provJob, err := prov.CreateFineTuneJob(ctx, req, provFileID, valFileID)
	if err != nil {
		h.failJob(ctx, jobID, "create job on provider failed: "+err.Error())
		return
	}

	h.ftQ.UpdateProviderJobID(ctx, jobID, provJob.ProviderJobID)
	h.ftQ.UpdateJobStatus(ctx, jobID, provJob.Status, nilIfEmpty(provJob.FineTunedModel), provJob.TrainedTokens, nilIfEmpty(provJob.Error), provJob.FinishedAt)

	h.ftQ.AddEvent(ctx, &queries.FineTuneEvent{
		JobID: jobID, Type: "message", Level: "info",
		Message: "Job submitted to " + prov.Name() + " (id=" + provJob.ProviderJobID + ", status=" + provJob.Status + ")",
		Data:    json.RawMessage("{}"),
	})
}

func (h *FineTuneHandler) syncJobStatus(ctx *fasthttp.RequestCtx, job *queries.FineTuneJob) {
	if job.ProviderJobID == "" {
		return
	}
	prov, ok := h.providers[job.Provider]
	if !ok {
		return
	}
	provJob, err := prov.GetFineTuneJob(ctx, job.ProviderJobID)
	if err != nil {
		return
	}
	if provJob.Status != job.Status {
		h.ftQ.UpdateJobStatus(ctx, job.ID, provJob.Status,
			nilIfEmpty(provJob.FineTunedModel), provJob.TrainedTokens,
			nilIfEmpty(provJob.Error), provJob.FinishedAt)
		job.Status = provJob.Status
		if provJob.FineTunedModel != "" {
			job.FineTunedModel = &provJob.FineTunedModel
		}
		job.TrainedTokens = provJob.TrainedTokens
	}
}

func (h *FineTuneHandler) failJob(ctx gocontext.Context, jobID, msg string) {
	now := time.Now()
	h.ftQ.UpdateJobStatus(ctx, jobID, "failed", nil, 0, &msg, &now)
	h.ftQ.AddEvent(ctx, &queries.FineTuneEvent{
		JobID: jobID, Type: "error", Level: "error", Message: msg,
		Data: json.RawMessage("{}"),
	})
}

func (h *FineTuneHandler) resolveProvider(modelName string) provider.FineTuneProvider {
	if p, ok := h.providers["mistral"]; ok {
		for _, prefix := range []string{"mistral", "codestral", "open-mistral", "pixtral", "ministral", "devstral", "magistral"} {
			if len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
				return p
			}
		}
	}
	if p, ok := h.providers["openai"]; ok {
		for _, prefix := range []string{"gpt-", "ft:", "davinci", "babbage"} {
			if len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
				return p
			}
		}
	}
	if p, ok := h.providers["together"]; ok {
		if _, ok2 := h.providers["mistral"]; !ok2 {
			return p
		}
		return p
	}
	for _, p := range h.providers {
		return p
	}
	return nil
}

func jobToAPI(j *queries.FineTuneJob) *model.FineTuneJob {
	var resultFiles []string
	json.Unmarshal(j.ResultFiles, &resultFiles)
	if resultFiles == nil {
		resultFiles = []string{}
	}

	var hp *model.Hyperparams
	json.Unmarshal(j.Hyperparameters, &hp)

	ftj := &model.FineTuneJob{
		ID:              j.ID,
		Object:          "fine_tuning.job",
		Model:           j.Model,
		FineTunedModel:  j.FineTunedModel,
		Provider:        j.Provider,
		Status:          j.Status,
		TrainingFile:    j.TrainingFile,
		ValidationFile:  j.ValidationFile,
		Hyperparameters: hp,
		ResultFiles:     resultFiles,
		TrainedTokens:   j.TrainedTokens,
		Error:           j.ErrorMessage,
		CreatedAt:       j.CreatedAt.Unix(),
	}
	if j.FinishedAt != nil {
		ts := j.FinishedAt.Unix()
		ftj.FinishedAt = &ts
	}
	return ftj
}

func getUserID(ctx *fasthttp.RequestCtx) string {
	if v := ctx.UserValue("user_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

