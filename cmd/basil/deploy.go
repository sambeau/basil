package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// usageError marks an error caused by how a command was invoked - a missing
// argument or an unknown flag - so main can exit 2, the conventional usage
// exit code, rather than 1.
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

// hookFailedError marks a deploy that activated its release but whose
// post-deploy hook failed: the site changed, so exit 0 would hide the hook
// failure from scripts and exit 1 would claim the deploy itself failed.
// Exit 3 says both truths at once.
type hookFailedError struct{ msg string }

func (e *hookFailedError) Error() string { return e.msg }

// exitCode maps run()'s error to the process exit code: 0 success, 2 usage,
// 3 deployed-but-hook-failed, 1 everything else. A RecordFailedError (the
// release is live but the deploy record was not written) stays exit 1: an
// unwritable record is the more urgent problem and must not be softened.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ue *usageError
	if errors.As(err, &ue) {
		return 2
	}
	var he *hookFailedError
	if errors.As(err, &he) {
		return 3
	}
	return 1
}

// addSiteFlags registers the --site/--config pair every deploy subcommand
// accepts, mirroring the server's own flags (main.go).
func addSiteFlags(flags *flag.FlagSet) (site, configPath *string) {
	site = flags.String("site", "", "Path to the site root")
	configPath = flags.String("config", "", "Path to config file")
	return site, configPath
}

// loadDeploySiteConfig resolves --site/--config exactly the way the server
// does: --site finds the config inside the active release, --config names it
// directly, and neither falls back to config.Load's search order (cwd,
// BASIL_CONFIG, ...).
func loadDeploySiteConfig(site, configPath string, stderr io.Writer, getenv func(string) string) (*config.Config, error) {
	if site != "" && configPath != "" {
		printDeployUsage(stderr)
		return nil, &usageError{err: errors.New("--site and --config are alternatives: --site finds the config inside the site's active release")}
	}
	if site != "" {
		resolved, err := config.ConfigPathForSite(site)
		if err != nil {
			return nil, err
		}
		configPath = resolved
	}
	cfg, err := config.Load(configPath, getenv)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// errNotSiteRoot refuses the legacy single-directory layout, which has no
// releases to deploy, roll back or list.
func errNotSiteRoot(verb string) error {
	return fmt.Errorf("this config is not inside a site root (releases/ + current), so there is nothing to %s\n  create a site with: basil --init <folder> --host <hostname> --admin <name>, or point --site at an existing site root", verb)
}

// newCLIEngine builds the deploy engine the way every CLI entry point needs
// it: publisher cli:<os user>, trigger cli, progress on stdout.
func newCLIEngine(cfg *config.Config, stdout io.Writer, getenv func(string) string) *deploy.Engine {
	eng := deploy.NewEngine(cfg)
	eng.Publisher = cliPublisher(getenv)
	eng.Trigger = deploy.TriggerCLI
	eng.Out = stdout
	return eng
}

// cliPublisher identifies who ran the command: the OS account, because a CLI
// deploy has shell access and no Basil account in hand. $USER is the
// fallback for static builds where os/user cannot resolve the uid.
func cliPublisher(getenv func(string) string) string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "cli:" + u.Username
	}
	if name := getenv("USER"); name != "" {
		return "cli:" + name
	}
	return "cli:unknown"
}

// shortRelease abbreviates a full commit SHA for display, matching the
// engine's own output.
func shortRelease(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// branchShortName is the plain branch the release ref names, for display and
// for the `basil deploy <branch>` hint. deploy.branch may be a bare name
// ("live", "main") or a fully-qualified ref; a refs/heads/<name> ref reduces
// to <name>, anything else (a bare name, or the rare refs/tags/<name>) is used
// as Git already resolves it.
func branchShortName(cfg *config.Config) string {
	b := cfg.Deploy.Branch
	if b == "" {
		b = config.DefaultReleaseBranch
	}
	return strings.TrimPrefix(b, "refs/heads/")
}

// devReleaseDriftNote returns a one-line note when the release branch in the
// site's bare repository is ahead of the live release, or "" when there is
// nothing to say or drift cannot be computed. It never fails: a dev-mode
// startup convenience must degrade to silence, not an error.
func devReleaseDriftNote(cfg *config.Config) string {
	if cfg.SiteRoot == "" {
		return ""
	}
	repo := cfg.BareRepoPath()
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return ""
	}
	branch := branchShortName(cfg)
	if _, err := gitOutput(repo, "rev-parse", "--verify", "--quiet", branch); err != nil {
		return ""
	}
	current, err := deploy.CurrentRelease(cfg.SiteRoot)
	if err != nil {
		return ""
	}
	liveSHA := filepath.Base(current)
	out, err := gitOutput(repo, "rev-list", "--count", liveSHA+".."+branch)
	if err != nil {
		return ""
	}
	n := strings.TrimSpace(out)
	if n == "" || n == "0" {
		return ""
	}
	plural := "commits"
	if n == "1" {
		plural = "commit"
	}
	return fmt.Sprintf("note: the release branch %q is %s %s ahead of the live release - deploy it with: basil deploy %s", branch, n, plural, branch)
}

// runDeployCommand handles `basil deploy <sha|branch|tag>`.
func runDeployCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil deploy", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	site, configPath := addSiteFlags(flags)
	noValidate := flags.Bool("no-validate", false, "Skip the validation gate (emergency override)")
	fromHook := flags.String("from-hook", "", "Run as a Git receive hook (pre-receive or post-receive); reads ref updates from stdin")

	// flag stops at the first positional, and `basil deploy live --site X`
	// is the natural way to type this - so parse, take the ref, and parse
	// the remainder for flags placed after it.
	if err := flags.Parse(args); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	if *fromHook != "" {
		// Hook mode takes no positional: the commits arrive on stdin as
		// "<old> <new> <ref>" lines, straight from Git.
		if flags.NArg() > 0 {
			printDeployUsage(stderr)
			return &usageError{err: fmt.Errorf("unexpected argument %q: --from-hook reads ref updates from stdin", flags.Arg(0))}
		}
		return runFromHook(*fromHook, *site, *configPath, os.Stdin, stdout, stderr, getenv)
	}
	if flags.NArg() == 0 {
		printDeployUsage(stderr)
		return &usageError{err: errors.New("missing <sha|branch|tag>: name the commit that should go live")}
	}
	ref := flags.Arg(0)
	if err := flags.Parse(flags.Args()[1:]); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	if flags.NArg() > 0 {
		printDeployUsage(stderr)
		return &usageError{err: fmt.Errorf("unexpected argument %q: deploy takes one commit", flags.Arg(0))}
	}

	cfg, err := loadDeploySiteConfig(*site, *configPath, stderr, getenv)
	if err != nil {
		return err
	}
	if cfg.SiteRoot == "" {
		return errNotSiteRoot("deploy to")
	}

	eng := newCLIEngine(cfg, stdout, getenv)
	eng.NoValidate = *noValidate

	// The engine prints its own progress (deploying/deployed lines) to
	// stdout. A running server notices the activation through its own
	// watcher on `current`; the CLI must not signal it.
	res, err := eng.Deploy(ref)
	if err != nil {
		var vErr *deploy.ValidationFailedError
		if errors.As(err, &vErr) {
			// One grep-able file:line[:col]: message per line, then the
			// reassurance the design promises (§5.4).
			for _, v := range vErr.Errors {
				fmt.Fprintln(stderr, v.String())
			}
			fmt.Fprintln(stderr, "Release rejected. The live site is unchanged.")
			return fmt.Errorf("release %s failed validation with %d error(s)", shortRelease(vErr.CommitSHA), len(vErr.Errors))
		}
		if rfErr := reportRecordFailure(stdout, stderr, err); rfErr != nil {
			return rfErr
		}
		return err
	}

	if res.Outcome == deploy.OutcomeDeployed {
		fmt.Fprintf(stdout, "Live: %s\n", shortRelease(res.CommitSHA))
	}
	if res.Reason != "" {
		// The engine already reported the hook failure loudly (DEPLOY
		// WARNING + rollback advice). This error only changes the exit
		// code: the release is live, but scripts must see that the hook
		// failed.
		return &hookFailedError{msg: fmt.Sprintf("release %s is live, but the post-deploy hook failed (see above)", shortRelease(res.CommitSHA))}
	}
	return nil
}

// reportRecordFailure handles the deploy-succeeded-record-failed outcome
// distinctly: the site changed (or was confirmed live), only the deploy
// record was not written. The message must never claim the deploy failed;
// the non-zero exit flags the record problem for scripts.
func reportRecordFailure(stdout, stderr io.Writer, err error) error {
	var rfErr *deploy.RecordFailedError
	if !errors.As(err, &rfErr) {
		return nil
	}
	if res := rfErr.Result; res != nil {
		fmt.Fprintf(stdout, "Live: %s\n", shortRelease(res.CommitSHA))
	}
	fmt.Fprintf(stderr, "The release is live; the deploy did NOT fail. Only writing the deploy record failed: %v\n", rfErr.Err)
	fmt.Fprintf(stderr, "basil releases and basil rollback may not see this deploy until the record is writable again.\n")
	return err
}

// runRollbackCommand handles `basil rollback [id]`.
func runRollbackCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil rollback", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	site, configPath := addSiteFlags(flags)

	// As with deploy, flags may follow the positional: `basil rollback 3 --site X`.
	if err := flags.Parse(args); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	target := ""
	if flags.NArg() > 0 {
		target = flags.Arg(0)
		if err := flags.Parse(flags.Args()[1:]); err != nil {
			printDeployUsage(stderr)
			return &usageError{err: err}
		}
		if flags.NArg() > 0 {
			printDeployUsage(stderr)
			return &usageError{err: fmt.Errorf("unexpected argument %q: rollback takes at most one release id", flags.Arg(0))}
		}
	}

	cfg, err := loadDeploySiteConfig(*site, *configPath, stderr, getenv)
	if err != nil {
		return err
	}
	if cfg.SiteRoot == "" {
		return errNotSiteRoot("roll back")
	}

	eng := newCLIEngine(cfg, stdout, getenv)
	res, err := eng.Rollback(target)
	if err != nil {
		if rfErr := reportRecordFailure(stdout, stderr, err); rfErr != nil {
			return rfErr
		}
		// The engine's errors are already actionable: "no previous release"
		// names the empty record, "pruned" suggests basil deploy <sha>.
		return err
	}
	fmt.Fprintf(stdout, "Live: %s\n", shortRelease(res.CommitSHA))
	return nil
}

// runReleasesCommand handles `basil releases`: the deploy record as a table,
// newest first, with the live release marked.
func runReleasesCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil releases", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	site, configPath := addSiteFlags(flags)

	if err := flags.Parse(args); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	if flags.NArg() > 0 {
		printDeployUsage(stderr)
		return &usageError{err: fmt.Errorf("unexpected argument %q", flags.Arg(0))}
	}

	cfg, err := loadDeploySiteConfig(*site, *configPath, stderr, getenv)
	if err != nil {
		return err
	}
	if cfg.SiteRoot == "" {
		return errNotSiteRoot("list releases for")
	}

	// A record that was never written is an empty record, not an error -
	// and reading must not create the database file.
	recordPath := cfg.DeployDBPath()
	if _, statErr := os.Stat(recordPath); os.IsNotExist(statErr) {
		fmt.Fprintln(stdout, "No deploys recorded yet.")
		return nil
	}

	rec, err := deploy.OpenRecord(recordPath)
	if err != nil {
		return err
	}
	defer rec.Close()

	entries, err := rec.List(0)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(stdout, "No deploys recorded yet.")
		return nil
	}

	liveSHA := ""
	if current, err := deploy.CurrentRelease(cfg.SiteRoot); err == nil {
		liveSHA = filepath.Base(current)
	}

	fmt.Fprintf(stdout, "  %-4s %-13s %-17s %-9s %-15s %-21s %s\n",
		"SEQ", "RELEASE", "WHEN", "TRIGGER", "PUBLISHER", "AUTHOR", "OUTCOME")
	fmt.Fprintln(stdout, strings.Repeat("-", 92))
	for _, e := range entries {
		marker := " "
		if liveSHA != "" && e.CommitSHA == liveSHA {
			marker = "*"
		}
		fmt.Fprintf(stdout, "%s %-4d %-13s %-17s %-9s %-15s %-21s %s\n",
			marker, e.Seq, shortRelease(e.CommitSHA),
			e.StartedAt.Local().Format("2006-01-02 15:04"),
			e.Trigger, clip(e.Publisher, 15), clip(e.AuthorName, 21), e.Outcome)
		// Any outcome can carry a reason — a deployed release whose hook
		// failed most of all — so print it whenever there is one.
		if e.Reason != "" {
			fmt.Fprintf(stdout, "       %s\n", clip(e.Reason, 100))
		}
	}
	if liveSHA != "" {
		fmt.Fprintln(stdout, "\n* = the live release")
	}
	return nil
}

// runStatusCommand handles `basil status`: what is live, and whether the
// release branch has commits the live release does not.
func runStatusCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	site, configPath := addSiteFlags(flags)

	if err := flags.Parse(args); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	if flags.NArg() > 0 {
		printDeployUsage(stderr)
		return &usageError{err: fmt.Errorf("unexpected argument %q", flags.Arg(0))}
	}

	cfg, err := loadDeploySiteConfig(*site, *configPath, stderr, getenv)
	if err != nil {
		return err
	}

	// Status reports; it does not insist. A legacy layout or a missing
	// repository is stated plainly and exits 0 - only being unable to
	// answer at all is an error.
	if cfg.SiteRoot == "" {
		fmt.Fprintln(stdout, "This config is not inside a site root (releases/ + current): nothing is deployed here.")
		return nil
	}

	liveSHA := ""
	if current, err := deploy.CurrentRelease(cfg.SiteRoot); err == nil {
		liveSHA = filepath.Base(current)
		line := "live: " + shortRelease(liveSHA)
		if e := latestActivation(cfg, liveSHA); e != nil {
			line += fmt.Sprintf("  (deploy #%d, %s, by %s)", e.Seq, e.StartedAt.Local().Format("2006-01-02 15:04"), e.Publisher)
		}
		fmt.Fprintln(stdout, line)
	} else {
		fmt.Fprintf(stdout, "no active release: %v\n", err)
	}

	repo := cfg.BareRepoPath()
	if info, statErr := os.Stat(repo); statErr != nil || !info.IsDir() {
		fmt.Fprintf(stdout, "no repository at %s: pushes have nowhere to arrive (a site created with basil --init has one)\n", repo)
		return nil
	}

	branch := branchShortName(cfg)
	if _, err := gitOutput(repo, "rev-parse", "--verify", "--quiet", branch); err != nil {
		fmt.Fprintf(stdout, "the release branch '%s' does not exist in %s yet\n", branch, repo)
		return nil
	}
	if liveSHA == "" {
		fmt.Fprintf(stdout, "nothing is live yet; deploy the release branch with: basil deploy %s\n", branch)
		return nil
	}

	out, err := gitOutput(repo, "rev-list", "--count", liveSHA+".."+branch)
	if err != nil {
		fmt.Fprintf(stdout, "cannot compare the live release with the release branch '%s': %v\n", branch, err)
		return nil
	}
	switch n := strings.TrimSpace(out); n {
	case "0":
		fmt.Fprintf(stdout, "the release branch '%s' matches the live release\n", branch)
	case "1":
		fmt.Fprintf(stdout, "the release branch '%s' is 1 commit ahead of the live release - deploy it with: basil deploy %s\n", branch, branch)
	default:
		fmt.Fprintf(stdout, "the release branch '%s' is %s commits ahead of the live release - deploy it with: basil deploy %s\n", branch, n, branch)
	}
	return nil
}

// latestActivation returns the most recent record entry that put liveSHA
// live, or nil when the record cannot say (no database, no matching row).
func latestActivation(cfg *config.Config, liveSHA string) *deploy.Entry {
	recordPath := cfg.DeployDBPath()
	if _, err := os.Stat(recordPath); err != nil {
		return nil
	}
	rec, err := deploy.OpenRecord(recordPath)
	if err != nil {
		return nil
	}
	defer rec.Close()
	entries, err := rec.List(0)
	if err != nil {
		return nil
	}
	for i := range entries {
		e := entries[i]
		if e.CommitSHA != liveSHA {
			continue
		}
		if e.Outcome == deploy.OutcomeDeployed || e.Outcome == deploy.OutcomeRolledBack {
			return &e
		}
	}
	return nil
}

// runCheckCommand handles `basil check`: bootstrap preconditions, each
// reported plainly with a fix hint. Hard failures (layout, release, repo,
// hostname, DNS) make the command exit non-zero; environmental observations
// a local process cannot prove either way (port 80 reachability from
// outside, certificate issuance) are reported as notes.
func runCheckCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	site, configPath := addSiteFlags(flags)

	if err := flags.Parse(args); err != nil {
		printDeployUsage(stderr)
		return &usageError{err: err}
	}
	if flags.NArg() > 0 {
		printDeployUsage(stderr)
		return &usageError{err: fmt.Errorf("unexpected argument %q", flags.Arg(0))}
	}

	failed := 0
	fail := func(name, format string, a ...any) {
		failed++
		fmt.Fprintf(stdout, "FAIL  %s: %s\n", name, fmt.Sprintf(format, a...))
	}
	pass := func(name, format string, a ...any) {
		fmt.Fprintf(stdout, "ok    %s: %s\n", name, fmt.Sprintf(format, a...))
	}
	note := func(name, format string, a ...any) {
		fmt.Fprintf(stdout, "note  %s: %s\n", name, fmt.Sprintf(format, a...))
	}

	cfg, err := loadDeploySiteConfig(*site, *configPath, stderr, getenv)
	if err != nil {
		var ue *usageError
		if errors.As(err, &ue) {
			return err
		}
		fail("config", "%v", err)
		return errors.New("1 check failed")
	}
	pass("config", "loads")

	// --- layout: releases/ + current, repository, active release --------
	if cfg.SiteRoot == "" {
		fail("site root", "this config is not inside a site root (releases/ + current) - create one with basil --init")
		note("release", "skipped (no site root)")
		note("repository", "skipped (no site root)")
	} else {
		pass("site root", "%s", cfg.SiteRoot)
		checkActiveRelease(cfg, pass, fail)
		checkRepository(cfg, pass, fail)
	}

	// --- server.host (DESIGN §7.1: identity, not listener) --------------
	host := cfg.Server.Host
	manualCert := cfg.Server.HTTPS.Cert != "" && cfg.Server.HTTPS.Key != ""
	switch {
	case host != "":
		pass("server.host", "%s", host)
	case cfg.Server.Dev:
		note("server.host", "not set (allowed with --dev)")
	case manualCert:
		note("server.host", "not set (allowed with a manually configured certificate)")
	default:
		fail("server.host", "not set - a public server needs its hostname: set server.host in %s", config.ConfigFileName)
	}

	// --- DNS -------------------------------------------------------------
	if host != "" {
		checkDNS(host, pass, fail, note)
	} else {
		note("dns", "skipped (no server.host)")
	}

	// --- port 80 and the certificate: ACME HTTP-01 preconditions ---------
	if cfg.Server.HTTPS.Auto {
		checkPort80(note)
		checkCertificate(cfg, host, pass, note)
	} else if manualCert {
		checkManualCertificate(cfg, pass, fail)
		note("port 80", "skipped (not using automatic certificates)")
	} else {
		note("port 80", "skipped (no HTTPS configured)")
		note("certificate", "skipped (no HTTPS configured)")
	}

	if failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	fmt.Fprintln(stdout, "\nAll checks passed.")
	return nil
}

func checkActiveRelease(cfg *config.Config, pass, fail func(name, format string, a ...any)) {
	current, err := deploy.CurrentRelease(cfg.SiteRoot)
	if err != nil {
		fail("release", "no active release (%v) - deploy one with: basil deploy <sha|branch>", err)
		return
	}
	if info, err := os.Stat(current); err != nil || !info.IsDir() {
		fail("release", "current points at %s, which does not exist - deploy again: basil deploy <sha|branch>", current)
		return
	}
	pass("release", "%s is active", shortRelease(filepath.Base(current)))
}

// checkRepository verifies the bare repository exists and, crucially, that
// it does not resolve inside anything the server serves: a repository under
// public_dir hands the site's entire history to anyone with a browser.
func checkRepository(cfg *config.Config, pass, fail func(name, format string, a ...any)) {
	repo := cfg.BareRepoPath()
	info, err := os.Stat(repo)
	if err != nil || !info.IsDir() {
		fail("repository", "%s does not exist - pushes have nowhere to arrive (a site created with basil --init has one)", repo)
		return
	}
	pass("repository", "%s", repo)

	type servedRoot struct{ label, path string }
	roots := []servedRoot{
		{"public_dir", cfg.PublicDir},
		{"site.path", cfg.Site.Path},
	}
	for i := range cfg.Static {
		roots = append(roots, servedRoot{fmt.Sprintf("static[%d].root", i), cfg.Static[i].Root})
	}

	for _, root := range roots {
		if root.path == "" {
			continue
		}
		if pathContains(root.path, repo) {
			fail("repository placement", "%s resolves inside the served root %s (%s) - anyone could download the site's history; move the repository or change %s", repo, root.label, root.path, root.label)
			return
		}
	}
	pass("repository placement", "not inside any served root")
}

func checkDNS(host string, pass, fail, note func(name, format string, a ...any)) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		fail("dns", "%s does not resolve (%v) - create an A/AAAA record pointing it at this server", host, err)
		return
	}
	local := localAddrSet()
	for _, addr := range addrs {
		if local[addr] {
			pass("dns", "%s resolves to %s (a local interface has that address)", host, addr)
			return
		}
	}
	note("dns", "%s resolves to %s, but no local interface has that address - cannot confirm locally (normal behind NAT or a load balancer)", host, strings.Join(addrs, ", "))
}

func localAddrSet() map[string]bool {
	set := map[string]bool{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return set
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			set[ipNet.IP.String()] = true
		}
	}
	return set
}

// checkPort80 is honest about what a local check can prove: whether
// something on this machine has port 80, and nothing about reachability
// from outside. It never fails the run.
func checkPort80(note func(name, format string, a ...any)) {
	ln, err := net.Listen("tcp", ":80")
	switch {
	case err == nil:
		ln.Close()
		note("port 80", "free - nothing is listening, so the server (which answers ACME challenges there) is probably not running; reachability from outside cannot be verified from here")
	case errors.Is(err, syscall.EADDRINUSE):
		note("port 80", "in use (likely the server); reachability from outside cannot be verified from here")
	case errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM):
		note("port 80", "cannot test - binding it needs root or CAP_NET_BIND_SERVICE (the server has the same requirement)")
	default:
		note("port 80", "cannot test (%v)", err)
	}
}

// checkCertificate looks in the autocert cache the server uses. Absence is
// not a failure: issuance is eager at startup (server.go probeCertificate),
// so the honest report is "will be obtained".
func checkCertificate(cfg *config.Config, host string, pass, note func(name, format string, a ...any)) {
	if host == "" {
		note("certificate", "skipped (no server.host to hold a certificate for)")
		return
	}
	cached := filepath.Join(cfg.Server.HTTPS.CacheDir, host)
	if info, err := os.Stat(cached); err == nil && info.Mode().IsRegular() {
		pass("certificate", "cached for %s in %s", host, cfg.Server.HTTPS.CacheDir)
		return
	}
	note("certificate", "none cached for %s yet - the server obtains one at startup (fix DNS and port 80 first if that fails)", host)
}

func checkManualCertificate(cfg *config.Config, pass, fail func(name, format string, a ...any)) {
	missing := []string{}
	for _, f := range []string{cfg.Server.HTTPS.Cert, cfg.Server.HTTPS.Key} {
		if _, err := os.Stat(f); err != nil {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		fail("certificate", "configured but missing: %s", strings.Join(missing, ", "))
		return
	}
	pass("certificate", "manually configured (%s)", cfg.Server.HTTPS.Cert)
}

// pathContains reports whether p lives at or under root, after resolving
// both through symlinks - `current` is one, so a naive prefix test lies.
// config.RealPath resolves paths that do not exist yet as well, so a served
// root the operator has not created cannot slip the repository past this.
func pathContains(root, p string) bool {
	root = config.RealPath(root)
	p = config.RealPath(p)
	if root == p {
		return true
	}
	return strings.HasPrefix(p, root+string(filepath.Separator))
}

// clip shortens a value to fit its table column, counting runes rather than
// bytes so a multi-byte name (author names routinely are) is never cut mid-
// character.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}

func printDeployUsage(w io.Writer) {
	fmt.Fprintf(w, `basil deploy - Publish, inspect and undo releases

Usage:
  basil deploy <sha|branch|tag> [options]   Validate and activate a commit from the site's repository
  basil rollback [id] [options]             Re-activate the previous release, or one named by
                                            sequence number or SHA prefix (see basil releases)
  basil releases [options]                  The deploy record; * marks the live release
  basil status [options]                    What is live, and whether the release branch is ahead
  basil check [options]                     Verify bootstrap preconditions (layout, repository,
                                            hostname, DNS, port 80, certificate)

Options:
  --site PATH        Path to the site root (finds the config in the active release)
  --config PATH      Path to config file (alternative to --site; default: auto-detect)
  --no-validate      deploy only: skip the validation gate (emergency override)
  --from-hook=WHICH  deploy only: run as the pre-receive or post-receive Git hook,
                     reading "<old> <new> <ref>" lines from stdin. Invoked by the
                     hooks Basil installs in <site root>/site.git, not by hand

The running server picks up an activation on its own; deploying with the
server stopped activates the release for its next start.

Exit codes:
  0  success
  1  failure (a failed or rejected deploy leaves the live site unchanged)
  2  usage error
  3  deploy only: the release went live but its post-deploy hook failed

Examples:
  basil deploy live --site /srv/mysite
  basil deploy 4f2a1c9
  basil rollback
  basil rollback 3
  basil releases --site /srv/mysite
  basil status
  basil check --site /srv/mysite

`)
}
