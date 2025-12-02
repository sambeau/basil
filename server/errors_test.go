package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/config"
)

func TestRenderDevErrorPage_ParseError(t *testing.T) {
	devErr := &DevError{
		Type:    "parse",
		File:    "/path/to/test.pars",
		Line:    10,
		Column:  5,
		Message: "unexpected token '}'",
	}

	w := httptest.NewRecorder()
	renderDevErrorPage(w, devErr)

	resp := w.Result()
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	body := w.Body.String()

	// Check content type
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	// Check error type badge
	if !strings.Contains(body, "parse error") {
		t.Error("expected 'parse error' in output")
	}

	// Check file path
	if !strings.Contains(body, "/path/to/test.pars") {
		t.Error("expected file path in output")
	}

	// Check line number
	if !strings.Contains(body, ">10<") {
		t.Error("expected line number 10 in output")
	}

	// Check error message
	if !strings.Contains(body, "unexpected token") {
		t.Error("expected error message in output")
	}

	// Note: live reload script is injected by middleware, not renderDevErrorPage directly
}

func TestRenderDevErrorPage_RuntimeError(t *testing.T) {
	devErr := &DevError{
		Type:    "runtime",
		File:    "/app/handler.pars",
		Line:    25,
		Message: "dictionary destructuring requires a dictionary value, got BUILTIN",
	}

	w := httptest.NewRecorder()
	renderDevErrorPage(w, devErr)

	body := w.Body.String()

	if !strings.Contains(body, "runtime error") {
		t.Error("expected 'runtime error' in output")
	}

	if !strings.Contains(body, "dictionary destructuring") {
		t.Error("expected error message in output")
	}
}

func TestRenderDevErrorPage_FileNotFound(t *testing.T) {
	devErr := &DevError{
		Type:    "file",
		File:    "/missing/handler.pars",
		Message: "no such file or directory",
	}

	w := httptest.NewRecorder()
	renderDevErrorPage(w, devErr)

	body := w.Body.String()

	if !strings.Contains(body, "file error") {
		t.Error("expected 'file error' in output")
	}

	if !strings.Contains(body, "/missing/handler.pars") {
		t.Error("expected file path in output")
	}
}

func TestGetSourceContext_MiddleOfFile(t *testing.T) {
	// Create a temp file with test content
	content := ""
	for i := 1; i <= 20; i++ {
		content += "line " + strings.Repeat("x", i) + "\n"
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.pars")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	lines := getSourceContext(tmpFile, 10, 3)

	if len(lines) != 7 { // 3 before + error line + 3 after
		t.Errorf("expected 7 lines, got %d", len(lines))
	}

	// Check line numbers
	expectedStart := 7
	for i, line := range lines {
		expectedNum := expectedStart + i
		if line.Number != expectedNum {
			t.Errorf("line %d: expected number %d, got %d", i, expectedNum, line.Number)
		}
		if (line.Number == 10) != line.IsError {
			t.Errorf("line %d: IsError mismatch", line.Number)
		}
	}
}

func TestGetSourceContext_StartOfFile(t *testing.T) {
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.pars")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	lines := getSourceContext(tmpFile, 2, 3)

	// Should start at line 1 (can't go before)
	if lines[0].Number != 1 {
		t.Errorf("expected first line to be 1, got %d", lines[0].Number)
	}

	// Error line should be marked
	var errorLineFound bool
	for _, line := range lines {
		if line.Number == 2 && line.IsError {
			errorLineFound = true
		}
	}
	if !errorLineFound {
		t.Error("error line not marked correctly")
	}
}

func TestGetSourceContext_EndOfFile(t *testing.T) {
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.pars")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	lines := getSourceContext(tmpFile, 5, 3)

	// Should include line 5 (the error line)
	var lastLine SourceLine
	for _, line := range lines {
		lastLine = line
	}
	if lastLine.Number != 5 {
		t.Errorf("expected last line to be 5, got %d", lastLine.Number)
	}
}

func TestGetSourceContext_FileNotFound(t *testing.T) {
	lines := getSourceContext("/nonexistent/file.pars", 10, 3)

	if lines != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestHighlightParsley_Keywords(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"let x = 1", `<span class="kw">let</span>`},
		{"fn foo() {}", `<span class="kw">fn</span>`},
		{"if true else false", `<span class="kw">if</span>`},
		{"for i in list", `<span class="kw">for</span>`},
		{"export name", `<span class="kw">export</span>`},
	}

	for _, tc := range tests {
		result := highlightParsley(tc.input)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("input %q: expected to contain %q, got %q", tc.input, tc.contains, result)
		}
	}
}

func TestHighlightParsley_Strings(t *testing.T) {
	result := highlightParsley(`let name = "hello"`)
	// Should contain string class for the quoted portion
	if !strings.Contains(result, `class="str"`) {
		t.Errorf("expected string highlighting class, got %q", result)
	}
	// Should also have the keyword
	if !strings.Contains(result, `class="kw"`) {
		t.Errorf("expected keyword highlighting, got %q", result)
	}
}

func TestHighlightParsley_Numbers(t *testing.T) {
	result := highlightParsley("let x = 42")
	if !strings.Contains(result, `class="num"`) {
		t.Errorf("expected number highlighting, got %q", result)
	}
}

func TestHighlightParsley_Comments(t *testing.T) {
	result := highlightParsley("// this is a comment")
	if !strings.Contains(result, `class="comment"`) {
		t.Errorf("expected comment highlighting, got %q", result)
	}
}

func TestHighlightParsley_HTMLEscape(t *testing.T) {
	// The < and > should be escaped to &lt; and &gt;
	result := highlightParsley("<div>test</div>")
	if strings.Contains(result, "<div>") && !strings.Contains(result, "&lt;") {
		t.Errorf("expected HTML to be escaped, got %q", result)
	}
}

func TestHighlightParsley_QuotesReadable(t *testing.T) {
	// Quotes should appear as " in the output, not as &#34;
	result := highlightParsley(`let name = "hello"`)

	// Should NOT contain &#34;
	if strings.Contains(result, "&#34;") {
		t.Errorf("quotes should be readable as \", not &#34;, got %q", result)
	}

	// Should contain actual quote marks (escaped properly for HTML attribute context is fine)
	// The string "hello" should be visible
	if !strings.Contains(result, `"hello"`) {
		t.Errorf("expected readable string with quotes, got %q", result)
	}
}

func TestHighlightParsley_HTMLTagAttributes(t *testing.T) {
	// HTML tags with attributes should be readable
	result := highlightParsley(`<img src="/logo.png" alt="Logo"/>`)

	// Should NOT contain &#34;
	if strings.Contains(result, "&#34;") {
		t.Errorf("quotes in tag attributes should be readable, got %q", result)
	}

	// Should contain the tag
	if !strings.Contains(result, `class="tag"`) {
		t.Errorf("expected tag highlighting, got %q", result)
	}
}

func TestExtractLineInfo_ParseError(t *testing.T) {
	msg := "parse error in /app/test.pars: unexpected token"
	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "/app/test.pars" {
		t.Errorf("expected file '/app/test.pars', got %q", file)
	}
	if cleanMsg != "unexpected token" {
		t.Errorf("expected clean message 'unexpected token', got %q", cleanMsg)
	}
	// Line/col may be 0 if not in message
	_ = line
	_ = col
}

func TestExtractLineInfo_WithLineNumber(t *testing.T) {
	msg := "error at line 42: something went wrong"
	_, line, _, _ := extractLineInfo(msg)

	if line != 42 {
		t.Errorf("expected line 42, got %d", line)
	}
}

func TestExtractLineInfo_ScriptError(t *testing.T) {
	msg := "script error in /path/to/handler.pars: not a function: DICTIONARY"
	file, _, _, cleanMsg := extractLineInfo(msg)

	if file != "/path/to/handler.pars" {
		t.Errorf("expected file path, got %q", file)
	}
	if cleanMsg != "not a function: DICTIONARY" {
		t.Errorf("expected clean message, got %q", cleanMsg)
	}
}

func TestExtractLineInfo_ModuleParseErrors(t *testing.T) {
	// Test the multi-line parse errors format with "module" prefix
	msg := `parse errors in module ./app/pages/home.pars:
  expected identifier as dictionary key, got opening tag at line 6, column 3
  line 31, column 7: unexpected 'Page'`

	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "./app/pages/home.pars" {
		t.Errorf("expected file './app/pages/home.pars', got %q", file)
	}
	if line != 6 {
		t.Errorf("expected line 6, got %d", line)
	}
	if col != 3 {
		t.Errorf("expected column 3, got %d", col)
	}
	// Clean message should have the error details (trimmed of leading whitespace)
	if !strings.Contains(cleanMsg, "expected identifier") {
		t.Errorf("expected clean message to contain error, got %q", cleanMsg)
	}
}

func TestHandleScriptError_DevMode(t *testing.T) {
	// Create a mock handler with dev mode enabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/test/handler.pars",
	}

	w := httptest.NewRecorder()
	h.handleScriptError(w, "runtime", "/test/handler.pars", "test error message")

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML (dev error page)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html, got %s", ct)
	}

	// Should contain the error message
	if !strings.Contains(body, "test error message") {
		t.Error("expected error message in body")
	}

	// Note: live reload script is injected by middleware, not tested here
}

func TestHandleScriptError_ProdMode(t *testing.T) {
	// Create a mock handler with dev mode disabled
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: false,
		},
	}
	s := &Server{config: cfg}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/test/handler.pars",
	}

	w := httptest.NewRecorder()
	h.handleScriptError(w, "runtime", "/test/handler.pars", "test error message")

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be plain text (generic error)
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/html") {
		t.Errorf("expected plain text in prod mode, got %s", ct)
	}

	// Should NOT contain detailed error info
	if strings.Contains(body, "test error message") {
		t.Error("should not expose error details in production")
	}

	// Should be generic error
	if !strings.Contains(body, "Internal Server Error") {
		t.Error("expected generic error message")
	}
}
