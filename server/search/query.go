package search

import (
	"regexp"
	"strings"
)

// fts5SpecialChars are special characters in FTS5 that need escaping
var fts5SpecialChars = regexp.MustCompile(`(["()])`)

// SanitizeQuery converts a user-friendly query to FTS5 syntax
// - Converts space-separated terms to AND logic: "hello world" → "hello AND world"
// - Preserves quoted phrases: "hello world" → "hello world"
// - Converts hyphen prefix to NOT: "hello -world" → "hello NOT world"
// - Escapes special FTS5 characters in terms
func SanitizeQuery(query string, raw bool) string {
	if raw {
		// Raw mode: pass through unchanged
		return query
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Parse query into tokens
	tokens := parseQueryTokens(query)
	if len(tokens) == 0 {
		return ""
	}

	// Build FTS5 query
	var parts []string
	for _, token := range tokens {
		if token.isPhrase {
			// Quoted phrase - preserve as-is
			parts = append(parts, token.value)
		} else if token.isNegation {
			// Negation term
			parts = append(parts, "NOT "+escapeToken(token.value))
		} else {
			// Regular term - escape special chars
			parts = append(parts, escapeToken(token.value))
		}
	}

	// Join with AND operator (Google-like default)
	return strings.Join(parts, " AND ")
}

type queryToken struct {
	value      string
	isPhrase   bool
	isNegation bool
}

// parseQueryTokens parses a query string into tokens
func parseQueryTokens(query string) []queryToken {
	var tokens []queryToken
	inQuotes := false
	currentToken := strings.Builder{}
	isNegation := false

	for i := 0; i < len(query); i++ {
		ch := query[i]

		if ch == '"' {
			if inQuotes {
				// End of quoted phrase
				phrase := currentToken.String()
				if phrase != "" {
					tokens = append(tokens, queryToken{
						value:    `"` + phrase + `"`,
						isPhrase: true,
					})
				}
				currentToken.Reset()
				inQuotes = false
			} else {
				// Start of quoted phrase
				inQuotes = true
			}
			continue
		}

		if inQuotes {
			// Inside quotes - add everything
			currentToken.WriteByte(ch)
			continue
		}

		// Outside quotes
		if ch == ' ' {
			// Space separator
			token := strings.TrimSpace(currentToken.String())
			if token != "" {
				tokens = append(tokens, queryToken{
					value:      token,
					isNegation: isNegation,
				})
			}
			currentToken.Reset()
			isNegation = false
			continue
		}

		if ch == '-' && currentToken.Len() == 0 {
			// Negation prefix (only at start of term)
			isNegation = true
			continue
		}

		currentToken.WriteByte(ch)
	}

	// Add final token
	token := strings.TrimSpace(currentToken.String())
	if token != "" {
		tokens = append(tokens, queryToken{
			value:      token,
			isPhrase:   inQuotes, // Handle unclosed quote
			isNegation: isNegation,
		})
	}

	return tokens
}

// escapeToken escapes special FTS5 characters in a search term
func escapeToken(token string) string {
	// Escape double quotes and parentheses
	return fts5SpecialChars.ReplaceAllString(token, `\$1`)
}
