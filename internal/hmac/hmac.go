package hmac

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

// ValidateSignature validates a signature against the payload using the provided HMAC key
func ValidateSignature(payload []byte, signature string, key string) bool {
	if key == "" {
		// HMAC validation disabled
		return true
	}

	expectedSignature := calculateSignature(payload, key)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ValidateRequest validates the HMAC signature of an incoming request
func ValidateRequest(r *http.Request, hmacKey string) error {
	if hmacKey == "" {
		// HMAC validation disabled
		return nil
	}

	// Get the signature from the header
	signature := r.Header.Get("X-TFC-Task-Signature")
	if signature == "" {
		return fmt.Errorf("missing X-TFC-Task-Signature header")
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Calculate expected signature
	expectedSignature := calculateSignature(body, hmacKey)

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid HMAC signature")
	}

	return nil
}

// calculateSignature calculates the HMAC-SHA256 signature for the given payload
func calculateSignature(payload []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// GenerateKey generates a random HMAC key for configuration
func GenerateKey() (string, error) {
	// Generate a 32-byte random key
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}
