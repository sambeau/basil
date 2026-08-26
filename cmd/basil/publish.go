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
// inside a clone and ASKS THE SERVER which branch releases (FEAT-157), so a
// clone needs no committed setting naming it and an operator who retargets
// site.git's HEAD needs no change on any client.
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
	top, err := clientGitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("%s is not a git repository: run basil publish inside a clone of your site (git clone https://<host>/.git)", dir)
	}
	top = strings.TrimSpace(top)

	if _, err := clientGitOutput(top, "remote", "get-url", "origin"); err != nil {
		return fmt.Errorf("no 'origin' remote in %s: publish pushes to origin, so clone your site from the server first", top)
	}

	// --- config: loaded for its warnings, not for the release branch ------
	// Nothing here decides where the publish goes any more. It is still read
	// because a clone is precisely where a key that was removed from basil.yaml
	// would otherwise sit unnoticed in the file everyone pulls from.
	cfgPath, err := findConfigUp(top)
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath, getenv)
	if err != nil {
		return fmt.Errorf("loading %s: %w", cfgPath, err)
	}
	for _, w := range config.ReleaseWarnings(cfg) {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	headSHA, err := clientGitOutput(top, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("cannot read HEAD in %s: %w (is there any commit to publish?)", top, err)
	}
	headSHA = strings.TrimSpace(headSHA)

	// Open Question resolution: a dirty tree warns and proceeds - publish
	// pushes the committed state, never the uncommitted edits.
	if st, err := clientGitOutput(top, "status", "--porcelain"); err == nil && strings.TrimSpace(st) != "" {
		fmt.Fprintf(stderr, "warning: you have uncommitted changes; publishing the committed state %s\n", shortRelease(headSHA))
	}

	// --- the release branch and its tip: origin, via one ls-remote --------
	// `ls-remote --symref origin HEAD` answers both questions in the round
	// trip publish already made: the "ref: refs/heads/<branch> HEAD" line is
	// the server's release branch, and the sha line beside it is that
	// branch's tip - the closest we get to "what is deployed" without a
	// status endpoint. When origin is unreachable, fall back to what this
	// clone cached about it (refs/remotes/origin/HEAD, which git clone wrote
	// from the server's HEAD) and degrade drift to a warning, never a
	// failure: publish must work without the network answering.
	base := ""
	branch := ""
	reachable := true
	if out, err := clientGitOutput(top, "ls-remote", "--symref", "origin", "HEAD"); err != nil {
		reachable = false
		fmt.Fprintf(stderr, "warning: could not reach origin to check drift (%v); using the last known state from this clone\n", err)
		branch = cachedReleaseBranch(top)
		if branch == "" {
			return fmt.Errorf("cannot publish: origin is unreachable and this clone does not know which branch releases (no refs/remotes/origin/HEAD) - retry when the server answers")
		}
		if cached, cerr := clientGitOutput(top, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch); cerr == nil {
			base = strings.TrimSpace(cached)
		}
	} else {
		branch = symrefBranch(out)
		if branch == "" {
			// Git advertises HEAD as a symref only when it resolves, so the
			// two states that produce silence here - a detached HEAD, and a
			// HEAD retargeted at a branch nobody has pushed yet - are
			// indistinguishable over the protocol. Say so, and name the fix
			// for each: guessing a branch is the mistake FEAT-157 removed.
			return fmt.Errorf("origin advertises no release branch: its HEAD is either detached, or names a branch that does not exist there yet\n  on the server: basil check (it reports which), then git -C <site root>/site.git symbolic-ref HEAD refs/heads/<branch>\n  to create a newly named release branch, push it once explicitly: git push origin HEAD:refs/heads/<branch>")
		}
		// The branch name is server-advertised, and git reads a leading-dash
		// positional as an OPTION: a plain-HTTP or MITM'd origin advertising
		// e.g. `ref: refs/heads/--upload-pack=<cmd>` would smuggle that option
		// into the `git fetch` used to classify a first publish, executing <cmd>
		// locally (BUG-037). Validate here, at the single point every downstream
		// use flows from - the classify fetch, the HEAD:<ref> pushes, the refspec
		// config - so none of them can be handed a hostile value.
		if err := validateReleaseBranch(branch); err != nil {
			return err
		}
		base = symrefTip(out) // "" when the branch does not exist there yet
	}
	ref := "refs/heads/" + branch

	// --- unreachable origin with no cached base ---------------------------
	// A base of "" is only a genuine first publish when origin ANSWERED and
	// lacked the branch. When origin was unreachable AND this clone has no
	// cached remote-tracking ref, there is no server-side tip to diff against,
	// so the range to publish is unknown - it is not a first publish, and
	// dumping the whole tree would misrepresent it. Show what we can about
	// local HEAD and stop; never push in this state.
	if !reachable && base == "" {
		fmt.Fprintf(stdout, "Cannot compute the publish plan for %q: origin is unreachable and this clone has no cached state for it, so the range to publish against the server is unknown.\n", branch)
		fmt.Fprintf(stdout, "Local HEAD is %s.\n", shortRelease(headSHA))
		if *dryRun {
			fmt.Fprintln(stdout, "\n--dry-run: nothing was pushed.")
			return nil
		}
		return fmt.Errorf("cannot publish %q: origin is unreachable and no cached state exists to compute the plan - retry when the server answers", branch)
	}

	// --- nothing to publish -----------------------------------------------
	if base == headSHA {
		fmt.Fprintf(stdout, "Nothing to publish: %s is already the tip of %q on origin.\n", shortRelease(headSHA), branch)
		return nil
	}

	// --- first publish onto a fresh server (BUG-037, graduation) ----------
	// A graduated project - `basil --init` locally, then `git remote add
	// origin` + push to a `basil --init --server` hub - has a history unrelated
	// to the hub's starter commit. Its first `basil publish` is therefore a
	// non-fast-forward, which the server honours exactly once, while its deploy
	// record still shows nothing but the init release (see fromhook.go,
	// acceptsStarterOverwrite). Detect that state here and offer to make the one
	// forced push, rather than computing a `<starter>..HEAD` range whose left
	// side this repo has never seen and letting git rev-list exit 128.
	//
	// The offer is made ONLY for genuinely unrelated histories, and only when
	// origin ANSWERED (a live tip is what the force is measured against). An
	// ordinary divergence - a clone that has merely fallen behind a shared
	// branch - is not this case and must never be force-pushed: it falls through
	// to the normal path below, which refuses the non-fast-forward as before.
	if reachable && base != "" && unrelatedHistories(top, branch, base, headSHA) {
		return firstPublish(top, ref, branch, headSHA, stdin, stdout, stderr, *yes, *dryRun)
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
		// Defensive, and believed unreachable: git advertises HEAD as a symref
		// only when it RESOLVES, so a branch name from symrefBranch implies a
		// sha beside it, and the unreachable-origin path with no cached base
		// already returned above. It stays because the alternative to a wrong
		// message here is a wrong RANGE below - if the invariant ever breaks,
		// publishing the whole tree should at least announce that it is doing
		// so. (A hub whose HEAD names a not-yet-pushed branch does not reach
		// here at all: it fails earlier, naming the one-off first push.)
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
	// meaningful when origin answered - a cached base cannot say where the
	// server is now, and the unreachable case has already warned about
	// exactly that. `base != ""` is the same defensive guard as above (when
	// origin answers, a branch implies a tip); it is kept so this line can
	// never render a range against an empty sha.
	if reachable && base != "" {
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

	// Configure the refspec whenever it is currently unset so a later bare
	// `git push` from this clone also publishes - not only on a first publish.
	// Best-effort convenience: the explicit HEAD:<ref> above is what actually
	// makes this push work, so configureReleasePush ignores any failure. It
	// also re-points a refspec it wrote earlier for a branch that no longer
	// releases, which is announced: a silently rewritten git config is worse
	// than the stale one it replaces.
	if stale := configureReleasePush(top, ref); stale != "" {
		fmt.Fprintf(stdout, "Re-pointed this clone's `git push` default from %q to %q (origin's release branch changed).\n", stale, branch)
	}

	fmt.Fprintf(stdout, "Published %s to %q.\n", shortRelease(headSHA), branch)
	return nil
}

// unrelatedHistories reports whether the server's release tip and local HEAD
// share no common ancestor - the signature of a first publish from a graduated
// project (its history is unrelated to the hub's `basil --init` starter
// commit), and the ONLY divergence for which publish offers a forced push.
//
// The probe has to be authoritative, which means the server tip must be a local
// object before `git merge-base` can rule on it. A graduated project has never
// fetched the starter commit; but neither has a clone that has simply fallen
// behind fetched the newer tip - and merge-base cannot tell "no common
// ancestor" from "one side is an object I have never seen": both exit non-zero.
// So the tip is fetched first. Only when it is in hand AND merge-base still
// reports no base are the histories genuinely unrelated. A fetch that cannot
// land the object leaves the question open, and an open question is never a
// force: an ordinary behind-the-remote clone must fall through to the normal
// path, which refuses the non-fast-forward and tells the developer to rebase.
func unrelatedHistories(dir, branch, serverTip, head string) bool {
	// Best-effort: land the server tip locally so merge-base can see it. A
	// failure just means the guard below stays conservative and offers nothing.
	// `--end-of-options` is defence in depth: the branch is already validated at
	// its source (validateReleaseBranch), but pinning the positional here means a
	// leading-dash value could never be read as a git option even if a future
	// caller reached this without validating first.
	_ = clientGit(dir, "fetch", "--quiet", "origin", "--end-of-options", branch)
	if clientGit(dir, "cat-file", "-e", serverTip+"^{commit}") != nil {
		return false // tip not local: cannot judge, so never force
	}
	// Both commits are local; merge-base is authoritative. Exit 0 => a shared
	// ancestor (ordinary, possibly behind); non-zero => none in common.
	return clientGit(dir, "merge-base", serverTip, head) != nil
}

// firstPublish handles the one non-fast-forward the server allows: replacing
// the hub's `basil --init` starter site with a graduated project's own history
// on its very first publish. It mirrors runPublish's voice - a plan, a
// confirmation (skipped by --yes, previewed by --dry-run), the push streamed
// live so the server's remote: lines appear - but the push is `--force`, and it
// is reached only for unrelated histories, the sole state the server honours it
// in. The equivalent manual crossing, `git push --force origin HEAD:<branch>`,
// still works for anyone who prefers to type it.
func firstPublish(dir, ref, branch, head string, stdin io.Reader, stdout, stderr io.Writer, yes, dryRun bool) error {
	// base "" means "nothing on the server to diff against": the whole tree is
	// what this publish introduces, which is exactly right - the starter site it
	// replaces shares no history with it.
	count, files, err := publishPlan(dir, "", head)
	if err != nil {
		return fmt.Errorf("cannot determine what would be published: %w", err)
	}
	commitWord := pluralWord(count, "commit", "commits")

	// The banner describes what the push WILL attempt, not what the server
	// currently holds: the client cannot tell a hub still on its starter site
	// from one that already carries a real but unrelated release (a second
	// project pointed at a used hub). Both look like unrelated history from here;
	// only the server knows which, and it accepts this force just once - while
	// its deploy record is init-only. Claiming "placeholder site" would be false
	// in the second case and the rejection that followed would seem to come from
	// nowhere. Stating the condition keeps the message true either way.
	fmt.Fprintf(stdout, "First publish to %q on origin.\n", branch)
	fmt.Fprintln(stdout, "\nYour project's history is unrelated to what this server currently publishes,")
	fmt.Fprintln(stdout, "so publishing will force-replace the release branch with your history. The")
	fmt.Fprintln(stdout, "server allows this only if it has never had a real release - otherwise it will")
	fmt.Fprintln(stdout, "refuse, and nothing changes.")
	fmt.Fprintf(stdout, "\n%d %s:\n", count, commitWord)
	if log, err := publishLog(dir, "", head); err == nil && log != "" {
		for _, line := range strings.Split(strings.TrimRight(log, "\n"), "\n") {
			fmt.Fprintf(stdout, "  %s\n", line)
		}
	}
	fmt.Fprintf(stdout, "\n%s changed:\n", plural(len(files), "file"))
	for _, f := range files {
		fmt.Fprintf(stdout, "  %s\n", f)
	}

	// --- --dry-run: preview and push nothing ------------------------------
	if dryRun {
		fmt.Fprintln(stdout, "\n--dry-run: nothing was pushed.")
		return nil
	}

	// --- confirmation: not skippable without --yes ------------------------
	if !yes {
		fmt.Fprintf(stderr, "\nReplace the starter site and publish %d %s to %q? [y/N] ", count, commitWord, branch)
		var response string
		// A read error (EOF on an empty pipe) leaves response empty, a No:
		// silence never becomes consent - least of all for a forced push.
		fmt.Fscanln(stdin, &response)
		if response != "y" && response != "Y" {
			fmt.Fprintln(stdout, "Cancelled. Nothing was pushed.")
			return nil
		}
	}

	// --- the forced push: streamed so the server's remote: lines appear ----
	fmt.Fprintf(stdout, "\nPublishing %s to %q (replacing the starter site)...\n", shortRelease(head), branch)
	if err := streamGit(dir, stdout, stderr, "push", "--force", "origin", "HEAD:"+ref); err != nil {
		// The refusal (a hub that already has a real release; a validation
		// failure) has already streamed as remote: lines. Do not swallow it.
		return fmt.Errorf("publish failed: the push to %q was rejected (see the messages above)", branch)
	}

	// From here on this clone shares history with origin, so configure the
	// refspec once for the ordinary publishes that follow.
	if stale := configureReleasePush(dir, ref); stale != "" {
		fmt.Fprintf(stdout, "Re-pointed this clone's `git push` default from %q to %q (origin's release branch changed).\n", stale, branch)
	}
	fmt.Fprintf(stdout, "Published %s to %q.\n", shortRelease(head), branch)
	return nil
}

// publishPlan counts the commits and lists the files HEAD adds over base. When
// base is empty (nothing known on the server to diff against) everything
// reachable from HEAD is new, so the whole tree is the change set.
func publishPlan(dir, base, head string) (int, []string, error) {
	if base == "" {
		countOut, err := clientGitOutput(dir, "rev-list", "--count", head)
		if err != nil {
			return 0, nil, err
		}
		filesOut, err := clientGitOutput(dir, "ls-tree", "-r", "--name-only", head)
		if err != nil {
			return 0, nil, err
		}
		return atoiTrim(countOut), splitLines(filesOut), nil
	}
	countOut, err := clientGitOutput(dir, "rev-list", "--count", base+".."+head)
	if err != nil {
		return 0, nil, err
	}
	filesOut, err := clientGitOutput(dir, "diff", "--name-only", base, head)
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
	return clientGitOutput(dir, "log", "--oneline", "--no-decorate", rangeArg)
}

// configureReleasePush stores remote.origin.push so a bare `git push` from
// this clone publishes to the release branch. Best-effort; errors ignored.
//
// It also re-points a refspec THIS function wrote for a branch that no longer
// releases. Publish learns the branch from the server every time, so a
// retarget needs no client change — but a clone that had published once kept
// `HEAD:refs/heads/live` forever, and after the retarget a bare `git push`
// from it published nothing, silently, to a branch the hub stores and never
// deploys. That is the "no client change" promise failing in the quietest
// possible way.
//
// Only the exact shape this function writes is rewritten. Anything else in
// remote.origin.push is the developer's own (a multi-ref refspec, a forced
// one, a different source) and is left alone: publish may repair its own
// convenience setting, never edit someone's git config out from under them.
// Returns the previous branch when it re-pointed one, "" otherwise.
func configureReleasePush(dir, ref string) string {
	want := "HEAD:" + ref
	existing, err := clientGitOutput(dir, "config", "--get", "remote.origin.push")
	if err != nil || strings.TrimSpace(existing) == "" {
		_ = clientGit(dir, "config", "remote.origin.push", want)
		return ""
	}
	current := strings.TrimSpace(existing)
	if current == want {
		return ""
	}
	stale, ok := strings.CutPrefix(current, "HEAD:refs/heads/")
	if !ok || stale == "" {
		return "" // not ours to touch
	}
	_ = clientGit(dir, "config", "remote.origin.push", want)
	return stale
}

// clientGitEnv is the environment for every git subprocess publish starts.
//
// GIT_TERMINAL_PROMPT=0 so a missing credential fails fast (the OS keychain or
// a configured helper must supply the API key) rather than hanging a script on
// a prompt that nothing will answer.
//
// It deliberately does NOT set GIT_CONFIG_NOSYSTEM, which the rest of the CLI
// does (see runGit and gitOutput in init.go). Those calls only ever touch a
// local repository, where keeping a machine's system gitconfig out of Basil's
// own bookkeeping is worth something. Publish is different: it talks to the
// remote, and credential helpers are routinely configured in exactly the file
// that variable suppresses. On a stock Mac the osxkeychain helper comes from
// the gitconfig Xcode ships:
//
//	file:/Applications/…/usr/share/git-core/gitconfig	osxkeychain
//
// so setting GIT_CONFIG_NOSYSTEM took the operator's stored API key away and
// every publish died with "could not read Password … terminal prompts
// disabled" — while their own `git push` to the same remote worked (BUG-038).
//
// The rule this encodes: publish is a wrapper around the operator's git, so it
// reads the operator's git configuration, all of it.
func clientGitEnv() []string {
	return append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
}

// clientGit runs a git command in dir, discarding its output. It is publish's
// counterpart to runGit; see clientGitEnv for why publish needs its own.
func clientGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = clientGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// clientGitOutput runs a git command in dir and returns its stdout. It is
// publish's counterpart to gitOutput.
func clientGitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = clientGitEnv()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// streamGit runs git with stdout/stderr wired straight to the caller's
// writers, so a push's remote: lines appear as the server emits them.
func streamGit(dir string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = clientGitEnv()
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
			return "", fmt.Errorf("no %s in %s or its parents: run basil publish inside a clone of your site", config.ConfigFileName, start)
		}
		dir = parent
	}
}

// validateReleaseBranch rejects a server-advertised branch name that git could
// misread as an option or a malformed ref. The name arrives over the wire from
// origin's HEAD, so a hostile or misconfigured server must not be able to steer
// a local git invocation with it. The rules mirror the safe subset of
// git-check-ref-format(1): no leading '-' (the option-injection vector), no
// path traversal or ref-syntax metacharacters, nothing empty. The offending
// value is quoted, never interpolated raw, so the diagnostic itself cannot
// mislead.
func validateReleaseBranch(branch string) error {
	reject := func(reason string) error {
		return fmt.Errorf("origin advertised an unusable release branch name %q (%s); refusing to publish - check the server's site.git HEAD", branch, reason)
	}
	switch {
	case branch == "":
		return reject("empty")
	case strings.HasPrefix(branch, "-"):
		// The load-bearing case: git reads a leading-dash positional as an
		// option (e.g. --upload-pack=<cmd>), which would execute locally.
		return reject("starts with '-'")
	case strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/"):
		return reject("starts or ends with '/'")
	case strings.HasSuffix(branch, "."):
		return reject("ends with '.'")
	case strings.HasSuffix(branch, ".lock"):
		return reject("ends with '.lock'")
	case strings.Contains(branch, ".."):
		return reject("contains '..'")
	case strings.Contains(branch, "//"):
		return reject("contains '//'")
	case strings.Contains(branch, "@{"):
		return reject("contains '@{'")
	}
	// No ASCII control characters, whitespace, or ref-syntax metacharacters
	// (~^:?*[ and backslash) anywhere in the name.
	for _, r := range branch {
		if r <= ' ' || r == 0x7f || strings.ContainsRune("~^:?*[\\", r) {
			return reject("contains a control character or one of ~^:?*[\\ or whitespace")
		}
	}
	return nil
}

// symrefBranch reads the branch out of `ls-remote --symref origin HEAD`,
// whose first line is "ref: refs/heads/<branch>\tHEAD". A detached or
// unreadable HEAD on the server produces no such line, and "" says so.
func symrefBranch(out string) string {
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "ref: ")
		if !ok {
			continue
		}
		ref := firstField(rest)
		if branch, ok := strings.CutPrefix(ref, "refs/heads/"); ok && branch != "" {
			return branch
		}
	}
	return ""
}

// symrefTip reads the sha of HEAD from the same output - the release
// branch's tip on the server. In practice it is always present when
// symrefBranch found a branch, because git advertises HEAD as a symref only
// when it resolves: a HEAD naming a branch that does not exist there yet is
// advertised as neither, and the caller refuses that state by name. "" is
// therefore a should-not-happen, handled rather than trusted.
func symrefTip(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ref: ") {
			continue
		}
		return firstField(line)
	}
	return ""
}

// cachedReleaseBranch is what this clone last knew of the server's HEAD:
// git clone writes refs/remotes/origin/HEAD from it, so the answer is right
// unless the operator retargeted since the last fetch - and an unreachable
// origin cannot be asked.
func cachedReleaseBranch(dir string) string {
	out, err := clientGitOutput(dir, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if err != nil {
		return ""
	}
	ref := strings.TrimSpace(out)
	branch, ok := strings.CutPrefix(ref, "refs/remotes/origin/")
	if !ok {
		return ""
	}
	return branch
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

Run inside a clone of your site (git clone https://<host>/.git). publish asks
the server which branch releases (its site.git HEAD), shows the commits and
files it would send, reports drift against origin, then asks to confirm before
pushing git push origin HEAD:<release-branch> with the server's output
streamed live.

The first publish from a graduated project (basil --init locally, then
git remote add + push to a basil --init --server hub) has a history unrelated
to the server's starter site, so it can only be a non-fast-forward. publish
detects that one state, explains it, and - once confirmed - makes the single
forced push the server allows while it still carries only its starter release.

Options:
  --yes        Skip the confirmation prompt (for scripts)
  --dry-run    Show what would be published and stop without pushing. On a first
               publish it may run a git fetch to classify the state (downloading
               objects, updating remote-tracking refs); it still pushes nothing.

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
