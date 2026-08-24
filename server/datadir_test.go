package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
)

// siteRootFixture builds a config with the two anchors genuinely separate,
// the way the site-root layout has them.
func siteRootFixture(t *testing.T) (cfg *config.Config, releaseDir, dataDir string) {
	t.Helper()
	root := t.TempDir()
	releaseDir = filepath.Join(root, "releases", "r1")
	dataDir = filepath.Join(root, "data")
	must(os.MkdirAll(filepath.Join(releaseDir, "site"), 0755))
	must(os.MkdirAll(filepath.Join(dataDir, "uploads"), 0755))

	cfg = config.Defaults()
	cfg.SiteRoot = root
	cfg.ReleaseDir = releaseDir
	cfg.DataDir = dataDir
	cfg.Server.Dev = true
	cfg.Site.Path = filepath.Join(releaseDir, "site")
	cfg.PublicDir = filepath.Join(releaseDir, "public")
	config.ResolvePaths(cfg)
	return cfg, releaseDir, dataDir
}

// The uploads directory is durable and servable without living inside
// public_dir, following the /__p/ and /__img/ pattern.
func TestUploadsServedFromTheDataRoot(t *testing.T) {
	cfg, _, dataDir := siteRootFixture(t)
	must(os.WriteFile(filepath.Join(dataDir, "uploads", "note.txt"), []byte("durable"), 0644))
	must(os.MkdirAll(filepath.Join(dataDir, "uploads", "sub"), 0755))

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	tests := []struct {
		name string
		path string
		want int
		body string
	}{
		{"serves an uploaded file", "/__uploads/note.txt", http.StatusOK, "durable"},
		{"missing file is 404", "/__uploads/nope.txt", http.StatusNotFound, ""},
		{"no directory listing", "/__uploads/sub/", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, http.NoBody)
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d (body %q)", rec.Code, tt.want, rec.Body.String())
			}
			if tt.body != "" && !strings.Contains(rec.Body.String(), tt.body) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.body)
			}
		})
	}

	t.Run("no escape from the uploads directory", func(t *testing.T) {
		// The mux cleans ../ before the handler sees it; what matters is
		// that nothing outside uploads/ is ever served.
		must(os.WriteFile(filepath.Join(dataDir, "secret.txt"), []byte("private"), 0644))
		for _, p := range []string{"/__uploads/../secret.txt", "/__uploads/sub/../../secret.txt"} {
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, httptest.NewRequest("GET", p, http.NoBody))
			if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "private") {
				t.Errorf("%s served a file from outside uploads/", p)
			}
		}
	})
}

// A handler may write to the data root, and may not write into the release.
// A write inside the release would be destroyed by the next deploy.
func TestHandlerWritesLandOutsideTheRelease(t *testing.T) {
	cfg, releaseDir, dataDir := siteRootFixture(t)
	cfg.Security.AllowWrite = []string{filepath.Join(dataDir, "uploads")}

	siteDir := cfg.Site.Path
	must(os.WriteFile(filepath.Join(siteDir, "index.pars"), []byte(
		`"durable\n" ==> text(@`+filepath.Join(dataDir, "uploads", "written.txt")+`)
"wrote to the data root"
`), 0644))
	must(os.MkdirAll(filepath.Join(siteDir, "bad"), 0755))
	must(os.WriteFile(filepath.Join(siteDir, "bad", "index.pars"), []byte(
		`"doomed\n" ==> text(@`+filepath.Join(releaseDir, "site", "written.txt")+`)
"wrote into the release"
`), 0644))

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	handler := newSiteHandler(s, siteDir, s.scriptCache)

	// The permitted write succeeds, in the data root.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "uploads", "written.txt")); err != nil {
		t.Errorf("the permitted write did not happen: %v", err)
	}

	// The write into the release is refused, and nothing appears there.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/bad/", http.NoBody))
	if _, err := os.Stat(filepath.Join(releaseDir, "site", "written.txt")); err == nil {
		t.Error("a handler wrote inside the release; the next deploy would destroy it")
	}

	// And no stray file anywhere else in the release either.
	must(filepath.Walk(releaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "written.txt") {
			t.Errorf("runtime write inside the release: %s", path)
		}
		return nil
	}))
}

// hostPolicy must refuse issuance when no hostname is configured. Returning
// nil there lets a stranger burn the site's Let's Encrypt rate limit by
// putting any hostname in SNI.
func TestHostPolicyRefusesAnEmptyHost(t *testing.T) {
	cfg := config.Defaults()
	cfg.ReleaseDir = t.TempDir()
	cfg.DataDir = cfg.ReleaseDir
	s := &Server{config: cfg}

	policy := s.hostPolicy()
	if policy == nil {
		t.Fatal("hostPolicy returned nil, which allows any hostname in SNI")
	}
	if err := policy(t.Context(), "someone-elses.example.com"); err == nil {
		t.Error("expected the request to be refused")
	}

	cfg.Server.Host = "mysite.example.com"
	policy = s.hostPolicy()
	if err := policy(t.Context(), "mysite.example.com"); err != nil {
		t.Errorf("the configured host was refused: %v", err)
	}
	if err := policy(t.Context(), "someone-elses.example.com"); err == nil {
		t.Error("a hostname other than the configured one should be refused")
	}
}

// The certificate cache used to resolve against the process working
// directory, so where a site's certificates lived depended on where the
// operator was standing.
func TestCertCacheDirIsAnchoredToTheDataRoot(t *testing.T) {
	cfg, _, dataDir := siteRootFixture(t)
	s := &Server{config: cfg}

	elsewhere := t.TempDir()
	t.Chdir(elsewhere)

	got := s.certCacheDir()
	if want := filepath.Join(dataDir, "certs"); got != want {
		t.Errorf("certCacheDir = %q, want %q", got, want)
	}
	if strings.HasPrefix(got, elsewhere) {
		t.Errorf("the certificate cache followed the working directory: %q", got)
	}
}

// basil.data_dir is how site code finds somewhere durable to write.
func TestBasilContextExposesTheDataRoot(t *testing.T) {
	cfg, releaseDir, dataDir := siteRootFixture(t)
	siteDir := cfg.Site.Path
	must(os.WriteFile(filepath.Join(siteDir, "index.pars"), []byte(`basil.data_dir`), 0644))

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", "test-commit", &stdout, &stderr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	handler := newSiteHandler(s, siteDir, s.scriptCache)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), dataDir) {
		t.Errorf("basil.data_dir = %q, want %q", rec.Body.String(), dataDir)
	}
	if strings.Contains(rec.Body.String(), releaseDir) {
		t.Errorf("basil.data_dir points inside the release: %q", rec.Body.String())
	}
}

// A search index is written at runtime. Anchored to the handler root, as it
// was, every @SEARCH index sits inside the release and is destroyed by the
// next deploy.
func TestSearchIndexLandsInTheDataRoot(t *testing.T) {
	_, releaseDir, dataDir := siteRootFixture(t)

	env := evaluator.NewEnvironment()
	env.RootPath = releaseDir
	env.DataPath = dataDir

	inst, err := createSearchInstance(SearchOptions{Path: "posts.db", Tokenizer: "porter", HighlightTag: "mark", SnippetLen: 100}, env)
	if err != nil {
		t.Fatalf("createSearchInstance: %v", err)
	}
	defer inst.db.Close()

	want := filepath.Join(dataDir, "search", "posts.db")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("index not created at %s: %v", want, err)
	}
	if _, err := os.Stat(filepath.Join(releaseDir, "posts.db")); err == nil {
		t.Error("the index was written inside the release")
	}
}

// Without a data root (pars, not basil) the old behaviour is preserved.
func TestSearchIndexKeepsScriptRelativeBehaviourWithoutADataRoot(t *testing.T) {
	dir := t.TempDir()
	env := evaluator.NewEnvironment()
	env.RootPath = dir

	inst, err := createSearchInstance(SearchOptions{Path: "posts.db", Tokenizer: "porter", HighlightTag: "mark", SnippetLen: 100}, env)
	if err != nil {
		t.Fatalf("createSearchInstance: %v", err)
	}
	defer inst.db.Close()

	if _, err := os.Stat(filepath.Join(dir, "posts.db")); err != nil {
		t.Errorf("index not created next to the script: %v", err)
	}
}
