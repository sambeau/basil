package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

// validBasilYAML passes config.Load and config.Validate: host set, and
// Defaults() supplies https.auto for production validation.
const validBasilYAML = "server:\n  host: example.com\nsite:\n  path: ./site\n"

func writeRelease(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestValidateCleanRelease(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":      validBasilYAML,
		"site/index.pars": "<h1>\"hello\"</h1>\n",
		"site/about.pars": "<p>\"about\"</p>\n",
		"deploy.pars":     "1 + 1\n",
	})
	if errs := Validate(dir); errs != nil {
		t.Errorf("Validate returned errors for a clean release: %v", errs)
	}
}

func TestValidateCollectsErrorsAcrossAllFiles(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":       validBasilYAML,
		"site/index.pars":  "<h1>\"hello\"</h1>\n",
		"site/broken.pars": "let x = = 2\n",
		"deploy.pars":      "let y = = 3\n",
	})

	errs := Validate(dir)
	if len(errs) < 2 {
		t.Fatalf("expected errors from both broken files, got %v", errs)
	}

	var files []string
	for _, e := range errs {
		files = append(files, e.File)
		if e.Line <= 0 {
			t.Errorf("%s: parse error has no line number: %+v", e.File, e)
		}
		if e.Message == "" {
			t.Errorf("%s: parse error has no message", e.File)
		}
	}
	joined := strings.Join(files, " ")
	if !strings.Contains(joined, filepath.Join("site", "broken.pars")) {
		t.Errorf("site/broken.pars not reported: %v", errs)
	}
	if !strings.Contains(joined, "deploy.pars") {
		t.Errorf("deploy.pars not reported: %v", errs)
	}
	// Files that parse contribute nothing.
	if strings.Contains(joined, "index.pars") {
		t.Errorf("the valid index.pars was reported as an error: %v", errs)
	}
}

func TestValidateMissingConfig(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"site/index.pars": "<h1>\"hello\"</h1>\n",
	})
	errs := Validate(dir)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error for the missing config, got %v", errs)
	}
	if errs[0].File != "basil.yaml" {
		t.Errorf("error attributed to %q, want basil.yaml", errs[0].File)
	}
}

func TestValidateBrokenConfig(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":      "server:\n  host: example.com\nlogging:\n  level: banana\nsite:\n  path: ./site\n",
		"site/index.pars": "<h1>\"hello\"</h1>\n",
	})
	errs := Validate(dir)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one config error, got %v", errs)
	}
	if errs[0].File != "basil.yaml" || !strings.Contains(errs[0].Message, "log level") {
		t.Errorf("unexpected config error: %+v", errs[0])
	}
}

// Broken code AND a broken config are all reported in one pass: the
// developer gets the whole bill, not an instalment.
func TestValidateReportsCodeAndConfigTogether(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":       "logging:\n  level: banana\n",
		"site/broken.pars": "let x = = 2\n",
	})
	errs := Validate(dir)
	if len(errs) != 2 {
		t.Fatalf("expected one parse error and one config error, got %v", errs)
	}
}

// A dangling .pars symlink would be a 500 at request time, so validation
// refuses it now.
func TestValidateDanglingParsSymlink(t *testing.T) {
	dir := writeRelease(t, map[string]string{
		"basil.yaml":      validBasilYAML,
		"site/index.pars": "<h1>\"hello\"</h1>\n",
	})
	if err := os.Symlink("missing.pars", filepath.Join(dir, "site", "dangling.pars")); err != nil {
		t.Fatal(err)
	}
	errs := Validate(dir)
	if len(errs) != 1 {
		t.Fatalf("expected one error for the dangling symlink, got %v", errs)
	}
	if want := filepath.Join("site", "dangling.pars"); errs[0].File != want {
		t.Errorf("error attributed to %q, want %q", errs[0].File, want)
	}
}

func TestValidationErrorString(t *testing.T) {
	e := ValidationError{File: "site/x.pars", Line: 3, Column: 7, Message: "unexpected '='"}
	if got, want := e.String(), "site/x.pars:3:7: unexpected '='"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	cfg := ValidationError{File: "basil.yaml", Message: "not loadable"}
	if got, want := cfg.String(), "basil.yaml: not loadable"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// A release root that cannot be walked at all means NOTHING was validated —
// that must surface as a real error labelled with the release directory,
// not disappear or masquerade as a per-file problem.
func TestValidateUnwalkableReleaseRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "no-such-release")
	errs := Validate(dir)
	found := false
	for _, e := range errs {
		if e.File == dir && strings.Contains(e.Message, "walking the release") {
			found = true
		}
	}
	if !found {
		t.Errorf("Validate on an unwalkable root returned %v, want a walking-the-release error labelled %q", errs, dir)
	}
}

// siteRootRelease writes a release inside a real site-root layout —
// releases/<id>, a `current` link and a site.git — so ResolveAnchors finds a
// site root and the served-roots check has a repository to protect.
func siteRootRelease(t *testing.T, yaml string) string {
	t.Helper()
	root := t.TempDir()
	release := filepath.Join(root, config.ReleasesDirName, "abc1234")
	if err := os.MkdirAll(filepath.Join(release, "site"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, config.BareRepoName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, "site", "index.pars"), []byte("<h1>\"hello\"</h1>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(release, config.ConfigFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(config.ReleasesDirName, "abc1234"), filepath.Join(root, config.CurrentLinkName)); err != nil {
		t.Skipf("symlinks are not available here: %v", err)
	}
	return release
}

// A release may choose its own served roots, and one of them pointed at the
// site root would publish site.git — the whole history, unpublished branches
// included — as unauthenticated static files. The gate refuses the push, so
// the developer reads it in their terminal and no release directory is ever
// activated.
func TestValidateRejectsServedRootOverTheRepository(t *testing.T) {
	release := siteRootRelease(t, validBasilYAML+"static:\n  - path: /s/\n    root: ../..\n")

	errs := Validate(release)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %v", errs)
	}
	if errs[0].File != config.ConfigFileName {
		t.Errorf("error file = %q, want %q", errs[0].File, config.ConfigFileName)
	}
	if !strings.Contains(errs[0].Message, config.BareRepoName) {
		t.Errorf("error should name the repository, got: %s", errs[0].Message)
	}
	if !strings.Contains(errs[0].Message, "static[0].root") {
		t.Errorf("error should name the served root that reaches it, got: %s", errs[0].Message)
	}
}

// The other half of the same property: a static root that stays inside the
// release is ordinary configuration and must go straight through.
func TestValidateAcceptsStaticRootInsideTheRelease(t *testing.T) {
	release := siteRootRelease(t, validBasilYAML+"static:\n  - path: /s/\n    root: ./public\n")
	if err := os.MkdirAll(filepath.Join(release, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if errs := Validate(release); errs != nil {
		t.Errorf("Validate rejected a release serving its own public directory: %v", errs)
	}
}
