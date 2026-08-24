package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Site-root layout (FEAT-152 / DESIGN-git-deploy §4):
//
//	/srv/mysite/            ← the site root
//	  site.git/             bare repository
//	  releases/<id>/        one directory per deployed commit
//	  current -> releases/… the live release
//	  data/                 never touched by a deploy
const (
	// BareRepoName is the bare repository inside a site root.
	BareRepoName = "site.git"
	// ReleasesDirName holds one directory per deployed commit.
	ReleasesDirName = "releases"
	// CurrentLinkName is the symlink pointing at the active release.
	CurrentLinkName = "current"
	// DataDirName is the default data root, relative to the site root.
	DataDirName = "data"
	// UploadsDirName is the directory under the data root that is served
	// over HTTP at UploadsURLPrefix.
	UploadsDirName = "uploads"
	// CertsDirName is the default certificate cache, relative to the data root.
	CertsDirName = "certs"
	// UploadsURLPrefix is the URL prefix the uploads directory is served
	// under, following the existing /__p/ and /__img/ pattern.
	UploadsURLPrefix = "/__uploads/"
	// ConfigFileName is the name of the configuration file.
	ConfigFileName = "basil.yaml"
	// DefaultReleaseBranch is the branch a push must move to publish a
	// release (DESIGN-git-deploy §7). Not configurable in FEAT-153;
	// FEAT-154 adds deploy.branch.
	DefaultReleaseBranch = "live"
)

// IsSiteRoot reports whether dir looks like a Basil site root: it has a
// releases/ directory and a current entry. The legacy single-directory
// layout has neither.
func IsSiteRoot(dir string) bool {
	if dir == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(dir, ReleasesDirName)); err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Lstat(filepath.Join(dir, CurrentLinkName)); err != nil {
		return false
	}
	return true
}

// ConfigPathForSite returns the config file to load for a site root or a
// legacy project directory. In the site-root layout the config ships inside
// the release, so it is read through the `current` symlink.
func ConfigPathForSite(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid site path %q: %w", dir, err)
	}
	if IsSiteRoot(abs) {
		// Resolve `current` here, once, and return the release path it
		// named. Returning <root>/current/basil.yaml would leave the link
		// to be followed a second time when the anchors are derived, and a
		// deploy that re-points it in between - exactly the window a
		// restart-on-deploy sequence opens - would run one release's config
		// against another release's files.
		link := filepath.Join(abs, CurrentLinkName)
		release := link
		if target, err := os.Readlink(link); err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(abs, target)
			}
			release = filepath.Clean(target)
		}
		path := filepath.Join(release, ConfigFileName)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("site %s has no active release: %s is not readable (is the %q symlink broken?)", abs, path, CurrentLinkName)
		}
		return path, nil
	}
	path := filepath.Join(abs, ConfigFileName)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("no %s in %s (and it is not a site root: no %s/ and no %s)", ConfigFileName, abs, ReleasesDirName, CurrentLinkName)
	}
	return path, nil
}

// ResolveAnchors sets cfg.SiteRoot, cfg.ReleaseDir and cfg.DataDir from the
// directory the config file was loaded from.
//
// Two layouts are supported:
//
//   - Site root: configDir is <site root>/current (or <site root>/releases/<id>).
//     ReleaseDir is the active release, DataDir defaults to <site root>/data.
//   - Legacy single directory: ReleaseDir is the project directory and DataDir
//     defaults to the same place, which is exactly the pre-FEAT-152 behaviour.
func ResolveAnchors(cfg *Config, configDir string) {
	configDir = filepath.Clean(configDir)
	releaseDir := configDir
	siteRoot := ""

	parent := filepath.Dir(configDir)
	switch {
	case filepath.Base(configDir) == CurrentLinkName && IsSiteRoot(parent):
		siteRoot = parent
		// Resolve the active release through `current`, once, at load time.
		// A later deploy re-points the symlink; this process keeps serving
		// the release it started with until it is told otherwise.
		// Only the link itself is followed - resolving the whole path would
		// rewrite the operator's own spelling of it (/var vs /private/var).
		if target, err := os.Readlink(configDir); err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(parent, target)
			}
			releaseDir = filepath.Clean(target)
		}
	case filepath.Base(parent) == ReleasesDirName && IsSiteRoot(filepath.Dir(parent)):
		siteRoot = filepath.Dir(parent)
	}

	cfg.SiteRoot = siteRoot
	cfg.ReleaseDir = releaseDir

	switch {
	case cfg.DataDir == "":
		if siteRoot != "" {
			cfg.DataDir = filepath.Join(siteRoot, DataDirName)
		} else {
			// Legacy layout: state lives in the project directory.
			cfg.DataDir = releaseDir
		}
	case !filepath.IsAbs(cfg.DataDir):
		anchor := siteRoot
		if anchor == "" {
			anchor = releaseDir
		}
		cfg.DataDir = filepath.Join(anchor, cfg.DataDir)
	default:
		cfg.DataDir = filepath.Clean(cfg.DataDir)
	}
}

// underRelease resolves a code path against the release directory.
func underRelease(cfg *Config, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.ReleaseDir, path)
}

// underData resolves a persistent-state path against the data root.
func underData(cfg *Config, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.DataDir, path)
}

// isLogSink reports whether a logging output value names a stream rather
// than a file.
func isLogSink(v string) bool {
	switch v {
	case "", "stderr", "stdout", "response", "none", "discard":
		return true
	}
	return false
}

// ResolvePaths resolves every configured path against its anchor. It is
// idempotent: paths that are already absolute are left alone.
//
// Code — replaced by every deploy, read-only at runtime — resolves against
// ReleaseDir. Persistent state — which must survive a deploy — resolves
// against DataDir. Nothing resolves against the process working directory.
func ResolvePaths(cfg *Config) {
	// --- Code: ReleaseDir ---------------------------------------------
	for i := range cfg.Static {
		cfg.Static[i].Root = underRelease(cfg, cfg.Static[i].Root)
		cfg.Static[i].File = underRelease(cfg, cfg.Static[i].File)
	}
	for i := range cfg.Routes {
		cfg.Routes[i].Handler = underRelease(cfg, cfg.Routes[i].Handler)
		cfg.Routes[i].PublicDir = underRelease(cfg, cfg.Routes[i].PublicDir)
	}
	cfg.Site.Path = underRelease(cfg, cfg.Site.Path)
	cfg.PublicDir = underRelease(cfg, cfg.PublicDir)
	for code, path := range cfg.ErrorPages {
		cfg.ErrorPages[code] = underRelease(cfg, path)
	}

	// Apply global public_dir to the root route if not specified
	for i := range cfg.Routes {
		if cfg.Routes[i].Path == "/" && cfg.Routes[i].PublicDir == "" && cfg.PublicDir != "" {
			cfg.Routes[i].PublicDir = cfg.PublicDir
		}
	}

	// --- Persistent state: DataDir ------------------------------------
	cfg.Database.Path = underData(cfg, cfg.Database.Path)
	cfg.Images.CacheDir = underData(cfg, cfg.Images.CacheDir)
	cfg.Dev.LogDatabase = underData(cfg, cfg.Dev.LogDatabase)

	// https.cache_dir has always defaulted to "certs"; before FEAT-152 that
	// landed wherever the operator happened to be standing.
	if cfg.Server.HTTPS.CacheDir == "" {
		cfg.Server.HTTPS.CacheDir = CertsDirName
	}
	cfg.Server.HTTPS.CacheDir = underData(cfg, cfg.Server.HTTPS.CacheDir)
	cfg.Server.HTTPS.Cert = underData(cfg, cfg.Server.HTTPS.Cert)
	cfg.Server.HTTPS.Key = underData(cfg, cfg.Server.HTTPS.Key)

	if !isLogSink(cfg.Logging.Output) {
		cfg.Logging.Output = underData(cfg, cfg.Logging.Output)
	}
	if !isLogSink(cfg.Logging.Parsley.Output) {
		cfg.Logging.Parsley.Output = underData(cfg, cfg.Logging.Parsley.Output)
	}

	// security.allow_write names where site code may write at runtime. A
	// write inside the release would be destroyed by the next deploy, so
	// the whitelist resolves against the data root (FEAT-152, open item 1).
	for i := range cfg.Security.AllowWrite {
		cfg.Security.AllowWrite[i] = underData(cfg, cfg.Security.AllowWrite[i])
	}
}

// UploadsDir returns the durable directory that site code may write to and
// that Basil serves at UploadsURLPrefix.
func (c *Config) UploadsDir() string {
	if c.DataDir == "" {
		return ""
	}
	return filepath.Join(c.DataDir, UploadsDirName)
}

// WritePolicy returns the directories site code may write to: the configured
// security.allow_write entries plus the uploads directory.
//
// Uploads are a convention, not a setting. The uploads directory is created
// by `basil --init`, named to site code as basil.uploads_dir and served at
// UploadsURLPrefix; if it also had to be whitelisted by hand, the durable
// place to write would need a basil.yaml edit before it worked, which is the
// configuration step the site-root layout exists to remove (FEAT-152).
func (c *Config) WritePolicy() []string {
	policy := make([]string, 0, len(c.Security.AllowWrite)+1)
	policy = append(policy, []string(c.Security.AllowWrite)...)
	if uploads := c.UploadsDir(); uploads != "" {
		policy = append(policy, uploads)
	}
	return policy
}

// AuthDBPath returns the path of the authentication database. It lives in
// the data root: it holds the API keys people deploy with, so losing it to a
// deploy would lock everyone out of the mechanism that restores it.
func (c *Config) AuthDBPath() string {
	return filepath.Join(c.DataDir, ".basil-auth.db")
}

// DeployDBPath returns the path of the deploy record database. Like the auth
// database it lives in the data root: the record must survive the deploys it
// describes.
func (c *Config) DeployDBPath() string {
	return filepath.Join(c.DataDir, "deploy.db")
}

// BareRepoPath returns the bare repository for the site, or "" when there is
// no site root (the legacy layout keeps the repository in the project
// directory).
func (c *Config) BareRepoPath() string {
	if c.SiteRoot == "" {
		return ""
	}
	return filepath.Join(c.SiteRoot, BareRepoName)
}
