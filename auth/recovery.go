package auth

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultRecoveryCodeCount is the number of recovery codes generated.
	DefaultRecoveryCodeCount = 8

	// RecoveryCodeLength is the length of each segment (3 segments of 4 chars).
	RecoveryCodeSegmentLength = 4
	RecoveryCodeSegments      = 3
)

// recoveryCodeChars are the characters used in recovery codes.
// Excludes ambiguous characters: 0, O, 1, I, L
var recoveryCodeChars = []byte("23456789ABCDEFGHJKMNPQRSTUVWXYZ")

// GenerateRecoveryCodes creates new recovery codes for a user.
// Returns the plaintext codes (shown once to user) and stores hashes in database.
func (d *DB) GenerateRecoveryCodes(userID string, count int) ([]string, error) {
	if count == 0 {
		count = DefaultRecoveryCodeCount
	}

	// Delete existing codes
	_, err := d.db.Exec("DELETE FROM recovery_codes WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("deleting old recovery codes: %w", err)
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code := generateRecoveryCode()
		codes[i] = code

		// Hash the code for storage
		hash, err := bcrypt.GenerateFromPassword([]byte(normalizeCode(code)), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hashing recovery code: %w", err)
		}

		_, err = d.db.Exec(
			"INSERT INTO recovery_codes (id, user_id, code_hash, created_at) VALUES (?, ?, ?, ?)",
			generateID("rec"), userID, string(hash), time.Now().UTC(),
		)
		if err != nil {
			return nil, fmt.Errorf("saving recovery code: %w", err)
		}
	}

	return codes, nil
}

// ValidateRecoveryCode checks if a recovery code is valid and burns it on success.
// Returns true if the code was valid and has been consumed.
func (d *DB) ValidateRecoveryCode(userID, code string) (bool, error) {
	normalized := normalizeCode(code)

	// Get all unused codes for user
	rows, err := d.db.Query(
		"SELECT id, code_hash FROM recovery_codes WHERE user_id = ? AND used_at IS NULL",
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("querying recovery codes: %w", err)
	}

	// Collect codes to check (need to close rows before updating)
	type codeRecord struct {
		id   string
		hash string
	}
	var codes []codeRecord
	for rows.Next() {
		var rec codeRecord
		if err := rows.Scan(&rec.id, &rec.hash); err != nil {
			rows.Close()
			return false, fmt.Errorf("scanning recovery code: %w", err)
		}
		codes = append(codes, rec)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return false, err
	}

	// Check each code
	for _, rec := range codes {
		if bcrypt.CompareHashAndPassword([]byte(rec.hash), []byte(normalized)) == nil {
			// Mark as used
			_, err := d.db.Exec(
				"UPDATE recovery_codes SET used_at = ? WHERE id = ?",
				time.Now().UTC(), rec.id,
			)
			if err != nil {
				return false, fmt.Errorf("marking recovery code as used: %w", err)
			}
			return true, nil
		}
	}

	return false, nil
}

// GetRecoveryCodeCount returns the count of unused recovery codes for a user.
func (d *DB) GetRecoveryCodeCount(userID string) (int, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM recovery_codes WHERE user_id = ? AND used_at IS NULL",
		userID,
	).Scan(&count)
	return count, err
}

// generateRecoveryCode creates a single recovery code in format XXXX-XXXX-XXXX.
func generateRecoveryCode() string {
	segments := make([]string, RecoveryCodeSegments)
	for i := 0; i < RecoveryCodeSegments; i++ {
		segment := make([]byte, RecoveryCodeSegmentLength)
		for j := 0; j < RecoveryCodeSegmentLength; j++ {
			idx := randByte() % byte(len(recoveryCodeChars))
			segment[j] = recoveryCodeChars[idx]
		}
		segments[i] = string(segment)
	}
	return strings.Join(segments, "-")
}

// normalizeCode removes dashes and converts to uppercase for comparison.
func normalizeCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(code, "-", ""))
}

// randByte returns a cryptographically random byte.
func randByte() byte {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]
}
