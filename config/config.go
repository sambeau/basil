package config

import "time"

// Config represents the complete Basil configuration
type Config struct {
	BaseDir  string         `yaml:"-"` // Directory containing config file, for resolving relative paths
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	Database DatabaseConfig `yaml:"database"`
	Static   []StaticRoute  `yaml:"static"`
	Routes   []Route        `yaml:"routes"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite, postgres, mysql (only sqlite supported currently)
	Path   string `yaml:"path"`   // For sqlite: path to database file
	DSN    string `yaml:"dsn"`    // For postgres/mysql: connection string (future)
}

// ServerConfig holds server settings
type ServerConfig struct {
	Host  string      `yaml:"host"`
	Port  int         `yaml:"port"`
	Dev   bool        `yaml:"-"` // Set via CLI flag, not config
	HTTPS HTTPSConfig `yaml:"https"`
	Proxy ProxyConfig `yaml:"proxy"`
}

// HTTPSConfig holds TLS/HTTPS settings
type HTTPSConfig struct {
	Auto     bool   `yaml:"auto"`      // Use Let's Encrypt
	Email    string `yaml:"email"`     // ACME email for Let's Encrypt notifications
	CacheDir string `yaml:"cache_dir"` // Directory to store certificates (default: "certs")
	Cert     string `yaml:"cert"`      // Manual cert path (overrides auto)
	Key      string `yaml:"key"`       // Manual key path (overrides auto)
}

// ProxyConfig holds reverse proxy settings
type ProxyConfig struct {
	Trusted    bool     `yaml:"trusted"`     // Trust X-Forwarded-* headers
	TrustedIPs []string `yaml:"trusted_ips"` // Optional: restrict to specific proxies
}

// SecurityConfig holds security header settings
type SecurityConfig struct {
	HSTS               HSTSConfig `yaml:"hsts"`                 // HTTP Strict Transport Security
	ContentTypeOptions string     `yaml:"content_type_options"` // X-Content-Type-Options (default: "nosniff")
	FrameOptions       string     `yaml:"frame_options"`        // X-Frame-Options (default: "DENY")
	XSSProtection      string     `yaml:"xss_protection"`       // X-XSS-Protection (default: "1; mode=block")
	ReferrerPolicy     string     `yaml:"referrer_policy"`      // Referrer-Policy (default: "strict-origin-when-cross-origin")
	CSP                string     `yaml:"csp"`                  // Content-Security-Policy
	PermissionsPolicy  string     `yaml:"permissions_policy"`   // Permissions-Policy (formerly Feature-Policy)
}

// HSTSConfig holds HSTS (HTTP Strict Transport Security) settings
type HSTSConfig struct {
	Enabled           bool   `yaml:"enabled"`             // Enable HSTS header
	MaxAge            string `yaml:"max_age"`             // HSTS max-age in seconds (default: "31536000" = 1 year)
	IncludeSubDomains bool   `yaml:"include_subdomains"`  // Include subdomains in HSTS
	Preload           bool   `yaml:"preload"`             // Allow HSTS preload list submission
}

// StaticRoute maps URL paths to static files/directories
type StaticRoute struct {
	Path string `yaml:"path"` // URL path prefix (e.g., /static/)
	Root string `yaml:"root"` // Directory to serve (for directories)
	File string `yaml:"file"` // Single file to serve (for files like favicon.ico)
}

// Route maps URL paths to Parsley handlers
type Route struct {
	Path    string        `yaml:"path"`    // URL path pattern (supports * wildcard)
	Handler string        `yaml:"handler"` // Path to Parsley script
	Auth    string        `yaml:"auth"`    // "required", "optional", or empty
	Cache   time.Duration `yaml:"cache"`   // Response cache TTL (0 = no cache)
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level   string               `yaml:"level"`   // debug, info, warn, error
	Format  string               `yaml:"format"`  // json or text
	Output  string               `yaml:"output"`  // stderr, stdout, or file path
	Parsley ParsleyLoggingConfig `yaml:"parsley"` // Parsley script log() output
}

// ParsleyLoggingConfig holds Parsley-specific logging settings
type ParsleyLoggingConfig struct {
	Output string `yaml:"output"` // stderr, stdout, file path, or "response"
}

// Defaults returns a Config with sensible defaults
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "",
			Port: 443,
			HTTPS: HTTPSConfig{
				Auto: true,
			},
			Proxy: ProxyConfig{
				Trusted: false,
			},
		},
		Security: SecurityConfig{
			HSTS: HSTSConfig{
				Enabled:           true,
				MaxAge:            "31536000", // 1 year
				IncludeSubDomains: true,
				Preload:           false,
			},
			ContentTypeOptions: "nosniff",
			FrameOptions:       "DENY",
			XSSProtection:      "1; mode=block",
			ReferrerPolicy:     "strict-origin-when-cross-origin",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stderr",
			Parsley: ParsleyLoggingConfig{
				Output: "stderr",
			},
		},
	}
}
