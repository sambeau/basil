package auth

import (
	"context"
	"fmt"
	"time"
)

// RateLimitResult holds the result of a rate limit check
type RateLimitResult struct {
	Allowed       bool
	Reason        string
	NextAvailable time.Time
	CurrentCount  int
	Limit         int
}

// CheckVerificationRateLimit checks if a user can request another verification email
// Enforces:
// - Per-user cooldown (5 minutes between sends)
// - Per-user daily limit (10 sends/day)
// - Per-email daily limit (20 sends/day across all users)
func (d *DB) CheckVerificationRateLimit(ctx context.Context, userID, email string, cooldown time.Duration, dailyLimit int) (*RateLimitResult, error) {
	now := time.Now()

	// Check per-user cooldown (most recent send)
	query := `
		SELECT last_sent_at
		FROM email_verifications
		WHERE user_id = ?
		ORDER BY last_sent_at DESC
		LIMIT 1
	`

	var lastSentAt time.Time
	err := d.db.QueryRowContext(ctx, query, userID).Scan(&lastSentAt)
	if err == nil {
		// Token exists, check cooldown
		nextAvailable := lastSentAt.Add(cooldown)
		if now.Before(nextAvailable) {
			return &RateLimitResult{
				Allowed:       false,
				Reason:        "cooldown period not elapsed",
				NextAvailable: nextAvailable,
			}, nil
		}
	}
	// No previous token or error (first send) - continue

	// Check per-user daily limit
	userCountQuery := `
		SELECT COUNT(*)
		FROM email_verifications
		WHERE user_id = ? AND created_at > ?
	`

	var userCount int
	dayAgo := now.Add(-24 * time.Hour)
	err = d.db.QueryRowContext(ctx, userCountQuery, userID, dayAgo).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("checking user daily limit: %w", err)
	}

	if userCount >= dailyLimit {
		return &RateLimitResult{
			Allowed:       false,
			Reason:        "daily limit exceeded",
			CurrentCount:  userCount,
			Limit:         dailyLimit,
			NextAvailable: now.Add(24 * time.Hour),
		}, nil
	}

	// Check per-email daily limit (prevent spam to victim email addresses)
	emailCountQuery := `
		SELECT COUNT(*)
		FROM email_verifications
		WHERE email = ? AND created_at > ?
	`

	var emailCount int
	err = d.db.QueryRowContext(ctx, emailCountQuery, email, dayAgo).Scan(&emailCount)
	if err != nil {
		return nil, fmt.Errorf("checking email daily limit: %w", err)
	}

	// Email limit is 2x user limit (20 sends/day)
	emailLimit := dailyLimit * 2
	if emailCount >= emailLimit {
		return &RateLimitResult{
			Allowed:       false,
			Reason:        "email address limit exceeded",
			CurrentCount:  emailCount,
			Limit:         emailLimit,
			NextAvailable: now.Add(24 * time.Hour),
		}, nil
	}

	return &RateLimitResult{
		Allowed:      true,
		CurrentCount: userCount,
		Limit:        dailyLimit,
	}, nil
}

// CheckDeveloperEmailRateLimit checks rate limits for developer-initiated emails (notification API)
// Enforces global per-site limits:
// - maxPerHour sends per hour
// - maxPerDay sends per day
func (d *DB) CheckDeveloperEmailRateLimit(ctx context.Context, maxPerHour, maxPerDay int) (*RateLimitResult, error) {
	now := time.Now()

	// Check hourly limit
	hourAgo := now.Add(-1 * time.Hour)
	hourCountQuery := `
		SELECT COUNT(*)
		FROM email_logs
		WHERE email_type = 'notification' AND created_at > ?
	`

	var hourCount int
	err := d.db.QueryRowContext(ctx, hourCountQuery, hourAgo).Scan(&hourCount)
	if err != nil {
		return nil, fmt.Errorf("checking hourly rate limit: %w", err)
	}

	if hourCount >= maxPerHour {
		return &RateLimitResult{
			Allowed:       false,
			Reason:        "hourly rate limit exceeded",
			CurrentCount:  hourCount,
			Limit:         maxPerHour,
			NextAvailable: now.Add(1 * time.Hour),
		}, nil
	}

	// Check daily limit
	dayAgo := now.Add(-24 * time.Hour)
	dayCountQuery := `
		SELECT COUNT(*)
		FROM email_logs
		WHERE email_type = 'notification' AND created_at > ?
	`

	var dayCount int
	err = d.db.QueryRowContext(ctx, dayCountQuery, dayAgo).Scan(&dayCount)
	if err != nil {
		return nil, fmt.Errorf("checking daily rate limit: %w", err)
	}

	if dayCount >= maxPerDay {
		return &RateLimitResult{
			Allowed:       false,
			Reason:        "daily rate limit exceeded",
			CurrentCount:  dayCount,
			Limit:         maxPerDay,
			NextAvailable: now.Add(24 * time.Hour),
		}, nil
	}

	return &RateLimitResult{
		Allowed:      true,
		CurrentCount: dayCount,
		Limit:        maxPerDay,
	}, nil
}
