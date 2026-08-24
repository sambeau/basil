package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// writeRelease materialises a minimal release directory under
// <root>/releases/<id>: a site/index.pars that renders body, and a
// basil.yaml (extraYAML appended) so the release can be activated the way a
// deployed release is - config loaded from inside it.
func writeRelease(t *testing.T, root, id, body, extraYAML string) string {
	t.Helper()
	dir := filepath.Join(root, config.ReleasesDirName, id)
	must(os.MkdirAll(filepath.Join(dir, "site"), 0755))
	must(os.WriteFile(filepath.Join(dir, "site", "index.pars"), []byte(fmt.Sprintf("%q", body)), 0644))
	yaml := "site:\n  path: site\ndev:\n  cache: true\n" + extraYAML
	must(os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte(yaml), 0644))
	return dir
}

// activationFixture builds a real site root - releases/A, releases/B and a
// genuine `current` symlink at releases/A - unlike siteRootFixture, which
// fakes the layout without the link.
func activationFixture(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	writeRelease(t, root, "A", "release A content", "")
	writeRelease(t, root, "B", "release B content", "")
	must(os.MkdirAll(filepath.Join(root, config.DataDirName), 0755))
	must(os.Symlink(filepath.Join(config.ReleasesDirName, "A"), filepath.Join(root, config.CurrentLinkName)))
	return root
}

// newSiteRootServer starts a server the way cmd/basil does for --site: the
// config is found through the `current` symlink and loaded from inside the
// active release.
func newSiteRootServer(t *testing.T, root string) (*Server, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cfgPath, err := config.ConfigPathForSite(root)
	if err != nil {
		t.Fatalf("ConfigPathForSite: %v", err)
	}
	cfg, loadedPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("LoadWithPath: %v", err)
	}
	cfg.Server.Dev = true
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var stdout, stderr bytes.Buffer
	s, err := New(cfg, loadedPath, "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(s.Close)
	return s, &stdout, &stderr
}

// get serves one request through the same indirection Run's middleware
// chain wraps, which is where activation must be visible.
func get(s *Server, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.servingHandler().ServeHTTP(rec, httptest.NewRequest("GET", path, http.NoBody))
	return rec
}

func TestSwapReleaseChangesServedOutput(t *testing.T) {
	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	if got := get(s, "/").Body.String(); !strings.Contains(got, "release A content") {
		t.Fatalf("before swap: expected release A content, got %q", got)
	}

	if err := deploy.SetCurrent(root, filepath.Join(root, config.ReleasesDirName, "B")); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	if err := s.SwapRelease(); err != nil {
		t.Fatalf("SwapRelease: %v", err)
	}

	if got := get(s, "/").Body.String(); !strings.Contains(got, "release B content") {
		t.Errorf("after swap: expected release B content, got %q", got)
	}
	if want := filepath.Join(root, config.ReleasesDirName, "B"); s.config.ReleaseDir != want {
		t.Errorf("after swap: config.ReleaseDir = %q, want %q", s.config.ReleaseDir, want)
	}
}

// The headline activation guarantee: a request that entered its handler
// before the swap completes against the release it started on, while a
// fresh request sees the new release.
func TestSwapReleaseInFlightRequestFinishesOnOldRelease(t *testing.T) {
	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	// Model an in-flight request: a handler on the pre-swap mux that has
	// started running and blocks mid-work, then finishes its job against
	// the handlers of the mux it entered - exactly the position a slow
	// request is in when a deploy lands.
	entered := make(chan struct{})
	unblock := make(chan struct{})
	oldMux := s.mux
	oldMux.HandleFunc("/inflight", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-unblock
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		oldMux.ServeHTTP(w, r2)
	})

	done := make(chan struct{})
	rec := httptest.NewRecorder()
	go func() {
		defer close(done)
		s.servingHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/inflight", http.NoBody))
	}()
	<-entered

	// Swap mid-flight.
	must(deploy.SetCurrent(root, filepath.Join(root, config.ReleasesDirName, "B")))
	if err := s.SwapRelease(); err != nil {
		t.Fatalf("SwapRelease: %v", err)
	}

	// A fresh request sees the new release immediately...
	if got := get(s, "/").Body.String(); !strings.Contains(got, "release B content") {
		t.Errorf("fresh request after swap: expected release B content, got %q", got)
	}
	// ...including on the very path the blocked request entered: the new
	// mux has no crafted /inflight handler, so the site catch-all serves it
	// from release B.
	if got := get(s, "/inflight").Body.String(); !strings.Contains(got, "release B content") {
		t.Errorf("fresh request to /inflight after swap: expected release B content from the new mux, got %q", got)
	}

	// The in-flight request completes with the old release's content.
	close(unblock)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
	if got := rec.Body.String(); !strings.Contains(got, "release A content") {
		t.Errorf("in-flight request: expected release A content, got %q", got)
	}
}

func TestSwapReleaseFailureKeepsOldReleaseServing(t *testing.T) {
	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	// A release whose config does not parse.
	broken := filepath.Join(root, config.ReleasesDirName, "broken")
	must(os.MkdirAll(filepath.Join(broken, "site"), 0755))
	must(os.WriteFile(filepath.Join(broken, "site", "index.pars"), []byte(`"broken release"`), 0644))
	must(os.WriteFile(filepath.Join(broken, config.ConfigFileName), []byte("site: [not: valid: yaml"), 0644))
	must(deploy.SetCurrent(root, broken))

	prevConfig, prevMux, prevState := s.config, s.mux, s.serving.Load()

	err := s.SwapRelease()
	if err == nil {
		t.Fatal("SwapRelease: expected an error for a release with a broken config")
	}

	// State untouched, old release still serving.
	if s.config != prevConfig || s.mux != prevMux || s.serving.Load() != prevState {
		t.Error("failed swap must leave the serving state untouched")
	}
	if got := get(s, "/").Body.String(); !strings.Contains(got, "release A content") {
		t.Errorf("after failed swap: expected release A content, got %q", got)
	}

	// Same again for a config that parses but does not validate.
	invalid := writeRelease(t, root, "invalid", "invalid release", "server:\n  port: -1\n")
	must(deploy.SetCurrent(root, invalid))
	if err := s.SwapRelease(); err == nil {
		t.Fatal("SwapRelease: expected an error for a release with an invalid config")
	}
	if got := get(s, "/").Body.String(); !strings.Contains(got, "release A content") {
		t.Errorf("after failed swap: expected release A content, got %q", got)
	}

	// And for a missing current link entirely.
	must(os.Remove(filepath.Join(root, config.CurrentLinkName)))
	if err := s.SwapRelease(); err == nil {
		t.Fatal("SwapRelease: expected an error when current cannot be resolved")
	}
	if got := get(s, "/").Body.String(); !strings.Contains(got, "release A content") {
		t.Errorf("after failed swap: expected release A content, got %q", got)
	}
}

func TestSwapReleaseLegacyLayoutErrors(t *testing.T) {
	dir := t.TempDir()
	must(os.MkdirAll(filepath.Join(dir, "site"), 0755))
	must(os.WriteFile(filepath.Join(dir, "site", "index.pars"), []byte(`"legacy"`), 0644))

	cfg := config.Defaults()
	cfg.ReleaseDir = dir
	cfg.DataDir = dir
	cfg.Server.Dev = true
	cfg.Site.Path = filepath.Join(dir, "site")
	config.ResolvePaths(cfg)

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	prevState := s.serving.Load()
	err = s.SwapRelease()
	if err == nil {
		t.Fatal("SwapRelease: expected an error in the legacy layout")
	}
	if !strings.Contains(err.Error(), "site-root") {
		t.Errorf("error should say the layout is not a site root, got: %v", err)
	}
	if s.serving.Load() != prevState {
		t.Error("legacy-layout SwapRelease must not touch the serving state")
	}
}

func TestSwapReleaseClearsCaches(t *testing.T) {
	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	// Populate the response cache (dev.cache: true keeps it enabled in dev
	// mode) and prime the script cache through a real request.
	req := httptest.NewRequest("GET", "/cached", http.NoBody)
	s.responseCache.Set(req, time.Hour, http.StatusOK, http.Header{}, []byte("stale"))
	if s.responseCache.Get(req) == nil {
		t.Fatal("response cache entry should exist before the swap")
	}
	get(s, "/")

	must(deploy.SetCurrent(root, filepath.Join(root, config.ReleasesDirName, "B")))
	if err := s.SwapRelease(); err != nil {
		t.Fatalf("SwapRelease: %v", err)
	}

	if s.responseCache.Get(req) != nil {
		t.Error("response cache entry survived the swap")
	}
}

// A failed swap must not clear anything either: the old release keeps
// serving with the caches it had.
func TestSwapReleaseFailureLeavesCachesIntact(t *testing.T) {
	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	req := httptest.NewRequest("GET", "/cached", http.NoBody)
	s.responseCache.Set(req, time.Hour, http.StatusOK, http.Header{}, []byte("still here"))

	must(os.Remove(filepath.Join(root, config.CurrentLinkName)))
	if err := s.SwapRelease(); err == nil {
		t.Fatal("SwapRelease: expected an error with no current link")
	}
	if s.responseCache.Get(req) == nil {
		t.Error("failed swap cleared the response cache")
	}
}

func TestSwapReleaseWarnsOnListenerConfigChange(t *testing.T) {
	root := activationFixture(t)
	s, _, stderr := newSiteRootServer(t, root)
	oldPort := s.config.Server.Port

	relisten := writeRelease(t, root, "relisten", "relisten release", "server:\n  port: 9313\n  bind: 10.0.0.1\n")
	must(deploy.SetCurrent(root, relisten))
	if err := s.SwapRelease(); err != nil {
		t.Fatalf("SwapRelease: %v", err)
	}

	// The release is served, but the listener settings are not applied.
	if got := get(s, "/").Body.String(); !strings.Contains(got, "relisten release") {
		t.Errorf("expected the new release's content, got %q", got)
	}
	if s.config.Server.Port != oldPort {
		t.Errorf("server.port applied live: got %d, want %d", s.config.Server.Port, oldPort)
	}
	if s.config.Server.Bind != "" {
		t.Errorf("server.bind applied live: got %q", s.config.Server.Bind)
	}
	warnings := stderr.String()
	for _, want := range []string{"server.port", "server.bind", "restart required"} {
		if !strings.Contains(warnings, want) {
			t.Errorf("expected a %q warning, stderr: %s", want, warnings)
		}
	}
}

// fsnotifyDelivers probes whether filesystem events are delivered in this
// environment at all, so the watcher test can distinguish "fsnotify cannot
// work here" from "the watcher is broken".
func fsnotifyDelivers(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return false
	}
	defer fw.Close()
	if err := fw.Add(dir); err != nil {
		return false
	}
	if err := os.WriteFile(filepath.Join(dir, "probe"), []byte("x"), 0644); err != nil {
		return false
	}
	select {
	case <-fw.Events:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

func TestCurrentLinkWatcherActivatesRepointedRelease(t *testing.T) {
	if !fsnotifyDelivers(t) {
		t.Skip("fsnotify does not deliver events in this environment")
	}

	root := activationFixture(t)
	s, _, _ := newSiteRootServer(t, root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clw, err := newCurrentLinkWatcher(s)
	if err != nil {
		t.Fatalf("newCurrentLinkWatcher: %v", err)
	}
	clw.Start(ctx)
	defer clw.Close()

	// Re-point `current` the way the deploy CLI does, from outside the
	// server's own code path.
	must(deploy.SetCurrent(root, filepath.Join(root, config.ReleasesDirName, "B")))

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(get(s, "/").Body.String(), "release B content") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("watcher did not activate release B; still serving %q", get(s, "/").Body.String())
}
