package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads configuration from a file with ENV interpolation.
// If configPath is empty, it searches default locations.
// Validation of HTTPS settings is deferred until Validate() is called
// after CLI flags (like --dev) have been applied.
func Load(configPath string, getenv func(string) string) (*Config, error) {
	cfg, _, err := LoadWithPath(configPath, getenv)
	return cfg, err
}

// LoadWithPath reads configuration and returns both the config and the resolved path.
// This is useful when the caller needs to know the actual config file location.
func LoadWithPath(configPath string, getenv func(string) string) (*Config, string, error) {
	path, err := resolveConfigPath(configPath, getenv)
	if err != nil {
		return nil, "", err
	}

	// Get absolute path and directory for resolving relative paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve config path: %w", err)
	}
	baseDir := filepath.Dir(absPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config: %w", err)
	}

	// Interpolate environment variables
	data = interpolateEnv(data, getenv)

	cfg := Defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, "", fmt.Errorf("failed to parse config: %w", err)
	}

	// Track secrets after loading
	if cfg.Session.Secret.IsSecret() {
		cfg.Secrets.MarkSecret("session.secret")
	}

	// Set base directory for resolving relative paths
	cfg.BaseDir = baseDir

	// Resolve relative paths in static routes
	for i := range cfg.Static {
		if cfg.Static[i].Root != "" && !filepath.IsAbs(cfg.Static[i].Root) {
			cfg.Static[i].Root = filepath.Join(baseDir, cfg.Static[i].Root)
		}
		if cfg.Static[i].File != "" && !filepath.IsAbs(cfg.Static[i].File) {
			cfg.Static[i].File = filepath.Join(baseDir, cfg.Static[i].File)
		}
	}

	// Resolve relative paths in routes
	for i := range cfg.Routes {
		if cfg.Routes[i].Handler != "" && !filepath.IsAbs(cfg.Routes[i].Handler) {
			cfg.Routes[i].Handler = filepath.Join(baseDir, cfg.Routes[i].Handler)
		}
		if cfg.Routes[i].PublicDir != "" && !filepath.IsAbs(cfg.Routes[i].PublicDir) {
			cfg.Routes[i].PublicDir = filepath.Join(baseDir, cfg.Routes[i].PublicDir)
		}
	}

	// Apply global public_dir to root route if not specified
	for i := range cfg.Routes {
		if cfg.Routes[i].Path == "/" && cfg.Routes[i].PublicDir == "" && cfg.PublicDir != "" {
			if filepath.IsAbs(cfg.PublicDir) {
				cfg.Routes[i].PublicDir = cfg.PublicDir
			} else {
				cfg.Routes[i].PublicDir = filepath.Join(baseDir, cfg.PublicDir)
			}
		}
	}

	// Resolve relative sqlite path
	if cfg.SQLite != "" && !filepath.IsAbs(cfg.SQLite) {
		cfg.SQLite = filepath.Join(baseDir, cfg.SQLite)
	}

	// Resolve relative site path
	if cfg.Site != "" && !filepath.IsAbs(cfg.Site) {
		cfg.Site = filepath.Join(baseDir, cfg.Site)
	}

	// Resolve relative public_dir path
	if cfg.PublicDir != "" && !filepath.IsAbs(cfg.PublicDir) {
		cfg.PublicDir = filepath.Join(baseDir, cfg.PublicDir)
	}

	// Resolve relative paths in security.allow_write
	for i := range cfg.Security.AllowWrite {
		if !filepath.IsAbs(cfg.Security.AllowWrite[i]) {
			cfg.Security.AllowWrite[i] = filepath.Join(baseDir, cfg.Security.AllowWrite[i])
		}
	}

	// Run non-HTTPS validation only - HTTPS validation deferred until Validate()
	if err := validateBasic(cfg); err != nil {
		return nil, "", err
	}

	return cfg, absPath, nil
}

// Validate performs full configuration validation including HTTPS settings.
// Call this after applying CLI overrides (like --dev).
func Validate(cfg *Config) error {
	if err := validateBasic(cfg); err != nil {
		return err
	}
	return validateHTTPS(cfg)
}

// Warnings returns non-fatal configuration issues that should be reported to the user.
// These are problems that won't prevent the server from starting but likely indicate
// a misconfiguration.
func Warnings(cfg *Config) []string {
	var warnings []string

	// Warn if no routes are configured AND not using site mode
	// Site mode uses filesystem-based routing and doesn't need explicit routes
	if len(cfg.Routes) == 0 && cfg.Site == "" {
		warnings = append(warnings, "no routes configured - the server will return 404 for all requests")
	}

	// Email verification warnings
	if cfg.Auth.Enabled && cfg.Auth.EmailVerification.Enabled {
		// Warn about sandbox/test domains in production
		if !cfg.Server.Dev {
			if cfg.Auth.EmailVerification.Provider == "mailgun" {
				domain := cfg.Auth.EmailVerification.Mailgun.Domain
				if strings.Contains(domain, "sandbox") || strings.HasPrefix(domain, "mg.") && strings.HasSuffix(domain, ".mailgun.org") {
					warnings = append(warnings, "email verification: using Mailgun sandbox domain in production mode - emails will only be delivered to authorized recipients")
				}
			}
			if cfg.Auth.EmailVerification.Provider == "resend" {
				from := cfg.Auth.EmailVerification.Resend.From
				if strings.Contains(from, "onboarding@resend.dev") {
					warnings = append(warnings, "email verification: using Resend test domain (onboarding@resend.dev) in production mode")
				}
			}
		}

		// Warn if verification enabled but no HTTPS configured (verification links need HTTPS)
		if !cfg.Server.Dev && !cfg.Server.HTTPS.Auto && cfg.Server.HTTPS.Cert == "" {
			warnings = append(warnings, "email verification: enabled but HTTPS not configured - verification links will use HTTP which is insecure")
		}

		// Warn if provider not properly configured
		switch cfg.Auth.EmailVerification.Provider {
		case "mailgun":
			if cfg.Auth.EmailVerification.Mailgun.APIKey == "" || cfg.Auth.EmailVerification.Mailgun.Domain == "" {
				warnings = append(warnings, "email verification: Mailgun selected but api_key or domain not configured")
			}
		case "resend":
			if cfg.Auth.EmailVerification.Resend.APIKey == "" {
				warnings = append(warnings, "email verification: Resend selected but api_key not configured")
			}
		case "":
			warnings = append(warnings, "email verification: enabled but no provider specified (mailgun or resend)")
		default:
			warnings = append(warnings, fmt.Sprintf("email verification: unknown provider %q (supported: mailgun, resend)", cfg.Auth.EmailVerification.Provider))
		}
	}

	// Warn if routes exist but none have handlers that exist
	// (This would be caught at runtime, but an early warning is helpful)

	return warnings
}

// resolveConfigPath finds the config file to use.
// Search order: explicit path > BASIL_CONFIG env > ./basil.yaml > ~/.config/basil/basil.yaml
func resolveConfigPath(explicit string, getenv func(string) string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file not found: %s", explicit)
		}
		return explicit, nil
	}

	// Try BASIL_CONFIG environment variable
	if envPath := getenv("BASIL_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", fmt.Errorf("BASIL_CONFIG file not found: %s", envPath)
		}
		return envPath, nil
	}

	// Try ./basil.yaml
	if _, err := os.Stat("basil.yaml"); err == nil {
		return "basil.yaml", nil
	}

	// Try ~/.config/basil/basil.yaml
	home, err := os.UserHomeDir()
	if err == nil {
		xdgPath := filepath.Join(home, ".config", "basil", "basil.yaml")
		if _, err := os.Stat(xdgPath); err == nil {
			return xdgPath, nil
		}
	}

	return "", fmt.Errorf("no config file found (tried BASIL_CONFIG, basil.yaml, ~/.config/basil/basil.yaml)")
}

// envPattern matches ${VAR} or ${VAR:-default}
var envPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

// interpolateEnv replaces ${VAR} and ${VAR:-default} patterns with environment values.
func interpolateEnv(data []byte, getenv func(string) string) []byte {
	return envPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		parts := envPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := string(parts[1])
		value := getenv(varName)

		if value == "" && len(parts) >= 3 && len(parts[2]) > 0 {
			value = string(parts[2])
		}

		return []byte(value)
	})
}

// validateBasic checks non-HTTPS configuration for errors.
func validateBasic(cfg *Config) error {
	var errs []string

	// Server validation
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("invalid port: %d (must be 1-65535)", cfg.Server.Port))
	}

	// Site and routes are mutually exclusive
	if cfg.Site != "" && len(cfg.Routes) > 0 {
		errs = append(errs, "site and routes are mutually exclusive - use site for filesystem routing or routes for explicit routing")
	}

	// Static routes validation
	for i, s := range cfg.Static {
		if s.Path == "" {
			errs = append(errs, fmt.Sprintf("static[%d]: path is required", i))
		}
		if s.Root == "" && s.File == "" {
			errs = append(errs, fmt.Sprintf("static[%d]: either root or file is required", i))
		}
		if s.Root != "" && s.File != "" {
			errs = append(errs, fmt.Sprintf("static[%d]: cannot specify both root and file", i))
		}
	}

	// Routes validation
	for i, r := range cfg.Routes {
		if r.Path == "" {
			errs = append(errs, fmt.Sprintf("routes[%d]: path is required", i))
		}
		if r.Handler == "" {
			errs = append(errs, fmt.Sprintf("routes[%d]: handler is required", i))
		}
		if r.Auth != "" && r.Auth != "required" && r.Auth != "optional" && r.Auth != "none" {
			errs = append(errs, fmt.Sprintf("routes[%d]: auth must be 'required', 'optional', 'none', or empty", i))
		}
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.Logging.Level] {
		errs = append(errs, fmt.Sprintf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logging.Level))
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.Logging.Format] {
		errs = append(errs, fmt.Sprintf("invalid log format: %s (must be json or text)", cfg.Logging.Format))
	}

	// CORS validation
	if len(cfg.CORS.Origins) > 0 {
		// Cannot use wildcard origin with credentials
		if cfg.CORS.Credentials && cfg.CORS.Origins.Contains("*") {
			errs = append(errs, "cors: cannot use origins '*' with credentials true (browsers reject this)")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// validateHTTPS checks HTTPS-specific configuration.
func validateHTTPS(cfg *Config) error {
	// Skip HTTPS validation in dev mode
	if cfg.Server.Dev {
		return nil
	}

	var errs []string

	// Production requires HTTPS
	if !cfg.Server.HTTPS.Auto && (cfg.Server.HTTPS.Cert == "" || cfg.Server.HTTPS.Key == "") {
		errs = append(errs, "production mode requires https.auto=true or both https.cert and https.key")
	}
	if cfg.Server.HTTPS.Auto && cfg.Server.HTTPS.Email == "" {
		errs = append(errs, "https.auto requires https.email for Let's Encrypt notifications")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// ParseSize parses a size string like "10MB", "1GB", "500KB" to bytes.
// Supports: B, KB, MB, GB (case insensitive).
// Returns 0 for empty string.
func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.TrimSpace(strings.ToUpper(s))

	// Check suffixes in order of length (longest first) to avoid "B" matching before "MB"
	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			numStr := strings.TrimSuffix(s, sf.suffix)
			numStr = strings.TrimSpace(numStr)
			var num int64
			if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
				return 0, fmt.Errorf("invalid size number: %s", numStr)
			}
			return num * sf.mult, nil
		}
	}

	// Try parsing as plain number (bytes)
	var num int64
	if _, err := fmt.Sscanf(s, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid size format: %s (use B, KB, MB, or GB suffix)", s)
	}
	return num, nil
}

// ApplyDeveloper applies a named developer profile to the configuration.
// Only non-zero values in the developer config override the base config.
// Returns an error if the profile name doesn't exist.
func ApplyDeveloper(cfg *Config, profileName string) error {
	if cfg.Developers == nil {
		return fmt.Errorf("no developer profiles defined in config")
	}

	dev, ok := cfg.Developers[profileName]
	if !ok {
		// List available profiles in error message
		var names []string
		for name := range cfg.Developers {
			names = append(names, name)
		}
		return fmt.Errorf("unknown developer profile %q (available: %s)", profileName, strings.Join(names, ", "))
	}

	// Apply port override
	if dev.Port != 0 {
		cfg.Server.Port = dev.Port
	}

	// Apply sqlite override
	if dev.SQLite != "" {
		path := dev.SQLite
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.BaseDir, path)
		}
		cfg.SQLite = path
	}

	// Apply handlers override - update all routes
	if dev.Handlers != "" {
		handlersDir := dev.Handlers
		if !filepath.IsAbs(handlersDir) {
			handlersDir = filepath.Join(cfg.BaseDir, handlersDir)
		}
		for i := range cfg.Routes {
			if cfg.Routes[i].Handler != "" {
				// Get just the filename and replace the directory
				base := filepath.Base(cfg.Routes[i].Handler)
				cfg.Routes[i].Handler = filepath.Join(handlersDir, base)
			}
		}
	}

	// Apply static/public_dir override
	if dev.Static != "" {
		staticDir := dev.Static
		if !filepath.IsAbs(staticDir) {
			staticDir = filepath.Join(cfg.BaseDir, staticDir)
		}
		cfg.PublicDir = staticDir
		// Also update routes that reference the public dir
		for i := range cfg.Routes {
			if cfg.Routes[i].PublicDir != "" {
				cfg.Routes[i].PublicDir = staticDir
			}
		}
	}

	// Apply logging overrides (only non-zero values)
	if dev.Logging.Level != "" {
		cfg.Logging.Level = dev.Logging.Level
	}
	if dev.Logging.Format != "" {
		cfg.Logging.Format = dev.Logging.Format
	}
	if dev.Logging.Output != "" {
		cfg.Logging.Output = dev.Logging.Output
	}
	if dev.Logging.Quiet {
		cfg.Logging.Quiet = true
	}

	return nil
}
