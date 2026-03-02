package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

// Engine handles balance checks and deductions.
type Engine struct {
	pricing    *PricingTable
	credits    *queries.CreditQueries
	autoTopup  *AutoTopupService
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
		return ErrInsufficientBalance
	}

	if balance < estimatedCost {
		return ErrInsufficientBalance
	}
	return nil
}

// Deduct calculates actual cost and atomically deducts from balance.
func (e *Engine) Deduct(ctx context.Context, userID, modelID string, inputTokens, outputTokens int, usageLogID string) (int64, error) {
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
		fmt.Sprintf("Model: %s, in: %d, out: %d", modelID, inputTokens, outputTokens),
		refID,
	)
	if err != nil {
		return 0, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// DeductImage calculates image generation cost and atomically deducts from balance.
func (e *Engine) DeductImage(ctx context.Context, userID, modelID string, imageCount int, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateImageCost(modelID, imageCount)
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
		fmt.Sprintf("Image: %s, count: %d", modelID, imageCount),
		refID,
	)
	if err != nil {
		return 0, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// DeductVideo calculates video generation cost and atomically deducts from balance.
func (e *Engine) DeductVideo(ctx context.Context, userID, modelID string, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateVideoCost(modelID)
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
		return 0, err
	}
	e.triggerAutoTopup(userID)
	return cost, nil
}

// Deposit adds credits to a user's balance.
func (e *Engine) Deposit(ctx context.Context, userID string, amountCents int64, description string) error {
	return e.credits.Deposit(ctx, userID, amountCents, description)
}

// GetBalance returns the user's current balance.
func (e *Engine) GetBalance(ctx context.Context, userID string) (int64, error) {
	return e.credits.GetBalance(ctx, userID)
}
