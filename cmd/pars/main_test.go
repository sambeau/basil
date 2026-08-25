package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestEvaluateInlinePLN tests that -e outputs PLN representation by default
func TestEvaluateInlinePLN(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "number",
			code:     "1 + 2",
			expected: "3\n",
		},
		{
			name:     "string",
			code:     `"hello"`,
			expected: `"hello"` + "\n",
		},
		{
			name:     "array",
			code:     "[1, 2, 3]",
			expected: "[1, 2, 3]\n",
		},
		{
			name:     "dictionary",
			code:     "{a: 1}",
			expected: "{a: 1}\n",
		},
		{
			name:     "regex match",
			code:     `"hi" ~ /(\w+)/`,
			expected: `["hi", "hi"]` + "\n",
		},
		{
			name:     "null",
			code:     "null",
			expected: "null\n",
		},
		{
			name:     "boolean true",
			code:     "true",
			expected: "true\n",
		},
		{
			name:     "boolean false",
			code:     "false",
			expected: "false\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./pars", "-e", tt.code)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, output)
			}
			if string(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(output))
			}
		})
	}
}

// TestEvaluateInlineRaw tests that -e --raw outputs raw print string
func TestEvaluateInlineRaw(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		flag     string
		expected string
	}{
		{
			name:     "raw string with --raw",
			code:     `"hello"`,
			flag:     "--raw",
			expected: "hello\n",
		},
		{
			name:     "raw string with -r",
			code:     `"hello"`,
			flag:     "-r",
			expected: "hello\n",
		},
		{
			name:     "raw array",
			code:     "[1,2,3]",
			flag:     "--raw",
			expected: "123\n",
		},
		{
			name:     "raw HTML",
			code:     `"<b>hi</b>"`,
			flag:     "-r",
			expected: "<b>hi</b>\n",
		},
		{
			name:     "raw null (no output)",
			code:     "null",
			flag:     "--raw",
			expected: "",
		},
		{
			name:     "raw dictionary",
			code:     "{a: 1, b: 2}",
			flag:     "-r",
			expected: "{a: 1, b: 2}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./pars", "-e", tt.code, tt.flag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, output)
			}
			if string(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(output))
			}
		})
	}
}

// TestEvaluateInlineRawPrettyPrint tests --raw with -pp for HTML formatting
func TestEvaluateInlineRawPrettyPrint(t *testing.T) {
	// Build the binary first to ensure it's up to date
	buildCmd := exec.Command("go", "build", "-o", "pars", "./cmd/pars")
	buildCmd.Dir = "../.."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build pars: %v", err)
	}

	code := `"<div><span>x</span></div>"`
	cmd := exec.Command("./pars", "-e", code, "-r", "-pp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Check that output contains HTML structure and indentation (pretty-printed)
	outputStr := string(output)
	if !strings.Contains(outputStr, "<div>") || !strings.Contains(outputStr, "\n") {
		t.Errorf("Expected pretty-printed HTML with structure, got: %q", outputStr)
	}
}

// TestFormatFileWrite exercises the refactored formatFile (now backed by the
// shared pkg/parsley/format pipeline) via its file-touching -w path: messy
// source is rewritten, a parse error is reported and never mangles the file,
// and already-formatted source is a no-op.
func TestFormatFileWrite(t *testing.T) {
	dir := t.TempDir()

	// Messy source is rewritten in place.
	messy := dir + "/messy.pars"
	if err := os.WriteFile(messy, []byte("let    x    =    5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := formatFile(messy, true, false, false); err != nil {
		t.Fatalf("formatFile(-w) on messy source: %v", err)
	}
	got, _ := os.ReadFile(messy)
	if string(got) != "let x = 5\n" {
		t.Errorf("messy source not reformatted: got %q", got)
	}

	// Re-running is a byte-for-byte no-op.
	if err := formatFile(messy, true, false, false); err != nil {
		t.Fatalf("formatFile(-w) idempotent run: %v", err)
	}
	got2, _ := os.ReadFile(messy)
	if string(got2) != "let x = 5\n" {
		t.Errorf("formatting not idempotent: got %q", got2)
	}

	// A parse error returns an error and leaves the file untouched.
	broken := dir + "/broken.pars"
	const brokenSrc = "let x = = 2\n"
	if err := os.WriteFile(broken, []byte(brokenSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := formatFile(broken, true, false, false); err == nil {
		t.Error("expected a parse error, got nil")
	}
	gotBroken, _ := os.ReadFile(broken)
	if string(gotBroken) != brokenSrc {
		t.Errorf("parse error mangled the file: got %q", gotBroken)
	}
}

// TestMain ensures the binary is built before running tests
func TestMain(m *testing.M) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "pars", ".")
	if err := buildCmd.Run(); err != nil {
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	_ = os.Remove("pars")

	os.Exit(code)
}
