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
// auth, git, images, logging, compression, security, CORS, proxy) and for
// data_dir, the anchor all of their persistent paths were resolved against.
// A change is reported with a restart-required warning and the running values
// are kept, so the served config never disagrees with the subsystems behind
// it.
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
	// What loading the new release decided to ignore — an operator-owned
	// setting it tried to change, a key that no longer exists — is reported
	// on every swap, not only at startup: a deploy is exactly when a release
	// introduces one.
	for _, w := range config.ReleaseWarnings(newCfg) {
		s.logWarn("%s", w)
	}

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

	s.serving.Store(&serveState{mux: s.mux, config: newCfg, assetBundle: newBundle})
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
// subsystem behind it. data_dir joins them as the anchor those subsystems
// resolved their paths against. Each changed section is kept at its running
// value with one restart-required warning.
func (s *Server) carryRestartRequiredSettings(newCfg *config.Config) {
	s.carryListenerSettings(newCfg)

	old := s.config
	carry := func(name string, changed bool, keep func()) {
		if changed {
			s.logWarn("%s changed in the new release but its subsystem was built at startup - restart required for it to take effect", name)
		}
		keep()
	}

	// data_dir is deliberately absent from this list. It is an ANCHOR, not a
	// section, and FEAT-157 moved its defence to where anchors are decided:
	// on a site root the load pins it to <site root>/data and warns
	// (config.ResolveAnchors), so both configs here have already resolved it
	// identically — and SwapRelease returns early off site roots, where the
	// key is the operator speaking and must be honoured. A carry() entry
	// could only ever fire on a difference that cannot occur.

	carry("database", !reflect.DeepEqual(newCfg.Database, old.Database), func() { newCfg.Database = old.Database })
	carry("session", !reflect.DeepEqual(newCfg.Session, old.Session), func() { newCfg.Session = old.Session })
	carry("auth", !reflect.DeepEqual(newCfg.Auth, old.Auth), func() { newCfg.Auth = old.Auth })
	carry("dev", !reflect.DeepEqual(newCfg.Dev, old.Dev), func() { newCfg.Dev = old.Dev })
	carry("images", !reflect.DeepEqual(newCfg.Images, old.Images), func() { newCfg.Images = old.Images })
	carry("logging", !reflect.DeepEqual(newCfg.Logging, old.Logging), func() { newCfg.Logging = old.Logging })
	carry("compression", !reflect.DeepEqual(newCfg.Compression, old.Compression), func() { newCfg.Compression = old.Compression })
	carry("security", !reflect.DeepEqual(newCfg.Security, old.Security), func() { newCfg.Security = old.Security })
	carry("cors", !reflect.DeepEqual(newCfg.CORS, old.CORS), func() { newCfg.CORS = old.CORS })
	carry("server.proxy", !reflect.DeepEqual(newCfg.Server.Proxy, old.Server.Proxy), func() { newCfg.Server.Proxy = old.Server.Proxy })

	// deploy.* is deliberately absent from this list: this process builds
	// nothing from it. The receive hooks and the CLI read deploy.keep out of
	// the active release's own file at every invocation, so pinning a stale
	// copy here would make the served config disagree with what the next push
	// will actually do — the opposite of what carrying is for.

	// The git endpoint is not a config section any more (FEAT-157): its
	// switch is basil.gitEnabled in site.git, which no deploy can move. It
	// still gets the restart-required warning, because the handler was built
	// at startup and an operator who flips the switch should not think a
	// deploy applied it.
	s.warnGitSwitchChanged()
}

// warnGitSwitchChanged reports an operator flipping basil.gitEnabled while
// the server runs. Silent when there is no repository to ask, and never
// changes what is served: the endpoint the handler was built with stands
// until a restart.
func (s *Server) warnGitSwitchChanged() {
	repo := s.config.BareRepoPath()
	if repo == "" {
		return
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return
	}
	now, err := deploy.GitEnabled(repo)
	if err != nil {
		s.logWarn("git: %v", err)
	}
	if now != s.gitSwitch {
		s.logWarn("basil.gitEnabled is now %v in %s but the git endpoint was built at startup - restart required for it to take effect", now, repo)
	}
}

// currentLinkDebounce coalesces the burst of filesystem events a symlink
// swap produces (create of the temp link, rename over `current`) into one
// check of the link.
const currentLinkDebounce = 100 * time.Millisecond

// currentLinkPoll is how often the link is re-read regardless of filesystem
// events, because on some platforms a deploy produces no event for it at all.
// macOS is the case in hand: fsnotify's kqueue backend reports directory
// changes by re-scanning and diffing the names it finds, and SetCurrent's
// rename of a temp link over `current` leaves that name in place, so nothing
// ever looks new - the temp link is usually gone again before the re-scan
// runs, so even it goes unreported. The per-file watch is no help either,
// since kqueue resolves a symlink and ends up watching the release directory
// rather than the link. inotify does deliver the rename, so on Linux this
// poll is only a backstop behind an activation that has already happened.
const currentLinkPoll = time.Second

// currentLinkWatcher watches the site root for `current` being re-pointed
// and activates the new release in the running server. The deploy CLI runs
// in a separate process, so this watcher is the only way a running server
// learns about a deploy; it runs in production as well as dev. It is
// deliberately separate from the dev Watcher, which watches individual
// source files for hot reload.
//
// Filesystem events are treated as a hint that something may have happened,
// never as the fact itself: every path leads to re-reading the link and
// comparing it with the release being served. That keeps activation correct
// on platforms whose events miss the swap entirely (see currentLinkPoll) and
// idempotent where they arrive in bursts.
type currentLinkWatcher struct {
	server   *Server
	siteRoot string
	fw       *fsnotify.Watcher
	done     chan struct{}

	// failed is the release whose activation failed, skipped until `current`
	// changes again; loop's goroutine owns it. Without it a release that
	// cannot be built would be retried - and its failure logged - on every
	// poll tick.
	failed string
	// linkErr is the last error reported for reading the link, for the same
	// reason: a broken link must not produce a line of log a second.
	linkErr string
}

// newCurrentLinkWatcher creates a watcher on the server's site root. The
// caller must be in the site-root layout.
func newCurrentLinkWatcher(s *Server) (*currentLinkWatcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	siteRoot := s.config.SiteRoot
	if err := fw.Add(siteRoot); err != nil {
		fw.Close()
		return nil, err
	}
	return &currentLinkWatcher{server: s, siteRoot: siteRoot, fw: fw, done: make(chan struct{})}, nil
}

// Start begins watching. The loop stops when ctx is cancelled or the
// watcher is closed.
func (w *currentLinkWatcher) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *currentLinkWatcher) loop(ctx context.Context) {
	defer close(w.done)

	// The debounce timer starts on the first interesting event and is pushed
	// back by each following one; the link is checked when it fires.
	var timer *time.Timer
	var fire <-chan time.Time

	poll := time.NewTicker(currentLinkPoll)
	defer poll.Stop()

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
			w.activateIfRepointed()

		case <-poll.C:
			w.activateIfRepointed()

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.server.logError("release watcher: %v", err)
		}
	}
}

// activateIfRepointed re-reads `current` and activates the release it names
// if that is not the one already being served. Comparing the link against
// the live release - rather than acting on an event, or on a remembered
// target - is what lets the event path and the poll share the work: whichever
// notices a deploy first activates it, and the other then finds nothing to
// do. It is called only from loop's goroutine.
func (w *currentLinkWatcher) activateIfRepointed() {
	target, err := deploy.CurrentRelease(w.siteRoot)
	if err != nil {
		if msg := err.Error(); msg != w.linkErr {
			w.linkErr = msg
			w.server.logError("release watcher: %v - still serving the current release", err)
		}
		return
	}
	w.linkErr = ""
	if target == w.server.liveConfig().ReleaseDir || target == w.failed {
		return
	}

	w.server.logInfo("%s re-pointed to %s - activating new release", config.CurrentLinkName, filepath.Base(target))
	if err := w.server.SwapRelease(); err != nil {
		// A failed swap must be loud: the operator just deployed something
		// and the server is still serving the old release. Record the release
		// so the failure is reported once rather than on every poll tick,
		// until `current` moves somewhere else.
		w.failed = target
		w.server.logError("release activation FAILED: %v - still serving the previous release", err)
		return
	}
	// Forget the failure once anything else has activated: a release that
	// failed is worth another attempt when it is deployed again, since the
	// reason it failed may have been fixed in the meantime.
	w.failed = ""
}

// Close stops the watcher and waits for its loop to exit.
func (w *currentLinkWatcher) Close() error {
	err := w.fw.Close()
	<-w.done
	return err
}
