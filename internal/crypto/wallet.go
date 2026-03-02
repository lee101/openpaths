package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"

	"github.com/mr-tron/base58"
	"github.com/openpaths/openpaths/internal/model"
	"golang.org/x/crypto/hkdf"
)

func deriveDepositKeypair(seed []byte, index int64) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	info := make([]byte, 8)
	binary.BigEndian.PutUint64(info, uint64(index))

	hkdfReader := hkdf.New(sha256.New, seed, []byte("openpaths-deposit"), info)
	keySeed := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(hkdfReader, keySeed); err != nil {
		return nil, nil, fmt.Errorf("hkdf derivation: %w", err)
	}

	priv := ed25519.NewKeyFromSeed(keySeed)
	pub := priv.Public().(ed25519.PublicKey)
	return pub, priv, nil
}

func GetDepositPubkey(seed []byte, index int64) (string, error) {
	pub, _, err := deriveDepositKeypair(seed, index)
	if err != nil {
		return "", err
	}
	return base58.Encode(pub), nil
}

func GenerateReferencePubkey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base58.Encode(b), nil
}

func BuildSolanaPayURL(intent *model.CryptoCheckoutIntent) string {
	params := url.Values{}
	params.Set("amount", intent.AmountUI)
	params.Set("label", "OpenPath")
	params.Set("message", fmt.Sprintf("Credits: $%.2f", intent.AmountUSD))
	if intent.Mint != "" {
		params.Set("spl-token", intent.Mint)
	}
	return fmt.Sprintf("solana:%s?%s", intent.DepositPubkey, params.Encode())
}

func BuildTransferTransaction(from, to string, amount uint64, blockhash string, privateKey ed25519.PrivateKey) (string, error) {
	fromBytes, err := base58.Decode(from)
	if err != nil || len(fromBytes) != 32 {
		return "", fmt.Errorf("invalid from pubkey")
	}
	toBytes, err := base58.Decode(to)
	if err != nil || len(toBytes) != 32 {
		return "", fmt.Errorf("invalid to pubkey")
	}
	blockhashBytes, err := base58.Decode(blockhash)
	if err != nil || len(blockhashBytes) != 32 {
		return "", fmt.Errorf("invalid blockhash")
	}

	systemProgramID := make([]byte, 32)

	header := []byte{1, 0, 1}

	accountKeys := make([]byte, 0, 96)
	accountKeys = append(accountKeys, fromBytes...)
	accountKeys = append(accountKeys, toBytes...)
	accountKeys = append(accountKeys, systemProgramID...)

	instructionData := make([]byte, 12)
	instructionData[0] = 2 // Transfer instruction
	binary.LittleEndian.PutUint64(instructionData[4:], amount)

	message := make([]byte, 0, 256)
	message = append(message, header...)
	message = append(message, 3) // num_account_keys
	message = append(message, accountKeys...)
	message = append(message, blockhashBytes...)
	message = append(message, 1)    // num_instructions
	message = append(message, 2)    // program_id_index
	message = append(message, 2)    // num_accounts
	message = append(message, 0, 1) // account indices
	message = append(message, byte(len(instructionData)))
	message = append(message, instructionData...)

	signature := ed25519.Sign(privateKey, message)

	tx := make([]byte, 0, 1+64+len(message))
	tx = append(tx, 1) // num_signatures
	tx = append(tx, signature...)
	tx = append(tx, message...)

	return encodeBase64(tx), nil
}

func encodeBase64(data []byte) string {
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

func DecodeHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd length hex string")
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		var v byte
		for j := 0; j < 2; j++ {
			c := s[i*2+j]
			switch {
			case c >= '0' && c <= '9':
				v = v*16 + c - '0'
			case c >= 'a' && c <= 'f':
				v = v*16 + c - 'a' + 10
			case c >= 'A' && c <= 'F':
				v = v*16 + c - 'A' + 10
			default:
				return nil, fmt.Errorf("invalid hex char: %c", c)
			}
		}
		b[i] = v
	}
	return b, nil
}
