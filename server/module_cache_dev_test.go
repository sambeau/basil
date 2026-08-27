package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
)

// The module cache is the last cache Basil owns that decided for itself
// whether an entry could be trusted, rather than being able to tell. It cached
// unconditionally and left dev-mode freshness to the file watcher, so an edit
// the watcher missed - a directory created after startup, a save that lost its
// debounce race, a request that re-stored the pre-edit module after the
// invalidation - served stale code until the server restarted (BUG-048).
//
// These tests exercise the contract at the request boundary and never touch
// the watcher, because the watcher is exactly what must not be load-bearing:
// dev mode is correct because an entry can prove it is current, not because
// something noticed. That entries are still *used* when they are current is
// invisible from out here and is tested in
// pkg/parsley/evaluator/module_cache_test.go.

// moduleCacheFixture writes a handler that imports a component, which in turn
// imports a helper. The version the handler reports comes from the helper, two
// imports down, so an edit to it is only seen by a cache that tracks what a
// module was built from rather than merely which file it is.
func moduleCacheFixture(t *testing.T, dev, devCache bool) (*Server, string) {
	t.Helper()
	evaluator.ClearModuleCache()

	dir := t.TempDir()

	// The files live a directory down, as real ones would.
	componentDir := filepath.Join(dir, "components")
	if err := os.MkdirAll(componentDir, 0o755); err != nil {
		t.Fatalf("mkdir components: %v", err)
	}

	helper := filepath.Join(componentDir, "helper.pars")
	writeHelperVersion(t, helper, 1)

	component := `let helper = import @~/components/helper.pars
export version = helper.version
`
	if err := os.WriteFile(filepath.Join(componentDir, "shows.pars"), []byte(component), 0o644); err != nil {
		t.Fatalf("write component: %v", err)
	}

	handlerPath := filepath.Join(dir, "api.pars")
	handler := `let api = import @std/api
let shows = import @~/components/shows.pars

export get = api.public(fn(req) {
    { version: shows.version }
})
`
	if err := os.WriteFile(handlerPath, []byte(handler), 0o644); err != nil {
		t.Fatalf("write handler: %v", err)
	}

	cfg := &config.Config{
		ReleaseDir: dir,
		DataDir:    dir,
		Server:     config.ServerConfig{Host: "localhost", Port: 8080, Dev: dev},
		Dev:        config.DevConfig{Cache: devCache},
		Routes:     []config.Route{{Path: "/api/version", Handler: handlerPath, Type: "api"}},
		Logging:    config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return srv, helper
}

func writeHelperVersion(t *testing.T, path string, version int) {
	t.Helper()
	src := "export version = " + strconv.Itoa(version) + "\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
}

// requestVersion issues a request and returns the version the handler reported.
func requestVersion(t *testing.T, srv *Server) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/version", http.NoBody)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	version, ok := body["version"].(float64)
	if !ok {
		t.Fatalf("no version in response: %s", rec.Body.String())
	}
	return int(version)
}

// TestModuleCacheDevModeSeesEdits is the regression test for BUG-048. No
// watcher runs here and nothing invalidates anything: the second request sees
// the edit because the cached entry can tell it is out of date.
//
// The file edited is two imports down from the handler and the component
// between them is untouched, which is the shape the report was actually made
// of - the file you edit is rarely the one the page imports directly.
func TestModuleCacheDevModeSeesEdits(t *testing.T) {
	srv, helper := moduleCacheFixture(t, true, false)

	if got := requestVersion(t, srv); got != 1 {
		t.Fatalf("first request: expected version 1, got %d", got)
	}

	time.Sleep(10 * time.Millisecond) // distinguishable modification time
	writeHelperVersion(t, helper, 2)

	if got := requestVersion(t, srv); got != 2 {
		t.Errorf("after edit: expected version 2, got %d - the module cache served a stale import in dev mode", got)
	}
}

// TestModuleCacheProductionCaches is the other half of the spec. Production
// must still cache: an edit under the running server is not meant to be picked
// up, and a fix that made dev fresh by making everything uncached would pass
// the test above and fail here.
func TestModuleCacheProductionCaches(t *testing.T) {
	srv, helper := moduleCacheFixture(t, false, false)

	if got := requestVersion(t, srv); got != 1 {
		t.Fatalf("first request: expected version 1, got %d", got)
	}

	time.Sleep(10 * time.Millisecond)
	writeHelperVersion(t, helper, 2)

	if got := requestVersion(t, srv); got != 1 {
		t.Errorf("production: expected the cached version 1, got %d - the module cache is not caching", got)
	}
}

// TestModuleCacheDevCacheOptsBackIn covers dev.cache, the escape hatch that
// makes dev mode cache the way production does: entries trusted as they stand,
// with no stat and no revalidation.
func TestModuleCacheDevCacheOptsBackIn(t *testing.T) {
	srv, helper := moduleCacheFixture(t, true, true)

	if got := requestVersion(t, srv); got != 1 {
		t.Fatalf("first request: expected version 1, got %d", got)
	}

	time.Sleep(10 * time.Millisecond)
	writeHelperVersion(t, helper, 2)

	if got := requestVersion(t, srv); got != 1 {
		t.Errorf("dev with dev.cache=true: expected the cached version 1, got %d", got)
	}
}

// TestConfigNoCache pins the one decision every cache consults.
func TestConfigNoCache(t *testing.T) {
	tests := []struct {
		name     string
		dev      bool
		devCache bool
		want     bool
	}{
		{"production", false, false, false},
		{"production ignores dev.cache", false, true, false},
		{"dev", true, false, true},
		{"dev with dev.cache", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{Dev: tt.dev},
				Dev:    config.DevConfig{Cache: tt.devCache},
			}
			if got := cfg.NoCache(); got != tt.want {
				t.Errorf("NoCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
