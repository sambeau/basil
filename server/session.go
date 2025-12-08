package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sambeau/basil/config"
)

// SessionStore defines the interface for session storage backends
type SessionStore interface {
	// Load retrieves session data from the request
	Load(r *http.Request) (*SessionData, error)
	// Save persists session data to the response
	Save(w http.ResponseWriter, session *SessionData) error
	// Clear removes the session
	Clear(w http.ResponseWriter) error
}

// CookieSessionStore stores sessions in encrypted cookies
type CookieSessionStore struct {
	config *config.SessionConfig
	secret string
}

// NewCookieSessionStore creates a new cookie-based session store
func NewCookieSessionStore(cfg *config.SessionConfig, secret string) *CookieSessionStore {
	return &CookieSessionStore{
		config: cfg,
		secret: secret,
	}
}

// Load retrieves and decrypts session data from the cookie
func (s *CookieSessionStore) Load(r *http.Request) (*SessionData, error) {
	cookie, err := r.Cookie(s.config.CookieName)
	if err == http.ErrNoCookie {
		// No cookie - return fresh session
		return NewSessionData(s.config.MaxAge), nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt session
	session, err := decryptSession(cookie.Value, s.secret)
	if err != nil {
		// Invalid/tampered cookie - return fresh session
		// This is not an error - could be old key, corrupted data, etc.
		return NewSessionData(s.config.MaxAge), nil
	}

	// Check expiration
	if session.IsExpired() {
		return NewSessionData(s.config.MaxAge), nil
	}

	return session, nil
}

// Save encrypts and stores session data in a cookie
func (s *CookieSessionStore) Save(w http.ResponseWriter, session *SessionData) error {
	// Encrypt session
	encrypted, err := encryptSession(session, s.secret)
	if err != nil {
		return fmt.Errorf("failed to encrypt session: %w", err)
	}

	// Build cookie
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    encrypted,
		Path:     "/",
		MaxAge:   int(s.config.MaxAge.Seconds()),
		Secure:   s.isSecure(),
		HttpOnly: s.config.HttpOnly,
		SameSite: parseSameSite(s.config.SameSite),
	}

	http.SetCookie(w, cookie)
	return nil
}

// Clear removes the session cookie
func (s *CookieSessionStore) Clear(w http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:     s.config.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		Secure:   s.isSecure(),
		HttpOnly: s.config.HttpOnly,
		SameSite: parseSameSite(s.config.SameSite),
	}

	http.SetCookie(w, cookie)
	return nil
}

// isSecure returns the Secure flag, defaulting to true if not explicitly set
func (s *CookieSessionStore) isSecure() bool {
	if s.config.Secure != nil {
		return *s.config.Secure
	}
	return true // Default to secure in production
}

// parseSameSite converts string to http.SameSite
func parseSameSite(s string) http.SameSite {
	switch s {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// Session provides a request-scoped session interface for Parsley handlers
type Session struct {
	data    *SessionData
	store   SessionStore
	writer  http.ResponseWriter
	dirty   bool
	cleared bool
}

// NewSession creates a new session wrapper
func NewSession(data *SessionData, store SessionStore, w http.ResponseWriter) *Session {
	return &Session{
		data:   data,
		store:  store,
		writer: w,
	}
}

// Get retrieves a value from the session
func (s *Session) Get(key string) interface{} {
	return s.data.Data[key]
}

// Set stores a value in the session
func (s *Session) Set(key string, value interface{}) {
	s.data.Data[key] = value
	s.dirty = true
}

// Delete removes a value from the session
func (s *Session) Delete(key string) {
	delete(s.data.Data, key)
	s.dirty = true
}

// Clear removes all session data
func (s *Session) Clear() {
	s.data.Data = make(map[string]interface{})
	s.data.Flash = make(map[string]string)
	s.cleared = true
	s.dirty = true
}

// All returns all session data
func (s *Session) All() map[string]interface{} {
	return s.data.Data
}

// Has checks if a key exists in the session
func (s *Session) Has(key string) bool {
	_, ok := s.data.Data[key]
	return ok
}

// Flash sets a flash message (one-time message, cleared after read)
func (s *Session) Flash(key string, message string) {
	s.data.Flash[key] = message
	s.dirty = true
}

// GetFlash retrieves and removes a flash message
func (s *Session) GetFlash(key string) (string, bool) {
	msg, ok := s.data.Flash[key]
	if ok {
		delete(s.data.Flash, key)
		s.dirty = true
	}
	return msg, ok
}

// GetAllFlash retrieves and removes all flash messages
func (s *Session) GetAllFlash() map[string]string {
	flash := s.data.Flash
	if len(flash) > 0 {
		s.data.Flash = make(map[string]string)
		s.dirty = true
	}
	return flash
}

// HasFlash checks if any flash messages exist
func (s *Session) HasFlash() bool {
	return len(s.data.Flash) > 0
}

// Commit saves the session if it has been modified
func (s *Session) Commit() error {
	if !s.dirty {
		return nil
	}

	if s.cleared && len(s.data.Data) == 0 && len(s.data.Flash) == 0 {
		// Session was cleared and is empty - delete the cookie
		return s.store.Clear(s.writer)
	}

	return s.store.Save(s.writer, s.data)
}

// Regenerate creates a new session ID while preserving data
// For cookie sessions, this just updates the expiration
func (s *Session) Regenerate(maxAge time.Duration) {
	s.data.ExpiresAt = time.Now().Add(maxAge)
	s.dirty = true
}

// IsDirty returns true if the session has been modified
func (s *Session) IsDirty() bool {
	return s.dirty
}
