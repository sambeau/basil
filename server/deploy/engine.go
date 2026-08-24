package deploy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/sambeau/basil/server/config"
)

// HookFileName is the post-deploy hook, looked for in the release ROOT (not
// site/). Its presence is the whole configuration: convention, like
// index.pars, never a config key.
const HookFileName = "deploy.pars"

// DefaultLockWait is how long Deploy and Rollback wait for a concurrent
// deploy to finish before giving up. Long enough to ride out a normal
// deploy, short enough that a human at the CLI gets an answer.
const DefaultLockWait = 30 * time.Second

// Engine runs the deploy pipeline: resolve, lock, materialise, validate,
// activate, hook, record, prune. It is transport-agnostic — it takes a
// commit and a trigger label and knows nothing about pushes or HTTP.
//
// The zero value is not useful; build one with NewEngine or fill in every
// path field. Fields are plain so the CLI (and tests) can override per-run
// settings after construction.
type Engine struct {
	SiteRoot   string // The site root (holds releases/, current, the lock)
	RepoDir    string // Repository commits are resolved and extracted from
	RecordPath string // Deploy record database (cfg.DeployDBPath())
	Keep       int    // Releases to retain when pruning (deploy.keep)
	Publisher  string // Who pushed the button: Basil account, or cli:<user>
	Trigger    string // TriggerCLI, TriggerHook, ... (defaults to TriggerCLI)
	NoValidate bool   // Emergency override: skip the validation gate

	// LockWait is how long to wait for the deploy lock; 0 refuses
	// immediately when another deploy holds it. NewEngine sets
	// DefaultLockWait.
	LockWait time.Duration

	// AfterActivate, when set, runs after `current` is re-pointed and
	// before the post-deploy hook — the in-process swap point for a
	// running server (FEAT-154) and for tests.
	AfterActivate func()

	// Out receives progress and warnings (prune failures, hook failures).
	// Defaults to os.Stderr when nil.
	Out io.Writer
}

// NewEngine builds an engine from a loaded config. Cross-check any override
// the caller applies afterwards (Publisher, Trigger, NoValidate) — the
// config contributes only paths and deploy.keep.
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		SiteRoot:   cfg.SiteRoot,
		RepoDir:    cfg.BareRepoPath(),
		RecordPath: cfg.DeployDBPath(),
		Keep:       cfg.Deploy.Keep,
		Trigger:    TriggerCLI,
		LockWait:   DefaultLockWait,
	}
}

// Result reports what one Deploy or Rollback did.
type Result struct {
	CommitSHA  string        // The activated (or no-op'd) commit
	ReleaseDir string        // Its directory under releases/
	Outcome    string        // OutcomeDeployed, OutcomeNoOp, OutcomeRolledBack
	Reason     string        // Non-fatal trouble (hook failure); empty otherwise
	Duration   time.Duration // Wall time for the whole pipeline
	Pruned     []string      // Release directories removed by pruning
}

// ValidationFailedError is returned when the validation gate refuses a
// release. It carries the structured errors so callers (the CLI now, the
// push path in FEAT-154) can render file:line diagnostics rather than one
// flattened string.
type ValidationFailedError struct {
	CommitSHA string
	Errors    []ValidationError
}

func (e *ValidationFailedError) Error() string {
	lines := make([]string, len(e.Errors))
	for i, v := range e.Errors {
		lines[i] = "  " + v.String()
	}
	return fmt.Sprintf("release %s failed validation:\n%s", shortSHA(e.CommitSHA), strings.Join(lines, "\n"))
}

// RecordFailedError is returned when the pipeline's outcome IS in effect on
// disk — the release activated (or was already live) — but writing the
// deploy record failed. Callers must not report the deploy itself as failed:
// the site changed (or was confirmed live); only the bookkeeping did not.
// The Result is also returned alongside this error so callers can report
// what is live.
type RecordFailedError struct {
	Result *Result
	Err    error
}

func (e *RecordFailedError) Error() string {
	sha := ""
	if e.Result != nil {
		sha = shortSHA(e.Result.CommitSHA)
	}
	return fmt.Sprintf("release %s is live, but writing the deploy record failed: %v", sha, e.Err)
}

func (e *RecordFailedError) Unwrap() error { return e.Err }

// commitIdentity is the commit author, read from the commit itself. The
// publisher (who triggered the deploy) is a separate Engine field: the two
// routinely differ, and the record stores both.
type commitIdentity struct {
	name  string
	email string
}

// Deploy runs the full pipeline for a branch, tag or SHA already present in
// RepoDir. Activation is last among the things that can fail, so every
// failure path leaves the previous release live; directories this run
// created are removed, and every outcome — failure, rejection, no-op — is
// recorded before returning.
func (e *Engine) Deploy(refOrSHA string) (*Result, error) {
	start := time.Now()
	trigger := e.trigger()

	rec, err := OpenRecord(e.RecordPath)
	if err != nil {
		return nil, err
	}
	defer rec.Close()

	// Resolve before locking: it only reads the repository, and an
	// unresolvable ref should not queue behind a running deploy.
	sha, author, err := e.resolve(refOrSHA)
	if err != nil {
		// No SHA to record, so record the ref as given: the record answers
		// "what happened", including "someone deployed a typo".
		return nil, e.recordFailure(rec, refOrSHA, commitIdentity{}, trigger, start, err)
	}

	lock, err := AcquireLock(e.SiteRoot, e.LockWait)
	if err != nil {
		return nil, e.recordFailure(rec, sha, author, trigger, start, err)
	}
	defer lock.Release()

	releasesDir := filepath.Join(e.SiteRoot, config.ReleasesDirName)
	fmt.Fprintf(e.out(), "deploying %s\n", shortSHA(sha))

	// Idempotency: deploying the active commit is a recorded no-op — but
	// only while the release directory actually exists. A dangling
	// `current` (the directory was removed by hand) means the site is down,
	// and deploy must repair it by falling through to Materialise.
	if current, err := CurrentRelease(e.SiteRoot); err == nil && filepath.Base(current) == sha {
		if info, statErr := os.Stat(current); statErr == nil && info.IsDir() {
			res := &Result{CommitSHA: sha, ReleaseDir: current, Outcome: OutcomeNoOp, Duration: time.Since(start)}
			if err := rec.Add(e.entry(sha, author, trigger, start, OutcomeNoOp, "already the active release")); err != nil {
				return res, &RecordFailedError{Result: res, Err: err}
			}
			fmt.Fprintf(e.out(), "%s is already live — nothing to do\n", shortSHA(sha))
			return res, nil
		}
		fmt.Fprintf(e.out(), "%s points at a missing release directory — re-materialising %s\n", config.CurrentLinkName, shortSHA(sha))
	}

	// Materialise is idempotent and returns an existing releases/<sha>
	// as-is, so note NOW whether the directory pre-dates this run: cleanup
	// on failure must only remove what this run created, never a past
	// release that rollback may still need.
	preExisting := false
	if info, err := os.Stat(filepath.Join(releasesDir, sha)); err == nil && info.IsDir() {
		preExisting = true
	}

	releaseDir, err := Materialise(e.RepoDir, sha, releasesDir)
	if err != nil {
		return nil, e.recordFailure(rec, sha, author, trigger, start, err)
	}
	removeIfOurs := func() {
		if !preExisting {
			os.RemoveAll(releaseDir)
		}
	}

	if !e.NoValidate {
		if verrs := Validate(releaseDir); len(verrs) > 0 {
			removeIfOurs()
			vErr := &ValidationFailedError{CommitSHA: sha, Errors: verrs}
			entry := e.entry(sha, author, trigger, start, OutcomeRejected, joinValidationErrors(verrs))
			if recErr := rec.Add(entry); recErr != nil {
				return nil, fmt.Errorf("%w (and recording the rejection failed: %v)", vErr, recErr)
			}
			return nil, vErr
		}
	} else {
		fmt.Fprintf(e.out(), "WARNING: validation skipped (--no-validate)\n")
	}

	if err := SetCurrent(e.SiteRoot, releaseDir); err != nil {
		removeIfOurs()
		return nil, e.recordFailure(rec, sha, author, trigger, start, err)
	}
	if e.AfterActivate != nil {
		e.AfterActivate()
	}

	// The release is live from here on: nothing below may unwind it. A
	// failing hook is recorded and shouted about, never rolled back — a
	// half-run migration is not improved by reverting the code under it.
	reason := ""
	if hookPath := filepath.Join(releaseDir, HookFileName); fileExists(hookPath) {
		if hookErr := runHook(hookPath); hookErr != nil {
			reason = fmt.Sprintf("post-deploy hook %s failed: %v", HookFileName, hookErr)
			fmt.Fprintf(e.out(), "DEPLOY WARNING: %s\n", reason)
			fmt.Fprintf(e.out(), "The release is live. Inspect the hook's work and roll back deliberately if needed: basil rollback\n")
		}
	}

	res := &Result{
		CommitSHA:  sha,
		ReleaseDir: releaseDir,
		Outcome:    OutcomeDeployed,
		Reason:     reason,
		Duration:   time.Since(start),
	}
	// Crash window, documented: between SetCurrent above and this Add, a
	// crash (power loss, kill -9) leaves the release live but unrecorded. A
	// later bare `basil rollback` then resolves "previous" from the record
	// and picks an older release than the operator expects. Accepted for
	// now — a write-ahead activation row is deliberately deferred.
	if err := rec.Add(e.entry(sha, author, trigger, start, OutcomeDeployed, reason)); err != nil {
		// The release IS live; only the bookkeeping failed. Pruning is
		// skipped too: it leans on the record to know which previous
		// release to protect.
		return res, &RecordFailedError{Result: res, Err: err}
	}

	// Never prune the previous successfully-activated release: a serving
	// process may still be on it (debounced watcher, failed swap, no
	// watcher at all), and it is what rollback rolls back to.
	protect := []string{releaseDir}
	if prevSHA, prevErr := previousDeployedSHA(rec, sha); prevErr == nil && prevSHA != "" {
		protect = append(protect, filepath.Join(releasesDir, prevSHA))
	}
	pruned, err := Prune(releasesDir, e.Keep, protect...)
	if err != nil {
		// Old directories lingering is an inconvenience, not a failed
		// deploy; the next deploy prunes again.
		fmt.Fprintf(e.out(), "warning: pruning old releases: %v\n", err)
	}
	res.Pruned = pruned

	fmt.Fprintf(e.out(), "deployed %s in %s\n", shortSHA(sha), res.Duration.Round(time.Millisecond))
	return res, nil
}

// Rollback re-activates a release already on disk. target "" means the
// previous successfully activated release; otherwise target is a record
// sequence number or a SHA prefix. Rollback never re-materialises — being a
// symlink swap is what makes it fast enough to be the emergency answer — so
// a pruned release is refused plainly rather than rebuilt.
func (e *Engine) Rollback(target string) (*Result, error) {
	start := time.Now()

	rec, err := OpenRecord(e.RecordPath)
	if err != nil {
		return nil, err
	}
	defer rec.Close()

	// Refusals below have no resolved SHA yet, but the record requires one
	// (Add rejects an empty commit_sha), so they are recorded under the
	// target as the operator gave it — matching how Deploy records a typo'd
	// ref — or "rollback" for the bare form.
	recTarget := target
	if recTarget == "" {
		recTarget = "rollback"
	}

	lock, err := AcquireLock(e.SiteRoot, e.LockWait)
	if err != nil {
		return nil, e.recordRollbackFailure(rec, recTarget, commitIdentity{}, start, err)
	}
	defer lock.Release()

	currentSHA := ""
	if current, err := CurrentRelease(e.SiteRoot); err == nil {
		currentSHA = filepath.Base(current)
	}

	releasesDir := filepath.Join(e.SiteRoot, config.ReleasesDirName)

	var sha string
	if target == "" {
		sha, err = previousDeployedSHA(rec, currentSHA)
	} else {
		sha, err = resolveReleaseTarget(rec, releasesDir, target)
	}
	if err != nil {
		return nil, e.recordRollbackFailure(rec, recTarget, commitIdentity{}, start, err)
	}

	// The author on a rollback entry is the author of the commit being
	// re-activated. Best-effort: the record is still written if the commit
	// has since been garbage-collected from the repository.
	author, _ := e.commitAuthor(sha)

	if sha == currentSHA {
		res := &Result{CommitSHA: sha, ReleaseDir: filepath.Join(releasesDir, sha), Outcome: OutcomeNoOp, Duration: time.Since(start)}
		if err := rec.Add(e.entry(sha, author, TriggerRollback, start, OutcomeNoOp, "already the active release")); err != nil {
			return res, &RecordFailedError{Result: res, Err: err}
		}
		fmt.Fprintf(e.out(), "%s is already live — nothing to do\n", shortSHA(sha))
		return res, nil
	}

	releaseDir := filepath.Join(releasesDir, sha)
	if info, statErr := os.Stat(releaseDir); statErr != nil || !info.IsDir() {
		cause := fmt.Errorf("release %s is no longer on disk (pruned?) — rollback never re-materialises; deploy it instead: basil deploy %s", shortSHA(sha), shortSHA(sha))
		return nil, e.recordRollbackFailure(rec, sha, author, start, cause)
	}

	if err := SetCurrent(e.SiteRoot, releaseDir); err != nil {
		return nil, e.recordRollbackFailure(rec, sha, author, start, err)
	}
	if e.AfterActivate != nil {
		e.AfterActivate()
	}

	res := &Result{
		CommitSHA:  sha,
		ReleaseDir: releaseDir,
		Outcome:    OutcomeRolledBack,
		Duration:   time.Since(start),
	}
	if err := rec.Add(e.entry(sha, author, TriggerRollback, start, OutcomeRolledBack, "")); err != nil {
		// The rollback IS in effect; only the bookkeeping failed.
		return res, &RecordFailedError{Result: res, Err: err}
	}
	fmt.Fprintf(e.out(), "rolled back to %s\n", shortSHA(sha))
	return res, nil
}

// previousDeployedSHA picks what a bare `basil rollback` means: the most
// recently activated commit that is not the one currently live.
func previousDeployedSHA(rec *Record, currentSHA string) (string, error) {
	entries, err := rec.List(0)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Outcome != OutcomeDeployed && entry.Outcome != OutcomeRolledBack {
			continue
		}
		if entry.CommitSHA == currentSHA {
			continue
		}
		return entry.CommitSHA, nil
	}
	return "", errors.New("nothing to roll back to: the record has no previous successful deploy")
}

// resolveReleaseTarget turns a `basil rollback <target>` argument — a record
// sequence number or a SHA prefix — into a full SHA. Candidates are the
// union of recorded SHAs and directories under releases/, so anything the
// operator can see in `basil releases` or on disk is addressable. An
// ambiguous prefix is refused, not guessed.
func resolveReleaseTarget(rec *Record, releasesDir, target string) (string, error) {
	entries, err := rec.List(0)
	if err != nil {
		return "", err
	}

	seq, seqErr := strconv.ParseInt(target, 10, 64)
	if seqErr == nil {
		for _, entry := range entries {
			if entry.Seq == seq {
				return entry.CommitSHA, nil
			}
		}
		// Not a recorded sequence number — but an all-digit target may
		// still be a perfectly good SHA prefix (hex has ten digits), so
		// fall through to prefix matching rather than refusing here.
	}

	candidates := map[string]bool{}
	for _, entry := range entries {
		candidates[entry.CommitSHA] = true
	}
	if dirs, err := os.ReadDir(releasesDir); err == nil {
		for _, d := range dirs {
			if d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
				candidates[d.Name()] = true
			}
		}
	}

	var matches []string
	for sha := range candidates {
		if strings.HasPrefix(sha, target) {
			matches = append(matches, sha)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		if seqErr == nil {
			return "", fmt.Errorf("no deploy #%d in the record and no release SHA starts with %q (see basil releases)", seq, target)
		}
		return "", fmt.Errorf("no release matches %q (see basil releases)", target)
	default:
		return "", fmt.Errorf("%q is ambiguous: it matches %d releases (see basil releases)", target, len(matches))
	}
}

// resolve turns a branch, tag or (abbreviated) SHA into a full commit SHA
// and reads the commit's author.
func (e *Engine) resolve(refOrSHA string) (string, commitIdentity, error) {
	out, err := gitOutput(e.RepoDir, "rev-parse", "--verify", "--quiet", refOrSHA+"^{commit}")
	if err != nil {
		return "", commitIdentity{}, fmt.Errorf("cannot resolve %q to a commit in %s: %w", refOrSHA, e.RepoDir, err)
	}
	sha := strings.TrimSpace(out)
	author, err := e.commitAuthor(sha)
	if err != nil {
		return "", commitIdentity{}, err
	}
	return sha, author, nil
}

// commitAuthor reads the author fields from the commit itself — the record
// stores who WROTE the release separately from who published it.
func (e *Engine) commitAuthor(sha string) (commitIdentity, error) {
	out, err := gitOutput(e.RepoDir, "log", "-1", "--format=%an%x00%ae", sha)
	if err != nil {
		return commitIdentity{}, fmt.Errorf("reading the author of %s: %w", shortSHA(sha), err)
	}
	name, email, _ := strings.Cut(strings.TrimSpace(out), "\x00")
	return commitIdentity{name: name, email: email}, nil
}

// runHook executes the post-deploy hook the way cmd/pars executes a script:
// parse, then evaluate with @env and @args populated. It returns an error
// for a parse failure or a runtime error object; the caller decides how
// loudly to report it (loudly).
func runHook(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	l := lexer.NewWithFilename(string(src), path)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.StructuredErrors(); len(errs) > 0 {
		return errs[0]
	}

	env := evaluator.NewEnvironmentWithArgs(nil)
	env.Filename = path
	// EXPLICIT security policy, deliberately permissive: the hook is
	// operator-triggered code (basil deploy at a shell), so it may exec and
	// write, the same stance `pars <script>` takes for a script run by hand.
	// A nil policy would behave the same today only by accident — cmd/pars
	// always builds a policy and a nil one skips the exec check entirely —
	// so the decision is written down here where it can be seen. FEAT-154's
	// push trigger runs hooks on behalf of remote users and MUST revisit
	// this policy before reusing runHook.
	env.Security = &evaluator.SecurityPolicy{
		AllowWriteAll:   true,
		AllowExecuteAll: true,
	}
	result := evaluator.Eval(program, env)
	if errObj, ok := result.(*evaluator.Error); ok {
		if errObj.Line > 0 {
			return fmt.Errorf("line %d: %s", errObj.Line, errObj.Message)
		}
		return errors.New(errObj.Message)
	}
	return nil
}

// recordFailure writes a failed entry and returns the original cause
// (wrapped so errors.Is still sees ErrLocked and friends). A failure that
// cannot even be recorded surfaces both errors: an unrecordable deploy
// system is itself an emergency.
func (e *Engine) recordFailure(rec *Record, sha string, author commitIdentity, trigger string, start time.Time, cause error) error {
	if recErr := rec.Add(e.entry(sha, author, trigger, start, OutcomeFailed, cause.Error())); recErr != nil {
		return fmt.Errorf("%w (and recording the failure failed: %v)", cause, recErr)
	}
	return cause
}

func (e *Engine) recordRollbackFailure(rec *Record, sha string, author commitIdentity, start time.Time, cause error) error {
	if recErr := rec.Add(e.entry(sha, author, TriggerRollback, start, OutcomeFailed, cause.Error())); recErr != nil {
		return fmt.Errorf("%w (and recording the failure failed: %v)", cause, recErr)
	}
	return cause
}

func (e *Engine) entry(sha string, author commitIdentity, trigger string, start time.Time, outcome, reason string) Entry {
	return Entry{
		CommitSHA:   sha,
		Trigger:     trigger,
		Publisher:   e.Publisher,
		AuthorName:  author.name,
		AuthorEmail: author.email,
		StartedAt:   start,
		Duration:    time.Since(start),
		Outcome:     outcome,
		Reason:      reason,
	}
}

func (e *Engine) out() io.Writer {
	if e.Out != nil {
		return e.Out
	}
	return os.Stderr
}

func (e *Engine) trigger() string {
	if e.Trigger != "" {
		return e.Trigger
	}
	return TriggerCLI
}

// gitOutput runs git in dir with the same env hygiene as extractCommit and
// cmd/basil's runGit: no system config, no prompting.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
