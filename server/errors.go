package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// SourceLine represents a line of source code for display.
type SourceLine struct {
	Number  int
	Content string
	IsError bool
}

// extractLineInfo attempts to extract file, line, and column information from an error message.
// Returns the cleaned message without the location prefix.
func extractLineInfo(errMsg string) (file string, line, col int, cleanMsg string) {
	cleanMsg = errMsg

	// Common patterns:
	// "parse error in /path/file.pars: message"
	// "parse errors in module ./path/file.pars:\n  message"
	// "in module ./path/file.pars: line X, column Y: message"
	// "/path/file.pars:12: message"
	// "/path/file.pars:12:5: message"
	// "script error in /path/file.pars: message"

	// Pattern: "in module <path>: <message>" (starts with "in module")
	if strings.HasPrefix(errMsg, "in module ") {
		rest := errMsg[10:] // skip "in module "
		// Find the colon after the path
		if before, after, ok := strings.Cut(rest, ":"); ok {
			file = before
			// Clean message starts after the colon (and any whitespace)
			remaining := after
			cleanMsg = strings.TrimLeft(remaining, " \n\t")
		}
	}

	// Pattern: "error[s] in [module] <path>: <message>"
	if file == "" {
		if _, after, ok := strings.Cut(errMsg, " in "); ok {
			rest := after
			// Handle "module ./path:" format
			if strings.HasPrefix(rest, "module ") {
				rest = rest[7:] // skip "module "
			}
			// Find the colon after the path (could be ": " or ":\n")
			if before, after, ok := strings.Cut(rest, ":"); ok {
				file = before
				// Clean message starts after the colon (and any whitespace)
				remaining := after
				cleanMsg = strings.TrimLeft(remaining, " \n\t")
			}
		}
	}

	// Pattern: "<path>:<line>: <message>" or "<path>:<line>:<col>: <message>"
	// Try to extract line number from file path if it contains ':'
	if file != "" {
		parts := strings.Split(file, ":")
		if len(parts) >= 2 {
			file = parts[0]
			if n, err := strconv.Atoi(parts[1]); err == nil {
				line = n
			}
			if len(parts) >= 3 {
				if n, err := strconv.Atoi(parts[2]); err == nil {
					col = n
				}
			}
		}
	}

	// Check for "at line X, column Y" pattern (captures both line and column)
	if line == 0 || col == 0 {
		atLineColPattern := regexp.MustCompile(`at line (\d+),?\s*column\s*(\d+)`)
		if matches := atLineColPattern.FindStringSubmatch(errMsg); len(matches) > 2 {
			if line == 0 {
				if n, err := strconv.Atoi(matches[1]); err == nil {
					line = n
				}
			}
			if col == 0 {
				if n, err := strconv.Atoi(matches[2]); err == nil {
					col = n
				}
			}
		}
	}

	// Check for "at line X" pattern (without column)
	if line == 0 {
		linePattern := regexp.MustCompile(`at line (\d+)`)
		if matches := linePattern.FindStringSubmatch(errMsg); len(matches) > 1 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				line = n
			}
		}
	}

	// Check for "line X, col Y" or "line X, column Y" pattern
	if line == 0 {
		lineColPattern := regexp.MustCompile(`line (\d+)(?:,?\s*col(?:umn)?\s*(\d+))?`)
		if matches := lineColPattern.FindStringSubmatch(errMsg); len(matches) > 1 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				line = n
			}
			if len(matches) > 2 && matches[2] != "" {
				if n, err := strconv.Atoi(matches[2]); err == nil {
					col = n
				}
			}
		}
	}

	return file, line, col, cleanMsg
}

// createErrorEnv creates an environment for rendering error pages
func (s *Server) createErrorEnv(r *http.Request, code int, err error) *evaluator.Environment {
	env := evaluator.NewEnvironment()

	// Load shared devtools components (for dev error pages)
	if s.config.Server.Dev {
		loadDevToolsComponents(env)
	}

	// Basic error information
	errorMap := map[string]any{
		"code":    code,
		"message": http.StatusText(code),
	}

	// In dev mode, add detailed error information
	if s.config.Server.Dev && err != nil {
		errorMap["details"] = err.Error()

		// Try to extract file, line, column from error message
		// Format: "message at file:line:col"
		var file string
		var line, col int
		errMsg := err.Error()

		// Parse "message at file:line:col" format
		if parts := regexp.MustCompile(` at (.+):(\d+):(\d+)$`).FindStringSubmatch(errMsg); len(parts) == 4 {
			file = parts[1]
			line, _ = strconv.Atoi(parts[2])
			col, _ = strconv.Atoi(parts[3])
			// Extract just the message part
			errMsg = regexp.MustCompile(` at .+:\d+:\d+$`).ReplaceAllString(errMsg, "")
		}

		// Always set message_text
		errorMap["message_text"] = errMsg

		if file != "" {
			// Make file path relative to base directory for cleaner display
			displayFile := file
			if s.config.BaseDir != "" && strings.HasPrefix(file, s.config.BaseDir) {
				displayFile = strings.TrimPrefix(file, s.config.BaseDir)
				displayFile = strings.TrimPrefix(displayFile, "/")
			}
			errorMap["file"] = displayFile
			errorMap["line"] = line
			errorMap["column"] = col

			// Try to get source context (use original absolute path)
			if sourceLines := s.getSourceContext(file, line, 3); len(sourceLines) > 0 {
				// Convert to array of maps for Parsley
				linesArray := make([]any, len(sourceLines))
				for i, sl := range sourceLines {
					linesArray[i] = map[string]any{
						"number":   sl.Number,
						"content":  sl.Content,
						"is_error": sl.IsError,
					}
				}
				errorMap["source"] = linesArray
			}
		}

		// Add request information
		errorMap["request"] = map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"query":  r.URL.RawQuery,
		}

		// Add params (sanitized for display)
		errorMap["params"] = extractSanitizedParams(r)
	}

	errorObj, _ := parsley.ToParsley(errorMap)
	env.Set("error", errorObj)

	// Add Basil metadata for error templates and expose as `basil`
	basilMap := map[string]any{
		"version": s.version,
		"dev":     s.config.Server.Dev,
	}
	basilObj, _ := parsley.ToParsley(basilMap)
	env.BasilCtx = basilObj.(*evaluator.Dictionary)
	env.Set("basil", env.BasilCtx)

	return env
}

// sensitiveParamPatterns are field names that should be redacted in error displays
var sensitiveParamPatterns = []string{
	"password", "passwd", "pwd",
	"secret", "token", "key", "auth",
	"credential", "api_key", "apikey",
	"private", "session",
}

// isSensitiveParam checks if a parameter name looks sensitive
func isSensitiveParam(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range sensitiveParamPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// truncateValue truncates long values for display
func truncateValue(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "…"
}

// extractSanitizedParams extracts request params for error display
// Redacts sensitive fields and truncates long values
func extractSanitizedParams(r *http.Request) []map[string]any {
	const maxValueLen = 200
	params := make([]map[string]any, 0)

	// Helper to add a param
	addParam := func(name, value, source string) {
		var displayValue string
		redacted := false
		if isSensitiveParam(name) {
			displayValue = "••••••••"
			redacted = true
		} else {
			displayValue = truncateValue(value, maxValueLen)
		}
		params = append(params, map[string]any{
			"name":     name,
			"value":    displayValue,
			"source":   source,
			"redacted": redacted,
		})
	}

	// Query parameters
	for key, values := range r.URL.Query() {
		for _, v := range values {
			addParam(key, v, "query")
		}
	}

	// Form parameters (POST/PUT/PATCH)
	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		contentType := r.Header.Get("Content-Type")

		if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			// Parse form if not already parsed
			if r.PostForm == nil {
				_ = r.ParseForm()
			}
			for key, values := range r.PostForm {
				for _, v := range values {
					addParam(key, v, "form")
				}
			}
		} else if strings.HasPrefix(contentType, "multipart/form-data") {
			// Parse multipart if not already parsed
			if r.MultipartForm == nil {
				_ = r.ParseMultipartForm(32 << 20)
			}
			if r.MultipartForm != nil {
				// Text values only (not file uploads)
				for key, values := range r.MultipartForm.Value {
					for _, v := range values {
						addParam(key, v, "form")
					}
				}
				// Note file uploads (without content)
				for key, files := range r.MultipartForm.File {
					for _, fh := range files {
						params = append(params, map[string]any{
							"name":     key,
							"value":    fmt.Sprintf("[file: %s, %d bytes]", fh.Filename, fh.Size),
							"source":   "file",
							"redacted": false,
						})
					}
				}
			}
		} else if strings.HasPrefix(contentType, "application/json") {
			// For JSON, try to extract top-level keys
			if r.Body != nil {
				body, _ := io.ReadAll(r.Body)
				// Restore body for subsequent reads
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				var data map[string]any
				if json.Unmarshal(body, &data) == nil {
					for key, value := range data {
						valueStr := fmt.Sprintf("%v", value)
						addParam(key, valueStr, "json")
					}
				}
			}
		}
	}

	return params
}

// getSourceContext extracts source lines around an error line
func (s *Server) getSourceContext(filePath string, errorLine, contextLines int) []SourceLine {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []SourceLine
	scanner := bufio.NewScanner(file)
	lineNum := 1
	startLine := max(errorLine-contextLines, 1)
	endLine := errorLine + contextLines

	for scanner.Scan() {
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, SourceLine{
				Number:  lineNum,
				Content: scanner.Text(),
				IsError: lineNum == errorLine,
			})
		}
		if lineNum > endLine {
			break
		}
		lineNum++
	}

	return lines
}

// renderPreludeError renders an error page from the prelude
// Returns true if successfully rendered, false if fallback needed
func (s *Server) renderPreludeError(w http.ResponseWriter, r *http.Request, code int, err error) bool {
	// Determine which error page to use
	var pageName string
	if s.config.Server.Dev && err != nil {
		// Dev mode with error: use detailed dev error page
		pageName = "errors/dev_error.pars"
	} else {
		// Try specific error code page
		pageName = fmt.Sprintf("errors/%d.pars", code)
	}

	// Get the AST from prelude
	program := GetPreludeAST(pageName)
	if program == nil {
		// Try fallback to 500 page
		if code != 500 {
			program = GetPreludeAST("errors/500.pars")
		}
		if program == nil {
			// No error page available
			return false
		}
	}

	// Create environment with error details
	env := s.createErrorEnv(r, code, err)

	// Evaluate the error page
	result := evaluator.Eval(program, env)

	// If evaluation failed, don't recurse - return false for fallback
	if _, isErr := result.(*evaluator.Error); isErr {
		s.logError("error rendering error page %s: %s", pageName, result.Inspect())
		return false
	}

	// Convert to Go value - should be a string or array of strings
	value := parsley.FromParsley(result)

	var html string
	switch v := value.(type) {
	case string:
		html = v
	case []any:
		// Join array elements into a string
		var parts []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			} else {
				parts = append(parts, fmt.Sprint(item))
			}
		}
		html = strings.Join(parts, "")
	default:
		s.logError("error page %s did not return a string or array, got %T", pageName, value)
		return false
	}

	// Write the response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprint(w, html)
	return true
}

// handle404 renders a 404 error page
func (s *Server) handle404(w http.ResponseWriter, r *http.Request) {
	// Try prelude error page first
	if !s.renderPreludeError(w, r, http.StatusNotFound, nil) {
		// Fallback to plain text
		http.NotFound(w, r)
	}
}

// handle500 renders a 500 error page
func (s *Server) handle500(w http.ResponseWriter, r *http.Request, err error) {
	// Try prelude error page first
	if !s.renderPreludeError(w, r, http.StatusInternalServerError, err) {
		// Fallback - in dev mode, check if dev_error.pars itself has errors
		if s.config.Server.Dev {
			if _, parseErr := GetPreludeASTWithError("errors/dev_error.pars"); parseErr != nil {
				// dev_error.pars has a parse error - show both errors
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "Handler Error:\n%v\n\n---\n\nAdditionally, dev_error.pars failed to render:\n%v\n", err, parseErr)
				return
			}
		}
		// Fallback to plain text
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
