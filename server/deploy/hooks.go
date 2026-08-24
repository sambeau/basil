package deploy

// Receive-hook templates and installation (FEAT-154). The hooks are two-line
// sh scripts — deploy logic lives in `basil deploy --from-hook`, never in
// shell — and Basil installs them itself: an operator who has to install a
// hook by hand is an operator whose repository one day silently stops
// deploying.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookMarker identifies a hook Basil generated. InstallHooks rewrites a file
// carrying this line freely (path drift, template changes) and refuses to
// touch one without it: an operator's hand-written hook must never be
// silently destroyed.
const hookMarker = "# Basil-managed hook."

// hookNames are the receive hooks Basil installs: pre-receive validates
// before any ref moves, post-receive activates after the ref has moved.
var hookNames = [...]string{"pre-receive", "post-receive"}

// ErrForeignHook reports an existing hook that Basil did not generate.
// Callers can errors.Is for it to tell "an operator has their own hook here"
// apart from ordinary filesystem trouble.
var ErrForeignHook = errors.New("existing hook was not installed by Basil")

// hookScript renders one hook. The basil binary path is baked in absolute at
// install time: Git executes hooks with its own environment, and the
// operator's PATH — the one that found `basil` at install time — is not part
// of it.
func hookScript(basilPath, hookName string) string {
	return fmt.Sprintf(`#!/bin/sh
%s Do not edit: Basil rewrites it (basil --init, server startup).
# The binary path is absolute because hooks run with Git's PATH, not yours.
exec %s deploy --from-hook=%s
`, hookMarker, shellQuote(basilPath), hookName)
}

// shellQuote makes a path safe to splice into the sh script: single-quoted,
// with each embedded single quote escaped the POSIX way (close the quote,
// backslash-escape the quote, reopen).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// InstallHooks writes Basil's pre-receive and post-receive hooks into
// bareRepoDir/hooks, baking in the path of the running binary. It is
// idempotent, and rewriting on any content difference is what implements
// both "re-install if missing" and healing binary-path drift (the basil
// binary moved or was upgraded in place elsewhere). A hook that exists but
// was not Basil-generated is refused with ErrForeignHook.
func InstallHooks(bareRepoDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating the basil binary for the receive hooks: %w", err)
	}
	return installHooks(bareRepoDir, exe)
}

// InstallHooksAt is InstallHooks with an explicit basil binary path, for
// callers where os.Executable() is not the binary the hooks should run —
// notably tests, whose executable is the test binary.
func InstallHooksAt(bareRepoDir, basilPath string) error {
	return installHooks(bareRepoDir, basilPath)
}

// installHooks is InstallHooks with the binary path injectable for tests.
func installHooks(bareRepoDir, basilPath string) error {
	hooksDir := filepath.Join(bareRepoDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", hooksDir, err)
	}
	for _, name := range hookNames {
		path := filepath.Join(hooksDir, name)
		want := hookScript(basilPath, name)

		existing, err := os.ReadFile(path)
		switch {
		case err == nil:
			if string(existing) == want {
				// Already correct; just guarantee it is executable (a
				// chmod-ed hook is a hook that silently never runs).
				if err := os.Chmod(path, 0o755); err != nil {
					return fmt.Errorf("making %s executable: %w", path, err)
				}
				continue
			}
			if !strings.Contains(string(existing), hookMarker) {
				return fmt.Errorf("%s: %w — move it aside if Basil should manage this hook", path, ErrForeignHook)
			}
		case !os.IsNotExist(err):
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if err := os.WriteFile(path, []byte(want), 0o755); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		// WriteFile's mode only applies to a file it creates; a rewritten
		// hook keeps its old mode unless told otherwise.
		if err := os.Chmod(path, 0o755); err != nil {
			return fmt.Errorf("making %s executable: %w", path, err)
		}
	}
	return nil
}
