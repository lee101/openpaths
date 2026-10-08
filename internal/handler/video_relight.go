package handler

import (
	"context"
	"math"
)

const (
	relightMinBillSeconds = 3
	relightMaxBillSeconds = 15
)

func relightBillableSeconds(measured float64) float64 {
	return math.Min(relightMaxBillSeconds, math.Max(relightMinBillSeconds, math.Ceil(measured-1e-9)))
}

func probeRelightOutputDuration(ctx context.Context, videoURL string) (float64, error) {
	return probeRecastOutputDuration(ctx, videoURL)
}
