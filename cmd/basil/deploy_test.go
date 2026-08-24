package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/deploy"
)

// emptyEnv is getenv for tests: nothing comes from the environment.
func emptyEnv(string) string { return "" }

// deployFixture is a real site root made by `basil --init` (bare repo,
// release 1, current) plus a working clone to push new commits from. The
// host is localhost so `basil check`'s DNS check works without a network.
type deployFixture struct {
	root string // site root
	work string // clone of <root>/site.git
}

func newDeployFixture(t *testing.T) *deployFixture {
	t.Helper()
	requireGit(t)

	tmp := t.TempDir()
	root := filepath.Join(tmp, "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Host = "localhost"
	if err := runInitCommand(opts); err != nil {
		t.Fatalf("runInitCommand: %v", err)
	}

	// Disarm the receive hooks --init installed. Inside `go test`,
	// os.Executable() is the TEST binary, so a fixture push firing
	// pre-receive would re-enter the test suite recursively instead of
	// running the CLI. The hook logic is exercised by driving runFromHook
	// directly (fromhook_test.go); real hook execution belongs to the
	// end-to-end verification against a real basil binary.
	disarmHooks(t, filepath.Join(root, "site.git"))

	work := filepath.Join(tmp, "work")
	testGit(t, tmp, "clone", "--quiet", filepath.Join(root, "site.git"), work)
	testGit(t, work, "config", "user.name", "Test Author")
	testGit(t, work, "config", "user.email", "author@example.com")
	return &deployFixture{root: root, work: work}
}

// commitAndPush writes one file in the clone, commits, pushes to the release
// branch and returns the new commit's full SHA.
func (f *deployFixture) commitAndPush(t *testing.T, name, content, msg string) string {
	t.Helper()
	path := filepath.Join(f.work, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	testGit(t, f.work, "add", "-A")
	testGit(t, f.work, "commit", "--quiet", "--no-verify", "-m", msg)
	testGit(t, f.work, "push", "--quiet", "origin", releaseBranch)
	return testGit(t, f.work, "rev-parse", "HEAD")
}

// currentSHA reads which release the site root's `current` link points at.
func (f *deployFixture) currentSHA(t *testing.T) string {
	t.Helper()
	target, err := os.Readlink(filepath.Join(f.root, "current"))
	if err != nil {
		t.Fatalf("reading current: %v", err)
	}
	return filepath.Base(target)
}

// recordEntries reads the deploy record the way the engine writes it.
func (f *deployFixture) recordEntries(t *testing.T) []deploy.Entry {
	t.Helper()
	rec, err := deploy.OpenRecord(filepath.Join(f.root, "data", "deploy.db"))
	if err != nil {
		t.Fatalf("opening deploy record: %v", err)
	}
	defer rec.Close()
	entries, err := rec.List(0)
	if err != nil {
		t.Fatalf("listing deploy record: %v", err)
	}
	return entries
}

// disarmHooks replaces the repository's receive hooks with no-ops so test
// pushes are stored without re-entering the test binary.
func disarmHooks(t *testing.T, repoDir string) {
	t.Helper()
	for _, name := range []string{"pre-receive", "post-receive"} {
		path := filepath.Join(repoDir, "hooks", name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("disarming %s: %v", name, err)
		}
	}
}

func testGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// --- deploy ---------------------------------------------------------------

func TestDeployCommand_EndToEnd(t *testing.T) {
	f := newDeployFixture(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// Flags after the positional, the way the usage examples write it:
	// flag.Parse stops at the first positional, so this order needs the
	// re-parse in runDeployCommand.
	var stdout, stderr bytes.Buffer
	err := runDeployCommand([]string{sha, "--site", f.root}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("deploy failed: %v\nstderr: %s", err, stderr.String())
	}
	if code := exitCode(err); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	if got := f.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want %s", got, sha)
	}

	out := stdout.String()
	if !strings.Contains(out, "deployed "+sha[:12]) {
		t.Errorf("no deployed line on stdout:\n%s", out)
	}
	if !strings.Contains(out, "Live: "+sha[:12]) {
		t.Errorf("no Live: line on stdout:\n%s", out)
	}

	// Two entries: release 1 written by --init, then this deploy.
	entries := f.recordEntries(t)
	if len(entries) != 2 {
		t.Fatalf("record has %d entries, want 2 (init + this deploy)", len(entries))
	}
	e := entries[0]
	if e.CommitSHA != sha || e.Outcome != deploy.OutcomeDeployed {
		t.Errorf("record entry = %s/%s, want %s/%s", e.CommitSHA, e.Outcome, sha, deploy.OutcomeDeployed)
	}
	if e.Trigger != deploy.TriggerCLI {
		t.Errorf("trigger = %q, want %q", e.Trigger, deploy.TriggerCLI)
	}
	if !strings.HasPrefix(e.Publisher, "cli:") {
		t.Errorf("publisher = %q, want a cli:<user> identity", e.Publisher)
	}
	if e.AuthorName != "Test Author" {
		t.Errorf("author = %q, want the commit author", e.AuthorName)
	}
}

func TestDeployCommand_BrokenCommitIsRejected(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/broken.pars", "let x = = 2\n", "broken parsley")

	var stdout, stderr bytes.Buffer
	err := runDeployCommand([]string{"--site", f.root, sha}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("a broken release was accepted")
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	// The rejection must carry file:line diagnostics and the reassurance.
	errOut := stderr.String()
	if !strings.Contains(errOut, "site/broken.pars:1") {
		t.Errorf("stderr has no file:line diagnostic:\n%s", errOut)
	}
	if !strings.Contains(errOut, "The live site is unchanged") {
		t.Errorf("stderr does not say the live site is safe:\n%s", errOut)
	}

	if got := f.currentSHA(t); got != before {
		t.Errorf("current moved to %s; a rejected release must not activate", got)
	}
	if _, statErr := os.Stat(filepath.Join(f.root, "releases", sha)); !os.IsNotExist(statErr) {
		t.Error("the rejected release directory was left on disk")
	}

	// Newest entry is the rejection; the one below it is --init's release 1.
	entries := f.recordEntries(t)
	if len(entries) != 2 || entries[0].Outcome != deploy.OutcomeRejected {
		t.Errorf("the rejection was not recorded: %+v", entries)
	}
}

func TestDeployCommand_UnresolvableRef(t *testing.T) {
	f := newDeployFixture(t)

	var stdout, stderr bytes.Buffer
	err := runDeployCommand([]string{"--site", f.root, "no-such-branch"}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("an unresolvable ref was accepted")
	}
	if !strings.Contains(err.Error(), "no-such-branch") {
		t.Errorf("the error does not name the bad ref: %v", err)
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

// --- rollback --------------------------------------------------------------

func TestRollbackCommand_HappyPath(t *testing.T) {
	f := newDeployFixture(t)
	sha2 := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	sha3 := f.commitAndPush(t, "site/index.pars", "<h1>\"v3\"</h1>\n", "v3")

	var out bytes.Buffer
	if err := runDeployCommand([]string{"--site", f.root, sha2}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy v2: %v\n%s", err, out.String())
	}
	if err := runDeployCommand([]string{"--site", f.root, sha3}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy v3: %v\n%s", err, out.String())
	}

	var stdout, stderr bytes.Buffer
	if err := runRollbackCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("rollback failed: %v\nstderr: %s", err, stderr.String())
	}
	if got := f.currentSHA(t); got != sha2 {
		t.Errorf("current points at %s, want the previous release %s", got, sha2)
	}
	if !strings.Contains(stdout.String(), "Live: "+sha2[:12]) {
		t.Errorf("rollback does not say what is live now:\n%s", stdout.String())
	}
}

func TestRollbackCommand_NothingToRollBackTo(t *testing.T) {
	f := newDeployFixture(t)

	var stdout, stderr bytes.Buffer
	err := runRollbackCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("rollback on a fresh site should refuse")
	}
	if !strings.Contains(err.Error(), "nothing to roll back to") {
		t.Errorf("the error does not name the problem: %v", err)
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestRollbackCommand_PrunedRelease(t *testing.T) {
	f := newDeployFixture(t)
	sha2 := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	sha3 := f.commitAndPush(t, "site/index.pars", "<h1>\"v3\"</h1>\n", "v3")

	var out bytes.Buffer
	if err := runDeployCommand([]string{"--site", f.root, sha2}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy v2: %v\n%s", err, out.String())
	}
	if err := runDeployCommand([]string{"--site", f.root, sha3}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy v3: %v\n%s", err, out.String())
	}
	// Simulate pruning: remove v2's release directory.
	if err := os.RemoveAll(filepath.Join(f.root, "releases", sha2)); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runRollbackCommand([]string{"--site", f.root, sha2[:8]}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("rollback to a pruned release should refuse")
	}
	if !strings.Contains(err.Error(), "basil deploy") {
		t.Errorf("the error does not suggest basil deploy <sha>: %v", err)
	}
}

// --- releases --------------------------------------------------------------

func TestReleasesCommand_TableWithLiveMarker(t *testing.T) {
	f := newDeployFixture(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	if err := runDeployCommand([]string{"--site", f.root, sha}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy: %v\n%s", err, out.String())
	}

	var stdout, stderr bytes.Buffer
	if err := runReleasesCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("releases failed: %v", err)
	}
	table := stdout.String()
	for _, want := range []string{"SEQ", "RELEASE", "PUBLISHER", "AUTHOR", sha[:12], "deployed", "Test Author"} {
		if !strings.Contains(table, want) {
			t.Errorf("table is missing %q:\n%s", want, table)
		}
	}
	if !strings.Contains(table, "* ") || !strings.Contains(table, "* = the live release") {
		t.Errorf("the live release is not marked:\n%s", table)
	}
}

func TestReleasesCommand_EmptyRecord(t *testing.T) {
	f := newDeployFixture(t)

	// --init writes release 1 into the record; remove the database to get a
	// genuinely recordless site (e.g. a hand-built layout).
	dbPath := filepath.Join(f.root, "data", "deploy.db")
	if err := os.Remove(dbPath); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runReleasesCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("an empty record must not be an error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No deploys recorded yet") {
		t.Errorf("no friendly empty message:\n%s", stdout.String())
	}
	// Reading must not create the database.
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Error("basil releases created the deploy record database")
	}
}

// --- status ----------------------------------------------------------------

func TestStatusCommand_AheadCount(t *testing.T) {
	f := newDeployFixture(t)
	sha2 := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	if err := runDeployCommand([]string{"--site", f.root, sha2}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy: %v\n%s", err, out.String())
	}

	// In sync after the deploy.
	var stdout, stderr bytes.Buffer
	if err := runStatusCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "live: "+sha2[:12]) {
		t.Errorf("status does not report the live release:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "matches the live release") {
		t.Errorf("status should report the branch in sync:\n%s", stdout.String())
	}

	// Push one commit without deploying: the branch is now ahead.
	f.commitAndPush(t, "site/index.pars", "<h1>\"v3\"</h1>\n", "v3")
	stdout.Reset()
	if err := runStatusCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "1 commit ahead") {
		t.Errorf("status does not report the ahead count:\n%s", got)
	}
	if !strings.Contains(got, "basil deploy "+releaseBranch) {
		t.Errorf("status does not say how to catch up:\n%s", got)
	}
}

// --- check -----------------------------------------------------------------

func TestCheckCommand_HealthyFixturePasses(t *testing.T) {
	f := newDeployFixture(t)

	var stdout, stderr bytes.Buffer
	err := runCheckCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("check failed on a healthy site: %v\n%s", err, stdout.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"ok    config",
		"ok    site root",
		"ok    release",
		"ok    repository",
		"ok    repository placement",
		"ok    server.host: localhost",
		"ok    dns",
		"All checks passed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("check output is missing %q:\n%s", want, out)
		}
	}
}

func TestCheckCommand_RepoInsideServedRootFails(t *testing.T) {
	f := newDeployFixture(t)

	// Point public_dir at the site root, which contains site.git.
	cfgPath := filepath.Join(f.root, "current", "basil.yaml")
	yaml, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(yaml), "public_dir: ./public", "public_dir: ../..", 1)
	if edited == string(yaml) {
		t.Fatal("could not rewrite public_dir in the fixture config")
	}
	if err := os.WriteFile(cfgPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err = runCheckCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("check passed with the repository inside public_dir")
	}
	out := stdout.String()
	if !strings.Contains(out, "FAIL  repository placement") {
		t.Errorf("no repository placement failure:\n%s", out)
	}
	if !strings.Contains(out, "public_dir") {
		t.Errorf("the failure does not name the served root:\n%s", out)
	}
	if !strings.Contains(err.Error(), "check(s) failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- usage and exit codes ---------------------------------------------------

func TestDeployCommands_UsageErrorsExit2(t *testing.T) {
	cases := []struct {
		name string
		run  func(stdout, stderr *bytes.Buffer) error
	}{
		{"deploy missing arg", func(stdout, stderr *bytes.Buffer) error {
			return runDeployCommand(nil, stdout, stderr, emptyEnv)
		}},
		{"deploy unknown flag", func(stdout, stderr *bytes.Buffer) error {
			return runDeployCommand([]string{"--bogus"}, stdout, stderr, emptyEnv)
		}},
		{"deploy extra arg", func(stdout, stderr *bytes.Buffer) error {
			return runDeployCommand([]string{"one", "two"}, stdout, stderr, emptyEnv)
		}},
		{"deploy site and config together", func(stdout, stderr *bytes.Buffer) error {
			return runDeployCommand([]string{"--site", "a", "--config", "b", "live"}, stdout, stderr, emptyEnv)
		}},
		{"rollback extra args", func(stdout, stderr *bytes.Buffer) error {
			return runRollbackCommand([]string{"one", "two"}, stdout, stderr, emptyEnv)
		}},
		{"releases unknown flag", func(stdout, stderr *bytes.Buffer) error {
			return runReleasesCommand([]string{"--bogus"}, stdout, stderr, emptyEnv)
		}},
		{"status unexpected arg", func(stdout, stderr *bytes.Buffer) error {
			return runStatusCommand([]string{"what"}, stdout, stderr, emptyEnv)
		}},
		{"check unknown flag", func(stdout, stderr *bytes.Buffer) error {
			return runCheckCommand([]string{"--bogus"}, stdout, stderr, emptyEnv)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := tc.run(&stdout, &stderr)
			if err == nil {
				t.Fatal("expected a usage error")
			}
			if code := exitCode(err); code != 2 {
				t.Errorf("exit code = %d, want 2 (err: %v)", code, err)
			}
			if !strings.Contains(stderr.String(), "Usage:") {
				t.Errorf("usage was not printed to stderr:\n%s", stderr.String())
			}
		})
	}
}

// The dispatch in run() must route the new subcommands, not fall through to
// the server.
func TestRun_DispatchesDeploySubcommands(t *testing.T) {
	f := newDeployFixture(t)

	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"releases", "--site", f.root}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("run dispatch failed: %v", err)
	}
	// A fresh --init site already has release 1 in its record.
	if !strings.Contains(stdout.String(), deploy.TriggerInit) || !strings.Contains(stdout.String(), "SEQ") {
		t.Errorf("releases did not run (or release 1 is missing):\n%s", stdout.String())
	}
}

func TestExitCode(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Errorf("exitCode(nil) = %d, want 0", got)
	}
	if got := exitCode(os.ErrNotExist); got != 1 {
		t.Errorf("exitCode(plain error) = %d, want 1", got)
	}
	if got := exitCode(&usageError{err: os.ErrInvalid}); got != 2 {
		t.Errorf("exitCode(usage error) = %d, want 2", got)
	}
	if got := exitCode(&hookFailedError{msg: "hook failed"}); got != 3 {
		t.Errorf("exitCode(hook failure) = %d, want 3", got)
	}
}

// A deploy whose release went live but whose post-deploy hook failed exits 3:
// 0 would hide the failure from scripts, 1 would claim the deploy failed.
func TestDeployCommand_HookFailureExitsThree(t *testing.T) {
	f := newDeployFixture(t)
	sha := f.commitAndPush(t, "deploy.pars", "fail(\"deliberate hook failure\")\n", "failing hook")

	var stdout, stderr bytes.Buffer
	err := runDeployCommand([]string{"--site", f.root, sha}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("a hook failure must surface in the exit code")
	}
	if code := exitCode(err); code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}

	// The release IS live — the exit code flags the hook, not the deploy.
	if got := f.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want %s: a failed hook must not roll back", got, sha)
	}
	out := stdout.String()
	if !strings.Contains(out, "Live: "+sha[:12]) {
		t.Errorf("no Live: line on stdout:\n%s", out)
	}
	if !strings.Contains(out, "DEPLOY WARNING") {
		t.Errorf("hook failure was not reported loudly:\n%s", out)
	}

	// basil releases shows the hook-failure reason on the deployed row too,
	// not only on failed/rejected rows.
	var table, tableErr bytes.Buffer
	if err := runReleasesCommand([]string{"--site", f.root}, &table, &tableErr, emptyEnv); err != nil {
		t.Fatalf("releases: %v", err)
	}
	if !strings.Contains(table.String(), "deliberate hook failure") {
		t.Errorf("releases hides the deployed row's hook-failure reason:\n%s", table.String())
	}
}

// clip counts runes, not bytes: a multi-byte name must never be cut
// mid-character.
func TestClipRuneSafe(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"héllo wörld", 8, "héllo..."},
		{"日本語日本語", 5, "日本..."},
		{"日本語", 2, "日本"}, // n <= 3: no room for an ellipsis
	}
	for _, c := range cases {
		if got := clip(c.in, c.n); got != c.want {
			t.Errorf("clip(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
