package server

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
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
		Message: "destructuring requires a dictionary or record, got BUILTIN",
	}

	w := httptest.NewRecorder()
	renderDevErrorPage(w, devErr)

	body := w.Body.String()

	if !strings.Contains(body, "runtime error") {
		t.Error("expected 'runtime error' in output")
	}

	if !strings.Contains(body, "destructuring requires") {
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

func TestExtractLineInfo_ModuleRuntimeError(t *testing.T) {
	// Test runtime error from a module - the format from evaluator when a module has an error
	// This is the format: "in module <path>: line X, column Y: <message>"
	msg := "in module ./app/pages/scouts.pars: line 18, column 20: dot notation can only be used on dictionaries, got BUILTIN"

	file, line, col, cleanMsg := extractLineInfo(msg)

	if file != "./app/pages/scouts.pars" {
		t.Errorf("expected file './app/pages/scouts.pars', got %q", file)
	}
	if line != 18 {
		t.Errorf("expected line 18, got %d", line)
	}
	if col != 20 {
		t.Errorf("expected column 20, got %d", col)
	}
	// Clean message should be stripped of the module prefix and line/col info
	if strings.Contains(cleanMsg, "in module") {
		t.Errorf("clean message should not contain 'in module', got %q", cleanMsg)
	}
}

func TestHandleScriptError_DevMode(t *testing.T) {
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
	req := httptest.NewRequest("GET", "/test", nil)
	h.handleScriptError(w, req, "runtime", "/test/handler.pars", "test error message")

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

func TestHandleScriptErrorWithLocation_ModuleError(t *testing.T) {
	// Test that module errors show the correct file path, not the parent handler path
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, configPath: "/app/basil.yaml"}
	h := &parsleyHandler{
		server:     s,
		scriptPath: "/app/app.pars", // The parent handler
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// Simulate error message from a module - this is the format from evaluator
	moduleErrMsg := "in module ./app/pages/scouts.pars: line 18, column 20: dot notation can only be used on dictionaries, got BUILTIN"
	h.handleScriptErrorWithLocation(w, req, "runtime", h.scriptPath, moduleErrMsg, 0, 0)

	body := w.Body.String()

	// The error page should show the MODULE path, not the parent handler path
	if strings.Contains(body, "app.pars") && !strings.Contains(body, "scouts.pars") {
		t.Error("error page should show module path (scouts.pars), not parent handler path (app.pars)")
	}

	// Should contain the correct file path
	if !strings.Contains(body, "scouts.pars") {
		t.Error("expected error page to show scouts.pars")
	}

	// Should show line 18
	if !strings.Contains(body, ":18") && !strings.Contains(body, "18") {
		t.Error("expected error page to show line 18")
	}
}

func TestHandleScriptErrorWithLocation_ModuleNotFound(t *testing.T) {
	// Test module-not-found errors show correct module file (no line info available)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg, configPath: "/app/basil.yaml"}
	h := &parsleyHandler{
		server: s,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	moduleErrMsg := "in module ./app/pages/scouts.pars: module not found: ./app/pages/std/table"
	h.handleScriptErrorWithLocation(w, req, "runtime", h.scriptPath, moduleErrMsg, 0, 0)

	body := w.Body.String()

	// Should show the module where the import failed (scouts.pars), not the parent
	if !strings.Contains(body, "scouts.pars") {
		t.Errorf("expected error page to show scouts.pars, body contains: %s", body[:min(500, len(body))])
	}

	// Should NOT show app.pars as the primary file
	// (it might appear in the message but not in the file-path span)
	if strings.Contains(body, `class="file-path">./app/app.pars`) {
		t.Error("error page should not show app.pars as primary file path")
	}
}

func TestImproveErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		wantImproved string
		wantHint     string
	}{
		{
			name:         "missing parentheses",
			message:      "expected '(', got 'x'",
			wantImproved: "Missing parentheses",
			wantHint:     "Function parameters need parentheses: fn(x) { ... } or fn(a, b) { ... }",
		},
		{
			name:         "python comment",
			message:      "unexpected '#'",
			wantImproved: "Invalid comment syntax",
			wantHint:     "Use // for comments, not #. Parsley uses C-style comments: // single line or /* multi-line */",
		},
		{
			name:         "console.log",
			message:      "identifier not found: console",
			wantImproved: "'console' is not defined",
			wantHint:     "Use log() for debugging output. Example: log(\"value:\", myVar)",
		},
		{
			name:         "print function",
			message:      "identifier not found: print",
			wantImproved: "'print' is not defined",
			wantHint:     "Use log() for output. Example: log(\"hello world\")",
		},
		{
			name:         "document DOM",
			message:      "identifier not found: document",
			wantImproved: "'document' is not defined",
			wantHint:     "Parsley runs on the server, not in the browser. DOM APIs are not available.",
		},
		{
			name:         "window browser global",
			message:      "identifier not found: window",
			wantImproved: "'window' is not defined",
			wantHint:     "Parsley runs on the server, not in the browser. Browser globals are not available.",
		},
		{
			name:         "require Node.js",
			message:      "identifier not found: require",
			wantImproved: "'require' is not defined",
			wantHint:     "Use 'import' to load modules. Example: import utils from \"./utils.pars\"",
		},
		{
			name:         "unrecognized component tag",
			message:      "expected identifier as dictionary key, got opening tag",
			wantImproved: "Unrecognized component tag",
			wantHint:     "Is the component imported? Check that the import path is correct and the component is exported.",
		},
		{
			name:         "unexpected uppercase (undefined component)",
			message:      "unexpected 'MyComponent'",
			wantImproved: "'MyComponent' is not defined",
			wantHint:     "Did you forget to import MyComponent? Component names must start with an uppercase letter.",
		},
		{
			name:         "no improvement needed",
			message:      "some other error",
			wantImproved: "some other error",
			wantHint:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImproved, gotHint := improveErrorMessage(tt.message)
			if gotImproved != tt.wantImproved {
				t.Errorf("improved = %q, want %q", gotImproved, tt.wantImproved)
			}
			if gotHint != tt.wantHint {
				t.Errorf("hint = %q, want %q", gotHint, tt.wantHint)
			}
		})
	}
}
func TestHandleScriptError_ProdMode(t *testing.T) {
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
	req := httptest.NewRequest("GET", "/test", nil)
	h.handleScriptError(w, req, "runtime", "/test/handler.pars", "test error message")

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML (prelude error page)
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML in prod mode, got %s", ct)
	}

	// Should contain 500 error message
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}

	// Should NOT contain detailed error info (test error message shouldn't appear)
	if strings.Contains(body, "test error message") {
		t.Error("should not expose detailed error details in production")
	}
}

func TestCreateErrorEnv(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	req := httptest.NewRequest("GET", "/test/path?foo=bar", nil)
	err := fmt.Errorf("test error")

	env := s.createErrorEnv(req, 404, err)

	// Check that error object was set
	errorObj, ok := env.Get("error")
	if !ok {
		t.Fatal("expected 'error' to be set in environment")
	}
	if errorObj == nil {
		t.Fatal("error object should not be nil")
	}

	// Check that basil object was set on BasilCtx
	if env.BasilCtx == nil {
		t.Fatal("expected BasilCtx to be set in environment")
	}
}

func TestRenderPreludeError_404(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)
	err := fmt.Errorf("not found")

	success := s.renderPreludeError(w, req, 404, err)

	if !success {
		t.Fatal("expected renderPreludeError to succeed")
	}

	resp := w.Result()
	body := w.Body.String()

	// Should return 404
	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 404 error page content
	if !strings.Contains(body, "404") {
		t.Errorf("expected body to contain '404', got: %s", body)
	}
	if !strings.Contains(body, "not found") {
		t.Errorf("expected body to contain 'not found', got: %s", body)
	}
}

func TestRenderPreludeError_500(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	err := fmt.Errorf("server error")

	success := s.renderPreludeError(w, req, 500, err)

	if !success {
		t.Fatal("expected renderPreludeError to succeed")
	}

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 500 error page content
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
	if !strings.Contains(body, "Internal Server Error") {
		t.Error("expected body to contain 'Internal Server Error'")
	}
}

func TestHandle404(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)

	s.handle404(w, req)

	resp := w.Result()
	body := w.Body.String()

	// Should return 404
	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 404 content
	if !strings.Contains(body, "404") {
		t.Error("expected body to contain '404'")
	}
}

func TestHandle500(t *testing.T) {
	// Initialize prelude before running tests
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Dev: true,
		},
	}
	s := &Server{config: cfg}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	err := fmt.Errorf("test error")

	s.handle500(w, req, err)

	resp := w.Result()
	body := w.Body.String()

	// Should return 500
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	// Should be HTML
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %s", ct)
	}

	// Should contain 500 content
	if !strings.Contains(body, "500") {
		t.Error("expected body to contain '500'")
	}
}
