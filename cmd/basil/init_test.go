package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

// requireGit skips a test on a machine with no git. `basil --init` needs it:
// it creates the repository and deploys the first release from it.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func initOpts(path string, out, errOut *bytes.Buffer) initOptions {
	return initOptions{
		Folder:  path,
		Host:    "mysite.example.com",
		Admin:   "sam",
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
