package config

import (
	"slices"
	"time"
)

// Config represents the complete Basil configuration.
//
// Paths are resolved against one of two anchors, never against the process
// working directory (FEAT-152):
//
//   - ReleaseDir — the site's code. Replaced wholesale on every deploy, so
//     nothing written at runtime may live here.
//   - DataDir — persistent state. Survives every deploy: databases, the auth
//     database, certificates, logs, caches, uploads.
type Config struct {
	// SiteRoot is the site root directory when the site-root layout is in
	// use (site.git/, releases/, current, data/). Empty in the legacy
	// single-directory layout.
	SiteRoot string `yaml:"-"`
	// ReleaseDir anchors every code path: site.path, public_dir,
	// routes[].handler, static[], error_pages. In the site-root layout this
	// is the active release (resolved through the `current` symlink); in the
	// legacy layout it is the directory containing basil.yaml.
	ReleaseDir string `yaml:"-"`
	// DataDir anchors every persistent path: database.path, the auth
	// database, https.cache_dir, logging output, dev.log_database,
	// images.cache_dir, security.allow_write. Defaults to <site root>/data,
	// or to the project directory in the legacy layout.
	DataDir     string                     `yaml:"data_dir"`
	Server      ServerConfig               `yaml:"server"`
	Security    SecurityConfig             `yaml:"security"`
	CORS        CORSConfig                 `yaml:"cors"`
	Compression CompressionConfig          `yaml:"compression"`
	Auth        AuthConfig                 `yaml:"auth"`
	Session     SessionConfig              `yaml:"session"`
	Deploy      DeployConfig               `yaml:"deploy"`
	Dev         DevConfig                  `yaml:"dev"`
	Database    DatabaseConfig             `yaml:"database"`    // Database configuration
	Images      ImageConfig                `yaml:"images"`      // Image transformation and caching configuration
	PublicDir   string                     `yaml:"public_dir"`  // Directory for static files, paths under this are rewritten to web URLs (default: "./public")
	Site        SiteConfig                 `yaml:"site"`        // Site mode configuration (filesystem-based routing)
	ErrorPages  map[int]string             `yaml:"error_pages"` // Custom error pages: status code -> .pars file (e.g. 404: ./errors/404.pars)
	Static      []StaticRoute              `yaml:"static"`
	Routes      []Route                    `yaml:"routes"`
	Logging     LoggingConfig              `yaml:"logging"`
	Developers  map[string]DeveloperConfig `yaml:"developers"` // Named developer profiles for per-developer overrides
	Meta        map[string]any             `yaml:"meta"`       // Custom metadata accessible as meta.* in Parsley
	Secrets     *SecretTracker             `yaml:"-"`          // Tracks which config paths contain secrets (for DevTools)

	// operatorOverrides holds one message per operator-owned setting this
	// config tried to decide (see operator.go). Unexported because it is
	// not configuration: it is a record of what loading decided, reported
	// through Warnings.
	operatorOverrides []string

	// retiredKeys holds one message per removed key this config still
	// carries (see operator.go). Same reasoning, and reported through the
	// same channel plus ReleaseWarnings, which `basil publish` prints so
	// a stale key in the repository everyone pulls from cannot linger unseen.
	retiredKeys []string

	// requestedGitEndpoint records that this config said git.enabled: true —
	// the only way the removed /.git endpoint was ever switched on. Read by
	// initGit, which needs to tell an operator who actually used that
	// endpoint from an ordinary local project (see operator.go).
	requestedGitEndpoint bool
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Path string `yaml:"path"` // Path to SQLite database file
}

// ImageConfig holds image transformation and caching settings
type ImageConfig struct {
	CacheDir       string `yaml:"cache_dir"`       // Directory for cached transformed images (default: "./cache/images")
	MaxWidth       int    `yaml:"max_width"`       // Maximum output width in pixels (default: 4096)
	MaxHeight      int    `yaml:"max_height"`      // Maximum output height in pixels (default: 4096)
	DefaultQuality int    `yaml:"default_quality"` // Default quality 1-100, 0 = format-specific default (JPEG: 85, WebP: 80, PNG: lossless)
	DefaultFormat  string `yaml:"default_format"`  // Default output format: "", "jpeg", "png", "webp" ("" = preserve original)
}

// DeveloperDBConfig holds per-developer database overrides
type DeveloperDBConfig struct {
	Path string `yaml:"path"` // Override database path
}

// SiteConfig holds site mode (filesystem-based routing) settings
type SiteConfig struct {
	Path  string        `yaml:"path"`  // Directory for filesystem-based routing
	Cache time.Duration `yaml:"cache"` // Response cache TTL (0 = no cache)
}

// DeveloperConfig holds per-developer overrides
// All fields are optional - only non-zero values override the base config
type DeveloperConfig struct {
	Port      int               `yaml:"port"`       // Override server.port
	Database  DeveloperDBConfig `yaml:"database"`   // Override database settings
	Handlers  string            `yaml:"handlers"`   // Override handlers directory (for routes)
	PublicDir string            `yaml:"public_dir"` // Override public_dir
	Logging   LoggingConfig     `yaml:"logging"`    // Override logging settings
}

// ServerConfig holds server settings.
//
// Host and Bind are different things and must not be conflated: Host is the
// public hostname this site answers to (the certificate name, the WebAuthn
// relying-party id, the hostname printed in deploy instructions), while Bind
// is the local interface the listener attaches to. A public server's hostname
// is routinely an address this machine does not own - NAT, Docker, an elastic
// IP or a load balancer in front - so binding it would fail.
type ServerConfig struct {
	Host  string      `yaml:"host"` // Public hostname (certificates, rpID, links)
	Bind  string      `yaml:"bind"` // Listener interface ("" = all interfaces)
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
	HSTS               HSTSConfig    `yaml:"hsts"`                 // HTTP Strict Transport Security
	ContentTypeOptions string        `yaml:"content_type_options"` // X-Content-Type-Options (default: "nosniff")
	FrameOptions       string        `yaml:"frame_options"`        // X-Frame-Options (default: "DENY")
	XSSProtection      string        `yaml:"xss_protection"`       // X-XSS-Protection (default: "1; mode=block")
	ReferrerPolicy     string        `yaml:"referrer_policy"`      // Referrer-Policy (default: "strict-origin-when-cross-origin")
	CSP                string        `yaml:"csp"`                  // Content-Security-Policy
	PermissionsPolicy  string        `yaml:"permissions_policy"`   // Permissions-Policy (formerly Feature-Policy)
	AllowWrite         StringOrSlice `yaml:"allow_write"`          // Directories where handlers can write files (e.g., ["./data", "./uploads"])
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
	MaxAge      int           `yaml:"max_age"`     // Preflight cache duration in seconds
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
func (s *StringOrSlice) UnmarshalYAML(unmarshal func(any) error) error {
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
	return slices.Contains(s, str)
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	Enabled           bool                    `yaml:"enabled"`            // Enable authentication
	Registration      string                  `yaml:"registration"`       // "open" (anyone can register) or "closed" (invite only)
	SessionTTL        time.Duration           `yaml:"session_ttl"`        // Session duration (default: 24h)
	ProtectedPaths    []ProtectedPath         `yaml:"protected_paths"`    // URL path prefixes that require authentication
	LoginPath         string                  `yaml:"login_path"`         // Path to redirect unauthenticated users (default: "/login")
	EmailVerification EmailVerificationConfig `yaml:"email_verification"` // Email verification settings (FEAT-084)
	Recovery          RecoveryConfig          `yaml:"recovery"`           // Recovery method settings (FEAT-084)
}

// EmailVerificationConfig holds email verification settings (FEAT-084)
type EmailVerificationConfig struct {
	Enabled             bool                  `yaml:"enabled"`              // Enable email verification
	Provider            string                `yaml:"provider"`             // "mailgun" or "resend"
	Mailgun             MailgunConfig         `yaml:"mailgun"`              // Mailgun-specific settings
	Resend              ResendConfig          `yaml:"resend"`               // Resend-specific settings
	RequireVerification bool                  `yaml:"require_verification"` // Block protected routes until verified
	TokenTTL            time.Duration         `yaml:"token_ttl"`            // Verification token lifetime (default: 1h)
	ResendCooldown      time.Duration         `yaml:"resend_cooldown"`      // Minimum time between resend requests (default: 5m)
	MaxSendsPerDay      int                   `yaml:"max_sends_per_day"`    // Per user/email abuse limit (default: 10)
	TemplateVars        EmailTemplateVars     `yaml:"template_vars"`        // Template variables
	DeveloperEmails     DeveloperEmailsConfig `yaml:"developer_emails"`     // Developer notification API settings
}

// MailgunConfig holds Mailgun-specific settings
type MailgunConfig struct {
	APIKey string `yaml:"api_key"` // Mailgun API key
	Domain string `yaml:"domain"`  // Mailgun domain
	Region string `yaml:"region"`  // "us" or "eu"
	From   string `yaml:"from"`    // From email address
}

// ResendConfig holds Resend-specific settings
type ResendConfig struct {
	APIKey string `yaml:"api_key"` // Resend API key
	From   string `yaml:"from"`    // From email address
}

// EmailTemplateVars holds email template variables
type EmailTemplateVars struct {
	SiteName string `yaml:"site_name"` // Site name for email templates
	SiteURL  string `yaml:"site_url"`  // Site URL for email templates
}

// DeveloperEmailsConfig holds settings for developer notification API
type DeveloperEmailsConfig struct {
	Enabled    bool `yaml:"enabled"`      // Enable developer email API (default: true)
	MaxPerHour int  `yaml:"max_per_hour"` // Per-site rate limit (default: 50)
	MaxPerDay  int  `yaml:"max_per_day"`  // Per-site rate limit (default: 200)
}

// RecoveryConfig holds recovery method settings (FEAT-084)
type RecoveryConfig struct {
	CodesEnabled bool `yaml:"codes_enabled"` // Enable recovery codes (default: true)
	EmailEnabled bool `yaml:"email_enabled"` // Enable email recovery (requires verified email)
}

// ProtectedPath represents a URL path prefix that requires authentication.
// Supports both simple string paths and paths with role requirements.
type ProtectedPath struct {
	Path  string   // URL path prefix (e.g., "/dashboard")
	Roles []string // Required roles (empty = any authenticated user)
}

// UnmarshalYAML implements yaml.Unmarshaler to handle both string and object formats.
// Supports:
//   - Simple string: "/dashboard"
//   - Object: {path: "/admin", roles: ["admin"]}
func (p *ProtectedPath) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string first
	var path string
	if err := unmarshal(&path); err == nil {
		p.Path = path
		p.Roles = nil
		return nil
	}

	// Try object format
	var obj struct {
		Path  string   `yaml:"path"`
		Roles []string `yaml:"roles"`
	}
	if err := unmarshal(&obj); err != nil {
		return err
	}
	p.Path = obj.Path
	p.Roles = obj.Roles
	return nil
}

// DeployConfig holds deploy engine settings (FEAT-153/FEAT-154). Validation
// is always on (the override is the --no-validate CLI flag, never config) and
// the post-deploy hook is a convention (deploy.pars in the release root), so
// neither gets a key here. Nor does the release branch: it is site.git's HEAD
// (FEAT-157), an operator fact a release cannot rewrite.
type DeployConfig struct {
	Keep int `yaml:"keep"` // Releases to retain when pruning (default: 5, minimum 2 enforced at prune time); the active and previous releases are always kept
}

// SessionConfig holds session storage settings
type SessionConfig struct {
	Store      string        `yaml:"store"`       // Storage backend: "cookie" (default) or "sqlite"
	Secret     SecretString  `yaml:"secret"`      // Encryption secret (use !secret auto for auto-generation)
	MaxAge     time.Duration `yaml:"max_age"`     // Session lifetime (default: 24h)
	CookieName string        `yaml:"cookie_name"` // Cookie name (default: "_basil_session")
	Secure     *bool         `yaml:"secure"`      // HTTPS only (default: true in production)
	HTTPOnly   bool          `yaml:"http_only"`   // No JavaScript access (default: true)
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
	Cache          bool   `yaml:"cache"`            // Enable response caching in dev mode (default: false)
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
	Auth      string        `yaml:"auth"`       // "required", "optional", "none", or empty
	Roles     []string      `yaml:"roles"`      // Required roles (used with auth: required)
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
		Deploy: DeployConfig{
			Keep: 5,
		},
		Images: ImageConfig{
			CacheDir:       "./cache/images",
			MaxWidth:       4096,
			MaxHeight:      4096,
			DefaultQuality: 0,  // Format-specific default
			DefaultFormat:  "", // Preserve original format
		},
		Session: SessionConfig{
			Store:      "cookie",
			Secret:     NewSecretString("auto"), // Auto-generate by default
			MaxAge:     24 * time.Hour,
			CookieName: "_basil_session",
			HTTPOnly:   true,
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
		Secrets: NewSecretTracker(),
	}
}
