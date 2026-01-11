package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sambeau/basil/server/config"
)

func testSessionConfig() *config.SessionConfig {
	secure := false
	return &config.SessionConfig{
		Store:      "cookie",
		CookieName: "_test_session",
		MaxAge:     time.Hour,
		Secure:     &secure,
		HttpOnly:   true,
		SameSite:   "Lax",
	}
}

func TestCookieSessionStore_NewSession(t *testing.T) {
	cfg := testSessionConfig()
	store := NewCookieSessionStore(cfg, "test-secret", true) // devMode=true for tests

	// Request with no cookie
	req := httptest.NewRequest("GET", "/", nil)
	session, err := store.Load(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.Data) != 0 {
		t.Error("expected empty session data")
	}
	if session.IsExpired() {
		t.Error("new session should not be expired")
	}
}

func TestCookieSessionStore_SaveAndLoad(t *testing.T) {
	cfg := testSessionConfig()
	secret := "test-secret-key"
	store := NewCookieSessionStore(cfg, secret, true)

	// Create and save a session
	session := NewSessionData(time.Hour)
	session.Data["user_id"] = float64(123)
	session.Data["username"] = "testuser"

	w := httptest.NewRecorder()
	if err := store.Save(w, session); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Check cookie was set
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "_test_session" {
		t.Errorf("expected cookie name '_test_session', got '%s'", cookies[0].Name)
	}

	// Load session from cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookies[0])

	loaded, err := store.Load(req)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Data["user_id"] != float64(123) {
		t.Errorf("expected user_id 123, got %v", loaded.Data["user_id"])
	}
	if loaded.Data["username"] != "testuser" {
		t.Errorf("expected username 'testuser', got %v", loaded.Data["username"])
	}
}

func TestCookieSessionStore_Clear(t *testing.T) {
	cfg := testSessionConfig()
	store := NewCookieSessionStore(cfg, "test-secret", true)

	w := httptest.NewRecorder()
	if err := store.Clear(w); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", cookies[0].MaxAge)
	}
}

func TestCookieSessionStore_ExpiredSession(t *testing.T) {
	cfg := testSessionConfig()
	secret := "test-secret"
	store := NewCookieSessionStore(cfg, secret, true)

	// Create an expired session
	session := &SessionData{
		Data:      map[string]interface{}{"test": "value"},
		Flash:     make(map[string]string),
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
	}

	w := httptest.NewRecorder()
	if err := store.Save(w, session); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load should return fresh session
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(w.Result().Cookies()[0])

	loaded, err := store.Load(req)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Should be fresh session (empty data)
	if len(loaded.Data) != 0 {
		t.Errorf("expected empty data for expired session, got %v", loaded.Data)
	}
}

func TestCookieSessionStore_InvalidCookie(t *testing.T) {
	cfg := testSessionConfig()
	store := NewCookieSessionStore(cfg, "test-secret", true)

	// Request with invalid cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "_test_session",
		Value: "invalid-not-base64!!!",
	})

	// Should return fresh session, not error
	session, err := store.Load(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(session.Data) != 0 {
		t.Error("expected fresh session for invalid cookie")
	}
}

func TestCookieSessionStore_WrongSecret(t *testing.T) {
	cfg := testSessionConfig()
	store1 := NewCookieSessionStore(cfg, "secret1", true)
	store2 := NewCookieSessionStore(cfg, "secret2", true)

	// Save with store1
	session := NewSessionData(time.Hour)
	session.Data["test"] = "value"

	w := httptest.NewRecorder()
	store1.Save(w, session)

	// Load with store2 (different secret)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(w.Result().Cookies()[0])

	loaded, err := store2.Load(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return fresh session
	if len(loaded.Data) != 0 {
		t.Error("expected fresh session when decryption fails")
	}
}

func TestSession_GetSet(t *testing.T) {
	data := NewSessionData(time.Hour)
	session := NewSession(data, nil, nil)

	// Set and get
	session.Set("key", "value")
	if session.Get("key") != "value" {
		t.Errorf("expected 'value', got %v", session.Get("key"))
	}

	// Get non-existent
	if session.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent key")
	}

	// Has
	if !session.Has("key") {
		t.Error("expected Has to return true")
	}
	if session.Has("nonexistent") {
		t.Error("expected Has to return false for nonexistent")
	}
}

func TestSession_Delete(t *testing.T) {
	data := NewSessionData(time.Hour)
	session := NewSession(data, nil, nil)

	session.Set("key", "value")
	session.Delete("key")

	if session.Has("key") {
		t.Error("expected key to be deleted")
	}
}

func TestSession_Clear(t *testing.T) {
	data := NewSessionData(time.Hour)
	data.Data["key1"] = "value1"
	data.Data["key2"] = "value2"
	data.Flash["msg"] = "hello"

	session := NewSession(data, nil, nil)
	session.Clear()

	if len(session.All()) != 0 {
		t.Error("expected empty data after clear")
	}
	if session.HasFlash() {
		t.Error("expected no flash after clear")
	}
}

func TestSession_Flash(t *testing.T) {
	data := NewSessionData(time.Hour)
	session := NewSession(data, nil, nil)

	// Set flash
	session.Flash("success", "Operation completed!")

	if !session.HasFlash() {
		t.Error("expected HasFlash to return true")
	}

	// Get flash (should remove it)
	msg, ok := session.GetFlash("success")
	if !ok || msg != "Operation completed!" {
		t.Errorf("expected 'Operation completed!', got '%s'", msg)
	}

	// Second get should fail
	_, ok = session.GetFlash("success")
	if ok {
		t.Error("flash should be consumed after first read")
	}
}

func TestSession_GetAllFlash(t *testing.T) {
	data := NewSessionData(time.Hour)
	session := NewSession(data, nil, nil)

	session.Flash("success", "Saved!")
	session.Flash("info", "Note this")

	flash := session.GetAllFlash()
	if len(flash) != 2 {
		t.Errorf("expected 2 flash messages, got %d", len(flash))
	}

	// Should be cleared
	if session.HasFlash() {
		t.Error("flash should be cleared after GetAllFlash")
	}
}

func TestSession_Dirty(t *testing.T) {
	data := NewSessionData(time.Hour)
	session := NewSession(data, nil, nil)

	if session.IsDirty() {
		t.Error("new session should not be dirty")
	}

	session.Set("key", "value")
	if !session.IsDirty() {
		t.Error("session should be dirty after Set")
	}
}

func TestSession_Commit(t *testing.T) {
	cfg := testSessionConfig()
	store := NewCookieSessionStore(cfg, "test-secret", true)

	// Test 1: Not dirty - commit should be no-op
	data1 := NewSessionData(time.Hour)
	w1 := httptest.NewRecorder()
	session1 := NewSession(data1, store, w1)

	if err := session1.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if len(w1.Result().Cookies()) != 0 {
		t.Error("expected no cookie for non-dirty session")
	}

	// Test 2: Make dirty and commit
	data2 := NewSessionData(time.Hour)
	w2 := httptest.NewRecorder()
	session2 := NewSession(data2, store, w2)

	session2.Set("key", "value")
	if err := session2.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if len(w2.Result().Cookies()) != 1 {
		t.Error("expected cookie after commit")
	}
}

func TestSession_CommitCleared(t *testing.T) {
	cfg := testSessionConfig()
	store := NewCookieSessionStore(cfg, "test-secret", true)

	data := NewSessionData(time.Hour)
	data.Data["existing"] = "data"
	w := httptest.NewRecorder()
	session := NewSession(data, store, w)

	session.Clear()
	if err := session.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Should have delete cookie (MaxAge -1)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("expected 1 cookie")
	}
	if cookies[0].MaxAge != -1 {
		t.Errorf("expected MaxAge -1 for cleared session, got %d", cookies[0].MaxAge)
	}
}

func TestSession_Regenerate(t *testing.T) {
	data := NewSessionData(time.Hour)
	originalExpiry := data.ExpiresAt

	time.Sleep(10 * time.Millisecond) // Ensure time difference

	session := NewSession(data, nil, nil)
	session.Regenerate(2 * time.Hour)

	if !session.IsDirty() {
		t.Error("session should be dirty after Regenerate")
	}
	if !data.ExpiresAt.After(originalExpiry) {
		t.Error("expiry should be extended")
	}
}

func TestParseSameSite(t *testing.T) {
	tests := []struct {
		input    string
		expected http.SameSite
	}{
		{"Strict", http.SameSiteStrictMode},
		{"Lax", http.SameSiteLaxMode},
		{"None", http.SameSiteNoneMode},
		{"", http.SameSiteLaxMode},
		{"invalid", http.SameSiteLaxMode},
	}

	for _, tt := range tests {
		result := parseSameSite(tt.input)
		if result != tt.expected {
			t.Errorf("parseSameSite(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestCookieSessionStore_SecureFlag(t *testing.T) {
	t.Run("dev mode defaults to insecure", func(t *testing.T) {
		cfg := &config.SessionConfig{
			CookieName: "_test_session",
			MaxAge:     time.Hour,
			// Secure not set
		}
		store := NewCookieSessionStore(cfg, "secret", true) // devMode=true
		if store.isSecure() {
			t.Error("expected Secure=false in dev mode when not explicitly set")
		}
	})

	t.Run("production mode defaults to secure", func(t *testing.T) {
		cfg := &config.SessionConfig{
			CookieName: "_test_session",
			MaxAge:     time.Hour,
			// Secure not set
		}
		store := NewCookieSessionStore(cfg, "secret", false) // devMode=false
		if !store.isSecure() {
			t.Error("expected Secure=true in production mode when not explicitly set")
		}
	})

	t.Run("explicit secure=true overrides dev mode", func(t *testing.T) {
		secure := true
		cfg := &config.SessionConfig{
			CookieName: "_test_session",
			MaxAge:     time.Hour,
			Secure:     &secure,
		}
		store := NewCookieSessionStore(cfg, "secret", true) // devMode=true
		if !store.isSecure() {
			t.Error("expected Secure=true when explicitly set, even in dev mode")
		}
	})

	t.Run("explicit secure=false overrides production mode", func(t *testing.T) {
		secure := false
		cfg := &config.SessionConfig{
			CookieName: "_test_session",
			MaxAge:     time.Hour,
			Secure:     &secure,
		}
		store := NewCookieSessionStore(cfg, "secret", false) // devMode=false
		if store.isSecure() {
			t.Error("expected Secure=false when explicitly set, even in production mode")
		}
	})
}
