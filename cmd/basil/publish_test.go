package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// publishFixture is a bare repository seeded with one commit on the release
// branch (basil.yaml + a site file) plus a working clone to publish from. It
// is deliberately lighter than deployFixture: publish is a client verb, so it
// needs a clone and a reachable origin, not a full site root.
type publishFixture struct {
	bare   string // the "server" bare repository (origin)
	work   string // the clone publish runs in
	branch string // the release branch (deploy.branch in the clone's basil.yaml)
	seed   string // the seed commit SHA (origin's release-branch tip)
}

func newPublishFixture(t *testing.T, branch string) *publishFixture {
	t.Helper()
	requireGit(t)

	tmp := t.TempDir()
	bare := filepath.Join(tmp, "site.git")
	testGit(t, tmp, "init", "--bare", "--initial-branch="+branch, bare)

	// Seed the release branch: basil.yaml carries deploy.branch so the clone
	// (and publish) read the branch from committed config, not a default.
	seedDir := filepath.Join(tmp, "seed")
	testGit(t, tmp, "init", "--initial-branch="+branch, seedDir)
	testGit(t, seedDir, "config", "user.name", "Seed Author")
	testGit(t, seedDir, "config", "user.email", "seed@example.com")
	writePubFile(t, filepath.Join(seedDir, "basil.yaml"),
		"server:\n  host: localhost\ndeploy:\n  branch: "+branch+"\n")
	writePubFile(t, filepath.Join(seedDir, "site", "index.pars"), "<h1>\"v1\"</h1>\n")
	testGit(t, seedDir, "add", "-A")
	testGit(t, seedDir, "commit", "--quiet", "--no-verify", "-m", "seed")
	// Push explicitly to the bare repo (the seed has no remote configured).
	testGit(t, seedDir, "push", "--quiet", bare, branch+":"+branch)
	seed := testGit(t, seedDir, "rev-parse", "HEAD")

	work := filepath.Join(tmp, "work")
	testGit(t, tmp, "clone", "--quiet", bare, work)
	testGit(t, work, "config", "user.name", "Test Author")
	testGit(t, work, "config", "user.email", "author@example.com")

	return &publishFixture{bare: bare, work: work, branch: branch, seed: seed}
}

// commitLocal writes and commits a file in the clone WITHOUT pushing, so a
// following publish has something to send. Returns the new commit SHA.
func (f *publishFixture) commitLocal(t *testing.T, name, content, msg string) string {
	t.Helper()
	writePubFile(t, filepath.Join(f.work, name), content)
	testGit(t, f.work, "add", "-A")
	testGit(t, f.work, "commit", "--quiet", "--no-verify", "-m", msg)
	return testGit(t, f.work, "rev-parse", "HEAD")
}

// originTip returns the release-branch tip on the bare origin, for asserting a
// ref did or did not move.
func (f *publishFixture) originTip(t *testing.T) string {
	t.Helper()
	return testGit(t, f.bare, "rev-parse", f.branch)
}

// installRejectHook makes the bare repo refuse the next push with a
// file:line diagnostic, mimicking the validation gate's rejection.
func (f *publishFixture) installRejectHook(t *testing.T) {
	t.Helper()
	hook := "#!/bin/sh\n" +
		"echo 'site/broken.pars:1:1: syntax error: unexpected =' >&2\n" +
		"echo 'Release rejected. The live site is unchanged.' >&2\n" +
		"exit 1\n"
	path := filepath.Join(f.bare, "hooks", "pre-receive")
	if err := os.WriteFile(path, []byte(hook), 0o755); err != nil {
		t.Fatalf("installing reject hook: %v", err)
	}
}

// installAcceptHook makes the bare repo print a remote: line on a successful
// push, so tests can prove the server's output is streamed through.
func (f *publishFixture) installAcceptHook(t *testing.T) {
	t.Helper()
	hook := "#!/bin/sh\ncat >/dev/null\necho 'Deployed the release' \nexit 0\n"
	path := filepath.Join(f.bare, "hooks", "post-receive")
	if err := os.WriteFile(path, []byte(hook), 0o755); err != nil {
		t.Fatalf("installing accept hook: %v", err)
	}
}

func writePubFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- publish success --------------------------------------------------------

func TestPublish_SuccessPushesAndStreams(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	head := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}
	if code := exitCode(err); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	got := out.String()
	// The server's remote: line was streamed live.
	if !strings.Contains(got, "remote: Deployed the release") {
		t.Errorf("streamed remote: output not seen:\n%s", got)
	}
	// The deployed commit is reported.
	if !strings.Contains(got, "Published "+head[:12]) {
		t.Errorf("deployed sha not reported:\n%s", got)
	}
	// The ref actually moved on origin.
	if tip := f.originTip(t); tip != head {
		t.Errorf("origin %q is at %s, want %s", f.branch, tip, head)
	}
}

// --- publish rejection ------------------------------------------------------

func TestPublish_RejectionExitsNonZeroAndLeavesOriginUnmoved(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installRejectHook(t)
	f.commitLocal(t, "site/broken.pars", "let x = = 2\n", "broken")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatal("a rejected push must surface as an error")
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	got := out.String()
	if !strings.Contains(got, "remote:") || !strings.Contains(got, "broken.pars:1") {
		t.Errorf("streamed remote: rejection with file:line not seen:\n%s", got)
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("origin moved to %s; a rejected publish must not move the release ref (want %s)", tip, f.seed)
	}
}

// --- --dry-run --------------------------------------------------------------

func TestPublish_DryRunShowsPlanButPushesNothing(t *testing.T) {
	f := newPublishFixture(t, "live")
	head := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--dry-run"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("dry-run failed: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "site/index.pars") {
		t.Errorf("dry-run did not list the changed file:\n%s", got)
	}
	if !strings.Contains(got, "--dry-run: nothing was pushed") {
		t.Errorf("dry-run did not announce it pushed nothing:\n%s", got)
	}
	if !strings.Contains(got, head[:12]) {
		t.Errorf("dry-run did not name the commit to publish:\n%s", got)
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("origin moved to %s during --dry-run; nothing must be pushed (want %s)", tip, f.seed)
	}
}

// --- confirmation gate ------------------------------------------------------

func TestPublish_PipedNoAborts(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work}, strings.NewReader("n\n"), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("an aborted publish is not an error: %v", err)
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("abort was not reported:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("origin moved after a declined prompt (want %s, got %s)", f.seed, tip)
	}
}

func TestPublish_ConfirmationNotSkippableWithoutYes(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// Empty stdin (EOF) is silence, not consent: publish must abort.
	var out bytes.Buffer
	err := runPublish([]string{f.work}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("aborted publish returned an error: %v", err)
	}
	if !strings.Contains(out.String(), "Publish 1 commit to") {
		t.Errorf("the confirmation prompt was not shown:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("empty stdin should abort, not proceed:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("origin moved without confirmation (want %s, got %s)", f.seed, tip)
	}
}

func TestPublish_YesProceedsWithoutConsumingStdin(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	head := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// stdin holds a "n" that must be ignored because --yes is set.
	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader("n\n"), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("--yes publish failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "Cancelled") {
		t.Errorf("--yes must not consult stdin:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != head {
		t.Errorf("--yes did not publish (origin at %s, want %s)", tip, head)
	}
}

// --- nothing to publish -----------------------------------------------------

func TestPublish_NothingToPublish(t *testing.T) {
	f := newPublishFixture(t, "live")
	// No local commits: HEAD already equals origin's release tip.
	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Nothing to publish") {
		t.Errorf("publish should report nothing to do:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("origin moved with nothing to publish")
	}
}

// --- server unreachable: drift degrades to a warning ------------------------

func TestPublish_UnreachableOriginDegradesToWarning(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// Point origin at nothing: ls-remote can no longer reach it, but the
	// clone still has its cached remote-tracking ref to plan from.
	testGit(t, f.work, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--dry-run"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("publish should degrade, not fail: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "could not reach origin") {
		t.Errorf("unreachable origin was not reported as a warning:\n%s", got)
	}
	if !strings.Contains(got, "site/index.pars") {
		t.Errorf("plan should still be computed from the cached ref:\n%s", got)
	}
}

// --- deploy.branch: main round trip -----------------------------------------

func TestPublish_NonDefaultBranchRoundTrip(t *testing.T) {
	f := newPublishFixture(t, "main")
	f.installAcceptHook(t)
	head := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("publish to main failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `to "main"`) {
		t.Errorf("publish did not target the configured branch main:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != head {
		t.Errorf("main is at %s, want %s", tip, head)
	}
}

// --- actionable errors ------------------------------------------------------

func TestPublish_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := runPublish([]string{dir}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatal("publish outside a git repo must error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error is not actionable: %v", err)
	}
}

func TestPublish_UsageErrorExitsTwo(t *testing.T) {
	var out bytes.Buffer
	err := runPublish([]string{"--bogus"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatal("an unknown flag must be a usage error")
	}
	if code := exitCode(err); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("usage not printed:\n%s", out.String())
	}
}

// --- dispatch ---------------------------------------------------------------

func TestRun_DispatchesPublish(t *testing.T) {
	f := newPublishFixture(t, "live")
	var out bytes.Buffer
	// No local commits and --yes: publish reports nothing to do and exits 0.
	if err := run(t.Context(), []string{"publish", f.work, "--yes"}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("run dispatch to publish failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Nothing to publish") {
		t.Errorf("publish did not run via dispatch:\n%s", out.String())
	}
}
