package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func readHook(t *testing.T, repo, name string) (content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(repo, "hooks", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data), info.Mode()
}

func TestInstallHooks_FreshInstall(t *testing.T) {
	repo := t.TempDir()
	if err := installHooks(repo, "/opt/basil/bin/basil"); err != nil {
		t.Fatalf("installHooks: %v", err)
	}

	for _, name := range hookNames {
		content, mode := readHook(t, repo, name)
		if !strings.HasPrefix(content, "#!/bin/sh\n") {
			t.Errorf("%s does not start with a sh shebang:\n%s", name, content)
		}
		if !strings.Contains(content, hookMarker) {
			t.Errorf("%s is missing the Basil marker line", name)
		}
		if want := "exec '/opt/basil/bin/basil' deploy --from-hook=" + name; !strings.Contains(content, want) {
			t.Errorf("%s does not exec the baked-in binary: want %q in:\n%s", name, want, content)
		}
		if runtime.GOOS != "windows" && mode.Perm()&0o111 == 0 {
			t.Errorf("%s is not executable (mode %v)", name, mode)
		}
	}
}

func TestInstallHooks_ReinstallAfterDeletion(t *testing.T) {
	repo := t.TempDir()
	if err := installHooks(repo, "/usr/local/bin/basil"); err != nil {
		t.Fatalf("installHooks: %v", err)
	}
	if err := os.Remove(filepath.Join(repo, "hooks", "pre-receive")); err != nil {
		t.Fatal(err)
	}

	if err := installHooks(repo, "/usr/local/bin/basil"); err != nil {
		t.Fatalf("re-install after deletion: %v", err)
	}
	content, _ := readHook(t, repo, "pre-receive")
	if !strings.Contains(content, hookMarker) {
		t.Error("deleted hook was not re-installed")
	}
}

func TestInstallHooks_BinaryPathDriftRewritten(t *testing.T) {
	repo := t.TempDir()
	if err := installHooks(repo, "/old/place/basil"); err != nil {
		t.Fatalf("installHooks: %v", err)
	}
	if err := installHooks(repo, "/new/place/basil"); err != nil {
		t.Fatalf("re-install with a moved binary: %v", err)
	}

	for _, name := range hookNames {
		content, _ := readHook(t, repo, name)
		if strings.Contains(content, "/old/place/basil") {
			t.Errorf("%s still names the old binary path:\n%s", name, content)
		}
		if !strings.Contains(content, "'/new/place/basil'") {
			t.Errorf("%s was not rewritten to the new binary path:\n%s", name, content)
		}
	}
}

func TestInstallHooks_Idempotent(t *testing.T) {
	repo := t.TempDir()
	if err := installHooks(repo, "/usr/local/bin/basil"); err != nil {
		t.Fatalf("installHooks: %v", err)
	}
	before, _ := readHook(t, repo, "post-receive")
	if err := installHooks(repo, "/usr/local/bin/basil"); err != nil {
		t.Fatalf("second installHooks: %v", err)
	}
	after, _ := readHook(t, repo, "post-receive")
	if before != after {
		t.Error("an unchanged hook was modified by re-install")
	}
}

func TestInstallHooks_ForeignHookRefused(t *testing.T) {
	repo := t.TempDir()
	hooksDir := filepath.Join(repo, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := "#!/bin/sh\n# the operator's own hook\nexit 0\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-receive"), []byte(custom), 0o755); err != nil {
		t.Fatal(err)
	}

	err := installHooks(repo, "/usr/local/bin/basil")
	if err == nil {
		t.Fatal("installHooks overwrote a foreign hook without error")
	}
	if !errors.Is(err, ErrForeignHook) {
		t.Errorf("error is not ErrForeignHook: %v", err)
	}
	content, _ := readHook(t, repo, "pre-receive")
	if content != custom {
		t.Errorf("foreign hook was modified:\n%s", content)
	}
}

func TestHookScript_QuotesAwkwardPaths(t *testing.T) {
	script := hookScript("/opt/my tools/it's basil", "pre-receive")
	if want := `exec '/opt/my tools/it'\''s basil' deploy --from-hook=pre-receive`; !strings.Contains(script, want) {
		t.Errorf("path not shell-quoted: want %q in:\n%s", want, script)
	}
}
