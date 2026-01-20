package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// SessionData represents the encrypted session payload
type SessionData struct {
	Data      map[string]interface{} `json:"d"`           // Session data
	Flash     map[string]string      `json:"f,omitempty"` // Flash messages
	ExpiresAt time.Time              `json:"e"`           // Expiration time
}

// NewSessionData creates a new empty session with the given expiration
func NewSessionData(maxAge time.Duration) *SessionData {
	return &SessionData{
		Data:      make(map[string]interface{}),
		Flash:     make(map[string]string),
		ExpiresAt: time.Now().Add(maxAge),
	}
}

// IsExpired returns true if the session has expired
func (s *SessionData) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// deriveKey derives a 32-byte AES-256 key from a secret string using SHA-256
func deriveKey(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

// encryptSession encrypts session data using AES-256-GCM
// Returns base64-encoded string: base64(nonce[12] + ciphertext + tag[16])
func encryptSession(data *SessionData, secret string) (string, error) {
	// Serialize to JSON
	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Derive key
	key := deriveKey(secret)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt (nonce is prepended to ciphertext)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Base64 encode
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptSession decrypts session data using AES-256-GCM
// Expects base64-encoded string: base64(nonce[12] + ciphertext + tag[16])
func decryptSession(encoded string, secret string) (*SessionData, error) {
	// Base64 decode
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	// Derive key
	key := deriveKey(secret)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Check minimum size (nonce + tag)
	if len(ciphertext) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// Deserialize JSON
	var data SessionData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, err
	}

	// Initialize maps if nil (empty JSON objects)
	if data.Data == nil {
		data.Data = make(map[string]interface{})
	}
	if data.Flash == nil {
		data.Flash = make(map[string]string)
	}

	return &data, nil
}

// generateRandomSecret generates a cryptographically secure random secret
// for development mode. Returns a 32-byte hex-encoded string (64 chars).
func generateRandomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// SignPLN signs a PLN string with HMAC-SHA256 for secure transport.
// Returns: "hmac:base64(pln)" where hmac is base64-encoded HMAC-SHA256.
func SignPLN(pln string, secret string) string {
	key := deriveKey(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(pln))
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))
	plnEncoded := base64.StdEncoding.EncodeToString([]byte(pln))
	return sig + ":" + plnEncoded
}

// VerifyPLN verifies and extracts a signed PLN string.
// Returns the original PLN string if valid, or an error if invalid.
func VerifyPLN(signed string, secret string) (string, error) {
	parts := strings.SplitN(signed, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid signed PLN format")
	}

	sig, plnEncoded := parts[0], parts[1]

	// Decode the PLN
	plnBytes, err := base64.StdEncoding.DecodeString(plnEncoded)
	if err != nil {
		return "", errors.New("invalid PLN encoding")
	}
	pln := string(plnBytes)

	// Verify HMAC
	key := deriveKey(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(pln))
	expectedSig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", errors.New("PLN signature verification failed")
	}

	return pln, nil
}
