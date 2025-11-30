package config

import "time"

// Config represents the complete Basil configuration
type Config struct {
	BaseDir string        `yaml:"-"` // Directory containing config file, for resolving relative paths
	Server  ServerConfig  `yaml:"server"`
	Static  []StaticRoute `yaml:"static"`
	Routes  []Route       `yaml:"routes"`
	Logging LoggingConfig `yaml:"logging"`
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
	Auto  bool   `yaml:"auto"`  // Use Let's Encrypt
	Email string `yaml:"email"` // ACME email
	Cert  string `yaml:"cert"`  // Manual cert path
	Key   string `yaml:"key"`   // Manual key path
}

// ProxyConfig holds reverse proxy settings
type ProxyConfig struct {
	Trusted    bool     `yaml:"trusted"`     // Trust X-Forwarded-* headers
	TrustedIPs []string `yaml:"trusted_ips"` // Optional: restrict to specific proxies
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
