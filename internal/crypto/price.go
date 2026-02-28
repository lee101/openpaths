package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	bagsAPIBaseURL = "https://public-api-v2.bags.fm/api/v1"
	solMint        = "So11111111111111111111111111111111111111112"
)

type PriceFeed struct {
	solPrice      float64
	solMu         sync.RWMutex
	codexPrice    float64
	codexMu       sync.RWMutex
	codexMint     string
	bagsAPIKey    string
	stopCh        chan struct{}
}

func NewPriceFeed(codexMint, bagsAPIKey string) *PriceFeed {
	return &PriceFeed{
		solPrice:   100.0,
		codexPrice: 0.001,
		codexMint:  codexMint,
		bagsAPIKey: bagsAPIKey,
		stopCh:     make(chan struct{}),
	}
}

func (p *PriceFeed) Start() {
	go p.solPriceLoop()
	go p.codexPriceLoop()
}

func (p *PriceFeed) Stop() {
	close(p.stopCh)
}

func (p *PriceFeed) SOLPrice() float64 {
	p.solMu.RLock()
	defer p.solMu.RUnlock()
	return p.solPrice
}

func (p *PriceFeed) CODEXPrice() float64 {
	p.codexMu.RLock()
	defer p.codexMu.RUnlock()
	return p.codexPrice
}

func (p *PriceFeed) CODEXMint() string {
	return p.codexMint
}

func (p *PriceFeed) solPriceLoop() {
	p.updateSOLPrice()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.updateSOLPrice()
		case <-p.stopCh:
			return
		}
	}
}

func (p *PriceFeed) updateSOLPrice() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd", nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var result struct {
		Solana struct {
			USD float64 `json:"usd"`
		} `json:"solana"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return
	}
	if result.Solana.USD > 0 {
		p.solMu.Lock()
		p.solPrice = result.Solana.USD
		p.solMu.Unlock()
		log.Printf("SOL price: $%.2f", result.Solana.USD)
	}
}

func (p *PriceFeed) codexPriceLoop() {
	p.updateCODEXPrice()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.updateCODEXPrice()
		case <-p.stopCh:
			return
		}
	}
}

func (p *PriceFeed) updateCODEXPrice() {
	if p.codexMint == "" || p.bagsAPIKey == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testAmount := "100000000" // 0.1 SOL
	quoteURL := fmt.Sprintf("%s/trade/quote?inputMint=%s&outputMint=%s&amount=%s&slippageMode=manual&slippageBps=100",
		bagsAPIBaseURL, solMint, p.codexMint, testAmount)

	req, err := http.NewRequestWithContext(ctx, "GET", quoteURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("x-api-key", p.bagsAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var result struct {
		Success  bool `json:"success"`
		Response struct {
			InAmount  string `json:"inAmount"`
			OutAmount string `json:"outAmount"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		return
	}

	inAmt, err := strconv.ParseFloat(result.Response.InAmount, 64)
	if err != nil || inAmt <= 0 {
		return
	}
	outAmt, err := strconv.ParseFloat(result.Response.OutAmount, 64)
	if err != nil || outAmt <= 0 {
		return
	}

	solPrice := p.SOLPrice()
	codexPerSOL := (outAmt / inAmt) * 10
	newPrice := solPrice / codexPerSOL

	if newPrice > 0 && newPrice < 1000 {
		p.codexMu.Lock()
		p.codexPrice = newPrice
		p.codexMu.Unlock()
		log.Printf("CODEX price: $%.6f (%.0f/SOL)", newPrice, codexPerSOL)
	}
}
