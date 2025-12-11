package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--version"}, stdout, stderr, func(s string) string { return "" })

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "basil version") {
		t.Errorf("expected version output, got %q", output)
	}
}

func TestRunHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--help"}, stdout, stderr, func(s string) string { return "" })

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "basil - A web server for Parsley") {
		t.Errorf("expected help output, got %q", output)
	}
	if !strings.Contains(output, "--config") {
		t.Errorf("expected --config in help, got %q", output)
	}
	if !strings.Contains(output, "--dev") {
		t.Errorf("expected --dev in help, got %q", output)
	}
}

func TestRunInvalidFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--invalid-flag"}, stdout, stderr, func(s string) string { return "" })

	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunMissingConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := run(context.Background(), []string{"--config", "/nonexistent/config.yaml"}, stdout, stderr, func(s string) string { return "" })

	if err == nil {
		t.Error("expected error for missing config")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("expected 'config file not found' error, got %q", err.Error())
	}
}

func TestCLI_InitCommand(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "myapp")

	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"--init", projectPath}, &stdout, &stderr, os.Getenv)
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Check success message
	output := stdout.String()
	if !strings.Contains(output, "Created new Basil project") {
		t.Error("success message not printed")
	}
	if !strings.Contains(output, "Get started:") {
		t.Error("missing 'Get started' instructions")
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(projectPath, "basil.yaml")); err != nil {
		t.Error("basil.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "site", "index.pars")); err != nil {
		t.Error("site/index.pars not created")
	}
	if _, err := os.Stat(filepath.Join(projectPath, "public")); err != nil {
		t.Error("public/ directory not created")
	}
}
