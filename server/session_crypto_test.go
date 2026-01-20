package server

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestDeriveKey(t *testing.T) {
	key1 := deriveKey("secret1")
	key2 := deriveKey("secret2")
	key1Again := deriveKey("secret1")

	// Keys should be 32 bytes (256 bits)
	if len(key1) != 32 {
		t.Errorf("expected key length 32, got %d", len(key1))
	}

	// Same secret should produce same key
	if string(key1) != string(key1Again) {
		t.Error("same secret produced different keys")
	}

	// Different secrets should produce different keys
	if string(key1) == string(key2) {
		t.Error("different secrets produced same key")
	}
}

func TestEncryptDecryptSession(t *testing.T) {
	secret := "test-secret-key-for-sessions"

	// Create session data
	session := NewSessionData(24 * time.Hour)
	session.Data["user_id"] = float64(123) // JSON numbers become float64
	session.Data["username"] = "testuser"
	session.Flash["success"] = "Welcome back!"

	// Encrypt
	encrypted, err := encryptSession(session, secret)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Should be valid base64
	if _, err := base64.StdEncoding.DecodeString(encrypted); err != nil {
		t.Errorf("encrypted value is not valid base64: %v", err)
	}

	// Decrypt
	decrypted, err := decryptSession(encrypted, secret)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	// Verify data
	if decrypted.Data["user_id"] != float64(123) {
		t.Errorf("expected user_id 123, got %v", decrypted.Data["user_id"])
	}
	if decrypted.Data["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", decrypted.Data["username"])
	}
	if decrypted.Flash["success"] != "Welcome back!" {
		t.Errorf("expected flash message, got %v", decrypted.Flash["success"])
	}
	if decrypted.IsExpired() {
		t.Error("session should not be expired")
	}
}

func TestEncryptDecryptEmptySession(t *testing.T) {
	secret := "test-secret"
	session := NewSessionData(time.Hour)

	encrypted, err := encryptSession(session, secret)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := decryptSession(encrypted, secret)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if len(decrypted.Data) != 0 {
		t.Errorf("expected empty data, got %v", decrypted.Data)
	}
	if len(decrypted.Flash) != 0 {
		t.Errorf("expected empty flash, got %v", decrypted.Flash)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	secret1 := "secret-key-1"
	secret2 := "secret-key-2"

	session := NewSessionData(time.Hour)
	session.Data["test"] = "value"

	encrypted, err := encryptSession(session, secret1)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = decryptSession(encrypted, secret2)
	if err == nil {
		t.Error("expected decryption to fail with wrong key")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := decryptSession("not-valid-base64!!!", "secret")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecryptTooShort(t *testing.T) {
	// Valid base64 but too short for nonce + tag
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err := decryptSession(short, "secret")
	if err == nil {
		t.Error("expected error for ciphertext too short")
	}
}

func TestDecryptTamperedData(t *testing.T) {
	secret := "test-secret"
	session := NewSessionData(time.Hour)
	session.Data["test"] = "value"

	encrypted, err := encryptSession(session, secret)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Tamper with the data
	decoded, _ := base64.StdEncoding.DecodeString(encrypted)
	decoded[len(decoded)-1] ^= 0xFF // Flip bits in last byte
	tampered := base64.StdEncoding.EncodeToString(decoded)

	_, err = decryptSession(tampered, secret)
	if err == nil {
		t.Error("expected error for tampered data")
	}
}

func TestSessionExpiration(t *testing.T) {
	// Create expired session
	session := &SessionData{
		Data:      make(map[string]interface{}),
		Flash:     make(map[string]string),
		ExpiresAt: time.Now().Add(-time.Hour), // 1 hour ago
	}

	if !session.IsExpired() {
		t.Error("session should be expired")
	}

	// Create valid session
	validSession := NewSessionData(time.Hour)
	if validSession.IsExpired() {
		t.Error("session should not be expired")
	}
}

func TestEncryptionIsUnique(t *testing.T) {
	secret := "test-secret"
	session := NewSessionData(time.Hour)
	session.Data["test"] = "value"

	// Encrypt same data twice
	encrypted1, _ := encryptSession(session, secret)
	encrypted2, _ := encryptSession(session, secret)

	// Should produce different ciphertexts (random nonce)
	if encrypted1 == encrypted2 {
		t.Error("encryptions should be unique due to random nonce")
	}

	// But both should decrypt to same data
	decrypted1, _ := decryptSession(encrypted1, secret)
	decrypted2, _ := decryptSession(encrypted2, secret)

	if decrypted1.Data["test"] != decrypted2.Data["test"] {
		t.Error("both should decrypt to same data")
	}
}

func TestGenerateRandomSecret(t *testing.T) {
	secret1, err := generateRandomSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	secret2, err := generateRandomSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}

	// Should be valid base64
	decoded, err := base64.StdEncoding.DecodeString(secret1)
	if err != nil {
		t.Errorf("secret is not valid base64: %v", err)
	}

	// Should be 32 bytes decoded
	if len(decoded) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(decoded))
	}

	// Should be unique
	if secret1 == secret2 {
		t.Error("generated secrets should be unique")
	}
}

func TestSignPLN(t *testing.T) {
	secret := "test-secret-key"
	pln := `{name: "Alice", age: 30}`

	signed := SignPLN(pln, secret)

	// Should contain a colon separator
	if !containsColon(signed) {
		t.Error("signed PLN should contain colon separator")
	}

	// Should be able to verify
	verified, err := VerifyPLN(signed, secret)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if verified != pln {
		t.Errorf("expected %q, got %q", pln, verified)
	}
}

func TestSignPLNWithRecord(t *testing.T) {
	secret := "test-secret"
	pln := `@Person({name: "Alice", age: 30}) @errors {name: "required"}`

	signed := SignPLN(pln, secret)
	verified, err := VerifyPLN(signed, secret)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if verified != pln {
		t.Errorf("expected %q, got %q", pln, verified)
	}
}

func TestVerifyPLNWrongSecret(t *testing.T) {
	secret1 := "secret-1"
	secret2 := "secret-2"
	pln := `{name: "Alice"}`

	signed := SignPLN(pln, secret1)

	_, err := VerifyPLN(signed, secret2)
	if err == nil {
		t.Error("expected verification to fail with wrong secret")
	}
}

func TestVerifyPLNTampered(t *testing.T) {
	secret := "test-secret"
	pln := `{name: "Alice"}`

	signed := SignPLN(pln, secret)

	// Tamper with the signature
	tampered := "x" + signed[1:]
	_, err := VerifyPLN(tampered, secret)
	if err == nil {
		t.Error("expected verification to fail with tampered signature")
	}
}

func TestVerifyPLNInvalidFormat(t *testing.T) {
	secret := "test-secret"

	// Missing colon
	_, err := VerifyPLN("nocolonhere", secret)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
