package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
