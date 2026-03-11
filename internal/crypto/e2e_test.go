//go:build manual

// Manual integration test for real SOL payments on mainnet.
//
// This test sends REAL SOL on mainnet. It costs real money.
// It is NOT included in CI. Run manually only:
//
//   go test -tags manual -run TestCryptoPaymentE2E -v -timeout 5m ./internal/crypto/
//
// Prerequisites:
//   - Server running (localhost:8080 by default, or set SERVER_URL)
//   - SOLANA_PRIVATE_KEY set in .env (sender wallet, must be funded)
//   - HELIUS_API_KEY set in .env (for RPC)
//   - config.yaml: crypto.min_topup_usd set to 0.01 for small-amount testing
//
// Environment overrides:
//   SERVER_URL=http://localhost:8080  (default)
//   CRYPTO_TEST_USD=0.02             (default, USD amount to test with)

package crypto_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mr-tron/base58"
)

func init() {
	// Load .env files
	loadEnvFile(".env")
	loadEnvFile("../../.env")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		// Strip quotes
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func TestCryptoPaymentE2E(t *testing.T) {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	privKeyB58 := os.Getenv("SOLANA_PRIVATE_KEY")
	if privKeyB58 == "" {
		t.Fatal("SOLANA_PRIVATE_KEY not set")
	}
	privKeyBytes, err := base58.Decode(privKeyB58)
	if err != nil || len(privKeyBytes) != 64 {
		t.Fatalf("invalid SOLANA_PRIVATE_KEY: %v (len=%d)", err, len(privKeyBytes))
	}
	senderPrivKey := ed25519.PrivateKey(privKeyBytes)
	senderPubKey := senderPrivKey.Public().(ed25519.PublicKey)
	senderAddr := base58.Encode(senderPubKey)
	t.Logf("Sender wallet: %s", senderAddr)

	heliusKey := os.Getenv("HELIUS_API_KEY")
	rpcURL := "https://api.mainnet-beta.solana.com"
	if heliusKey != "" {
		rpcURL = fmt.Sprintf("https://mainnet.helius-rpc.com/?api-key=%s", heliusKey)
	}
	t.Logf("RPC: %s", rpcURL[:50]+"...")

	// --- 1. Register test user ---
	email := fmt.Sprintf("cryptotest_%d@test.local", rand.Int63())
	password := "testpassword123"
	t.Logf("Registering test user: %s", email)

	signupBody, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(serverURL+"/auth/register", "application/json", bytes.NewReader(signupBody))
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("register failed (%d): %s", resp.StatusCode, body)
	}

	var authResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(body, &authResp)
	token := authResp.Token
	if token == "" {
		t.Fatal("no token in register response")
	}
	t.Logf("Got JWT: %s...%s", token[:8], token[len(token)-4:])

	// --- 2. Get initial balance ---
	balReq, _ := http.NewRequest("GET", serverURL+"/account/balance", nil)
	balReq.Header.Set("Authorization", "Bearer "+token)
	balResp, err := http.DefaultClient.Do(balReq)
	if err != nil {
		t.Fatalf("balance request failed: %v", err)
	}
	balBody, _ := io.ReadAll(balResp.Body)
	balResp.Body.Close()
	t.Logf("Initial balance response: %s", balBody)

	var balData struct {
		Balance int64 `json:"balance"` // hundredths-of-a-cent
	}
	json.Unmarshal(balBody, &balData)
	initialBalance := balData.Balance
	t.Logf("Initial balance: %d (hundredths-of-cent)", initialBalance)

	// --- 3. Create crypto checkout ---
	testUSD := 0.02 // ~0.00014 SOL at $140
	if v := os.Getenv("CRYPTO_TEST_USD"); v != "" {
		fmt.Sscanf(v, "%f", &testUSD)
	}
	t.Logf("Creating checkout for $%.4f via SOL", testUSD)

	checkoutBody, _ := json.Marshal(map[string]interface{}{
		"method":     "sol",
		"amount_usd": testUSD,
	})
	checkoutReq, _ := http.NewRequest("POST", serverURL+"/crypto/checkout", bytes.NewReader(checkoutBody))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.Header.Set("Authorization", "Bearer "+token)
	checkoutResp, err := http.DefaultClient.Do(checkoutReq)
	if err != nil {
		t.Fatalf("checkout request failed: %v", err)
	}
	checkoutRespBody, _ := io.ReadAll(checkoutResp.Body)
	checkoutResp.Body.Close()

	if checkoutResp.StatusCode != 200 {
		t.Fatalf("checkout failed (%d): %s", checkoutResp.StatusCode, checkoutRespBody)
	}

	var checkout struct {
		IntentID       string  `json:"intent_id"`
		DepositAddress string  `json:"deposit_address"`
		AmountUI       string  `json:"amount_ui"`
		AmountUSD      float64 `json:"amount_usd"`
		Method         string  `json:"method"`
	}
	json.Unmarshal(checkoutRespBody, &checkout)
	t.Logf("Checkout created: id=%s deposit=%s amount=%s SOL ($%.4f)",
		checkout.IntentID, checkout.DepositAddress, checkout.AmountUI, checkout.AmountUSD)

	// Parse SOL amount to lamports
	var solAmount float64
	fmt.Sscanf(checkout.AmountUI, "%f", &solAmount)
	lamports := uint64(solAmount * 1e9)
	if lamports == 0 {
		t.Fatal("calculated 0 lamports")
	}
	t.Logf("Sending %d lamports (%.9f SOL) to %s", lamports, solAmount, checkout.DepositAddress)

	// --- 4. Get recent blockhash ---
	blockhash, err := e2eGetBlockhash(rpcURL)
	if err != nil {
		t.Fatalf("getLatestBlockhash: %v", err)
	}
	t.Logf("Blockhash: %s", blockhash)

	// --- 5. Build and send SOL transfer ---
	txBase64, err := e2eBuildTransfer(senderAddr, checkout.DepositAddress, lamports, blockhash, senderPrivKey)
	if err != nil {
		t.Fatalf("buildTransfer: %v", err)
	}

	txSig, err := e2eSendTx(rpcURL, txBase64)
	if err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}
	t.Logf("TX sent: %s", txSig)
	t.Logf("Solscan: https://solscan.io/tx/%s", txSig)

	// --- 6. Wait for confirmation ---
	t.Log("Waiting for on-chain confirmation...")
	confirmed := false
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		status, err := e2eGetSigStatus(rpcURL, txSig)
		if err == nil && status != "" {
			t.Logf("TX status: %s", status)
			if status == "finalized" || status == "confirmed" {
				confirmed = true
				break
			}
		}
	}
	if !confirmed {
		t.Fatal("transaction not confirmed after 60s")
	}

	// --- 7. Poll checkout status ---
	t.Log("Polling checkout status...")
	paid := false
	for i := 0; i < 60; i++ {
		time.Sleep(3 * time.Second)
		statusResp, err := http.Get(fmt.Sprintf("%s/crypto/checkout/%s", serverURL, checkout.IntentID))
		if err != nil {
			continue
		}
		statusBody, _ := io.ReadAll(statusResp.Body)
		statusResp.Body.Close()

		var st struct {
			Status string `json:"status"`
			TxSig  string `json:"tx_sig"`
		}
		json.Unmarshal(statusBody, &st)
		t.Logf("Checkout status: %s (tx: %s)", st.Status, st.TxSig)

		if st.Status == "paid" {
			paid = true
			break
		}
		if st.Status == "expired" {
			t.Fatal("checkout expired before payment was detected")
		}
	}
	if !paid {
		t.Fatal("checkout not marked as paid after 3 minutes")
	}

	// --- 8. Verify balance increased ---
	time.Sleep(2 * time.Second)
	balReq2, _ := http.NewRequest("GET", serverURL+"/account/balance", nil)
	balReq2.Header.Set("Authorization", "Bearer "+token)
	balResp2, err := http.DefaultClient.Do(balReq2)
	if err != nil {
		t.Fatalf("final balance request: %v", err)
	}
	balBody2, _ := io.ReadAll(balResp2.Body)
	balResp2.Body.Close()

	var balData2 struct {
		Balance int64 `json:"balance"`
	}
	json.Unmarshal(balBody2, &balData2)
	finalBalance := balData2.Balance
	t.Logf("Final balance: %d (was %d, diff=%d)", finalBalance, initialBalance, finalBalance-initialBalance)

	if finalBalance <= initialBalance {
		t.Fatalf("balance did not increase: before=%d after=%d", initialBalance, finalBalance)
	}

	t.Logf("SUCCESS: Sent %.9f SOL ($%.4f), balance increased by %d units",
		solAmount, checkout.AmountUSD, finalBalance-initialBalance)
}

// --- Solana helpers (self-contained, no package imports) ---

func e2eGetBlockhash(rpcURL string) (string, error) {
	body, err := e2eRPC(rpcURL, "getLatestBlockhash", []interface{}{map[string]string{"commitment": "finalized"}})
	if err != nil {
		return "", err
	}
	var resp struct {
		Result struct {
			Value struct {
				Blockhash string `json:"blockhash"`
			} `json:"value"`
		} `json:"result"`
	}
	json.Unmarshal(body, &resp)
	if resp.Result.Value.Blockhash == "" {
		return "", fmt.Errorf("empty blockhash: %s", body)
	}
	return resp.Result.Value.Blockhash, nil
}

func e2eBuildTransfer(from, to string, lamports uint64, blockhash string, privKey ed25519.PrivateKey) (string, error) {
	fromBytes, err := base58.Decode(from)
	if err != nil || len(fromBytes) != 32 {
		return "", fmt.Errorf("invalid from: %v", err)
	}
	toBytes, err := base58.Decode(to)
	if err != nil || len(toBytes) != 32 {
		return "", fmt.Errorf("invalid to: %v", err)
	}
	bhBytes, err := base58.Decode(blockhash)
	if err != nil || len(bhBytes) != 32 {
		return "", fmt.Errorf("invalid blockhash: %v", err)
	}

	systemProgram := make([]byte, 32)

	ixData := make([]byte, 12)
	ixData[0] = 2 // Transfer
	binary.LittleEndian.PutUint64(ixData[4:], lamports)

	msg := []byte{1, 0, 1}
	msg = append(msg, 3)
	msg = append(msg, fromBytes...)
	msg = append(msg, toBytes...)
	msg = append(msg, systemProgram...)
	msg = append(msg, bhBytes...)
	msg = append(msg, 1)
	msg = append(msg, 2)
	msg = append(msg, 2)
	msg = append(msg, 0, 1)
	msg = append(msg, byte(len(ixData)))
	msg = append(msg, ixData...)

	sig := ed25519.Sign(privKey, msg)
	tx := []byte{1}
	tx = append(tx, sig...)
	tx = append(tx, msg...)

	return e2eBase64(tx), nil
}

func e2eSendTx(rpcURL, txBase64 string) (string, error) {
	body, err := e2eRPC(rpcURL, "sendTransaction", []interface{}{
		txBase64,
		map[string]interface{}{"encoding": "base64", "skipPreflight": false},
	})
	if err != nil {
		return "", err
	}
	var resp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		} `json:"error"`
	}
	json.Unmarshal(body, &resp)
	if resp.Error != nil {
		return "", fmt.Errorf("sendTransaction: %s (%v)", resp.Error.Message, resp.Error.Data)
	}
	if resp.Result == "" {
		return "", fmt.Errorf("empty sig: %s", body)
	}
	return resp.Result, nil
}

func e2eGetSigStatus(rpcURL, sig string) (string, error) {
	body, err := e2eRPC(rpcURL, "getSignatureStatuses", []interface{}{[]string{sig}})
	if err != nil {
		return "", err
	}
	var resp struct {
		Result struct {
			Value []struct {
				ConfirmationStatus string      `json:"confirmationStatus"`
				Err                interface{} `json:"err"`
			} `json:"value"`
		} `json:"result"`
	}
	json.Unmarshal(body, &resp)
	if len(resp.Result.Value) == 0 || resp.Result.Value[0].ConfirmationStatus == "" {
		return "", nil
	}
	if resp.Result.Value[0].Err != nil {
		return "", fmt.Errorf("tx error: %v", resp.Result.Value[0].Err)
	}
	return resp.Result.Value[0].ConfirmationStatus, nil
}

func e2eRPC(rpcURL, method string, params interface{}) ([]byte, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
	})
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("429 rate limited")
	}
	return io.ReadAll(resp.Body)
}

func e2eBase64(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(data)+2)/3*4)
	for i := 0; i < len(data); i += 3 {
		var b uint32
		remaining := len(data) - i
		b = uint32(data[i]) << 16
		if remaining > 1 {
			b |= uint32(data[i+1]) << 8
		}
		if remaining > 2 {
			b |= uint32(data[i+2])
		}
		result = append(result, chars[(b>>18)&0x3F])
		result = append(result, chars[(b>>12)&0x3F])
		if remaining > 1 {
			result = append(result, chars[(b>>6)&0x3F])
		} else {
			result = append(result, '=')
		}
		if remaining > 2 {
			result = append(result, chars[b&0x3F])
		} else {
			result = append(result, '=')
		}
	}
	return string(result)
}
