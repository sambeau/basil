package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/basil/server/auth/email"
	"github.com/sambeau/basil/server/config"
)

// MockEmailProvider implements email.Provider for testing
type MockEmailProvider struct {
	SentEmails []SentEmail
	ShouldFail bool
}

type SentEmail struct {
	To      string
	Subject string
	Body    string
}

func (m *MockEmailProvider) Send(ctx context.Context, msg *email.Message) (string, error) {
	if m.ShouldFail {
		return "", email.ErrSendFailed
	}

	body := msg.HTML
	if body == "" {
		body = msg.Text
	}

	m.SentEmails = append(m.SentEmails, SentEmail{
		To:      msg.To[0],
		Subject: msg.Subject,
		Body:    body,
	})
	return "mock-message-id", nil
}

func (m *MockEmailProvider) Name() string {
	return "mock"
}

func (m *MockEmailProvider) GetLastEmail() *SentEmail {
	if len(m.SentEmails) == 0 {
		return nil
	}
	return &m.SentEmails[len(m.SentEmails)-1]
}

func (m *MockEmailProvider) Reset() {
	m.SentEmails = nil
	m.ShouldFail = false
}

// TestVerificationFlow_Complete tests the full verification flow from registration to verification
func TestVerificationFlow_Complete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	mockProvider := &MockEmailProvider{}

	// Create email service with mock provider
	service := &EmailService{
		provider: mockProvider,
		db:       db,
		baseURL:  "https://example.com",
		config: &config.EmailVerificationConfig{
			Enabled:        true,
			TokenTTL:       1 * time.Hour,
			ResendCooldown: 5 * time.Minute,
			MaxSendsPerDay: 10,
		},
	}

	// Step 1: Create user
	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.EmailVerifiedAt != nil {
		t.Error("New user should not have verified email")
	}

	// Step 2: Send verification email
	err = service.SendVerificationEmail(ctx, user)
	if err != nil {
		t.Fatalf("SendVerificationEmail failed: %v", err)
	}

	// Check email was sent
	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(mockProvider.SentEmails))
	}

	email := mockProvider.GetLastEmail()
	if email.To != user.Email {
		t.Errorf("Email sent to %q, want %q", email.To, user.Email)
	}
	if !strings.Contains(email.Subject, "Verify") && !strings.Contains(email.Subject, "verify") {
		t.Errorf("Email subject %q doesn't contain 'Verify'", email.Subject)
	}

	// Extract token from email body
	// Format: /__auth/verify-email?token=<token>
	tokenStart := strings.Index(email.Body, "?token=")
	if tokenStart == -1 {
		t.Fatal("Email doesn't contain verification token")
	}
	tokenStart += len("?token=")
	// Token is 64 hex chars - extract only those
	tokenEnd := tokenStart + 64
	if tokenEnd > len(email.Body) {
		t.Fatalf("Email body too short for token, body length = %d", len(email.Body))
	}
	token := email.Body[tokenStart:tokenEnd]

	// Validate token format (64 hex chars)
	if len(token) != 64 {
		t.Errorf("Token length = %d, want 64, token = %q", len(token), token)
	}
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Token contains non-hex character: %c", c)
			break
		}
	}

	// Step 3: Verify token
	verification, err := db.LookupVerificationToken(ctx, token)
	if err != nil {
		t.Fatalf("LookupVerificationToken failed: %v", err)
	}

	if verification.UserID != user.ID {
		t.Errorf("Verification user_id = %q, want %q", verification.UserID, user.ID)
	}
	if verification.Email != user.Email {
		t.Errorf("Verification email = %q, want %q", verification.Email, user.Email)
	}
	if verification.ConsumedAt != nil {
		t.Error("Token should not be consumed yet")
	}

	// Step 4: Mark email as verified
	err = db.MarkEmailVerified(ctx, user.ID)
	if err != nil {
		t.Fatalf("MarkEmailVerified failed: %v", err)
	}

	// Step 5: Consume token
	err = db.ConsumeVerificationToken(ctx, verification.ID)
	if err != nil {
		t.Fatalf("ConsumeVerificationToken failed: %v", err)
	}

	// Step 6: Verify user email is marked as verified
	updatedUser, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if updatedUser.EmailVerifiedAt == nil {
		t.Error("User email should be verified")
	}

	// Step 7: Try to use token again (should fail)
	verification2, err := db.LookupVerificationToken(ctx, token)
	if err == nil && verification2 != nil {
		t.Error("Consumed token should not be returned")
	}
}

// TestVerificationFlow_Resend tests resending verification with rate limiting
func TestVerificationFlow_Resend(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	mockProvider := &MockEmailProvider{}

	service := &EmailService{
		provider: mockProvider,
		db:       db,
		baseURL:  "https://example.com",
		config: &config.EmailVerificationConfig{
			Enabled:        true,
			TokenTTL:       1 * time.Hour,
			ResendCooldown: 5 * time.Minute,
			MaxSendsPerDay: 10,
		},
	}

	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// First send - should succeed
	err = service.SendVerificationEmail(ctx, user)
	if err != nil {
		t.Fatalf("First send failed: %v", err)
	}

	if len(mockProvider.SentEmails) != 1 {
		t.Fatalf("Expected 1 email after first send, got %d", len(mockProvider.SentEmails))
	}

	// Immediate resend - should fail due to cooldown
	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("Immediate resend should be blocked by cooldown")
	}
	if result.Reason != "cooldown period not elapsed" {
		t.Errorf("Reason = %q, want 'cooldown period not elapsed'", result.Reason)
	}

	// Simulate cooldown expiry by backdating the token
	query := `UPDATE email_verifications SET last_sent_at = ? WHERE user_id = ?`
	_, err = db.db.ExecContext(ctx, query, time.Now().Add(-6*time.Minute), user.ID)
	if err != nil {
		t.Fatalf("Failed to backdate token: %v", err)
	}

	// Resend after cooldown - should succeed
	result, err = db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, 10)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit after cooldown failed: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Resend after cooldown should be allowed, got: %s", result.Reason)
	}

	err = service.SendVerificationEmail(ctx, user)
	if err != nil {
		t.Fatalf("Resend after cooldown failed: %v", err)
	}

	if len(mockProvider.SentEmails) != 2 {
		t.Fatalf("Expected 2 emails after resend, got %d", len(mockProvider.SentEmails))
	}
}

// TestRateLimiting_DailyLimit tests the daily rate limit enforcement
func TestRateLimiting_DailyLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	mockProvider := &MockEmailProvider{}

	service := &EmailService{
		provider: mockProvider,
		db:       db,
		baseURL:  "https://example.com",
		config: &config.EmailVerificationConfig{
			Enabled:        true,
			TokenTTL:       1 * time.Hour,
			ResendCooldown: 5 * time.Minute,
			MaxSendsPerDay: 10,
		},
	}

	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	dailyLimit := 10

	// Send 10 emails (at limit)
	for i := 0; i < dailyLimit; i++ {
		// Backdate tokens to bypass cooldown
		if i > 0 {
			query := `UPDATE email_verifications SET last_sent_at = ? WHERE user_id = ?`
			_, err = db.db.ExecContext(ctx, query, time.Now().Add(-6*time.Minute), user.ID)
			if err != nil {
				t.Fatalf("Failed to backdate token: %v", err)
			}
		}

		err = service.SendVerificationEmail(ctx, user)
		if err != nil {
			t.Fatalf("Send %d failed: %v", i+1, err)
		}
	}

	// 11th send should be blocked
	query := `UPDATE email_verifications SET last_sent_at = ? WHERE user_id = ?`
	_, err = db.db.ExecContext(ctx, query, time.Now().Add(-6*time.Minute), user.ID)
	if err != nil {
		t.Fatalf("Failed to backdate token: %v", err)
	}

	result, err := db.CheckVerificationRateLimit(ctx, user.ID, user.Email, 5*time.Minute, dailyLimit)
	if err != nil {
		t.Fatalf("CheckVerificationRateLimit failed: %v", err)
	}

	if result.Allowed {
		t.Error("11th send should be blocked by daily limit")
	}
	if result.Reason != "daily limit exceeded" {
		t.Errorf("Reason = %q, want 'daily limit exceeded'", result.Reason)
	}
}

// TestEmailProvider_Failure tests handling of provider failures
func TestEmailProvider_Failure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	mockProvider := &MockEmailProvider{
		ShouldFail: true,
	}

	service := &EmailService{
		provider: mockProvider,
		db:       db,
		baseURL:  "https://example.com",
		config: &config.EmailVerificationConfig{
			Enabled:        true,
			TokenTTL:       1 * time.Hour,
			ResendCooldown: 5 * time.Minute,
			MaxSendsPerDay: 10,
		},
	}

	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err = service.SendVerificationEmail(ctx, user)
	if err == nil {
		t.Error("SendVerificationEmail should fail when provider fails")
	}

	// Check error contains expected message
	if !errors.Is(err, email.ErrSendFailed) && !strings.Contains(err.Error(), "send") {
		t.Errorf("Expected send failure error, got: %v", err)
	}
}
