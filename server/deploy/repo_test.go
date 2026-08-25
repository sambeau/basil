package deploy

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newBareRepo(t *testing.T, branch string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := filepath.Join(t.TempDir(), "site.git")
	run(t, "", "init", "--bare", "--initial-branch="+branch, dir)
	return dir
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestReleaseBranch(t *testing.T) {
	repo := newBareRepo(t, "live")

	branch, err := ReleaseBranch(repo)
	if err != nil {
		t.Fatalf("ReleaseBranch: %v", err)
	}
	if branch != "live" {
		t.Errorf("ReleaseBranch = %q, want live", branch)
	}
	if ref, err := ReleaseRef(repo); err != nil || ref != "refs/heads/live" {
		t.Errorf("ReleaseRef = %q, %v; want refs/heads/live", ref, err)
	}

	// Retargeting HEAD is the whole interface, including at a branch that
	// does not exist yet: the answer is what HEAD names, not what exists.
	run(t, repo, "symbolic-ref", "HEAD", "refs/heads/shipping")
	if branch, err := ReleaseBranch(repo); err != nil || branch != "shipping" {
		t.Errorf("after retargeting, ReleaseBranch = %q, %v; want shipping", branch, err)
	}

	// A branch name with slashes survives the refs/heads/ strip intact.
	run(t, repo, "symbolic-ref", "HEAD", "refs/heads/release/v2")
	if branch, err := ReleaseBranch(repo); err != nil || branch != "release/v2" {
		t.Errorf("ReleaseBranch = %q, %v; want release/v2", branch, err)
	}
}

// Detached and unreadable are one case to the caller: HEAD names no release
// branch. Both must name the command that fixes it.
func TestReleaseBranchNamesTheFix(t *testing.T) {
	repo := newBareRepo(t, "live")
	work := t.TempDir()
	run(t, "", "init", "--initial-branch=live", work)
	run(t, work, "config", "user.email", "t@example.com")
	run(t, work, "config", "user.name", "t")
	run(t, work, "commit", "--quiet", "--allow-empty", "-m", "seed")
	run(t, work, "push", "--quiet", repo, "live:live")
	sha := run(t, work, "rev-parse", "HEAD")

	run(t, repo, "update-ref", "--no-deref", "HEAD", sha) // detach
	_, err := ReleaseBranch(repo)
	if err == nil {
		t.Fatal("a detached HEAD must not yield a release branch")
	}
	if !strings.Contains(err.Error(), "symbolic-ref HEAD refs/heads/<branch>") {
		t.Errorf("error does not name the fix: %v", err)
	}

	if _, err := ReleaseBranch(filepath.Join(t.TempDir(), "nothing-here")); err == nil {
		t.Error("a missing repository must not yield a release branch")
	}
}

// The off-switch: absent and true serve, false does not, and anything
// unreadable fails towards serving rather than silently shutting the deploy
// path.
func TestGitEnabled(t *testing.T) {
	repo := newBareRepo(t, "live")

	if on, err := GitEnabled(repo); !on || err != nil {
		t.Errorf("absent basil.gitEnabled must mean enabled, got %v, %v", on, err)
	}
	run(t, repo, "config", "basil.gitEnabled", "true")
	if on, err := GitEnabled(repo); !on || err != nil {
		t.Errorf("basil.gitEnabled true must mean enabled, got %v, %v", on, err)
	}
	run(t, repo, "config", "basil.gitEnabled", "false")
	if on, err := GitEnabled(repo); on || err != nil {
		t.Errorf("basil.gitEnabled false must mean disabled, got %v, %v", on, err)
	}
	run(t, repo, "config", "basil.gitEnabled", "off")
	if on, err := GitEnabled(repo); on || err != nil {
		t.Errorf("git's other spellings of false must mean disabled, got %v, %v", on, err)
	}
	run(t, repo, "config", "basil.gitEnabled", "flase")
	if on, _ := GitEnabled(repo); !on {
		t.Error("an unreadable value must not switch the deploy path off")
	}
	if on, err := GitEnabled(filepath.Join(t.TempDir(), "nothing-here")); !on || err != nil {
		t.Errorf("a missing repository must not report the switch off, got %v, %v", on, err)
	}
}

// "I turned it off and it stayed on" must never be silent. Unset and
// unreadable both fail towards serving, and only the second is a surprise —
// so only the second returns something for the caller to log.
func TestGitEnabledDistinguishesUnsetFromUnreadable(t *testing.T) {
	repo := newBareRepo(t, "live")

	if _, err := GitEnabled(repo); err != nil {
		t.Errorf("an unset key is the normal case and must be silent, got: %v", err)
	}

	run(t, repo, "config", "basil.gitEnabled", "flase")
	on, err := GitEnabled(repo)
	if !on {
		t.Fatal("a typo'd boolean must leave the endpoint on")
	}
	if err == nil {
		t.Fatal("a typo'd boolean must be reported, not swallowed")
	}
	// The warning has to be actionable: which repository, and what git said.
	if !strings.Contains(err.Error(), repo) {
		t.Errorf("the warning does not name the repository: %v", err)
	}
	if !strings.Contains(err.Error(), "basil.gitEnabled") {
		t.Errorf("the warning does not name the key: %v", err)
	}
	if !strings.Contains(err.Error(), "bad boolean") {
		t.Errorf("the warning does not relay git's reason: %v", err)
	}
}

// GIT_DIR in the inherited environment OVERRIDES cmd.Dir, so a server started
// with one set could have another repository answer for this one — which
// branch publishes, and whether the endpoint is served. Both readers name the
// repository with an explicit path, which beats the environment.
func TestRepoReadersIgnoreGitDirInTheEnvironment(t *testing.T) {
	ours := newBareRepo(t, "live")
	theirs := newBareRepo(t, "attacker")
	run(t, theirs, "config", "basil.gitEnabled", "false")

	t.Setenv("GIT_DIR", theirs)

	// Sanity: the environment really would win if the path were not pinned.
	if got := run(t, ours, "symbolic-ref", "HEAD"); got != "refs/heads/attacker" {
		t.Fatalf("precondition failed: GIT_DIR did not override the working directory (got %q)", got)
	}

	if branch, err := ReleaseBranch(ours); err != nil || branch != "live" {
		t.Errorf("ReleaseBranch = %q, %v; want live from the named repository", branch, err)
	}
	if on, err := GitEnabled(ours); !on || err != nil {
		t.Errorf("GitEnabled = %v, %v; want the named repository's absent key (enabled)", on, err)
	}
}
