// Package auth provides passkey-based authentication for Basil.
package auth

import "time"

// User represents an authenticated user.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"` // Optional
	Role      string    `json:"role"`            // "admin" or "editor"
	CreatedAt time.Time `json:"created_at"`
}

// APIKey represents an API key for a user.
type APIKey struct {
	ID         string     `json:"id"`         // "key_xyz789"
	UserID     string     `json:"user_id"`    // Owner
	Name       string     `json:"name"`       // Label, e.g., "MacBook Git"
	KeyHash    string     `json:"-"`          // bcrypt hash, never exposed
	KeyPrefix  string     `json:"key_prefix"` // "bsl_...k2m9" for display
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// Role constants
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
)

// Credential represents a WebAuthn credential (passkey) for a user.
type Credential struct {
	ID              []byte    `json:"id"`               // WebAuthn credential ID
	UserID          string    `json:"user_id"`          // Owner of this credential
	PublicKey       []byte    `json:"public_key"`       // Public key (not secret)
	SignCount       uint32    `json:"sign_count"`       // Replay protection counter
	Transports      []string  `json:"transports"`       // e.g., ["internal", "usb"]
	AttestationType string    `json:"attestation_type"` // e.g., "none", "packed"
	CreatedAt       time.Time `json:"created_at"`
}

// Session represents an authenticated session.
type Session struct {
	ID        string    `json:"id"`      // Random token
	UserID    string    `json:"user_id"` // Session owner
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RecoveryCode represents a one-time recovery code.
type RecoveryCode struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CodeHash  string    `json:"-"` // bcrypt hash, never exposed
	UsedAt    time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Config holds authentication configuration.
type Config struct {
	Enabled      bool          `yaml:"enabled"`
	Registration string        `yaml:"registration"` // "open" or "closed"
	SessionTTL   time.Duration `yaml:"session_ttl"`  // Default: 24h
	DatabasePath string        `yaml:"-"`            // Set automatically
}

// DefaultConfig returns auth config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Enabled:      false,
		Registration: "closed",
		SessionTTL:   24 * time.Hour,
	}
}
