package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/deploy"
)

// mapEnv is getenv for tests: only what the map holds exists.
func mapEnv(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

// runHookLines drives runFromHook the way Git would: one "<old> <new> <ref>"
// line per updated ref on stdin.
func runHookLines(f *deployFixture, which string, getenv func(string) string, lines ...string) (stdout, stderr bytes.Buffer, err error) {
	stdin := strings.NewReader(strings.Join(lines, "\n") + "\n")
	err = runFromHook(which, f.root, "", stdin, &stdout, &stderr, getenv)
	return stdout, stderr, err
}

func refLine(old, new, ref string) string {
	return fmt.Sprintf("%s %s %s", old, new, ref)
}

// A push that moves anything but the release ref is store-and-stop: both
// hooks exit 0 without touching the engine.
func TestFromHook_BranchPushIsNoOp(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"feature\"</h1>\n", "feature work")

	for _, which := range []string{"pre-receive", "post-receive"} {
		stdout, _, err := runHookLines(f, which, emptyEnv, refLine(zeroSHA, sha, "refs/heads/feature"))
		if err != nil {
			t.Fatalf("%s on a feature branch: %v", which, err)
		}
		if stdout.Len() != 0 {
			t.Errorf("%s on a feature branch produced output: %q", which, stdout.String())
		}
	}
	if got := f.currentSHA(t); got != before {
		t.Errorf("a feature-branch push changed the live release: %s -> %s", before, got)
	}
	if entries := f.recordEntries(t); len(entries) != 1 { // release 1 only
		t.Errorf("record grew to %d entries from a feature-branch push", len(entries))
	}
}

// pre-receive then post-receive on a good release: checked, then deployed,
// with the publisher taken from BASIL_PUBLISHER (the transport exports it).
func TestFromHook_ReleasePushDeploysAndRecordsPublisher(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	env := mapEnv(map[string]string{"BASIL_PUBLISHER": "alice"})
	line := refLine(before, sha, "refs/heads/live")

	stdout, _, err := runHookLines(f, "pre-receive", env, line)
	if err != nil {
		t.Fatalf("pre-receive: %v\noutput: %s", err, stdout.String())
	}
	if out := stdout.String(); !strings.Contains(out, "Checking release "+shortRelease(sha)) || !strings.Contains(out, "ok") {
		t.Errorf("pre-receive output = %q, want a Checking … ok line", out)
	}
	// Prepared, not yet live, and nothing recorded for the preparation.
	if got := f.currentSHA(t); got != before {
		t.Errorf("pre-receive activated the release: current = %s", got)
	}
	if entries := f.recordEntries(t); len(entries) != 1 {
		t.Fatalf("pre-receive success recorded %d entries, want 1 (release 1 only)", len(entries))
	}

	stdout, _, err = runHookLines(f, "post-receive", env, line)
	if err != nil {
		t.Fatalf("post-receive: %v\noutput: %s", err, stdout.String())
	}
	if out := stdout.String(); !strings.Contains(out, "Deployed "+shortRelease(sha)) {
		t.Errorf("post-receive output = %q, want a Deployed line", out)
	}
	if got := f.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want %s", got, sha)
	}
	newest := f.recordEntries(t)[0]
	if newest.CommitSHA != sha || newest.Outcome != deploy.OutcomeDeployed {
		t.Errorf("newest entry = %+v, want deployed %s", newest, sha)
	}
	if newest.Trigger != deploy.TriggerPush || newest.Publisher != "alice" {
		t.Errorf("identity: trigger=%q publisher=%q, want push/alice", newest.Trigger, newest.Publisher)
	}
}

// Without BASIL_PUBLISHER the publisher is honestly generic.
func TestFromHook_PublisherDefaultsToPush(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	if _, _, err := runHookLines(f, "post-receive", emptyEnv, refLine(before, sha, "refs/heads/live")); err != nil {
		t.Fatalf("post-receive: %v", err)
	}
	if newest := f.recordEntries(t)[0]; newest.Publisher != "push" {
		t.Errorf("publisher = %q, want \"push\"", newest.Publisher)
	}
}

// A broken release is refused before the ref moves: file:line reaches the
// developer, the live site is untouched, and the rejection is recorded.
func TestFromHook_BrokenReleaseRefused(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/broken.pars", "let x = = 2\n", "broken parsley")

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(before, sha, "refs/heads/live"))
	if err == nil {
		t.Fatal("pre-receive accepted a broken release")
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "broken.pars:1") {
		t.Errorf("output lacks file:line for the parse error:\n%s", out)
	}
	if want := fmt.Sprintf("Release rejected. The live site is unchanged (still %s).", shortRelease(before)); !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
	// The prefix Git adds must not be duplicated by us.
	if strings.Contains(out, "remote:") {
		t.Errorf("output hand-writes the remote: prefix:\n%s", out)
	}
	if got := f.currentSHA(t); got != before {
		t.Errorf("a refused push changed the live release: %s -> %s", before, got)
	}
	if _, statErr := os.Stat(filepath.Join(f.root, "releases", sha)); !os.IsNotExist(statErr) {
		t.Errorf("the rejected release directory was left behind (err=%v)", statErr)
	}
	newest := f.recordEntries(t)[0]
	if newest.CommitSHA != sha || newest.Outcome != deploy.OutcomeRejected {
		t.Errorf("newest entry = %+v, want rejected %s", newest, sha)
	}
}

// A valid but unformatted release is ACCEPTED: pre-receive succeeds (exit 0,
// nothing refused), and a non-fatal warning naming the unformatted file and
// the fix `basil fmt -w` is relayed to the developer. The server never
// rewrites code, only reports.
func TestFromHook_UnformattedReleaseWarnsButSucceeds(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	// Valid Parsley, but not in canonical form.
	sha := f.commitAndPush(t, "site/index.pars", "let    x    =    5\n", "unformatted but valid")

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(before, sha, "refs/heads/live"))
	if err != nil {
		t.Fatalf("pre-receive refused a valid (if unformatted) release: %v\noutput: %s", err, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "ok") {
		t.Errorf("pre-receive did not accept the release:\n%s", out)
	}
	if !strings.Contains(out, "warning") || !strings.Contains(out, "index.pars") {
		t.Errorf("output lacks a warning naming the unformatted file:\n%s", out)
	}
	if !strings.Contains(out, "basil fmt -w") {
		t.Errorf("warning does not name the fix 'basil fmt -w':\n%s", out)
	}
	// The prefix Git adds must not be duplicated by us.
	if strings.Contains(out, "remote:") {
		t.Errorf("output hand-writes the remote: prefix:\n%s", out)
	}
	// Warning is not a rejection: the release was prepared, not refused, and
	// its directory survives for post-receive to activate.
	if _, statErr := os.Stat(filepath.Join(f.root, "releases", sha)); statErr != nil {
		t.Errorf("prepared release directory missing after an accepted push: %v", statErr)
	}
}

// A well-formatted release warns about nothing: the accept line stands alone.
func TestFromHook_FormattedReleaseWarnsNothing(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "let x = 5\n", "already formatted")

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(before, sha, "refs/heads/live"))
	if err != nil {
		t.Fatalf("pre-receive: %v\noutput: %s", err, stdout.String())
	}
	if out := stdout.String(); strings.Contains(out, "warning") || strings.Contains(out, "basil fmt -w") {
		t.Errorf("a formatted release produced a formatting warning:\n%s", out)
	}
}

func TestFromHook_ReleaseBranchDeletionRefused(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(before, zeroSHA, "refs/heads/live"))
	if err == nil {
		t.Fatal("pre-receive accepted deleting the release branch")
	}
	if out := stdout.String(); !strings.Contains(out, "the release branch cannot be deleted") {
		t.Errorf("output = %q, want the deletion refusal", out)
	}
}

// A real non-fast-forward: two commits that share history but where the old
// tip is not an ancestor of the proposed new one. Refused once the site has
// deployed anything of its own — the graduation exception below covers only a
// hub still sitting on its init starter release.
func TestFromHook_ForcePushRefused(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	onLive := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	deployRelease(t, f, before, onLive)

	// Step back one commit and commit something else, so `rewritten` and
	// `onLive` are siblings.
	rewritten := rewriteHistory(t, f)

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(onLive, rewritten, "refs/heads/live"))
	if err == nil {
		t.Fatal("pre-receive accepted a non-fast-forward on the release branch")
	}
	if out := stdout.String(); !strings.Contains(out, "force-pushing the release branch rewrites release history") {
		t.Errorf("output = %q, want the force-push refusal", out)
	}
	// init + the one real deploy; the refused force-push added nothing.
	if entries := f.recordEntries(t); len(entries) != 2 {
		t.Errorf("a refused force-push reached the engine: %d record entries, want 2", len(entries))
	}
}

// deployRelease drives post-receive so the fixture has a real (non-init)
// deploy in its record — the state in which the graduation exception is over.
func deployRelease(t *testing.T, f *deployFixture, old, new string) {
	t.Helper()
	stdout, stderr, err := runHookLines(f, "post-receive", emptyEnv, refLine(old, new, "refs/heads/live"))
	if err != nil {
		t.Fatalf("deploying %s: %v\n%s%s", shortRelease(new), err, stdout.String(), stderr.String())
	}
}

// rewriteHistory produces a sibling of the current tip in the clone and
// stores it in the bare repo, the way a real force-push candidate would
// already exist when pre-receive judges it.
func rewriteHistory(t *testing.T, f *deployFixture) string {
	t.Helper()
	testGit(t, f.work, "reset", "--hard", "HEAD~1")
	if err := os.WriteFile(filepath.Join(f.work, "site", "index.pars"), []byte("<h1>\"rewritten\"</h1>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testGit(t, f.work, "add", "-A")
	testGit(t, f.work, "commit", "--quiet", "--no-verify", "-m", "rewritten")
	testGit(t, f.work, "push", "--quiet", "origin", "HEAD:refs/heads/rewrite")
	return testGit(t, f.work, "rev-parse", "HEAD")
}

// Graduation (FEAT-156): server init seeds the release branch with its own
// starter commit, so the first publish from a local site that grew up
// elsewhere is a non-fast-forward. While the record shows nothing but that
// init release, it is accepted — otherwise graduation needs shell access.
func TestFromHook_StarterSiteNonFastForwardAccepted(t *testing.T) {
	f := newDeployFixture(t)
	onLive := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	rewritten := rewriteHistory(t, f)

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(onLive, rewritten, "refs/heads/live"))
	if err != nil {
		t.Fatalf("pre-receive refused the first publish onto a starter site: %v\n%s", err, stdout.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "starter site created by 'basil --init'") {
		t.Errorf("output = %q, want the one-time starter-overwrite notice", out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("output = %q, want the release to have been checked", out)
	}
	if strings.Contains(out, "force-pushing the release branch") {
		t.Errorf("output = %q, want no force-push refusal", out)
	}
}

// The other side of the boundary: one real deploy and the exception is spent
// for good, even though the record still contains the init release.
func TestFromHook_StarterExceptionEndsAfterFirstRealDeploy(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	onLive := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	deployRelease(t, f, before, onLive)
	rewritten := rewriteHistory(t, f)

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(onLive, rewritten, "refs/heads/live"))
	if err == nil {
		t.Fatal("pre-receive accepted a non-fast-forward after a real deploy")
	}
	out := stdout.String()
	if !strings.Contains(out, "force-pushing the release branch rewrites release history") {
		t.Errorf("output = %q, want the shipped force-push refusal", out)
	}
	if strings.Contains(out, "starter site created by") {
		t.Errorf("output = %q, want no starter-overwrite notice", out)
	}
}

// A record that cannot be read is not evidence of a fresh hub: the shipped
// refusal stands, and the reason is named so an operator can tell the two
// refusals apart.
func TestFromHook_MissingRecordRefusesNonFastForward(t *testing.T) {
	f := newDeployFixture(t)
	onLive := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	rewritten := rewriteHistory(t, f)
	if err := os.Remove(filepath.Join(f.root, "data", "deploy.db")); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(onLive, rewritten, "refs/heads/live"))
	if err == nil {
		t.Fatal("pre-receive accepted a non-fast-forward with no deploy record")
	}
	out := stdout.String()
	if !strings.Contains(out, "cannot read the deploy record") {
		t.Errorf("output = %q, want the unreadable-record explanation", out)
	}
	if !strings.Contains(out, "force-pushing the release branch rewrites release history") {
		t.Errorf("output = %q, want the shipped force-push refusal", out)
	}
}

// Fast-forwards never consult the record and never mention the exception —
// in both states of the hub.
func TestFromHook_FastForwardUnaffectedByStarterException(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	first := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	// Fresh hub: record holds the init release only.
	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, refLine(before, first, "refs/heads/live"))
	if err != nil {
		t.Fatalf("pre-receive refused a fast-forward on a fresh hub: %v\n%s", err, stdout.String())
	}
	if out := stdout.String(); strings.Contains(out, "starter site created by") {
		t.Errorf("a fast-forward invoked the starter exception: %q", out)
	}

	// After a real deploy: still accepted, still silent about the exception.
	deployRelease(t, f, before, first)
	second := f.commitAndPush(t, "site/index.pars", "<h1>\"v3\"</h1>\n", "v3")
	stdout, _, err = runHookLines(f, "pre-receive", emptyEnv, refLine(first, second, "refs/heads/live"))
	if err != nil {
		t.Fatalf("pre-receive refused a fast-forward after a deploy: %v\n%s", err, stdout.String())
	}
	if out := stdout.String(); strings.Contains(out, "starter site created by") {
		t.Errorf("a fast-forward invoked the starter exception: %q", out)
	}
}

// One refused ref fails the whole pre-receive — Git's own semantics: the
// hook judges the push as a unit, and a non-zero exit rejects every ref in
// it. The acceptable ref is still checked and reported.
func TestFromHook_OneRefusalFailsTheWholePush(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	stdout, _, err := runHookLines(f, "pre-receive", emptyEnv,
		refLine(before, sha, "refs/heads/live"),
		refLine(before, zeroSHA, "refs/heads/live"), // deletion sneaking in alongside
	)
	if err == nil {
		t.Fatal("pre-receive accepted a push containing a refused update")
	}
	out := stdout.String()
	if !strings.Contains(out, "ok") || !strings.Contains(out, "cannot be deleted") {
		t.Errorf("output should show both the passing check and the refusal:\n%s", out)
	}
}

// deploy.branch accepts the long ref forms, so a tag can be the release ref.
func TestFromHook_TagReleaseRef(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"tagged\"</h1>\n", "tagged release")
	testGit(t, f.work, "tag", "production")
	testGit(t, f.work, "push", "--quiet", "origin", "production")

	// The config ships inside the release, so point the ACTIVE release's
	// config at the tag ref.
	configPath := filepath.Join(f.root, "current", "basil.yaml")
	appendConfig := "\ndeploy:\n  branch: refs/tags/production\n"
	existing, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(existing, appendConfig...), 0o644); err != nil {
		t.Fatal(err)
	}

	line := refLine(zeroSHA, sha, "refs/tags/production")
	if stdout, _, err := runHookLines(f, "pre-receive", emptyEnv, line); err != nil {
		t.Fatalf("pre-receive on the tag ref: %v\noutput: %s", err, stdout.String())
	}
	if _, _, err := runHookLines(f, "post-receive", emptyEnv, line); err != nil {
		t.Fatalf("post-receive on the tag ref: %v", err)
	}
	if got := f.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want the tagged commit %s", got, sha)
	}

	// And with a tag release ref, moving refs/heads/live publishes nothing.
	if _, _, err := runHookLines(f, "post-receive", emptyEnv, refLine(before, sha, "refs/heads/live")); err != nil {
		t.Fatalf("post-receive on refs/heads/live with a tag release ref: %v", err)
	}
}

// With no --site/--config, the site root is derived from the repository the
// hook runs in: site.git's parent is the site root.
func TestFromHook_ResolvesSiteRootFromGitDir(t *testing.T) {
	f := newDeployFixture(t)
	before := f.currentSHA(t)
	sha := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	env := mapEnv(map[string]string{"GIT_DIR": filepath.Join(f.root, "site.git")})

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader(refLine(before, sha, "refs/heads/live") + "\n")
	if err := runFromHook("post-receive", "", "", stdin, &stdout, &stderr, env); err != nil {
		t.Fatalf("runFromHook with GIT_DIR: %v", err)
	}
	if got := f.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want %s", got, sha)
	}
}

func TestFromHook_RefusesUnknownHookName(t *testing.T) {
	f := newDeployFixture(t)
	_, _, err := runHookLines(f, "update", emptyEnv)
	if err == nil {
		t.Fatal("an unknown hook name was accepted")
	}
	if code := exitCode(err); code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
}

func TestFromHook_MalformedStdinRefused(t *testing.T) {
	f := newDeployFixture(t)
	_, _, err := runHookLines(f, "pre-receive", emptyEnv, "this is not a ref update")
	if err == nil {
		t.Fatal("malformed stdin was accepted")
	}
	if !strings.Contains(err.Error(), "malformed ref update line") {
		t.Errorf("error = %v, want a malformed-line explanation", err)
	}
}
