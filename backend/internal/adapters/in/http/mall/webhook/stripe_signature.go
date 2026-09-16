// backend/internal/adapters/in/http/mall/webhook/stripe_signature.go
package mallHandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func verifyStripeSignature(signatureHeader string, body []byte, secret string, now time.Time, tolerance time.Duration) error {
	timestamp, signatures, err := parseStripeSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	signedAt := time.Unix(timestamp, 0).UTC()
	if tolerance > 0 {
		difference := now.Sub(signedAt)
		if difference < 0 {
			difference = -difference
		}
		if difference > tolerance {
			return errors.New("timestamp_out_of_tolerance")
		}
	}

	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, signature := range signatures {
		if subtleEqHex(expected, signature) {
			return nil
		}
	}

	return errors.New("signature_mismatch")
}

func parseStripeSignatureHeader(header string) (timestamp int64, v1Signatures []string, err error) {
	parts := strings.Split(header, ",")
	var timestampText string
	signatures := make([]string, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch {
		case strings.HasPrefix(part, "t="):
			timestampText = strings.TrimSpace(strings.TrimPrefix(part, "t="))

		case strings.HasPrefix(part, "v1="):
			signature := strings.TrimSpace(strings.TrimPrefix(part, "v1="))
			if signature != "" {
				signatures = append(signatures, signature)
			}
		}
	}

	if timestampText == "" || len(signatures) == 0 {
		return 0, nil, errors.New("invalid_signature_header")
	}

	timestamp, err = strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return 0, nil, errors.New("invalid_signature_timestamp")
	}

	return timestamp, signatures, nil
}

func subtleEqHex(left string, right string) bool {
	leftBytes := []byte(strings.ToLower(strings.TrimSpace(left)))
	rightBytes := []byte(strings.ToLower(strings.TrimSpace(right)))
	if len(leftBytes) != len(rightBytes) {
		return false
	}

	var difference byte
	for index := 0; index < len(leftBytes); index++ {
		difference |= leftBytes[index] ^ rightBytes[index]
	}

	return difference == 0
}
