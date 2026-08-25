package server

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"io"
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

// A percent-encoded service must not slip the role gate: ?service=git-
// receive%2dpack decodes to receive-pack for the advertisement, so it must
// decode to receive-pack for the gate too. (Regression PoC for the FEAT-154
// review: raw-substring gate vs decoded advertisement disagreed.)
func TestGitHandler_ViewerCannotPushPercentEncoded(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	authDB, key := newTestAuthDB(t, "Vera", auth.RoleViewer)
	h := newTestGitHandler(t, cfg, authDB)

	for _, target := range []string{
		"/.git/info/refs?service=git-receive-pack",   // plain form
		"/.git/info/refs?service=git-receive%2dpack", // %2d = '-'
	} {
		req := tlsRequest("GET", target)
		req.SetBasicAuth("vera", key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("viewer at %s: got %d, want 403 (body: %s)", target, w.Code, w.Body.String())
		}
		if body := w.Body.String(); !strings.Contains(body, "editor or admin") {
			t.Errorf("viewer at %s: refusal should name the required role, got: %q", target, body)
		}
	}
}

// The same percent-encoded receive-pack advertisement an editor requests is
// allowed — the gate decodes, it does not blanket-refuse encoded forms.
func TestGitHandler_EditorMayPushPercentEncoded(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: false}}
	authDB, key := newTestAuthDB(t, "Eddy", auth.RoleEditor)
	h := newTestGitHandler(t, cfg, authDB)

	req := tlsRequest("GET", "/.git/info/refs?service=git-receive%2dpack")
	req.SetBasicAuth("eddy", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("editor percent-encoded push advertisement: got %d, want 200 (body: %s)", w.Code, w.Body.String())
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

func TestGitHandler_GitService(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Dev: true}}
	h := newTestGitHandler(t, cfg, nil)

	tests := []struct {
		method  string
		path    string
		query   string
		service string
		ok      bool
	}{
		{"GET", "/.git/info/refs", "service=git-upload-pack", "upload-pack", true},
		{"GET", "/.git/info/refs", "service=git-receive-pack", "receive-pack", true},
		// Percent-encoded hyphen (%2d = '-') must decode to the same service
		// the advertisement runs, so the role gate cannot be slipped.
		{"GET", "/.git/info/refs", "service=git-receive%2dpack", "receive-pack", true},
		{"GET", "/.git/info/refs", "service=git-upload%2dpack", "upload-pack", true},
		// Unknown / missing / dumb-protocol services are not git requests.
		{"GET", "/.git/info/refs", "service=git-evil-pack", "", false},
		{"GET", "/.git/info/refs", "", "", false},
		{"POST", "/.git/git-upload-pack", "", "upload-pack", true},
		{"POST", "/.git/git-receive-pack", "", "receive-pack", true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path+"?"+tt.query, http.NoBody)
		service, ok := h.gitService(req)
		if service != tt.service || ok != tt.ok {
			t.Errorf("gitService(%s %s?%s) = (%q, %v), want (%q, %v)",
				tt.method, tt.path, tt.query, service, ok, tt.service, tt.ok)
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

// A gzipped RPC body is bounded by its DECOMPRESSED size, so a tiny "gzip
// bomb" cannot inflate without limit into git's stdin. limitReader refuses
// (errors) rather than truncating; a normal body passes straight through.
func TestGitHandler_BoundedGzip(t *testing.T) {
	gzipOf := func(n int) []byte {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := zw.Write(bytes.Repeat([]byte("A"), n)); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	// Under the cap: the full decompressed payload reads back cleanly.
	small := gzipOf(4096)
	gz, err := gzip.NewReader(bytes.NewReader(small))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(&limitReader{r: gz, limit: 1 << 20})
	if err != nil {
		t.Fatalf("normal body was refused: %v", err)
	}
	if len(got) != 4096 {
		t.Errorf("decompressed %d bytes, want 4096", len(got))
	}

	// Over the cap: a body that inflates past the limit is refused. The
	// compressed input here is a few dozen bytes; decompressed it is 1 MiB.
	bomb := gzipOf(1 << 20)
	if len(bomb) > 4096 {
		t.Fatalf("test bomb is not actually small on the wire: %d bytes", len(bomb))
	}
	gz, err = gzip.NewReader(bytes.NewReader(bomb))
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(&limitReader{r: gz, limit: 64 << 10}) // 64 KiB cap
	if !errors.Is(err, errPushTooLarge) {
		t.Errorf("over-limit decompressed body: err = %v, want errPushTooLarge", err)
	}
}
