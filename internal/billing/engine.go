package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

// Engine handles balance checks and deductions.
type Engine struct {
	pricing   *PricingTable
	credits   *queries.CreditQueries
	autoTopup *AutoTopupService
}

func NewEngine(pricing *PricingTable, credits *queries.CreditQueries) *Engine {
	return &Engine{pricing: pricing, credits: credits}
}

func (e *Engine) SetAutoTopup(svc *AutoTopupService) {
	e.autoTopup = svc
}

func (e *Engine) triggerAutoTopup(userID string) {
	if e.autoTopup != nil {
		e.autoTopup.CheckAndTopup(userID)
	}
}

// PreCheck verifies the user has a minimum balance to attempt a request.
func (e *Engine) PreCheck(ctx context.Context, userID, modelID string, estimatedMaxOutput int) error {
	balance, err := e.credits.GetBalance(ctx, userID)
	if err != nil {
		return fmt.Errorf("balance lookup: %w", err)
	}

	estimatedCost, err := e.pricing.EstimateMaxCost(modelID, estimatedMaxOutput)
	if err != nil {
		// Unknown model pricing; allow if balance is positive
		if balance > 0 {
			return nil
		}
		e.triggerAutoTopup(userID)
		return ErrInsufficientBalance
	}

	if balance < estimatedCost {
		e.triggerAutoTopup(userID)
		return ErrInsufficientBalance
	}
	return nil
}

// PreCheckFixed verifies the user can cover a precomputed request cost.
func (e *Engine) PreCheckFixed(ctx context.Context, userID string, estimatedCost int64) error {
	if estimatedCost <= 0 {
		return nil
	}
	balance, err := e.credits.GetBalance(ctx, userID)
	if err != nil {
		return fmt.Errorf("balance lookup: %w", err)
	}
	if balance < estimatedCost {
		e.triggerAutoTopup(userID)
		return ErrInsufficientBalance
	}
	return nil
}

// Deduct calculates actual cost and atomically deducts from balance.
func (e *Engine) Deduct(ctx context.Context, userID, modelID string, inputTokens, outputTokens int, reasoningEffort, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateCost(modelID, inputTokens, outputTokens)
	if err != nil {
		return 0, err
	}

	if cost == 0 {
		return 0, nil
	}

	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}

	err = e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		formatUsageDescription(modelID, inputTokens, outputTokens, reasoningEffort),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// DeductImage calculates image generation cost and atomically deducts from balance.
func (e *Engine) DeductImage(ctx context.Context, userID, modelID string, imageCount int, usageLogID string) (int64, error) {
	return e.DeductImageWithInputs(ctx, userID, modelID, imageCount, 0, usageLogID)
}

func (e *Engine) DeductImageWithInputs(ctx context.Context, userID, modelID string, outputImageCount, inputImageCount int, usageLogID string) (int64, error) {
	return e.DeductImageWithInputsAndSize(ctx, userID, modelID, outputImageCount, inputImageCount, "", usageLogID)
}

func (e *Engine) DeductImageWithInputsAndSize(ctx context.Context, userID, modelID string, outputImageCount, inputImageCount int, size, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateImageCostWithInputsAndSize(modelID, outputImageCount, inputImageCount, size)
	if err != nil {
		return 0, err
	}
	if cost == 0 {
		return 0, nil
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	err = e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		formatImageUsageDescription(modelID, outputImageCount, inputImageCount, size),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

func (e *Engine) DeductOutpaint(ctx context.Context, userID, modelID string, inputWidth, inputHeight, outputWidth, outputHeight, outputImageCount int, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateOutpaintCost(modelID, inputWidth, inputHeight, outputWidth, outputHeight, outputImageCount)
	if err != nil {
		return 0, err
	}
	if cost == 0 {
		return 0, nil
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	err = e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		formatOutpaintUsageDescription(modelID, inputWidth, inputHeight, outputWidth, outputHeight, outputImageCount),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

func formatImageUsageDescription(modelID string, outputImageCount, inputImageCount int, size string) string {
	parts := []string{
		fmt.Sprintf("Image: %s", modelID),
		fmt.Sprintf("outputs: %d", outputImageCount),
		fmt.Sprintf("inputs: %d", inputImageCount),
	}
	if size != "" {
		parts = append(parts, fmt.Sprintf("size: %s", size))
	}
	return strings.Join(parts, ", ")
}

func formatOutpaintUsageDescription(modelID string, inputWidth, inputHeight, outputWidth, outputHeight, outputImageCount int) string {
	return strings.Join([]string{
		fmt.Sprintf("Image: %s", modelID),
		fmt.Sprintf("outputs: %d", outputImageCount),
		fmt.Sprintf("input: %dx%d", inputWidth, inputHeight),
		fmt.Sprintf("output: %dx%d", outputWidth, outputHeight),
	}, ", ")
}

// DeductVideo calculates video generation cost and atomically deducts from balance.
func (e *Engine) DeductVideo(ctx context.Context, userID, modelID string, durationSeconds int, hasVideoInput bool, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateVideoCost(modelID, durationSeconds, hasVideoInput)
	if err != nil {
		return 0, err
	}
	if cost == 0 {
		return 0, nil
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	err = e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		fmt.Sprintf("Video: %s", modelID),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// DeductAudio calculates audio/transcription cost and atomically deducts from balance.
func (e *Engine) DeductAudio(ctx context.Context, userID, modelID string, durationSeconds int, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateAudioCost(modelID, durationSeconds)
	if err != nil {
		return 0, err
	}
	if cost == 0 {
		return 0, nil
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	err = e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		fmt.Sprintf("Audio: %s, duration_seconds: %d", modelID, durationSeconds),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// DeductFixed atomically deducts a precomputed usage charge in hundredths-of-a-cent.
func (e *Engine) DeductFixed(ctx context.Context, userID, modelID string, cost int64, description, usageLogID string) error {
	if cost <= 0 {
		return nil
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	if description == "" {
		description = fmt.Sprintf("Usage: %s", modelID)
	}
	err := e.credits.DeductWithTransaction(ctx, userID, cost,
		model.TxTypeUsageDeduction,
		description,
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return err
	}
	e.triggerAutoTopup(userID)
	return nil
}

// Deposit adds credits to a user's balance.
func (e *Engine) Deposit(ctx context.Context, userID string, amountCents int64, description string) error {
	return e.credits.Deposit(ctx, userID, amountCents, description)
}

// GetBalance returns the user's current balance.
func (e *Engine) GetBalance(ctx context.Context, userID string) (int64, error) {
	return e.credits.GetBalance(ctx, userID)
}

func formatUsageDescription(modelID string, inputTokens, outputTokens int, reasoningEffort string) string {
	parts := []string{fmt.Sprintf("Model: %s", modelID)}
	if reasoningEffort != "" {
		parts = append(parts, fmt.Sprintf("reasoning: %s", reasoningEffort))
	}
	parts = append(parts,
		fmt.Sprintf("in: %d", inputTokens),
		fmt.Sprintf("out: %d", outputTokens),
	)
	return strings.Join(parts, ", ")
}
