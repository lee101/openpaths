package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mr-tron/base58"
	"github.com/openpath/openpath/internal/billing"
	"github.com/openpath/openpath/internal/model"
)

type CryptoStore interface {
	NextDepositIndex(ctx context.Context) (int64, error)
	CreateCheckoutIntent(ctx context.Context, intent *model.CryptoCheckoutIntent) error
	GetCheckoutIntent(ctx context.Context, id string) (*model.CryptoCheckoutIntent, error)
	ListPendingCheckouts(ctx context.Context) ([]*model.CryptoCheckoutIntent, error)
	ListUnsweptCheckouts(ctx context.Context) ([]*model.CryptoCheckoutIntent, error)
	UpdateCheckoutStatus(ctx context.Context, id string, status model.CryptoCheckoutStatus, txSig string) error
	MarkCheckoutSwept(ctx context.Context, id string) error
	MarkExpiredCheckouts(ctx context.Context) (int64, error)
}

type Config struct {
	Enabled          bool
	WalletPubkey     string
	HDWalletSeed     []byte
	CodexTokenMint   string
	BagsAPIKey       string
	MinTopupUSD      float64
}

type Service struct {
	cfg       Config
	store     CryptoStore
	billing   *billing.Engine
	rpc       *RPCClient
	prices    *PriceFeed
	stopCh    chan struct{}

	sseSubscribers map[string][]chan string
	sseMu          sync.RWMutex

	intentLastChecked   map[string]time.Time
	intentLastCheckedMu sync.RWMutex
}

func NewService(cfg Config, store CryptoStore, billing *billing.Engine, rpc *RPCClient, prices *PriceFeed) *Service {
	return &Service{
		cfg:               cfg,
		store:             store,
		billing:           billing,
		rpc:               rpc,
		prices:            prices,
		stopCh:            make(chan struct{}),
		sseSubscribers:    make(map[string][]chan string),
		intentLastChecked: make(map[string]time.Time),
	}
}

func (s *Service) Start() {
	go s.paymentPoller()
	go s.expiredCleaner()
	go s.sweeper()
}

func (s *Service) Stop() {
	close(s.stopCh)
}

func (s *Service) Prices() *PriceFeed { return s.prices }

func (s *Service) CreateCheckout(ctx context.Context, userID string, req *model.CryptoCheckoutRequest) (*model.CryptoCheckoutResponse, error) {
	if s.cfg.WalletPubkey == "" {
		return nil, fmt.Errorf("crypto payments not configured")
	}
	if req.Method != model.CryptoMethodSOL && req.Method != model.CryptoMethodCODEX {
		return nil, fmt.Errorf("method must be 'sol' or 'codex'")
	}
	if req.AmountUSD < s.cfg.MinTopupUSD {
		return nil, fmt.Errorf("minimum topup is $%.0f", s.cfg.MinTopupUSD)
	}

	usdAmount := req.AmountUSD
	creditsCents := int64(usdAmount * 10000) // hundredths-of-a-cent per $1

	var amountUI string
	var amountLamports uint64
	var mint string

	if req.Method == model.CryptoMethodSOL {
		solPrice := s.prices.SOLPrice()
		solAmount := usdAmount / solPrice
		amountUI = fmt.Sprintf("%.6f", solAmount)
		amountLamports = uint64(solAmount * 1e9)
	} else {
		if s.cfg.CodexTokenMint == "" {
			return nil, fmt.Errorf("CODEX token not configured")
		}
		mint = s.cfg.CodexTokenMint
		// No discount -- market rate
		codexPrice := s.prices.CODEXPrice()
		codexAmount := usdAmount / codexPrice
		amountUI = fmt.Sprintf("%.2f", codexAmount)
		amountLamports = uint64(codexAmount * 1e9)
	}

	depositIndex, err := s.store.NextDepositIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("deposit index: %w", err)
	}

	depositPubkey, err := GetDepositPubkey(s.cfg.HDWalletSeed, depositIndex)
	if err != nil {
		return nil, fmt.Errorf("derive deposit address: %w", err)
	}

	now := time.Now()
	intent := &model.CryptoCheckoutIntent{
		UserID:         userID,
		Method:         req.Method,
		AmountUSD:      usdAmount,
		AmountLamports: amountLamports,
		AmountUI:       amountUI,
		DepositIndex:   depositIndex,
		DepositPubkey:  depositPubkey,
		Mint:           mint,
		Status:         model.CryptoStatusPending,
		CreditsCents:   creditsCents,
		ExpiresAt:      now.Add(20 * time.Minute),
		HonorUntil:     now.Add(7 * 24 * time.Hour),
	}

	if err := s.store.CreateCheckoutIntent(ctx, intent); err != nil {
		return nil, fmt.Errorf("create checkout: %w", err)
	}

	solanaPayURL := BuildSolanaPayURL(intent)

	log.Printf("crypto checkout: id=%s method=%s usd=%.2f amount=%s deposit=%s",
		intent.ID, intent.Method, usdAmount, amountUI, depositPubkey)

	return &model.CryptoCheckoutResponse{
		IntentID:       intent.ID,
		DepositAddress: depositPubkey,
		SolanaPayURL:   solanaPayURL,
		AmountUI:       amountUI,
		AmountUSD:      usdAmount,
		Method:         string(req.Method),
		TokenMint:      mint,
		ExpiresAt:      intent.ExpiresAt.Format(time.RFC3339),
		CreditsCents:   creditsCents,
	}, nil
}

func (s *Service) GetCheckoutStatus(ctx context.Context, id string) (*model.CryptoCheckoutIntent, error) {
	return s.store.GetCheckoutIntent(ctx, id)
}

// SSE support

func (s *Service) Subscribe(intentID string) chan string {
	ch := make(chan string, 10)
	s.sseMu.Lock()
	s.sseSubscribers[intentID] = append(s.sseSubscribers[intentID], ch)
	s.sseMu.Unlock()
	return ch
}

func (s *Service) Unsubscribe(intentID string, ch chan string) {
	s.sseMu.Lock()
	channels := s.sseSubscribers[intentID]
	for i, c := range channels {
		if c == ch {
			s.sseSubscribers[intentID] = append(channels[:i], channels[i+1:]...)
			break
		}
	}
	if len(s.sseSubscribers[intentID]) == 0 {
		delete(s.sseSubscribers, intentID)
	}
	s.sseMu.Unlock()
	close(ch)
}

func (s *Service) broadcast(intentID string, status model.CryptoCheckoutStatus, txSig string) {
	s.sseMu.RLock()
	channels := s.sseSubscribers[intentID]
	s.sseMu.RUnlock()
	if len(channels) == 0 {
		return
	}
	data, _ := json.Marshal(map[string]string{"status": string(status), "tx_sig": txSig})
	for _, ch := range channels {
		select {
		case ch <- string(data):
		default:
		}
	}
}

// Background workers

const (
	freshPollInterval  = 30 * time.Second
	honorPollInterval  = 1 * time.Hour
	maxBackoffInterval = 5 * time.Minute
)

func (s *Service) paymentPoller() {
	baseInterval := 30 * time.Second
	currentInterval := baseInterval
	for {
		select {
		case <-time.After(currentInterval):
			rateLimited := s.checkPendingPayments()
			if rateLimited {
				currentInterval = min(currentInterval*2, maxBackoffInterval)
			} else {
				currentInterval = baseInterval
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) checkPendingPayments() bool {
	ctx := context.Background()
	intents, err := s.store.ListPendingCheckouts(ctx)
	if err != nil {
		log.Printf("crypto poller: list pending: %v", err)
		return false
	}

	rateLimited := false
	now := time.Now()

	for _, intent := range intents {
		if now.After(intent.HonorUntil) {
			s.store.UpdateCheckoutStatus(ctx, intent.ID, model.CryptoStatusExpired, "")
			s.broadcast(intent.ID, model.CryptoStatusExpired, "")
			s.intentLastCheckedMu.Lock()
			delete(s.intentLastChecked, intent.ID)
			s.intentLastCheckedMu.Unlock()
			continue
		}

		var checkInterval time.Duration
		if now.Before(intent.ExpiresAt) {
			checkInterval = freshPollInterval
		} else {
			checkInterval = honorPollInterval
		}

		s.intentLastCheckedMu.RLock()
		lastCheck, exists := s.intentLastChecked[intent.ID]
		s.intentLastCheckedMu.RUnlock()
		if exists && now.Sub(lastCheck) < checkInterval {
			continue
		}

		txSig, verified, wasRateLimited := s.verifyPaymentOnChain(intent)

		s.intentLastCheckedMu.Lock()
		s.intentLastChecked[intent.ID] = now
		s.intentLastCheckedMu.Unlock()

		if wasRateLimited {
			rateLimited = true
			continue
		}

		if verified {
			s.processPayment(intent, txSig)
			s.intentLastCheckedMu.Lock()
			delete(s.intentLastChecked, intent.ID)
			s.intentLastCheckedMu.Unlock()
		} else if now.After(intent.ExpiresAt) {
			s.broadcast(intent.ID, model.CryptoStatusExpired, "")
		}
	}
	return rateLimited
}

func (s *Service) verifyPaymentOnChain(intent *model.CryptoCheckoutIntent) (string, bool, bool) {
	params := []interface{}{
		intent.DepositPubkey,
		map[string]interface{}{"limit": 10},
	}

	body, err := s.rpc.Call("getSignaturesForAddress", params, 10*time.Second)
	if err != nil {
		return "", false, isRateLimit(err)
	}

	var rpcResp struct {
		Result []struct {
			Signature string      `json:"signature"`
			Err       interface{} `json:"err"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return "", false, false
	}
	if rpcResp.Error != nil {
		rl := rpcResp.Error.Code == 429 || strings.Contains(strings.ToLower(rpcResp.Error.Message), "rate")
		return "", false, rl
	}

	for _, sig := range rpcResp.Result {
		if sig.Err != nil {
			continue
		}
		if s.verifyTxDetails(intent, sig.Signature) {
			return sig.Signature, true, false
		}
	}
	return "", false, false
}

func (s *Service) verifyTxDetails(intent *model.CryptoCheckoutIntent, txSig string) bool {
	params := []interface{}{
		txSig,
		map[string]interface{}{
			"encoding":                       "jsonParsed",
			"maxSupportedTransactionVersion": 0,
		},
	}

	body, err := s.rpc.Call("getTransaction", params, 10*time.Second)
	if err != nil {
		return false
	}

	var txResp struct {
		Result struct {
			Meta struct {
				Err              interface{} `json:"err"`
				PreBalances      []uint64    `json:"preBalances"`
				PostBalances     []uint64    `json:"postBalances"`
				PreTokenBalances []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					UITokenAmount struct {
						Amount string `json:"amount"`
					} `json:"uiTokenAmount"`
				} `json:"preTokenBalances"`
				PostTokenBalances []struct {
					AccountIndex  int    `json:"accountIndex"`
					Mint          string `json:"mint"`
					Owner         string `json:"owner"`
					UITokenAmount struct {
						Amount string `json:"amount"`
					} `json:"uiTokenAmount"`
				} `json:"postTokenBalances"`
			} `json:"meta"`
			Transaction struct {
				Message struct {
					AccountKeys []struct {
						Pubkey string `json:"pubkey"`
					} `json:"accountKeys"`
				} `json:"message"`
			} `json:"transaction"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &txResp); err != nil {
		return false
	}
	if txResp.Result.Meta.Err != nil {
		return false
	}

	targetPubkey := intent.DepositPubkey

	if intent.Mint == "" {
		// SOL transfer
		walletIndex := -1
		for i, key := range txResp.Result.Transaction.Message.AccountKeys {
			if key.Pubkey == targetPubkey {
				walletIndex = i
				break
			}
		}
		if walletIndex == -1 || walletIndex >= len(txResp.Result.Meta.PreBalances) || walletIndex >= len(txResp.Result.Meta.PostBalances) {
			return false
		}
		pre := txResp.Result.Meta.PreBalances[walletIndex]
		post := txResp.Result.Meta.PostBalances[walletIndex]
		if post <= pre {
			return false
		}
		received := post - pre
		minExpected := uint64(float64(intent.AmountLamports) * 0.99)
		return received >= minExpected
	}

	// SPL token transfer
	var preAmount, postAmount uint64
	for _, bal := range txResp.Result.Meta.PreTokenBalances {
		if bal.Mint == intent.Mint && bal.Owner == targetPubkey {
			preAmount, _ = strconv.ParseUint(bal.UITokenAmount.Amount, 10, 64)
			break
		}
	}
	for _, bal := range txResp.Result.Meta.PostTokenBalances {
		if bal.Mint == intent.Mint && bal.Owner == targetPubkey {
			postAmount, _ = strconv.ParseUint(bal.UITokenAmount.Amount, 10, 64)
			break
		}
	}
	if postAmount <= preAmount {
		return false
	}
	received := postAmount - preAmount
	minExpected := uint64(float64(intent.AmountLamports) * 0.99)
	return received >= minExpected
}

func (s *Service) processPayment(intent *model.CryptoCheckoutIntent, txSig string) {
	ctx := context.Background()
	if err := s.store.UpdateCheckoutStatus(ctx, intent.ID, model.CryptoStatusPaid, txSig); err != nil {
		log.Printf("crypto: update status failed: %v", err)
		return
	}
	s.broadcast(intent.ID, model.CryptoStatusPaid, txSig)

	desc := fmt.Sprintf("Crypto %s: $%.2f, tx: %s", intent.Method, intent.AmountUSD, truncate(txSig, 16))
	if err := s.billing.Deposit(ctx, intent.UserID, intent.CreditsCents, desc); err != nil {
		log.Printf("crypto: deposit credits failed: %v", err)
		return
	}

	log.Printf("crypto payment: user=%s credits=%d method=%s tx=%s", intent.UserID, intent.CreditsCents, intent.Method, txSig)
}

func (s *Service) expiredCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if n, err := s.store.MarkExpiredCheckouts(ctx); err != nil {
				log.Printf("crypto: mark expired: %v", err)
			} else if n > 0 {
				log.Printf("crypto: expired %d checkouts", n)
			}
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) sweeper() {
	time.Sleep(30 * time.Second) // initial delay
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		s.sweepDeposits()
		select {
		case <-ticker.C:
		case <-s.stopCh:
			return
		}
	}
}

func (s *Service) sweepDeposits() {
	ctx := context.Background()
	intents, err := s.store.ListUnsweptCheckouts(ctx)
	if err != nil {
		log.Printf("crypto sweep: list unswept: %v", err)
		return
	}
	if len(intents) == 0 {
		return
	}

	for _, intent := range intents {
		if intent.DepositPubkey == "" {
			continue
		}
		balance, err := s.getAccountBalance(intent.DepositPubkey)
		if err != nil {
			continue
		}
		minBalance := uint64(5000)
		if balance <= minBalance {
			s.store.MarkCheckoutSwept(ctx, intent.ID)
			continue
		}
		amount := balance - minBalance
		if intent.Mint == "" {
			txSig, err := s.sweepSOL(intent.DepositIndex, amount)
			if err != nil {
				log.Printf("crypto sweep: SOL from %s: %v", intent.DepositPubkey, err)
				continue
			}
			log.Printf("crypto sweep: %.6f SOL from %s tx=%s", float64(amount)/1e9, intent.DepositPubkey, txSig)
		} else {
			log.Printf("crypto sweep: SPL token sweep not yet implemented for %s", intent.DepositPubkey)
			continue
		}
		s.store.MarkCheckoutSwept(ctx, intent.ID)
	}
}

func (s *Service) getAccountBalance(pubkey string) (uint64, error) {
	params := []interface{}{pubkey, map[string]string{"commitment": "confirmed"}}
	body, err := s.rpc.Call("getBalance", params, 10*time.Second)
	if err != nil {
		return 0, err
	}
	var result struct {
		Result struct {
			Value uint64 `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	return result.Result.Value, nil
}

func (s *Service) sweepSOL(depositIndex int64, amount uint64) (string, error) {
	pub, priv, err := deriveDepositKeypair(s.cfg.HDWalletSeed, depositIndex)
	if err != nil {
		return "", err
	}

	blockhash, err := s.getRecentBlockhash()
	if err != nil {
		return "", fmt.Errorf("blockhash: %w", err)
	}

	fromPubkey := base58.Encode(pub)
	tx, err := BuildTransferTransaction(fromPubkey, s.cfg.WalletPubkey, amount, blockhash, priv)
	if err != nil {
		return "", fmt.Errorf("build tx: %w", err)
	}

	sendParams := []interface{}{
		tx,
		map[string]interface{}{"encoding": "base64"},
	}
	body, err := s.rpc.Call("sendTransaction", sendParams, 30*time.Second)
	if err != nil {
		return "", err
	}

	var result struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("rpc: %s", result.Error.Message)
	}
	return result.Result, nil
}

func (s *Service) getRecentBlockhash() (string, error) {
	params := []interface{}{map[string]string{"commitment": "confirmed"}}
	body, err := s.rpc.Call("getLatestBlockhash", params, 10*time.Second)
	if err != nil {
		return "", err
	}
	var result struct {
		Result struct {
			Value struct {
				Blockhash string `json:"blockhash"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Result.Value.Blockhash, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
