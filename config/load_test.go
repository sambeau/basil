package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Server.Port != 443 {
		t.Errorf("expected default port 443, got %d", cfg.Server.Port)
	}
	if cfg.Server.HTTPS.Auto != true {
		t.Error("expected default https.auto to be true")
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level 'info', got %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("expected default log format 'text', got %q", cfg.Logging.Format)
	}
}

func TestInterpolateEnv(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "TEST_HOST":
			return "example.com"
		case "TEST_PORT":
			return "9000"
		default:
			return ""
		}
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple substitution",
			input:    "host: ${TEST_HOST}",
			expected: "host: example.com",
		},
		{
			name:     "with default (env set)",
			input:    "host: ${TEST_HOST:-localhost}",
			expected: "host: example.com",
		},
		{
			name:     "with default (env not set)",
			input:    "host: ${UNSET_VAR:-localhost}",
			expected: "host: localhost",
		},
		{
			name:     "multiple substitutions",
			input:    "addr: ${TEST_HOST}:${TEST_PORT}",
			expected: "addr: example.com:9000",
		},
		{
			name:     "no substitution needed",
			input:    "static: value",
			expected: "static: value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(interpolateEnv([]byte(tt.input), getenv))
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "basil.yaml")

	configContent := `
server:
  host: localhost
  port: 8080

logging:
  level: debug
  format: json
  output: stderr

static:
  - path: /assets/
    root: ./public

routes:
  - path: /
    handler: index.parsley
  - path: /api/*
    handler: api.parsley
    auth: required
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath, os.Getenv)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify server config
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected host 'localhost', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	// Verify logging config
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level 'debug', got %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("expected log format 'json', got %q", cfg.Logging.Format)
	}

	// Verify static routes - paths are now resolved to absolute
	if len(cfg.Static) != 1 {
		t.Fatalf("expected 1 static route, got %d", len(cfg.Static))
	}
	if cfg.Static[0].Path != "/assets/" {
		t.Errorf("expected static path '/assets/', got %q", cfg.Static[0].Path)
	}
	expectedStaticRoot := filepath.Join(dir, "public")
	if cfg.Static[0].Root != expectedStaticRoot {
		t.Errorf("expected static root %q, got %q", expectedStaticRoot, cfg.Static[0].Root)
	}

	// Verify dynamic routes - handlers are now resolved to absolute
	if len(cfg.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(cfg.Routes))
	}
	expectedHandler := filepath.Join(dir, "index.parsley")
	if cfg.Routes[0].Handler != expectedHandler {
		t.Errorf("expected handler %q, got %q", expectedHandler, cfg.Routes[0].Handler)
	}
	if cfg.Routes[1].Auth != "required" {
		t.Errorf("expected auth 'required', got %q", cfg.Routes[1].Auth)
	}
}

func TestLoadWithEnvInterpolation(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "basil.yaml")

	configContent := `
server:
  host: ${BASIL_HOST:-localhost}
  port: 8080

logging:
  level: info
  format: text
  output: stderr
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Test with env var set
	getenv := func(key string) string {
		if key == "BASIL_HOST" {
			return "production.example.com"
		}
		return ""
	}

	cfg, err := Load(configPath, getenv)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "production.example.com" {
		t.Errorf("expected host 'production.example.com', got %q", cfg.Server.Host)
	}

	// Test with env var not set (should use default)
	getenvEmpty := func(key string) string { return "" }
	cfg, err = Load(configPath, getenvEmpty)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("expected host 'localhost' (default), got %q", cfg.Server.Host)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		expectErr bool
		errSubstr string
	}{
		{
			name: "valid minimal config (dev mode)",
			config: `
server:
  host: localhost
  port: 8080
logging:
  level: info
  format: text
  output: stderr
`,
			expectErr: false,
		},
		{
			name: "invalid port",
			config: `
server:
  port: 99999
logging:
  level: info
  format: text
  output: stderr
`,
			expectErr: true,
			errSubstr: "invalid port",
		},
		{
			name: "invalid log level",
			config: `
server:
  port: 8080
logging:
  level: verbose
  format: text
  output: stderr
`,
			expectErr: true,
			errSubstr: "invalid log level",
		},
		{
			name: "invalid log format",
			config: `
server:
  port: 8080
logging:
  level: info
  format: xml
  output: stderr
`,
			expectErr: true,
			errSubstr: "invalid log format",
		},
		{
			name: "static route without root or file",
			config: `
server:
  port: 8080
logging:
  level: info
  format: text
  output: stderr
static:
  - path: /assets/
`,
			expectErr: true,
			errSubstr: "either root or file is required",
		},
		{
			name: "route without handler",
			config: `
server:
  port: 8080
logging:
  level: info
  format: text
  output: stderr
routes:
  - path: /
`,
			expectErr: true,
			errSubstr: "handler is required",
		},
		{
			name: "invalid auth value",
			config: `
server:
  port: 8080
logging:
  level: info
  format: text
  output: stderr
routes:
  - path: /
    handler: test.parsley
    auth: mandatory
`,
			expectErr: true,
			errSubstr: "auth must be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "basil.yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Load performs basic validation
			_, err := Load(configPath, os.Getenv)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveConfigPath(t *testing.T) {
	// Test explicit path not found
	_, err := resolveConfigPath("/nonexistent/path/basil.yaml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}

	// Test explicit path found
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	resolved, err := resolveConfigPath(configPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resolved != configPath {
		t.Errorf("expected %q, got %q", configPath, resolved)
	}
}

func TestWarnings(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		wantWarn string
	}{
		{
			name:     "no routes",
			cfg:      &Config{},
			wantWarn: "no routes configured",
		},
		{
			name: "has routes",
			cfg: &Config{
				Routes: []Route{{Path: "/", Handler: "./app.pars"}},
			},
			wantWarn: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := Warnings(tt.cfg)
			if tt.wantWarn == "" {
				if len(warnings) > 0 {
					t.Errorf("expected no warnings, got %v", warnings)
				}
			} else {
				found := false
				for _, w := range warnings {
					if contains(w, tt.wantWarn) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got %v", tt.wantWarn, warnings)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"", 0, false},
		{"1024", 1024, false},
		{"1B", 1, false},
		{"1KB", 1024, false},
		{"1kb", 1024, false},
		{"10KB", 10 * 1024, false},
		{"1MB", 1024 * 1024, false},
		{"10MB", 10 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"  5MB  ", 5 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"MB", 0, true},   // No number
		{"abc", 0, true},  // Not a number
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDevConfigDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Dev.LogMaxSize != "10MB" {
		t.Errorf("expected default log_max_size '10MB', got %q", cfg.Dev.LogMaxSize)
	}
	if cfg.Dev.LogTruncatePct != 25 {
		t.Errorf("expected default log_truncate_pct 25, got %d", cfg.Dev.LogTruncatePct)
	}
}
