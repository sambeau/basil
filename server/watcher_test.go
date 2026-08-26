package server

import "testing"

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
