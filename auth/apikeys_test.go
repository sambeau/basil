package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	plaintext, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	// Check plaintext format
	if !strings.HasPrefix(plaintext, "bsl_live_") {
		t.Errorf("plaintext should start with bsl_live_, got %q", plaintext[:10])
	}

	// Check prefix format: first 12 chars + "..." + last 4 chars
	if !strings.Contains(prefix, "...") {
		t.Errorf("prefix should contain ..., got %q", prefix)
	}
	// Prefix should be 12 + 3 + 4 = 19 chars
	if len(prefix) != 19 {
		t.Errorf("prefix length = %d, want 19", len(prefix))
	}
	// First part should match plaintext
	if !strings.HasPrefix(plaintext, prefix[:12]) {
		t.Errorf("prefix first part doesn't match plaintext")
	}
	// Last part should match plaintext
	if !strings.HasSuffix(plaintext, prefix[15:]) {
		t.Errorf("prefix last part doesn't match plaintext")
	}

	// Check hash is not empty
	if len(hash) == 0 {
		t.Error("hash is empty")
	}
}

func TestAPIKeyCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a user first
	user, err := db.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create API key
	key, plaintext, err := db.CreateAPIKey(user.ID, "test-key")
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Verify key ID format
	if !strings.HasPrefix(key.ID, "key_") {
		t.Errorf("key ID should start with key_, got %q", key.ID)
	}

	// Verify key properties
	if key.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", key.UserID, user.ID)
	}
	if key.Name != "test-key" {
		t.Errorf("Name = %q, want %q", key.Name, "test-key")
	}
	if key.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// Verify plaintext format
	if !strings.HasPrefix(plaintext, "bsl_live_") {
		t.Errorf("plaintext should start with bsl_live_, got %q", plaintext[:10])
	}

	// Get keys for user
	keys, err := db.GetAPIKeys(user.ID)
	if err != nil {
		t.Fatalf("GetAPIKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("GetAPIKeys returned %d keys, want 1", len(keys))
	}

	// Get single key
	gotKey, err := db.GetAPIKey(key.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}
	if gotKey.ID != key.ID {
		t.Errorf("GetAPIKey ID = %q, want %q", gotKey.ID, key.ID)
	}

	// Delete key
	err = db.DeleteAPIKey(key.ID)
	if err != nil {
		t.Fatalf("DeleteAPIKey failed: %v", err)
	}

	// Verify deleted
	keys, err = db.GetAPIKeys(user.ID)
	if err != nil {
		t.Fatalf("GetAPIKeys after delete failed: %v", err)
	}
	if len(keys) != 0 {
		t.Error("key still exists after delete")
	}
}

func TestValidateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a user
	user, err := db.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create API key
	_, plaintext, err := db.CreateAPIKey(user.ID, "test-key")
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Validate correct key
	gotUser, err := db.ValidateAPIKey(plaintext)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}
	if gotUser == nil {
		t.Fatal("ValidateAPIKey returned nil user")
	}
	if gotUser.ID != user.ID {
		t.Errorf("user ID = %q, want %q", gotUser.ID, user.ID)
	}

	// Validate incorrect key
	gotUser, err = db.ValidateAPIKey("bsl_live_invalidkey")
	if err != nil {
		t.Fatalf("ValidateAPIKey with invalid key failed: %v", err)
	}
	if gotUser != nil {
		t.Error("expected nil user for invalid key")
	}

	// Validate empty key
	gotUser, err = db.ValidateAPIKey("")
	if err != nil {
		t.Fatalf("ValidateAPIKey with empty key failed: %v", err)
	}
	if gotUser != nil {
		t.Error("expected nil user for empty key")
	}
}

func TestGetAllAPIKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	alice, _ := db.CreateUser("Alice", "alice@example.com")
	bob, _ := db.CreateUser("Bob", "bob@example.com")

	// Create keys for each
	db.CreateAPIKey(alice.ID, "alice-key")
	db.CreateAPIKey(bob.ID, "bob-key")

	// Get all keys
	keys, err := db.GetAllAPIKeys()
	if err != nil {
		t.Fatalf("GetAllAPIKeys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("GetAllAPIKeys returned %d keys, want 2", len(keys))
	}
}

func TestUpdateAPIKeyLastUsed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")
	key, _, _ := db.CreateAPIKey(user.ID, "test-key")

	// Initial LastUsedAt should be nil (zero time)
	if key.LastUsedAt != nil {
		t.Error("expected nil LastUsedAt initially")
	}

	// Update last used
	err := db.UpdateAPIKeyLastUsed(key.ID)
	if err != nil {
		t.Fatalf("UpdateAPIKeyLastUsed failed: %v", err)
	}

	// Verify update
	updatedKey, err := db.GetAPIKey(key.ID)
	if err != nil {
		t.Fatalf("GetAPIKey failed: %v", err)
	}
	if updatedKey.LastUsedAt == nil {
		t.Error("LastUsedAt is still nil after update")
	}
}
