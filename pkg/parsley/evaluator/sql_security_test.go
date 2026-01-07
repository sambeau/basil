package evaluator

import (
	"strings"
	"testing"
)

// TestIsValidSQLIdentifier tests the SQL identifier validation function
func TestIsValidSQLIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid identifiers
		{"simple name", "users", true},
		{"with underscore", "user_id", true},
		{"starts with underscore", "_private", true},
		{"with numbers", "table123", true},
		{"mixed case", "TableName", true},
		{"single char", "a", true},
		{"max length", strings.Repeat("a", 64), true},

		// Invalid identifiers (SQL injection attempts)
		{"sql injection semicolon", "users; DROP TABLE users--", false},
		{"sql injection quote", "users' OR '1'='1", false},
		{"sql injection comment", "users--", false},
		{"sql injection union", "users UNION SELECT", false},
		{"path traversal", "../../../etc/passwd", false},
		{"space", "user name", false},
		{"starts with number", "123table", false},
		{"special chars dash", "user-id", false},
		{"special chars dot", "user.id", false},
		{"special chars dollar", "$userid", false},
		{"special chars at", "@userid", false},
		{"backtick", "user`id", false},
		{"double quote", `user"id`, false},
		{"single quote", "user'id", false},
		{"parentheses", "user(id)", false},
		{"brackets", "user[id]", false},
		{"asterisk", "user*", false},
		{"percent", "user%", false},
		{"semicolon", "user;", false},
		{"pipe", "user|id", false},
		{"ampersand", "user&id", false},
		{"equals", "user=id", false},
		{"less than", "user<id", false},
		{"greater than", "user>id", false},
		{"exclamation", "user!", false},
		{"question mark", "user?", false},
		{"newline", "user\nid", false},
		{"tab", "user\tid", false},
		{"null byte", "user\x00id", false},
		{"empty string", "", false},
		{"too long", strings.Repeat("a", 65), false},
		{"unicode", "user_名前", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSQLIdentifier(tt.input)
			if result != tt.valid {
				t.Errorf("isValidSQLIdentifier(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}

// TestValidateSQLIdentifier tests the error-returning validation function
func TestValidateSQLIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid", "users", false},
		{"injection attempt", "users; DROP TABLE", true},
		{"path traversal", "../etc/passwd", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLIdentifier(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("validateSQLIdentifier(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

// TestValidateSQLIdentifiers tests batch validation
func TestValidateSQLIdentifiers(t *testing.T) {
	tests := []struct {
		name      string
		inputs    []string
		wantError bool
	}{
		{"all valid", []string{"users", "posts", "user_id"}, false},
		{"one invalid", []string{"users", "posts; DROP TABLE", "user_id"}, true},
		{"multiple invalid", []string{"users; DROP", "posts' OR", "valid_id"}, true},
		{"empty list", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSQLIdentifiers(tt.inputs)
			if (err != nil) != tt.wantError {
				t.Errorf("validateSQLIdentifiers(%v) error = %v, wantError %v", tt.inputs, err, tt.wantError)
			}
		})
	}
}

// TestQuoteSQLIdentifier tests identifier quoting
func TestQuoteSQLIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{"simple", "users", `"users"`, false},
		{"with underscore", "user_id", `"user_id"`, false},
		{"injection attempt", "users; DROP", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := quoteSQLIdentifier(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("quoteSQLIdentifier(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
			if result != tt.want {
				t.Errorf("quoteSQLIdentifier(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

// TestIsSQLKeyword tests SQL keyword detection
func TestIsSQLKeyword(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		isKeyword  bool
	}{
		{"SELECT uppercase", "SELECT", true},
		{"select lowercase", "select", true},
		{"SeLeCt mixed case", "SeLeCt", true},
		{"FROM", "FROM", true},
		{"WHERE", "WHERE", true},
		{"DROP", "DROP", true},
		{"TABLE", "TABLE", true},
		{"not a keyword", "users", false},
		{"similar to keyword", "SELECTING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSQLKeyword(tt.input)
			if result != tt.isKeyword {
				t.Errorf("isSQLKeyword(%q) = %v, want %v", tt.input, result, tt.isKeyword)
			}
		})
	}
}

// TestSQLInjectionVectors tests common SQL injection attack patterns
func TestSQLInjectionVectors(t *testing.T) {
	// Real-world SQL injection patterns that should be blocked
	injectionVectors := []string{
		// Classic SQL injection
		"'; DROP TABLE users--",
		"' OR '1'='1",
		"' OR 1=1--",
		"admin'--",
		"admin' #",
		"admin'/*",
		
		// UNION-based injection
		"' UNION SELECT NULL--",
		"' UNION ALL SELECT",
		"1' UNION SELECT * FROM users--",
		
		// Boolean-based blind injection
		"1' AND '1'='1",
		"1' AND '1'='2",
		
		// Time-based blind injection
		"1' AND SLEEP(5)--",
		"1'; WAITFOR DELAY '00:00:05'--",
		
		// Stacked queries
		"1'; DROP TABLE users;--",
		"1'; DELETE FROM users WHERE '1'='1",
		
		// Comment injection
		"user/**/name",
		"user--name",
		"user#name",
		
		// Encoded injection
		"user%27%20OR%20",
		"user%00",
		
		// NoSQL-style injection attempts
		"user[$ne]",
		"user{$ne:null}",
	}

	for _, vector := range injectionVectors {
		t.Run(vector, func(t *testing.T) {
			if isValidSQLIdentifier(vector) {
				t.Errorf("SECURITY FAILURE: SQL injection vector passed validation: %q", vector)
			}
		})
	}
}

// TestSQLIdentifierEdgeCases tests edge cases and boundary conditions
func TestSQLIdentifierEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Boundary conditions
		{"exactly 64 chars", strings.Repeat("a", 64), true},
		{"65 chars", strings.Repeat("a", 65), false},
		{"single underscore", "_", true},
		{"double underscore", "__", true},
		{"triple underscore", "___private", true},
		
		// Case sensitivity
		{"all caps", "USERS", true},
		{"all lowercase", "users", true},
		{"camelCase", "userId", true},
		{"PascalCase", "UserId", true},
		{"snake_case", "user_id", true},
		
		// Numbers
		{"trailing number", "user1", true},
		{"middle number", "u1ser", true},
		{"leading number", "1user", false},
		{"all numbers", "12345", false},
		
		// Whitespace variations
		{"leading space", " users", false},
		{"trailing space", "users ", false},
		{"internal space", "user name", false},
		{"tab", "user\tname", false},
		{"newline", "user\nname", false},
		{"carriage return", "user\rname", false},
		
		// Control characters
		{"null byte", "user\x00", false},
		{"bell", "user\x07", false},
		{"backspace", "user\x08", false},
		
		// Empty and nil-like
		{"empty string", "", false},
		{"just spaces", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSQLIdentifier(tt.input)
			if result != tt.valid {
				t.Errorf("isValidSQLIdentifier(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}
