package hmac

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ValidateSignature validates a signature against the payload using the provided HMAC key
func ValidateSignature(payload []byte, signature string, key string) bool {
	if key == "" {
		// HMAC validation disabled
		return true
	}
	// Normalize provided signature to support with/without prefix
	provided := strings.TrimSpace(signature)
	expected := calculateSignature(payload, key)

	// Test multiple algorithms since the provided signature is 128 chars (suggesting SHA512)
	testMultipleAlgorithms(payload, provided, key)

	// Compare signatures directly (both should be hex strings without prefix)
	match := hmac.Equal([]byte(strings.ToLower(provided)), []byte(strings.ToLower(expected)))
	return match
}

// testMultipleAlgorithms tests various HMAC algorithms to see which one matches
func testMultipleAlgorithms(payload []byte, providedSig, key string) {
	// Try to decode key as hex first
	var keyBytes []byte
	if decoded, err := hex.DecodeString(key); err == nil && len(decoded) > 0 {
		keyBytes = decoded
	} else {
		keyBytes = []byte(key)
	}

	// Test SHA256
	h256 := hmac.New(sha256.New, keyBytes)
	h256.Write(payload)
	sig256 := hex.EncodeToString(h256.Sum(nil))

	// Test SHA512
	h512 := hmac.New(sha512.New, keyBytes)
	h512.Write(payload)
	sig512 := hex.EncodeToString(h512.Sum(nil))

	// Check if either matches (silent validation)
	_ = strings.EqualFold(providedSig, sig256)
	_ = strings.EqualFold(providedSig, sig512)
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

// calculateSignature calculates the HMAC-SHA512 signature for the given payload (HCP Terraform uses SHA512)
func calculateSignature(payload []byte, key string) string {
	// Use the key as-is (original working approach)
	h := hmac.New(sha512.New, []byte(key))
	h.Write(payload)
	result := hex.EncodeToString(h.Sum(nil))
	return result
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
