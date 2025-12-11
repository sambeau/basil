package config

import "time"

// Config represents the complete Basil configuration
type Config struct {
	BaseDir     string                     `yaml:"-"` // Directory containing config file, for resolving relative paths
	Server      ServerConfig               `yaml:"server"`
	Security    SecurityConfig             `yaml:"security"`
	CORS        CORSConfig                 `yaml:"cors"`
	Compression CompressionConfig          `yaml:"compression"`
	Auth        AuthConfig                 `yaml:"auth"`
	Session     SessionConfig              `yaml:"session"`
	Git         GitConfig                  `yaml:"git"`
	Dev         DevConfig                  `yaml:"dev"`
	SQLite     string                     `yaml:"sqlite"`     // Path to SQLite database file (e.g., "./data.db")
	PublicDir  string                     `yaml:"public_dir"` // Directory for static files, paths under this are rewritten to web URLs (default: "./public")
	Site       string                     `yaml:"site"`       // Directory for filesystem-based routing (mutually exclusive with routes)
	Static     []StaticRoute              `yaml:"static"`
	Routes     []Route                    `yaml:"routes"`
	Logging    LoggingConfig              `yaml:"logging"`
	Developers map[string]DeveloperConfig `yaml:"developers"` // Named developer profiles for per-developer overrides
}

// DeveloperConfig holds per-developer overrides
// All fields are optional - only non-zero values override the base config
type DeveloperConfig struct {
	Port     int           `yaml:"port"`     // Override server.port
	SQLite   string        `yaml:"sqlite"`   // Override sqlite path
	Handlers string        `yaml:"handlers"` // Override handlers directory (for routes)
	Static   string        `yaml:"static"`   // Override public_dir
	Logging  LoggingConfig `yaml:"logging"`  // Override logging settings
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
	Enabled           bool   `yaml:"enabled"`            // Enable HSTS header
	MaxAge            string `yaml:"max_age"`            // HSTS max-age in seconds (default: "31536000" = 1 year)
	IncludeSubDomains bool   `yaml:"include_subdomains"` // Include subdomains in HSTS
	Preload           bool   `yaml:"preload"`            // Allow HSTS preload list submission
}

// CORSConfig holds CORS (Cross-Origin Resource Sharing) settings
type CORSConfig struct {
	Origins     StringOrSlice `yaml:"origins"`     // "*" or list of allowed origins
	Methods     []string      `yaml:"methods"`     // Allowed HTTP methods (default: GET, HEAD, POST)
	Headers     []string      `yaml:"headers"`     // Allowed request headers
	Expose      []string      `yaml:"expose"`      // Response headers exposed to browser
	Credentials bool          `yaml:"credentials"` // Allow credentials (cookies, auth headers)
	MaxAge      int           `yaml:"maxAge"`      // Preflight cache duration in seconds
}

// CompressionConfig holds HTTP response compression settings
type CompressionConfig struct {
	Enabled bool   `yaml:"enabled"`  // Enable gzip/zstd compression (default: true)
	Level   string `yaml:"level"`    // Compression level: "fastest", "default", "best", "none" (default: "default")
	MinSize int    `yaml:"min_size"` // Minimum response size to compress in bytes (default: 1024)
	Zstd    bool   `yaml:"zstd"`     // Enable Zstd compression for supporting browsers (default: false)
}

// StringOrSlice supports YAML fields that can be either a string or a slice of strings
type StringOrSlice []string

// UnmarshalYAML implements yaml.Unmarshaler to handle both string and []string
func (s *StringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single string
	if err := unmarshal(&single); err == nil {
		*s = []string{single}
		return nil
	}

	var slice []string
	if err := unmarshal(&slice); err != nil {
		return err
	}
	*s = slice
	return nil
}

// Contains checks if the slice contains the given string
func (s StringOrSlice) Contains(str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	Enabled      bool          `yaml:"enabled"`      // Enable authentication
	Registration string        `yaml:"registration"` // "open" (anyone can register) or "closed" (invite only)
	SessionTTL   time.Duration `yaml:"session_ttl"`  // Session duration (default: 24h)
}

// GitConfig holds Git server settings
type GitConfig struct {
	Enabled     bool `yaml:"enabled"`      // Enable Git HTTP server at /.git/
	RequireAuth bool `yaml:"require_auth"` // Require API key authentication (default: true)
}

// SessionConfig holds session storage settings
type SessionConfig struct {
	Store      string        `yaml:"store"`       // Storage backend: "cookie" (default) or "sqlite"
	Secret     string        `yaml:"secret"`      // Encryption secret (required in production, auto-generated in dev)
	MaxAge     time.Duration `yaml:"max_age"`     // Session lifetime (default: 24h)
	CookieName string        `yaml:"cookie_name"` // Cookie name (default: "_basil_session")
	Secure     *bool         `yaml:"secure"`      // HTTPS only (default: true in production)
	HttpOnly   bool          `yaml:"http_only"`   // No JavaScript access (default: true)
	SameSite   string        `yaml:"same_site"`   // SameSite policy: "Lax", "Strict", "None" (default: "Lax")
	// SQLite-specific options (only used when store: sqlite)
	Table   string        `yaml:"table"`   // Table name (default: "_sessions")
	Cleanup time.Duration `yaml:"cleanup"` // Cleanup interval for expired sessions (default: 1h)
}

// DevConfig holds dev tools settings (only used when --dev flag is enabled)
type DevConfig struct {
	LogDatabase    string `yaml:"log_database"`     // Path to dev log database file (default: auto-generated)
	LogMaxSize     string `yaml:"log_max_size"`     // Maximum log database size (default: "10MB")
	LogTruncatePct int    `yaml:"log_truncate_pct"` // Percentage to delete when truncating (default: 25)
}

// StaticRoute maps URL paths to static files/directories
type StaticRoute struct {
	Path string `yaml:"path"` // URL path prefix (e.g., /static/)
	Root string `yaml:"root"` // Directory to serve (for directories)
	File string `yaml:"file"` // Single file to serve (for files like favicon.ico)
}

// Route maps URL paths to Parsley handlers
type Route struct {
	Path      string        `yaml:"path"`       // URL path pattern (supports * wildcard)
	Handler   string        `yaml:"handler"`    // Path to Parsley script
	Auth      string        `yaml:"auth"`       // "required", "optional", or empty
	Cache     time.Duration `yaml:"cache"`      // Response cache TTL (0 = no cache)
	PublicDir string        `yaml:"public_dir"` // Directory for static files for this route
	Type      string        `yaml:"type"`       // Route type: "api" for API modules, empty for page handlers
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level   string               `yaml:"level"`   // debug, info, warn, error
	Format  string               `yaml:"format"`  // json or text
	Output  string               `yaml:"output"`  // stderr, stdout, or file path
	Quiet   bool                 `yaml:"quiet"`   // suppress request logs
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
		CORS: CORSConfig{
			// Empty by default - CORS disabled unless configured
			Methods: []string{"GET", "HEAD", "POST"},
			MaxAge:  86400, // 24 hours
		},
		Compression: CompressionConfig{
			Enabled: true,
			Level:   "default",
			MinSize: 1024,
			Zstd:    false,
		},
		Auth: AuthConfig{
			Enabled:      false,
			Registration: "closed",
			SessionTTL:   24 * time.Hour,
		},
		Git: GitConfig{
			Enabled:     false,
			RequireAuth: true,
		},
		Session: SessionConfig{
			Store:      "cookie",
			MaxAge:     24 * time.Hour,
			CookieName: "_basil_session",
			HttpOnly:   true,
			SameSite:   "Lax",
			Table:      "_sessions",
			Cleanup:    1 * time.Hour,
		},
		Dev: DevConfig{
			LogMaxSize:     "10MB",
			LogTruncatePct: 25,
		},
		PublicDir: "./public",
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
