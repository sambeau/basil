package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// VerificationTokenBytes is the number of random bytes for verification tokens (32 bytes = 256 bits)
	VerificationTokenBytes = 32
	// BcryptCost is the work factor for bcrypt hashing
	BcryptCost = 12
)

// EmailVerification represents an email verification token
type EmailVerification struct {
	ID         string
	UserID     string
	Email      string
	TokenHash  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	SendCount  int
	LastSentAt time.Time
	CreatedAt  time.Time
}

// GenerateVerificationToken generates a cryptographically random verification token
func GenerateVerificationToken() (string, error) {
	b := make([]byte, VerificationTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken hashes a token using bcrypt
func HashToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hashing token: %w", err)
	}
	return string(hash), nil
}

// TokenLookupHash returns a fast, deterministic lookup hash (SHA-256, hex) for a
// verification token. Unlike the bcrypt token_hash it is indexable, letting
// LookupVerificationToken find the single candidate row directly instead of
// bcrypt-comparing against every outstanding token. The token's 256 bits of
// entropy make a plain hash safe here — there is nothing low-entropy to brute force.
func TokenLookupHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// StoreVerificationToken stores a verification token in the database. tokenLookup
// is the indexed SHA-256 lookup hash (see TokenLookupHash); tokenHash is the
// bcrypt hash used as the authoritative comparison.
func (d *DB) StoreVerificationToken(ctx context.Context, userID, email, tokenHash, tokenLookup string, expiresAt time.Time) (string, error) {
	id := generateID("evt_") // email verification token

	query := `
		INSERT INTO email_verifications (id, user_id, email, token_hash, token_lookup, expires_at, last_sent_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := d.db.ExecContext(ctx, query, id, userID, email, tokenHash, tokenLookup, expiresAt, now, now)
	if err != nil {
		return "", fmt.Errorf("storing verification token: %w", err)
	}

	return id, nil
}

// LookupVerificationToken looks up a verification token by token string.
//
// The candidate is found via the indexed token_lookup hash — an O(1) lookup —
// and confirmed with a single bcrypt comparison, instead of bcrypt-scanning every
// outstanding token. Rows created before the token_lookup column existed have a
// NULL lookup and fall back to the (bounded, self-clearing) legacy scan.
func (d *DB) LookupVerificationToken(ctx context.Context, token string) (*EmailVerification, error) {
	lookup := TokenLookupHash(token)

	query := `
		SELECT id, user_id, email, token_hash, expires_at, consumed_at, send_count, last_sent_at, created_at
		FROM email_verifications
		WHERE token_lookup = ? AND consumed_at IS NULL AND expires_at > ?
	`

	rows, err := d.db.QueryContext(ctx, query, lookup, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying verification tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ev, err := scanVerification(rows)
		if err != nil {
			return nil, err
		}
		// Confirm with bcrypt as the authoritative comparison.
		if bcrypt.CompareHashAndPassword([]byte(ev.TokenHash), []byte(token)) == nil {
			return ev, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating verification tokens: %w", err)
	}

	// Fall back to legacy rows that predate the token_lookup column.
	return d.lookupVerificationTokenLegacy(ctx, token)
}

// lookupVerificationTokenLegacy scans only rows without a token_lookup hash
// (created before that column was added). This set is bounded and disappears as
// old tokens expire and are cleaned up, so it cannot be used to force an
// unbounded bcrypt scan.
func (d *DB) lookupVerificationTokenLegacy(ctx context.Context, token string) (*EmailVerification, error) {
	query := `
		SELECT id, user_id, email, token_hash, expires_at, consumed_at, send_count, last_sent_at, created_at
		FROM email_verifications
		WHERE token_lookup IS NULL AND consumed_at IS NULL AND expires_at > ?
	`

	rows, err := d.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying verification tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ev, err := scanVerification(rows)
		if err != nil {
			return nil, err
		}
		if bcrypt.CompareHashAndPassword([]byte(ev.TokenHash), []byte(token)) == nil {
			return ev, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating verification tokens: %w", err)
	}

	return nil, fmt.Errorf("token not found or expired")
}

// scanVerification scans a single email_verifications row (in the column order
// used by the lookup queries) into an EmailVerification.
func scanVerification(rows *sql.Rows) (*EmailVerification, error) {
	var ev EmailVerification
	var consumedAt sql.NullTime
	if err := rows.Scan(&ev.ID, &ev.UserID, &ev.Email, &ev.TokenHash, &ev.ExpiresAt, &consumedAt, &ev.SendCount, &ev.LastSentAt, &ev.CreatedAt); err != nil {
		return nil, fmt.Errorf("scanning verification token: %w", err)
	}
	if consumedAt.Valid {
		ev.ConsumedAt = &consumedAt.Time
	}
	return &ev, nil
}

// ConsumeVerificationToken marks a verification token as consumed
func (d *DB) ConsumeVerificationToken(ctx context.Context, tokenID string) error {
	query := `
		UPDATE email_verifications
		SET consumed_at = ?
		WHERE id = ? AND consumed_at IS NULL
	`

	result, err := d.db.ExecContext(ctx, query, time.Now(), tokenID)
	if err != nil {
		return fmt.Errorf("consuming verification token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("token not found or already consumed")
	}

	return nil
}

// MarkEmailVerified marks a user's email as verified
func (d *DB) MarkEmailVerified(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET email_verified_at = ?
		WHERE id = ?
	`

	_, err := d.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}

	return nil
}

// CleanupExpiredTokens deletes expired verification tokens (for periodic cleanup)
func (d *DB) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM email_verifications
		WHERE expires_at < ?
	`

	result, err := d.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("cleaning up expired tokens: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("checking rows affected: %w", err)
	}

	return count, nil
}

// IncrementSendCount increments the send count for a verification token
func (d *DB) IncrementSendCount(ctx context.Context, tokenID string) error {
	query := `
		UPDATE email_verifications
		SET send_count = send_count + 1, last_sent_at = ?
		WHERE id = ?
	`

	_, err := d.db.ExecContext(ctx, query, time.Now(), tokenID)
	if err != nil {
		return fmt.Errorf("incrementing send count: %w", err)
	}

	return nil
}

// InvalidateUserVerificationTokens invalidates all verification tokens for a user
func (d *DB) InvalidateUserVerificationTokens(ctx context.Context, userID string) error {
	query := `
		UPDATE email_verifications
		SET consumed_at = ?
		WHERE user_id = ? AND consumed_at IS NULL
	`

	_, err := d.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("invalidating user tokens: %w", err)
	}

	return nil
}
