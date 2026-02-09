package auth

import (
	"strings"
	"testing"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")

	codes, err := db.GenerateRecoveryCodes(user.ID, 8)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}

	if len(codes) != 8 {
		t.Errorf("got %d codes, want 8", len(codes))
	}

	// Check format: XXXX-XXXX-XXXX
	for i, code := range codes {
		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			t.Errorf("code %d format wrong: %q", i, code)
			continue
		}
		for _, part := range parts {
			if len(part) != 4 {
				t.Errorf("code %d segment wrong length: %q", i, code)
			}
		}
	}

	// All codes should be unique
	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestValidateRecoveryCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "alice@example.com")
	codes, _ := db.GenerateRecoveryCodes(user.ID, 3)

	// Valid code works
	valid, err := db.ValidateRecoveryCode(user.ID, codes[0])
	if err != nil {
		t.Fatalf("ValidateRecoveryCode failed: %v", err)
	}
	if !valid {
		t.Error("expected valid code to work")
	}

	// Same code fails second time (burned)
	valid, err = db.ValidateRecoveryCode(user.ID, codes[0])
	if err != nil {
		t.Fatalf("ValidateRecoveryCode failed: %v", err)
	}
	if valid {
		t.Error("expected burned code to fail")
	}

	// Other codes still work
	valid, _ = db.ValidateRecoveryCode(user.ID, codes[1])
	if !valid {
		t.Error("expected other code to work")
	}

	// Invalid code fails
	valid, _ = db.ValidateRecoveryCode(user.ID, "XXXX-XXXX-XXXX")
	if valid {
		t.Error("expected invalid code to fail")
	}

	// Wrong user fails
	user2, _ := db.CreateUser("Bob", "")
	valid, _ = db.ValidateRecoveryCode(user2.ID, codes[2])
	if valid {
		t.Error("expected wrong user's code to fail")
	}
}

func TestValidateRecoveryCode_CaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "")
	codes, _ := db.GenerateRecoveryCodes(user.ID, 1)

	// Lowercase should work
	lower := strings.ToLower(codes[0])
	valid, _ := db.ValidateRecoveryCode(user.ID, lower)
	if !valid {
		t.Error("expected lowercase code to work")
	}
}

func TestValidateRecoveryCode_WithoutDashes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "")
	codes, _ := db.GenerateRecoveryCodes(user.ID, 1)

	// Without dashes should work
	noDashes := strings.ReplaceAll(codes[0], "-", "")
	valid, _ := db.ValidateRecoveryCode(user.ID, noDashes)
	if !valid {
		t.Error("expected code without dashes to work")
	}
}

func TestGetRecoveryCodeCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "")
	codes, _ := db.GenerateRecoveryCodes(user.ID, 5)

	count, err := db.GetRecoveryCodeCount(user.ID)
	if err != nil {
		t.Fatalf("GetRecoveryCodeCount failed: %v", err)
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}

	// Use one code
	db.ValidateRecoveryCode(user.ID, codes[0])

	count, _ = db.GetRecoveryCodeCount(user.ID)
	if count != 4 {
		t.Errorf("count after use = %d, want 4", count)
	}
}

func TestGenerateRecoveryCodes_ReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := db.CreateUser("Alice", "")

	// Generate first set
	codes1, _ := db.GenerateRecoveryCodes(user.ID, 3)

	// Generate new set
	codes2, _ := db.GenerateRecoveryCodes(user.ID, 3)

	// Old codes should not work
	valid, _ := db.ValidateRecoveryCode(user.ID, codes1[0])
	if valid {
		t.Error("old code should not work after regeneration")
	}

	// New codes should work
	valid, _ = db.ValidateRecoveryCode(user.ID, codes2[0])
	if !valid {
		t.Error("new code should work")
	}
}

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ABCD-EFGH-IJKL", "ABCDEFGHIJKL"},
		{"abcd-efgh-ijkl", "ABCDEFGHIJKL"},
		{"AbCd-EfGh-IjKl", "ABCDEFGHIJKL"},
		{"ABCDEFGHIJKL", "ABCDEFGHIJKL"},
		{"abcdefghijkl", "ABCDEFGHIJKL"},
	}

	for _, tt := range tests {
		got := normalizeCode(tt.input)
		if got != tt.want {
			t.Errorf("normalizeCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateRecoveryCode_Format(t *testing.T) {
	// Generate many codes and check they're all valid format
	for range 100 {
		code := generateRecoveryCode()

		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			t.Fatalf("code has wrong number of segments: %q", code)
		}

		for _, part := range parts {
			if len(part) != 4 {
				t.Fatalf("segment has wrong length: %q", part)
			}
			for _, c := range part {
				if !strings.ContainsRune(string(recoveryCodeChars), c) {
					t.Fatalf("invalid character in code: %c in %q", c, code)
				}
			}
		}
	}
}
