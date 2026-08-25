package main

// `basil deploy --from-hook=<pre-receive|post-receive>`: the receive-hook
// side of Git deploy (FEAT-154). The hooks server/deploy.InstallHooks writes
// are two-line pass-throughs to this mode; all logic lives here where it can
// be tested. Everything printed here reaches the developer's terminal
// prefixed `remote:` by Git itself — never add that prefix by hand.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// zeroSHA is Git's all-zeroes object name: <old> all-zeroes marks a ref
// creation, <new> all-zeroes a deletion.
const zeroSHA = "0000000000000000000000000000000000000000"

// refUpdate is one line of the hook protocol: "<old-sha> <new-sha> <ref-name>".
// pre-receive and post-receive both receive this format on stdin.
type refUpdate struct {
	old, new, ref string
}

// parseRefUpdates reads the hook protocol from stdin. A malformed line is an
// error, not a skip: it means something other than Git is driving the hook.
func parseRefUpdates(r io.Reader) ([]refUpdate, error) {
	var updates []refUpdate
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed ref update line %q: want \"<old-sha> <new-sha> <ref-name>\"", line)
		}
		updates = append(updates, refUpdate{old: fields[0], new: fields[1], ref: fields[2]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ref updates: %w", err)
	}
	return updates, nil
}

// hookSiteRoot resolves the site root when the hook was invoked without
// --site/--config: hooks run with cwd = the bare repository (and GIT_DIR set
// to it), and by the FEAT-152 layout site.git's parent IS the site root.
func hookSiteRoot(getenv func(string) string) (string, error) {
	repoDir := getenv("GIT_DIR")
	if repoDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolving the repository directory: %w", err)
		}
		repoDir = cwd
	}
	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return "", fmt.Errorf("resolving the repository directory %q: %w", repoDir, err)
	}
	root := filepath.Dir(abs)
	if !config.IsSiteRoot(root) {
		return "", fmt.Errorf("%s is not inside a site root (%s has no %s/ + %s): --from-hook expects to run from <site root>/%s", abs, root, config.ReleasesDirName, config.CurrentLinkName, config.BareRepoName)
	}
	return root, nil
}

// hookPublisher is who to record as having pushed the button. The transport
// (server/git.go, the transport unit) exports the authenticated account as
// BASIL_PUBLISHER before invoking git; a push that arrives without it is
// still honestly labelled.
func hookPublisher(getenv func(string) string) string {
	if who := getenv("BASIL_PUBLISHER"); who != "" {
		return who
	}
	return "push"
}

// runFromHook handles one hook invocation. which is "pre-receive" or
// "post-receive"; stdin carries the ref update lines.
func runFromHook(which, site, configPath string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) error {
	switch which {
	case "pre-receive", "post-receive":
	default:
		printDeployUsage(stderr)
		return &usageError{err: fmt.Errorf("--from-hook=%q: must be pre-receive or post-receive", which)}
	}

	if site == "" && configPath == "" {
		root, err := hookSiteRoot(getenv)
		if err != nil {
			return err
		}
		site = root
	}
	cfg, err := loadDeploySiteConfig(site, configPath, stderr, getenv)
	if err != nil {
		return err
	}
	if cfg.SiteRoot == "" {
		return errNotSiteRoot("deploy to")
	}

	updates, err := parseRefUpdates(stdin)
	if err != nil {
		return err
	}

	eng := deploy.NewEngine(cfg)
	eng.Trigger = deploy.TriggerPush
	eng.Publisher = hookPublisher(getenv)
	eng.Out = stdout
	// A push runs deploy.pars on behalf of a remote pusher, so the hook is
	// sandboxed: no @shell/@exec, writes scoped to the data dir. Full shell
	// power stays with `basil deploy` at the server shell (HookSandbox false).
	eng.HookSandbox = true

	// The release branch is site.git's HEAD and nothing else (FEAT-157): the
	// config that would otherwise name it ships inside the release, so a
	// deploy could un-protect the branch the refusals below guard. An
	// unreadable or detached HEAD refuses the push - guessing a branch here
	// is the same mistake in a quieter form.
	releaseRef, err := deploy.ReleaseRef(cfg.BareRepoPath())
	if err != nil {
		fmt.Fprintln(stdout, err)
		return err
	}

	switch which {
	case "pre-receive":
		return runPreReceive(cfg, eng, updates, releaseRef, stdout)
	default:
		return runPostReceive(cfg, eng, updates, releaseRef, stdout, stderr)
	}
}

// runPreReceive validates before any ref moves. Git's pre-receive is
// all-or-nothing: one non-zero exit rejects the ENTIRE push, every ref in it
// — so every update is still examined (each problem gets reported once, not
// drip-fed across retries), and any single refusal fails the run.
func runPreReceive(cfg *config.Config, eng *deploy.Engine, updates []refUpdate, releaseRef string, stdout io.Writer) error {
	var refusal error
	for _, u := range updates {
		if u.ref != releaseRef {
			// Store-and-stop: any other ref is simply accepted into the
			// repository and published to nobody.
			continue
		}
		if u.new == zeroSHA {
			fmt.Fprintln(stdout, "the release branch cannot be deleted")
			refusal = errors.New("release branch deletion refused")
			continue
		}
		if u.old != zeroSHA {
			ancestor, err := isAncestor(cfg.BareRepoPath(), u.old, u.new)
			if err != nil {
				fmt.Fprintf(stdout, "cannot check release-branch ancestry: %v\n", err)
				refusal = err
				continue
			}
			if !ancestor && !acceptsStarterOverwrite(cfg, eng, stdout) {
				fmt.Fprintln(stdout, "force-pushing the release branch rewrites release history, which the deploy record and rollback rely on — it is refused for everyone")
				refusal = errors.New("release branch force-push refused")
				continue
			}
		}

		fmt.Fprintf(stdout, "Checking release %s… ", shortRelease(u.new))
		releaseDir, err := eng.Prepare(u.new)
		if err != nil {
			fmt.Fprintln(stdout) // finish the "Checking" line before the errors
			var vErr *deploy.ValidationFailedError
			if errors.As(err, &vErr) {
				// One file:line[:col]: message per line; Git prefixes each
				// with remote: on the developer's side.
				for _, v := range vErr.Errors {
					fmt.Fprintln(stdout, v.String())
				}
			} else {
				fmt.Fprintln(stdout, err)
			}
			fmt.Fprintf(stdout, "Release rejected. The live site is unchanged%s.\n", stillLive(cfg.SiteRoot))
			refusal = fmt.Errorf("release %s refused", shortRelease(u.new))
			continue
		}
		fmt.Fprintln(stdout, "ok")
		// Formatting is a warning, never a gate: the release passed validation
		// and WILL go live. This runs only after Prepare succeeds (so parse
		// errors are already out of the way), reports on a separate non-fatal
		// channel, and never touches `refusal` — the push stands regardless.
		warnUnformatted(stdout, releaseDir)
		warnListenerChange(stdout, cfg.SiteRoot, releaseDir)
	}
	return refusal
}

// warnListenerChange relays the non-fatal listener notice for a release that
// has already been accepted (FEAT-156). Listener settings stay deployable —
// renaming a site over git is legitimate — so this is a warning, never a
// gate: it reports at push time, where the mistake is one commit deep, and
// NEVER touches the push's exit status. Silence is correct whenever there is
// nothing to compare against (no active release, an unloadable config on
// either side) or the live site is not public.
func warnListenerChange(stdout io.Writer, siteRoot, releaseDir string) {
	active, err := deploy.CurrentRelease(siteRoot)
	if err != nil {
		return
	}
	changes := deploy.ListenerChanges(active, releaseDir)
	if len(changes) == 0 {
		return
	}
	fmt.Fprintln(stdout, "warning: this release changes how the live site is served:")
	for _, c := range changes {
		fmt.Fprintf(stdout, "warning:   %s\n", c)
	}
	fmt.Fprintln(stdout, "The change takes effect when the server restarts, not now. If it was not intended, revert it before then: git revert HEAD && git push.")
}

// acceptsStarterOverwrite decides the single exception to the release-branch
// force-push refusal (FEAT-156, graduation). Server init seeds the release
// branch with its own starter commit, so a developer's local history and the
// hub's are unrelated and the FIRST publish from a graduated site is always a
// non-fast-forward. Refusing it would make graduation impossible without shell
// access on the box.
//
// The exception is decided from the deploy record, never from ancestry: only
// the record can say that nothing but the init starter release has ever gone
// live. Once one real deploy is recorded — successful or not — this returns
// false and the refusal is byte-identical to the shipped one. A record that
// cannot be read is not an exception either (OnlyInitDeployed is conservative);
// the read error is reported so an operator can tell "refused because history
// exists" from "refused because the record is unreadable".
//
// The record is read UNDER THE SITE DEPLOY LOCK. Without it two pushes racing
// at a fresh hub could both read "nothing but init" and both be granted the
// one-time exception: Engine.Deploy holds the same lock from before it
// activates until after it writes the deployed row, so taking it here makes
// the second push wait and then see the first push's row. The lock is
// released before returning — Prepare and Deploy each take it again for
// themselves, and holding it across them would deadlock this process against
// its own file lock.
func acceptsStarterOverwrite(cfg *config.Config, eng *deploy.Engine, stdout io.Writer) bool {
	starter, err := onlyInitDeployedLocked(cfg, eng)
	if err != nil {
		fmt.Fprintf(stdout, "cannot read the deploy record (%v) — treating this site as already deployed\n", err)
		return false
	}
	if !starter {
		return false
	}
	fmt.Fprintln(stdout, "replacing the starter site created by 'basil --init' with your first release — this is the one non-fast-forward the release branch allows, and it will not be allowed again")
	return true
}

// onlyInitDeployedLocked answers deploy.OnlyInitDeployed with the site's
// deploy lock held, so a deploy running in another process cannot be halfway
// through recording itself while the answer is read. A lock that cannot be
// taken is reported as a read failure, which the caller already treats as
// "assume this site has history" — the conservative direction.
func onlyInitDeployedLocked(cfg *config.Config, eng *deploy.Engine) (bool, error) {
	lock, err := deploy.AcquireLock(cfg.SiteRoot, eng.LockWait)
	if err != nil {
		return false, fmt.Errorf("waiting for the deploy lock: %w", err)
	}
	defer lock.Release()
	return deploy.OnlyInitDeployed(cfg.DeployDBPath())
}

// warnUnformatted relays a non-fatal formatting notice for a release that has
// already been accepted. The server never rewrites code (D4b); it only names
// the unformatted files and the fix. It NEVER affects the push's exit status.
func warnUnformatted(stdout io.Writer, releaseDir string) {
	unformatted := deploy.Unformatted(releaseDir)
	if len(unformatted) == 0 {
		return
	}
	fmt.Fprintf(stdout, "warning: %d file(s) are not formatted:\n", len(unformatted))
	for _, f := range unformatted {
		fmt.Fprintf(stdout, "  %s\n", f)
	}
	fmt.Fprintln(stdout, "Run 'basil fmt -w' to format them. The push was accepted.")
}

// runPostReceive activates. By the time this hook runs the ref HAS moved: a
// failure here cannot reject the push, so it is reported loudly (Git relays
// it) and recorded by the engine, and the non-zero exit tells Git to show
// the developer something went wrong.
func runPostReceive(cfg *config.Config, eng *deploy.Engine, updates []refUpdate, releaseRef string, stdout, stderr io.Writer) error {
	var failure error
	for _, u := range updates {
		if u.ref != releaseRef || u.new == zeroSHA {
			continue
		}
		if err := recheckFastForward(cfg, eng, u, stdout, stderr); err != nil {
			failure = err
			continue
		}
		// Validation already ran in pre-receive; Deploy still takes its
		// normal path (including re-validation) rather than a trusting
		// fast lane.
		res, err := eng.Deploy(u.new)
		if err != nil {
			if rfErr := reportRecordFailure(stdout, stderr, err); rfErr != nil {
				failure = rfErr
				continue
			}
			fmt.Fprintf(stderr, "DEPLOY FAILED after the push was accepted: %v\n", err)
			fmt.Fprintf(stderr, "The push itself stands; the live site is unchanged. Fix the cause, then deploy on the server: basil deploy %s\n", shortRelease(u.new))
			failure = err
			continue
		}
		fmt.Fprintf(stdout, "Deployed %s (%s)\n", shortRelease(res.CommitSHA), res.Duration.Round(time.Millisecond))
		if res.Reason != "" {
			// The engine already shouted about the post-deploy hook failure;
			// the non-zero exit keeps it visible to the developer's push.
			failure = &hookFailedError{msg: fmt.Sprintf("release %s is live, but the post-deploy hook failed (see above)", shortRelease(res.CommitSHA))}
		}
	}
	return failure
}

// recheckFastForward re-runs pre-receive's ancestry gate, here, after the ref
// has moved. The two hooks each read the release branch from site.git's HEAD,
// and they read it at DIFFERENT MOMENTS: an operator retargeting HEAD between
// them leaves an update that pre-receive waved through as "any other ref" —
// store-and-stop, no ancestry check, no starter-overwrite check — arriving
// here as the release branch, about to be published. Every other pre-receive
// gate re-runs on its own (Deploy re-validates); ancestry is the one that
// does not, so it is re-asserted.
//
// This cannot un-move the ref — post-receive never can. It refuses to
// PUBLISH history that nothing checked, loudly, leaving the live site alone.
func recheckFastForward(cfg *config.Config, eng *deploy.Engine, u refUpdate, stdout, stderr io.Writer) error {
	if u.old == zeroSHA {
		return nil // ref creation: there is no earlier value to fast-forward from
	}
	ancestor, err := isAncestor(cfg.BareRepoPath(), u.old, u.new)
	if err != nil {
		fmt.Fprintf(stderr, "DEPLOY REFUSED: cannot check release-branch ancestry: %v\n", err)
		return err
	}
	if ancestor {
		return nil
	}
	// Graduation's one-time exception, decided from the deploy record exactly
	// as pre-receive decides it — but silently: when pre-receive granted it,
	// it already said so, and repeating the announcement here would suggest a
	// second exception was spent.
	if starter, serr := onlyInitDeployedLocked(cfg, eng); serr == nil && starter {
		return nil
	}
	fmt.Fprintf(stderr, "DEPLOY REFUSED: %s is not a fast-forward of %s, and nothing checked it — the release branch moved while this push was in flight (was site.git's HEAD retargeted mid-push?).\n", shortRelease(u.new), shortRelease(u.old))
	fmt.Fprintf(stderr, "The push itself stands; the live site is unchanged%s. Review the branch, then deploy on the server if it is what you meant: basil deploy %s\n", stillLive(cfg.SiteRoot), shortRelease(u.new))
	return fmt.Errorf("release %s refused: not a fast-forward and ungated", shortRelease(u.new))
}

// stillLive names what stayed live, for the rejection message: " (still
// 4f2a1c9)" or "" when nothing is live to name.
func stillLive(siteRoot string) string {
	current, err := deploy.CurrentRelease(siteRoot)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(" (still %s)", shortRelease(filepath.Base(current)))
}

// isAncestor reports whether old is an ancestor of new in the repository at
// repoDir — i.e. whether the update is a fast-forward. `git merge-base
// --is-ancestor` answers with its exit code: 0 yes, 1 no, anything else is a
// real error.
//
// The repository is named with --git-dir, which beats the GIT_DIR the hook
// inherits. cmd.Dir alone would NOT: a GIT_DIR in the environment overrides
// the working directory, so the ancestry gate protecting this repository's
// release branch could be answered by another repository entirely — and a
// half-deleted site.git would let git walk up into an enclosing one.
func isAncestor(repoDir, old, new string) (bool, error) {
	cmd := exec.Command("git", "--git-dir="+repoDir, "merge-base", "--is-ancestor", old, new)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %s", shortRelease(old), shortRelease(new), msg)
}
