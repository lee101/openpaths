package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	maxPriceAge = 20 * time.Minute
	rateFactor  = 0.97
)

const (
	bagsAPIBaseURL = "https://public-api-v2.bags.fm/api/v1"
	solMint        = "So11111111111111111111111111111111111111112"
)

type PriceFeed struct {
	solPrice   float64
	solAt      time.Time
	solMu      sync.RWMutex
	codexPrice float64
	codexAt    time.Time
	codexMu    sync.RWMutex
	codexMint  string
	bagsAPIKey string
	stopCh     chan struct{}
}

func NewPriceFeed(codexMint, bagsAPIKey string) *PriceFeed {
	return &PriceFeed{
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
	if p.solPrice <= 0 || time.Since(p.solAt) > maxPriceAge {
		return 0
	}
	return p.solPrice * rateFactor
}

func (p *PriceFeed) CODEXPrice() float64 {
	p.codexMu.RLock()
	defer p.codexMu.RUnlock()
	if p.codexPrice <= 0 || time.Since(p.codexAt) > maxPriceAge {
		return 0
	}
	return p.codexPrice * rateFactor
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

func fetchJSONPrice(url string, extract func([]byte) float64) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0
	}
	return extract(body)
}

func (p *PriceFeed) updateSOLPrice() {
	sources := []func() float64{
		func() float64 {
			return fetchJSONPrice("https://api.coinbase.com/v2/prices/SOL-USD/spot", func(b []byte) float64 {
				var r struct {
					Data struct {
						Amount string `json:"amount"`
					} `json:"data"`
				}
				if json.Unmarshal(b, &r) != nil {
					return 0
				}
				v, _ := strconv.ParseFloat(r.Data.Amount, 64)
				return v
			})
		},
		func() float64 {
			return fetchJSONPrice("https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd", func(b []byte) float64 {
				var r struct {
					Solana struct {
						USD float64 `json:"usd"`
					} `json:"solana"`
				}
				if json.Unmarshal(b, &r) != nil {
					return 0
				}
				return r.Solana.USD
			})
		},
		func() float64 {
			return fetchJSONPrice("https://api.binance.com/api/v3/ticker/price?symbol=SOLUSDT", func(b []byte) float64 {
				var r struct {
					Price string `json:"price"`
				}
				if json.Unmarshal(b, &r) != nil {
					return 0
				}
				v, _ := strconv.ParseFloat(r.Price, 64)
				return v
			})
		},
	}
	var got []float64
	for _, f := range sources {
		if v := f(); v > 1 && v < 100000 {
			got = append(got, v)
		}
	}
	if len(got) == 0 {
		log.Printf("SOL price: all sources failed")
		return
	}
	sort.Float64s(got)
	price := got[len(got)/2]
	if len(got) == 2 {
		price = got[0]
	}
	p.solMu.Lock()
	p.solPrice = price
	p.solAt = time.Now()
	p.solMu.Unlock()
	log.Printf("SOL price: $%.2f (%d sources)", price, len(got))
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
		p.codexAt = time.Now()
		p.codexMu.Unlock()
		log.Printf("CODEX price: $%.6f (%.0f/SOL)", newPrice, codexPerSOL)
	}
}
