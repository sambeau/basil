package deploy

// Operator-owned facts recorded in the bare repository (FEAT-157).
//
// basil.yaml ships inside the release, so anything it says about the server
// can be rewritten by a deploy. Two facts must not be: which branch
// publishes, and whether /.git is served at all. Both live in <site
// root>/site.git, where only a shell on the box can change them —
// `HEAD` for the release branch (git's own record of it, and already what a
// fresh clone checks out) and the `basil.gitEnabled` git-config key for the
// endpoint switch, where git has no native equivalent.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReleaseBranch returns the branch whose movement publishes a release: the
// branch site.git's HEAD names, with refs/heads/ stripped. It is the single
// source of truth — there is no config fallback, because a fallback is
// exactly the thing a release could steer.
//
// A HEAD that names no branch (detached, or unreadable because the
// repository is missing or broken) is an error naming the one command that
// fixes it. The caller decides what that means: a push is refused, a display
// hint falls back to the default name.
func ReleaseBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "HEAD")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = err.Error()
		}
		return "", headError(repoDir, reason)
	}
	ref := strings.TrimSpace(string(out))
	if !strings.HasPrefix(ref, "refs/heads/") {
		return "", headError(repoDir, fmt.Sprintf("HEAD names %q, which is not a branch", ref))
	}
	branch := strings.TrimPrefix(ref, "refs/heads/")
	if branch == "" {
		return "", headError(repoDir, "HEAD names an empty branch")
	}
	return branch, nil
}

// ReleaseRef is ReleaseBranch as the fully-qualified ref the hooks compare
// against, since the hook protocol speaks in refs.
func ReleaseRef(repoDir string) (string, error) {
	branch, err := ReleaseBranch(repoDir)
	if err != nil {
		return "", err
	}
	return "refs/heads/" + branch, nil
}

func headError(repoDir, reason string) error {
	return fmt.Errorf("cannot read the release branch from %s: %s — set it with: git -C %s symbolic-ref HEAD refs/heads/<branch>",
		filepath.Join(repoDir, "HEAD"), reason, repoDir)
}

// GitEnabled reports whether the operator has left the Git endpoint on.
// `git config basil.gitEnabled false` in the bare repository switches it off
// — clone and push both — for operators who deploy at the shell and want no
// /.git served at all.
//
// Absent is enabled: the endpoint is on whenever the repository exists, and
// nobody should ever have to write the key to get today's behaviour. Anything
// that is not a readable false is also enabled — a switch that turns itself
// off on a typo would take down the deploy path it exists to control.
func GitEnabled(repoDir string) bool {
	// --local: the switch is a fact about THIS repository. Without it a
	// basil.gitEnabled in the operator's ~/.gitconfig would decide it for
	// every site the account serves.
	cmd := exec.Command("git", "config", "--local", "--bool", "--get", "basil.gitEnabled")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return true // unset, or unreadable
	}
	return strings.TrimSpace(string(out)) != "false"
}
