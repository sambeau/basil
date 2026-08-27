package evaluator

import (
	"os"
	"path/filepath"
	"sync"
)

// The module cache holds the result of every `import` of a file, keyed by
// absolute path. Importing is expensive - read, lex, parse, then evaluate the
// module's top level - and a page is mostly components, so caching it is worth
// a great deal.
//
// The difficulty is knowing when an entry has gone out of date. Basil used to
// answer that from outside, with the dev-mode file watcher calling
// InvalidateModule, which made correctness depend on an fs event arriving:
// every gap in event delivery was a gap in correctness, and there were three
// (BUG-048). An entry now carries the stamps of the files it was built from
// and checks them itself, so a stale entry is detectable without anyone having
// been told anything.

// fileStamp identifies a file's contents cheaply. Modification time and size
// together are what every build tool of this shape uses; they are compared by
// value, so modTime is kept as Unix nanoseconds rather than a time.Time, whose
// equality also considers monotonic readings and location.
//
// The residual gap is a write that lands within the filesystem's mtime
// resolution of the previous one and leaves the size unchanged. On APFS and
// ext4 that resolution is a nanosecond, so this needs two writes in the same
// nanosecond; on a filesystem with one-second stamps it would need two saves
// in the same second, at the same length.
type fileStamp struct {
	modUnixNano int64
	size        int64
}

func stampOf(path string) (fileStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}, err
	}
	return fileStamp{modUnixNano: info.ModTime().UnixNano(), size: info.Size()}, nil
}

// cachedModule is one import's result, with everything needed to tell whether
// it is still current.
type cachedModule struct {
	dict *Dictionary

	// deps stamps every file this module was built from: the module itself,
	// and the transitive closure of the files it imports.
	//
	// The closure is the whole point. environmentToDict returns
	// &Dictionary{Pairs: pairs, Env: env}, so a cached module carries its own
	// environment with its nested imports already resolved inside it. Stamping
	// only the module's own file would serve a component built against last
	// week's copy of a helper it imports, and would do it for the component
	// that had most reason to be shared.
	deps map[string]fileStamp
}

// ModuleCache caches imported modules.
type ModuleCache struct {
	mu      sync.RWMutex
	modules map[string]*cachedModule
}

// Global module cache
var moduleCache = &ModuleCache{
	modules: make(map[string]*cachedModule),
}

// lookup returns the cached module for absPath.
//
// When validate is set - dev mode, where the sources can change under a
// running server - the entry is served only if every file it was built from
// still has the stamp it had at the time. Otherwise the entry is trusted as
// it stands, which is right in production, where a release's files do not
// change while it is being served and the stats would always agree.
func (c *ModuleCache) lookup(absPath string, validate bool) (*Dictionary, bool) {
	c.mu.RLock()
	entry, ok := c.modules[absPath]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !validate {
		return entry.dict, true
	}

	for path, want := range entry.deps {
		got, err := stampOf(path)
		if err != nil || got != want {
			return nil, false
		}
	}
	return entry.dict, true
}

// store caches a module built from the file stamped by stamp, importing
// directDeps.
//
// stamp must have been taken *before* the file was read. A file rewritten
// while it was being read then has a modification time later than the stamp
// stored against it, so the entry fails its own next validation and is
// re-read. That closes by construction the window where a request that began
// before an edit finishes after it and puts the pre-edit module back into the
// cache - which it used to do with nothing left to invalidate it.
func (c *ModuleCache) store(absPath string, stamp fileStamp, dict *Dictionary, directDeps []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	deps := map[string]fileStamp{absPath: stamp}
	for _, dep := range directDeps {
		if dep == absPath {
			continue
		}
		entry, ok := c.modules[dep]
		if !ok {
			// A dependency with no entry is one whose own dependencies we
			// cannot enumerate, so an entry for this module could not be
			// validated honestly. Decline to cache rather than cache
			// something we would have to trust blindly.
			return
		}
		for path, depStamp := range entry.deps {
			deps[path] = depStamp
		}
	}

	c.modules[absPath] = &cachedModule{dict: dict, deps: deps}
}

// ClearModuleCache clears all cached modules.
//
// Entries validate themselves, so this is not needed for freshness. It exists
// for the cases where the cached objects themselves must go: ReloadScripts and
// release activation, where cached modules may hold database connections
// belonging to the outgoing release.
func ClearModuleCache() {
	moduleCache.mu.Lock()
	defer moduleCache.mu.Unlock()
	moduleCache.modules = make(map[string]*cachedModule)
}

// InvalidateModule removes a module from the cache. The path may be absolute
// or relative; both forms are tried.
//
// Like ClearModuleCache this is no longer required for correctness - the dev
// file watcher calls it, and an entry would notice the same edit by itself on
// the next import. It is kept because it costs nothing and does not depend on
// file stamps, which a filesystem with a coarse clock could in principle blur.
func InvalidateModule(path string) {
	moduleCache.mu.Lock()
	defer moduleCache.mu.Unlock()

	if absPath, err := filepath.Abs(path); err == nil {
		delete(moduleCache.modules, absPath)
	}
	delete(moduleCache.modules, path)
}

// moduleDepSet collects the files imported while a module is being evaluated,
// so the module can record what it was built from. A module's functions run in
// environments descended from its own, so an import from inside a function
// body registers against the module that defined it.
type moduleDepSet struct {
	mu    sync.Mutex
	paths map[string]bool
}

func newModuleDepSet() *moduleDepSet {
	return &moduleDepSet{paths: make(map[string]bool)}
}

func (d *moduleDepSet) add(path string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.paths[path] = true
}

func (d *moduleDepSet) snapshot() []string {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	paths := make([]string, 0, len(d.paths))
	for path := range d.paths {
		paths = append(paths, path)
	}
	return paths
}
