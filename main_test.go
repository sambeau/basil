package main

import (
	"bytes"
	"context"
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
