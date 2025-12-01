package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

const (
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "__basil_session"

	// DefaultSessionTTL is the default session duration.
	DefaultSessionTTL = 24 * time.Hour
)

// CreateSession creates a new session for the user and stores it in the database.
func (d *DB) CreateSession(userID string, ttl time.Duration) (*Session, error) {
	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	session := &Session{
		ID:        token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	_, err = d.db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		session.ID, session.UserID, session.CreatedAt, session.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return session, nil
}

// ValidateSession checks if a session token is valid and returns the associated user.
// Returns nil, nil if the session doesn't exist or is expired.
func (d *DB) ValidateSession(token string) (*User, error) {
	var session Session
	err := d.db.QueryRow(
		"SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?",
		token,
	).Scan(&session.ID, &session.UserID, &session.CreatedAt, &session.ExpiresAt)

	if err != nil {
		return nil, nil // Session not found
	}

	// Check expiry
	if time.Now().UTC().After(session.ExpiresAt) {
		// Clean up expired session
		d.DeleteSession(token)
		return nil, nil
	}

	// Load user
	return d.GetUser(session.UserID)
}

// DeleteSession removes a session (logout).
func (d *DB) DeleteSession(token string) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE id = ?", token)
	return err
}

// DeleteUserSessions removes all sessions for a user.
func (d *DB) DeleteUserSessions(userID string) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// CleanExpiredSessions removes all expired sessions.
func (d *DB) CleanExpiredSessions() (int64, error) {
	result, err := d.db.Exec(
		"DELETE FROM sessions WHERE expires_at < ?",
		time.Now().UTC(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// generateSessionToken creates a cryptographically random session token.
func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// --- Cookie helpers ---

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, session *Session, secure bool) {
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetSessionToken extracts the session token from the request cookie.
func GetSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
