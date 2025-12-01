package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// SQLite driver
	_ "modernc.org/sqlite"
)

// DB wraps the auth database connection.
type DB struct {
	db   *sql.DB
	path string
}

// schema defines the auth database tables.
const schema = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credentials (
	id BLOB PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	public_key BLOB NOT NULL,
	sign_count INTEGER DEFAULT 0,
	transports TEXT,
	attestation_type TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS recovery_codes (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	code_hash TEXT NOT NULL,
	used_at TIMESTAMP,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_credentials_user ON credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON recovery_codes(user_id);
`

// OpenDB opens the auth database, creating it if necessary.
// The database is stored separately from the app database for security.
func OpenDB(basePath string) (*DB, error) {
	// Auth database is always .basil-auth.db in the config directory
	dbPath := filepath.Join(basePath, ".basil-auth.db")

	// Create database file with restrictive permissions
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("creating auth database: %w", err)
		}
		f.Close()
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening auth database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &DB{db: db, path: dbPath}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// generateID creates a random ID with the given prefix.
func generateID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// --- User operations ---

// CreateUser creates a new user and returns the user with generated ID.
func (d *DB) CreateUser(name, email string) (*User, error) {
	user := &User{
		ID:        generateID("usr"),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}

	_, err := d.db.Exec(
		"INSERT INTO users (id, name, email, created_at) VALUES (?, ?, ?, ?)",
		user.ID, user.Name, nullString(user.Email), user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID.
func (d *DB) GetUser(id string) (*User, error) {
	user := &User{}
	var email sql.NullString

	err := d.db.QueryRow(
		"SELECT id, name, email, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Name, &email, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	user.Email = email.String
	return user, nil
}

// GetUserByEmail retrieves a user by email address.
func (d *DB) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	var emailVal sql.NullString

	err := d.db.QueryRow(
		"SELECT id, name, email, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Name, &emailVal, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	user.Email = emailVal.String
	return user, nil
}

// ListUsers returns all users.
func (d *DB) ListUsers() ([]*User, error) {
	rows, err := d.db.Query(
		"SELECT id, name, email, created_at FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var email sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &email, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		user.Email = email.String
		users = append(users, user)
	}

	return users, rows.Err()
}

// DeleteUser deletes a user and all their credentials/sessions/recovery codes.
func (d *DB) DeleteUser(id string) error {
	result, err := d.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// UserCount returns the total number of users.
func (d *DB) UserCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// --- Credential operations ---

// SaveCredential stores a WebAuthn credential for a user.
func (d *DB) SaveCredential(cred *Credential) error {
	transports := ""
	if len(cred.Transports) > 0 {
		// Simple join for storage
		for i, t := range cred.Transports {
			if i > 0 {
				transports += ","
			}
			transports += t
		}
	}

	_, err := d.db.Exec(
		`INSERT INTO credentials (id, user_id, public_key, sign_count, transports, attestation_type, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cred.ID, cred.UserID, cred.PublicKey, cred.SignCount,
		nullString(transports), nullString(cred.AttestationType), cred.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("saving credential: %w", err)
	}

	return nil
}

// GetCredentialsByUser returns all credentials for a user.
func (d *DB) GetCredentialsByUser(userID string) ([]*Credential, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, public_key, sign_count, transports, attestation_type, created_at
		 FROM credentials WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %w", err)
	}
	defer rows.Close()

	var creds []*Credential
	for rows.Next() {
		cred := &Credential{}
		var transports, attestationType sql.NullString
		if err := rows.Scan(&cred.ID, &cred.UserID, &cred.PublicKey, &cred.SignCount,
			&transports, &attestationType, &cred.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning credential: %w", err)
		}
		if transports.String != "" {
			cred.Transports = splitString(transports.String, ",")
		}
		cred.AttestationType = attestationType.String
		creds = append(creds, cred)
	}

	return creds, rows.Err()
}

// GetCredential retrieves a credential by ID.
func (d *DB) GetCredential(id []byte) (*Credential, error) {
	cred := &Credential{}
	var transports, attestationType sql.NullString

	err := d.db.QueryRow(
		`SELECT id, user_id, public_key, sign_count, transports, attestation_type, created_at
		 FROM credentials WHERE id = ?`,
		id,
	).Scan(&cred.ID, &cred.UserID, &cred.PublicKey, &cred.SignCount,
		&transports, &attestationType, &cred.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting credential: %w", err)
	}

	if transports.String != "" {
		cred.Transports = splitString(transports.String, ",")
	}
	cred.AttestationType = attestationType.String
	return cred, nil
}

// UpdateCredentialSignCount updates the sign count for replay protection.
func (d *DB) UpdateCredentialSignCount(id []byte, signCount uint32) error {
	_, err := d.db.Exec(
		"UPDATE credentials SET sign_count = ? WHERE id = ?",
		signCount, id,
	)
	return err
}

// --- Helper functions ---

// nullString converts an empty string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// splitString splits a string by separator.
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}
