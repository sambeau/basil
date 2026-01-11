package auth

import (
	"context"
	"testing"
	"time"
)

func TestGenerateVerificationToken(t *testing.T) {
	// Generate multiple tokens to verify uniqueness
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateVerificationToken()
		if err != nil {
			t.Fatalf("GenerateVerificationToken failed: %v", err)
		}

		// Check length (32 bytes = 64 hex chars)
		if len(token) != 64 {
			t.Errorf("Token length = %d, want 64", len(token))
		}

		// Check uniqueness
		if tokens[token] {
			t.Errorf("Duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestHashToken(t *testing.T) {
	token := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	hash1, err := HashToken(token)
	if err != nil {
		t.Fatalf("HashToken failed: %v", err)
	}

	// Hash should not be empty
	if hash1 == "" {
		t.Error("Hash is empty")
	}

	// Hash should be different from token
	if hash1 == token {
		t.Error("Hash equals token (should be bcrypt hashed)")
	}

	// Hashing same token again should produce different hash (bcrypt includes salt)
	hash2, err := HashToken(token)
	if err != nil {
		t.Fatalf("Second HashToken failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Bcrypt hashes are identical (should have different salts)")
	}
}

func TestVerificationTokenFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Generate and store token
	token, err := GenerateVerificationToken()
	if err != nil {
		t.Fatalf("GenerateVerificationToken failed: %v", err)
	}

	tokenHash, err := HashToken(token)
	if err != nil {
		t.Fatalf("HashToken failed: %v", err)
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	tokenID, err := db.StoreVerificationToken(ctx, user.ID, user.Email, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("StoreVerificationToken failed: %v", err)
	}

	if tokenID == "" {
		t.Error("tokenID is empty")
	}

	// Lookup token (should succeed)
	verification, err := db.LookupVerificationToken(ctx, token)
	if err != nil {
		t.Fatalf("LookupVerificationToken failed: %v", err)
	}
	if verification == nil {
		t.Fatal("verification is nil")
	}
	if verification.UserID != user.ID {
		t.Errorf("UserID = %s, want %s", verification.UserID, user.ID)
	}
	if verification.Email != user.Email {
		t.Errorf("Email = %s, want %s", verification.Email, user.Email)
	}

	// Mark email verified
	if err := db.MarkEmailVerified(ctx, user.ID); err != nil {
		t.Fatalf("MarkEmailVerified failed: %v", err)
	}

	// Verify user is marked as verified
	updatedUser, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if updatedUser.EmailVerifiedAt == nil {
		t.Error("EmailVerifiedAt is nil after verification")
	}

	// Consume token
	if err := db.ConsumeVerificationToken(ctx, verification.ID); err != nil {
		t.Fatalf("ConsumeVerificationToken failed: %v", err)
	}

	// Lookup consumed token (should return nil with no error, or error)
	verification2, err := db.LookupVerificationToken(ctx, token)
	// Either no error with nil result, or an error is acceptable for consumed tokens
	if err == nil && verification2 != nil {
		t.Error("Consumed token was returned (should be nil)")
	}
}

func TestLookupVerificationToken_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Lookup non-existent token (should return error)
	verification, err := db.LookupVerificationToken(ctx, "nonexistenttoken1234567890")
	if err == nil {
		t.Fatal("Expected error for non-existent token")
	}
	if verification != nil {
		t.Error("Non-existent token returned a result")
	}
}

func TestLookupVerificationToken_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, err := db.CreateUser("Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Generate and store expired token
	token, err := GenerateVerificationToken()
	if err != nil {
		t.Fatalf("GenerateVerificationToken failed: %v", err)
	}

	tokenHash, err := HashToken(token)
	if err != nil {
		t.Fatalf("HashToken failed: %v", err)
	}

	// Set expiry in the past
	expiresAt := time.Now().Add(-1 * time.Hour)
	_, err = db.StoreVerificationToken(ctx, user.ID, user.Email, tokenHash, expiresAt)
	if err != nil {
		t.Fatalf("StoreVerificationToken failed: %v", err)
	}

	// Lookup expired token (should return error)
	verification, err := db.LookupVerificationToken(ctx, token)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
	if verification != nil {
		t.Error("Expired token was returned (should be nil)")
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test users
	user1, _ := db.CreateUser("User 1", "user1@example.com")
	user2, _ := db.CreateUser("User 2", "user2@example.com")

	// Create expired token for user1
	token1, _ := GenerateVerificationToken()
	hash1, _ := HashToken(token1)
	db.StoreVerificationToken(ctx, user1.ID, user1.Email, hash1, time.Now().Add(-2*time.Hour))

	// Create valid token for user2
	token2, _ := GenerateVerificationToken()
	hash2, _ := HashToken(token2)
	db.StoreVerificationToken(ctx, user2.ID, user2.Email, hash2, time.Now().Add(1*time.Hour))

	// Cleanup expired tokens
	deleted, err := db.CleanupExpiredTokens(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredTokens failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Deleted %d tokens, want 1", deleted)
	}

	// Verify expired token is gone
	verification1, _ := db.LookupVerificationToken(ctx, token1)
	if verification1 != nil {
		t.Error("Expired token still exists after cleanup")
	}

	// Verify valid token still exists
	verification2, _ := db.LookupVerificationToken(ctx, token2)
	if verification2 == nil {
		t.Error("Valid token was deleted during cleanup")
	}
}

func TestInvalidateUserVerificationTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test user
	user, _ := db.CreateUser("Test User", "test@example.com")

	// Create multiple tokens for same user
	token1, _ := GenerateVerificationToken()
	hash1, _ := HashToken(token1)
	db.StoreVerificationToken(ctx, user.ID, user.Email, hash1, time.Now().Add(1*time.Hour))

	token2, _ := GenerateVerificationToken()
	hash2, _ := HashToken(token2)
	db.StoreVerificationToken(ctx, user.ID, user.Email, hash2, time.Now().Add(1*time.Hour))

	// Verify both tokens exist
	v1, _ := db.LookupVerificationToken(ctx, token1)
	v2, _ := db.LookupVerificationToken(ctx, token2)
	if v1 == nil || v2 == nil {
		t.Fatal("Tokens not created properly")
	}

	// Invalidate all tokens for user
	if err := db.InvalidateUserVerificationTokens(ctx, user.ID); err != nil {
		t.Fatalf("InvalidateUserVerificationTokens failed: %v", err)
	}

	// Verify both tokens are gone
	v1, _ = db.LookupVerificationToken(ctx, token1)
	v2, _ = db.LookupVerificationToken(ctx, token2)
	if v1 != nil || v2 != nil {
		t.Error("Tokens still exist after invalidation")
	}
}
