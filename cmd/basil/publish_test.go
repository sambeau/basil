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
	branch string // the release branch (the bare repository's HEAD)
	seed   string // the seed commit SHA (origin's release-branch tip)
}

func newPublishFixture(t *testing.T, branch string) *publishFixture {
	t.Helper()
	requireGit(t)

	tmp := t.TempDir()
	bare := filepath.Join(tmp, "site.git")
	testGit(t, tmp, "init", "--bare", "--initial-branch="+branch, bare)

	// Seed the release branch. Nothing in basil.yaml names it: publish asks
	// origin for its HEAD, which `git init --bare --initial-branch` above
	// already points at the branch under test (FEAT-157).
	seedDir := filepath.Join(tmp, "seed")
	testGit(t, tmp, "init", "--initial-branch="+branch, seedDir)
	testGit(t, seedDir, "config", "user.name", "Seed Author")
	testGit(t, seedDir, "config", "user.email", "seed@example.com")
	writePubFile(t, filepath.Join(seedDir, "basil.yaml"), "server:\n  host: localhost\n")
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

// graduatedClone builds a local project whose history is UNRELATED to the bare
// origin's starter commit and connects it as a graduated site would: an
// independently-initialised repo (its own root commit, not a clone of the
// server), a basil.yaml so publish finds its config, `git remote add origin`,
// and a push of its own branch that stores it without touching the release
// branch. This is the state BUG-037 is about - the first `basil publish` from
// it can only be a non-fast-forward. Returns the local repo path and its HEAD.
func (f *publishFixture) graduatedClone(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	local := filepath.Join(tmp, "mysite")
	testGit(t, tmp, "init", "--initial-branch="+f.branch, local)
	testGit(t, local, "config", "user.name", "Local Author")
	testGit(t, local, "config", "user.email", "local@example.com")
	writePubFile(t, filepath.Join(local, "basil.yaml"), "server:\n  host: localhost\n")
	writePubFile(t, filepath.Join(local, "site", "index.pars"), "<h1>\"written on my laptop\"</h1>\n")
	testGit(t, local, "add", "-A")
	testGit(t, local, "commit", "--quiet", "--no-verify", "-m", "my first page")
	head := testGit(t, local, "rev-parse", "HEAD")
	testGit(t, local, "remote", "add", "origin", f.bare)
	// Store the branch on origin without publishing it: the release branch is
	// still the server's starter commit, unrelated to this history.
	testGit(t, local, "push", "--quiet", "origin", f.branch+":refs/heads/main")
	return local, head
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

// --- server unreachable with no cached base: not a first publish ------------

// TestPublish_UnreachableOriginNoCachedRefIsNotFirstPublish reproduces the
// case where origin cannot be reached AND the clone has no cached
// remote-tracking ref: the plan cannot be computed, so publish must not
// mislabel it "first publish" nor dump the whole tree as the change set.
func TestPublish_UnreachableOriginNoCachedRefIsNotFirstPublish(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// Drop the cached remote-tracking ref and point origin at nothing, so
	// ls-remote fails and there is no cached base to plan from.
	testGit(t, f.work, "update-ref", "-d", "refs/remotes/origin/live")
	testGit(t, f.work, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--dry-run"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err != nil {
		t.Fatalf("--dry-run should not fail in this state: %v\n%s", err, out.String())
	}
	got := out.String()
	if strings.Contains(got, "first publish") {
		t.Errorf("unreachable origin with no cached ref must not claim first publish:\n%s", got)
	}
	// The whole tree must not be dumped as the plan (basil.yaml is only ever
	// listed by the whole-tree change set).
	if strings.Contains(got, "basil.yaml") {
		t.Errorf("the whole tree must not be dumped as the plan:\n%s", got)
	}
	if !strings.Contains(got, "Cannot compute the publish plan") {
		t.Errorf("expected the plan-unknown message:\n%s", got)
	}
}

// TestPublish_UnreachableOriginNoCachedRefDoesNotPush confirms that without
// --dry-run this same state errors out and never pushes.
func TestPublish_UnreachableOriginNoCachedRefDoesNotPush(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	testGit(t, f.work, "update-ref", "-d", "refs/remotes/origin/live")
	testGit(t, f.work, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatal("publish must not proceed when the plan cannot be computed")
	}
	if strings.Contains(out.String(), "first publish") {
		t.Errorf("must not claim first publish:\n%s", out.String())
	}
	// origin still points at the gone path, so nothing could have been pushed.
}

// --- a reachable origin that advertises no release branch -------------------

// Git advertises HEAD as a symref only when it resolves, so an origin whose
// HEAD names a branch nobody has pushed yet looks exactly like a detached one
// from a clone. publish refuses both and names the fix for each, rather than
// guessing a branch and publishing somewhere nobody asked for.
func TestPublish_ReachableOriginWithUnbornHEAD(t *testing.T) {
	f := newPublishFixture(t, "live")
	testGit(t, f.bare, "symbolic-ref", "HEAD", "refs/heads/prod")
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--dry-run"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatalf("publish must not guess a branch when origin advertises none:\n%s", out.String())
	}
	for _, want := range []string{"basil check", "symbolic-ref HEAD refs/heads/", "git push origin HEAD:refs/heads/"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %q: %v", want, err)
		}
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("live moved to %s; nothing should have been pushed", tip)
	}
}

// --- first-use refspec config -----------------------------------------------

// TestPublish_SetsPushRefspecOnExistingBranch verifies a successful publish
// from a clone with no remote.origin.push sets it to the release refspec, even
// when the branch already exists on origin (base != "").
func TestPublish_SetsPushRefspecOnExistingBranch(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// A fresh clone has no remote.origin.push configured.
	if existing, err := gitOutput(f.work, "config", "--get", "remote.origin.push"); err == nil && strings.TrimSpace(existing) != "" {
		t.Fatalf("precondition failed: clone already has remote.origin.push = %q", existing)
	}

	var out bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}

	got := testGit(t, f.work, "config", "--get", "remote.origin.push")
	if want := "HEAD:refs/heads/live"; got != want {
		t.Errorf("remote.origin.push = %q, want %q", got, want)
	}
}

// --- first publish from a graduated project (BUG-037) -----------------------

// The headline case: a project whose history is unrelated to the server's
// starter commit publishes for the first time through `basil publish` (NOT raw
// git). publish detects the unrelated history, makes the one forced push, and
// the release branch moves onto the local project's commit. A SECOND publish is
// then an ordinary fast-forward that needs no force.
func TestPublish_FirstPublishFromGraduatedProject(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	local, head := f.graduatedClone(t)

	var out bytes.Buffer
	if err := runPublish([]string{local, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("first publish from a graduated project failed: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "First publish") {
		t.Errorf("output did not announce the first publish:\n%s", got)
	}
	if !strings.Contains(got, "force-replace") && !strings.Contains(got, "unrelated") {
		t.Errorf("output did not explain the forced first-publish replacement:\n%s", got)
	}
	if !strings.Contains(got, "remote: Deployed the release") {
		t.Errorf("the server's remote: line was not streamed:\n%s", got)
	}
	if tip := f.originTip(t); tip != head {
		t.Fatalf("live is at %s, want the graduated head %s", tip, head)
	}

	// The second publish is an ordinary fast-forward: no force, no first-publish
	// language, and the ref advances normally.
	writePubFile(t, filepath.Join(local, "site", "index.pars"), "<h1>\"v2\"</h1>\n")
	testGit(t, local, "add", "-A")
	testGit(t, local, "commit", "--quiet", "--no-verify", "-m", "v2 page")
	second := testGit(t, local, "rev-parse", "HEAD")

	var out2 bytes.Buffer
	if err := runPublish([]string{local, "--yes"}, strings.NewReader(""), &out2, &out2, emptyEnv); err != nil {
		t.Fatalf("second publish failed: %v\n%s", err, out2.String())
	}
	if strings.Contains(out2.String(), "First publish") {
		t.Errorf("the second publish must not be treated as a first publish:\n%s", out2.String())
	}
	if tip := f.originTip(t); tip != second {
		t.Errorf("second publish left live at %s, want %s", tip, second)
	}
}

// --dry-run on the first-publish state previews the replacement and pushes
// nothing: the release branch stays at the server's starter commit.
func TestPublish_FirstPublishDryRunPushesNothing(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	local, _ := f.graduatedClone(t)

	var out bytes.Buffer
	if err := runPublish([]string{local, "--dry-run"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("first-publish --dry-run failed: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "First publish") {
		t.Errorf("--dry-run did not identify the first-publish state:\n%s", got)
	}
	if !strings.Contains(got, "--dry-run: nothing was pushed") {
		t.Errorf("--dry-run did not announce it pushed nothing:\n%s", got)
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("--dry-run moved live to %s; the starter commit %s must stay", tip, f.seed)
	}
}

// The forced first publish is not skippable without --yes: a declined prompt
// leaves the starter commit in place.
func TestPublish_FirstPublishPromptAborts(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	local, _ := f.graduatedClone(t)

	var out bytes.Buffer
	if err := runPublish([]string{local}, strings.NewReader("n\n"), &out, &out, emptyEnv); err != nil {
		t.Fatalf("an aborted first publish is not an error: %v", err)
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("the declined first publish was not reported:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("a declined first publish moved live to %s, want %s", tip, f.seed)
	}
}

// The guard against over-forcing: an ordinary divergence - a clone that shares
// history with origin but has fallen behind - is NOT a first publish and must
// never be offered a force. The classifier must return false, and publish must
// keep today's behaviour (no forced push, the ref unmoved).
func TestPublish_OrdinaryDivergenceIsNotOfferedAForce(t *testing.T) {
	f := newPublishFixture(t, "live")

	// Advance origin beyond what the clone has: another clone commits and
	// pushes, so origin's release tip is ahead of the clone's HEAD but shares
	// the seed as a common ancestor.
	other := filepath.Join(t.TempDir(), "other")
	testGit(t, filepath.Dir(other), "clone", "--quiet", f.bare, other)
	testGit(t, other, "config", "user.name", "Other Author")
	testGit(t, other, "config", "user.email", "other@example.com")
	writePubFile(t, filepath.Join(other, "site", "index.pars"), "<h1>\"ahead\"</h1>\n")
	testGit(t, other, "add", "-A")
	testGit(t, other, "commit", "--quiet", "--no-verify", "-m", "ahead")
	ahead := testGit(t, other, "rev-parse", "HEAD")
	testGit(t, other, "push", "--quiet", "origin", "live")

	// The classifier: origin tip (ahead) and the clone's HEAD (seed) share the
	// seed, so this is NOT unrelated history.
	if unrelatedHistories(f.work, "live", ahead, f.seed) {
		t.Fatalf("ordinary divergence was misclassified as unrelated history")
	}

	// And publish must not force it: the clone is behind, so there is nothing to
	// publish and the ref is untouched.
	var out bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "First publish") {
		t.Errorf("ordinary divergence was offered a first-publish force:\n%s", out.String())
	}
	if tip := f.originTip(t); tip != ahead {
		t.Errorf("publish moved live to %s; an ordinary divergence must leave it at %s", tip, ahead)
	}
}

// A local commit diverging from a shared branch (not merely behind) is likewise
// not unrelated history: the classifier must see the shared ancestor and refuse
// the force, so the server's ordinary non-fast-forward refusal stands.
func TestPublish_DivergentLocalCommitIsNotUnrelated(t *testing.T) {
	f := newPublishFixture(t, "live")

	// Origin advances.
	other := filepath.Join(t.TempDir(), "other")
	testGit(t, filepath.Dir(other), "clone", "--quiet", f.bare, other)
	testGit(t, other, "config", "user.name", "Other Author")
	testGit(t, other, "config", "user.email", "other@example.com")
	writePubFile(t, filepath.Join(other, "site", "index.pars"), "<h1>\"theirs\"</h1>\n")
	testGit(t, other, "add", "-A")
	testGit(t, other, "commit", "--quiet", "--no-verify", "-m", "theirs")
	testGit(t, other, "push", "--quiet", "origin", "live")
	aheadSHA := testGit(t, other, "rev-parse", "HEAD")

	// The clone commits its own divergent change on top of the seed.
	mine := f.commitLocal(t, "site/index.pars", "<h1>\"mine\"</h1>\n", "mine")

	if unrelatedHistories(f.work, "live", aheadSHA, mine) {
		t.Fatalf("a divergent-but-shared history was misclassified as unrelated")
	}
}

// The classifier is also correct in the affirmative: two genuinely unrelated
// roots have no merge base, so unrelatedHistories reports true.
func TestUnrelatedHistories_ClassifiesUnrelatedRootsAsTrue(t *testing.T) {
	f := newPublishFixture(t, "live")
	local, head := f.graduatedClone(t)
	if !unrelatedHistories(local, "live", f.seed, head) {
		t.Errorf("two unrelated roots were not classified as unrelated (seed=%s head=%s)", f.seed, head)
	}
}

// --- security: a hostile server-advertised branch name (BUG-037) ------------

// The branch publish acts on comes from origin's advertised HEAD. git reads a
// leading-dash positional as an OPTION, so a plain-HTTP or MITM'd server that
// advertises `ref: refs/heads/--upload-pack=<cmd>` could smuggle that option
// into publish's `git fetch` and execute <cmd> locally. Real git refuses to
// store such a ref, so the injection can only arrive over the wire - which is
// exactly why the validator, not git's own ref checks, is the guard. It must
// reject every option-injection and malformed shape, and accept ordinary names.
func TestValidateReleaseBranch_RejectsInjectionAndMalformed(t *testing.T) {
	bad := []string{
		"",
		"-upload-pack=touch /tmp/pwned",
		"--upload-pack=touch /tmp/pwned",
		"-o",
		"live/../../etc",
		"a..b",
		"/live",
		"live/",
		"live//prod",
		"live.lock",
		"trailingdot.",
		"has space",
		"tilde~1",
		"caret^",
		"colon:ref",
		"star*",
		"question?",
		"bracket[1]",
		"back\\slash",
		"tab\tname",
		"ctrl\x01char",
		"branch@{0}",
	}
	for _, name := range bad {
		if err := validateReleaseBranch(name); err == nil {
			t.Errorf("validateReleaseBranch(%q) = nil; want a refusal", name)
		}
	}

	good := []string{"live", "main", "prod", "release-2.0", "feature/x", "v1.2.3"}
	for _, name := range good {
		if err := validateReleaseBranch(name); err != nil {
			t.Errorf("validateReleaseBranch(%q) = %v; want nil", name, err)
		}
	}
}

// Defence in depth: even if a hostile branch name reached the classify fetch
// (it cannot - validateReleaseBranch stops it upstream), `--end-of-options`
// pins it as a positional so git can never read it as an option. Drive
// unrelatedHistories with an injection name and prove the side-effect file the
// option would have executed is never created.
func TestUnrelatedHistories_FetchDoesNotExecuteInjectedOption(t *testing.T) {
	f := newPublishFixture(t, "live")
	marker := filepath.Join(t.TempDir(), "PWNED")
	evil := "--upload-pack=touch " + marker

	// serverTip is irrelevant: the fetch runs first, and it is the thing under
	// test. A missing local tip just makes the classifier return false.
	_ = unrelatedHistories(f.work, evil, f.seed, f.seed)

	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("the injected --upload-pack option executed: %s was created", marker)
	}
}

// --- a non-default release branch (HEAD: main) round trip -------------------

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
		t.Errorf("publish did not target the branch origin's HEAD names (main):\n%s", out.String())
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

// --- the release branch comes from the server (FEAT-157) --------------------

// An operator retargeting site.git's HEAD is the whole interface for changing
// the release branch: the next publish from an UNCHANGED clone must follow it.
func TestPublish_FollowsHEADRetargetWithNoClientChange(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)

	// The server now releases from "shipping", which already exists there.
	testGit(t, f.bare, "branch", "shipping", f.seed)
	testGit(t, f.bare, "symbolic-ref", "HEAD", "refs/heads/shipping")

	head := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("publish after a HEAD retarget failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `to "shipping"`) {
		t.Errorf("publish did not follow origin's retargeted HEAD:\n%s", out.String())
	}
	if tip := testGit(t, f.bare, "rev-parse", "shipping"); tip != head {
		t.Errorf("shipping is at %s, want %s", tip, head)
	}
	if tip := testGit(t, f.bare, "rev-parse", "live"); tip != f.seed {
		t.Errorf("live moved to %s; only the branch HEAD names should have been published to", tip)
	}
}

// The "no client change" promise has a second half: a clone that already
// published once carries remote.origin.push pointing at the branch that
// released THEN. After a retarget, publish itself still follows the server —
// but a bare `git push` from that clone would keep pushing the old branch,
// which the hub stores and never deploys. Nothing fails, nothing is said, and
// the developer believes they have published. So publish re-points the
// refspec it wrote, and says so.
func TestPublish_RePointsItsOwnRefspecAfterAHEADRetarget(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)

	// First publish: sets the refspec at the branch releasing today.
	v2 := f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	var first bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &first, &first, emptyEnv); err != nil {
		t.Fatalf("first publish failed: %v\n%s", err, first.String())
	}
	if got := testGit(t, f.work, "config", "--get", "remote.origin.push"); got != "HEAD:refs/heads/live" {
		t.Fatalf("precondition failed: remote.origin.push = %q", got)
	}
	if strings.Contains(first.String(), "Re-pointed") {
		t.Errorf("the first publish had nothing to re-point:\n%s", first.String())
	}

	// The operator retargets HEAD.
	testGit(t, f.bare, "branch", "shipping", f.seed)
	testGit(t, f.bare, "symbolic-ref", "HEAD", "refs/heads/shipping")

	f.commitLocal(t, "site/index.pars", "<h1>\"v3\"</h1>\n", "v3")
	var second bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &second, &second, emptyEnv); err != nil {
		t.Fatalf("publish after the retarget failed: %v\n%s", err, second.String())
	}
	if got := testGit(t, f.work, "config", "--get", "remote.origin.push"); got != "HEAD:refs/heads/shipping" {
		t.Errorf("remote.origin.push = %q, want HEAD:refs/heads/shipping", got)
	}
	// Rewriting a developer's git config silently is worse than leaving it
	// stale: the change has to be visible in the output.
	if !strings.Contains(second.String(), "Re-pointed") {
		t.Errorf("publish re-pointed the refspec without saying so:\n%s", second.String())
	}

	// The point of the setting: a bare `git push` now reaches the branch that
	// actually releases.
	bare := f.commitLocal(t, "site/index.pars", "<h1>\"v4\"</h1>\n", "v4")
	testGit(t, f.work, "push", "origin")
	if tip := testGit(t, f.bare, "rev-parse", "shipping"); tip != bare {
		t.Errorf("a bare git push left shipping at %s, want %s", tip, bare)
	}
	if tip := testGit(t, f.bare, "rev-parse", "live"); tip != v2 {
		t.Errorf("a bare git push moved live to %s; the branch that no longer releases should have stayed at %s", tip, v2)
	}
}

// A refspec publish did not write is the developer's own and must survive
// untouched, retarget or no retarget: repairing our own convenience setting
// is one thing, editing someone's git config is another.
func TestPublish_LeavesAForeignRefspecAlone(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)

	const custom = "+refs/heads/main:refs/heads/mirror"
	testGit(t, f.work, "config", "remote.origin.push", custom)

	testGit(t, f.bare, "branch", "shipping", f.seed)
	testGit(t, f.bare, "symbolic-ref", "HEAD", "refs/heads/shipping")

	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	var out bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}
	if got := testGit(t, f.work, "config", "--get", "remote.origin.push"); got != custom {
		t.Errorf("publish rewrote a refspec it did not write: %q, want %q", got, custom)
	}
	if strings.Contains(out.String(), "Re-pointed") {
		t.Errorf("publish claimed to re-point a refspec it left alone:\n%s", out.String())
	}
}

// A clone's committed deploy.branch is dead weight and must not steer the
// publish - the exact confusion FEAT-157 removed.
func TestPublish_IgnoresAndReportsRetiredBranchKey(t *testing.T) {
	f := newPublishFixture(t, "live")
	f.installAcceptHook(t)
	writePubFile(t, filepath.Join(f.work, "basil.yaml"),
		"server:\n  host: localhost\ndeploy:\n  branch: shipping\n")
	testGit(t, f.work, "add", "-A")
	testGit(t, f.work, "commit", "--quiet", "--no-verify", "-m", "stale key")
	head := testGit(t, f.work, "rev-parse", "HEAD")

	var out bytes.Buffer
	if err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out.String())
	}
	got := out.String()
	if !strings.Contains(got, `to "live"`) {
		t.Errorf("the committed deploy.branch steered the publish:\n%s", got)
	}
	if !strings.Contains(got, "deploy.branch is no longer read") {
		t.Errorf("publish did not report the retired key in the clone:\n%s", got)
	}
	if tip := f.originTip(t); tip != head {
		t.Errorf("live is at %s, want %s", tip, head)
	}
}

// A detached HEAD on the server names no release branch: publish stops and
// names the one command that fixes it, rather than guessing a branch.
func TestPublish_DetachedOriginHEADNamesTheFix(t *testing.T) {
	f := newPublishFixture(t, "live")
	testGit(t, f.bare, "update-ref", "--no-deref", "HEAD", f.seed)
	f.commitLocal(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	var out bytes.Buffer
	err := runPublish([]string{f.work, "--yes"}, strings.NewReader(""), &out, &out, emptyEnv)
	if err == nil {
		t.Fatalf("publish must not proceed when origin names no release branch:\n%s", out.String())
	}
	if !strings.Contains(err.Error(), "symbolic-ref HEAD refs/heads/") {
		t.Errorf("error does not name the fix: %v", err)
	}
	if !strings.Contains(err.Error(), "detached") {
		t.Errorf("error does not name the state: %v", err)
	}
	if tip := f.originTip(t); tip != f.seed {
		t.Errorf("live moved to %s; nothing should have been pushed", tip)
	}
}
