package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sambeau/basil/server/config"
)

// runPublishCommand handles `basil publish [dir]`: it pushes the current
// commit of a working clone to the site's release branch, showing what would
// be published and asking to confirm first. It is not --site-based: it runs
// inside a clone and reads the release branch from that clone's committed
// basil.yaml.
func runPublishCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	return runPublish(args, os.Stdin, stdout, stderr, getenv)
}

// runPublish is the testable core: stdin is injected so the confirmation
// prompt can be driven from a pipe.
func runPublish(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil publish", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "Skip the confirmation prompt")
	dryRun := flags.Bool("dry-run", false, "Show what would be published and stop without pushing")

	// flag.Parse stops at the first positional, so `basil publish DIR --yes`
	// needs the directory taken out and the remainder re-parsed for flags
	// placed after it - the same shape deploy/rollback use.
	if err := flags.Parse(args); err != nil {
		printPublishUsage(stderr)
		return &usageError{err: err}
	}
	dir := "."
	if flags.NArg() > 0 {
		dir = flags.Arg(0)
		if err := flags.Parse(flags.Args()[1:]); err != nil {
			printPublishUsage(stderr)
			return &usageError{err: err}
		}
		if flags.NArg() > 0 {
			printPublishUsage(stderr)
			return &usageError{err: fmt.Errorf("unexpected argument %q: publish takes at most one directory", flags.Arg(0))}
		}
	}

	// --- the clone: a git repository with an origin remote ----------------
	top, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("%s is not a git repository: run basil publish inside a clone of your site (git clone https://<host>/.git)", dir)
	}
	top = strings.TrimSpace(top)

	if _, err := gitOutput(top, "remote", "get-url", "origin"); err != nil {
		return fmt.Errorf("no 'origin' remote in %s: publish pushes to origin, so clone your site from the server first", top)
	}

	// --- config: the release branch comes from the clone's basil.yaml -----
	cfgPath, err := findConfigUp(top)
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath, getenv)
	if err != nil {
		return fmt.Errorf("loading %s: %w", cfgPath, err)
	}
	ref := cfg.Deploy.ReleaseRef() // refs/heads/<branch> (or a qualified ref)
	branch := branchShortName(cfg)

	headSHA, err := gitOutput(top, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("cannot read HEAD in %s: %w (is there any commit to publish?)", top, err)
	}
	headSHA = strings.TrimSpace(headSHA)

	// Open Question resolution: a dirty tree warns and proceeds - publish
	// pushes the committed state, never the uncommitted edits.
	if st, err := gitOutput(top, "status", "--porcelain"); err == nil && strings.TrimSpace(st) != "" {
		fmt.Fprintf(stderr, "warning: you have uncommitted changes; publishing the committed state %s\n", shortRelease(headSHA))
	}

	// --- the server's release-branch tip: origin, via git -----------------
	// origin is the site's Git endpoint, so ls-remote reaching it is the
	// closest we get to "what is deployed" without a status endpoint (a
	// server round-trip is the spec's stated preference but has no endpoint
	// yet - deferred). When origin is unreachable, fall back to the clone's
	// cached remote-tracking ref and degrade drift to a warning, never a
	// failure: publish must work without the network answering.
	base := ""
	reachable := true
	if out, err := gitOutput(top, "ls-remote", "origin", ref); err != nil {
		reachable = false
		fmt.Fprintf(stderr, "warning: could not reach origin to check drift (%v); using the last known state from this clone\n", err)
		if cached, cerr := gitOutput(top, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch); cerr == nil {
			base = strings.TrimSpace(cached)
		}
	} else {
		base = firstField(out) // "" when the branch does not exist on origin yet
	}

	// --- nothing to publish -----------------------------------------------
	if base == headSHA {
		fmt.Fprintf(stdout, "Nothing to publish: %s is already the tip of %q on origin.\n", shortRelease(headSHA), branch)
		return nil
	}

	count, files, err := publishPlan(top, base, headSHA)
	if err != nil {
		return fmt.Errorf("cannot determine what would be published: %w", err)
	}
	if count == 0 {
		// base != head but HEAD adds nothing over it: HEAD is behind or has
		// diverged from origin. Pushing would be a no-op or a rejected
		// non-fast-forward; either way there is nothing new to send.
		fmt.Fprintf(stdout, "Nothing to publish: HEAD (%s) has no commits beyond %q on origin (%s). Fetch and rebase if origin is ahead.\n",
			shortRelease(headSHA), branch, shortRelease(base))
		return nil
	}

	// --- the plan: range, changed files, drift ----------------------------
	commitWord := pluralWord(count, "commit", "commits")
	if base == "" {
		fmt.Fprintf(stdout, "Publishing to %q on origin (the release branch does not exist there yet - first publish).\n", branch)
	} else {
		fmt.Fprintf(stdout, "Publishing to %q on origin (%s..%s).\n", branch, shortRelease(base), shortRelease(headSHA))
	}
	fmt.Fprintf(stdout, "\n%d %s:\n", count, commitWord)
	if log, err := publishLog(top, base, headSHA); err == nil && log != "" {
		for _, line := range strings.Split(strings.TrimRight(log, "\n"), "\n") {
			fmt.Fprintf(stdout, "  %s\n", line)
		}
	}
	fmt.Fprintf(stdout, "\n%s changed:\n", plural(len(files), "file"))
	for _, f := range files {
		fmt.Fprintf(stdout, "  %s\n", f)
	}

	// Drift: how far the server's release branch trails this publish. Only
	// meaningful when origin answered; unreachable already warned above.
	if reachable {
		fmt.Fprintf(stdout, "\ndrift: the release branch %q on origin is %d %s behind HEAD (this publish closes it).\n", branch, count, commitWord)
	}

	// --- --dry-run: stop here, push nothing -------------------------------
	if *dryRun {
		fmt.Fprintln(stdout, "\n--dry-run: nothing was pushed.")
		return nil
	}

	// --- confirmation: not skippable without --yes ------------------------
	if !*yes {
		fmt.Fprintf(stderr, "\nPublish %d %s to %q? [y/N] ", count, commitWord, branch)
		var response string
		// A read error (EOF on an empty pipe, a closed stdin) leaves response
		// empty, which is a No: silence never becomes consent.
		fmt.Fscanln(stdin, &response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(stdout, "Cancelled. Nothing was pushed.")
			return nil
		}
	}

	// --- the push: streamed so the server's remote: lines appear live -----
	fmt.Fprintf(stdout, "\nPushing %s to %q...\n", shortRelease(headSHA), branch)
	if err := streamGit(top, stdout, stderr, "push", "origin", "HEAD:"+ref); err != nil {
		// The rejection reason (validation file:line, non-fast-forward) has
		// already streamed as remote: lines. Do not swallow it: exit non-zero.
		return fmt.Errorf("publish failed: the push to %q was rejected (see the messages above)", branch)
	}

	// Configure the refspec on first use so a later bare `git push` also
	// publishes. Best-effort convenience: the explicit HEAD:<ref> above is
	// what actually makes this push work, so any failure here is ignored.
	if reachable && base == "" {
		configureReleasePush(top, ref)
	}

	fmt.Fprintf(stdout, "Published %s to %q.\n", shortRelease(headSHA), branch)
	return nil
}

// publishPlan counts the commits and lists the files HEAD adds over base. When
// base is empty (the release branch does not exist on origin yet) everything
// reachable from HEAD is new, so the whole tree is the change set.
func publishPlan(dir, base, head string) (int, []string, error) {
	if base == "" {
		countOut, err := gitOutput(dir, "rev-list", "--count", head)
		if err != nil {
			return 0, nil, err
		}
		filesOut, err := gitOutput(dir, "ls-tree", "-r", "--name-only", head)
		if err != nil {
			return 0, nil, err
		}
		return atoiTrim(countOut), splitLines(filesOut), nil
	}
	countOut, err := gitOutput(dir, "rev-list", "--count", base+".."+head)
	if err != nil {
		return 0, nil, err
	}
	filesOut, err := gitOutput(dir, "diff", "--name-only", base, head)
	if err != nil {
		return 0, nil, err
	}
	return atoiTrim(countOut), splitLines(filesOut), nil
}

// publishLog is the one-line-per-commit summary of what HEAD adds over base.
func publishLog(dir, base, head string) (string, error) {
	rangeArg := head
	if base != "" {
		rangeArg = base + ".." + head
	}
	return gitOutput(dir, "log", "--oneline", "--no-decorate", rangeArg)
}

// configureReleasePush stores remote.origin.push so a bare `git push` from
// this clone publishes to the release branch. Best-effort; errors ignored.
func configureReleasePush(dir, ref string) {
	if existing, err := gitOutput(dir, "config", "--get", "remote.origin.push"); err == nil && strings.TrimSpace(existing) != "" {
		return
	}
	_ = runGit(dir, "config", "remote.origin.push", "HEAD:"+ref)
}

// streamGit runs git with stdout/stderr wired straight to the caller's
// writers, so a push's remote: lines appear as the server emits them. It keeps
// GIT_TERMINAL_PROMPT=0 so a missing credential fails fast (the OS keychain or
// a configured helper must supply the API key) rather than hanging on a prompt.
func streamGit(dir string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// findConfigUp walks up from start looking for basil.yaml, so publish finds
// the clone's config whether it is run from the repo root or a subdirectory.
func findConfigUp(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", start, err)
	}
	for {
		candidate := filepath.Join(dir, config.ConfigFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s in %s or its parents: publish needs the site config to find the release branch", config.ConfigFileName, start)
		}
		dir = parent
	}
}

// firstField returns the first whitespace-delimited token of s (an ls-remote
// line is "<sha>\t<ref>"), or "" when s is empty.
func firstField(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// splitLines returns the non-empty lines of s.
func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// atoiTrim parses a git --count output, returning 0 on anything unexpected.
func atoiTrim(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

// pluralWord picks the singular or plural word for n (no count prefix, unlike
// plural in fmt.go which renders "N noun").
func pluralWord(n int, singular, pluralForm string) string {
	if n == 1 {
		return singular
	}
	return pluralForm
}

func printPublishUsage(w io.Writer) {
	fmt.Fprintf(w, `basil publish - Push the current commit to the release branch

Usage:
  basil publish [dir] [options]

Run inside a clone of your site (git clone https://<host>/.git). publish reads
the release branch from the clone's basil.yaml, shows the commits and files it
would send, reports drift against origin, then asks to confirm before pushing
git push origin HEAD:<release-branch> with the server's output streamed live.

Options:
  --yes        Skip the confirmation prompt (for scripts)
  --dry-run    Show what would be published and stop without pushing

Exit codes:
  0  success (or nothing to publish, or cancelled)
  1  the push was rejected, or publish could not run
  2  usage error

Examples:
  basil publish
  basil publish --dry-run
  basil publish --yes
`)
}
