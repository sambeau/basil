package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
	"github.com/sambeau/basil/server/deploy"
)

// requireGit skips a test on a machine with no git. `basil --init` needs it:
// it creates the repository and deploys the first release from it.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

// initOpts builds server-mode options: the server topology is what these
// tests are about, and it is opt-in since FEAT-156.
func initOpts(path string, out, errOut *bytes.Buffer) initOptions {
	return initOptions{
		Folder:  path,
		Host:    "mysite.example.com",
		Admin:   "sam",
		Server:  true,
		Stdout:  out,
		Stderr:  errOut,
		Geteuid: func() int { return 1000 },
		Getenv:  func(string) string { return "" },
	}
}

func TestInitCommand_Success(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	// --- the site-root layout ---
	assertDirExists(t, filepath.Join(root, "site.git"))
	assertDirExists(t, filepath.Join(root, "releases"))
	assertDirExists(t, filepath.Join(root, "data"))
	assertDirExists(t, filepath.Join(root, "data", "uploads"))

	if !config.IsSiteRoot(root) {
		t.Fatal("the created tree is not recognised as a site root")
	}

	// --- current points at an actual release with the site in it ---
	link, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatalf("current is not a symlink: %v", err)
	}
	if !strings.HasPrefix(link, "releases/") {
		t.Errorf("current should point inside releases/, got %q", link)
	}
	assertFileExists(t, filepath.Join(root, "current", "basil.yaml"))
	assertFileExists(t, filepath.Join(root, "current", "site", "index.pars"))
	assertFileExists(t, filepath.Join(root, "current", ".githooks", "pre-commit"))

	// --- the generated config loads, and anchors where it should ---
	cfgPath, err := config.ConfigPathForSite(root)
	if err != nil {
		t.Fatalf("ConfigPathForSite: %v", err)
	}
	cfg, err := config.Load(cfgPath, os.Getenv)
	if err != nil {
		t.Fatalf("generated basil.yaml does not load: %v", err)
	}
	if cfg.SiteRoot != root {
		t.Errorf("SiteRoot = %q, want %q", cfg.SiteRoot, root)
	}
	wantData := filepath.Join(root, "data")
	if cfg.DataDir != wantData {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, wantData)
	}
	if !strings.HasPrefix(cfg.Site.Path, filepath.Join(cfg.ReleaseDir)) {
		t.Errorf("site.path = %q, want it inside the release", cfg.Site.Path)
	}
	if cfg.Server.Host != "mysite.example.com" {
		t.Errorf("server.host = %q, want the --host value", cfg.Server.Host)
	}
	// --host is written, so a public server starts without a config edit.
	if err := config.Validate(cfg); err != nil {
		t.Errorf("generated config does not pass production validation: %v", err)
	}

	// --- the admin account and its key ---
	assertFileExists(t, filepath.Join(root, "data", ".basil-auth.db"))
	out := stdout.String()
	if !strings.Contains(out, "API key: bsl_") {
		t.Errorf("no API key printed:\n%s", out)
	}
	if !strings.Contains(out, "shown once") {
		t.Error("the API key is not described as unrepeatable")
	}
	if !strings.Contains(out, "git clone https://sam@mysite.example.com/.git") {
		t.Errorf("no exact clone command printed:\n%s", out)
	}
	if !strings.Contains(out, "release 1") {
		t.Errorf("the first release is not reported:\n%s", out)
	}

	indexContent := readFile(t, filepath.Join(root, "current", "site", "index.pars"))
	if indexContent != "<h1>\"🌿 Hello from Basil 👋\"</h1>\n" {
		t.Errorf("unexpected index.pars content: %q", indexContent)
	}
}

// The repository must be clonable immediately: a clone of a site with no
// commits is the cold-start deadlock.
func TestInitCommand_RepositoryIsClonable(t *testing.T) {
	requireGit(t)
	tmp := t.TempDir()
	root := filepath.Join(tmp, "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	clone := filepath.Join(tmp, "clone")
	cmd := exec.Command("git", "clone", "--quiet", filepath.Join(root, "site.git"), clone)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %v: %s", err, out)
	}

	assertFileExists(t, filepath.Join(clone, "site", "index.pars"))
	assertFileExists(t, filepath.Join(clone, "basil.yaml"))
	assertFileExists(t, filepath.Join(clone, ".githooks", "pre-commit"))

	branch, err := exec.Command("git", "-C", clone, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("reading the checked-out branch: %v", err)
	}
	if strings.TrimSpace(string(branch)) != releaseBranch {
		t.Errorf("clone checked out %q, want the release branch %q", strings.TrimSpace(string(branch)), releaseBranch)
	}
}

// The pre-commit hook --init writes formats staged .pars files with the
// operator's `basil` binary, not the standalone `pars` tool that requiring it
// was the wart FEAT-155 removed.
func TestInitCommand_PreCommitHookUsesBasilFmt(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	hookPath := filepath.Join(root, "current", ".githooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("reading pre-commit hook: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "basil fmt -w") {
		t.Errorf("pre-commit hook does not run 'basil fmt -w':\n%s", content)
	}
	// The wart FEAT-155 removed was invoking the standalone `pars` binary.
	// ".pars" (the file suffix) and "parsley" are fine; `pars fmt` / `pars `
	// as a command, and guarding on `command -v pars`, are not.
	if strings.Contains(content, "pars fmt") || strings.Contains(content, "command -v pars ") {
		t.Errorf("pre-commit hook still invokes the standalone 'pars' tool:\n%s", content)
	}
	if !strings.Contains(content, "command -v basil") {
		t.Errorf("pre-commit hook does not guard on 'command -v basil':\n%s", content)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("pre-commit hook mode = %v, want 0755", info.Mode().Perm())
	}
}

// The hook is not just correct text: run its body against a real staged .pars
// with `basil` on PATH and it formats the file and re-stages it.
func TestInitCommand_PreCommitHookFormatsStagedPars(t *testing.T) {
	requireGit(t)

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	basilExe := filepath.Join(binDir, "basil")
	build := exec.Command("go", "build", "-o", basilExe, "github.com/sambeau/basil/cmd/basil")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building basil: %v\n%s", err, out)
	}

	// A fresh clone-like repo with the hook wired in via core.hooksPath.
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".githooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".githooks", "pre-commit"), []byte(preCommitHook), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "--initial-branch=main")
	runGit("config", "core.hooksPath", ".githooks")
	runGit("config", "user.name", "Test")
	runGit("config", "user.email", "test@example.com")

	messy := filepath.Join(repo, "page.pars")
	if err := os.WriteFile(messy, []byte(fmtMessy), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "page.pars")
	runGit("commit", "-m", "add page")

	// The working tree file was rewritten in place by the hook.
	if got := readFile(t, messy); got != fmtFormatted {
		t.Errorf("pre-commit hook did not format page.pars: got %q want %q", got, fmtFormatted)
	}
	// And the committed blob is the formatted version (the hook re-staged it).
	show := exec.Command("git", "show", "HEAD:page.pars")
	show.Dir = repo
	show.Env = append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	committed, err := show.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if string(committed) != fmtFormatted {
		t.Errorf("committed blob not formatted: got %q want %q", committed, fmtFormatted)
	}
}

// No runtime state may live inside the release, because a deploy replaces it.
func TestInitCommand_NoStateInsideTheRelease(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	release, err := filepath.EvalSymlinks(filepath.Join(root, "current"))
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.Walk(release, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := filepath.Base(path)
		switch {
		case strings.HasSuffix(name, ".db"), name == "certs", name == "logs", name == "uploads":
			t.Errorf("runtime state inside the release: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitCommand_RequiresHost(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Host = ""

	err := runInitCommand(opts)
	if err == nil {
		t.Fatal("expected --init to refuse without --host")
	}
	if !strings.Contains(err.Error(), "--host is required") {
		t.Errorf("the error does not name the fix: %v", err)
	}
	if _, statErr := os.Stat(root); statErr == nil {
		t.Error("a refused init should not have created anything")
	}
}

func TestInitCommand_RequiresAdminWhenNotInteractive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Admin = ""
	opts.Interactive = false

	err := runInitCommand(opts)
	if err == nil {
		t.Fatal("expected --init to refuse without --admin")
	}
	if !strings.Contains(err.Error(), "--admin is required") {
		t.Errorf("the error does not name the fix: %v", err)
	}
	// The one thing it must never do is guess.
	if !strings.Contains(err.Error(), "$USER") {
		t.Errorf("the error should say the name is not derived from the environment: %v", err)
	}
}

func TestInitCommand_PromptsForAdminWhenInteractive(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Admin = ""
	opts.Interactive = true
	opts.Stdin = strings.NewReader("alice\n")

	if err := runInitCommand(opts); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Admin account name:") {
		t.Error("no prompt was shown")
	}
	if !strings.Contains(stdout.String(), "admin account 'alice'") {
		t.Errorf("the prompted name was not used:\n%s", stdout.String())
	}
}

// $USER is never consulted, even when it is set to something plausible.
func TestInitCommand_NeverDerivesAdminFromEnvironment(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Admin = ""
	opts.Interactive = false
	opts.Getenv = func(k string) string {
		if k == "USER" || k == "SUDO_USER" {
			return "deploybot"
		}
		return ""
	}

	if err := runInitCommand(opts); err == nil {
		t.Fatal("expected a refusal, not a name taken from the environment")
	}
}

func TestInitCommand_WarnsWhenAdminIsRoot(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Admin = "root"

	if err := runInitCommand(opts); err != nil {
		t.Fatalf("--init should accept an explicit 'root': %v", err)
	}
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("no warning about the 'root' account name:\n%s", stderr.String())
	}
}

// Running as uid 0 with no SUDO_USER must warn and print the exact command,
// or the operator meets this as a permission error on the first request.
func TestInitCommand_WarnsAboutRootOwnedTree(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Geteuid = func() int { return 0 }
	opts.Getenv = func(string) string { return "" }

	if err := runInitCommand(opts); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}
	msg := stderr.String()
	if !strings.Contains(msg, "owned by root") {
		t.Errorf("no ownership warning:\n%s", msg)
	}
	if !strings.Contains(msg, "chown -R") {
		t.Errorf("the warning does not print the command to run:\n%s", msg)
	}
}

func TestInitCommand_FolderIsFile(t *testing.T) {
	requireGit(t)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := runInitCommand(initOpts(filePath, &stdout, &stderr))
	if err == nil {
		t.Fatal("expected error when path is a file")
	}
	if !strings.Contains(err.Error(), "is a file, not a folder") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitCommand_FolderNotEmpty(t *testing.T) {
	requireGit(t)
	projectPath := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "existing.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := runInitCommand(initOpts(projectPath, &stdout, &stderr))
	if err == nil {
		t.Fatal("expected error when folder is not empty")
	}
	if !strings.Contains(err.Error(), "is not empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitCommand_FolderEmptyOK(t *testing.T) {
	requireGit(t)
	projectPath := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(projectPath, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed on empty folder: %v", err)
	}
	assertFileExists(t, filepath.Join(projectPath, "current", "basil.yaml"))
}

func TestInitCommand_RelativePath(t *testing.T) {
	requireGit(t)
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts("./myproject", &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed with relative path: %v", err)
	}
	assertFileExists(t, filepath.Join(tmpDir, "myproject", "current", "basil.yaml"))
}

func TestInitCommand_LocalHostGetsADevPort(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "local")
	var stdout, stderr bytes.Buffer
	opts := initOpts(root, &stdout, &stderr)
	opts.Host = "localhost"

	if err := runInitCommand(opts); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}
	yaml := readFile(t, filepath.Join(root, "current", "basil.yaml"))
	if !strings.Contains(yaml, "port: 8080") {
		t.Errorf("a localhost site should get a dev port:\n%s", yaml)
	}
	if !strings.Contains(stdout.String(), "--dev") {
		t.Error("a localhost site should be started with --dev")
	}
}

// Test helpers

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("file does not exist: %s (error: %v)", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("path is a directory, not a file: %s", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("directory does not exist: %s (error: %v)", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("path is a file, not a directory: %s", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}

// --- regression tests for the FEAT-152 review -----------------------------

// A site created by --init must be usable with no basil.yaml edit at all:
// the config must load, the Git server the printed clone command talks to
// must be on, and the durable place to write must be writable.
func TestInitCommand_GeneratedConfigNeedsNoEditing(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	path, err := config.ConfigPathForSite(root)
	if err != nil {
		t.Fatalf("ConfigPathForSite: %v", err)
	}
	cfg, err := config.Load(path, func(string) string { return "" })
	if err != nil {
		t.Fatalf("the generated config does not load: %v", err)
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("the generated config does not validate: %v", err)
	}

	// The summary prints `git clone https://…/.git`; that 404s unless the
	// endpoint is served. Nothing in the config can switch it off any more
	// (FEAT-157) — the endpoint follows the repository, and the operator's
	// switch lives inside it — so what the generated config must NOT do is
	// carry the retired keys.
	if warns := config.ReleaseWarnings(cfg); len(warns) != 0 {
		t.Errorf("the generated config carries settings loading has to ignore: %v", warns)
	}

	// Site code must be able to write to the durable location the same
	// release advertises as basil.uploads_dir.
	uploads := cfg.UploadsDir()
	if uploads == "" {
		t.Fatal("UploadsDir is empty")
	}
	var writable bool
	for _, dir := range cfg.WritePolicy() {
		if dir == uploads {
			writable = true
		}
	}
	if !writable {
		t.Errorf("the uploads directory is not writable by site code: %v", cfg.WritePolicy())
	}
}

// The data root holds the auth database, its SQLite sidecars and the
// certificate cache. On a shared host 0755 lets every local account read
// them.
func TestInitCommand_DataRootIsPrivate(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(root, "data"),
		filepath.Join(root, "data", "uploads"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			t.Errorf("%s is %04o: the data root must not be readable by other accounts", dir, mode)
		}
	}

	// Every file in the data root that holds credentials must be private too.
	for _, name := range []string{".basil-auth.db", ".basil-auth.db-wal", ".basil-auth.db-shm"} {
		path := filepath.Join(root, "data", name)
		info, err := os.Stat(path)
		if err != nil {
			continue // sidecars may be checkpointed away
		}
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			t.Errorf("%s is %04o, want 0600: it holds user rows and API key hashes", name, mode)
		}
	}
}

// A host that is not a hostname must be refused before anything is created:
// the API key is printed once, so a config that turns out to be unloadable
// costs the operator the credential and the whole tree.
func TestInitCommand_RejectsUnusableHost(t *testing.T) {
	requireGit(t)
	for _, host := range []string{
		"*.example.com", // * is a YAML alias indicator
		"mysite.example.com\nauth:\n  enabled: false", // config injection
		"my site.example.com",
		"-leading-hyphen.example.com",
	} {
		t.Run(host, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "mysite")
			var stdout, stderr bytes.Buffer
			opts := initOpts(root, &stdout, &stderr)
			opts.Host = host

			err := runInitCommand(opts)
			if err == nil {
				t.Fatalf("--host %q was accepted", host)
			}
			if _, statErr := os.Stat(root); !os.IsNotExist(statErr) {
				t.Errorf("a refused init created %s anyway", root)
			}
			if strings.Contains(stdout.String(), "API key") {
				t.Error("a refused init printed an API key")
			}
		})
	}
}

// A hostname that is fine must stay fine, and it must be quoted in the file.
func TestInitCommand_HostIsQuotedInConfig(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")
	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand failed: %v", err)
	}
	yaml := readFile(t, filepath.Join(root, "current", "basil.yaml"))
	if !strings.Contains(yaml, `host: "mysite.example.com"`) {
		t.Errorf("the host should be written as a quoted scalar:\n%s", yaml)
	}
}

// A failure part-way through must not leave a half-built tree: --init
// refuses to re-enter a non-empty directory, so the next attempt would fail
// with "folder is not empty", which points at the wrong problem.
func TestCleanupInitTree(t *testing.T) {
	t.Run("removes a root it created", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		mustMkdirAll(t, filepath.Join(root, "releases", "abc"))
		cleanupInitTree(root, true)
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Errorf("%s survived cleanup", root)
		}
	})

	t.Run("empties a root the operator created", func(t *testing.T) {
		root := t.TempDir()
		mustMkdirAll(t, filepath.Join(root, "releases", "abc"))
		mustMkdirAll(t, filepath.Join(root, "data"))
		mustMkdirAll(t, filepath.Join(root, "site.git"))
		cleanupInitTree(root, false)
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("cleanup left %d entries behind: %v", len(entries), entries)
		}
		if _, err := os.Stat(root); err != nil {
			t.Errorf("cleanup removed a directory it did not create: %v", err)
		}
	})
}

// The local layout makes the same promise with a different list of entries:
// a failed local init leaves nothing behind, and never eats a directory the
// operator made themselves.
func TestCleanupLocalInitTree(t *testing.T) {
	// The entries local init adds, in the order it adds them.
	seed := func(t *testing.T, root string) {
		t.Helper()
		mustMkdirAll(t, filepath.Join(root, "site"))
		mustMkdirAll(t, filepath.Join(root, "public"))
		mustMkdirAll(t, filepath.Join(root, ".githooks"))
		mustMkdirAll(t, filepath.Join(root, ".git"))
		for _, name := range []string{"basil.yaml", ".gitignore"} {
			if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("removes a root it created", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		seed(t, root)
		cleanupLocalInitTree(root, true)
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Errorf("%s survived cleanup", root)
		}
	})

	t.Run("empties a root the operator created", func(t *testing.T) {
		root := t.TempDir()
		seed(t, root)
		cleanupLocalInitTree(root, false)
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("cleanup left %d entries behind: %v", len(entries), entries)
		}
		if _, err := os.Stat(root); err != nil {
			t.Errorf("cleanup removed a directory it did not create: %v", err)
		}
	})
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
}

// Running as root, the target must not already exist, and its parent must
// not be writable by other accounts: --init shells out to git several times
// between the emptiness check and the last write, and every write it makes
// follows symlinks.
func TestInitCommand_RootRefusesATamperableTarget(t *testing.T) {
	requireGit(t)

	t.Run("pre-created target", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		mustMkdirAll(t, root) // empty, so the non-root path would accept it

		var stdout, stderr bytes.Buffer
		opts := initOpts(root, &stdout, &stderr)
		opts.Geteuid = func() int { return 0 }

		err := runInitCommand(opts)
		if err == nil {
			t.Fatal("root accepted a directory another account could have prepared")
		}
		if !strings.Contains(err.Error(), "must not exist yet") {
			t.Errorf("the error does not name the problem: %v", err)
		}
	})

	t.Run("world-writable parent", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "shared")
		mustMkdirAll(t, parent)
		if err := os.Chmod(parent, 0o777); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(parent, "mysite")

		var stdout, stderr bytes.Buffer
		opts := initOpts(root, &stdout, &stderr)
		opts.Geteuid = func() int { return 0 }

		err := runInitCommand(opts)
		if err == nil {
			t.Fatal("root created a site under a world-writable parent")
		}
		if !strings.Contains(err.Error(), "writable by other accounts") {
			t.Errorf("the error does not name the problem: %v", err)
		}
	})

	t.Run("an ordinary user is unaffected", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "shared")
		mustMkdirAll(t, parent)
		if err := os.Chmod(parent, 0o777); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(parent, "mysite")
		mustMkdirAll(t, root)

		var stdout, stderr bytes.Buffer
		if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
			t.Fatalf("the non-root path should be unchanged: %v", err)
		}
	})
}

// --init IS the first deploy, so release 1 must be in the deploy record.
// The observable consequence of omitting it: the FIRST `basil rollback`
// after the first real deploy refuses with "nothing to roll back to" even
// though the starter release is on disk.
func TestInitCommand_RecordsRelease1AndFirstRollbackWorks(t *testing.T) {
	f := newDeployFixture(t)
	starter := f.currentSHA(t)

	// The record already holds release 1, written by --init.
	entries := f.recordEntries(t)
	if len(entries) != 1 {
		t.Fatalf("record has %d entries after --init, want 1", len(entries))
	}
	e := entries[0]
	if e.CommitSHA != starter || e.Outcome != deploy.OutcomeDeployed {
		t.Errorf("release 1 entry = %s/%s, want %s/%s", e.CommitSHA, e.Outcome, starter, deploy.OutcomeDeployed)
	}
	if e.Trigger != deploy.TriggerInit || e.Publisher != "init" {
		t.Errorf("identity: trigger=%q publisher=%q, want init/init", e.Trigger, e.Publisher)
	}
	if e.AuthorName != "Basil" {
		t.Errorf("author = %q, want the starter commit's author", e.AuthorName)
	}

	// Deploy a real release, then roll back: rollback must find release 1.
	sha2 := f.commitAndPush(t, "site/index.pars", "<h1>\"v2\"</h1>\n", "v2")
	var out bytes.Buffer
	if err := runDeployCommand([]string{"--site", f.root, sha2}, &out, &out, emptyEnv); err != nil {
		t.Fatalf("deploy v2: %v\n%s", err, out.String())
	}
	var stdout, stderr bytes.Buffer
	if err := runRollbackCommand([]string{"--site", f.root}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("the first rollback must find --init's release 1: %v\nstderr: %s", err, stderr.String())
	}
	if got := f.currentSHA(t); got != starter {
		t.Errorf("current points at %s, want the starter release %s", got, starter)
	}
}

// --init installs the receive hooks, so a new site deploys on push from the
// first day, with the running binary's path baked in.
func TestInitCommand_InstallsReceiveHooks(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(initOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("runInitCommand: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pre-receive", "post-receive"} {
		path := filepath.Join(root, config.BareRepoName, "hooks", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s hook missing: %v", name, err)
		}
		content := string(data)
		if !strings.Contains(content, "deploy --from-hook="+name) {
			t.Errorf("%s hook does not invoke basil deploy --from-hook:\n%s", name, content)
		}
		if !strings.Contains(content, exe) {
			t.Errorf("%s hook does not bake in the binary path %s:\n%s", name, exe, content)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s hook is not executable (mode %v)", name, info.Mode())
		}
	}
}

// --- local mode (FEAT-156) ------------------------------------------------

// localInitOpts builds the default (local) options: no --server, no --host,
// no --admin, which is the whole point of the mode.
func localInitOpts(path string, out, errOut *bytes.Buffer) initOptions {
	return initOptions{
		Folder:  path,
		Stdout:  out,
		Stderr:  errOut,
		Geteuid: func() int { return 1000 },
		Getenv:  func(string) string { return "" },
	}
}

// The default mode is a plain folder a hobbyist can read at a glance: it must
// carry none of the server topology, and it must run without an edit.
func TestLocalInit_Success(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(localInitOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("local init failed: %v", err)
	}

	assertFileExists(t, filepath.Join(root, "basil.yaml"))
	assertFileExists(t, filepath.Join(root, "site", "index.pars"))
	assertFileExists(t, filepath.Join(root, "public", ".keep"))
	assertFileExists(t, filepath.Join(root, ".gitignore"))

	// Nothing from the server topology.
	for _, name := range []string{"site.git", "releases", "current", "data", ".basil-auth.db"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("local init created the server-mode entry %q", name)
		}
	}
	if config.IsSiteRoot(root) {
		t.Error("a local folder must not be a site root")
	}

	// The config loads and validates, as init proved before reporting success.
	cfg, err := config.Load(filepath.Join(root, "basil.yaml"), func(string) string { return "" })
	if err != nil {
		t.Fatalf("the generated config does not load: %v", err)
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("the generated config does not validate: %v", err)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("server.host = %q, want the localhost default", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("server.port = %d, want 8080", cfg.Server.Port)
	}

	yaml := readFile(t, filepath.Join(root, "basil.yaml"))
	for _, block := range []string{"auth:", "git:", "https:"} {
		if strings.Contains(yaml, block) {
			t.Errorf("the local config carries a %q block:\n%s", block, yaml)
		}
	}
	// The layering discipline is discoverable from day one.
	if !strings.Contains(yaml, "# developers:") || !strings.Contains(yaml, "-as sam") {
		t.Errorf("no commented developers example naming -as <name>:\n%s", yaml)
	}
	if !strings.Contains(yaml, "# data_dir:") || !strings.Contains(yaml, "# database:") {
		t.Errorf("the commented data_dir/database stanzas are missing:\n%s", yaml)
	}

	// The .gitignore guards the future push path.
	ignore := readFile(t, filepath.Join(root, ".gitignore"))
	for _, pattern := range []string{
		".basil-auth.db*", "dev_logs.db*", "*.db", "*.db-wal", "*.db-shm",
		"certs/", "cache/", "search/", "uploads/", ".DS_Store", "*.swp",
		// Credentials and logs: in the legacy layout a manual TLS
		// certificate (https.cert/key) and a file log sink both resolve to
		// the project directory, so both land next to the code.
		"*.pem", "*.key", "*.crt", "*.log", ".env",
	} {
		if !strings.Contains(ignore, pattern) {
			t.Errorf(".gitignore does not cover %q:\n%s", pattern, ignore)
		}
	}

	// The summary belongs to the mode.
	out := stdout.String()
	if strings.Contains(out, "API key") {
		t.Errorf("local init printed an API key:\n%s", out)
	}
	if strings.Contains(out, "git clone") {
		t.Errorf("local init printed a clone command:\n%s", out)
	}
	if !strings.Contains(out, "cd mysite && basil --dev") {
		t.Errorf("the next step is not printed:\n%s", out)
	}
	if !strings.Contains(out, "--server") || !strings.Contains(out, "deployment guide") {
		t.Errorf("no graduation pointer:\n%s", out)
	}
	// One physical line: it is a signpost, and a wrapped one reads as a
	// paragraph the reader is meant to study.
	var pointer string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "deployment guide") {
			pointer = line
		}
	}
	if !strings.Contains(pointer, "--server") {
		t.Errorf("the graduation pointer is split across lines:\n%s", out)
	}
}

// An explicit --host is accepted locally and validated with the same rules.
func TestLocalInit_ExplicitHost(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		var stdout, stderr bytes.Buffer
		opts := localInitOpts(root, &stdout, &stderr)
		opts.Host = "mysite.local"
		if err := runInitCommand(opts); err != nil {
			t.Fatalf("local init with --host failed: %v", err)
		}
		if yaml := readFile(t, filepath.Join(root, "basil.yaml")); !strings.Contains(yaml, `host: "mysite.local"`) {
			t.Errorf("the --host value was not written:\n%s", yaml)
		}
	})

	// Local mode is one shape (FEAT-156 open question 3): --host writes
	// server.host and switches nothing else, so a public hostname still gets
	// the dev port and no https block. The server's own template is the one
	// that reads IsLocalHost.
	t.Run("a public hostname does not switch mode", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		var stdout, stderr bytes.Buffer
		opts := localInitOpts(root, &stdout, &stderr)
		opts.Host = "mysite.example.com"
		if err := runInitCommand(opts); err != nil {
			t.Fatalf("local init with a public --host failed: %v", err)
		}
		yaml := readFile(t, filepath.Join(root, "basil.yaml"))
		if !strings.Contains(yaml, `host: "mysite.example.com"`) {
			t.Errorf("the --host value was not written:\n%s", yaml)
		}
		if !strings.Contains(yaml, "port: 8080") {
			t.Errorf("a public --host must not switch the local template to port 443:\n%s", yaml)
		}
		if strings.Contains(yaml, "https:") {
			t.Errorf("a public --host must not add an https block locally:\n%s", yaml)
		}
		if strings.Contains(stdout.String(), "API key") {
			t.Errorf("a public --host must not build server topology:\n%s", stdout.String())
		}
	})

	t.Run("validated", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		var stdout, stderr bytes.Buffer
		opts := localInitOpts(root, &stdout, &stderr)
		opts.Host = "*.example.com"
		if err := runInitCommand(opts); err == nil {
			t.Fatal("an unusable --host was accepted")
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Error("a refused init created the folder anyway")
		}
	})
}

// The folder is clone-shaped from day one, so graduating later is two
// commands rather than a restructure.
func TestLocalInit_CreatesGitRepository(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(localInitOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("local init failed: %v", err)
	}

	// A normal repository, not bare: the work tree is the folder itself.
	assertDirExists(t, filepath.Join(root, ".git"))

	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}
	if branch := git("rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
	if count := git("rev-list", "--count", "HEAD"); count != "1" {
		t.Errorf("commit count = %s, want 1", count)
	}
	if hooks := git("config", "core.hooksPath"); hooks != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", hooks)
	}
	if status := git("status", "--porcelain"); status != "" {
		t.Errorf("the initial commit left the tree dirty:\n%s", status)
	}

	hook := filepath.Join(root, ".githooks", "pre-commit")
	info, err := os.Stat(hook)
	if err != nil {
		t.Fatalf("pre-commit hook missing: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("pre-commit hook is not executable (mode %v)", info.Mode())
	}
	if !strings.Contains(stdout.String(), "git repository") {
		t.Errorf("the summary does not mention the repository:\n%s", stdout.String())
	}
}

// --no-git opts out even where git is available, and takes the hook with it:
// an inert .githooks/ in a folder with no repository is a puzzle.
func TestLocalInit_NoGit(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	opts := localInitOpts(root, &stdout, &stderr)
	opts.NoGit = true
	if err := runInitCommand(opts); err != nil {
		t.Fatalf("local init --no-git failed: %v", err)
	}

	for _, name := range []string{".git", ".githooks"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("--no-git created %s", name)
		}
	}
	assertFileExists(t, filepath.Join(root, "basil.yaml"))
	// Local init writes to stdout only, so that is where a stray warning
	// would land: the summary must not mention a repository it did not make.
	if strings.Contains(stdout.String(), "git repository") {
		t.Errorf("--no-git reported a repository:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "warning") {
		t.Errorf("--no-git warned about git:\n%s", stdout.String())
	}
}

// Git is a nicety, not a gate: with no git on PATH the folder is still
// created, and nothing is said about it.
func TestLocalInit_WithoutGitOnPath(t *testing.T) {
	t.Setenv("PATH", "")
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	if err := runInitCommand(localInitOpts(root, &stdout, &stderr)); err != nil {
		t.Fatalf("local init must not need git: %v", err)
	}
	assertFileExists(t, filepath.Join(root, "basil.yaml"))
	for _, name := range []string{".git", ".githooks"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("%s was created without git", name)
		}
	}
	// Nothing is said about it. Local init writes to stdout only, so that is
	// the stream a warning or an apology would appear on.
	out := stdout.String()
	if strings.Contains(out, "git repository") || strings.Contains(out, "warning") {
		t.Errorf("a missing git was remarked upon:\n%s", out)
	}
}

func TestLocalInit_Refusals(t *testing.T) {
	t.Run("--admin names --server", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "mysite")
		var stdout, stderr bytes.Buffer
		opts := localInitOpts(root, &stdout, &stderr)
		opts.Admin = "sam"

		err := runInitCommand(opts)
		if err == nil {
			t.Fatal("--admin was accepted without --server")
		}
		if !strings.Contains(err.Error(), "--server") {
			t.Errorf("the error does not name the fix: %v", err)
		}
		if _, statErr := os.Stat(root); !os.IsNotExist(statErr) {
			t.Error("a refused init created the folder anyway")
		}
	})

	t.Run("non-empty target", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "existing")
		mustMkdirAll(t, root)
		if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hi"), 0644); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer

		err := runInitCommand(localInitOpts(root, &stdout, &stderr))
		if err == nil {
			t.Fatal("local init wrote into a non-empty folder")
		}
		if !strings.Contains(err.Error(), "is not empty") {
			t.Errorf("unexpected error message: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(root, "notes.txt")); statErr != nil {
			t.Error("the refusal removed the operator's own file")
		}
	})
}

// The flag combinations are refused at the CLI, where the user typed them.
func TestCLI_InitFlagCombinations(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"--server without --init", []string{"--server"}, "--init"},
		{"--no-git without --init", []string{"--no-git"}, "--init"},
		{"--no-git with --server", []string{"--init", "x", "--server", "--no-git"}, "--no-git"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(context.Background(), tt.args, &stdout, &stderr, func(string) string { return "" })
			if err == nil {
				t.Fatalf("%v was accepted", tt.args)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("the error does not name %q: %v", tt.want, err)
			}
		})
	}
}

// --server is still the whole FEAT-152 topology, and it is what the flag
// selects: this is the one assertion that the split did not move it.
func TestCLI_ServerInitBuildsTheTopology(t *testing.T) {
	requireGit(t)
	root := filepath.Join(t.TempDir(), "mysite")

	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{
		"--init", root, "--server",
		"--host", "mysite.example.com",
		"--admin", "sam",
	}, &stdout, &stderr, os.Getenv)
	if err != nil {
		t.Fatalf("server init failed: %v", err)
	}

	assertDirExists(t, filepath.Join(root, "site.git"))
	assertDirExists(t, filepath.Join(root, "releases"))
	assertDirExists(t, filepath.Join(root, "data"))
	assertFileExists(t, filepath.Join(root, "current", "basil.yaml"))
	if !strings.Contains(stdout.String(), "API key: bsl_") {
		t.Errorf("no API key printed:\n%s", stdout.String())
	}
}
