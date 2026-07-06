package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// VerifyZaloWebhookSignature checks the X-ZEvent-Signature header Zalo sends
// on every webhook call. Confirmed against current developers.zalo.me docs:
// mac = sha256(appId + rawBody + timestamp + OAsecretKey), where
// OAsecretKey is the same secret_key issued for the Zalo App (no separate
// webhook secret to configure). appID/timestamp come from the JSON body
// itself (app_id, timestamp fields), rawBody is the exact request bytes.
func VerifyZaloWebhookSignature(appID, rawBody, timestamp, secretKey, signature string) bool {
	sum := sha256.Sum256([]byte(appID + rawBody + timestamp + secretKey))
	expected := hex.EncodeToString(sum[:])
	// hmac.Equal is constant-time — avoids leaking timing info about how
	// much of the signature matched.
	return hmac.Equal([]byte(expected), []byte(signature))
}
