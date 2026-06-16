package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalidSignature = errors.New("invalid webhook signature")

// ValidateGitHubSignature validates the X-Hub-Signature-256 header from GitHub
func ValidateGitHubSignature(payload []byte, signature, secret string) error {
	if signature == "" {
		return ErrInvalidSignature
	}

	// GitHub sends: sha256=<hex>
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return ErrInvalidSignature
	}

	return nil
}