package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/format"
)

const (
	fmtMessy     = "let    x    =    5\n"
	fmtFormatted = "let x = 5\n"
	fmtBroken    = "let x = = 2\n"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestFmt_WriteTreeInPlace formats a directory tree in place, ignoring non-.pars
// files and never descending into .git.
func TestFmt_WriteTreeInPlace(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pars")
	b := filepath.Join(dir, "sub", "b.pars")
	txt := filepath.Join(dir, "sub", "c.txt")
	git := filepath.Join(dir, ".git", "hooks", "x.pars")
	writeFile(t, a, fmtMessy)
	writeFile(t, b, fmtMessy)
	writeFile(t, txt, fmtMessy)
	writeFile(t, git, fmtMessy)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand([]string{"-w", dir}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	want, _ := format.FormatSource(a, fmtMessy)
	if got := readFile(t, a); got != want {
		t.Errorf("a.pars not formatted: got %q want %q", got, want)
	}
	if got := readFile(t, b); got != want {
		t.Errorf("sub/b.pars not formatted: got %q want %q", got, want)
	}
	if got := readFile(t, txt); got != fmtMessy {
		t.Errorf("non-.pars c.txt was modified: %q", got)
	}
	if got := readFile(t, git); got != fmtMessy {
		t.Errorf(".git file was modified (walk descended into .git): %q", got)
	}
}

// TestFmt_ListExitsNonZero lists only unformatted files and returns a non-zero
// exit so it is usable as a CI gate.
func TestFmt_ListExitsNonZero(t *testing.T) {
	dir := t.TempDir()
	messy := filepath.Join(dir, "messy.pars")
	clean := filepath.Join(dir, "clean.pars")
	writeFile(t, messy, fmtMessy)
	writeFile(t, clean, fmtFormatted)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand([]string{"-l", dir}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("expected non-zero exit for unformatted files, got nil error")
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	out := stdout.String()
	if !strings.Contains(out, messy) {
		t.Errorf("expected %s listed, got: %q", messy, out)
	}
	if strings.Contains(out, clean) {
		t.Errorf("formatted file should not be listed, got: %q", out)
	}
	// The unformatted file must not be modified by -l.
	if got := readFile(t, messy); got != fmtMessy {
		t.Errorf("-l modified the file: %q", got)
	}
}

// TestFmt_ListCleanTree is a clean no-op with no output and a zero exit.
func TestFmt_ListCleanTree(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.pars"), fmtFormatted)
	writeFile(t, filepath.Join(dir, "b.pars"), fmtFormatted)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand([]string{"-l", dir}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("expected clean no-op, got error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected no output for a formatted tree, got: %q", stdout.String())
	}
}

// TestFmt_Diff shows a diff without modifying the file.
func TestFmt_Diff(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.pars")
	writeFile(t, f, fmtMessy)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand([]string{"-d", f}, &stdout, &stderr, emptyEnv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.HasPrefix(out, "diff "+f+"\n") {
		t.Errorf("expected diff header for %s, got: %q", f, out)
	}
	if !strings.Contains(out, "-1:") || !strings.Contains(out, "+1:") {
		t.Errorf("expected -/+ diff lines, got: %q", out)
	}
	if got := readFile(t, f); got != fmtMessy {
		t.Errorf("-d modified the file: %q", got)
	}
}

// TestFmt_ParseErrorReported reports file:line and never mangles the file.
func TestFmt_ParseErrorReported(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "broken.pars")
	writeFile(t, f, fmtBroken)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand([]string{"-w", f}, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("expected a parse error, got nil")
	}
	if code := exitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), f+":1:") {
		t.Errorf("expected file:line diagnostic (%s:1:...), got: %q", f, stderr.String())
	}
	if got := readFile(t, f); got != fmtBroken {
		t.Errorf("a parse error must not modify the file, got: %q", got)
	}
}

// TestFmt_WriteAlreadyFormatted is a byte-for-byte no-op on a clean tree.
func TestFmt_WriteAlreadyFormatted(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.pars")
	writeFile(t, f, fmtFormatted)

	before, err := os.Stat(f)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := runFmtCommand([]string{"-w", dir}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readFile(t, f); got != fmtFormatted {
		t.Errorf("formatted file changed: %q", got)
	}
	after, _ := os.Stat(f)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Errorf("clean file was rewritten (modtime changed)")
	}
}

// TestFmt_SingleFileStdout prints formatted output for a lone file argument and
// leaves the file untouched (pars fmt parity).
func TestFmt_SingleFileStdout(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.pars")
	writeFile(t, f, fmtMessy)

	var stdout, stderr bytes.Buffer
	if err := runFmtCommand([]string{f}, &stdout, &stderr, emptyEnv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.String() != fmtFormatted {
		t.Errorf("stdout = %q, want %q", stdout.String(), fmtFormatted)
	}
	if got := readFile(t, f); got != fmtMessy {
		t.Errorf("default mode modified the file: %q", got)
	}
}

// TestFmt_NoArgsWalksCwd confirms a bare `basil fmt` walks the current
// directory and defaults to list mode (non-zero exit when unformatted).
func TestFmt_NoArgsWalksCwd(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.pars")
	writeFile(t, f, fmtMessy)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	err := runFmtCommand(nil, &stdout, &stderr, emptyEnv)
	if err == nil {
		t.Fatal("expected non-zero exit for unformatted cwd tree")
	}
	if !strings.Contains(stdout.String(), "a.pars") {
		t.Errorf("expected a.pars listed from cwd walk, got: %q", stdout.String())
	}
}
