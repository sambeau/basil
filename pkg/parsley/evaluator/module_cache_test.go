package evaluator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// These tests reach into the cache directly, because the two things that
// matter here are invisible from outside it. That an edit is picked up can be
// seen at the HTTP surface (server/module_cache_dev_test.go); that an
// *unedited* module is still served from cache cannot, since a correct answer
// looks identical either way. Comparing the returned *Dictionary by identity
// tells the difference: the same pointer means the entry was reused, a
// different one means the module was read and evaluated again.

// devEnv returns an environment configured the way a dev-mode request is -
// which is to say a plain one. Revalidation is the default; TrustModuleCache
// is the opt-out.
func devEnv(t *testing.T, dir string) *Environment {
	t.Helper()
	env := NewEnvironment()
	env.Filename = filepath.Join(dir, "page.pars")
	env.RootPath = dir
	return env
}

// writeModule writes src to path and returns the path. Modification times are
// spaced so that a rewrite is distinguishable from the original even on a
// filesystem with a coarse clock, which is a property of this test rather than
// of the cache.
func writeModule(t *testing.T, path, src string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func importOrFail(t *testing.T, path string, env *Environment) *Dictionary {
	t.Helper()
	result := importModule(path, env)
	dict, ok := result.(*Dictionary)
	if !ok {
		t.Fatalf("import %s: %v", path, result.Inspect())
	}
	return dict
}

// TestModuleCache_ServesUnchangedModuleFromCache is the half of the contract
// that the request-level tests cannot see. Dev mode caches; it just refuses to
// serve an entry it cannot show is current.
func TestModuleCache_ServesUnchangedModuleFromCache(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()
	writeModule(t, filepath.Join(dir, "shows.pars"), "export version = 1\n")

	env := devEnv(t, dir)
	first := importOrFail(t, "~/shows.pars", env)
	second := importOrFail(t, "~/shows.pars", env)

	if first != second {
		t.Error("an unchanged module was evaluated twice; dev mode is not caching at all")
	}
}

// TestModuleCache_RereadsChangedModule is the other half: the stamp must
// actually be checked.
func TestModuleCache_RereadsChangedModule(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "shows.pars")
	writeModule(t, path, "export version = 1\n")

	env := devEnv(t, dir)
	first := importOrFail(t, "~/shows.pars", env)

	time.Sleep(10 * time.Millisecond) // distinguishable modification time
	writeModule(t, path, "export version = 2\n")

	second := importOrFail(t, "~/shows.pars", env)
	if first == second {
		t.Fatal("an edited module was served from cache")
	}
	assertVersion(t, second, 2)
}

// TestModuleCache_RereadsOnTransitiveChange is the test a per-file stamp fails.
//
// environmentToDict returns &Dictionary{Pairs: pairs, Env: env}, so a cached
// module carries its own environment with its nested imports already resolved
// inside it. shows.pars is untouched here and its own stamp still matches; it
// is stale all the same, because the helper it was built against changed. A
// cache that stamped only the file it was asked for would serve last week's
// helper through every component that imports it — and the more widely shared
// the helper, the more places would be wrong.
func TestModuleCache_RereadsOnTransitiveChange(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()

	helper := filepath.Join(dir, "helper.pars")
	writeModule(t, helper, "export version = 1\n")
	writeModule(t, filepath.Join(dir, "shows.pars"), `let helper = import @~/helper.pars
export version = helper.version
`)

	env := devEnv(t, dir)
	first := importOrFail(t, "~/shows.pars", env)
	assertVersion(t, first, 1)

	time.Sleep(10 * time.Millisecond)
	writeModule(t, helper, "export version = 2\n") // the helper, not shows.pars

	second := importOrFail(t, "~/shows.pars", env)
	if first == second {
		t.Fatal("shows.pars was served from cache after the helper it imports changed")
	}
	assertVersion(t, second, 2)
}

// TestModuleCache_ProductionDoesNotStat pins the other mode: with
// TrustModuleCache set the entry is served as it stands, which is right where
// a release's files do not change while it is being served.
func TestModuleCache_ProductionDoesNotStat(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "shows.pars")
	writeModule(t, path, "export version = 1\n")

	env := devEnv(t, dir)
	env.TrustModuleCache = true

	first := importOrFail(t, "~/shows.pars", env)

	time.Sleep(10 * time.Millisecond)
	writeModule(t, path, "export version = 2\n")

	if second := importOrFail(t, "~/shows.pars", env); first != second {
		t.Error("production re-read an edited module; the entry should have been trusted")
	}
}

// TestModuleCache_StaleWriteLosesToItsOwnStamp covers the race that used to
// leave a module stale with nothing able to correct it: a request reads a
// module, an edit lands, and the request then stores what it read. The stamp
// was taken before the read, so the entry it writes describes the pre-edit
// file and fails its own next validation.
func TestModuleCache_StaleWriteLosesToItsOwnStamp(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "shows.pars")
	writeModule(t, path, "export version = 1\n")

	stale, err := stampOf(path)
	if err != nil {
		t.Fatalf("stampOf: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	writeModule(t, path, "export version = 2\n")

	// The in-flight request completes and stores what it read, under the stamp
	// it took before reading.
	env := devEnv(t, dir)
	moduleCache.store(path, stale, &Dictionary{Pairs: map[string]ast.Expression{}, Env: env}, nil)

	if _, ok := moduleCache.lookup(path, true); ok {
		t.Error("a module stored under a pre-edit stamp was served; the entry did not detect itself as stale")
	}
}

func assertVersion(t *testing.T, dict *Dictionary, want int64) {
	t.Helper()
	expr, ok := dict.Pairs["version"]
	if !ok {
		t.Fatal("module has no exported `version`")
	}
	obj := Eval(expr, dict.Env)
	got, ok := obj.(*Integer)
	if !ok {
		t.Fatalf("version is %s, not an integer", obj.Type())
	}
	if got.Value != want {
		t.Errorf("version = %d, want %d", got.Value, want)
	}
}

// TestModuleCache_DefaultEnvironmentRevalidates pins the polarity of
// TrustModuleCache, which is the whole reason the field is named for the
// exception rather than the rule.
//
// A bare NewEnvironment - the pars CLI, the REPL, an embedder, anything that
// has never heard of this field - revalidates. Getting it wrong that way round
// costs a stat per import; getting it wrong the other way round is BUG-048,
// and it took three of these to notice. Nothing outside the server sets the
// field at all, so this test is what keeps those callers right.
func TestModuleCache_DefaultEnvironmentRevalidates(t *testing.T) {
	ClearModuleCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "shows.pars")
	writeModule(t, path, "export version = 1\n")

	env := NewEnvironment()
	env.Filename = filepath.Join(dir, "session.pars")
	env.RootPath = dir

	if env.TrustModuleCache {
		t.Fatal("a fresh Environment trusts the module cache; the safe default is to revalidate")
	}

	first := importOrFail(t, "~/shows.pars", env)
	assertVersion(t, first, 1)

	time.Sleep(10 * time.Millisecond)
	writeModule(t, path, "export version = 2\n")

	second := importOrFail(t, "~/shows.pars", env)
	if first == second {
		t.Fatal("an edited module was served from cache to a default environment")
	}
	assertVersion(t, second, 2)
}
