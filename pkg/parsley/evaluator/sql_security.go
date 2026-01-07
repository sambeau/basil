package evaluator

import (
	"fmt"
	"regexp"
	"strings"
)

// SQL identifier validation for security
//
// SECURITY CRITICAL:
// - Prevents SQL injection via user-controlled table/column names
// - Used before fmt.Sprintf string interpolation in SQL queries
// - Must be called for ANY user-provided identifier in SQL context
//
// AI MAINTENANCE GUIDE:
// - If adding new SQL query building code, ALWAYS validate identifiers
// - Never trust identifiers from dictionaries, arrays, or user input
// - Search codebase for "fmt.Sprintf.*FROM" to find validation sites
// - Test with SQL injection payloads: "; DROP TABLE", "' OR '1'='1"

var sqlIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// maxSQLIdentifierLength is the maximum length for a SQL identifier.
// SQLite supports up to 1000 bytes, but we use a conservative 64 for safety.
const maxSQLIdentifierLength = 64

// isValidSQLIdentifier checks if a name is a valid SQL identifier.
//
// Valid identifiers:
// - Start with letter or underscore
// - Contain only letters, numbers, underscores
// - Length between 1 and 64 characters
// - No SQL keywords (checked separately)
//
// Invalid identifiers (SQL injection attempts):
// - "user; DROP TABLE users--"
// - "table' OR '1'='1"
// - "col`name"
// - "../../../etc/passwd"
// - "table name" (contains space)
func isValidSQLIdentifier(name string) bool {
	if name == "" || len(name) > maxSQLIdentifierLength {
		return false
	}
	return sqlIdentifierRegex.MatchString(name)
}

// validateSQLIdentifier returns an error if the identifier is invalid.
// Use this before interpolating identifiers into SQL strings.
func validateSQLIdentifier(name string) error {
	if !isValidSQLIdentifier(name) {
		return fmt.Errorf("invalid SQL identifier: %q (must be alphanumeric/underscore, max %d chars)", 
			name, maxSQLIdentifierLength)
	}
	return nil
}

// validateSQLIdentifiers validates multiple identifiers at once.
// Returns an error describing all invalid identifiers found.
func validateSQLIdentifiers(names []string) error {
	var invalid []string
	for _, name := range names {
		if !isValidSQLIdentifier(name) {
			invalid = append(invalid, name)
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("invalid SQL identifiers: %s", strings.Join(invalid, ", "))
	}
	return nil
}

// quoteSQLIdentifier validates and quotes an identifier for use in SQL.
// Returns quoted identifier (e.g., "user_id" -> "user_id") or error.
//
// Note: SQLite uses double quotes for identifiers, which prevents
// the identifier from being interpreted as a string literal.
func quoteSQLIdentifier(name string) (string, error) {
	if err := validateSQLIdentifier(name); err != nil {
		return "", err
	}
	// SQLite uses double quotes for identifiers
	return fmt.Sprintf(`"%s"`, name), nil
}

// isSQLKeyword checks if a name is a reserved SQL keyword.
// These keywords cannot be used as unquoted identifiers.
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true,
	"UPDATE": true, "DELETE": true, "CREATE": true, "DROP": true,
	"TABLE": true, "INDEX": true, "VIEW": true, "TRIGGER": true,
	"ALTER": true, "ADD": true, "COLUMN": true, "PRIMARY": true,
	"KEY": true, "FOREIGN": true, "REFERENCES": true, "CONSTRAINT": true,
	"CHECK": true, "DEFAULT": true, "UNIQUE": true, "NOT": true,
	"NULL": true, "AND": true, "OR": true, "IN": true, "EXISTS": true,
	"BETWEEN": true, "LIKE": true, "GLOB": true, "REGEXP": true,
	"IS": true, "AS": true, "ON": true, "USING": true, "JOIN": true,
	"LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true, "CROSS": true,
	"GROUP": true, "BY": true, "HAVING": true, "ORDER": true, "ASC": true,
	"DESC": true, "LIMIT": true, "OFFSET": true, "UNION": true, "ALL": true,
	"DISTINCT": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "CAST": true, "COLLATE": true, "BEGIN": true, "COMMIT": true,
	"ROLLBACK": true, "TRANSACTION": true, "SAVEPOINT": true, "RELEASE": true,
}

func isSQLKeyword(name string) bool {
	return sqlKeywords[strings.ToUpper(name)]
}
