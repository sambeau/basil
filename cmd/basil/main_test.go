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
	requireGit(t)
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "myapp")

	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{
		"--init", projectPath,
		"--server",
		"--host", "myapp.example.com",
		"--admin", "sam",
	}, &stdout, &stderr, os.Getenv)
	if err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Check success message
	output := stdout.String()
	if !strings.Contains(output, "Created Basil site") {
		t.Error("success message not printed")
	}
	if !strings.Contains(output, "Start the server:") {
		t.Error("missing start instructions")
	}

	// Verify the site-root layout exists
	for _, p := range []string{
		filepath.Join(projectPath, "site.git"),
		filepath.Join(projectPath, "releases"),
		filepath.Join(projectPath, "data"),
		filepath.Join(projectPath, "current", "basil.yaml"),
		filepath.Join(projectPath, "current", "site", "index.pars"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s not created", p)
		}
	}
}

// --host and --admin are only meaningful with --init.
func TestCLI_HostFlagWithoutInit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"--host", "example.com"}, &stdout, &stderr, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected --host without --init to be refused")
	}
	if !strings.Contains(err.Error(), "--init") {
		t.Errorf("the error does not explain the flag: %v", err)
	}
}
