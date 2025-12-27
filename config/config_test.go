package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCORSConfig_Validation_WildcardWithCredentials(t *testing.T) {
	yamlData := `
cors:
  origins: "*"
  credentials: true
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Should fail validation
	if err := validateBasic(cfg); err == nil {
		t.Error("Expected validation error for wildcard origin with credentials")
	} else if err.Error() != "configuration errors:\n  - cors: cannot use origins '*' with credentials true (browsers reject this)" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCORSConfig_Validation_SpecificOriginWithCredentials(t *testing.T) {
	yamlData := `
cors:
  origins: https://example.com
  credentials: true
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Should pass validation
	if err := validateBasic(cfg); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestStringOrSlice_SingleString(t *testing.T) {
	yamlData := `origins: "https://example.com"`

	var config struct {
		Origins StringOrSlice `yaml:"origins"`
	}

	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(config.Origins) != 1 {
		t.Errorf("Expected 1 origin, got %d", len(config.Origins))
	}
	if config.Origins[0] != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", config.Origins[0])
	}
}

func TestStringOrSlice_MultipleStrings(t *testing.T) {
	yamlData := `
origins:
  - https://example.com
  - https://app.example.com
`
	var config struct {
		Origins StringOrSlice `yaml:"origins"`
	}

	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(config.Origins) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(config.Origins))
	}
	if config.Origins[0] != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", config.Origins[0])
	}
	if config.Origins[1] != "https://app.example.com" {
		t.Errorf("Expected https://app.example.com, got %s", config.Origins[1])
	}
}

func TestStringOrSlice_Contains(t *testing.T) {
	s := StringOrSlice{"https://example.com", "https://app.example.com"}

	if !s.Contains("https://example.com") {
		t.Error("Expected Contains to return true for existing item")
	}
	if s.Contains("https://other.com") {
		t.Error("Expected Contains to return false for non-existing item")
	}
}

func TestCORSConfig_Defaults(t *testing.T) {
	cfg := Defaults()

	// CORS should be empty by default (disabled)
	if len(cfg.CORS.Origins) != 0 {
		t.Errorf("Expected no origins by default, got %d", len(cfg.CORS.Origins))
	}

	// Default methods
	if len(cfg.CORS.Methods) != 3 {
		t.Errorf("Expected 3 default methods, got %d", len(cfg.CORS.Methods))
	}

	// Default maxAge
	if cfg.CORS.MaxAge != 86400 {
		t.Errorf("Expected default maxAge 86400, got %d", cfg.CORS.MaxAge)
	}
}

func TestCORSConfig_Parse(t *testing.T) {
	yamlData := `
cors:
  origins:
    - https://example.com
    - https://app.example.com
  methods: [GET, POST, DELETE]
  headers: [Content-Type, Authorization]
  expose: [X-Total-Count]
  credentials: true
  maxAge: 3600
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(cfg.CORS.Origins) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(cfg.CORS.Origins))
	}
	if len(cfg.CORS.Methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(cfg.CORS.Methods))
	}
	if len(cfg.CORS.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(cfg.CORS.Headers))
	}
	if len(cfg.CORS.Expose) != 1 {
		t.Errorf("Expected 1 expose header, got %d", len(cfg.CORS.Expose))
	}
	if !cfg.CORS.Credentials {
		t.Error("Expected credentials to be true")
	}
	if cfg.CORS.MaxAge != 3600 {
		t.Errorf("Expected maxAge 3600, got %d", cfg.CORS.MaxAge)
	}
}

func TestProtectedPath_SimpleStrings(t *testing.T) {
	yamlData := `
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - /admin
    - /settings
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(cfg.Auth.ProtectedPaths) != 3 {
		t.Fatalf("Expected 3 protected paths, got %d", len(cfg.Auth.ProtectedPaths))
	}

	expected := []string{"/dashboard", "/admin", "/settings"}
	for i, exp := range expected {
		if cfg.Auth.ProtectedPaths[i].Path != exp {
			t.Errorf("Protected path %d: expected %q, got %q", i, exp, cfg.Auth.ProtectedPaths[i].Path)
		}
		if len(cfg.Auth.ProtectedPaths[i].Roles) != 0 {
			t.Errorf("Protected path %d: expected no roles, got %v", i, cfg.Auth.ProtectedPaths[i].Roles)
		}
	}
}

func TestProtectedPath_WithRoles(t *testing.T) {
	yamlData := `
auth:
  enabled: true
  protected_paths:
    - path: /admin
      roles: [admin]
    - path: /editors
      roles: [admin, editor]
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(cfg.Auth.ProtectedPaths) != 2 {
		t.Fatalf("Expected 2 protected paths, got %d", len(cfg.Auth.ProtectedPaths))
	}

	// First path: /admin with admin role
	if cfg.Auth.ProtectedPaths[0].Path != "/admin" {
		t.Errorf("Expected path /admin, got %q", cfg.Auth.ProtectedPaths[0].Path)
	}
	if len(cfg.Auth.ProtectedPaths[0].Roles) != 1 || cfg.Auth.ProtectedPaths[0].Roles[0] != "admin" {
		t.Errorf("Expected roles [admin], got %v", cfg.Auth.ProtectedPaths[0].Roles)
	}

	// Second path: /editors with admin and editor roles
	if cfg.Auth.ProtectedPaths[1].Path != "/editors" {
		t.Errorf("Expected path /editors, got %q", cfg.Auth.ProtectedPaths[1].Path)
	}
	if len(cfg.Auth.ProtectedPaths[1].Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(cfg.Auth.ProtectedPaths[1].Roles))
	}
}

func TestProtectedPath_Mixed(t *testing.T) {
	yamlData := `
auth:
  enabled: true
  protected_paths:
    - /dashboard
    - path: /admin
      roles: [admin]
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(cfg.Auth.ProtectedPaths) != 2 {
		t.Fatalf("Expected 2 protected paths, got %d", len(cfg.Auth.ProtectedPaths))
	}

	// First: simple string
	if cfg.Auth.ProtectedPaths[0].Path != "/dashboard" {
		t.Errorf("Expected path /dashboard, got %q", cfg.Auth.ProtectedPaths[0].Path)
	}
	if len(cfg.Auth.ProtectedPaths[0].Roles) != 0 {
		t.Errorf("Expected no roles for /dashboard, got %v", cfg.Auth.ProtectedPaths[0].Roles)
	}

	// Second: object with roles
	if cfg.Auth.ProtectedPaths[1].Path != "/admin" {
		t.Errorf("Expected path /admin, got %q", cfg.Auth.ProtectedPaths[1].Path)
	}
	if len(cfg.Auth.ProtectedPaths[1].Roles) != 1 {
		t.Errorf("Expected 1 role for /admin, got %d", len(cfg.Auth.ProtectedPaths[1].Roles))
	}
}

func TestProtectedPath_LoginPath(t *testing.T) {
	yamlData := `
auth:
  enabled: true
  login_path: /auth/signin
`
	cfg := Defaults()
	if err := yaml.Unmarshal([]byte(yamlData), cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if cfg.Auth.LoginPath != "/auth/signin" {
		t.Errorf("Expected login_path /auth/signin, got %q", cfg.Auth.LoginPath)
	}
}
