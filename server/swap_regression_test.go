package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// writeRoutesRelease materialises a routes-mode release: a page route, an API
// route, and no root route - so an unmatched path exercises handle404. The
// API route exercises apiHandler, which must not read swappable server fields
// per request.
func writeRoutesRelease(t *testing.T, root, id, body string) string {
	t.Helper()
	dir := filepath.Join(root, config.ReleasesDirName, id)
	must(os.MkdirAll(dir, 0755))
	must(os.WriteFile(filepath.Join(dir, "page.pars"), []byte(fmt.Sprintf("%q", body)), 0644))
	apiCode := "let api = import @std/api\n" +
		"export get = api.public(fn() { {body: " + fmt.Sprintf("%q", body) + "} })\n"
	must(os.WriteFile(filepath.Join(dir, "api.pars"), []byte(apiCode), 0644))
	yaml := "routes:\n" +
		"  - path: /page\n    handler: page.pars\n" +
		"  - path: /api/things\n    handler: api.pars\n    type: api\n"
	must(os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte(yaml), 0644))
	return dir
}

// Regression for the SwapRelease data race (FEAT-153 review F1.1): request
// paths that are not pinned to a release - the /__site.css closure
// (assetBundle), an API route (apiHandler's config and bundle reads), and the
// 404 error page (errors.go config reads) - hammered concurrently with
// repeated swaps. Run with -race: any unsynchronized read of a field
// SwapRelease writes fails the run.
func TestSwapReleaseRaceUnpinnedPaths(t *testing.T) {
	root := t.TempDir()
	writeRoutesRelease(t, root, "A", "race release A")
	writeRoutesRelease(t, root, "B", "race release B")
	must(os.MkdirAll(filepath.Join(root, config.DataDirName), 0755))
	must(os.Symlink(filepath.Join(config.ReleasesDirName, "A"), filepath.Join(root, config.CurrentLinkName)))
	writeSiteAuthDB(t, root)

	s, _, _ := newSiteRootServer(t, root)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	hit := func(path string) {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			rec := httptest.NewRecorder()
			s.servingHandler().ServeHTTP(rec, httptest.NewRequest("GET", path, http.NoBody))
		}
	}
	for _, p := range []string{"/__site.css", "/api/things", "/no-such-page-xyz"} {
		wg.Add(1)
		go hit(p)
	}

	targets := []string{"A", "B"}
	for i := 0; i < 40; i++ {
		rel := filepath.Join(root, config.ReleasesDirName, targets[i%2])
		must(deploy.SetCurrent(root, rel))
		if err := s.SwapRelease(); err != nil {
			t.Fatalf("SwapRelease: %v", err)
		}
	}
	close(stop)
	wg.Wait()
}

// The fragment cache has the same repoison shape as the response cache:
// user-chosen string keys shared across releases. Views pin a generation at
// construction, so a write through an old release's view - even one landing
// after Advance and Clear - can never be read through a newer view.
func TestFragmentCacheViewsIsolateGenerations(t *testing.T) {
	fc := newFragmentCache(false, 10)

	oldView := fc.view()
	oldView.Set("sidebar", "old release html", time.Minute)
	if got, ok := oldView.Get("sidebar"); !ok || got != "old release html" {
		t.Fatalf("old view should read its own write, got %q ok=%v", got, ok)
	}

	// The swap sequence: Advance, build new handlers (new view), Clear.
	fc.Advance()
	newView := fc.view()
	fc.Clear()

	// A write from an old-release request that finishes after the swap.
	oldView.Set("sidebar", "old release html", time.Minute)

	if got, ok := newView.Get("sidebar"); ok {
		t.Errorf("new view read the old release's post-swap write: %q", got)
	}
	newView.Set("sidebar", "new release html", time.Minute)
	if got, ok := newView.Get("sidebar"); !ok || got != "new release html" {
		t.Errorf("new view should read its own write, got %q ok=%v", got, ok)
	}
	// Invalidate through one view must not touch the other generation.
	newView.Invalidate("sidebar")
	if got, ok := oldView.Get("sidebar"); !ok || got != "old release html" {
		t.Errorf("old view's entry crossed generations, got %q ok=%v", got, ok)
	}
}

// writeCachedRelease is writeRelease with site-level response caching on.
func writeCachedRelease(t *testing.T, root, id, body string) string {
	t.Helper()
	dir := filepath.Join(root, config.ReleasesDirName, id)
	must(os.MkdirAll(filepath.Join(dir, "site"), 0755))
	must(os.WriteFile(filepath.Join(dir, "site", "index.pars"), []byte(fmt.Sprintf("%q", body)), 0644))
	yaml := "site:\n  path: site\n  cache: 60s\ndev:\n  cache: true\n"
	must(os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte(yaml), 0644))
	return dir
}

// Regression for cache repoison across a swap (FEAT-153 review F1.2): a
// request in flight on release A finishes - and writes the response cache -
// AFTER SwapRelease switched to release B and cleared the caches. A fresh
// request on the new serving surface must MISS and serve B's content, never
// A's body as a HIT.
func TestSwapReleaseCacheRepoisonAcrossSwap(t *testing.T) {
	root := t.TempDir()
	writeCachedRelease(t, root, "A", "release A content")
	writeCachedRelease(t, root, "B", "release B content")
	must(os.MkdirAll(filepath.Join(root, config.DataDirName), 0755))
	must(os.Symlink(filepath.Join(config.ReleasesDirName, "A"), filepath.Join(root, config.CurrentLinkName)))
	writeSiteAuthDB(t, root)

	s, _, _ := newSiteRootServer(t, root)

	entered := make(chan struct{})
	unblock := make(chan struct{})
	oldMux := s.mux
	oldMux.HandleFunc("/inflight", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-unblock
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		oldMux.ServeHTTP(w, r2) // renders (and would cache) "/" from release A
	})

	done := make(chan struct{})
	rec := httptest.NewRecorder()
	go func() {
		defer close(done)
		s.servingHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/inflight", http.NoBody))
	}()
	<-entered

	must(deploy.SetCurrent(root, filepath.Join(root, config.ReleasesDirName, "B")))
	if err := s.SwapRelease(); err != nil {
		t.Fatalf("SwapRelease: %v", err)
	}

	// The swap is complete and the caches were cleared. Now the old request
	// finishes: its inner "/" render uses URL.Path "/", so its cache write
	// collides with a front-door "/" request unless writes are
	// release-scoped.
	close(unblock)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight request did not complete")
	}

	// A fresh request through the NEW serving surface must miss the cache
	// and serve the new release.
	fresh := httptest.NewRecorder()
	s.servingHandler().ServeHTTP(fresh, httptest.NewRequest("GET", "/", http.NoBody))
	body := fresh.Body.String()
	xc := fresh.Header().Get("X-Cache")
	if strings.Contains(body, "release A content") {
		t.Errorf("cache repoison: new release's mux served old release content (X-Cache=%s)", xc)
	}
	if !strings.Contains(body, "release B content") {
		t.Errorf("fresh request after swap: expected release B content, got %q (X-Cache=%s)", body, xc)
	}
	if xc == "HIT" {
		t.Errorf("fresh request after swap must MISS the cleared cache, got X-Cache=HIT")
	}
}
