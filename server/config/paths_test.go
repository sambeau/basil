package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// everyPathKey exercises every configured path key that reaches a syscall.
// site: and routes: are mutually exclusive, so the routes keys are covered
// by routesYAML below.
const everyPathKey = `
server:
  host: example.com
  port: 443
  https:
    auto: true
    email: ops@example.com
    cache_dir: mycerts
    cert: tls/cert.pem
    key: tls/key.pem
public_dir: ./public
site:
  path: ./site
error_pages:
  404: ./errors/404.pars
static:
  - path: /favicon.ico
    file: ./public/favicon.ico
  - path: /assets/
    root: ./public/assets
database:
  path: ./data.db
images:
  cache_dir: ./cache/images
dev:
  log_database: ./dev_logs.db
logging:
  level: info
  format: text
  output: ./logs/basil.log
  parsley:
    output: ./logs/parsley.log
security:
  allow_write:
    - ./uploads
`

const routesYAML = `
server:
  host: example.com
public_dir: ./public
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/
    handler: ./handlers/api.pars
    public_dir: ./assets
`

// anchor names which of the two anchors a key must resolve against.
type anchor int

const (
	release anchor = iota
	data
)

func (a anchor) String() string {
	if a == release {
		return "ReleaseDir"
	}
	return "DataDir"
}

type pathCase struct {
	key  string
	want anchor
	rel  string // path relative to that anchor
	got  func(*Config) string
}

// pathTable is the audit table from FEAT-152, executable.
func pathTable() []pathCase {
	return []pathCase{
		// Code: replaced by every deploy.
		{"site.path", release, "site", func(c *Config) string { return c.Site.Path }},
		{"public_dir", release, "public", func(c *Config) string { return c.PublicDir }},
		{"static[].file", release, "public/favicon.ico", func(c *Config) string { return c.Static[0].File }},
		{"static[].root", release, "public/assets", func(c *Config) string { return c.Static[1].Root }},
		{"error_pages.404", release, "errors/404.pars", func(c *Config) string { return c.ErrorPages[404] }},

		// Persistent state: must survive a deploy.
		{"database.path", data, "data.db", func(c *Config) string { return c.Database.Path }},
		{"https.cache_dir", data, "mycerts", func(c *Config) string { return c.Server.HTTPS.CacheDir }},
		{"https.cert", data, "tls/cert.pem", func(c *Config) string { return c.Server.HTTPS.Cert }},
		{"https.key", data, "tls/key.pem", func(c *Config) string { return c.Server.HTTPS.Key }},
		{"images.cache_dir", data, "cache/images", func(c *Config) string { return c.Images.CacheDir }},
		{"dev.log_database", data, "dev_logs.db", func(c *Config) string { return c.Dev.LogDatabase }},
		{"logging.output", data, "logs/basil.log", func(c *Config) string { return c.Logging.Output }},
		{"logging.parsley.output", data, "logs/parsley.log", func(c *Config) string { return c.Logging.Parsley.Output }},
		{"security.allow_write[0]", data, "uploads", func(c *Config) string { return c.Security.AllowWrite[0] }},
	}
}

func routesTable() []pathCase {
	return []pathCase{
		{"routes[].handler", release, "handlers/index.pars", func(c *Config) string { return c.Routes[0].Handler }},
		{"routes[].public_dir (inherited)", release, "public", func(c *Config) string { return c.Routes[0].PublicDir }},
		{"routes[].public_dir", release, "assets", func(c *Config) string { return c.Routes[1].PublicDir }},
	}
}

// writeSiteRoot builds the site-root layout around a config file and returns
// the site root and the release directory.
func writeSiteRoot(t *testing.T, yaml string) (root, releaseDir string) {
	t.Helper()
	root = t.TempDir()
	releaseDir = filepath.Join(root, ReleasesDirName, "4f2a1c9")
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, DataDirName), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releaseDir, ConfigFileName), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(ReleasesDirName, "4f2a1c9"), filepath.Join(root, CurrentLinkName)); err != nil {
		t.Fatal(err)
	}
	return root, releaseDir
}

func writeLegacyProject(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestPathAnchors_SiteRoot is the table test the spec asks for: every
// configured path key, and which anchor it resolves against.
func TestPathAnchors_SiteRoot(t *testing.T) {
	root, releaseDir := writeSiteRoot(t, everyPathKey)
	dataDir := filepath.Join(root, DataDirName)

	cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.SiteRoot != root {
		t.Errorf("SiteRoot = %q, want %q", cfg.SiteRoot, root)
	}
	if cfg.ReleaseDir != releaseDir {
		t.Errorf("ReleaseDir = %q, want the active release %q", cfg.ReleaseDir, releaseDir)
	}
	if cfg.DataDir != dataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, dataDir)
	}

	for _, tc := range pathTable() {
		t.Run(tc.key, func(t *testing.T) {
			base := releaseDir
			if tc.want == data {
				base = dataDir
			}
			want := filepath.Join(base, tc.rel)
			if got := tc.got(cfg); got != want {
				t.Errorf("%s resolved to %q\n  want %s + %q = %q", tc.key, got, tc.want, tc.rel, want)
			}
			// The point of the split: state may never land in the release.
			if tc.want == data && strings.HasPrefix(tc.got(cfg), releaseDir+string(filepath.Separator)) {
				t.Errorf("%s landed inside the release, which a deploy replaces: %s", tc.key, tc.got(cfg))
			}
		})
	}
}

func TestPathAnchors_Routes(t *testing.T) {
	root, releaseDir := writeSiteRoot(t, routesYAML)

	cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, tc := range routesTable() {
		t.Run(tc.key, func(t *testing.T) {
			want := filepath.Join(releaseDir, tc.rel)
			if got := tc.got(cfg); got != want {
				t.Errorf("%s resolved to %q, want %q", tc.key, got, want)
			}
		})
	}
}

// The legacy single-directory layout keeps working, with the data root
// defaulting to the project directory - exactly the pre-FEAT-152 behaviour.
func TestPathAnchors_LegacyLayout(t *testing.T) {
	dir := writeLegacyProject(t, everyPathKey)

	cfg, err := Load(filepath.Join(dir, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.SiteRoot != "" {
		t.Errorf("SiteRoot = %q, want empty for the legacy layout", cfg.SiteRoot)
	}
	if cfg.ReleaseDir != dir {
		t.Errorf("ReleaseDir = %q, want %q", cfg.ReleaseDir, dir)
	}
	if cfg.DataDir != dir {
		t.Errorf("DataDir = %q, want the project directory %q", cfg.DataDir, dir)
	}

	for _, tc := range pathTable() {
		want := filepath.Join(dir, tc.rel)
		if got := tc.got(cfg); got != want {
			t.Errorf("%s resolved to %q, want %q", tc.key, got, want)
		}
	}
}

// Starting Basil from anywhere must produce identical paths. Before
// FEAT-152, https.cache_dir, images.cache_dir and the manual certificate
// paths moved with the operator's shell.
func TestResolutionIsIndependentOfWorkingDirectory(t *testing.T) {
	root, _ := writeSiteRoot(t, everyPathKey)
	cfgPath := filepath.Join(root, CurrentLinkName, ConfigFileName)

	elsewhere := t.TempDir()
	deeper := filepath.Join(elsewhere, "a", "b")
	if err := os.MkdirAll(deeper, 0755); err != nil {
		t.Fatal(err)
	}

	var baseline map[string]string
	for i, cwd := range []string{root, elsewhere, deeper} {
		t.Run(fmt.Sprintf("cwd%d", i), func(t *testing.T) {
			t.Chdir(cwd)
			cfg, err := Load(cfgPath, func(string) string { return "" })
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := map[string]string{
				"ReleaseDir": cfg.ReleaseDir,
				"DataDir":    cfg.DataDir,
			}
			for _, tc := range pathTable() {
				got[tc.key] = tc.got(cfg)
			}
			for k, v := range got {
				if !filepath.IsAbs(v) {
					t.Errorf("%s is not absolute: %q", k, v)
				}
				if strings.HasPrefix(v, cwd+string(filepath.Separator)) && !strings.HasPrefix(v, root+string(filepath.Separator)) {
					t.Errorf("%s resolved against the working directory: %q", k, v)
				}
			}
			if baseline == nil {
				baseline = got
				return
			}
			for k, want := range baseline {
				if got[k] != want {
					t.Errorf("%s differs by working directory: %q here, %q before", k, got[k], want)
				}
			}
		})
	}
}

func TestDataDirKey(t *testing.T) {
	t.Run("relative resolves against the site root", func(t *testing.T) {
		root, _ := writeSiteRoot(t, "server:\n  host: example.com\ndata_dir: ./state\n")
		cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if want := filepath.Join(root, "state"); cfg.DataDir != want {
			t.Errorf("DataDir = %q, want %q", cfg.DataDir, want)
		}
	})

	t.Run("absolute is left alone", func(t *testing.T) {
		root, _ := writeSiteRoot(t, "server:\n  host: example.com\ndata_dir: /srv/state\n")
		cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.DataDir != "/srv/state" {
			t.Errorf("DataDir = %q, want /srv/state", cfg.DataDir)
		}
		if cfg.Database.Path != "" {
			t.Errorf("unexpected database path %q", cfg.Database.Path)
		}
	})

	t.Run("relative resolves against the project in the legacy layout", func(t *testing.T) {
		dir := writeLegacyProject(t, "server:\n  host: example.com\ndata_dir: ./state\ndatabase:\n  path: ./app.db\n")
		cfg, err := Load(filepath.Join(dir, ConfigFileName), func(string) string { return "" })
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if want := filepath.Join(dir, "state"); cfg.DataDir != want {
			t.Errorf("DataDir = %q, want %q", cfg.DataDir, want)
		}
		if want := filepath.Join(dir, "state", "app.db"); cfg.Database.Path != want {
			t.Errorf("database.path = %q, want %q", cfg.Database.Path, want)
		}
	})
}

func TestUploadsAndAuthDBLiveInTheDataRoot(t *testing.T) {
	root, releaseDir := writeSiteRoot(t, everyPathKey)
	cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for name, got := range map[string]string{
		"uploads": cfg.UploadsDir(),
		"auth db": cfg.AuthDBPath(),
		"repo":    cfg.BareRepoPath(),
	} {
		if !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Errorf("%s is outside the site root: %q", name, got)
		}
		if strings.HasPrefix(got, releaseDir+string(filepath.Separator)) {
			t.Errorf("%s is inside the release, which a deploy replaces: %q", name, got)
		}
	}
}

func TestIsSiteRootAndConfigPathForSite(t *testing.T) {
	t.Run("site root", func(t *testing.T) {
		root, _ := writeSiteRoot(t, "server:\n  host: example.com\n")
		if !IsSiteRoot(root) {
			t.Error("expected a site root")
		}
		got, err := ConfigPathForSite(root)
		if err != nil {
			t.Fatalf("ConfigPathForSite: %v", err)
		}
		if want := filepath.Join(root, CurrentLinkName, ConfigFileName); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("legacy project", func(t *testing.T) {
		dir := writeLegacyProject(t, "server:\n  host: example.com\n")
		if IsSiteRoot(dir) {
			t.Error("a plain project directory is not a site root")
		}
		got, err := ConfigPathForSite(dir)
		if err != nil {
			t.Fatalf("ConfigPathForSite: %v", err)
		}
		if want := filepath.Join(dir, ConfigFileName); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("broken current symlink", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ReleasesDirName), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(ReleasesDirName, "gone"), filepath.Join(root, CurrentLinkName)); err != nil {
			t.Fatal(err)
		}
		_, err := ConfigPathForSite(root)
		if err == nil {
			t.Fatal("expected an error for a site with no active release")
		}
		if !strings.Contains(err.Error(), "no active release") {
			t.Errorf("the error does not name the problem: %v", err)
		}
	})

	t.Run("neither", func(t *testing.T) {
		if _, err := ConfigPathForSite(t.TempDir()); err == nil {
			t.Fatal("expected an error for a directory with no config")
		}
	})
}

// A public server must refuse to start without server.host: hostPolicy is
// open without it, so any hostname in SNI triggers an issuance attempt.
func TestValidateRequiresHostForPublicServer(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name:    "public server without host",
			mutate:  func(c *Config) {},
			wantErr: true,
		},
		{
			name:    "public server with host",
			mutate:  func(c *Config) { c.Server.Host = "example.com" },
			wantErr: false,
		},
		{
			name:    "dev mode is exempt",
			mutate:  func(c *Config) { c.Server.Dev = true },
			wantErr: false,
		},
		{
			name: "a manual certificate is exempt",
			mutate: func(c *Config) {
				c.Server.HTTPS.Auto = false
				c.Server.HTTPS.Cert = "/etc/tls/cert.pem"
				c.Server.HTTPS.Key = "/etc/tls/key.pem"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			cfg.Site.Path = "/srv/site"
			tt.mutate(cfg)
			err := Validate(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected a refusal")
				}
				if !strings.Contains(err.Error(), "server.host is required") {
					t.Errorf("the error does not name the fix: %v", err)
				}
				if !strings.Contains(err.Error(), "--dev") {
					t.Errorf("the error does not name the exceptions: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// https.email is recommended, not required: insisting on it would mean
// editing basil.yaml between `basil --init` and a working server.
func TestMissingACMEEmailWarnsButDoesNotRefuse(t *testing.T) {
	cfg := Defaults()
	cfg.Server.Host = "example.com"
	cfg.Site.Path = "/srv/site"
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var found bool
	for _, w := range Warnings(cfg) {
		if strings.Contains(w, "https.email") {
			found = true
		}
	}
	if !found {
		t.Error("no warning about the missing ACME contact address")
	}
}

// ResolvePaths runs on load and again after a developer profile is applied;
// running it twice must not move anything.
func TestResolvePathsIsIdempotent(t *testing.T) {
	root, _ := writeSiteRoot(t, everyPathKey)
	cfg, err := Load(filepath.Join(root, CurrentLinkName, ConfigFileName), func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := map[string]string{}
	for _, tc := range pathTable() {
		before[tc.key] = tc.got(cfg)
	}
	ResolvePaths(cfg)
	for _, tc := range pathTable() {
		if got := tc.got(cfg); got != before[tc.key] {
			t.Errorf("%s moved on a second resolve: %q -> %q", tc.key, before[tc.key], got)
		}
	}
}
