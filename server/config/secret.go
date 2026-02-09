package config

import (
	"crypto/rand"
	"encoding/base64"
	"sync"

	"gopkg.in/yaml.v3"
)

// SecretString wraps a string value that should be treated as sensitive.
// Secret values are hidden in DevTools and logs.
type SecretString struct {
	value    string
	isSecret bool
}

// NewSecretString creates a new SecretString with the given value.
func NewSecretString(value string) SecretString {
	return SecretString{value: value, isSecret: true}
}

// Value returns the actual secret value.
func (s SecretString) Value() string {
	return s.value
}

// IsSecret returns true if this value should be treated as sensitive.
func (s SecretString) IsSecret() bool {
	return s.isSecret
}

// String returns a redacted representation for logging.
func (s SecretString) String() string {
	if s.isSecret && s.value != "" {
		return "[hidden]"
	}
	return s.value
}

// IsAuto returns true if this is the special "auto" value for auto-generation.
func (s SecretString) IsAuto() bool {
	return s.value == "auto"
}

// UnmarshalYAML implements yaml.Unmarshaler to handle the !secret tag.
func (s *SecretString) UnmarshalYAML(node *yaml.Node) error {
	// Check if this node has the !secret tag
	if node.Tag == "!secret" {
		s.isSecret = true
	}

	// Decode the actual value
	var value string
	if err := node.Decode(&value); err != nil {
		return err
	}
	s.value = value
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (s SecretString) MarshalYAML() (any, error) {
	if s.isSecret {
		// Return a node with the !secret tag
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!secret",
			Value: s.value,
		}, nil
	}
	return s.value, nil
}

// SecretTracker tracks which config paths contain secret values.
// This is used by DevTools to know which values to hide.
type SecretTracker struct {
	mu    sync.RWMutex
	paths map[string]bool
}

// NewSecretTracker creates a new SecretTracker.
func NewSecretTracker() *SecretTracker {
	return &SecretTracker{
		paths: make(map[string]bool),
	}
}

// MarkSecret marks a config path as containing a secret value.
func (t *SecretTracker) MarkSecret(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paths[path] = true
}

// IsSecret returns true if the given path contains a secret value.
func (t *SecretTracker) IsSecret(path string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.paths[path]
}

// Paths returns all paths that contain secret values.
func (t *SecretTracker) Paths() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]string, 0, len(t.paths))
	for path := range t.paths {
		result = append(result, path)
	}
	return result
}

// GenerateSecureSecret generates a cryptographically secure random secret.
// Returns a 32-byte (256-bit) base64-encoded string.
func GenerateSecureSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// ResolveSecretValue resolves a SecretString value, handling the "auto" case.
// If the value is "auto", it generates a secure random value.
// If envValue is provided and non-empty, it overrides the configured value.
func ResolveSecretValue(s SecretString, envValue string) (string, error) {
	// Environment variable takes precedence
	if envValue != "" {
		return envValue, nil
	}

	// Handle "auto" - generate secure random
	if s.IsAuto() {
		return GenerateSecureSecret()
	}

	// Use configured value
	return s.Value(), nil
}
