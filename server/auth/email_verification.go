package auth

import (
	"context"
	"crypto/rand"
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

// StoreVerificationToken stores a verification token in the database
func (d *DB) StoreVerificationToken(ctx context.Context, userID, email, tokenHash string, expiresAt time.Time) (string, error) {
	id := generateID("evt_") // email verification token

	query := `
		INSERT INTO email_verifications (id, user_id, email, token_hash, expires_at, last_sent_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := d.db.ExecContext(ctx, query, id, userID, email, tokenHash, expiresAt, now, now)
	if err != nil {
		return "", fmt.Errorf("storing verification token: %w", err)
	}

	return id, nil
}

// LookupVerificationToken looks up a verification token by token string
func (d *DB) LookupVerificationToken(ctx context.Context, token string) (*EmailVerification, error) {
	// Get all unconsumed, non-expired tokens
	query := `
		SELECT id, user_id, email, token_hash, expires_at, consumed_at, send_count, last_sent_at, created_at
		FROM email_verifications
		WHERE consumed_at IS NULL AND expires_at > ?
	`

	rows, err := d.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("querying verification tokens: %w", err)
	}
	defer rows.Close()

	// Check each token hash against the provided token
	for rows.Next() {
		var ev EmailVerification
		var consumedAt sql.NullTime

		err := rows.Scan(&ev.ID, &ev.UserID, &ev.Email, &ev.TokenHash, &ev.ExpiresAt, &consumedAt, &ev.SendCount, &ev.LastSentAt, &ev.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning verification token: %w", err)
		}

		if consumedAt.Valid {
			ev.ConsumedAt = &consumedAt.Time
		}

		// Check if token matches hash
		if err := bcrypt.CompareHashAndPassword([]byte(ev.TokenHash), []byte(token)); err == nil {
			return &ev, nil
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating verification tokens: %w", err)
	}

	return nil, fmt.Errorf("token not found or expired")
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
