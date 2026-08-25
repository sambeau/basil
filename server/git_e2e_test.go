package server

// End-to-end tests for the FEAT-154 Git transport: a REAL git client driving
// a real `basil --init` site through the full server mux over httptest. The
// receive hooks are ARMED — they exec a basil binary built once per test run
// — so a push here exercises transport, hooks, engine and record together.
//
// The failure paths are the product: the core assertion of this phase is
// that a rejected push leaves the remote ref unmoved and the live site
// untouched.

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// --- fixtures --------------------------------------------------------------

var basilBuild struct {
	once sync.Once
	path string
	err  error
}

// basilBinary builds ./cmd/basil once per test run and returns the binary's
// path. Tests that need it skip when go or git is unavailable.
func basilBinary(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available")
	}
	basilBuild.once.Do(func() {
		dir, err := os.MkdirTemp("", "g2-basil-")
		if err != nil {
			basilBuild.err = err
			return
		}
		bin := filepath.Join(dir, "basil")
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/basil")
		cmd.Dir = ".." // server/ -> module root
		if out, err := cmd.CombinedOutput(); err != nil {
			basilBuild.err = fmt.Errorf("building basil: %v: %s", err, out)
			return
		}
		basilBuild.path = bin
	})
	if basilBuild.err != nil {
		t.Fatalf("basil binary: %v", basilBuild.err)
	}
	return basilBuild.path
}

// gitSite is a real `basil --init` site plus the admin credentials --init
// printed.
type gitSite struct {
	root     string // site root: site.git, releases/, current, data/
	admin    string // admin account name
	adminKey string // the admin's API key
	basil    string // the basil binary the hooks exec
}

// newGitSite runs `basil --init` into a fresh directory: bare repo with the
// starter site on the release branch, release 1 deployed, hooks installed
// and armed (pointing at the built basil binary), admin account with an API
// key.
func newGitSite(t *testing.T) *gitSite {
	t.Helper()
	bin := basilBinary(t)

	root := filepath.Join(t.TempDir(), "mysite")
	cmd := exec.Command(bin, "--init", root, "--host", "localhost", "--admin", "alice")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("basil --init: %v: %s", err, out)
	}

	// The API key is printed once: "  API key: <key>"
	var key string
	for _, line := range strings.Split(string(out), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "API key: "); found {
			key = strings.TrimSpace(rest)
			break
		}
	}
	if key == "" {
		t.Fatalf("could not find the API key in --init output:\n%s", out)
	}

	return &gitSite{root: root, admin: "alice", adminKey: key, basil: bin}
}

func (s *gitSite) repoDir() string { return filepath.Join(s.root, "site.git") }

// currentSHA reads which release the site root's `current` link points at.
func (s *gitSite) currentSHA(t *testing.T) string {
	t.Helper()
	target, err := os.Readlink(filepath.Join(s.root, "current"))
	if err != nil {
		t.Fatalf("reading current: %v", err)
	}
	return filepath.Base(target)
}

// refSHA resolves a ref in the site's bare repository ("" if it does not
// exist).
func (s *gitSite) refSHA(t *testing.T, ref string) string {
	t.Helper()
	out, err := gitRun(s.repoDir(), nil, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// recordEntries reads the deploy record the way the engine writes it.
func (s *gitSite) recordEntries(t *testing.T) []deploy.Entry {
	t.Helper()
	rec, err := deploy.OpenRecord(filepath.Join(s.root, "data", "deploy.db"))
	if err != nil {
		t.Fatalf("opening deploy record: %v", err)
	}
	defer rec.Close()
	entries, err := rec.List(0)
	if err != nil {
		t.Fatalf("listing deploy record: %v", err)
	}
	return entries
}

// startServer builds a real *Server over the site and serves its full mux
// through httptest. dev selects the dev-localhost relaxation (plain HTTP,
// no auth); useTLS wraps the mux in httptest's TLS server for the
// production-shaped cases.
func (s *gitSite) startServer(t *testing.T, dev, useTLS bool) (*Server, *httptest.Server) {
	t.Helper()

	cfgPath := filepath.Join(s.root, "current", config.ConfigFileName)
	cfg, absPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("loading %s: %v", cfgPath, err)
	}
	cfg.Server.Dev = dev

	srv, err := New(cfg, absPath, "test", "test", io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	t.Cleanup(func() {
		if srv.authDB != nil {
			srv.authDB.Close()
		}
		if srv.db != nil {
			srv.db.Close()
		}
		srv.cleanupDevTools()
	})

	// New() re-installed the hooks pointing at os.Executable() — which is
	// the TEST binary inside `go test`. Re-arm them at the real basil
	// binary, or a push would re-enter the test suite.
	if err := deploy.InstallHooksAt(s.repoDir(), s.basil); err != nil {
		t.Fatalf("re-arming hooks: %v", err)
	}

	var ts *httptest.Server
	if useTLS {
		ts = httptest.NewTLSServer(srv.servingHandler())
	} else {
		ts = httptest.NewServer(srv.servingHandler())
	}
	t.Cleanup(ts.Close)
	return srv, ts
}

// cloneURL builds the clone URL, optionally with embedded credentials (the
// username selects a client-side credential; the API key in the password
// field is what authenticates).
func cloneURL(ts *httptest.Server, user, key string) string {
	if key == "" {
		return ts.URL + "/.git"
	}
	u := strings.TrimPrefix(ts.URL, "http://")
	scheme := "http://"
	if after, found := strings.CutPrefix(ts.URL, "https://"); found {
		u = after
		scheme = "https://"
	}
	return scheme + user + ":" + key + "@" + u + "/.git"
}

// gitRun runs git with a hermetic environment and returns combined output.
// Unlike the must-style helpers it returns the error: rejected pushes are
// the point of half these tests.
func gitRun(dir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSL_NO_VERIFY=true", // httptest's TLS certificate is self-signed
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func gitMust(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitRun(dir, nil, args...)
	if err != nil {
		t.Fatalf("git %s: %v:\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

// clone clones url into a fresh directory and gives the clone a commit
// identity.
func clone(t *testing.T, url string) string {
	t.Helper()
	work := filepath.Join(t.TempDir(), "work")
	if out, err := gitRun(".", nil, "clone", "--quiet", url, work); err != nil {
		t.Fatalf("git clone %s: %v:\n%s", url, err, out)
	}
	gitMust(t, work, "config", "user.name", "Test Author")
	gitMust(t, work, "config", "user.email", "author@example.com")
	return work
}

// commit writes one file and commits it, returning the commit SHA.
func commit(t *testing.T, work, name, content, msg string) string {
	t.Helper()
	path := filepath.Join(work, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitMust(t, work, "add", "-A")
	gitMust(t, work, "commit", "--quiet", "--no-verify", "-m", msg)
	return gitMust(t, work, "rev-parse", "HEAD")
}

const releaseBranch = config.DefaultReleaseBranch

// --- the suite -------------------------------------------------------------

// A fresh `basil --init` site is clonable immediately, and the clone holds
// the starter site's history — working files, not an empty repository.
// (BUG-033 companion: there is something to clone before any push.)
func TestGitE2E_CloneFreshSite(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, true, false)

	work := clone(t, cloneURL(ts, "", ""))

	if _, err := os.Stat(filepath.Join(work, "site", "index.pars")); err != nil {
		t.Errorf("clone lacks the starter site: %v", err)
	}
	if _, err := os.Stat(filepath.Join(work, config.ConfigFileName)); err != nil {
		t.Errorf("clone lacks %s: %v", config.ConfigFileName, err)
	}
	if branch := gitMust(t, work, "branch", "--show-current"); branch != releaseBranch {
		t.Errorf("clone checked out %q, want the release branch %q", branch, releaseBranch)
	}
	if head := gitMust(t, work, "rev-parse", "HEAD"); head != site.refSHA(t, releaseBranch) {
		t.Errorf("clone HEAD %s does not match the server's release branch", head)
	}
}

// BUG-033 regression: a freshly initialised site accepts a FIRST push with
// no manual git configuration whatsoever. The old arrangement served a
// non-bare repository whose checked-out branch refused pushes until the
// operator set receive.denyCurrentBranch by hand; the bare repository makes
// the first push just work. This test closes BUG-033.
func TestGitE2E_BUG033_FreshSiteAcceptsFirstPush(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, true, false)

	work := clone(t, cloneURL(ts, "", ""))
	sha := commit(t, work, "site/index.pars", "<h1>\"first push\"</h1>\n", "first push")

	// No git config on the server repo, no config in the clone beyond the
	// commit identity: the push must simply work.
	out, err := gitRun(work, nil, "push", "origin", releaseBranch)
	if err != nil {
		t.Fatalf("BUG-033 reproduced: first push to a fresh site failed: %v:\n%s", err, out)
	}
	if got := site.refSHA(t, releaseBranch); got != sha {
		t.Errorf("release branch is %s, want the pushed %s", got, sha)
	}
}

// Pushing a feature branch stores it and publishes nothing: the branch is
// in the bare repository, the live site is untouched, and no deploy is
// recorded.
func TestGitE2E_PushFeatureBranchStoresOnly(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, true, false)

	liveBefore := site.currentSHA(t)
	recordBefore := len(site.recordEntries(t))

	work := clone(t, cloneURL(ts, "", ""))
	gitMust(t, work, "checkout", "--quiet", "-b", "feature")
	sha := commit(t, work, "site/wip.pars", "<h1>\"wip\"</h1>\n", "wip")

	out, err := gitRun(work, nil, "push", "origin", "feature")
	if err != nil {
		t.Fatalf("pushing a feature branch failed: %v:\n%s", err, out)
	}

	if got := site.refSHA(t, "refs/heads/feature"); got != sha {
		t.Errorf("feature branch stored as %q, want %s", got, sha)
	}
	if got := site.currentSHA(t); got != liveBefore {
		t.Errorf("live site moved to %s on a feature-branch push", got)
	}
	if got := len(site.recordEntries(t)); got != recordBefore {
		t.Errorf("deploy record grew to %d entries on a feature-branch push, want %d", got, recordBefore)
	}
}

// Pushing the release branch deploys: `current` re-points to the new
// release, and the record row carries trigger "push" and the authenticated
// account as publisher (D20). This is the production-shaped path: TLS,
// non-dev, HTTP Basic with the API key.
func TestGitE2E_PushReleaseBranchDeploysWithPublisher(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, false, true)

	work := clone(t, cloneURL(ts, site.admin, site.adminKey))
	sha := commit(t, work, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")

	out, err := gitRun(work, nil, "push", "origin", releaseBranch)
	if err != nil {
		t.Fatalf("release push failed: %v:\n%s", err, out)
	}

	if got := site.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want the pushed %s", got, sha)
	}
	entries := site.recordEntries(t)
	if len(entries) == 0 {
		t.Fatal("deploy record is empty after a release push")
	}
	last := entries[0]
	if last.CommitSHA != sha {
		// The record may list oldest-first; find the row for our SHA.
		found := false
		for _, e := range entries {
			if e.CommitSHA == sha {
				last, found = e, true
				break
			}
		}
		if !found {
			t.Fatalf("no record row for the pushed release %s", sha)
		}
	}
	if last.Trigger != deploy.TriggerPush {
		t.Errorf("record trigger = %q, want %q", last.Trigger, deploy.TriggerPush)
	}
	if last.Publisher != site.admin {
		t.Errorf("record publisher = %q, want the authenticated account %q (D20)", last.Publisher, site.admin)
	}
}

// An EDITOR (not admin) can publish the release branch: the push deploys and
// the record names the editor as publisher. Every other release-push test
// authenticates as the admin; this closes the role-matrix gap the spec review
// named — the editor role is what most contributors actually hold.
func TestGitE2E_EditorPublishesReleaseBranch(t *testing.T) {
	site := newGitSite(t)
	srv, ts := site.startServer(t, false, true)

	// Add an editor with an API key to the site's auth DB.
	editor, err := srv.authDB.CreateUserWithRole("Eddy", "eddy@example.com", auth.RoleEditor)
	if err != nil {
		t.Fatalf("creating editor: %v", err)
	}
	_, editorKey, err := srv.authDB.CreateAPIKey(editor.ID, "e2e")
	if err != nil {
		t.Fatalf("creating editor API key: %v", err)
	}

	work := clone(t, cloneURL(ts, "eddy", editorKey))
	sha := commit(t, work, "site/index.pars", "<h1>\"editor release\"</h1>\n", "editor release")

	out, err := gitRun(work, nil, "push", "origin", releaseBranch)
	if err != nil {
		t.Fatalf("editor release push failed: %v:\n%s", err, out)
	}

	if got := site.currentSHA(t); got != sha {
		t.Errorf("current points at %s, want the editor-pushed %s", got, sha)
	}
	var rec deploy.Entry
	found := false
	for _, e := range site.recordEntries(t) {
		if e.CommitSHA == sha {
			rec, found = e, true
			break
		}
	}
	if !found {
		t.Fatalf("no record row for the editor-pushed release %s", sha)
	}
	if rec.Trigger != deploy.TriggerPush {
		t.Errorf("record trigger = %q, want %q", rec.Trigger, deploy.TriggerPush)
	}
	if rec.Publisher != editor.Name {
		t.Errorf("record publisher = %q, want the editor %q", rec.Publisher, editor.Name)
	}
}

// A broken release commit is REJECTED before the ref moves: git push exits
// non-zero, the remote release branch is unmoved, the developer's terminal
// shows the file:line error, and the live site is unchanged. This is the
// core assertion of the phase.
func TestGitE2E_BrokenReleasePushRejected(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, true, false)

	refBefore := site.refSHA(t, releaseBranch)
	liveBefore := site.currentSHA(t)

	work := clone(t, cloneURL(ts, "", ""))
	commit(t, work, "site/broken.pars", "let x = = 2\n", "broken parsley")

	out, err := gitRun(work, nil, "push", "origin", releaseBranch)
	if err == nil {
		t.Fatalf("a broken release was accepted:\n%s", out)
	}
	if !strings.Contains(out, "broken.pars:1") {
		t.Errorf("push output lacks the file:line error:\n%s", out)
	}
	if !strings.Contains(out, "remote:") {
		t.Errorf("hook output did not reach the client as remote: lines:\n%s", out)
	}

	if got := site.refSHA(t, releaseBranch); got != refBefore {
		t.Errorf("REJECTED push moved the release branch: %s -> %s", refBefore, got)
	}
	if got := site.currentSHA(t); got != liveBefore {
		t.Errorf("REJECTED push changed the live site: %s -> %s", liveBefore, got)
	}
}

// The release branch cannot be force-pushed or deleted, by anyone, with no
// setting to permit it.
func TestGitE2E_ReleaseBranchForcePushAndDeleteRefused(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, true, false)

	work := clone(t, cloneURL(ts, "", ""))
	sha := commit(t, work, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	if out, err := gitRun(work, nil, "push", "origin", releaseBranch); err != nil {
		t.Fatalf("setup push failed: %v:\n%s", err, out)
	}

	// Rewrite history and force-push.
	gitMust(t, work, "reset", "--hard", "HEAD~1")
	commit(t, work, "site/index.pars", "<h1>\"rewritten\"</h1>\n", "rewritten")
	out, err := gitRun(work, nil, "push", "--force", "origin", releaseBranch)
	if err == nil {
		t.Fatalf("force-push of the release branch was accepted:\n%s", out)
	}
	if !strings.Contains(out, "force-push") {
		t.Errorf("force-push refusal does not say why:\n%s", out)
	}
	if got := site.refSHA(t, releaseBranch); got != sha {
		t.Errorf("refused force-push moved the release branch: want %s, got %s", sha, got)
	}

	// Delete the release branch.
	out, err = gitRun(work, nil, "push", "origin", ":"+releaseBranch)
	if err == nil {
		t.Fatalf("deletion of the release branch was accepted:\n%s", out)
	}
	if !strings.Contains(out, "cannot be deleted") {
		t.Errorf("deletion refusal does not say why:\n%s", out)
	}
	if got := site.refSHA(t, releaseBranch); got != sha {
		t.Errorf("refused deletion removed or moved the release branch: want %s, got %s", sha, got)
	}
}

// Role matrix over the real transport: any authenticated user clones; a
// viewer cannot push (and is told which role could); an editor can.
func TestGitE2E_RoleMatrix(t *testing.T) {
	site := newGitSite(t)
	srv, ts := site.startServer(t, false, true)

	viewer, err := srv.authDB.CreateUserWithRole("Vera", "vera@example.com", auth.RoleViewer)
	if err != nil {
		t.Fatalf("creating viewer: %v", err)
	}
	_, viewerKey, err := srv.authDB.CreateAPIKey(viewer.ID, "vera-key")
	if err != nil {
		t.Fatalf("viewer API key: %v", err)
	}
	editor, err := srv.authDB.CreateUserWithRole("Eddy", "eddy@example.com", auth.RoleEditor)
	if err != nil {
		t.Fatalf("creating editor: %v", err)
	}
	_, editorKey, err := srv.authDB.CreateAPIKey(editor.ID, "eddy-key")
	if err != nil {
		t.Fatalf("editor API key: %v", err)
	}

	// Viewer clones successfully…
	work := clone(t, cloneURL(ts, "vera", viewerKey))
	commit(t, work, "site/wip.pars", "<h1>\"wip\"</h1>\n", "wip")

	// …but cannot push, and the refusal names the required role.
	out, err := gitRun(work, nil, "push", "origin", releaseBranch)
	if err == nil {
		t.Fatalf("viewer push was accepted:\n%s", out)
	}
	if !strings.Contains(out, "editor or admin") {
		t.Errorf("viewer refusal should name the required role:\n%s", out)
	}

	// An editor pushes a feature branch fine.
	work2 := clone(t, cloneURL(ts, "eddy", editorKey))
	gitMust(t, work2, "checkout", "--quiet", "-b", "feature")
	sha := commit(t, work2, "site/feature.pars", "<h1>\"feature\"</h1>\n", "feature")
	if out, err := gitRun(work2, nil, "push", "origin", "feature"); err != nil {
		t.Fatalf("editor feature push failed: %v:\n%s", err, out)
	}
	if got := site.refSHA(t, "refs/heads/feature"); got != sha {
		t.Errorf("feature branch stored as %q, want %s", got, sha)
	}

	// Wrong key: authentication fails outright.
	if out, err := gitRun(".", nil, "clone", "--quiet", cloneURL(ts, "mallory", "not-a-key"), filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Errorf("clone with an invalid API key succeeded:\n%s", out)
	}
}

// No auth database, no Git: a non-dev server with a repository but no auth
// database refuses to start at all — the guarantee is stated at startup,
// not discovered per request.
func TestGitE2E_NoAuthDBNoGit(t *testing.T) {
	site := newGitSite(t)

	cfgPath := filepath.Join(site.root, "current", config.ConfigFileName)
	cfg, absPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.Server.Dev = false
	cfg.Auth.Enabled = false // no auth database will be opened

	_, err = New(cfg, absPath, "test", "test", io.Discard, io.Discard)
	if err == nil {
		t.Fatal("New() accepted Git deploy with no auth database")
	}
	if !strings.Contains(err.Error(), "auth database") {
		t.Errorf("refusal should name the auth database, got: %v", err)
	}

	// git.enabled: false is the escape hatch: the same site starts fine
	// with Git switched off.
	cfg2, absPath2, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg2.Server.Dev = false
	cfg2.Auth.Enabled = false
	cfg2.Git.Enabled = false
	srv, err := New(cfg2, absPath2, "test", "test", io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("New() with git.enabled: false failed: %v", err)
	}
	if srv.gitHandler != nil {
		t.Error("git.enabled: false still built a Git handler")
	}
	if srv.db != nil {
		srv.db.Close()
	}
	srv.cleanupDevTools()
}

// Plain HTTP: refused for a non-dev server even from localhost; allowed for
// dev-localhost. Verified against the real listener with a real client.
func TestGitE2E_PlainHTTPRefusedOutsideDev(t *testing.T) {
	site := newGitSite(t)
	_, ts := site.startServer(t, false, false) // non-dev, plain HTTP listener

	// The raw refusal, visible to any HTTP client.
	resp, err := http.Get(ts.URL + "/.git/info/refs?service=git-upload-pack")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("plain-HTTP Git request: got %d, want 403", resp.StatusCode)
	}
	if !bytes.Contains(body, []byte("API key")) {
		t.Errorf("refusal body should explain the exposure, got: %s", body)
	}

	// And the real client cannot clone, even with valid credentials.
	if out, err := gitRun(".", nil, "clone", "--quiet", cloneURL(ts, site.admin, site.adminKey), filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Errorf("clone over plain HTTP succeeded against a non-dev server:\n%s", out)
	}

	// The rest of the site is untouched by the Git refusal.
	resp2, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode == http.StatusForbidden {
		t.Errorf("the plain-HTTP refusal leaked beyond Git paths: GET / returned 403")
	}
}

// The startup re-install heals a deleted hook, and a foreign hook is a hard
// startup error naming the file — an operator hook silently swallowing
// deploys is the failure mode Basil refuses to run with.
func TestGitE2E_StartupHookInstall(t *testing.T) {
	site := newGitSite(t)
	preReceive := filepath.Join(site.repoDir(), "hooks", "pre-receive")

	// Delete a hook; startup brings it back.
	if err := os.Remove(preReceive); err != nil {
		t.Fatal(err)
	}
	site.startServer(t, true, false)
	if _, err := os.Stat(preReceive); err != nil {
		t.Errorf("startup did not re-install the deleted pre-receive hook: %v", err)
	}

	// Replace it with an operator's own script; startup refuses, naming it.
	if err := os.WriteFile(preReceive, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(site.root, "current", config.ConfigFileName)
	cfg, absPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.Server.Dev = true
	_, err = New(cfg, absPath, "test", "test", io.Discard, io.Discard)
	if err == nil {
		t.Fatal("New() accepted a foreign pre-receive hook")
	}
	if !strings.Contains(err.Error(), preReceive) {
		t.Errorf("foreign-hook error should name the file %s, got: %v", preReceive, err)
	}
	if !strings.Contains(err.Error(), "not installed by Basil") {
		t.Errorf("foreign-hook error should say whose hook it is, got: %v", err)
	}
}

// A repository that resolves inside a served root refuses startup, naming
// both paths. (The --init layout cannot produce this; a misconfigured
// public_dir can.)
func TestGitE2E_RepoInsideServedRootRefusesStartup(t *testing.T) {
	site := newGitSite(t)

	cfgPath := filepath.Join(site.root, "current", config.ConfigFileName)
	cfg, absPath, err := config.LoadWithPath(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.Server.Dev = true
	// Misconfigure: serve the whole site root as static files, putting
	// site.git inside a served root.
	cfg.Static = append(cfg.Static, config.StaticRoute{Path: "/files/", Root: site.root})

	_, err = New(cfg, absPath, "test", "test", io.Discard, io.Discard)
	if err == nil {
		t.Fatal("New() accepted a repository inside a served root")
	}
	if !strings.Contains(err.Error(), site.repoDir()) || !strings.Contains(err.Error(), site.root) {
		t.Errorf("refusal should name both paths, got: %v", err)
	}
}
