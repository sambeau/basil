package server

import (
	"bufio"
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// DevError holds information about an error to display in dev mode.
type DevError struct {
	Type    string // "parse", "runtime", "file"
	File    string // Full path to the file
	Line    int    // Line number (0 if unknown)
	Column  int    // Column number (0 if unknown)
	Message string // Error message
}

// SourceLine represents a line of source code for display.
type SourceLine struct {
	Number  int
	Content string
	IsError bool
}

// liveReloadScriptForError is the live reload script for error pages.
// Same as the main live reload but works standalone.
const liveReloadScriptForError = `<script>
(function() {
  let lastSeq = 0;
  const pollInterval = 1000;
  
  async function checkForChanges() {
    try {
      const resp = await fetch('/__livereload');
      const data = await resp.json();
      if (lastSeq === 0) {
        lastSeq = data.seq;
      } else if (data.seq !== lastSeq) {
        console.log('[LiveReload] Change detected, reloading...');
        location.reload();
      }
    } catch (e) {
      // Server might be restarting, retry
    }
    setTimeout(checkForChanges, pollInterval);
  }
  
  checkForChanges();
  console.log('[LiveReload] Connected (error page)');
})();
</script>`

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

// renderDevErrorPage generates an HTML error page for dev mode.
func renderDevErrorPage(w http.ResponseWriter, devErr *DevError) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	// Get source context if we have a file and line number
	var sourceLines []SourceLine
	if devErr.File != "" && devErr.Line > 0 {
		sourceLines = getSourceContext(devErr.File, devErr.Line, 5)
	}

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

	if devErr.File != "" {
		sb.WriteString("<span class=\"file-path\">")
		sb.WriteString(html.EscapeString(devErr.File))
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
	sb.WriteString(html.EscapeString(devErr.Message))
	sb.WriteString("</div>\n")

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

	// Live reload script
	sb.WriteString(liveReloadScriptForError)

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

	// HTML tags
	tagPattern = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9]*(?:\s[^>]*)?>|/>`)

	// Comments
	commentPattern = regexp.MustCompile(`//.*$`)

	// Function calls (identifier followed by parenthesis)
	fnCallPattern = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
)

// highlightParsley applies syntax highlighting to a line of Parsley code.
// It returns HTML with span elements for styling.
func highlightParsley(code string) string {
	// First, escape HTML entities
	escaped := html.EscapeString(code)

	// Apply highlighting in order of precedence
	// Use simpler patterns that work with Go's RE2 engine

	// Comments first (they contain everything after //)
	escaped = commentPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		return `<span class="comment">` + m + `</span>`
	})

	// Strings - the escaped quote is &#34; but we need a simpler approach
	// Just match common string patterns after HTML escaping
	stringEscPattern := regexp.MustCompile(`&#34;[^&]*&#34;`)
	escaped = stringEscPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		if strings.Contains(m, `class="`) {
			return m
		}
		return `<span class="str">` + m + `</span>`
	})

	// HTML tags in Parsley source - match &lt;tagname...&gt; patterns
	tagEscPattern := regexp.MustCompile(`&lt;/?[a-zA-Z][a-zA-Z0-9]*[^&]*?(?:&gt;|/&gt;)`)
	escaped = tagEscPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		if strings.Contains(m, `class="`) {
			return m
		}
		return `<span class="tag">` + m + `</span>`
	})

	// Numbers (but not inside already-highlighted spans)
	escaped = numberPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		return `<span class="num">` + m + `</span>`
	})

	// Keywords
	escaped = keywordPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		return `<span class="kw">` + m + `</span>`
	})

	// Function calls
	escaped = fnCallPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		idx := strings.LastIndex(m, "(")
		if idx == -1 {
			return m
		}
		fnName := strings.TrimSpace(m[:idx])
		if keywordPattern.MatchString(fnName) {
			return m
		}
		return `<span class="fn">` + fnName + `</span>(`
	})

	return escaped
}

// extractLineInfo attempts to extract file, line, and column information from an error message.
// Returns the cleaned message without the location prefix.
func extractLineInfo(errMsg string) (file string, line, col int, cleanMsg string) {
	cleanMsg = errMsg

	// Common patterns:
	// "parse error in /path/file.pars: message"
	// "/path/file.pars:12: message"
	// "/path/file.pars:12:5: message"
	// "script error in /path/file.pars: message"

	// Pattern: "error in <path>: <message>"
	if idx := strings.Index(errMsg, " in "); idx != -1 {
		rest := errMsg[idx+4:]
		if colonIdx := strings.Index(rest, ": "); colonIdx != -1 {
			file = rest[:colonIdx]
			cleanMsg = rest[colonIdx+2:]
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

	// Also check for "at line X" pattern in message
	if line == 0 {
		linePattern := regexp.MustCompile(`at line (\d+)`)
		if matches := linePattern.FindStringSubmatch(errMsg); len(matches) > 1 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				line = n
			}
		}
	}

	// Check for "line X, col Y" pattern
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
