package server

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
	"github.com/sambeau/basil/server/images"
	"golang.org/x/crypto/acme/autocert"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

// serveStaticFile serves a file with appropriate caching headers for dev/production.
// In dev mode, disables caching to ensure fresh content on every request.
// In production mode, uses http.ServeFile's built-in ETag support.
func serveStaticFile(w http.ResponseWriter, r *http.Request, filePath string, devMode bool) {
	if devMode {
		// Dev mode: disable all caching to prevent stale content issues
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
	http.ServeFile(w, r, filePath)
}

// ensureDataDir creates the data root if it is a directory of its own. In
// the legacy layout the data root is the project directory, which already
// exists, so there is nothing to do.
func (s *Server) ensureDataDir() error {
	dir := s.config.DataDir
	if dir == "" || dir == s.config.ReleaseDir {
		return nil
	}
	// 0700: this directory holds the auth database and its SQLite sidecars,
	// the certificate cache and the app database. On a shared host, 0755
	// would let every local account read them.
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating data directory %s: %w (if this tree was created by `sudo basil --init`, chown it to the account running Basil)", dir, err)
	}
	return nil
}

// newUploadsHandler serves the durable uploads directory under the data
// root. Site code writes here and a deploy never touches it, so uploads do
// not have to live inside public_dir. Directory listings are not served.
//
// The directory is opened as an os.Root, so no request can leave it. That is
// not paranoia about "..": http.Dir blocks those, but it follows symlinks,
// and uploads sits one level below the auth database, the certificate cache
// and the app database. Symlinks arrive in a directory like this routinely
// (rsync -l, tar -x, unzip), and os.Root refuses any component - symlink
// included - that resolves outside the root.
func newUploadsHandler(dir string, devMode bool) http.Handler {
	root, rootErr := os.OpenRoot(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rootErr != nil {
			http.NotFound(w, r)
			return
		}
		upath := strings.TrimPrefix(r.URL.Path, config.UploadsURLPrefix)
		if upath == "" || strings.HasSuffix(upath, "/") {
			http.NotFound(w, r)
			return
		}
		cleaned := path.Clean("/" + upath)
		f, err := root.Open(strings.TrimPrefix(cleaned, "/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		if devMode {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}

// Server represents a Basil web server instance.
type Server struct {
	config        *config.Config
	configPath    string
	version       string
	stdout        io.Writer
	stderr        io.Writer
	mux           *http.ServeMux
	server        *http.Server
	scriptCache   *scriptCache
	responseCache *responseCache
	fragmentCache *fragmentCache
	assetRegistry *assetRegistry
	imageRegistry *images.Registry
	assetBundle   *AssetBundle
	watcher       *Watcher
	db            *sql.DB // Database connection (nil if not configured)
	dbDriver      string  // Database driver name ("sqlite", etc.)
	rateLimiter   *rateLimiter

	// serving is the atomically-published request entry point. Run's
	// middleware chain wraps an indirection through this pointer rather than
	// the mux itself, so SwapRelease can hand new requests a rebuilt mux
	// while requests already dispatched finish on the handlers they entered.
	serving atomic.Pointer[serveState]
	// swapMu serialises SwapRelease: one release activation at a time.
	swapMu sync.Mutex

	// Session store (cookie-based by default)
	sessionStore  SessionStore
	sessionSecret string

	// Dev tools (nil if not in dev mode)
	devLog *DevLog

	// Auth system (nil if auth not enabled)
	authDB       *auth.DB
	authWebAuthn *auth.WebAuthnManager
	authHandlers *auth.Handlers
	authMW       *auth.Middleware

	// CSRF middleware
	csrfMW *CSRFMiddleware

	// CORS middleware
	corsMW *CORSMiddleware

	// Git server (nil if git not enabled)
	gitHandler *GitHandler

	// gitSwitch is basil.gitEnabled as this process read it from site.git at
	// startup, kept so a live deploy can notice an operator flipping it and
	// say a restart is needed (activate.go). Meaningful only when the
	// repository exists.
	gitSwitch bool

	// gitDegraded records the one server-side reason the endpoint is off
	// that is not the operator's switch: no authentication database, so
	// there are no API keys to authorise a push (initAuth).
	gitDegraded bool
}

// New creates a new Basil server with the given configuration.
func New(cfg *config.Config, configPath string, version, commit string, stdout, stderr io.Writer) (*Server, error) {
	s := &Server{
		config:        cfg,
		configPath:    configPath,
		version:       version,
		stdout:        stdout,
		stderr:        stderr,
		mux:           http.NewServeMux(),
		scriptCache:   newScriptCache(cfg.Server.Dev),
		responseCache: newResponseCache(cfg.Server.Dev, cfg.Dev.Cache),
		fragmentCache: newFragmentCache(cfg.Server.Dev, 1000),
		rateLimiter:   newRateLimiter(60, time.Minute),
		csrfMW:        NewCSRFMiddleware(cfg.Server.Dev),
	}

	// Let CSRF failures render the 403 error page (and any custom override)
	s.csrfMW.server = s

	// The data root holds everything that must survive a deploy. Create it
	// up front so a permission problem is reported as a permission problem
	// rather than as a database error on the first request.
	if err := s.ensureDataDir(); err != nil {
		return nil, err
	}

	// Initialize prelude (embedded assets and Parsley files)
	if err := initPrelude(commit); err != nil {
		return nil, fmt.Errorf("initializing prelude: %w", err)
	}

	// In dev mode, check if we're in the basil repo and enable prelude live reload
	if cfg.Server.Dev {
		s.initPreludeDevMode()
	}

	// Initialize CORS middleware if configured
	if len(cfg.CORS.Origins) > 0 {
		s.corsMW = NewCORSMiddleware(cfg.CORS)
	}

	// Warn about custom error pages that don't exist (the built-in page is
	// served instead when a configured page is missing or fails)
	for code, path := range cfg.ErrorPages {
		if _, err := os.Stat(path); err != nil {
			s.logWarn("error_pages: %d: %s not found, the built-in page will be used", code, path)
		}
	}

	// Initialize asset registry (logger for warnings, nil for production silent mode)
	if cfg.Server.Dev {
		s.assetRegistry = newAssetRegistry(s.logWarn)
	} else {
		s.assetRegistry = newAssetRegistry(nil)
	}

	// Initialize image registry for image() builtin
	if cfg.Server.Dev {
		s.imageRegistry = images.NewRegistry(
			cfg.Images.CacheDir,
			cfg.Images.MaxWidth,
			cfg.Images.MaxHeight,
			cfg.Images.DefaultQuality,
			cfg.Images.DefaultFormat,
			true, // devMode
			s.logWarn,
		)
	} else {
		s.imageRegistry = images.NewRegistry(
			cfg.Images.CacheDir,
			cfg.Images.MaxWidth,
			cfg.Images.MaxHeight,
			cfg.Images.DefaultQuality,
			cfg.Images.DefaultFormat,
			false, // devMode
			nil,
		)
	}

	// Initialize asset bundle (CSS/JS auto-bundling)
	if err := s.initAssetBundle(); err != nil {
		return nil, fmt.Errorf("initializing asset bundle: %w", err)
	}

	// Initialize session store
	if err := s.initSessions(); err != nil {
		return nil, fmt.Errorf("initializing sessions: %w", err)
	}

	// Initialize dev tools in dev mode
	if err := s.initDevTools(); err != nil {
		return nil, fmt.Errorf("initializing dev tools: %w", err)
	}

	// Initialize database connection if configured
	if err := s.initDatabase(); err != nil {
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing database: %w", err)
	}

	// Initialize auth system if enabled
	if err := s.initAuth(); err != nil {
		// Clean up database on auth init failure
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing auth: %w", err)
	}

	// Initialize Git server if enabled
	if err := s.initGit(); err != nil {
		if s.authDB != nil {
			s.authDB.Close()
		}
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("initializing git server: %w", err)
	}

	// Set up routes
	if err := s.setupRoutes(); err != nil {
		// Clean up on route setup failure
		if s.authDB != nil {
			s.authDB.Close()
		}
		if s.db != nil {
			s.db.Close()
		}
		s.cleanupDevTools()
		return nil, fmt.Errorf("setting up routes: %w", err)
	}

	// Publish the initial serving surface. From here on the mux, config and
	// bundle are only replaced through SwapRelease, which stores a new state
	// atomically.
	s.serving.Store(&serveState{mux: s.mux, config: cfg, assetBundle: s.assetBundle})

	return s, nil
}

// isProtectedPath checks if a URL path matches any protected path prefix.
// Returns the matching ProtectedPath if found, nil otherwise. It runs on the
// request path, so it reads the atomically-published live config, never
// s.config (rewritten by SwapRelease).
func (s *Server) isProtectedPath(urlPath string) *config.ProtectedPath {
	return protectedPathIn(s.liveConfig(), urlPath)
}

// protectedPathIn is isProtectedPath against an explicit config, for
// handlers that pinned their config at construction (see SwapRelease).
func protectedPathIn(cfg *config.Config, urlPath string) *config.ProtectedPath {
	for i := range cfg.Auth.ProtectedPaths {
		pp := &cfg.Auth.ProtectedPaths[i]
		// Match the path exactly or as a prefix
		// /dashboard matches /dashboard, /dashboard/, /dashboard/anything
		if urlPath == pp.Path ||
			strings.HasPrefix(urlPath, pp.Path+"/") ||
			(pp.Path != "/" && urlPath+"/" == pp.Path+"/") {
			return pp
		}
	}
	return nil
}

// getLoginPath returns the configured login path or the default. Request
// path: reads the live config.
func (s *Server) getLoginPath() string {
	if path := s.liveConfig().Auth.LoginPath; path != "" {
		return path
	}
	return "/login"
}

// initDevTools initializes dev tools (logging, etc.) in dev mode.
func (s *Server) initDevTools() error {
	if !s.config.Server.Dev {
		return nil
	}

	// Create dev log database with config overrides
	cfg := DefaultDevLogConfig()

	// Apply config overrides
	if s.config.Dev.LogDatabase != "" {
		cfg.Path = s.config.Dev.LogDatabase
	}
	if s.config.Dev.LogMaxSize != "" {
		size, err := config.ParseSize(s.config.Dev.LogMaxSize)
		if err != nil {
			return fmt.Errorf("parsing dev.log_max_size: %w", err)
		}
		if size > 0 {
			cfg.MaxSize = size
		}
	}
	if s.config.Dev.LogTruncatePct > 0 {
		cfg.TruncatePct = s.config.Dev.LogTruncatePct
	}

	// The dev log is persistent state, so it lives in the data root.
	// Fall back to a temp directory if that doesn't exist (e.g. in tests).
	dataDir := s.config.DataDir
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		dataDir = os.TempDir()
	}

	devLog, err := NewDevLog(dataDir, cfg)
	if err != nil {
		return fmt.Errorf("creating dev log: %w", err)
	}

	s.devLog = devLog
	s.logInfo("dev tools enabled, logs at: %s", devLog.Path())
	return nil
}

// cleanupDevTools closes dev tools resources.
func (s *Server) cleanupDevTools() {
	if s.devLog != nil {
		s.devLog.Close()
		s.devLog = nil
	}
}

// initAssetBundle initializes the CSS/JS asset bundle.
func (s *Server) initAssetBundle() error {
	s.assetBundle = buildAssetBundle(s.config, s.logWarn)
	return nil
}

// buildAssetBundle constructs and fills the CSS/JS asset bundle for a
// config. It is a function of the config alone so SwapRelease can build the
// new release's bundle before touching any server state.
func buildAssetBundle(cfg *config.Config, logWarn func(string, ...any)) *AssetBundle {
	// Determine handlers directory from routes or site config
	handlersDir := determineHandlersDir(cfg)
	publicDirName := filepath.Base(cfg.PublicDir)
	if handlersDir == "" {
		// No routes configured, create empty bundle
		return NewAssetBundle("", cfg.Server.Dev, publicDirName)
	}

	bundle := NewAssetBundle(handlersDir, cfg.Server.Dev, publicDirName)
	if err := bundle.Rebuild(); err != nil {
		// Log warning but don't fail - bundle just won't have content
		logWarn("failed to build asset bundle: %v", err)
	}
	return bundle
}

// determineHandlersDir finds the handler root directory for asset bundle discovery.
// In site mode, this is the parent of the site/ directory (the handler root).
// In route mode, this is the common ancestor of all handler files.
func determineHandlersDir(cfg *config.Config) string {
	// If using site (filesystem routing), use the parent of the site directory
	// This allows discovering CSS/JS in components/, public/, etc. at handler root level
	if cfg.Site.Path != "" {
		dir := filepath.Dir(cfg.Site.Path)
		// Resolve symlinks to ensure WalkDir can traverse the actual directory
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return resolved
		}
		return dir
	}

	// Otherwise, find common parent of all route handlers
	if len(cfg.Routes) == 0 {
		return ""
	}

	// Get directory of first handler
	commonDir := filepath.Dir(cfg.Routes[0].Handler)

	// Find common ancestor with all other handlers
	for _, route := range cfg.Routes[1:] {
		handlerDir := filepath.Dir(route.Handler)
		commonDir = commonAncestor(commonDir, handlerDir)
	}

	return commonDir
}

// commonAncestor returns the common ancestor directory of two paths.
func commonAncestor(path1, path2 string) string {
	// Clean paths
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)

	// Split into components
	parts1 := strings.Split(path1, string(filepath.Separator))
	parts2 := strings.Split(path2, string(filepath.Separator))

	// Find common prefix
	var common []string
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] == parts2[i] {
			common = append(common, parts1[i])
		} else {
			break
		}
	}

	if len(common) == 0 {
		return ""
	}

	result := filepath.Join(common...)

	// Fix: If original paths were absolute (Unix), filepath.Join loses the leading /
	// because it joins ["", "Users", ...] → "Users/..." instead of "/Users/..."
	if len(common) > 0 && common[0] == "" && !filepath.IsAbs(result) {
		result = string(filepath.Separator) + result
	}

	return result
}

// Close closes all server resources. Use this in tests; in production, use Run() with a context.
func (s *Server) Close() {
	s.cleanupDevTools()
	if s.authDB != nil {
		s.authDB.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

// initSessions initializes the session store.
func (s *Server) initSessions() error {
	cfg := &s.config.Session

	// Determine session secret using SecretString resolution
	// This handles: !secret auto (generate), !secret ${VAR} (env var), or literal value
	secret, err := config.ResolveSecretValue(cfg.Secret, os.Getenv("SESSION_SECRET"))
	if err != nil {
		return fmt.Errorf("resolving session secret: %w", err)
	}

	if secret == "" {
		if s.config.Server.Dev {
			// In dev mode, generate a random secret (sessions won't persist across restarts)
			secret, err = config.GenerateSecureSecret()
			if err != nil {
				return fmt.Errorf("generating dev session secret: %w", err)
			}
			s.logInfo("sessions: using auto-generated secret (dev mode)")
		} else {
			// In production, require explicit secret
			s.logWarn("sessions: no secret configured, sessions disabled")
			return nil
		}
	} else if cfg.Secret.IsAuto() {
		s.logInfo("sessions: using auto-generated secret")
	}

	s.sessionSecret = secret

	// Create cookie session store (default and currently only supported store)
	cookieStore := NewCookieSessionStore(cfg, secret, s.config.Server.Dev)
	s.sessionStore = cookieStore

	s.logInfo("sessions: cookie store initialized (max_age=%s, secure=%v)", cfg.MaxAge, cookieStore.isSecure())
	return nil
}

// initDatabase opens the SQLite database connection if configured. It runs
// at construction and again from ReloadDatabase (a dev-tools request path),
// so it reads the live config.
func (s *Server) initDatabase() error {
	// No database configured
	if s.liveConfig().Database.Path == "" {
		return nil
	}

	return s.initSQLite(s.liveConfig().Database.Path)
}

// initSQLite opens a SQLite database connection.
func (s *Server) initSQLite(path string) error {
	if path == "" {
		return fmt.Errorf("sqlite path is empty")
	}

	// Path should already be resolved by config loader, but handle relative just in case
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.liveConfig().DataDir, path)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return fmt.Errorf("opening sqlite database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("connecting to sqlite database: %w", err)
	}

	// Configure connection pool for SQLite
	// SQLite works best with a single writer, but can handle multiple readers
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Keep connection open indefinitely

	s.db = db
	s.dbDriver = "sqlite"

	s.logInfo("connected to SQLite database: %s", path)
	return nil
}

// ReloadDatabase closes the current database connection and reopens it.
// This is necessary after the database file is replaced (e.g., by upload).
func (s *Server) ReloadDatabase() error {
	if s.db == nil {
		return nil // No database configured
	}

	// Close existing connection
	if err := s.db.Close(); err != nil {
		s.logWarn("error closing database during reload: %v", err)
	}
	s.db = nil

	// Reopen the database
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("reopening database: %w", err)
	}

	s.logInfo("database connection reloaded")
	return nil
}

// initAuth initializes the authentication system if enabled.
func (s *Server) initAuth() error {
	if !s.config.Auth.Enabled {
		return nil
	}

	// On a site root auth.enabled is operator-owned (config.OperatorOwnedKeys):
	// it is on even for a release whose config never mentions auth, which is
	// the normal shape after FEAT-156's init split. That must not turn a
	// missing database into a dead server. Before FEAT-156 such a site started
	// with auth off and simply declined git; forcing the setting on would
	// instead make <data>/.basil-auth.db a startup requirement, so a site root
	// whose data dir has not got one yet could not be started at all — and the
	// fix (`basil users create`) needs a shell on the box, which is exactly the
	// state the operator-owned rule exists to avoid.
	//
	// So this one degrade is allowed, and only here: a MISSING database is
	// server-side state, never something the release's config can ask for. The
	// server runs with authentication and the git endpoint off (they are the
	// same guarantee — pushes are authorised out of this database) and says so
	// loudly. The legacy layout does not reach this code: nothing is forced
	// there, so auth.enabled false means auth off, as always.
	if s.config.SiteRoot != "" {
		if _, err := os.Stat(s.config.AuthDBPath()); os.IsNotExist(err) {
			s.logWarn("no authentication database at %s — running with authentication AND the git deploy endpoint DISABLED for this run; create an account with `basil users create` on the server and restart to turn them back on", s.config.AuthDBPath())
			s.config.Auth.Enabled = false
			s.gitDegraded = true
			return nil
		}
	}

	// Open auth database (separate from app database)
	authDB, err := auth.OpenDB(s.config.AuthDBPath())
	if err != nil {
		return fmt.Errorf("opening auth database: %w", err)
	}
	s.authDB = authDB

	// Determine RP ID and origin from server config
	rpID := s.config.Server.Host
	if rpID == "" {
		rpID = "localhost"
	}

	// Build origin
	var origin string
	if s.config.Server.Dev {
		origin = fmt.Sprintf("http://localhost:%d", s.config.Server.Port)
	} else if s.config.Server.HTTPS.Auto || s.config.Server.HTTPS.Cert != "" {
		origin = fmt.Sprintf("https://%s", rpID)
		if s.config.Server.Port != 443 {
			origin = fmt.Sprintf("%s:%d", origin, s.config.Server.Port)
		}
	} else {
		origin = fmt.Sprintf("http://%s:%d", rpID, s.config.Server.Port)
	}

	// Initialize WebAuthn
	webauthn, err := auth.NewWebAuthnManager(authDB, rpID, origin, rpID)
	if err != nil {
		authDB.Close()
		return fmt.Errorf("initializing webauthn: %w", err)
	}
	s.authWebAuthn = webauthn

	// Create handlers and middleware
	sessionTTL := s.config.Auth.SessionTTL
	if sessionTTL == 0 {
		sessionTTL = 24 * time.Hour
	}
	secure := !s.config.Server.Dev // Secure cookies in production
	regOpen := s.config.Auth.Registration == "open"

	// Initialize email service if enabled (FEAT-084)
	var emailService *auth.EmailService
	if s.config.Auth.EmailVerification.Enabled {
		emailService, err = auth.NewEmailService(&s.config.Auth.EmailVerification, authDB, origin)
		if err != nil {
			s.logWarn("failed to initialize email service: %v", err)
			emailService = nil
		} else {
			s.logInfo("email verification enabled (provider: %s)", s.config.Auth.EmailVerification.Provider)
		}
	}

	requireVerif := s.config.Auth.EmailVerification.RequireVerification

	s.authHandlers = auth.NewHandlers(authDB, webauthn, emailService, sessionTTL, secure, regOpen, requireVerif)
	s.authMW = auth.NewMiddleware(authDB)

	s.logInfo("authentication enabled (registration: %s)", s.config.Auth.Registration)
	return nil
}

// initGit initializes the Git endpoint (FEAT-154). It serves the site's bare
// repository — <site root>/site.git — and only that: Git deploy is on when
// the repository exists, and the operator's off-switch is `git config
// basil.gitEnabled false` inside that repository (FEAT-157 — it used to be
// git.enabled in basil.yaml, which a release could set). In the legacy layout
// (no site root) there is no bare repository, so there is no Git endpoint;
// the old behaviour of serving the live project directory over HTTP is gone
// (it is the arrangement BUG-033 grew from).
func (s *Server) initGit() error {
	repo := s.config.BareRepoPath()
	if repo == "" {
		// Legacy layout. Only an operator who switched the removed endpoint
		// on has lost anything, and the config is what says so — a project
		// directory that merely is a repository is what `basil --init` now
		// writes every time (config.RequestedGitEndpoint).
		if config.RequestedGitEndpoint(s.config) {
			s.logWarn("git: the endpoint serving the project directory at /.git was removed (it required pushes into a checked-out branch, BUG-033); to get Git deploy, run `basil --init <dir> --server` on the server and connect this folder to it")
		}
		return nil
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return nil // no repository, no Git
	}

	// A repository inside a served root would expose every version of every
	// file over plain HTTP. site.git is a sibling of releases/ so an --init
	// layout cannot get here; this guards misconfigured public_dir,
	// site.path and static roots, which are configurable.
	//
	// This runs BEFORE the off-switch and the degraded check, and must keep
	// doing so: it is a fact about where the repository sits on disk, not
	// about whether Basil serves /.git. Turning the Git endpoint off does
	// nothing to a static route that hands out site.git's objects — every
	// version of every file, including any secret ever committed — so
	// `basil.gitEnabled false` must not be a way past this guard.
	if err := config.CheckRepoOutsideServedRoots(s.config, repo); err != nil {
		return fmt.Errorf("refusing to serve Git: %w", err)
	}

	// The operator's off-switch, read from the repository rather than the
	// release. Recorded either way, so a later deploy can tell whether it
	// moved (activate.go).
	gitSwitch, switchErr := deploy.GitEnabled(repo)
	if switchErr != nil {
		// Enabled-because-unreadable is not the same as enabled-because-unset,
		// and only one of them is what the operator asked for.
		s.logWarn("git: %v", switchErr)
	}
	s.gitSwitch = gitSwitch
	if !s.gitSwitch {
		s.logInfo("git: /.git is not served - basil.gitEnabled is false in %s", repo)
		return nil
	}
	if s.gitDegraded {
		return nil // no auth database: initAuth already said so
	}

	// No auth database, no Git: pushes are authorised by API keys in that
	// database, so without it the endpoint cannot exist. Dev mode may run
	// without one because the handler only serves localhost then.
	if s.authDB == nil && !s.config.Server.Dev {
		return fmt.Errorf("git deploy requires the auth database: set auth.enabled: true (pushes are authorised by its API keys), or turn Git off with: git -C %s config basil.gitEnabled false", repo)
	}

	// (Re-)install the receive hooks: healing a deleted hook or a moved
	// basil binary here is what keeps a push deploying instead of silently
	// storing. A hook Basil did not write is a hard error naming the file —
	// an operator's hook silently pre-empting deploys is the failure mode
	// this refuses to allow.
	if err := deploy.InstallHooks(repo); err != nil {
		return fmt.Errorf("installing receive hooks: %w", err)
	}

	gitHandler, err := NewGitHandler(repo, s.authDB, s.config, s.stdout, s.stderr)
	if err != nil {
		return fmt.Errorf("creating git handler: %w", err)
	}

	s.gitHandler = gitHandler
	s.logInfo("git deploy enabled: serving %s at /.git/", repo)
	return nil
}

// setupRoutes configures the HTTP mux with static and dynamic routes.
func (s *Server) setupRoutes() error {
	// Register asset handler for publicUrl() files at /__p/
	s.mux.Handle("/__p/", newAssetHandler(s.assetRegistry, s.devLog != nil))

	// Register image handler for image() files at /__img/
	s.mux.Handle("/__img/", images.NewHandler(s.imageRegistry, s.devLog != nil))

	// Register the uploads handler for site-written files at /__uploads/.
	//
	// Only in the site-root layout, and only when `basil --init` actually
	// created the directory. A legacy project may already have a
	// <project>/uploads that it has been writing user-submitted files to on
	// the old advice to whitelist ./uploads; upgrading Basil must not
	// publish that directory to the internet behind the operator's back.
	//
	// The handler is wrapped like any other route, so auth.protected_paths
	// covers /__uploads/ - it is public unless the operator says otherwise.
	if uploads := s.config.UploadsDir(); uploads != "" && s.config.SiteRoot != "" {
		if info, err := os.Stat(uploads); err == nil && info.IsDir() {
			h := s.guardProtectedPath(newUploadsHandler(uploads, s.devLog != nil))
			s.mux.Handle(config.UploadsURLPrefix, s.applyAuthMiddleware(h, "optional"))
		}
	}

	// Register asset bundle routes. The closures pin the bundle current at
	// setup: the mux and its release's bundle age out together, and
	// s.assetBundle is rewritten by SwapRelease off the request path.
	bundle := s.assetBundle
	s.mux.HandleFunc("/__site.css", func(w http.ResponseWriter, r *http.Request) {
		bundle.ServeCSS(w, r)
	})
	s.mux.HandleFunc("/__site.js", func(w http.ResponseWriter, r *http.Request) {
		bundle.ServeJS(w, r)
	})

	// Register prelude asset handlers
	s.mux.HandleFunc("/__/js/", s.handlePreludeAsset)
	s.mux.HandleFunc("/__/css/", s.handlePreludeAsset)
	s.mux.HandleFunc("/__/public/", s.handlePreludeAsset)

	// In dev mode, add dev tools endpoints
	if s.config.Server.Dev {
		s.mux.Handle("/__livereload", newLiveReloadHandler(s))
		// Dev tools handler for /__/* routes (logs, etc.)
		devTools := newDevToolsHandler(s)
		s.mux.Handle("/__/", devTools)
		s.mux.Handle("/__", devTools)
	}

	// Register auth endpoints if auth is enabled
	if s.authHandlers != nil {
		s.mux.HandleFunc("/__auth/register/begin", s.authHandlers.BeginRegisterHandler)
		s.mux.HandleFunc("/__auth/register/finish", s.authHandlers.FinishRegisterHandler)
		s.mux.HandleFunc("/__auth/login/begin", s.authHandlers.BeginLoginHandler)
		s.mux.HandleFunc("/__auth/login/finish", s.authHandlers.FinishLoginHandler)
		s.mux.HandleFunc("/__auth/logout", s.authHandlers.LogoutHandler)
		s.mux.HandleFunc("/__auth/recover", s.authHandlers.RecoverHandler)
		s.mux.HandleFunc("/__auth/me", s.authHandlers.MeHandler)
		// Email verification endpoints (FEAT-084)
		s.mux.HandleFunc("/__auth/verify-email", s.authHandlers.VerifyEmailHandler)
		s.mux.HandleFunc("/__auth/resend-verification", s.authHandlers.ResendVerificationHandler)
		s.mux.HandleFunc("/__auth/verify-email-required", s.authHandlers.VerificationRequiredHandler)
		s.mux.HandleFunc("/__auth/recover/email", s.authHandlers.RecoverEmailHandler)
		s.mux.HandleFunc("/__auth/recover/verify", s.authHandlers.RecoverVerifyHandler)
	}

	// Register Git server if enabled
	if s.gitHandler != nil {
		s.mux.Handle("/.git/", s.gitHandler)
	}

	// Register explicit static routes (non-root paths like /favicon.ico)
	for _, static := range s.config.Static {
		if static.Path != "/" {
			if static.Root != "" {
				handler := http.StripPrefix(static.Path, http.FileServer(http.Dir(static.Root)))
				s.mux.Handle(static.Path, handler)
			} else if static.File != "" {
				filePath := static.File
				devMode := s.devLog != nil
				s.mux.HandleFunc(static.Path, func(w http.ResponseWriter, r *http.Request) {
					serveStaticFile(w, r, filePath, devMode)
				})
			}
		}
	}

	// Site mode: use filesystem-based routing
	// Optional auth wraps the whole site so the protected-paths check
	// (and handlers) can see the signed-in user.
	if s.config.Site.Path != "" {
		s.mux.Handle("/", s.applyAuthMiddleware(newSiteHandler(s, s.config.Site.Path, s.scriptCache), "optional"))
		s.logInfo("site mode enabled at %s", s.config.Site.Path)
		return nil
	}

	// Routes mode: explicit route-based routing
	// Register Parsley routes (specific paths)
	for _, route := range s.config.Routes {
		if route.Path == "/" {
			continue // Handle root separately as fallback
		}

		isAPI := isAPIRoute(route)
		var handler http.Handler
		var err error

		if isAPI {
			var ah *apiHandler
			ah, err = newAPIHandler(s, route, s.scriptCache)
			if ah != nil {
				// Pin the release being set up (see SwapRelease).
				ah.cfg = s.config
				ah.assetBundle = s.assetBundle
				handler = ah
			}
		} else {
			var ph *parsleyHandler
			ph, err = newParsleyHandler(s, route, s.scriptCache)
			if ph != nil {
				// Pin the release being set up (see SwapRelease).
				ph.cfg = s.config
				ph.assetBundle = s.assetBundle
				handler = ph
			}
		}
		if err != nil {
			return fmt.Errorf("creating handler for %s: %w", route.Path, err)
		}

		authMode := route.Auth
		if isAPI && authMode == "" {
			// For API routes, always run OptionalAuth to populate context without forcing login;
			// handler-level wrappers will enforce.
			authMode = "optional"
		}

		// Protected-path and role checks read the user from the request
		// context, so the auth middleware must wrap them (run first).
		// (auth: "none" explicitly disables protection)
		var finalHandler = handler
		if authMode != "none" && s.config.Auth.Enabled {
			finalHandler = s.protectedPathMiddleware(finalHandler, route.Roles)
		}
		finalHandler = s.applyAuthMiddleware(finalHandler, authMode)

		// Apply CSRF middleware for non-API routes with auth
		// API routes use API keys/bearer tokens, not cookies, so CSRF doesn't apply
		if !isAPI && (authMode == "required" || authMode == "optional") {
			finalHandler = s.csrfMW.Validate(finalHandler)
		}

		// If route has public_dir, wrap with static file fallback
		if route.PublicDir != "" {
			finalHandler = s.createRouteWithStaticFallback(route, finalHandler)
		}

		s.mux.Handle(route.Path, finalHandler)
		// For API routes, also register with trailing slash to handle sub-paths (e.g., /api/todos/123)
		if isAPI && !strings.HasSuffix(route.Path, "/") {
			s.mux.Handle(route.Path+"/", finalHandler)
		}
	}

	// Create fallback handler for "/" that serves:
	// 1. Static files from public_dir (if file exists)
	// 2. Root route handler (if configured)
	// 3. 404
	s.mux.Handle("/", s.createRootHandler())

	return nil
}

// isAPIRoute determines whether the route should be handled as an API module.
func isAPIRoute(route config.Route) bool {
	if strings.EqualFold(route.Type, "api") {
		return true
	}

	path := strings.TrimSuffix(route.Path, "/")
	if path == "" {
		path = "/"
	}

	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return true
	}

	return false
}

// guardProtectedPath enforces auth.protected_paths for a handler registered
// directly on the mux. Routes that go through the site handler get this from
// siteHandler; handlers registered on the mux would otherwise be reachable on
// a site that protects everything with auth.protected_paths: ["/"].
//
// It must run inside applyAuthMiddleware, which is what puts the user on the
// request.
func (s *Server) guardProtectedPath(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pp := s.isProtectedPath(r.URL.Path); pp != nil {
			user := auth.GetUser(r)
			if user == nil {
				s.handleUnauthenticated(w, r)
				return
			}
			if len(pp.Roles) > 0 && !sliceContains(pp.Roles, user.Role) {
				s.handleForbidden(w, r)
				return
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// applyAuthMiddleware wraps a handler with appropriate auth middleware.
// If authMode is "none", auth is explicitly disabled even for protected paths.
func (s *Server) applyAuthMiddleware(handler http.Handler, authMode string) http.Handler {
	// "none" explicitly disables auth for this route, even if under a protected path
	if authMode == "none" {
		// Still apply optional auth so user info is available if logged in
		if s.authMW != nil {
			return s.authMW.OptionalAuth(handler)
		}
		return handler
	}

	if s.authMW == nil {
		return handler
	}
	switch authMode {
	case "required":
		return s.authMW.RequireAuth(handler)
	case "optional":
		return s.authMW.OptionalAuth(handler)
	default:
		return s.authMW.OptionalAuth(handler)
	}
}

// protectedPathMiddleware checks if the request path is protected and enforces auth.
// routeRoles are explicit role requirements from the route config.
func (s *Server) protectedPathMiddleware(next http.Handler, routeRoles []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is protected by config
		pp := s.isProtectedPath(r.URL.Path)

		// Determine required roles (route config takes precedence)
		var requiredRoles []string
		if len(routeRoles) > 0 {
			requiredRoles = routeRoles
		} else if pp != nil {
			requiredRoles = pp.Roles
		}

		// If not protected and no route roles, continue
		if pp == nil && len(routeRoles) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Path is protected - check auth
		user := auth.GetUser(r)
		if user == nil {
			s.handleUnauthenticated(w, r)
			return
		}

		// Check role requirements
		if len(requiredRoles) > 0 && !sliceContains(requiredRoles, user.Role) {
			s.handleForbidden(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleUnauthenticated handles requests to protected paths from unauthenticated users.
func (s *Server) handleUnauthenticated(w http.ResponseWriter, r *http.Request) {
	if isAPIRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "HTTP-401",
				"message": "Unauthorized",
			},
		})
		return
	}

	// HTML request - redirect to login
	loginPath := s.getLoginPath()
	nextURL := r.URL.Path
	if r.URL.RawQuery != "" {
		nextURL += "?" + r.URL.RawQuery
	}
	redirectURL := loginPath + "?next=" + nextURL
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// handleForbidden handles requests from authenticated users without sufficient role.
func (s *Server) handleForbidden(w http.ResponseWriter, r *http.Request) {
	if isAPIRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "HTTP-403",
				"message": "Forbidden: insufficient role",
			},
		})
		return
	}

	// Try the error page (built-in, or a custom one from error_pages)
	if !s.renderPreludeError(w, r, http.StatusForbidden, nil) {
		// Fallback to plain text
		http.Error(w, "403 Forbidden", http.StatusForbidden)
	}
}

// isAPIRequest checks if a request expects JSON response.
func isAPIRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json")
}

// createRouteWithStaticFallback wraps a route handler with static file fallback.
// For a route like /admin with public_dir ./admin/public:
// - /admin/styles.css will try the handler, then ./admin/public/styles.css
func (s *Server) createRouteWithStaticFallback(route config.Route, handler http.Handler) http.Handler {
	routePath := strings.TrimSuffix(route.Path, "/")
	staticRoot := route.PublicDir

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For exact route path or paths that would be handled by sub-routes, use handler
		// Static files are for paths under this route that don't match other routes
		urlPath := r.URL.Path

		// Strip route prefix to get the file path within public_dir
		relativePath := strings.TrimPrefix(urlPath, routePath)
		if relativePath == "" {
			relativePath = "/"
		}

		// Try static file first (if not the route root itself)
		if relativePath != "/" && staticRoot != "" {
			filePath := filepath.Join(staticRoot, relativePath)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				serveStaticFile(w, r, filePath, s.devLog != nil)
				return
			}
		}

		// Fall back to route handler
		handler.ServeHTTP(w, r)
	})
}

// createRootHandler creates a handler that serves static files with route fallback
func (s *Server) createRootHandler() http.Handler {
	// Determine static file root - prefer route's public_dir, fall back to static config
	var staticRoot string
	var rootRoute *config.Route

	// Find root route to get its public_dir
	for i := range s.config.Routes {
		if s.config.Routes[i].Path == "/" {
			rootRoute = &s.config.Routes[i]
			staticRoot = rootRoute.PublicDir
			break
		}
	}

	// Fall back to explicit static route config if no route public_dir
	if staticRoot == "" {
		for _, static := range s.config.Static {
			if static.Path == "/" && static.Root != "" {
				staticRoot = static.Root
				break
			}
		}
	}

	// Find root route handler if configured
	var rootHandler http.Handler
	if rootRoute != nil {
		handler, err := newParsleyHandler(s, *rootRoute, s.scriptCache)
		if err == nil {
			// Pin the release being set up (see SwapRelease).
			handler.cfg = s.config
			handler.assetBundle = s.assetBundle
			rootHandler = s.applyAuthMiddleware(handler, rootRoute.Auth)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try static file first (if configured and file exists)
		if staticRoot != "" && r.URL.Path != "/" {
			filePath := filepath.Join(staticRoot, r.URL.Path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				serveStaticFile(w, r, filePath, s.devLog != nil)
				return
			}
		}

		// Fall back to root route handler
		if rootHandler != nil {
			rootHandler.ServeHTTP(w, r)
			return
		}

		// No handler - 404
		s.handle404(w, r)
	})
}

// ReloadScripts clears the script cache, response cache, and fragment cache,
// forcing all scripts to be re-parsed and responses to be regenerated.
// This is useful for production deployments when scripts are updated.
// In dev mode, this also triggers browser reload via the live reload mechanism.
func (s *Server) ReloadScripts() {
	s.scriptCache.clear()
	s.responseCache.Clear()
	s.fragmentCache.Clear()
	s.assetRegistry.Clear()
	s.imageRegistry.Clear()
	evaluator.ClearModuleCache() // Clear cached modules that may hold stale DB connections
	// Rebuild asset bundle
	if s.assetBundle != nil {
		if err := s.assetBundle.Rebuild(); err != nil {
			s.logWarn("failed to rebuild asset bundle: %v", err)
		}
	}
	// Trigger browser reload if watcher is active (dev mode)
	if s.watcher != nil {
		s.watcher.TriggerReload()
	}
	s.logInfo("caches cleared - scripts will be re-parsed on next request")
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	// Log version first
	fmt.Fprintf(s.stdout, "basil %s\n", s.version)

	// Snapshot the startup config once: the watchers started below can
	// trigger SwapRelease, which rewrites s.config concurrently with the
	// rest of this setup. Everything configured here (listener, middleware)
	// is startup-bound anyway - a swap carries these settings and warns.
	cfg := s.liveConfig()
	configPath := s.configPath

	addr := s.listenAddr()

	// Ensure databases are closed on shutdown
	if s.authDB != nil {
		defer func() {
			s.logInfo("closing auth database connection")
			s.authDB.Close()
		}()
	}
	if s.db != nil {
		defer func() {
			s.logInfo("closing database connection")
			s.db.Close()
		}()
	}

	// Close any Postgres/MySQL connections cached by Parsley handlers
	defer func() {
		s.logInfo("closing cached evaluator database connections")
		evaluator.ClearDBConnections()
	}()

	// In dev mode, start file watcher for hot reload
	if cfg.Server.Dev {
		watcher, err := NewWatcher(s, configPath, s.stdout, s.stderr)
		if err != nil {
			s.logError("failed to create watcher: %v", err)
		} else {
			// SwapRelease reads s.watcher under swapMu, and a SIGHUP (or a
			// deploy noticed by the current-link watcher) can trigger a swap
			// before Run reaches this line — so the write takes the same
			// lock, or it races with that read.
			s.swapMu.Lock()
			s.watcher = watcher
			s.swapMu.Unlock()
			if err := s.watcher.Start(ctx); err != nil {
				s.logError("failed to start watcher: %v", err)
			}
			defer s.watcher.Close()
		}
	}

	// In the site-root layout, watch for `current` being re-pointed by the
	// deploy CLI (a separate process) and activate the new release in place.
	// This is the cross-process activation channel, so it runs in production
	// as well as dev; the dev Watcher above remains the file-level reloader.
	if cfg.SiteRoot != "" {
		clw, err := newCurrentLinkWatcher(s)
		if err != nil {
			s.logError("failed to watch %s for deploys: %v (releases will activate on restart or SIGHUP)", config.CurrentLinkName, err)
		} else {
			clw.Start(ctx)
			defer clw.Close()
			s.logInfo("watching %s for deploys", filepath.Join(cfg.SiteRoot, config.CurrentLinkName))
		}
	}

	// Build handler chain. The chain wraps the serving indirection, not the
	// mux itself: SwapRelease publishes a rebuilt mux through s.serving, and
	// a snapshot of the mux here would pin the release forever.
	handler := s.servingHandler()

	// In dev mode, inject live reload script into HTML responses
	if cfg.Server.Dev {
		handler = injectLiveReload(handler)
	}

	// Add proxy header handling (must be before logging to get real IPs)
	handler = newProxyAware(handler, cfg.Server.Proxy)

	// Add security headers
	handler = newSecurityHeaders(handler, cfg.Security, cfg.Server.Dev)

	// Add CORS middleware if configured
	if s.corsMW != nil {
		handler = s.corsMW.Handler(handler)
	}

	// Wrap with request logging middleware (unless level is error-only)
	if cfg.Logging.Level != "error" {
		handler = newRequestLogger(handler, s.stdout, cfg.Logging.Format)
	}

	// Wrap with compression (compresses all responses)
	handler = newCompressionHandler(handler, cfg.Compression)

	// Wrap with panic recovery (outermost - guards every other middleware so a
	// panic becomes a logged 500 rather than a dropped connection)
	handler = newRecoverMiddleware(handler, s.stderr, cfg.Server.Dev)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if cfg.Server.Dev {
			fmt.Fprintf(s.stdout, "Starting Basil in development mode on http://%s\n", addr)
			errCh <- s.server.ListenAndServe()
		} else {
			fmt.Fprintf(s.stdout, "Starting Basil on https://%s\n", addr)
			errCh <- s.listenAndServeTLS()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		fmt.Fprintf(s.stdout, "\nShutting down gracefully...\n")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

// listenAddr returns the address to listen on based on configuration.
//
// The listener uses server.bind, never server.host. server.host is the
// public hostname - the certificate name and the address people type - and
// on NAT, Docker, or behind a load balancer it is not an address this
// machine owns, so binding it fails outright. Empty bind means all
// interfaces, which is what the port-80 ACME/redirect listener has always
// used. Dev mode still defaults to localhost so a development server is not
// exposed to the network by accident.
func (s *Server) listenAddr() string {
	cfg := s.liveConfig()
	bind := cfg.Server.Bind
	port := cfg.Server.Port

	if cfg.Server.Dev {
		if bind == "" {
			bind = "localhost"
		}
		if port == 443 {
			port = 8080
		}
	}

	return fmt.Sprintf("%s:%d", bind, port)
}

// listenAndServeTLS starts HTTPS server with TLS.
// Supports automatic Let's Encrypt certificates or manual certificate files.
func (s *Server) listenAndServeTLS() error {
	cfg := s.liveConfig().Server.HTTPS

	// Manual cert mode. Paths are anchored to the data root by the config
	// loader, so they do not move with the operator's shell.
	if cfg.Cert != "" && cfg.Key != "" {
		s.logInfo("using manual TLS certificates (%s)", cfg.Cert)
		return s.server.ListenAndServeTLS(cfg.Cert, cfg.Key)
	}

	// Auto cert mode using Let's Encrypt
	if !cfg.Auto {
		return fmt.Errorf("HTTPS requires either auto: true or cert/key paths")
	}

	return s.listenAndServeAutocert()
}

// certCacheDir returns the certificate cache directory. The config loader
// anchors https.cache_dir to the data root; this only covers configs built
// in code (tests) that never went through the loader.
func (s *Server) certCacheDir() string {
	cfg := s.liveConfig()
	dir := cfg.Server.HTTPS.CacheDir
	if dir == "" {
		dir = config.CertsDirName
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cfg.DataDir, dir)
	}
	return dir
}

// listenAndServeAutocert configures and starts the server with Let's Encrypt certificates.
func (s *Server) listenAndServeAutocert() error {
	cfg := s.liveConfig().Server.HTTPS

	// Certificates are persistent state: re-issuance is rate-limited by
	// Let's Encrypt, so the cache must survive a deploy and must not depend
	// on the directory the operator happened to be standing in.
	cacheDir := s.certCacheDir()
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("creating certificate cache %s: %w", cacheDir, err)
	}

	// Create autocert manager
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: s.hostPolicy(),
	}

	if cfg.Email != "" {
		manager.Email = cfg.Email
	}

	// Configure TLS
	s.server.TLSConfig = &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"}, // Enable HTTP/2
		MinVersion:     tls.VersionTLS12,
	}

	// Start HTTP redirect server on port 80 for ACME challenges and redirects
	ready := make(chan struct{})
	go s.runHTTPRedirect(manager, ready)

	s.logInfo("automatic TLS enabled via Let's Encrypt (cache: %s)", cacheDir)

	// Obtain the certificate now rather than on the first TLS handshake.
	// Left to autocert's lazy issuance, the developer's first `git clone` is
	// the request that triggers it, and an ACME failure surfaces as an
	// opaque TLS error with no clue whether DNS, port 80 or Basil is at
	// fault. (DESIGN-git-deploy 5.1.2)
	go s.obtainCertificate(manager, ready)

	// ListenAndServeTLS with empty cert/key uses TLSConfig
	return s.server.ListenAndServeTLS("", "")
}

// certificateProbeTimeout bounds the eager issuance attempt at startup.
var certificateProbeTimeout = 90 * time.Second

// certificateFailureCooldown is how long a failed startup probe suppresses
// the next one. Let's Encrypt caps failed authorizations at 5 per account per
// hostname per hour, so a server under `Restart=always` with broken DNS or a
// blocked port 80 - exactly the state this probe exists to diagnose - would
// otherwise spend that budget in seconds.
var certificateFailureCooldown = 15 * time.Minute

// certFailureMarker names the file that records the last failed probe.
func (s *Server) certFailureMarker() string {
	dataDir := s.liveConfig().DataDir
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, ".acme-probe-failed")
}

// recentCertificateFailure reports whether the previous start failed to
// obtain a certificate recently enough that asking again would just spend
// rate limit, and how long ago that was.
func (s *Server) recentCertificateFailure() (time.Duration, bool) {
	marker := s.certFailureMarker()
	if marker == "" {
		return 0, false
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		return 0, false
	}
	when, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	since := time.Since(when)
	if since < 0 || since > certificateFailureCooldown {
		return 0, false
	}
	return since, true
}

func (s *Server) recordCertificateFailure() {
	if marker := s.certFailureMarker(); marker != "" {
		os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0600)
	}
}

func (s *Server) clearCertificateFailure() {
	if marker := s.certFailureMarker(); marker != "" {
		os.Remove(marker)
	}
}

// certificateCached reports whether the autocert cache already holds a
// certificate for host. A cached certificate is renewed by autocert on its
// own schedule, so there is nothing for the startup probe to ask about, and
// asking would issue a request on every restart.
func (s *Server) certificateCached(manager *autocert.Manager, host string) bool {
	if manager.Cache == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := manager.Cache.Get(ctx, host)
	return err == nil
}

// obtainCertificate asks autocert for the configured host's certificate at
// startup and reports the outcome plainly, naming the two things that
// usually break: DNS and port 80.
func (s *Server) obtainCertificate(manager *autocert.Manager, ready <-chan struct{}) {
	host := s.liveConfig().Server.Host
	if host == "" {
		return // refused at config validation; nothing to ask for
	}

	if s.certificateCached(manager, host) {
		s.logInfo("TLS certificate for %s is already in the cache", host)
		return
	}

	s.certificateProbe(host, ready, manager.GetCertificate)
}

// certificateProbe is obtainCertificate without the autocert manager, so the
// cooldown, the diagnosis and the timeout path can be tested without asking
// a real ACME server for anything.
func (s *Server) certificateProbe(host string, ready <-chan struct{}, get func(*tls.ClientHelloInfo) (*tls.Certificate, error)) {
	if since, recent := s.recentCertificateFailure(); recent {
		s.logError("not asking for a TLS certificate for %s: the last attempt failed %s ago", host, since.Round(time.Second))
		s.logError("  check that %s resolves to this machine (DNS), and that port 80 is reachable from the internet for the ACME challenge", host)
		s.logError("  Let's Encrypt rate-limits failed attempts; the next one is allowed %s after the last failure, or on the first HTTPS request", certificateFailureCooldown)
		return
	}

	// The ACME HTTP-01 challenge is answered on port 80, so wait for that
	// listener before asking.
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
	}

	err := s.probeCertificate(host, get)
	if err != nil {
		s.recordCertificateFailure()
		s.logError("could not obtain a TLS certificate for %s: %v", host, err)
		s.logError("  check that %s resolves to this machine (DNS), and that port 80 is reachable from the internet for the ACME challenge", host)
		s.logError("  the server is running; it will retry on the first HTTPS request")
		return
	}
	s.clearCertificateFailure()
	s.logInfo("TLS certificate ready for %s", host)
}

// probeCertificate asks for host's certificate, bounded by
// certificateProbeTimeout. get is manager.GetCertificate in production and a
// stub in tests.
func (s *Server) probeCertificate(host string, get func(*tls.ClientHelloInfo) (*tls.Certificate, error)) error {
	ctx, cancel := context.WithTimeout(context.Background(), certificateProbeTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Describe a modern client so autocert issues the same (ECDSA)
		// certificate a browser handshake would ask for. Asking as a
		// different client would issue a second certificate later and
		// spend the rate limit twice.
		hello := &tls.ClientHelloInfo{
			ServerName:        host,
			SupportedVersions: []uint16{tls.VersionTLS13, tls.VersionTLS12},
			SignatureSchemes:  []tls.SignatureScheme{tls.ECDSAWithP256AndSHA256},
			SupportedCurves:   []tls.CurveID{tls.X25519, tls.CurveP256},
			SupportedPoints:   []uint8{0},
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}
		_, err := get(hello)
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// The inner goroutine is left running: GetCertificate takes no
		// context, so there is nothing to cancel. It ends when the ACME
		// call does, and writes to a buffered channel nobody reads.
		return fmt.Errorf("timed out after %s", certificateProbeTimeout)
	}
}

// hostPolicy returns a function that validates hostnames for certificate requests.
func (s *Server) hostPolicy() autocert.HostPolicy {
	host := s.liveConfig().Server.Host

	// No host configured: refuse every issuance request. Returning nil here
	// would tell autocert to attempt issuance for any hostname a stranger
	// supplies in SNI, which burns the site's Let's Encrypt rate limit from
	// outside. Config validation refuses to start a public server without
	// server.host; this is the belt to that pair of braces.
	if host == "" {
		return func(_ context.Context, name string) error {
			return fmt.Errorf("certificate request for %q refused: server.host is not configured", name)
		}
	}

	// Allow only the configured host
	return autocert.HostWhitelist(host)
}

// httpRedirectHandler builds the plain-HTTP (port 80) handler: ACME HTTP-01
// challenges at /.well-known/acme-challenge/ go to the autocert manager, and
// everything else — Git paths included — is redirected to HTTPS. The Git
// plain-HTTP refusal lives in the Git handler on the main listener, NOT
// here: a blanket refusal on this listener would swallow the ACME challenge
// and the server could never obtain or renew a certificate.
func httpRedirectHandler(manager *autocert.Manager) http.Handler {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build HTTPS URL
		target := "https://" + r.Host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	// autocert's handler passes ACME challenges to the manager and delegates
	// everything else to the redirect handler.
	return manager.HTTPHandler(redirectHandler)
}

// runHTTPRedirect starts an HTTP server on port 80 that:
// 1. Handles ACME HTTP-01 challenges for Let's Encrypt
// 2. Redirects all other requests to HTTPS
func (s *Server) runHTTPRedirect(manager *autocert.Manager, ready chan struct{}) {
	httpServer := &http.Server{
		Addr:              ":80",
		Handler:           httpRedirectHandler(manager),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		s.logError("HTTP redirect server cannot listen on port 80: %v", err)
		s.logError("  port 80 must be free for the ACME challenge, or no certificate can be obtained or renewed")
		close(ready)
		return
	}
	close(ready)

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		s.logError("HTTP redirect server error: %v", err)
	}
}

// initPreludeDevMode enables live reloading of prelude files from disk
// if BASIL_PRELUDE_PATH is set. This allows editing devtools styling
// without recompiling the server.
//
// Usage: BASIL_PRELUDE_PATH=/path/to/basil/server/prelude ./basil ...
func (s *Server) initPreludeDevMode() {
	preludePath := os.Getenv("BASIL_PRELUDE_PATH")
	if preludePath == "" {
		return
	}

	// Verify the path exists
	if info, err := os.Stat(preludePath); err != nil || !info.IsDir() {
		s.logWarn("BASIL_PRELUDE_PATH=%s does not exist or is not a directory", preludePath)
		return
	}

	EnablePreludeDevMode(preludePath)
	s.logInfo("prelude dev mode: live reload enabled from %s", preludePath)
}
