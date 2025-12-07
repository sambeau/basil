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
	role TEXT NOT NULL DEFAULT 'editor',
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

CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	key_hash TEXT NOT NULL,
	key_prefix TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	last_used_at TIMESTAMP,
	expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_credentials_user ON credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON recovery_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
`

// migrations tracks schema migrations to apply to existing databases.
var migrations = []string{
	// Migration 1: Add role column to users table
	`ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'editor'`,
	// Migration 2: Create api_keys table
	`CREATE TABLE IF NOT EXISTS api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		key_prefix TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP,
		expires_at TIMESTAMP
	)`,
	// Migration 3: Create api_keys index
	`CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id)`,
}

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

	// Enable WAL mode for concurrent access (CLI + server)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Set busy timeout to wait for locks (5 seconds)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting busy timeout: %w", err)
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

	d := &DB{db: db, path: dbPath}

	// Apply migrations for existing databases
	if err := d.applyMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying migrations: %w", err)
	}

	return d, nil
}

// applyMigrations applies schema migrations to existing databases.
func (d *DB) applyMigrations() error {
	for _, migration := range migrations {
		// Ignore errors - migrations are idempotent (CREATE IF NOT EXISTS, column already exists)
		d.db.Exec(migration)
	}
	return nil
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
// New users default to 'editor' role; use CreateUserWithRole for admin.
func (d *DB) CreateUser(name, email string) (*User, error) {
	return d.CreateUserWithRole(name, email, RoleEditor)
}

// CreateUserWithRole creates a new user with the specified role.
func (d *DB) CreateUserWithRole(name, email, role string) (*User, error) {
	if role == "" {
		role = RoleEditor
	}
	if role != RoleAdmin && role != RoleEditor {
		return nil, fmt.Errorf("invalid role: %s (must be 'admin' or 'editor')", role)
	}

	user := &User{
		ID:        generateID("usr"),
		Name:      name,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}

	_, err := d.db.Exec(
		"INSERT INTO users (id, name, email, role, created_at) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.Name, nullString(user.Email), user.Role, user.CreatedAt,
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
	var role sql.NullString

	err := d.db.QueryRow(
		"SELECT id, name, email, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Name, &email, &role, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	user.Email = email.String
	user.Role = role.String
	if user.Role == "" {
		user.Role = RoleEditor // Default for old records without role
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email address.
func (d *DB) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	var emailVal sql.NullString
	var role sql.NullString

	err := d.db.QueryRow(
		"SELECT id, name, email, role, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Name, &emailVal, &role, &user.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by email: %w", err)
	}

	user.Email = emailVal.String
	user.Role = role.String
	if user.Role == "" {
		user.Role = RoleEditor
	}
	return user, nil
}

// ListUsers returns all users.
func (d *DB) ListUsers() ([]*User, error) {
	rows, err := d.db.Query(
		"SELECT id, name, email, role, created_at FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var email sql.NullString
		var role sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &email, &role, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		user.Email = email.String
		user.Role = role.String
		if user.Role == "" {
			user.Role = RoleEditor
		}
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

// --- Additional user operations ---

// UpdateUser updates a user's name and/or email.
func (d *DB) UpdateUser(id, name, email string) error {
	if name == "" && email == "" {
		return fmt.Errorf("at least one of name or email must be provided")
	}

	// Build update query dynamically
	query := "UPDATE users SET "
	var args []interface{}

	if name != "" {
		query += "name = ?"
		args = append(args, name)
	}
	if email != "" {
		if name != "" {
			query += ", "
		}
		query += "email = ?"
		args = append(args, nullString(email).String)
	}
	query += " WHERE id = ?"
	args = append(args, id)

	result, err := d.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// SetUserRole changes a user's role.
func (d *DB) SetUserRole(id, role string) error {
	if role != RoleAdmin && role != RoleEditor {
		return fmt.Errorf("invalid role: %s (must be 'admin' or 'editor')", role)
	}

	result, err := d.db.Exec("UPDATE users SET role = ? WHERE id = ?", role, id)
	if err != nil {
		return fmt.Errorf("setting user role: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	return nil
}

// CountAdmins returns the count of admin users.
func (d *DB) CountAdmins() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = ?", RoleAdmin).Scan(&count)
	return count, err
}

// HasCredentials checks if a user has any passkey credentials.
func (d *DB) HasCredentials(userID string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM credentials WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting credentials: %w", err)
	}
	return count > 0, nil
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
