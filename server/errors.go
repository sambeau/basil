package server

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// DevError holds information about an error to display in dev mode.
type DevError struct {
	Type     string   // "parse", "runtime", "file"
	File     string   // Full path to the file
	Line     int      // Line number (0 if unknown)
	Column   int      // Column number (0 if unknown)
	Message  string   // Error message
	Hints    []string // Suggestions for fixing the error
	BasePath string   // Base path for making paths relative (project root)
}

// FromParsleyError creates a DevError from a structured ParsleyError.
func FromParsleyError(perr *perrors.ParsleyError, basePath string) *DevError {
	errType := "runtime"
	if perr.IsParseError() {
		errType = "parse"
	}
	return &DevError{
		Type:     errType,
		File:     perr.File,
		Line:     perr.Line,
		Column:   perr.Column,
		Message:  perr.Message,
		Hints:    perr.Hints,
		BasePath: basePath,
	}
}

// SourceLine represents a line of source code for display.
type SourceLine struct {
	Number  int
	Content string
	IsError bool
}

// errorPageStyles contains the inline CSS for the error page.
const errorPageStyles = `
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #eee;
    min-height: 100vh;
    padding: 2rem;
  }
  .error-container {
    max-width: 900px;
    margin: 0 auto;
  }
  h1 {
    font-size: 1.5rem;
    margin-bottom: 1.5rem;
    color: #ff6b6b;
  }
  .error-type {
    display: inline-block;
    background: #ff6b6b;
    color: #1a1a2e;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    margin-right: 0.5rem;
  }
  .error-location {
    background: #16213e;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    margin-bottom: 1rem;
    border-left: 4px solid #ff6b6b;
  }
  .file-path {
    color: #7f8c8d;
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 0.875rem;
    word-break: break-all;
  }
  .line-info {
    color: #f39c12;
    font-weight: 600;
  }
  .error-message {
    background: #16213e;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    margin-bottom: 1.5rem;
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 0.9rem;
    line-height: 1.6;
    color: #ff6b6b;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .source-code {
    background: #0f0f23;
    border-radius: 8px;
    overflow: hidden;
  }
  .source-header {
    background: #16213e;
    padding: 0.75rem 1rem;
    font-size: 0.8rem;
    color: #7f8c8d;
    border-bottom: 1px solid #2d2d44;
  }
  .source-lines {
    padding: 1rem 0;
    overflow-x: auto;
  }
  .source-line {
    display: flex;
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 0.875rem;
    line-height: 1.6;
  }
  .source-line.error-line {
    background: rgba(255, 107, 107, 0.15);
  }
  .line-number {
    width: 4rem;
    text-align: right;
    padding-right: 1rem;
    color: #4a4a6a;
    user-select: none;
    flex-shrink: 0;
  }
  .error-line .line-number {
    color: #ff6b6b;
  }
  .line-marker {
    width: 1.5rem;
    color: #ff6b6b;
    flex-shrink: 0;
  }
  .line-content {
    flex: 1;
    white-space: pre;
    padding-right: 1rem;
  }
  /* Syntax highlighting */
  .kw { color: #c678dd; }
  .str { color: #98c379; }
  .num { color: #d19a66; }
  .tag { color: #e06c75; }
  .attr { color: #d19a66; }
  .comment { color: #5c6370; font-style: italic; }
  .fn { color: #61afef; }
  
  .error-hint {
    background: #1a3a1a;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    margin-bottom: 1.5rem;
    font-size: 0.9rem;
    color: #98c379;
    border-left: 4px solid #98c379;
  }
  .hint-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    font-weight: 500;
  }
  .hint-row {
    display: flex;
    align-items: baseline;
    margin-left: 1.75rem;
  }
  .hint-prefix {
    width: 2rem;
    color: #5c8a5c;
    flex-shrink: 0;
  }
  .hint-code {
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
  }
  
  .footer {
    margin-top: 2rem;
    padding-top: 1rem;
    border-top: 1px solid #2d2d44;
    font-size: 0.8rem;
    color: #5c6370;
  }
  .footer code {
    background: #16213e;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
  }
</style>
`

// notFoundPageStyles contains the inline CSS for the 404 page.
const notFoundPageStyles = `
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #eee;
    min-height: 100vh;
    padding: 2rem;
  }
  .container {
    max-width: 700px;
    margin: 0 auto;
  }
  h1 {
    font-size: 1.5rem;
    margin-bottom: 1.5rem;
    color: #f39c12;
  }
  .status-code {
    display: inline-block;
    background: #f39c12;
    color: #1a1a2e;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    margin-right: 0.5rem;
  }
  .info-box {
    background: #16213e;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    margin-bottom: 1rem;
    border-left: 4px solid #f39c12;
  }
  .info-box h2 {
    font-size: 0.8rem;
    color: #7f8c8d;
    margin-bottom: 0.5rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .path {
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 0.9rem;
    color: #61afef;
    word-break: break-all;
  }
  .checked-list {
    margin-top: 0.5rem;
    padding-left: 1rem;
  }
  .checked-list li {
    font-family: 'SF Mono', Monaco, 'Courier New', monospace;
    font-size: 0.85rem;
    color: #7f8c8d;
    margin-bottom: 0.25rem;
    list-style: none;
  }
  .checked-list li:before {
    content: "✗ ";
    color: #e74c3c;
  }
  .hint {
    background: #16213e;
    border-radius: 8px;
    padding: 1rem 1.25rem;
    margin-top: 1.5rem;
    font-size: 0.85rem;
    color: #7f8c8d;
  }
  .hint code {
    background: #0f0f23;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    color: #98c379;
  }
  .footer {
    margin-top: 2rem;
    padding-top: 1rem;
    border-top: 1px solid #2d2d44;
    font-size: 0.8rem;
    color: #5c6370;
  }
</style>
`

// Dev404Info holds information about a 404 to display in dev mode.
type Dev404Info struct {
	RequestPath  string   // The URL path that was requested
	StaticRoot   string   // The public_dir that was checked (if any)
	CheckedPaths []string // Paths that were checked (relative)
	HasHandler   bool     // Whether a route handler exists
	RoutePath    string   // The route path that matched (e.g., "/" or "/admin")
	BasePath     string   // Base path for making paths relative
}

// renderDev404Page writes a styled 404 page for development mode.
func renderDev404Page(w http.ResponseWriter, info Dev404Info) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"utf-8\">\n")
	sb.WriteString("<title>404 Not Found</title>\n")
	sb.WriteString(notFoundPageStyles)
	sb.WriteString("</head>\n<body>\n")

	sb.WriteString("<div class=\"container\">\n")

	// Header
	sb.WriteString("<h1><span class=\"status-code\">404</span> Not Found</h1>\n")

	// Requested path
	sb.WriteString("<div class=\"info-box\">\n")
	sb.WriteString("<h2>Requested</h2>\n")
	sb.WriteString("<div class=\"path\">")
	sb.WriteString(html.EscapeString(info.RequestPath))
	sb.WriteString("</div>\n")
	sb.WriteString("</div>\n")

	// What was checked
	if len(info.CheckedPaths) > 0 || info.StaticRoot != "" {
		sb.WriteString("<div class=\"info-box\">\n")
		sb.WriteString("<h2>Checked</h2>\n")
		sb.WriteString("<ul class=\"checked-list\">\n")
		for _, p := range info.CheckedPaths {
			sb.WriteString("<li>")
			sb.WriteString(html.EscapeString(p))
			sb.WriteString("</li>\n")
		}
		sb.WriteString("</ul>\n")
		sb.WriteString("</div>\n")
	}

	// Hint
	sb.WriteString("<div class=\"hint\">\n")
	if info.StaticRoot == "" && !info.HasHandler {
		sb.WriteString("No <code>public_dir</code> or handler configured. ")
		sb.WriteString("Add a route in <code>basil.yaml</code> to handle this path.")
	} else if info.StaticRoot == "" {
		sb.WriteString("This path wasn't handled by your route. ")
		sb.WriteString("Add a <code>public_dir</code> to serve static files.")
	} else {
		sb.WriteString("Create the file in your <code>public_dir</code> or handle this path in your route handler.")
	}
	sb.WriteString("\n</div>\n")

	// Footer
	sb.WriteString("<div class=\"footer\">")
	sb.WriteString("This is a development-only page.")
	sb.WriteString("</div>\n")

	sb.WriteString("</div>\n") // .container

	sb.WriteString("</body>\n</html>")

	w.Write([]byte(sb.String()))
}

// makeRelativePath converts an absolute path to a relative path based on the base path.
// If the path cannot be made relative, it returns the original path.
func makeRelativePath(path, basePath string) string {
	if basePath == "" || path == "" {
		return path
	}
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return path
	}
	// Prefix with ./ for clarity if it doesn't start with ../
	if !strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

// makeMessageRelative replaces absolute paths in an error message with relative paths.
func makeMessageRelative(message, basePath string) string {
	if basePath == "" {
		return message
	}
	// Replace the base path with ./ in the message
	// Handle both with and without trailing slash
	baseWithSlash := strings.TrimSuffix(basePath, "/") + "/"
	message = strings.ReplaceAll(message, baseWithSlash, "./")
	message = strings.ReplaceAll(message, basePath, ".")
	return message
}

// improveErrorMessage rewrites confusing parser errors to be more helpful.
// Returns the improved message and an optional hint.
func improveErrorMessage(message string) (improved string, hint string) {
	// Strip cascade errors - only show the first error
	// Multi-line error messages often have cascade errors on subsequent lines
	lines := strings.Split(message, "\n")
	if len(lines) > 1 {
		// Keep only the first non-empty line (the primary error)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				message = line
				break
			}
		}
	}

	// Pattern: "expected '(', got 'x'"
	// This happens when parentheses are required but missing (e.g., fn x { } instead of fn(x) { })
	ifParenPattern := regexp.MustCompile(`expected '\(', got '([^']+)'`)
	if matches := ifParenPattern.FindStringSubmatch(message); len(matches) > 1 {
		// Check if this looks like a parameter name
		if matches[1] != "(" && matches[1] != "{" {
			improved = fmt.Sprintf("Missing parentheses")
			hint = "Function parameters need parentheses: fn(x) { ... } or fn(a, b) { ... }"
			return improved, hint
		}
	}

	// Pattern: "unexpected '#'" - Python-style comment
	if strings.Contains(message, "unexpected '#'") {
		improved = "Invalid comment syntax"
		hint = "Use // for comments, not #. Parsley uses C-style comments: // single line or /* multi-line */"
		return improved, hint
	}

	// Pattern: "expected identifier as dictionary key, got opening tag"
	// This happens when parser sees <ComponentName> but ComponentName is undefined
	dictKeyPattern := regexp.MustCompile(`expected identifier as dictionary key, got opening tag`)
	if dictKeyPattern.MatchString(message) {
		// Try to extract what tag was found from the source context
		// For now, give a generic but helpful message
		improved = "Unrecognized component tag"
		hint = "Is the component imported? Check that the import path is correct and the component is exported."
		return improved, hint
	}

	// Pattern: "unexpected 'SomeName'" where SomeName starts with uppercase
	// Usually a cascade error from undefined component
	unexpectedUpperPattern := regexp.MustCompile(`unexpected '([A-Z][a-zA-Z0-9]*)'`)
	if matches := unexpectedUpperPattern.FindStringSubmatch(message); len(matches) > 1 {
		componentName := matches[1]
		improved = fmt.Sprintf("'%s' is not defined", componentName)
		hint = fmt.Sprintf("Did you forget to import %s? Component names must start with an uppercase letter.", componentName)
		return improved, hint
	}

	// === Runtime error patterns ===

	// Pattern: "identifier not found: console" - JavaScript console.log()
	if strings.Contains(message, "identifier not found: console") {
		improved = "'console' is not defined"
		hint = "Use log() for debugging output. Example: log(\"value:\", myVar)"
		return improved, hint
	}

	// Pattern: "identifier not found: print" - Python print()
	if strings.Contains(message, "identifier not found: print") {
		improved = "'print' is not defined"
		hint = "Use log() for output. Example: log(\"hello world\")"
		return improved, hint
	}

	// Pattern: "identifier not found: document" - JavaScript DOM
	if strings.Contains(message, "identifier not found: document") {
		improved = "'document' is not defined"
		hint = "Parsley runs on the server, not in the browser. DOM APIs are not available."
		return improved, hint
	}

	// Pattern: "identifier not found: window" - JavaScript browser global
	if strings.Contains(message, "identifier not found: window") {
		improved = "'window' is not defined"
		hint = "Parsley runs on the server, not in the browser. Browser globals are not available."
		return improved, hint
	}

	// Pattern: "identifier not found: require" - Node.js require()
	if strings.Contains(message, "identifier not found: require") {
		improved = "'require' is not defined"
		hint = "Use 'import' to load modules. Example: import utils from \"./utils.pars\""
		return improved, hint
	}

	// No improvements - return original
	return message, ""
}

// renderDevErrorPage generates an HTML error page for dev mode.
func renderDevErrorPage(w http.ResponseWriter, devErr *DevError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	// Get source context if we have a file and line number
	var sourceLines []SourceLine
	if devErr.File != "" && devErr.Line > 0 {
		sourceLines = getSourceContext(devErr.File, devErr.Line, 5)
	}

	// Make paths relative for display
	displayFile := makeRelativePath(devErr.File, devErr.BasePath)

	// Improve the error message for common confusing cases
	improvedMessage, hint := improveErrorMessage(devErr.Message)
	displayMessage := makeMessageRelative(improvedMessage, devErr.BasePath)

	// Build the page
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"utf-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	sb.WriteString("<title>Error - Basil Dev</title>\n")
	sb.WriteString(errorPageStyles)
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString("<div class=\"error-container\">\n")

	// Header
	sb.WriteString("<h1>🌿 Parsley Error</h1>\n")

	// Error type and location
	sb.WriteString("<div class=\"error-location\">\n")
	sb.WriteString(fmt.Sprintf("<span class=\"error-type\">%s error</span>\n", html.EscapeString(devErr.Type)))

	if displayFile != "" {
		sb.WriteString("<span class=\"file-path\">")
		sb.WriteString(html.EscapeString(displayFile))
		if devErr.Line > 0 {
			sb.WriteString(fmt.Sprintf(" : <span class=\"line-info\">%d</span>", devErr.Line))
			if devErr.Column > 0 {
				sb.WriteString(fmt.Sprintf(" : <span class=\"line-info\">%d</span>", devErr.Column))
			}
		}
		sb.WriteString("</span>\n")
	}
	sb.WriteString("</div>\n")

	// Error message
	sb.WriteString("<div class=\"error-message\">")
	sb.WriteString(html.EscapeString(displayMessage))
	sb.WriteString("</div>\n")

	// Hint (if any) - prioritize structured hints from DevError, then improved message hints
	if len(devErr.Hints) > 0 {
		sb.WriteString("<div class=\"error-hint\">\n")
		sb.WriteString("<div class=\"hint-header\">💡 Try</div>\n")
		for i, h := range devErr.Hints {
			sb.WriteString("<div class=\"hint-row\">")
			if i == 0 {
				sb.WriteString("<span class=\"hint-prefix\"></span>")
			} else {
				sb.WriteString("<span class=\"hint-prefix\">or</span>")
			}
			sb.WriteString("<span class=\"hint-code\">")
			sb.WriteString(html.EscapeString(h))
			sb.WriteString("</span></div>\n")
		}
		sb.WriteString("</div>\n")
	} else if hint != "" {
		sb.WriteString("<div class=\"error-hint\">\n")
		sb.WriteString("<div class=\"hint-header\">💡 Hint</div>\n")
		sb.WriteString("<div class=\"hint-row\">")
		sb.WriteString("<span class=\"hint-prefix\"></span>")
		sb.WriteString("<span class=\"hint-code\">")
		sb.WriteString(html.EscapeString(hint))
		sb.WriteString("</span></div>\n")
		sb.WriteString("</div>\n")
	}

	// Source code context
	if len(sourceLines) > 0 {
		sb.WriteString("<div class=\"source-code\">\n")
		sb.WriteString("<div class=\"source-header\">Source</div>\n")
		sb.WriteString("<div class=\"source-lines\">\n")

		for _, line := range sourceLines {
			errorClass := ""
			marker := "  "
			if line.IsError {
				errorClass = " error-line"
				marker = "→ "
			}

			sb.WriteString(fmt.Sprintf("<div class=\"source-line%s\">", errorClass))
			sb.WriteString(fmt.Sprintf("<span class=\"line-number\">%d</span>", line.Number))
			sb.WriteString(fmt.Sprintf("<span class=\"line-marker\">%s</span>", marker))
			sb.WriteString("<span class=\"line-content\">")
			sb.WriteString(highlightParsley(line.Content))
			sb.WriteString("</span>")
			sb.WriteString("</div>\n")
		}

		sb.WriteString("</div>\n")
		sb.WriteString("</div>\n")
	}

	// Footer
	sb.WriteString("<div class=\"footer\">")
	sb.WriteString("Fix the error and save — this page will automatically reload.")
	sb.WriteString("</div>\n")

	sb.WriteString("</div>\n") // .error-container

	// Note: live reload script is injected by injectLiveReload middleware

	sb.WriteString("</body>\n</html>")

	w.Write([]byte(sb.String()))
}

// getSourceContext reads a file and returns lines around the error line.
func getSourceContext(filePath string, errorLine, contextLines int) []SourceLine {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []SourceLine
	scanner := bufio.NewScanner(file)
	lineNum := 0

	startLine := errorLine - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := errorLine + contextLines

	for scanner.Scan() {
		lineNum++
		if lineNum < startLine {
			continue
		}
		if lineNum > endLine {
			break
		}

		lines = append(lines, SourceLine{
			Number:  lineNum,
			Content: scanner.Text(),
			IsError: lineNum == errorLine,
		})
	}

	return lines
}

// Regex patterns for syntax highlighting
var (
	// Keywords
	keywordPattern = regexp.MustCompile(`\b(let|fn|if|else|for|in|export|import|true|false|nil|return|and|or|not)\b`)

	// Strings (double-quoted)
	stringPattern = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)

	// Numbers
	numberPattern = regexp.MustCompile(`\b\d+\.?\d*\b`)

	// HTML tags - match opening, closing, and self-closing tags
	tagPattern = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9]*[^>]*>`)

	// Comments
	commentPattern = regexp.MustCompile(`//.*$`)

	// Function calls (identifier followed by parenthesis)
	fnCallPattern = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\(`)
)

// highlightParsley applies syntax highlighting to a line of Parsley code.
// It returns HTML with span elements for styling.
// The approach: find all tokens to highlight, sort by position, then build output
// with proper HTML escaping for non-highlighted parts.
func highlightParsley(code string) string {
	type highlight struct {
		start int
		end   int
		class string
		text  string
	}

	var highlights []highlight

	// Find all matches for each pattern
	// Comments (highest priority - will override others)
	for _, m := range commentPattern.FindAllStringIndex(code, -1) {
		highlights = append(highlights, highlight{m[0], m[1], "comment", code[m[0]:m[1]]})
	}

	// Strings
	for _, m := range stringPattern.FindAllStringIndex(code, -1) {
		highlights = append(highlights, highlight{m[0], m[1], "str", code[m[0]:m[1]]})
	}

	// HTML tags
	for _, m := range tagPattern.FindAllStringIndex(code, -1) {
		highlights = append(highlights, highlight{m[0], m[1], "tag", code[m[0]:m[1]]})
	}

	// Keywords
	for _, m := range keywordPattern.FindAllStringIndex(code, -1) {
		highlights = append(highlights, highlight{m[0], m[1], "kw", code[m[0]:m[1]]})
	}

	// Numbers
	for _, m := range numberPattern.FindAllStringIndex(code, -1) {
		highlights = append(highlights, highlight{m[0], m[1], "num", code[m[0]:m[1]]})
	}

	// Function calls - extract just the function name
	for _, m := range fnCallPattern.FindAllStringSubmatchIndex(code, -1) {
		if len(m) >= 4 {
			// m[2]:m[3] is the captured group (function name)
			fnName := code[m[2]:m[3]]
			// Don't highlight if it's a keyword
			if !keywordPattern.MatchString(fnName) {
				highlights = append(highlights, highlight{m[2], m[3], "fn", fnName})
			}
		}
	}

	// Sort by start position
	for i := 0; i < len(highlights)-1; i++ {
		for j := i + 1; j < len(highlights); j++ {
			if highlights[j].start < highlights[i].start {
				highlights[i], highlights[j] = highlights[j], highlights[i]
			}
		}
	}

	// Remove overlapping highlights (keep first/higher priority)
	var filtered []highlight
	lastEnd := 0
	for _, h := range highlights {
		if h.start >= lastEnd {
			filtered = append(filtered, h)
			lastEnd = h.end
		}
	}

	// Build output
	var result strings.Builder
	pos := 0
	for _, h := range filtered {
		// Add escaped text before this highlight
		if h.start > pos {
			result.WriteString(escapeForCodeDisplay(code[pos:h.start]))
		}
		// Add highlighted text (escaped inside the span)
		result.WriteString(`<span class="`)
		result.WriteString(h.class)
		result.WriteString(`">`)
		result.WriteString(escapeForCodeDisplay(h.text))
		result.WriteString(`</span>`)
		pos = h.end
	}
	// Add remaining text
	if pos < len(code) {
		result.WriteString(escapeForCodeDisplay(code[pos:]))
	}

	return result.String()
}

// escapeForCodeDisplay escapes text for display in HTML code blocks.
// Only escapes < and > (which would be interpreted as HTML tags),
// and & (to prevent entity interpretation). Keeps quotes readable.
func escapeForCodeDisplay(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
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
		if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
			file = rest[:colonIdx]
			// Clean message starts after the colon (and any whitespace)
			remaining := rest[colonIdx+1:]
			cleanMsg = strings.TrimLeft(remaining, " \n\t")
		}
	}

	// Pattern: "error[s] in [module] <path>: <message>"
	if file == "" {
		if idx := strings.Index(errMsg, " in "); idx != -1 {
			rest := errMsg[idx+4:]
			// Handle "module ./path:" format
			if strings.HasPrefix(rest, "module ") {
				rest = rest[7:] // skip "module "
			}
			// Find the colon after the path (could be ": " or ":\n")
			if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
				file = rest[:colonIdx]
				// Clean message starts after the colon (and any whitespace)
				remaining := rest[colonIdx+1:]
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

	// Basic error information
	errorMap := map[string]interface{}{
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

		if file != "" {
			errorMap["file"] = file
			errorMap["line"] = line
			errorMap["column"] = col
			errorMap["message_text"] = errMsg

			// Try to get source context
			if sourceLines := s.getSourceContext(file, line, 3); len(sourceLines) > 0 {
				// Convert to array of maps for Parsley
				linesArray := make([]interface{}, len(sourceLines))
				for i, sl := range sourceLines {
					linesArray[i] = map[string]interface{}{
						"number":   sl.Number,
						"content":  sl.Content,
						"is_error": sl.IsError,
					}
				}
				errorMap["source"] = linesArray
			}
		}

		// Add request information
		errorMap["request"] = map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"query":  r.URL.RawQuery,
		}
	}

	errorObj, _ := parsley.ToParsley(errorMap)
	env.Set("error", errorObj)

	// Add Basil metadata
	basilMap := map[string]interface{}{
		"version": s.version,
		"dev":     s.config.Server.Dev,
	}
	basilObj, _ := parsley.ToParsley(basilMap)
	env.Set("basil", basilObj)

	return env
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
	startLine := errorLine - contextLines
	if startLine < 1 {
		startLine = 1
	}
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
	case []interface{}:
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
		// Fallback to plain text
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
