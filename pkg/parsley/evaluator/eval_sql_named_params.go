package evaluator

import (
	"strings"
	"unicode"
)

// sqlScanResult holds the result of scanning a SQL string for placeholders.
type sqlScanResult struct {
	hasNamed      bool     // true if any :name placeholders found
	hasPositional bool     // true if any ? placeholders found
	names         []string // named params in order of appearance (may repeat)
}

// scanSQLNamedParams scans a SQL string and returns information about
// placeholder usage. It correctly skips:
//   - :: sequences (Postgres type casts, e.g. column::text)
//   - Content inside single-quoted string literals ('...')
//   - Content inside $$ dollar-quoted strings (Postgres)
//   - :name where the character after : is not a letter or underscore
func scanSQLNamedParams(sql string) sqlScanResult {
	result := sqlScanResult{}
	i := 0
	n := len(sql)

	for i < n {
		ch := sql[i]

		// Single-quoted string literal: skip until closing '
		// Handle '' escape sequences inside strings.
		if ch == '\'' {
			i++
			for i < n {
				if sql[i] == '\'' {
					i++
					// '' is an escaped quote inside a string — continue
					if i < n && sql[i] == '\'' {
						i++
						continue
					}
					break
				}
				i++
			}
			continue
		}

		// Postgres $$ dollar-quoting: skip until matching $$
		if ch == '$' && i+1 < n && sql[i+1] == '$' {
			i += 2
			for i+1 < n {
				if sql[i] == '$' && sql[i+1] == '$' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// SQL line comments: -- until end of line
		if ch == '-' && i+1 < n && sql[i+1] == '-' {
			for i < n && sql[i] != '\n' {
				i++
			}
			continue
		}

		// SQL block comments: /* ... */
		if ch == '/' && i+1 < n && sql[i+1] == '*' {
			i += 2
			for i+1 < n {
				if sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		// Positional placeholder ?
		if ch == '?' {
			result.hasPositional = true
			i++
			continue
		}

		// Named parameter :name
		// Skip :: (Postgres type cast) — the second : will also be skipped here
		if ch == ':' {
			// Skip if preceded by : (i.e., this is the second colon in ::)
			if i > 0 && sql[i-1] == ':' {
				i++
				continue
			}
			// Skip if next char is also : (start of ::)
			if i+1 < n && sql[i+1] == ':' {
				i += 2
				continue
			}
			// Named param requires next char to be a letter or underscore
			if i+1 < n && (unicode.IsLetter(rune(sql[i+1])) || sql[i+1] == '_') {
				// Collect the identifier
				start := i + 1
				j := start
				for j < n && (unicode.IsLetter(rune(sql[j])) || unicode.IsDigit(rune(sql[j])) || sql[j] == '_') {
					j++
				}
				name := sql[start:j]
				result.hasNamed = true
				result.names = append(result.names, name)
				i = j
				continue
			}
			i++
			continue
		}

		i++
	}

	return result
}

// rewriteNamedParams rewrites a SQL string containing :name placeholders to
// driver-native placeholders ($N for postgres/sqlite, ? for mysql), and
// returns the rewritten SQL and the ordered params slice.
//
// propsDict is the evaluated params dictionary from the <SQL> tag.
// scan is the result of scanSQLNamedParams on the same SQL string.
// driver is "sqlite", "postgres", or "mysql".
func rewriteNamedParams(sql string, scan sqlScanResult, propsDict *Dictionary, env *Environment, driver string) (string, []any, *Error) {
	// Pre-evaluate all props into a Go map for fast lookup
	propValues := make(map[string]any, len(propsDict.Pairs))
	for key, expr := range propsDict.Pairs {
		val := Eval(expr, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}
		propValues[key] = objectToGoValue(val)
	}

	// Validate that every name in the scan has a corresponding prop
	for _, name := range scan.names {
		if _, ok := propValues[name]; !ok {
			available := make([]string, 0, len(propsDict.Pairs))
			for k := range propsDict.Pairs {
				available = append(available, ":"+k)
			}
			availStr := strings.Join(available, ", ")
			if availStr == "" {
				availStr = "(none)"
			}
			return "", nil, newSQLError("SQL-0005", map[string]any{
				"Name":      name,
				"Available": availStr,
			})
		}
	}

	// Rewrite SQL, replacing each :name with the driver-native placeholder
	var out strings.Builder
	params := make([]any, 0, len(scan.names))
	paramIdx := 1
	i := 0
	n := len(sql)

	for i < n {
		ch := sql[i]

		// Single-quoted string literal: copy verbatim
		if ch == '\'' {
			out.WriteByte(ch)
			i++
			for i < n {
				out.WriteByte(sql[i])
				if sql[i] == '\'' {
					i++
					if i < n && sql[i] == '\'' {
						out.WriteByte(sql[i])
						i++
						continue
					}
					break
				}
				i++
			}
			continue
		}

		// Postgres $$ dollar-quoting: copy verbatim
		if ch == '$' && i+1 < n && sql[i+1] == '$' {
			out.WriteByte(sql[i])
			out.WriteByte(sql[i+1])
			i += 2
			for i+1 < n {
				out.WriteByte(sql[i])
				if sql[i] == '$' && sql[i+1] == '$' {
					out.WriteByte(sql[i+1])
					i += 2
					break
				}
				i++
			}
			continue
		}

		// SQL line comments: copy verbatim
		if ch == '-' && i+1 < n && sql[i+1] == '-' {
			for i < n && sql[i] != '\n' {
				out.WriteByte(sql[i])
				i++
			}
			continue
		}

		// SQL block comments: copy verbatim
		if ch == '/' && i+1 < n && sql[i+1] == '*' {
			out.WriteByte(sql[i])
			out.WriteByte(sql[i+1])
			i += 2
			for i+1 < n {
				out.WriteByte(sql[i])
				if sql[i] == '*' && sql[i+1] == '/' {
					out.WriteByte(sql[i+1])
					i += 2
					break
				}
				i++
			}
			continue
		}

		// :: Postgres type cast: copy both colons verbatim
		if ch == ':' && i+1 < n && sql[i+1] == ':' {
			out.WriteByte(':')
			out.WriteByte(':')
			i += 2
			continue
		}

		// :name named parameter
		if ch == ':' && i+1 < n && (unicode.IsLetter(rune(sql[i+1])) || sql[i+1] == '_') {
			// Collect identifier
			j := i + 1
			for j < n && (unicode.IsLetter(rune(sql[j])) || unicode.IsDigit(rune(sql[j])) || sql[j] == '_') {
				j++
			}
			name := sql[i+1 : j]
			// Write driver-native placeholder
			out.WriteString(sqlPlaceholder(driver, paramIdx))
			paramIdx++
			params = append(params, propValues[name])
			i = j
			continue
		}

		out.WriteByte(ch)
		i++
	}

	return out.String(), params, nil
}
