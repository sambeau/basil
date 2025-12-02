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
)

// DevError holds information about an error to display in dev mode.
type DevError struct {
	Type     string // "parse", "runtime", "file"
	File     string // Full path to the file
	Line     int    // Line number (0 if unknown)
	Column   int    // Column number (0 if unknown)
	Message  string // Error message
	BasePath string // Base path for making paths relative (project root)
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

	// Pattern: "expected '(', got 'x'" after 'if' keyword
	// User wrote: if x > 5 { } (Go/Python style without parens)
	ifParenPattern := regexp.MustCompile(`expected '\(', got '([^']+)'`)
	if matches := ifParenPattern.FindStringSubmatch(message); len(matches) > 1 {
		// Check if this looks like a condition variable
		if matches[1] != "(" && matches[1] != "{" {
			improved = fmt.Sprintf("Missing parentheses around condition")
			hint = "Parsley requires parentheses: if (condition) { } and for (x in arr) { }"
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
			sb.WriteString(fmt.Sprintf(":<span class=\"line-info\">%d</span>", devErr.Line))
			if devErr.Column > 0 {
				sb.WriteString(fmt.Sprintf(":<span class=\"line-info\">%d</span>", devErr.Column))
			}
		}
		sb.WriteString("</span>\n")
	}
	sb.WriteString("</div>\n")

	// Error message
	sb.WriteString("<div class=\"error-message\">")
	sb.WriteString(html.EscapeString(displayMessage))
	sb.WriteString("</div>\n")

	// Hint (if any)
	if hint != "" {
		sb.WriteString("<div class=\"error-hint\">💡 ")
		sb.WriteString(html.EscapeString(hint))
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
	// "/path/file.pars:12: message"
	// "/path/file.pars:12:5: message"
	// "script error in /path/file.pars: message"

	// Pattern: "error[s] in [module] <path>: <message>"
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
