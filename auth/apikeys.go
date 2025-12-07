package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// GenerateAPIKey generates a new API key and returns the plaintext, hash, and prefix.
// The plaintext should be shown to the user once and never stored.
// The hash is stored in the database for validation.
// The prefix is stored for display purposes (e.g., "bsl_...k2m9").
func GenerateAPIKey() (plaintext string, hash string, prefix string, err error) {
	// Generate 32 random bytes
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	// Format: bsl_live_<base64url encoded>
	encoded := base64.RawURLEncoding.EncodeToString(random)
	plaintext = "bsl_live_" + encoded

	// Hash for storage
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", fmt.Errorf("hashing API key: %w", err)
	}
	hash = string(hashBytes)

	// Prefix for display: first 12 chars + "..." + last 4 chars
	prefix = plaintext[:12] + "..." + plaintext[len(plaintext)-4:]

	return plaintext, hash, prefix, nil
}

// CreateAPIKey creates a new API key for a user and returns it with the plaintext key.
// The plaintext key is only available at creation time.
func (d *DB) CreateAPIKey(userID, name string) (*APIKey, string, error) {
	// Verify user exists
	user, err := d.GetUser(userID)
	if err != nil {
		return nil, "", fmt.Errorf("checking user: %w", err)
	}
	if user == nil {
		return nil, "", fmt.Errorf("user not found: %s", userID)
	}

	// Generate key
	plaintext, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	key := &APIKey{
		ID:        generateID("key"),
		UserID:    userID,
		Name:      name,
		KeyHash:   hash,
		KeyPrefix: prefix,
		CreatedAt: time.Now().UTC(),
	}

	_, err = d.db.Exec(
		`INSERT INTO api_keys (id, user_id, name, key_hash, key_prefix, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		key.ID, key.UserID, key.Name, key.KeyHash, key.KeyPrefix, key.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("creating API key: %w", err)
	}

	return key, plaintext, nil
}

// GetAPIKeys returns all API keys for a user.
func (d *DB) GetAPIKeys(userID string) ([]*APIKey, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, key_hash, key_prefix, created_at, last_used_at, expires_at
		 FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

// GetAllAPIKeys returns all API keys in the database.
// Used for key validation (bcrypt requires checking all keys).
func (d *DB) GetAllAPIKeys() ([]*APIKey, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, key_hash, key_prefix, created_at, last_used_at, expires_at
		 FROM api_keys`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing all API keys: %w", err)
	}
	defer rows.Close()

	return scanAPIKeys(rows)
}

// GetAPIKey retrieves an API key by ID.
func (d *DB) GetAPIKey(id string) (*APIKey, error) {
	key := &APIKey{}
	var lastUsed, expires sql.NullTime

	err := d.db.QueryRow(
		`SELECT id, user_id, name, key_hash, key_prefix, created_at, last_used_at, expires_at
		 FROM api_keys WHERE id = ?`,
		id,
	).Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.KeyPrefix,
		&key.CreatedAt, &lastUsed, &expires)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting API key: %w", err)
	}

	if lastUsed.Valid {
		key.LastUsedAt = &lastUsed.Time
	}
	if expires.Valid {
		key.ExpiresAt = &expires.Time
	}

	return key, nil
}

// DeleteAPIKey deletes an API key by ID.
func (d *DB) DeleteAPIKey(id string) error {
	result, err := d.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting API key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	return nil
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key.
func (d *DB) UpdateAPIKeyLastUsed(id string) error {
	_, err := d.db.Exec(
		"UPDATE api_keys SET last_used_at = ? WHERE id = ?",
		time.Now().UTC(), id,
	)
	return err
}

// ValidateAPIKey validates an API key and returns the associated user.
// Returns nil, nil if the key is invalid or expired.
func (d *DB) ValidateAPIKey(key string) (*User, error) {
	// Key format check
	if !strings.HasPrefix(key, "bsl_live_") {
		return nil, nil
	}

	// Get all keys and check hash (bcrypt doesn't allow direct lookup)
	keys, err := d.GetAllAPIKeys()
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		// Check if expired
		if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
			continue
		}

		// Check hash
		if bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(key)) == nil {
			// Update last used
			d.UpdateAPIKeyLastUsed(k.ID)
			return d.GetUser(k.UserID)
		}
	}

	return nil, nil
}

// scanAPIKeys scans API key rows into a slice.
func scanAPIKeys(rows *sql.Rows) ([]*APIKey, error) {
	var keys []*APIKey
	for rows.Next() {
		key := &APIKey{}
		var lastUsed, expires sql.NullTime
		if err := rows.Scan(&key.ID, &key.UserID, &key.Name, &key.KeyHash, &key.KeyPrefix,
			&key.CreatedAt, &lastUsed, &expires); err != nil {
			return nil, fmt.Errorf("scanning API key: %w", err)
		}
		if lastUsed.Valid {
			key.LastUsedAt = &lastUsed.Time
		}
		if expires.Valid {
			key.ExpiresAt = &expires.Time
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}
