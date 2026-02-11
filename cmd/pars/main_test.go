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
	os.Remove("pars")

	os.Exit(code)
}
