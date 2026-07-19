package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create user first
	user, err := db.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create session
	session, err := db.CreateSession(user.ID, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("session ID is empty")
	}
	if session.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", session.UserID, user.ID)
	}
	if session.ExpiresAt.Before(time.Now()) {
		t.Error("session already expired")
	}
	if session.ExpiresAt.After(time.Now().Add(25 * time.Hour)) {
		t.Error("session expires too far in future")
	}
}

// TestSessionTokenStoredHashed verifies the raw token never lands in the database:
// the row is keyed by the token's hash, and the raw token still validates.
func TestSessionTokenStoredHashed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := db.CreateUser("Bob", "bob@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	session, err := db.CreateSession(user.ID, 0)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// The raw token must not be stored; its hash must be.
	var rawCount int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", session.ID).Scan(&rawCount); err != nil {
		t.Fatalf("query raw failed: %v", err)
	}
	if rawCount != 0 {
		t.Error("raw session token was stored in the database")
	}
	var hashCount int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", hashSessionToken(session.ID)).Scan(&hashCount); err != nil {
		t.Fatalf("query hash failed: %v", err)
	}
	if hashCount != 1 {
		t.Error("hashed session token was not stored")
	}

	// The raw token still validates.
	got, err := db.ValidateSession(session.ID)
	if err != nil || got == nil || got.ID != user.ID {
		t.Fatalf("ValidateSession(raw token) failed: user=%v err=%v", got, err)
	}
}

func TestValidateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")
	session, _ := db.CreateSession(user.ID, time.Hour)

	// Valid session
	got, err := db.ValidateSession(session.ID)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("ValidateSession returned nil for valid session")
	}
	if got.ID != user.ID {
		t.Errorf("user ID = %q, want %q", got.ID, user.ID)
	}

	// Invalid token
	got, err = db.ValidateSession("invalid-token")
	if err != nil {
		t.Fatalf("ValidateSession for invalid token failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for invalid session token")
	}
}

func TestValidateSession_Expired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")

	// Create expired session (negative TTL for testing)
	session, _ := db.CreateSession(user.ID, time.Millisecond)
	time.Sleep(10 * time.Millisecond) // Wait for expiry

	got, err := db.ValidateSession(session.ID)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired session")
	}
}

func TestDeleteSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")
	session, _ := db.CreateSession(user.ID, time.Hour)

	// Delete session
	err := db.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify deleted
	got, _ := db.ValidateSession(session.ID)
	if got != nil {
		t.Error("session still valid after delete")
	}
}

func TestDeleteUserSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")

	// Create multiple sessions
	s1, _ := db.CreateSession(user.ID, time.Hour)
	s2, _ := db.CreateSession(user.ID, time.Hour)

	// Delete all user sessions
	err := db.DeleteUserSessions(user.ID)
	if err != nil {
		t.Fatalf("DeleteUserSessions failed: %v", err)
	}

	// Verify both deleted
	got, _ := db.ValidateSession(s1.ID)
	if got != nil {
		t.Error("session 1 still valid")
	}
	got, _ = db.ValidateSession(s2.ID)
	if got != nil {
		t.Error("session 2 still valid")
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")

	// Create expired session
	db.CreateSession(user.ID, time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	// Create valid session
	validSession, _ := db.CreateSession(user.ID, time.Hour)

	// Clean expired
	count, err := db.CleanExpiredSessions()
	if err != nil {
		t.Fatalf("CleanExpiredSessions failed: %v", err)
	}
	if count != 1 {
		t.Errorf("cleaned %d sessions, want 1", count)
	}

	// Valid session still works
	got, _ := db.ValidateSession(validSession.ID)
	if got == nil {
		t.Error("valid session was incorrectly cleaned")
	}
}

func TestSessionCookie(t *testing.T) {
	session := &Session{
		ID:        "test-session-token",
		UserID:    "usr_123",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Set cookie
	w := httptest.NewRecorder()
	SetSessionCookie(w, session, false)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Errorf("cookie name = %q, want %q", cookie.Name, SessionCookieName)
	}
	if cookie.Value != session.ID {
		t.Errorf("cookie value = %q, want %q", cookie.Value, session.ID)
	}
	if !cookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Error("cookie should be SameSite=Lax")
	}
}

func TestSessionCookie_Secure(t *testing.T) {
	session := &Session{
		ID:        "test-session-token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	w := httptest.NewRecorder()
	SetSessionCookie(w, session, true) // secure=true

	cookie := w.Result().Cookies()[0]
	if !cookie.Secure {
		t.Error("cookie should be Secure in production")
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearSessionCookie(w, false)

	cookie := w.Result().Cookies()[0]
	if cookie.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1 for deletion", cookie.MaxAge)
	}
}

func TestGetSessionToken(t *testing.T) {
	// With cookie
	req := httptest.NewRequest("GET", "/", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: "my-token",
	})

	token := GetSessionToken(req)
	if token != "my-token" {
		t.Errorf("token = %q, want %q", token, "my-token")
	}

	// Without cookie
	req = httptest.NewRequest("GET", "/", http.NoBody)
	token = GetSessionToken(req)
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestGenerateSessionToken(t *testing.T) {
	t1, err := generateSessionToken()
	if err != nil {
		t.Fatalf("generateSessionToken failed: %v", err)
	}

	t2, _ := generateSessionToken()

	if t1 == t2 {
		t.Error("tokens should be unique")
	}

	// Should be base64 encoded 32 bytes = 44 chars
	if len(t1) < 40 {
		t.Errorf("token too short: %d chars", len(t1))
	}
}
