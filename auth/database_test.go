package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenDB(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	// Check database file was created
	dbPath := filepath.Join(tmpDir, ".basil-auth.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Check path is correct
	if db.Path() != dbPath {
		t.Errorf("Path() = %q, want %q", db.Path(), dbPath)
	}
}

func TestOpenDB_Permissions(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	db.Close()

	// Check file permissions (should be 0600)
	info, err := os.Stat(db.Path())
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	// On Unix, check permissions
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		t.Errorf("database has insecure permissions: %o", perm)
	}
}

func TestUserCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create user
	user, err := db.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == "" {
		t.Error("user ID is empty")
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want %q", user.Name, "Alice")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	// Get user by ID
	got, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetUser returned nil")
	}
	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}

	// Get user by email
	got, err = db.GetUserByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetUserByEmail returned nil")
	}
	if got.ID != user.ID {
		t.Errorf("ID = %q, want %q", got.ID, user.ID)
	}

	// Get non-existent user
	got, err = db.GetUser("nonexistent")
	if err != nil {
		t.Fatalf("GetUser for nonexistent failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent user")
	}

	// List users
	users, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("ListUsers returned %d users, want 1", len(users))
	}

	// User count
	count, err := db.UserCount()
	if err != nil {
		t.Fatalf("UserCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("UserCount = %d, want 1", count)
	}

	// Delete user
	err = db.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify deleted
	got, err = db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser after delete failed: %v", err)
	}
	if got != nil {
		t.Error("user still exists after delete")
	}

	// Delete non-existent
	err = db.DeleteUser("nonexistent")
	if err == nil {
		t.Error("expected error deleting nonexistent user")
	}
}

func TestUser_OptionalEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create user without email
	user, err := db.CreateUser("Bob", "")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Email != "" {
		t.Errorf("Email = %q, want empty", user.Email)
	}

	// Retrieve and verify
	got, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if got.Email != "" {
		t.Errorf("retrieved Email = %q, want empty", got.Email)
	}
}

func TestCredentialCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create user first
	user, err := db.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create credential
	cred := &Credential{
		ID:              []byte("credential-id-123"),
		UserID:          user.ID,
		PublicKey:       []byte("public-key-data"),
		SignCount:       0,
		Transports:      []string{"internal", "usb"},
		AttestationType: "none",
		CreatedAt:       time.Now().UTC(),
	}

	err = db.SaveCredential(cred)
	if err != nil {
		t.Fatalf("SaveCredential failed: %v", err)
	}

	// Get credentials by user
	creds, err := db.GetCredentialsByUser(user.ID)
	if err != nil {
		t.Fatalf("GetCredentialsByUser failed: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("got %d credentials, want 1", len(creds))
	}

	got := creds[0]
	if string(got.ID) != string(cred.ID) {
		t.Errorf("ID mismatch")
	}
	if string(got.PublicKey) != string(cred.PublicKey) {
		t.Errorf("PublicKey mismatch")
	}
	if len(got.Transports) != 2 || got.Transports[0] != "internal" {
		t.Errorf("Transports = %v, want [internal usb]", got.Transports)
	}

	// Get credential by ID
	got, err = db.GetCredential(cred.ID)
	if err != nil {
		t.Fatalf("GetCredential failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetCredential returned nil")
	}

	// Update sign count
	err = db.UpdateCredentialSignCount(cred.ID, 5)
	if err != nil {
		t.Fatalf("UpdateCredentialSignCount failed: %v", err)
	}

	got, _ = db.GetCredential(cred.ID)
	if got.SignCount != 5 {
		t.Errorf("SignCount = %d, want 5", got.SignCount)
	}

	// Credentials deleted with user (cascade)
	err = db.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	creds, err = db.GetCredentialsByUser(user.ID)
	if err != nil {
		t.Fatalf("GetCredentialsByUser after delete failed: %v", err)
	}
	if len(creds) != 0 {
		t.Error("credentials not deleted with user")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID("usr")
	id2 := generateID("usr")

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}

	if len(id1) < 10 {
		t.Errorf("ID too short: %q", id1)
	}

	if id1[:4] != "usr_" {
		t.Errorf("ID should start with prefix: %q", id1)
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		input string
		sep   string
		want  []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"internal,usb", ",", []string{"internal", "usb"}},
		{"single", ",", []string{"single"}},
		{"", ",", nil},
	}

	for _, tt := range tests {
		got := splitString(tt.input, tt.sep)
		if len(got) != len(tt.want) {
			t.Errorf("splitString(%q, %q) = %v, want %v", tt.input, tt.sep, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitString(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, got[i], tt.want[i])
			}
		}
	}
}

// setupTestDB creates a temporary test database.
func setupTestDB(t *testing.T) *DB {
	t.Helper()
	tmpDir := t.TempDir()

	db, err := OpenDB(tmpDir)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}

	return db
}
