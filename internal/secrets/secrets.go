package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type AuthType string

const (
	AuthTypeAPIKey AuthType = "apikey"
	AuthTypeOAuth2 AuthType = "oauth2"
)

type CheckmarxSecrets struct {
	AuthType      AuthType `json:"auth_type"`
	APIKey        string   `json:"api_key,omitempty"`
	OAuthClientID string   `json:"oauth_client_id,omitempty"`
	OAuthSecret   string   `json:"oauth_client_secret,omitempty"`
	OAuthTokenURL string   `json:"oauth_token_url,omitempty"`
}

// LoadSecrets loads secrets from a (decrypted) JSON file.
func LoadSecrets(path string) (*CheckmarxSecrets, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}

	// Try to decode as plaintext JSON first
	var s CheckmarxSecrets
	if err := json.Unmarshal(raw, &s); err == nil {
		// If it parses, encrypt and save for future use
		key, err := getEncryptionKey()
		if err != nil {
			return nil, err
		}
		encrypted, err := encryptAESGCM(raw, key)
		if err != nil {
			return nil, err
		}
		encPath := path + ".enc"
		if err := os.WriteFile(encPath, []byte(encrypted), 0600); err != nil {
			return nil, fmt.Errorf("failed to write encrypted secrets: %w", err)
		}
		fmt.Printf("[INFO] Secrets file was unencrypted. Encrypted version written to %s\n", encPath)
		return &s, nil
	}

	// If not plaintext, try to decrypt
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}
	decrypted, err := decryptAESGCM(string(raw), key)
	if err != nil {
		return nil, errors.New("failed to decrypt secrets: invalid format or key")
	}
	if err := json.Unmarshal(decrypted, &s); err != nil {
		return nil, fmt.Errorf("failed to decode decrypted secrets: %w", err)
	}
	return &s, nil
}

func getEncryptionKey() ([]byte, error) {
	keyB64 := os.Getenv("CHECKMARX_SECRETS_KEY")
	if keyB64 == "" {
		return nil, errors.New("CHECKMARX_SECRETS_KEY env var not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes (base64-encoded)")
	}
	return key, nil
}

func encryptAESGCM(plaintext, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAESGCM(ciphertextB64 string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertext[:gcm.NonceSize()]
	data := ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, data, nil)
}
