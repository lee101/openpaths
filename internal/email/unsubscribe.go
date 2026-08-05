package email

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// UnsubscribeToken returns a stable HMAC token for an email address so
// unsubscribe links work without a login session.
func UnsubscribeToken(secret, addr string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.ToLower(strings.TrimSpace(addr))))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func VerifyUnsubscribeToken(secret, addr, token string) bool {
	return hmac.Equal([]byte(UnsubscribeToken(secret, addr)), []byte(token))
}
