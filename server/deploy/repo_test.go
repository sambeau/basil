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

	if !GitEnabled(repo) {
		t.Error("absent basil.gitEnabled must mean enabled")
	}
	run(t, repo, "config", "basil.gitEnabled", "true")
	if !GitEnabled(repo) {
		t.Error("basil.gitEnabled true must mean enabled")
	}
	run(t, repo, "config", "basil.gitEnabled", "false")
	if GitEnabled(repo) {
		t.Error("basil.gitEnabled false must mean disabled")
	}
	run(t, repo, "config", "basil.gitEnabled", "off")
	if GitEnabled(repo) {
		t.Error("git's other spellings of false must mean disabled")
	}
	run(t, repo, "config", "basil.gitEnabled", "flase")
	if !GitEnabled(repo) {
		t.Error("an unreadable value must not switch the deploy path off")
	}
	if !GitEnabled(filepath.Join(t.TempDir(), "nothing-here")) {
		t.Error("a missing repository must not report the switch off")
	}
}
