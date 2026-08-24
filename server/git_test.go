package server

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
	"golang.org/x/crypto/acme/autocert"
)

// newTestGitHandler builds a GitHandler over a fresh bare repository.
func newTestGitHandler(t *testing.T, cfg *config.Config, authDB *auth.DB) *GitHandler {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	repo := filepath.Join(t.TempDir(), "site.git")
	cmd := exec.Command("git", "init", "--quiet", "--bare", repo)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v: %s", err, out)
	}
	var stdout, stderr bytes.Buffer
	h, err := NewGitHandler(repo, authDB, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("NewGitHandler: %v", err)
	}
	return h
}

// newTestAuthDB returns an auth DB holding one user of the given role and
// that user's API key.
func newTestAuthDB(t *testing.T, name, role string) (*auth.DB, string) {
	t.Helper()
	authDB, err := auth.OpenOrCreateDB(t.TempDir())
	if err != nil {
		t.Fatalf("OpenOrCreateDB: %v", err)
	}
	t.Cleanup(func() { authDB.Close() })
	user, err := authDB.CreateUserWithRole(name, name+"@test.example", role)
	if err != nil {
		t.Fatalf("CreateUserWithRole: %v", err)
	}
	_, plaintext, err := authDB.CreateAPIKey(user.ID, "test-key")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	return authDB, plaintext
}

// tlsRequest fakes a request that arrived over TLS.
func tlsRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, http.NoBody)
	req.TLS = &tls.ConnectionState{}
	req.RemoteAddr = "203.0.113.9:41000" // decidedly not localhost
	return req
}

// Git over plain HTTP is refused with an explanation, because Basic auth
// would put the API key on the wire in cleartext.
func TestGitHandler_PlainHTTPRefused(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	h := newTestGitHandler(t, cfg, nil)

	req := httptest.NewRequest("GET", "/.git/info/refs?service=git-upload-pack", http.NoBody)
	req.RemoteAddr = "203.0.113.9:41000"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("plain-HTTP request: got %d, want 403", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "API key") {
		t.Errorf("refusal body should explain the API key exposure, got: %q", body)
	}
}

// The dev-mode relaxation is dev AND localhost: dev mode alone does not
// permit plain HTTP from elsewhere.
func TestGitHandler_PlainHTTPRefusedNonLocalhostEvenInDev(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	req := httptest.NewRequest("GET", "/.git/info/refs?service=git-upload-pack", http.NoBody)
	req.RemoteAddr = "203.0.113.9:41000"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("plain-HTTP non-localhost dev request: got %d, want 403", w.Code)
	}
}

// Dev-mode localhost is the one relaxation: plain HTTP, no authentication.
func TestGitHandler_DevLocalhostAllowed(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	req := httptest.NewRequest("GET", "/.git/info/refs?service=git-upload-pack", http.NoBody)
	req.RemoteAddr = "127.0.0.1:41000"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dev-localhost advertisement: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/x-git-upload-pack-advertisement" {
		t.Errorf("Content-Type = %q", got)
	}
	if !strings.HasPrefix(w.Body.String(), "001e# service=git-upload-pack\n") {
		t.Errorf("advertisement does not start with the service pkt-line: %q", w.Body.String()[:min(40, w.Body.Len())])
	}
}

// No auth database, no Git: even over TLS the handler refuses rather than
// serving anything unauthenticated. (Startup refuses to reach this state
// outside dev mode; this is defence in depth.)
func TestGitHandler_NoAuthDBRefused(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	h := newTestGitHandler(t, cfg, nil)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, tlsRequest("GET", "/.git/info/refs?service=git-upload-pack"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("no-auth-DB request: got %d, want 403", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "auth database") {
		t.Errorf("refusal should name the missing auth database, got: %q", body)
	}
}

// Without credentials the handler challenges; authentication is not
// configurable away.
func TestGitHandler_AuthRequired(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	authDB, _ := newTestAuthDB(t, "Alice", auth.RoleAdmin)
	h := newTestGitHandler(t, cfg, authDB)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, tlsRequest("GET", "/.git/info/refs?service=git-upload-pack"))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request: got %d, want 401", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate challenge")
	}
}

// A viewer may fetch but not push, and the refusal says which role is
// required.
func TestGitHandler_ViewerCannotPush(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	authDB, key := newTestAuthDB(t, "Vera", auth.RoleViewer)
	h := newTestGitHandler(t, cfg, authDB)

	// Fetch advertisement: allowed for any authenticated user.
	req := tlsRequest("GET", "/.git/info/refs?service=git-upload-pack")
	req.SetBasicAuth("vera", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("viewer fetch advertisement: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	// Push advertisement: refused, naming the required role.
	req = tlsRequest("GET", "/.git/info/refs?service=git-receive-pack")
	req.SetBasicAuth("vera", key)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer push advertisement: got %d, want 403", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "editor or admin") {
		t.Errorf("role refusal should say which role is required, got: %q", body)
	}
}

// An editor gets through the role gate for pushes.
func TestGitHandler_EditorMayPush(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	authDB, key := newTestAuthDB(t, "Eddy", auth.RoleEditor)
	h := newTestGitHandler(t, cfg, authDB)

	req := tlsRequest("GET", "/.git/info/refs?service=git-receive-pack")
	req.SetBasicAuth("eddy", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("editor push advertisement: got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestGitHandler_IsPushRequest(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	tests := []struct {
		path   string
		query  string
		isPush bool
	}{
		{"/.git/info/refs", "service=git-upload-pack", false},
		{"/.git/info/refs", "service=git-receive-pack", true},
		{"/.git/git-upload-pack", "", false},
		{"/.git/git-receive-pack", "", true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("POST", tt.path+"?"+tt.query, http.NoBody)
		got := h.isPushRequest(req)
		if got != tt.isPush {
			t.Errorf("isPushRequest(%s?%s) = %v, want %v", tt.path, tt.query, got, tt.isPush)
		}
	}
}

func TestGitHandler_PathTraversal(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	paths := []string{
		"/.git/../../../etc/passwd",
		"/.git/objects/../../secret",
		"/.git/../.git/config",
	}

	for _, path := range paths {
		req := httptest.NewRequest("GET", path, http.NoBody)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("path %q: expected 400, got %d", path, w.Code)
		}
	}
}

// The repository's own files are never served: only the Smart HTTP paths
// exist. (The old handler exposed loose objects, packfiles and HEAD over
// the dumb protocol.)
func TestGitHandler_NoRawFileServing(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	for _, path := range []string{"/.git/HEAD", "/.git/config", "/.git/objects/info/packs", "/.git/hooks/pre-receive"} {
		req := httptest.NewRequest("GET", path, http.NoBody)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s: got %d, want 404", path, w.Code)
		}
	}
}

// The plain-HTTP refusal is scoped to the Git handler on the main listener.
// The port-80 redirect listener must keep answering ACME HTTP-01 challenges
// — a blanket refusal there would make the server unable to obtain or renew
// the very certificate the refusal demands (FEAT-154).
func TestHTTPRedirectHandler_ACMEChallengeStillAnswers(t *testing.T) {
	manager := &autocert.Manager{Prompt: autocert.AcceptTOS}
	h := httpRedirectHandler(manager)

	// The challenge path is handled by the autocert manager, not redirected
	// and not refused. (An unknown token yields 404 — what matters is that
	// the manager answered, so a real token would be served.)
	req := httptest.NewRequest("GET", "http://example.com/.well-known/acme-challenge/some-token", http.NoBody)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code == http.StatusMovedPermanently || w.Code == http.StatusFound {
		t.Fatalf("ACME challenge was redirected (%d) instead of answered", w.Code)
	}
	if w.Code == http.StatusForbidden {
		t.Fatalf("ACME challenge was refused (403); the plain-HTTP refusal must not cover the challenge path")
	}

	// Everything else on port 80 — Git paths included — is redirected to
	// HTTPS, not refused: the refusal lives on the Git handler itself.
	req = httptest.NewRequest("GET", "http://example.com/.git/info/refs?service=git-upload-pack", http.NoBody)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("Git path on redirect listener: got %d, want 301", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.HasPrefix(loc, "https://") {
		t.Errorf("redirect location = %q, want https://…", loc)
	}
}

// The bare repository must not sit inside anything the server serves files
// from; a repository under a served root would expose every version of every
// file.
func TestCheckRepoOutsideServedRoots(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "public", "site.git")

	cfg := &config.Config{
		PublicDir: filepath.Join(root, "public"),
		Site:      config.SiteConfig{Path: filepath.Join(root, "site")},
	}
	err := checkRepoOutsideServedRoots(cfg, repo)
	if err == nil {
		t.Fatal("repository inside public_dir was accepted")
	}
	if !strings.Contains(err.Error(), repo) || !strings.Contains(err.Error(), cfg.PublicDir) {
		t.Errorf("error should name both paths, got: %v", err)
	}

	// A sibling repository (the --init layout) is fine.
	cfg2 := &config.Config{
		PublicDir: filepath.Join(root, "current", "public"),
		Site:      config.SiteConfig{Path: filepath.Join(root, "current", "site")},
		DataDir:   filepath.Join(root, "data"),
	}
	if err := checkRepoOutsideServedRoots(cfg2, filepath.Join(root, "site.git")); err != nil {
		t.Fatalf("sibling repository refused: %v", err)
	}

	// static[].root is a served root too.
	cfg3 := &config.Config{
		Static: []config.StaticRoute{{Path: "/files/", Root: root}},
	}
	if err := checkRepoOutsideServedRoots(cfg3, filepath.Join(root, "site.git")); err == nil {
		t.Fatal("repository inside static[].root was accepted")
	}

	// The uploads directory (under data_dir) is served at /__uploads/.
	cfg4 := &config.Config{DataDir: root, SiteRoot: filepath.Dir(root)}
	if err := checkRepoOutsideServedRoots(cfg4, filepath.Join(root, "uploads", "site.git")); err == nil {
		t.Fatal("repository inside the uploads directory was accepted")
	}
}
