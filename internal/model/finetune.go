package model

import "time"

type FineTuneJobRequest struct {
	Model           string         `json:"model"`
	TrainingFile    string         `json:"training_file"`
	ValidationFile  string         `json:"validation_file,omitempty"`
	Hyperparameters *Hyperparams   `json:"hyperparameters,omitempty"`
	Suffix          string         `json:"suffix,omitempty"`
}

type Hyperparams struct {
	NEpochs        *int     `json:"n_epochs,omitempty"`
	BatchSize      *int     `json:"batch_size,omitempty"`
	LearningRate   *float64 `json:"learning_rate_multiplier,omitempty"`
}

type FineTuneJob struct {
	ID              string      `json:"id"`
	Object          string      `json:"object"`
	Model           string      `json:"model"`
	FineTunedModel  *string     `json:"fine_tuned_model"`
	Provider        string      `json:"provider"`
	Status          string      `json:"status"`
	TrainingFile    string      `json:"training_file"`
	ValidationFile  *string     `json:"validation_file"`
	Hyperparameters *Hyperparams `json:"hyperparameters,omitempty"`
	ResultFiles     []string    `json:"result_files"`
	TrainedTokens   int64       `json:"trained_tokens"`
	Error           *string     `json:"error,omitempty"`
	CreatedAt       int64       `json:"created_at"`
	FinishedAt      *int64      `json:"finished_at,omitempty"`
}

type FineTuneEvent struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Type      string `json:"type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

type FineTuneFile struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
}

type ProviderFineTuneJob struct {
	ProviderJobID  string
	Status         string
	FineTunedModel string
	TrainedTokens  int64
	Error          string
	FinishedAt     *time.Time
}
