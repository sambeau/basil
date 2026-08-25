package deploy

import (
	"path/filepath"
	"testing"
)

const (
	unfMessy     = "let    x    =    5\n"
	unfFormatted = "let x = 5\n"
	unfBroken    = "let x = = 2\n"
)

// Unformatted names only the .pars files whose formatting differs, using
// release-relative paths, and ignores non-.pars files.
func TestUnformattedFindsOnlyDifferingFiles(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":       validBasilYAML,
		"site/index.pars":  unfFormatted,
		"site/messy.pars":  unfMessy,
		"deploy.pars":      unfMessy,
		"site/notes.txt":   unfMessy, // not .pars: ignored even though "unformatted"
		"site/nested/x.md": unfMessy,
	})

	got := Unformatted(dir)
	want := map[string]bool{
		filepath.Join("site", "messy.pars"): true,
		"deploy.pars":                       true,
	}
	if len(got) != len(want) {
		t.Fatalf("Unformatted = %v, want the two unformatted .pars files", got)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("Unformatted reported unexpected file %q (want only %v)", f, want)
		}
	}
}

// A fully formatted release warns about nothing.
func TestUnformattedCleanReleaseIsEmpty(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":      validBasilYAML,
		"site/index.pars": unfFormatted,
		"deploy.pars":     "1 + 1\n",
	})
	if got := Unformatted(dir); got != nil {
		t.Errorf("Unformatted on a clean release = %v, want nil", got)
	}
}

// A file that will not parse is NOT a formatting warning: IsFormatted errors on
// it, and Unformatted skips it silently rather than warning or failing. (In the
// real flow Validate already rejected such a release; this is the guard.)
func TestUnformattedSkipsParseErrors(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":       validBasilYAML,
		"site/broken.pars": unfBroken,
		"site/messy.pars":  unfMessy,
	})
	got := Unformatted(dir)
	if len(got) != 1 || got[0] != filepath.Join("site", "messy.pars") {
		t.Errorf("Unformatted = %v, want only site/messy.pars (the broken file skipped)", got)
	}
}
