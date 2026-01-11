package auth

import (
	"testing"
	"time"
)

func TestNewWebAuthnManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr, err := NewWebAuthnManager(db, "localhost", "http://localhost:8080", "Test App")
	if err != nil {
		t.Fatalf("NewWebAuthnManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("manager is nil")
	}
	if mgr.webauthn == nil {
		t.Fatal("webauthn is nil")
	}
}

func TestBeginRegistration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr, _ := NewWebAuthnManager(db, "localhost", "http://localhost:8080", "Test App")

	options, challengeID, err := mgr.BeginRegistration("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("BeginRegistration failed: %v", err)
	}

	if options == nil {
		t.Fatal("options is nil")
	}
	if challengeID == "" {
		t.Fatal("challengeID is empty")
	}

	// Verify challenge was stored
	mgr.mu.RLock()
	_, ok := mgr.challenges[challengeID]
	mgr.mu.RUnlock()
	if !ok {
		t.Error("challenge was not stored")
	}
}

func TestBeginLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr, _ := NewWebAuthnManager(db, "localhost", "http://localhost:8080", "Test App")

	options, challengeID, err := mgr.BeginLogin()
	if err != nil {
		t.Fatalf("BeginLogin failed: %v", err)
	}

	if options == nil {
		t.Fatal("options is nil")
	}
	if challengeID == "" {
		t.Fatal("challengeID is empty")
	}
}

func TestCleanExpiredChallenges(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mgr, _ := NewWebAuthnManager(db, "localhost", "http://localhost:8080", "Test App")

	// Add some challenges manually
	mgr.mu.Lock()
	mgr.challenges["expired1"] = &challengeData{expiresAt: time.Now().Add(-time.Hour)}
	mgr.challenges["expired2"] = &challengeData{expiresAt: time.Now().Add(-time.Minute)}
	mgr.challenges["valid"] = &challengeData{expiresAt: time.Now().Add(time.Hour)}
	mgr.mu.Unlock()

	count := mgr.CleanExpiredChallenges()
	if count != 2 {
		t.Errorf("cleaned %d challenges, want 2", count)
	}

	mgr.mu.RLock()
	remaining := len(mgr.challenges)
	mgr.mu.RUnlock()

	if remaining != 1 {
		t.Errorf("%d challenges remaining, want 1", remaining)
	}
}

func TestTransportStrings(t *testing.T) {
	transports := transportStrings(nil)
	if transports != nil {
		t.Errorf("expected nil for empty input, got %v", transports)
	}
}

// Note: Full WebAuthn flow tests require mocking the WebAuthn responses,
// which is complex. The integration tests in handlers_test.go will cover
// the end-to-end flow with browser-like behavior.
