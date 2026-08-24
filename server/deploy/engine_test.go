package deploy

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// engineFixture is a real repository with the commit history the pipeline
// tests need: two good releases, a release that fails validation, a release
// whose post-deploy hook fails, and a final good release.
type engineFixture struct {
	siteRoot string
	repo     string
	good1    string // valid site, v1
	good2    string // valid site, v2
	broken   string // site/broken.pars does not parse
	hookFail string // deploy.pars calls fail()
	good3    string // valid site, v3 (hook removed)
}

func newEngineFixture(t *testing.T) engineFixture {
	t.Helper()
	requireGit(t)

	f := engineFixture{siteRoot: t.TempDir(), repo: t.TempDir()}
	runTestGit(t, f.repo, "init", "--initial-branch=live")
	runTestGit(t, f.repo, "config", "user.name", "Test Author")
	runTestGit(t, f.repo, "config", "user.email", "author@example.com")

	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(f.repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(msg string) string {
		t.Helper()
		runTestGit(t, f.repo, "add", "-A")
		runTestGit(t, f.repo, "commit", "-m", msg)
		return runTestGit(t, f.repo, "rev-parse", "HEAD")
	}

	write("basil.yaml", validBasilYAML)
	write("site/index.pars", "<h1>\"v1\"</h1>\n")
	f.good1 = commit("v1")

	write("site/index.pars", "<h1>\"v2\"</h1>\n")
	f.good2 = commit("v2")

	write("site/broken.pars", "let x = = 2\n")
	f.broken = commit("broken parsley")

	if err := os.Remove(filepath.Join(f.repo, "site", "broken.pars")); err != nil {
		t.Fatal(err)
	}
	write("deploy.pars", "fail(\"deliberate hook failure\")\n")
	f.hookFail = commit("failing hook")

	if err := os.Remove(filepath.Join(f.repo, "deploy.pars")); err != nil {
		t.Fatal(err)
	}
	write("site/index.pars", "<h1>\"v3\"</h1>\n")
	f.good3 = commit("v3")

	return f
}

func (f engineFixture) engine(out io.Writer) *Engine {
	if out == nil {
		out = io.Discard
	}
	return &Engine{
		SiteRoot:   f.siteRoot,
		RepoDir:    f.repo,
		RecordPath: filepath.Join(f.siteRoot, "data", "deploy.db"),
		Keep:       5,
		Publisher:  "cli:test",
		Trigger:    TriggerCLI,
		Out:        out,
	}
}

func (f engineFixture) releasesDir() string {
	return filepath.Join(f.siteRoot, "releases")
}

// currentSHA reads which release is live; "" when no release is active yet.
func currentSHA(t *testing.T, siteRoot string) string {
	t.Helper()
	current, err := CurrentRelease(siteRoot)
	if err != nil {
		return ""
	}
	return filepath.Base(current)
}

func recordEntries(t *testing.T, e *Engine) []Entry {
	t.Helper()
	rec, err := OpenRecord(e.RecordPath)
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()
	entries, err := rec.List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return entries
}

func TestDeploySuccess(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}
	res, err := e.Deploy(f.good2)
	if err != nil {
		t.Fatalf("deploying good2: %v", err)
	}

	if res.Outcome != OutcomeDeployed {
		t.Errorf("Outcome = %q, want %q", res.Outcome, OutcomeDeployed)
	}
	if res.CommitSHA != f.good2 {
		t.Errorf("CommitSHA = %q, want %q", res.CommitSHA, f.good2)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good2 {
		t.Errorf("current points at %q, want %q", got, f.good2)
	}
	// The previous release stays on disk: it is what rollback needs.
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.good1)); err != nil {
		t.Errorf("previous release was removed: %v", err)
	}

	entries := recordEntries(t, e)
	if len(entries) != 2 {
		t.Fatalf("record has %d entries, want 2", len(entries))
	}
	newest := entries[0]
	if newest.CommitSHA != f.good2 || newest.Outcome != OutcomeDeployed {
		t.Errorf("newest entry = %+v, want deployed %s", newest, f.good2)
	}
	if newest.Trigger != TriggerCLI || newest.Publisher != "cli:test" {
		t.Errorf("identity: trigger=%q publisher=%q", newest.Trigger, newest.Publisher)
	}
	if newest.AuthorName != "Test Author" || newest.AuthorEmail != "author@example.com" {
		t.Errorf("author = %q <%q>, want the commit author", newest.AuthorName, newest.AuthorEmail)
	}
	if newest.Reason != "" {
		t.Errorf("successful deploy has a reason: %q", newest.Reason)
	}
}

// Deploying a branch name resolves through the ref, not just raw SHAs.
func TestDeployResolvesBranchName(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	res, err := e.Deploy("live")
	if err != nil {
		t.Fatalf("deploying by branch name: %v", err)
	}
	if res.CommitSHA != f.good3 {
		t.Errorf("branch resolved to %q, want the tip %q", res.CommitSHA, f.good3)
	}
}

func TestDeployValidationFailure(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}

	_, err := e.Deploy(f.broken)
	if err == nil {
		t.Fatal("deploying a broken release did not fail")
	}
	var vErr *ValidationFailedError
	if !errors.As(err, &vErr) {
		t.Fatalf("error is %T, want *ValidationFailedError: %v", err, err)
	}
	if len(vErr.Errors) == 0 {
		t.Fatal("ValidationFailedError carries no errors")
	}
	first := vErr.Errors[0]
	if want := filepath.Join("site", "broken.pars"); first.File != want || first.Line != 1 {
		t.Errorf("validation error = %+v, want %s line 1", first, want)
	}

	// The previous release stays live; the partial directory is gone.
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current points at %q, want the previous release %q", got, f.good1)
	}
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.broken)); !os.IsNotExist(err) {
		t.Errorf("rejected release directory was not removed (err=%v)", err)
	}
	assertNoTempDirs(t, f.releasesDir())

	newest := recordEntries(t, e)[0]
	if newest.Outcome != OutcomeRejected {
		t.Errorf("recorded outcome %q, want %q", newest.Outcome, OutcomeRejected)
	}
	if !strings.Contains(newest.Reason, "broken.pars:1") {
		t.Errorf("recorded reason lacks file:line: %q", newest.Reason)
	}
}

// Only what this run created is removed on rejection: a directory that
// already existed as a past release survives being re-judged.
func TestDeployRejectedKeepsPreexistingDirectory(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := Materialise(f.repo, f.broken, f.releasesDir()); err != nil {
		t.Fatalf("pre-materialising: %v", err)
	}

	if _, err := e.Deploy(f.broken); err == nil {
		t.Fatal("deploying a broken release did not fail")
	}
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.broken)); err != nil {
		t.Errorf("a pre-existing release directory was removed: %v", err)
	}
}

func TestDeployUnknownRef(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	_, err := e.Deploy("no-such-branch")
	if err == nil {
		t.Fatal("deploying an unknown ref did not fail")
	}
	if !strings.Contains(err.Error(), "no-such-branch") {
		t.Errorf("error does not name the ref: %v", err)
	}

	// Nothing reached disk.
	if entries, err := os.ReadDir(f.releasesDir()); err == nil && len(entries) > 0 {
		t.Errorf("releases/ has %d entries after a failed resolve", len(entries))
	}
	if got := currentSHA(t, f.siteRoot); got != "" {
		t.Errorf("a release was activated: %q", got)
	}

	// The typo is in the record: failures are history too.
	newest := recordEntries(t, e)[0]
	if newest.Outcome != OutcomeFailed || newest.CommitSHA != "no-such-branch" {
		t.Errorf("recorded %q for %q, want failed for the ref as given", newest.Outcome, newest.CommitSHA)
	}
	if newest.Reason == "" {
		t.Error("failure recorded without a reason")
	}
}

func TestDeployNoValidateOverride(t *testing.T) {
	f := newEngineFixture(t)
	var out bytes.Buffer
	e := f.engine(&out)
	e.NoValidate = true

	res, err := e.Deploy(f.broken)
	if err != nil {
		t.Fatalf("--no-validate deploy failed: %v", err)
	}
	if res.Outcome != OutcomeDeployed {
		t.Errorf("Outcome = %q, want %q", res.Outcome, OutcomeDeployed)
	}
	if got := currentSHA(t, f.siteRoot); got != f.broken {
		t.Errorf("current points at %q, want the unvalidated release %q", got, f.broken)
	}
	if !strings.Contains(out.String(), "validation skipped") {
		t.Error("skipping validation was not announced")
	}
}

func TestDeployIdempotentNoOp(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("first deploy: %v", err)
	}
	releaseDir := filepath.Join(f.releasesDir(), f.good1)
	before, err := os.Stat(releaseDir)
	if err != nil {
		t.Fatal(err)
	}

	res, err := e.Deploy(f.good1)
	if err != nil {
		t.Fatalf("redeploying the active commit: %v", err)
	}
	if res.Outcome != OutcomeNoOp {
		t.Errorf("Outcome = %q, want %q", res.Outcome, OutcomeNoOp)
	}

	after, err := os.Stat(releaseDir)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("a no-op redeploy touched the release directory")
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current points at %q after a no-op", got)
	}

	entries := recordEntries(t, e)
	if len(entries) != 2 || entries[0].Outcome != OutcomeNoOp {
		t.Errorf("no-op not recorded: %+v", entries)
	}
}

func TestDeployHookFailureStaysLive(t *testing.T) {
	f := newEngineFixture(t)
	var out bytes.Buffer
	e := f.engine(&out)

	res, err := e.Deploy(f.hookFail)
	if err != nil {
		t.Fatalf("a hook failure must not fail the deploy: %v", err)
	}
	if res.Outcome != OutcomeDeployed {
		t.Errorf("Outcome = %q, want %q", res.Outcome, OutcomeDeployed)
	}
	if !strings.Contains(res.Reason, "deliberate hook failure") {
		t.Errorf("Reason does not carry the hook error: %q", res.Reason)
	}

	// No rollback: the release with the failed hook IS live.
	if got := currentSHA(t, f.siteRoot); got != f.hookFail {
		t.Errorf("current points at %q, want %q — the hook failure must not roll back", got, f.hookFail)
	}
	if !strings.Contains(out.String(), "DEPLOY WARNING") {
		t.Errorf("hook failure was not reported loudly:\n%s", out.String())
	}

	newest := recordEntries(t, e)[0]
	if newest.Outcome != OutcomeDeployed || !strings.Contains(newest.Reason, "deliberate hook failure") {
		t.Errorf("recorded entry = %+v, want deployed with the hook failure in the reason", newest)
	}
}

func TestDeployAfterActivateCallback(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)
	called := 0
	e.AfterActivate = func() { called++ }

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Errorf("AfterActivate ran %d times, want 1", called)
	}

	// Not called on a no-op: nothing was activated.
	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Errorf("AfterActivate ran on a no-op (%d calls)", called)
	}
}

func TestRollbackToPrevious(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Deploy(f.good2); err != nil {
		t.Fatal(err)
	}

	res, err := e.Rollback("")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if res.Outcome != OutcomeRolledBack || res.CommitSHA != f.good1 {
		t.Errorf("rollback = %q %q, want rolled-back to %q", res.Outcome, res.CommitSHA, f.good1)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current points at %q, want %q", got, f.good1)
	}

	newest := recordEntries(t, e)[0]
	if newest.Trigger != TriggerRollback || newest.Outcome != OutcomeRolledBack || newest.CommitSHA != f.good1 {
		t.Errorf("rollback entry = %+v", newest)
	}

	// And rolling forward again by explicit SHA prefix.
	res, err = e.Rollback(f.good2[:10])
	if err != nil {
		t.Fatalf("rollback by prefix: %v", err)
	}
	if res.CommitSHA != f.good2 || currentSHA(t, f.siteRoot) != f.good2 {
		t.Errorf("prefix rollback activated %q, want %q", res.CommitSHA, f.good2)
	}
}

func TestRollbackBySequenceNumber(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Deploy(f.good2); err != nil {
		t.Fatal(err)
	}

	// The first deploy is seq 1 in the record.
	res, err := e.Rollback("1")
	if err != nil {
		t.Fatalf("rollback by seq: %v", err)
	}
	if res.CommitSHA != f.good1 {
		t.Errorf("seq 1 resolved to %q, want %q", res.CommitSHA, f.good1)
	}
}

func TestRollbackRefusedWhenNothingPrevious(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	_, err := e.Rollback("")
	if err == nil {
		t.Fatal("rollback with no previous release did not fail")
	}
	if !strings.Contains(err.Error(), "nothing to roll back") {
		t.Errorf("unhelpful refusal: %v", err)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("refusal changed current to %q", got)
	}
}

func TestRollbackRefusedWhenPruned(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Deploy(f.good2); err != nil {
		t.Fatal(err)
	}
	// Simulate pruning the previous release out from under the record.
	if err := os.RemoveAll(filepath.Join(f.releasesDir(), f.good1)); err != nil {
		t.Fatal(err)
	}

	_, err := e.Rollback("")
	if err == nil {
		t.Fatal("rollback to a pruned release did not fail")
	}
	if !strings.Contains(err.Error(), "no longer on disk") {
		t.Errorf("refusal does not explain the prune: %v", err)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good2 {
		t.Errorf("refusal changed current to %q", got)
	}
	newest := recordEntries(t, e)[0]
	if newest.Outcome != OutcomeFailed || newest.Trigger != TriggerRollback {
		t.Errorf("pruned refusal not recorded: %+v", newest)
	}
}

func TestDeployRefusedWhileLocked(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil) // LockWait 0: refuse immediately

	held, err := AcquireLock(f.siteRoot, 0)
	if err != nil {
		t.Fatalf("taking the lock: %v", err)
	}
	defer held.Release()

	_, err = e.Deploy(f.good1)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("err = %v, want ErrLocked", err)
	}
	if got := currentSHA(t, f.siteRoot); got != "" {
		t.Errorf("a locked-out deploy activated %q", got)
	}
	newest := recordEntries(t, e)[0]
	if newest.Outcome != OutcomeFailed || !strings.Contains(newest.Reason, "another deploy") {
		t.Errorf("lock refusal not recorded: %+v", newest)
	}
}

// Two deploys of the same commit racing: the lock serialises them, so one
// deploys and the other sees an already-active commit and no-ops. Neither
// errors, nothing is left half-built.
func TestDeployRaceSerialises(t *testing.T) {
	f := newEngineFixture(t)

	results := make([]*Result, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		e := f.engine(nil)
		e.LockWait = 10 * time.Second
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = e.Deploy(f.good1)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("racer %d failed: %v", i, err)
		}
	}
	outcomes := map[string]int{}
	for _, res := range results {
		outcomes[res.Outcome]++
	}
	if outcomes[OutcomeDeployed] != 1 || outcomes[OutcomeNoOp] != 1 {
		t.Errorf("outcomes = %v, want exactly one deployed and one no-op", outcomes)
	}

	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current points at %q", got)
	}
	assertNoTempDirs(t, f.releasesDir())

	e := f.engine(nil)
	entries := recordEntries(t, e)
	if len(entries) != 2 {
		t.Errorf("record has %d entries, want 2 (one per racer)", len(entries))
	}
}

func TestDeployPrunesOldReleases(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)
	e.Keep = 1

	for _, sha := range []string{f.good1, f.good2} {
		if _, err := e.Deploy(sha); err != nil {
			t.Fatalf("deploying %s: %v", sha, err)
		}
	}
	res, err := e.Deploy(f.good3)
	if err != nil {
		t.Fatal(err)
	}

	// keep is clamped to 2 and the previously activated release is always
	// protected, so only good1 - old enough AND no longer the previous
	// release - goes during good3's deploy.
	if len(res.Pruned) != 1 || filepath.Base(res.Pruned[0]) != f.good1 {
		t.Errorf("Pruned = %v, want just the good1 release", res.Pruned)
	}
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.good1)); !os.IsNotExist(err) {
		t.Errorf("old release %s survived pruning (err=%v)", shortSHA(f.good1), err)
	}
	// The previous activated release is never pruned - a lagging server may
	// still serve it, and it is what rollback rolls back to.
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.good2)); err != nil {
		t.Errorf("the previous release was pruned: %v", err)
	}
	// The active release is never pruned, keep limit or not.
	if _, err := os.Stat(filepath.Join(f.releasesDir(), f.good3)); err != nil {
		t.Errorf("the active release was pruned: %v", err)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good3 {
		t.Errorf("current points at %q", got)
	}
}
