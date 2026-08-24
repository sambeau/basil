package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// Live release activation (FEAT-153). A deploy re-points <site root>/current
// at a new release directory; SwapRelease rebuilds the request-serving
// surface against it and publishes the result atomically. Requests that are
// already executing finish on the handlers they entered - handlers capture
// their release paths, caches and config at construction - while every
// request dispatched after the swap sees the new release.

// serveState is what the request path dispatches through: the mux serving
// the active release, together with the config and asset bundle it was built
// against. Run's middleware chain wraps an indirection over Server.serving
// rather than the mux itself, so storing a new state here is the whole
// visible act of activation. config and assetBundle live here - not read
// from the Server's plain fields - because SwapRelease rewrites those fields
// under swapMu, which request-path readers never take.
type serveState struct {
	mux         *http.ServeMux
	release     string // ReleaseDir this mux was built against ("" in tests without one)
	config      *config.Config
	assetBundle *AssetBundle
}

// liveConfig is the config of the release currently being served, for code
// that runs concurrently with SwapRelease (request paths, Run's background
// goroutines). Handlers that pin a config at construction should prefer the
// pin; everything else must come through here rather than s.config, which is
// rewritten under swapMu. The fallback covers construction time, before New
// publishes the first serveState - single-goroutine, so the plain read is
// safe there.
func (s *Server) liveConfig() *config.Config {
	if st := s.serving.Load(); st != nil {
		return st.config
	}
	return s.config
}

// liveBundle is liveConfig for the asset bundle.
func (s *Server) liveBundle() *AssetBundle {
	if st := s.serving.Load(); st != nil {
		return st.assetBundle
	}
	return s.assetBundle
}

// servingHandler returns the handler the middleware chain wraps: an
// indirection through s.serving. The pointer is read once per request, so a
// request is routed entirely by one release's mux even if a swap lands while
// it is being served.
func (s *Server) servingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.serving.Load().mux.ServeHTTP(w, r)
	})
}

// SwapRelease re-resolves <site root>/current and rebuilds the serving
// surface against the release it names: new config, new mux and handlers, a
// new asset bundle, and cleared script/response/fragment caches, asset and
// image registries and module cache (the same list ReloadScripts covers).
// The rebuilt mux is published atomically, so requests in flight complete
// against the release they started on - every handler pins its release
// paths, config and caches at construction - and new requests see the new
// one.
//
// Only config, configPath, mux and assetBundle are replaced; the caches and
// registries are shared instances cleared in place through their own locks.
// Handlers built by setupRoutes pin the values current while the swap holds
// swapMu, which is what keeps the replacement of these fields off the
// request path.
//
// Listener-affecting settings (server.port, server.bind, server.host,
// server.https) are not applied: the listener was bound at startup and
// cannot be re-bound live. The same goes for every config section whose
// subsystem is built once by New and not rebuilt here (database, session,
// auth, git, images, logging, compression, security, CORS, proxy). A change
// is reported with a restart-required warning and the running values are
// kept, so the served config never disagrees with the subsystems behind it.
//
// On any failure the previous release keeps serving: nothing is published,
// no cache is cleared, and the replaced fields are restored.
func (s *Server) SwapRelease() error {
	s.swapMu.Lock()
	defer s.swapMu.Unlock()

	siteRoot := s.config.SiteRoot
	if siteRoot == "" {
		return fmt.Errorf("swap release: %s is not inside a site-root layout (no %s/ directory and %s link); restart the server to pick up changes", s.config.ReleaseDir, config.ReleasesDirName, config.CurrentLinkName)
	}

	releaseDir, err := deploy.CurrentRelease(siteRoot)
	if err != nil {
		return fmt.Errorf("swap release: resolving active release: %w", err)
	}

	cfgPath := filepath.Join(releaseDir, config.ConfigFileName)
	newCfg, newCfgPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		return fmt.Errorf("swap release: %w", err)
	}
	// Carry the listener settings (and the --dev flag, which never comes
	// from yaml) and every restart-required section before validating:
	// validation rules depend on dev mode, and what gets validated must be
	// what will actually run.
	s.carryRestartRequiredSettings(newCfg)
	if err := config.Validate(newCfg); err != nil {
		return fmt.Errorf("swap release: validating %s: %w", cfgPath, err)
	}

	// Build the new release's asset bundle before touching server state.
	newBundle := buildAssetBundle(newCfg, s.logWarn)

	// New generations for the response and fragment caches BEFORE the new
	// handlers are built: handlers pin their generation at construction, so
	// the new release's entries can never collide with writes still arriving
	// from old-release requests. On failure the burned generation is
	// harmless - the old handlers keep reading and writing their own pinned
	// generation.
	s.responseCache.Advance()
	s.fragmentCache.Advance()

	// Rebuild the routes. setupRoutes reads these fields, so they are
	// swapped in first and restored if it fails; nothing is published until
	// the end, and request paths never read these fields directly (they go
	// through s.serving or a handler pin), so the transient values - a
	// failed release's config included - are invisible to every reader.
	prevConfig, prevConfigPath, prevMux, prevBundle := s.config, s.configPath, s.mux, s.assetBundle
	s.config = newCfg
	s.configPath = newCfgPath
	s.mux = http.NewServeMux()
	s.assetBundle = newBundle
	if err := s.setupRoutes(); err != nil {
		s.config, s.configPath, s.mux, s.assetBundle = prevConfig, prevConfigPath, prevMux, prevBundle
		return fmt.Errorf("swap release: rebuilding routes: %w", err)
	}

	// The new release is good: clear everything cached from the old one.
	// Cache keys are absolute paths under the old release directory or
	// content the old release rendered; cached modules may also hold paths
	// and DB handles from it.
	s.scriptCache.clear()
	s.responseCache.Clear()
	s.fragmentCache.Clear()
	s.assetRegistry.Clear()
	s.imageRegistry.Clear()
	evaluator.ClearModuleCache()

	s.serving.Store(&serveState{mux: s.mux, release: releaseDir, config: newCfg, assetBundle: newBundle})
	s.logInfo("activated release %s", filepath.Base(releaseDir))

	// Trigger browser reload if the dev watcher is active.
	if s.watcher != nil {
		s.watcher.TriggerReload()
	}
	return nil
}

// carryListenerSettings keeps the running listener's settings in newCfg,
// warning about each one the new release tried to change. The listener was
// bound before the first request and cannot be re-bound by a swap.
func (s *Server) carryListenerSettings(newCfg *config.Config) {
	old := s.config.Server
	warn := func(name string) {
		s.logWarn("%s changed in the new release but the listener cannot be re-bound live - restart required for it to take effect", name)
	}
	if newCfg.Server.Port != old.Port {
		warn("server.port")
	}
	if newCfg.Server.Bind != old.Bind {
		warn("server.bind")
	}
	if newCfg.Server.Host != old.Host {
		warn("server.host")
	}
	if newCfg.Server.HTTPS != old.HTTPS {
		warn("server.https")
	}
	newCfg.Server.Port = old.Port
	newCfg.Server.Bind = old.Bind
	newCfg.Server.Host = old.Host
	newCfg.Server.HTTPS = old.HTTPS
	newCfg.Server.Dev = old.Dev
}

// carryRestartRequiredSettings keeps the running values of every config
// section whose subsystem New builds once and SwapRelease does not rebuild:
// the database connection, session store, auth system, git server, image
// registry, logging/compression/security-header/proxy middleware and CORS
// were all constructed from the startup config, so applying a release's new
// values to s.config alone would make the served config lie about the
// subsystem behind it. Each changed section is kept at its running value
// with one restart-required warning.
func (s *Server) carryRestartRequiredSettings(newCfg *config.Config) {
	s.carryListenerSettings(newCfg)

	old := s.config
	carry := func(name string, changed bool, keep func()) {
		if changed {
			s.logWarn("%s changed in the new release but its subsystem was built at startup - restart required for it to take effect", name)
		}
		keep()
	}

	carry("database", !reflect.DeepEqual(newCfg.Database, old.Database), func() { newCfg.Database = old.Database })
	carry("session", !reflect.DeepEqual(newCfg.Session, old.Session), func() { newCfg.Session = old.Session })
	carry("auth", !reflect.DeepEqual(newCfg.Auth, old.Auth), func() { newCfg.Auth = old.Auth })
	carry("git", !reflect.DeepEqual(newCfg.Git, old.Git), func() { newCfg.Git = old.Git })
	carry("images", !reflect.DeepEqual(newCfg.Images, old.Images), func() { newCfg.Images = old.Images })
	carry("logging", !reflect.DeepEqual(newCfg.Logging, old.Logging), func() { newCfg.Logging = old.Logging })
	carry("compression", !reflect.DeepEqual(newCfg.Compression, old.Compression), func() { newCfg.Compression = old.Compression })
	carry("security", !reflect.DeepEqual(newCfg.Security, old.Security), func() { newCfg.Security = old.Security })
	carry("cors", !reflect.DeepEqual(newCfg.CORS, old.CORS), func() { newCfg.CORS = old.CORS })
	carry("server.proxy", !reflect.DeepEqual(newCfg.Server.Proxy, old.Server.Proxy), func() { newCfg.Server.Proxy = old.Server.Proxy })
}

// currentLinkDebounce coalesces the burst of filesystem events a symlink
// swap produces (create of the temp link, rename over `current`) into one
// activation.
const currentLinkDebounce = 100 * time.Millisecond

// currentLinkWatcher watches the site root for `current` being re-pointed
// and activates the new release in the running server. The deploy CLI runs
// in a separate process, so this watcher is the only way a running server
// learns about a deploy; it runs in production as well as dev. It is
// deliberately separate from the dev Watcher, which watches individual
// source files for hot reload.
type currentLinkWatcher struct {
	server *Server
	fw     *fsnotify.Watcher
	done   chan struct{}
}

// newCurrentLinkWatcher creates a watcher on the server's site root. The
// caller must be in the site-root layout.
func newCurrentLinkWatcher(s *Server) (*currentLinkWatcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fw.Add(s.config.SiteRoot); err != nil {
		fw.Close()
		return nil, err
	}
	return &currentLinkWatcher{server: s, fw: fw, done: make(chan struct{})}, nil
}

// Start begins watching. The loop stops when ctx is cancelled or the
// watcher is closed.
func (w *currentLinkWatcher) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *currentLinkWatcher) loop(ctx context.Context) {
	defer close(w.done)

	// The debounce timer starts on the first interesting event and is pushed
	// back by each following one; the swap runs when it fires.
	var timer *time.Timer
	var fire <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) != config.CurrentLinkName {
				continue
			}
			// A symlink swap surfaces as Create (rename target); Rename,
			// Remove and Write cover cruder ways of re-pointing the link
			// (ln -sfn, rm + ln). Ignore Chmod noise.
			if !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Rename) && !ev.Has(fsnotify.Remove) && !ev.Has(fsnotify.Write) {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(currentLinkDebounce)
				fire = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(currentLinkDebounce)
			}

		case <-fire:
			timer = nil
			fire = nil
			w.server.logInfo("%s re-pointed - activating new release", config.CurrentLinkName)
			if err := w.server.SwapRelease(); err != nil {
				// A failed swap must be loud: the operator just deployed
				// something and the server is still serving the old release.
				w.server.logError("release activation FAILED: %v - still serving the previous release", err)
			}

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.server.logError("release watcher: %v", err)
		}
	}
}

// Close stops the watcher and waits for its loop to exit.
func (w *currentLinkWatcher) Close() error {
	err := w.fw.Close()
	<-w.done
	return err
}
