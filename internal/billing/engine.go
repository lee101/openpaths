package billing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/email"
	"github.com/openpaths/openpaths/internal/model"
)

var ErrInsufficientBalance = errors.New("insufficient balance")

const billingAlertCooldown = 24 * time.Hour

// Engine handles balance checks and deductions.
type Engine struct {
	pricing   *PricingTable
	credits   *queries.CreditQueries
	autoTopup *AutoTopupService
	guards    *queries.GuardQueries
	appURL    string
}

func NewEngine(pricing *PricingTable, credits *queries.CreditQueries) *Engine {
	return &Engine{pricing: pricing, credits: credits}
}

func (e *Engine) SetAutoTopup(svc *AutoTopupService) {
	e.autoTopup = svc
}

// SetGuards wires the optional billshock-alert store (no alerts when unset).
func (e *Engine) SetGuards(g *queries.GuardQueries, appURL string) {
	e.guards = g
	e.appURL = appURL
}

// maybeBillingAlert emails the user when their balance crosses below their
// configured threshold, debounced once per cooldown. Safe to call inline.
func (e *Engine) maybeBillingAlert(ctx context.Context, userID string) {
	if e.guards == nil || e.credits == nil {
		return
	}
	enabled, threshold, lastAt, addr, err := e.guards.AlertState(ctx, userID)
	if err != nil || !enabled || threshold <= 0 || addr == "" {
		return
	}
	if lastAt != nil && time.Since(*lastAt) < billingAlertCooldown {
		return
	}
	balance, err := e.credits.GetBalance(ctx, userID)
	if err != nil {
		return
	}
	// Balance is in internal units (hundredths-of-a-cent); thresholds are USD cents.
	balanceCents := balance / 100
	if balanceCents >= threshold {
		return
	}
	if err := e.guards.MarkAlerted(ctx, userID); err != nil {
		return
	}
	url := e.appURL
	if url == "" {
		url = "https://openpaths.io"
	}
	body := fmt.Sprintf(`<div style="font-family:sans-serif;max-width:480px">
<h2>Low credit balance</h2>
<p>Your balance has dropped to <strong>$%.2f</strong>, below your alert threshold of $%.2f.</p>
<p><a href="%s/account">Top up your account</a> to avoid interruptions.</p>
</div>`, float64(balanceCents)/100, float64(threshold)/100, url)
	go func(to, html string) {
		if err := email.Send(to, "Low credit balance on OpenPaths", html); err != nil {
			log.Printf("billing-alert email send failed user=%s: %v", userID, err)
		}
	}(addr, body)
}

func (e *Engine) triggerAutoTopup(userID string) {
	if e.autoTopup != nil {
		e.autoTopup.CheckAndTopup(userID)
	}
}

// PreCheck verifies the user has a minimum balance to attempt a request.
func (e *Engine) PreCheck(ctx context.Context, userID, modelID string, estimatedMaxOutput int) error {
	if e.credits == nil {
		return nil
	}
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
	if estimatedCost <= 0 || e.credits == nil {
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
	return e.DeductWithCachedInput(ctx, userID, modelID, inputTokens, outputTokens, 0, reasoningEffort, usageLogID)
}

// DeductWithCachedInput calculates actual cost with cache-hit prompt tokens and
// atomically deducts from balance.
func (e *Engine) DeductWithCachedInput(ctx context.Context, userID, modelID string, inputTokens, outputTokens, cachedInputTokens int, reasoningEffort, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateCostWithCachedInput(modelID, inputTokens, outputTokens, cachedInputTokens)
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
		formatUsageDescriptionWithCache(modelID, inputTokens, outputTokens, cachedInputTokens, reasoningEffort),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	e.maybeBillingAlert(ctx, userID)
	return cost, nil
}

func (e *Engine) DeductRealtime(ctx context.Context, userID, modelID string, usage RealtimeUsage, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateRealtimeCost(modelID, usage)
	if err != nil || cost == 0 {
		return cost, err
	}
	var refID *string
	if usageLogID != "" {
		refID = &usageLogID
	}
	err = e.credits.DeductWithTransaction(ctx, userID, cost, model.TxTypeUsageDeduction,
		fmt.Sprintf("Realtime: %s (%d text in, %d audio in, %d image in, %d text out, %d audio out)",
			modelID, usage.TextInputTokens, usage.AudioInputTokens, usage.ImageInputTokens,
			usage.TextOutputTokens, usage.AudioOutputTokens), refID)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	e.maybeBillingAlert(ctx, userID)
	return cost, nil
}

func (e *Engine) RealtimeCost(modelID string, usage RealtimeUsage) (int64, error) {
	return e.pricing.CalculateRealtimeCost(modelID, usage)
}

// ImageCost returns the cost (in hundredths-of-a-cent) of an image request
// without deducting anything. Used for prechecks in multi-step pipelines.
func (e *Engine) ImageCost(modelID string, outputImageCount, inputImageCount int, size string) (int64, error) {
	return e.pricing.CalculateImageCostWithInputsAndSize(modelID, outputImageCount, inputImageCount, size)
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
	e.maybeBillingAlert(ctx, userID)
	return cost, nil
}

// DeductForecast charges for a single time-series forecast request.
func (e *Engine) DeductForecast(ctx context.Context, userID, modelID, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateForecastCost(modelID)
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
		fmt.Sprintf("Forecast: %s", modelID),
		refID,
	)
	if err != nil {
		e.triggerAutoTopup(userID)
		return cost, err
	}
	e.triggerAutoTopup(userID)
	e.maybeBillingAlert(ctx, userID)
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
	e.maybeBillingAlert(ctx, userID)
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

func formatUsageDescriptionWithCache(modelID string, inputTokens, outputTokens, cachedInputTokens int, reasoningEffort string) string {
	desc := formatUsageDescription(modelID, inputTokens, outputTokens, reasoningEffort)
	if cachedInputTokens <= 0 {
		return desc
	}
	return desc + fmt.Sprintf(", cache hit: %d", cachedInputTokens)
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
	return e.DeductVideoWithResolution(ctx, userID, modelID, durationSeconds, hasVideoInput, "", usageLogID)
}

func (e *Engine) DeductVideoWithResolution(ctx context.Context, userID, modelID string, durationSeconds int, hasVideoInput bool, resolution, usageLogID string) (int64, error) {
	return e.DeductVideoWithMediaInputs(ctx, userID, modelID, durationSeconds, hasVideoInput, 0, resolution, usageLogID)
}

func (e *Engine) DeductVideoWithMediaInputs(ctx context.Context, userID, modelID string, durationSeconds int, hasVideoInput bool, inputImageCount int, resolution, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateVideoCostWithMediaInputs(modelID, durationSeconds, hasVideoInput, inputImageCount, resolution)
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
	e.maybeBillingAlert(ctx, userID)
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
	e.maybeBillingAlert(ctx, userID)
	return cost, nil
}

func (e *Engine) DeductCharacters(ctx context.Context, userID, modelID string, characters int, usageLogID string) (int64, error) {
	cost, err := e.pricing.CalculateCharacterCost(modelID, characters)
	if err != nil || cost == 0 {
		return cost, err
	}
	return cost, e.DeductFixed(ctx, userID, modelID, cost, fmt.Sprintf("Speech: %s, characters: %d", modelID, characters), usageLogID)
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
