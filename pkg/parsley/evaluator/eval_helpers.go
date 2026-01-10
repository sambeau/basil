package evaluator

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// eval_helpers.go contains utility functions for the evaluator that don't directly
// perform evaluation. This includes natural sorting, type comparison, path security,
// currency formatting, and type checking helpers.

// naturalCompare compares two objects using natural sort order
// Returns true if a < b in natural sort order
func naturalCompare(a, b Object) bool {
	// Type-based ordering: numbers < strings
	aType := getTypeOrder(a)
	bType := getTypeOrder(b)

	if aType != bType {
		return aType < bType
	}

	// Both are numbers
	if aType == 0 {
		return compareNumbers(a, b)
	}

	// Both are strings - use natural string comparison
	if aType == 1 {
		aStr := a.(*String).Value
		bStr := b.(*String).Value
		return NaturalCompare(aStr, bStr) < 0
	}

	// Other types (shouldn't happen with current implementation)
	return false
}

// getTypeOrder returns a sort order for types
// 0 = numbers (Integer, Float)
// 1 = strings
// 2 = other
func getTypeOrder(obj Object) int {
	switch obj.Type() {
	case INTEGER_OBJ, FLOAT_OBJ:
		return 0
	case STRING_OBJ:
		return 1
	default:
		return 2
	}
}

// compareNumbers compares two numeric objects
func compareNumbers(a, b Object) bool {
	aVal := getNumericValue(a)
	bVal := getNumericValue(b)
	return aVal < bVal
}

// getNumericValue extracts numeric value as float64
func getNumericValue(obj Object) float64 {
	switch obj := obj.(type) {
	case *Integer:
		return float64(obj.Value)
	case *Float:
		return obj.Value
	default:
		return 0
	}
}

// NaturalCompare compares two strings using natural sort order.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Uses ASCII fast path for performance, falls back to Unicode for international text.
func NaturalCompare(a, b string) int {
	// Fast path: check if both strings are ASCII
	if isASCII(a) && isASCII(b) {
		return naturalCompareASCII(a, b)
	}
	// Fallback: Unicode-aware comparison
	return naturalCompareUnicode(a, b)
}

// isASCII checks if a string contains only ASCII characters
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}

// isDigitASCII checks if byte is ASCII digit (inlined for performance)
func isDigitASCII(c byte) bool {
	return '0' <= c && c <= '9'
}

// naturalCompareASCII compares two ASCII strings using natural sort order.
// This is the fast path - works on bytes directly with zero allocations.
func naturalCompareASCII(a, b string) int {
	ai, bi := 0, 0

	for ai < len(a) && bi < len(b) {
		ca, cb := a[ai], b[bi]

		// Both are digits - compare numerically
		if isDigitASCII(ca) && isDigitASCII(cb) {
			// Check for leading zeros (left-aligned comparison)
			if ca == '0' || cb == '0' {
				if result := compareLeftASCII(a, b, &ai, &bi); result != 0 {
					return result
				}
				continue
			}
			// Right-aligned comparison (no leading zeros)
			if result := compareRightASCII(a, b, &ai, &bi); result != 0 {
				return result
			}
			continue
		}

		// Regular character comparison
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}

		ai++
		bi++
	}

	// Shorter string comes first
	if ai < len(a) {
		return 1
	}
	if bi < len(b) {
		return -1
	}
	return 0
}

// compareRightASCII compares two right-aligned numbers (no leading zeros).
// The longest run of digits wins; if equal length, greater value wins.
func compareRightASCII(a, b string, ai, bi *int) int {
	bias := 0

	for {
		var ca, cb byte
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && isDigitASCII(ca)
		bIsDigit := bHasMore && isDigitASCII(cb)

		if !aIsDigit && !bIsDigit {
			return bias
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		if ca < cb {
			if bias == 0 {
				bias = -1
			}
		} else if ca > cb {
			if bias == 0 {
				bias = 1
			}
		}

		*ai++
		*bi++
	}
}

// compareLeftASCII compares two left-aligned numbers (with leading zeros).
// First different digit wins.
func compareLeftASCII(a, b string, ai, bi *int) int {
	for {
		var ca, cb byte
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && isDigitASCII(ca)
		bIsDigit := bHasMore && isDigitASCII(cb)

		if !aIsDigit && !bIsDigit {
			return 0
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		if ca < cb {
			return -1
		}
		if ca > cb {
			return 1
		}

		*ai++
		*bi++
	}
}

// naturalCompareUnicode compares two strings using natural sort order with full Unicode support.
// This handles non-ASCII digits (Arabic numerals, etc.) but is slower than ASCII path.
func naturalCompareUnicode(a, b string) int {
	aRunes := []rune(a)
	bRunes := []rune(b)

	ai, bi := 0, 0

	for ai < len(aRunes) && bi < len(bRunes) {
		ca, cb := aRunes[ai], bRunes[bi]

		// Both are digits - compare numerically
		if unicode.IsDigit(ca) && unicode.IsDigit(cb) {
			// Check for leading zeros
			if ca == '0' || cb == '0' {
				if result := compareLeftUnicode(aRunes, bRunes, &ai, &bi); result != 0 {
					return result
				}
				continue
			}
			if result := compareRightUnicode(aRunes, bRunes, &ai, &bi); result != 0 {
				return result
			}
			continue
		}

		// Regular character comparison
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}

		ai++
		bi++
	}

	// Shorter string comes first
	if ai < len(aRunes) {
		return 1
	}
	if bi < len(bRunes) {
		return -1
	}
	return 0
}

// compareRightUnicode compares two right-aligned numbers with Unicode digit support.
func compareRightUnicode(a, b []rune, ai, bi *int) int {
	bias := 0

	for {
		var ca, cb rune
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && unicode.IsDigit(ca)
		bIsDigit := bHasMore && unicode.IsDigit(cb)

		if !aIsDigit && !bIsDigit {
			return bias
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		// Get numeric value of Unicode digit
		aVal := digitValue(ca)
		bVal := digitValue(cb)

		if aVal < bVal {
			if bias == 0 {
				bias = -1
			}
		} else if aVal > bVal {
			if bias == 0 {
				bias = 1
			}
		}

		*ai++
		*bi++
	}
}

// compareLeftUnicode compares two left-aligned numbers with Unicode digit support.
func compareLeftUnicode(a, b []rune, ai, bi *int) int {
	for {
		var ca, cb rune
		aHasMore := *ai < len(a)
		bHasMore := *bi < len(b)

		if aHasMore {
			ca = a[*ai]
		}
		if bHasMore {
			cb = b[*bi]
		}

		aIsDigit := aHasMore && unicode.IsDigit(ca)
		bIsDigit := bHasMore && unicode.IsDigit(cb)

		if !aIsDigit && !bIsDigit {
			return 0
		}
		if !aIsDigit {
			return -1
		}
		if !bIsDigit {
			return 1
		}

		// Get numeric value of Unicode digit
		aVal := digitValue(ca)
		bVal := digitValue(cb)

		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}

		*ai++
		*bi++
	}
}

// digitValue returns the numeric value of a Unicode digit rune (0-9).
// For non-decimal digits, falls back to rune comparison.
func digitValue(r rune) int {
	// ASCII fast path
	if '0' <= r && r <= '9' {
		return int(r - '0')
	}
	// Unicode digit - use the digit value from Unicode
	// Most digit characters are in blocks of 10, starting at 0
	// This handles Arabic-Indic, Devanagari, etc.
	return int(r % 10)
}

// objectsEqual compares two objects for equality
func objectsEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch a := a.(type) {
	case *Integer:
		return a.Value == b.(*Integer).Value
	case *Float:
		return a.Value == b.(*Float).Value
	case *String:
		return a.Value == b.(*String).Value
	case *Boolean:
		return a.Value == b.(*Boolean).Value
	case *Null:
		return true
	default:
		return false
	}
}

// ============================================================================
// Currency Helpers
// ============================================================================

// currencyToSymbol returns the display symbol for a currency code
func currencyToSymbol(code string) string {
	switch code {
	case "USD":
		return "$"
	case "GBP":
		return "£"
	case "EUR":
		return "€"
	case "JPY":
		return "¥"
	case "CAD":
		return "CA$"
	case "AUD":
		return "AU$"
	case "HKD":
		return "HK$"
	case "SGD":
		return "S$"
	case "CNY":
		return "CN¥"
	default:
		return ""
	}
}

// ============================================================================
// Path Security Helpers
// ============================================================================

// checkPathAccess validates file system access based on security policy
func (e *Environment) checkPathAccess(path string, operation string) error {
	if e.Security == nil {
		// No policy = permissive defaults (for REPL and simple scripts)
		// Read: allowed
		// Write: allowed (changed to be permissive by default)
		// Execute: denied (still requires explicit permission)
		if operation == "execute" {
			return fmt.Errorf("execute access denied (use --allow-execute or -x)")
		}
		return nil
	}

	// Convert to absolute path and resolve symlinks for consistent comparison
	// This handles macOS /var -> /private/var symlinks and similar
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %s", err)
	}
	absPath = filepath.Clean(absPath)

	// Try to resolve symlinks. If the file doesn't exist (e.g., for write operations),
	// resolve the parent directory and append the filename.
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	} else {
		// File doesn't exist - try resolving parent directory
		dir := filepath.Dir(absPath)
		base := filepath.Base(absPath)
		if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
			absPath = filepath.Join(resolvedDir, base)
		}
	}

	switch operation {
	case "read":
		if e.Security.NoRead {
			return fmt.Errorf("file read access denied: %s", path)
		}
		// Check blacklist
		if isPathRestricted(absPath, e.Security.RestrictRead) {
			return fmt.Errorf("file read restricted: %s", path)
		}

	case "write":
		// First check if all writes are denied
		if e.Security.NoWrite {
			return fmt.Errorf("file write access denied: %s", path)
		}
		// Check blacklist (deny specific paths)
		if isPathRestricted(absPath, e.Security.RestrictWrite) {
			return fmt.Errorf("file write restricted: %s", path)
		}
		// If AllowWriteAll is true (default for pars), allow the write
		if e.Security.AllowWriteAll {
			return nil
		}
		// Otherwise check whitelist (used by basil server)
		if !isPathAllowed(absPath, e.Security.AllowWrite) {
			return fmt.Errorf("file write not allowed: %s", path)
		}

	case "execute":
		if e.Security.AllowExecuteAll {
			return nil // Unrestricted
		}
		if !isPathAllowed(absPath, e.Security.AllowExecute) {
			// Include helpful debug info in error message
			if len(e.Security.AllowExecute) > 0 {
				allowedStr := strings.Join(e.Security.AllowExecute, ", ")
				return fmt.Errorf("script execution not allowed: %s (resolved to: %s, allowed: %s)", path, absPath, allowedStr)
			}
			return fmt.Errorf("script execution not allowed: %s (no directories allowed)", path)
		}
	}

	return nil
}

// isPathAllowed checks if a path is within any allowed directory
func isPathAllowed(path string, allowList []string) bool {
	// Empty allow list means nothing is allowed
	if len(allowList) == 0 {
		return false
	}

	// Check if path is within any allowed directory
	for _, allowed := range allowList {
		// Resolve symlinks in allowed path for consistent comparison
		resolvedAllowed := allowed
		if resolved, err := filepath.EvalSymlinks(allowed); err == nil {
			resolvedAllowed = resolved
		}
		if path == resolvedAllowed || strings.HasPrefix(path, resolvedAllowed+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// isPathRestricted checks if a path is within any restricted directory
func isPathRestricted(path string, restrictList []string) bool {
	// Empty restrict list = no restrictions
	if len(restrictList) == 0 {
		return false
	}

	// Check if path is within any restricted directory
	for _, restricted := range restrictList {
		// Resolve symlinks in restricted path for consistent comparison
		resolvedRestricted := restricted
		if resolved, err := filepath.EvalSymlinks(restricted); err == nil {
			resolvedRestricted = resolved
		}
		if path == resolvedRestricted || strings.HasPrefix(path, resolvedRestricted+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// ============================================================================
// Type Checking Helpers
// ============================================================================

// typeExprEquals checks if an expression represents a type marker matching the target.
func typeExprEquals(expr ast.Expression, want string) bool {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value == want
	case *ast.ObjectLiteralExpression:
		if strObj, ok := v.Obj.(*String); ok {
			return strObj.Value == want
		}
	}
	return false
}

// isDatetimeDict checks if a dictionary is a datetime by looking for __type field
func isDatetimeDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		// Try AST literal first
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "datetime"
		}
		// Fall back to evaluation for runtime-created dictionaries
		typeObj := Eval(typeExpr, dict.Env)
		if strObj, ok := typeObj.(*String); ok {
			return strObj.Value == "datetime"
		}
	}
	return false
}

// isDurationDict checks if a dictionary is a duration by looking for __type field
func isDurationDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "duration"
		}
	}
	return false
}

// isRegexDict checks if a dictionary is a regex by looking for __type field
func isRegexDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "regex")
	}
	return false
}

// isPathDict checks if a dictionary is a path by looking for __type field
func isPathDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "path")
	}
	return false
}

// isUrlDict checks if a dictionary is a URL by looking for __type field
func isUrlDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "url")
	}
	return false
}

// isFileDict checks if a dictionary is a file handle by looking for __type field
func isFileDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "file")
	}
	return false
}

// isTagDict checks if a dictionary is a tag by looking for __type field
func isTagDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "tag")
	}
	return false
}

// isDirDict checks if a dictionary is a directory handle by looking for __type field
func isDirDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "dir")
	}
	return false
}

// isCommandHandle checks if a dictionary is a command handle by looking for __type field
func isCommandHandle(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "command")
	}
	return false
}

// isRequestDict checks if a dictionary is an HTTP request dictionary
func isRequestDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "request")
	}
	return false
}

// isResponseDict checks if a dictionary is an HTTP response dictionary
func isResponseDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		return typeExprEquals(typeExpr, "response")
	}
	return false
}

// isTruthy evaluates an object for truthiness using Python-style semantics.
// NULL, FALSE, empty strings, empty collections, and zero numbers are falsy.
func isTruthy(obj Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		// Python-style truthiness: empty collections and strings are falsy
		switch v := obj.(type) {
		case *String:
			return v.Value != ""
		case *Array:
			return len(v.Elements) > 0
		case *Dictionary:
			return len(v.Pairs) > 0
		case *Integer:
			return v.Value != 0
		case *Float:
			return v.Value != 0.0
		default:
			return true
		}
	}
}

// nativeBoolToParsBoolean converts a Go bool to a Parsley Boolean object
func nativeBoolToParsBoolean(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
