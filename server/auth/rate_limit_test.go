package auth

import (
	"context"
	"testing"
	"time"
)

func TestCheckVerificationRateLimit_Cooldown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Create a recently sent token
	token, _ := GenerateVerificationToken()
	hash, _ := HashToken(token)
	db.StoreVerificationToken(ctx, user.ID, user.Email, hash, time.Now().Add(1*time.Hour))

	// Check rate limit immediately (should fail due to cooldown)
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Rate limit should not allow send (cooldown period)")
	}
	if result.Reason != "cooldown period not elapsed" {
		t.Errorf("Reason = %q, want 'cooldown period not elapsed'", result.Reason)
	}
	if result.NextAvailable.IsZero() {
		t.Error("NextAvailable is zero")
	}
}

func TestCheckVerificationRateLimit_DailyLimitPerUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Create 10 tokens sent in the past (older than cooldown)
	for range 10 {
		token, _ := GenerateVerificationToken()
		hash, _ := HashToken(token)
		tokenID, _ := db.StoreVerificationToken(ctx, user.ID, user.Email, hash, time.Now().Add(1*time.Hour))
		// Backdate the last_sent_at to be older than cooldown
		db.GetDB().Exec(`UPDATE email_verifications SET last_sent_at = datetime('now', '-10 minutes') WHERE id = ?`, tokenID)
	}

	// Check rate limit (should fail due to daily limit)
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Rate limit should not allow send (daily limit reached)")
	}
	if result.Reason != "daily limit exceeded" {
		t.Errorf("Reason = %q, want 'daily limit exceeded'", result.Reason)
	}
	if result.CurrentCount != 10 {
		t.Errorf("CurrentCount = %d, want 10", result.CurrentCount)
	}
	if result.Limit != 10 {
		t.Errorf("Limit = %d, want 10", result.Limit)
	}
}

func TestCheckVerificationRateLimit_DailyLimitPerEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	email := "victim@example.com"

	// Create multiple users sending to same email (spam attack simulation)
	for i := range 20 {
		user, _ := db.CreateUser("User "+string(rune(i)), email)
		token, _ := GenerateVerificationToken()
		hash, _ := HashToken(token)
		tokenID, _ := db.StoreVerificationToken(ctx, user.ID, email, hash, time.Now().Add(1*time.Hour))
		// Backdate to avoid cooldown
		db.GetDB().Exec(`UPDATE email_verifications SET last_sent_at = datetime('now', '-10 minutes') WHERE id = ?`, tokenID)
	}

	// Try to send another email to same address (should fail)
	user, _ := db.CreateUser("Another User", email)
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Rate limit should not allow send (per-email daily limit)")
	}
	if result.Reason != "email address limit exceeded" {
		t.Errorf("Reason = %q, want 'email address limit exceeded'", result.Reason)
	}
}

func TestCheckVerificationRateLimit_Allowed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user with no previous sends
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Check rate limit (should be allowed)
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Rate limit should allow send: %s", result.Reason)
	}
	if result.Reason != "" {
		t.Errorf("Reason should be empty, got %q", result.Reason)
	}
}

func TestCheckVerificationRateLimit_CooldownExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Create token sent 6 minutes ago (cooldown is 5 minutes)
	token, _ := GenerateVerificationToken()
	hash, _ := HashToken(token)
	tokenID, _ := db.StoreVerificationToken(ctx, user.ID, user.Email, hash, time.Now().Add(1*time.Hour))
	db.GetDB().Exec(`UPDATE email_verifications SET last_sent_at = datetime('now', '-6 minutes') WHERE id = ?`, tokenID)

	// Check rate limit (should be allowed - cooldown expired)
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Rate limit should allow send after cooldown expired: %s", result.Reason)
	}
}

func TestCheckDeveloperEmailRateLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Log 50 notification emails in the past hour (at the hourly limit)
	for range 50 {
		log := &EmailLog{
			ID:        generateID("log"),
			UserID:    &user.ID,
			Recipient: "recipient@example.com",
			EmailType: "notification",
			Provider:  "mailgun",
			Status:    "sent",
			CreatedAt: time.Now().Add(-30 * time.Minute),
		}
		db.LogEmail(ctx, log)
	}

	// Check rate limit (should fail)
	result, err := db.CheckDeveloperEmailRateLimit(ctx, 50, 200)
	if err != nil {
		t.Fatalf("CheckDeveloperEmailRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Developer email rate limit should not allow send (hourly limit)")
	}
	if result.Reason != "hourly rate limit exceeded" {
		t.Errorf("Reason = %q, want 'hourly rate limit exceeded'", result.Reason)
	}
}

func TestCheckDeveloperEmailRateLimit_DailyLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Log 200 notification emails in the past 24 hours (at the daily limit)
	for range 200 {
		log := &EmailLog{
			ID:        generateID("log"),
			UserID:    &user.ID,
			Recipient: "recipient@example.com",
			EmailType: "notification",
			Provider:  "mailgun",
			Status:    "sent",
			CreatedAt: time.Now().Add(-12 * time.Hour),
		}
		db.LogEmail(ctx, log)
	}

	// Check rate limit (should fail)
	result, err := db.CheckDeveloperEmailRateLimit(ctx, 1000, 200)
	if err != nil {
		t.Fatalf("CheckDeveloperEmailRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Developer email rate limit should not allow send (daily limit)")
	}
	if result.Reason != "daily rate limit exceeded" {
		t.Errorf("Reason = %q, want 'daily rate limit exceeded'", result.Reason)
	}
}

func TestCheckDeveloperEmailRateLimit_Allowed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// No previous emails logged - should be allowed
	result, err := db.CheckDeveloperEmailRateLimit(ctx, 50, 200)
	if err != nil {
		t.Fatalf("CheckDeveloperEmailRateLimit failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Developer email rate limit should allow send: %s", result.Reason)
	}
}
