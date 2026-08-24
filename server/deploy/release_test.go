package deploy

import (
	"archive/tar"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// requireGit skips a test on a machine with no git; Materialise shells out
// to `git archive`.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// fixtureRepo builds a real repository with one commit containing a nested
// file, an executable and a symlink, and returns the repo dir and the
// commit's full SHA.
func fixtureRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	runTestGit(t, repo, "init", "--initial-branch=live")
	runTestGit(t, repo, "config", "user.name", "Test Author")
	runTestGit(t, repo, "config", "user.email", "author@example.com")

	if err := os.MkdirAll(filepath.Join(repo, "site"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"basil.yaml":      "server:\n  host: example.com\n",
		"site/index.pars": "<h1>\"hello\"</h1>\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "deploy.pars"), []byte("log(\"hi\")\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("site/index.pars", filepath.Join(repo, "link.pars")); err != nil {
		t.Fatal(err)
	}

	runTestGit(t, repo, "add", "-A")
	runTestGit(t, repo, "commit", "-m", "first")
	return repo, runTestGit(t, repo, "rev-parse", "HEAD")
}

func TestMaterialiseExtractsTheCommit(t *testing.T) {
	requireGit(t)
	repo, sha := fixtureRepo(t)
	releasesDir := filepath.Join(t.TempDir(), "releases")

	dir, err := Materialise(repo, sha, releasesDir)
	if err != nil {
		t.Fatalf("Materialise: %v", err)
	}
	if want := filepath.Join(releasesDir, sha); dir != want {
		t.Errorf("Materialise returned %q, want %q", dir, want)
	}

	got, err := os.ReadFile(filepath.Join(dir, "site", "index.pars"))
	if err != nil {
		t.Fatalf("nested file missing from release: %v", err)
	}
	if string(got) != "<h1>\"hello\"</h1>\n" {
		t.Errorf("file content differs from the commit: %q", got)
	}

	info, err := os.Stat(filepath.Join(dir, "deploy.pars"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Error("executable bit was not preserved")
	}

	target, err := os.Readlink(filepath.Join(dir, "link.pars"))
	if err != nil {
		t.Fatalf("symlink was not preserved: %v", err)
	}
	if target != "site/index.pars" {
		t.Errorf("symlink target = %q, want site/index.pars", target)
	}

	// The release is code, not a repository.
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Error(".git ended up inside the release")
	}
	assertNoTempDirs(t, releasesDir)
}

func TestMaterialiseUnknownSHALeavesNothingBehind(t *testing.T) {
	requireGit(t)
	repo, _ := fixtureRepo(t)
	releasesDir := filepath.Join(t.TempDir(), "releases")
	bogus := strings.Repeat("d", 40)

	_, err := Materialise(repo, bogus, releasesDir)
	if err == nil {
		t.Fatal("expected an error for an unknown sha")
	}
	if _, statErr := os.Stat(filepath.Join(releasesDir, bogus)); !os.IsNotExist(statErr) {
		t.Error("a release directory exists for a commit that does not")
	}
	assertNoTempDirs(t, releasesDir)
}

func TestMaterialiseIsIdempotent(t *testing.T) {
	requireGit(t)
	repo, sha := fixtureRepo(t)
	releasesDir := filepath.Join(t.TempDir(), "releases")

	first, err := Materialise(repo, sha, releasesDir)
	if err != nil {
		t.Fatalf("Materialise: %v", err)
	}
	// A marker that re-extraction would destroy.
	marker := filepath.Join(first, "marker")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := Materialise(repo, sha, releasesDir)
	if err != nil {
		t.Fatalf("second Materialise: %v", err)
	}
	if second != first {
		t.Errorf("second Materialise returned %q, want %q", second, first)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("an existing release was re-extracted instead of returned as-is")
	}
}

func TestExtractTarRejectsEscapingEntries(t *testing.T) {
	for name, entry := range map[string]string{
		"parent traversal": "../evil.txt",
		"nested traversal": "ok/../../evil.txt",
		"absolute path":    "/evil.txt",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "dest")
			if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			tw := tar.NewWriter(&buf)
			content := []byte("evil")
			if err := tw.WriteHeader(&tar.Header{Name: entry, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(content); err != nil {
				t.Fatal(err)
			}
			tw.Close()

			err := extractTar(&buf, dest)
			if err == nil {
				t.Fatal("an escaping entry was extracted")
			}
			if !strings.Contains(err.Error(), "escapes") {
				t.Errorf("the error does not name the problem: %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(root, "evil.txt")); !os.IsNotExist(statErr) {
				t.Error("the escaping file was written outside the destination")
			}
		})
	}
}

func TestExtractTarRejectsUnsupportedEntryTypes(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "fifo", Mode: 0o644, Typeflag: tar.TypeFifo}); err != nil {
		t.Fatal(err)
	}
	tw.Close()

	if err := extractTar(&buf, t.TempDir()); err == nil {
		t.Fatal("a fifo entry was accepted")
	}
}

// siteRootWithReleases builds a site root with the named (empty) release
// directories, no `current` yet.
func siteRootWithReleases(t *testing.T, shas ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, sha := range shas {
		if err := os.MkdirAll(filepath.Join(root, "releases", sha), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestSetCurrentCreatesAndSwapsTheLink(t *testing.T) {
	root := siteRootWithReleases(t, "aaa", "bbb")

	// First activation: no `current` exists yet (first deploy on a site
	// whose bootstrap failed part-way must still work).
	if err := SetCurrent(root, filepath.Join(root, "releases", "aaa")); err != nil {
		t.Fatalf("SetCurrent with no existing link: %v", err)
	}
	target, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatalf("readlink current: %v", err)
	}
	if want := filepath.Join("releases", "aaa"); target != want {
		t.Errorf("current -> %q, want the RELATIVE %q", target, want)
	}

	// Swap. The old release must survive: rollback depends on it.
	if err := SetCurrent(root, filepath.Join(root, "releases", "bbb")); err != nil {
		t.Fatalf("SetCurrent swap: %v", err)
	}
	got, err := CurrentRelease(root)
	if err != nil {
		t.Fatalf("CurrentRelease: %v", err)
	}
	if want := filepath.Join(root, "releases", "bbb"); got != want {
		t.Errorf("CurrentRelease = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "aaa")); err != nil {
		t.Error("the previous release vanished on activation")
	}
}

func TestSetCurrentRefusesAMissingReleaseAndLeavesTheLinkAlone(t *testing.T) {
	root := siteRootWithReleases(t, "aaa")
	if err := SetCurrent(root, filepath.Join(root, "releases", "aaa")); err != nil {
		t.Fatal(err)
	}

	err := SetCurrent(root, filepath.Join(root, "releases", "gone"))
	if err == nil {
		t.Fatal("activating a release that does not exist was accepted")
	}
	target, readErr := os.Readlink(filepath.Join(root, "current"))
	if readErr != nil || target != filepath.Join("releases", "aaa") {
		t.Errorf("current changed on a failed activation: %q, %v", target, readErr)
	}
	// No half-written temp links left behind.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "current.tmp-") {
			t.Errorf("stale temp link left behind: %s", e.Name())
		}
	}
}

func TestCurrentReleaseFailsWithoutALink(t *testing.T) {
	if _, err := CurrentRelease(t.TempDir()); err == nil {
		t.Fatal("expected an error for a site with no current link")
	}
}

func TestPruneKeepsTheNewestAndNeverTheActive(t *testing.T) {
	root := siteRootWithReleases(t, "r1", "r2", "r3", "r4", "r5")
	releasesDir := filepath.Join(root, "releases")

	// Stagger mtimes: r1 oldest ... r5 newest.
	base := time.Now().Add(-time.Hour)
	for i, name := range []string{"r1", "r2", "r3", "r4", "r5"} {
		ts := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(filepath.Join(releasesDir, name), ts, ts); err != nil {
			t.Fatal(err)
		}
	}
	// An in-flight extraction and a stray file must both survive pruning.
	if err := os.MkdirAll(filepath.Join(releasesDir, ".tmp-xyz"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releasesDir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// keep=2 with the OLDEST release active: r5, r4 are kept for being
	// newest, r1 for being active; r3 and r2 go.
	removed, err := Prune(releasesDir, 2, filepath.Join(releasesDir, "r1"))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("Prune removed %v, want exactly r3 and r2", removed)
	}
	for _, name := range []string{"r5", "r4", "r1", ".tmp-xyz", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(releasesDir, name)); err != nil {
			t.Errorf("%s should have survived pruning: %v", name, err)
		}
	}
	for _, name := range []string{"r2", "r3"} {
		if _, err := os.Stat(filepath.Join(releasesDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should have been pruned", name)
		}
	}
}

func TestPruneKeepZeroRemovesNothing(t *testing.T) {
	root := siteRootWithReleases(t, "r1", "r2", "r3")
	releasesDir := filepath.Join(root, "releases")

	// keep <= 0 is keep-everything, never delete-everything.
	for _, keep := range []int{0, -1} {
		removed, err := Prune(releasesDir, keep, filepath.Join(releasesDir, "r1"))
		if err != nil {
			t.Fatalf("Prune(keep=%d): %v", keep, err)
		}
		if len(removed) != 0 {
			t.Errorf("Prune(keep=%d) removed %v", keep, removed)
		}
	}
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("%d releases left, want 3", len(entries))
	}
}

func TestPruneMissingReleasesDirFails(t *testing.T) {
	if _, err := Prune(filepath.Join(t.TempDir(), "gone"), 2, ""); err == nil {
		t.Fatal("expected an error for a missing releases directory")
	}
}

func assertNoTempDirs(t *testing.T, releasesDir string) {
	t.Helper()
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("partial extraction left behind: %s", e.Name())
		}
	}
}
