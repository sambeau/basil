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
	"errors"
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
	// --git-dir, not cmd.Dir: a GIT_DIR in the inherited environment
	// OVERRIDES the working directory, so a server started with one set
	// would read its release branch out of some other repository entirely.
	// The explicit flag beats the environment variable, and it also stops
	// git walking up out of a half-deleted site.git into an enclosing repo.
	cmd := exec.Command("git", "--git-dir="+repoDir, "symbolic-ref", "HEAD")
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
//
// The second return separates the two ways of arriving at "enabled". An unset
// key is the normal case and reports nil: silence is correct. A key that is
// present but UNREADABLE — a typo'd boolean, a config git refuses to parse —
// still reports enabled, keeping the fail-open posture, but returns an error
// for the caller to log. An operator who wrote `false` and got a served
// endpoint must never have to guess why: "I turned it off and it stayed on"
// is the one outcome this switch may not produce silently.
func GitEnabled(repoDir string) (bool, error) {
	// --file, not --local with cmd.Dir: a GIT_DIR in the inherited
	// environment overrides the working directory, so cmd.Dir alone could
	// have another repository answer for this one. Naming the file also
	// keeps the switch a fact about THIS repository — a basil.gitEnabled in
	// the operator's ~/.gitconfig must not decide it for every site the
	// account serves.
	configFile := filepath.Join(repoDir, "config")
	cmd := exec.Command("git", "config", "--file", configFile, "--bool", "--get", "basil.gitEnabled")
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// Exit 1 is git's "no such key": unset, the normal case.
			return true, nil
		}
		reason := strings.TrimSpace(stderr.String())
		if reason == "" {
			reason = err.Error()
		}
		return true, fmt.Errorf("cannot read basil.gitEnabled from %s: %s — the git endpoint stays on; fix the value or remove the key (git config --file %s --unset basil.gitEnabled)",
			configFile, reason, configFile)
	}
	return strings.TrimSpace(string(out)) != "false", nil
}
