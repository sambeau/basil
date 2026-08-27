package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TestWatcher_HandlesPartFiles guards BUG-043.
//
// A Part is imported like any other module — <Part src={@./shows.part}/> goes
// through importModule — so an edited .part must invalidate the module cache
// and reload the browser, exactly as an edited .pars does.
//
// Leaving .part out did not stop Parts working, which is why it survived: a
// Part's own endpoint re-reads the file, so polling showed the new version
// while the copy rendered into the page stayed on the old one. The page loaded
// stale and corrected itself a second later — or never, for a Part with no
// part-refresh.
func TestWatcher_HandlesPartFiles(t *testing.T) {
	w := &Watcher{}

	// Extensions that must reload the browser when they change.
	for _, path := range []string{
		"site/shows.part",
		"site/index.pars",
		"lib/helpers.parsley",
		"public/style.css",
		"public/app.js",
	} {
		if !w.shouldTriggerReload(path) {
			t.Errorf("shouldTriggerReload(%q) = false, want true", path)
		}
	}

	// Every module type must clear the cache as well as reload the browser.
	// These are the same list by construction (isModuleExtension), which is the
	// point: a type that reloads but does not invalidate — or the reverse — is
	// exactly the shape this bug took.
	for _, ext := range []string{".part", ".pars", ".parsley"} {
		if !isModuleExtension(ext) {
			t.Errorf("isModuleExtension(%q) = false; an edit to one will not clear the module cache", ext)
		}
	}
	for _, ext := range []string{".css", ".js", ".html", ".db"} {
		if isModuleExtension(ext) {
			t.Errorf("isModuleExtension(%q) = true; it is not imported as a module", ext)
		}
	}

	// Runtime state must not reload: a dev server writing its own log database
	// would otherwise reload the browser in a loop.
	for _, path := range []string{
		"dev_logs.db",
		".basil-auth.db-wal",
		"notes.txt",
	} {
		if w.shouldTriggerReload(path) {
			t.Errorf("shouldTriggerReload(%q) = true, want false", path)
		}
	}
}

// TestWatcher_WatchesDirectoriesCreatedAfterStart guards half of BUG-048.
//
// fsnotify watches a directory, not its future children. watchDirRecursive ran
// once at Start, so a components/ subdirectory added while the server was up
// reported nothing at all: no browser reload, and nothing to invalidate the
// module cache with. Writing a new component and then editing it — the ordinary
// way a component comes into existence — was the one case that never reloaded.
func TestWatcher_WatchesDirectoriesCreatedAfterStart(t *testing.T) {
	root := t.TempDir()
	w, log := newTestWatcher(t, root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.eventLoop(ctx)

	// A directory that did not exist when the watcher started.
	nested := filepath.Join(root, "components", "shows")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	component := filepath.Join(nested, "shows.pars")
	if err := os.WriteFile(component, []byte("export a = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Editing it must be seen. Written after a beat so the create and the edit
	// cannot be one event, and so this asserts the steady state rather than the
	// initial write.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(component, []byte("export a = 2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !log.waitFor(component) {
		t.Errorf("editing %s raised nothing; a directory created after start is never watched", component)
	}
}

// TestWatcher_DebounceIsPerFile guards the other half of BUG-048.
//
// The debounce window was global and dropped the event outright, so saving two
// files within 100ms — a Save All, or a formatter touching a second file —
// discarded the second file's change entirely: no reload, no invalidation, and
// nothing left to notice it later. The window exists to collapse repeat events
// for one file, not to lose other files.
func TestWatcher_DebounceIsPerFile(t *testing.T) {
	root := t.TempDir()
	w, log := newTestWatcher(t, root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.eventLoop(ctx)

	first := filepath.Join(root, "shows.pars")
	second := filepath.Join(root, "venue.pars")

	// Both inside one debounce window, as a Save All would be.
	if err := os.WriteFile(first, []byte("export a = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(second, []byte("export b = 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !log.waitFor(first) {
		t.Errorf("%s raised nothing", first)
	}
	if !log.waitFor(second) {
		t.Errorf("%s raised nothing; the second file of a Save All was swallowed by the first file's debounce window", second)
	}
}

// newTestWatcher returns a Watcher watching root, and the log its handlers
// write to. It does not go through Start, which would also want a server.
func newTestWatcher(t *testing.T, root string) (*Watcher, *syncLog) {
	t.Helper()

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	t.Cleanup(func() { fsWatcher.Close() })

	log := &syncLog{}
	w := &Watcher{
		watcher:    fsWatcher,
		stdout:     log,
		stderr:     log,
		lastChange: make(map[string]time.Time),
	}
	if err := w.watchDirRecursive(root); err != nil {
		t.Fatalf("watchDirRecursive: %v", err)
	}
	return w, log
}

// syncLog collects watcher output. The event loop writes from its own
// goroutine, so reads and writes are locked.
type syncLog struct {
	mu    sync.Mutex
	lines strings.Builder
}

func (l *syncLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lines.Write(p)
}

// waitFor polls for up to two seconds for the watcher to report path.
func (l *syncLog) waitFor(path string) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		seen := strings.Contains(l.lines.String(), path)
		l.mu.Unlock()
		if seen {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
