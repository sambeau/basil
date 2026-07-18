package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
)

// newAuthTestServer builds a server with auth enabled and returns it plus a
// session cookie for a user with the given role.
func newAuthTestServer(t *testing.T, cfg *config.Config, dir, role string) (*Server, *http.Cookie) {
	t.Helper()

	// The auth database must exist before the server will start
	adb, err := auth.OpenOrCreateDB(dir)
	if err != nil {
		t.Fatalf("creating auth database: %v", err)
	}
	adb.Close()

	srv, err := New(cfg, filepath.Join(dir, "basil.yaml"), "test", "test", os.Stdout, os.Stderr)
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	t.Cleanup(func() {
		if srv.authDB != nil {
			srv.authDB.Close()
		}
		if srv.db != nil {
			srv.db.Close()
		}
		srv.cleanupDevTools()
	})

	user, err := srv.authDB.CreateUserWithRole("Test User", "test@example.com", role)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	sess, err := srv.authDB.CreateSession(user.ID, time.Hour)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	return srv, &http.Cookie{Name: auth.SessionCookieName, Value: sess.ID}
}

// Regression test: a signed-in user must get through a protected path in
// site mode. Previously the protected-path check ran before the auth
// middleware populated the request context, so every request bounced to
// the login page — even with a valid session.
func TestSiteModeProtectedPathAllowsSignedInUser(t *testing.T) {
	dir := t.TempDir()
	dashDir := filepath.Join(dir, "site", "dashboard")
	if err := os.MkdirAll(dashDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dashDir, "index.pars"), []byte(`<p>"secret"</p>`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.BaseDir = dir
	cfg.Server.Dev = true
	cfg.Server.Port = 8080
	cfg.Site.Path = filepath.Join(dir, "site")
	cfg.Auth.Enabled = true
	cfg.Auth.ProtectedPaths = []config.ProtectedPath{{Path: "/dashboard"}}
	cfg.PublicDir = ""

	srv, cookie := newAuthTestServer(t, cfg, dir, "editor")

	// Signed in: should reach the handler
	req := httptest.NewRequest("GET", "/dashboard/", http.NoBody)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("signed-in user: expected 200, got %d (location %q)", rec.Code, rec.Header().Get("Location"))
	}

	// Signed out: should redirect to login with ?next=
	req2 := httptest.NewRequest("GET", "/dashboard/", http.NoBody)
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Errorf("anonymous user: expected 302, got %d", rec2.Code)
	}
	if loc := rec2.Header().Get("Location"); loc != "/login?next=/dashboard/" {
		t.Errorf("anonymous user: expected redirect to /login?next=/dashboard/, got %q", loc)
	}
}

// Regression test: same as above, in routes mode.
func TestRoutesModeProtectedPathAllowsSignedInUser(t *testing.T) {
	dir := t.TempDir()
	handlers := filepath.Join(dir, "handlers")
	if err := os.MkdirAll(handlers, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(handlers, "dash.pars"), []byte(`<p>"secret"</p>`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.BaseDir = dir
	cfg.Server.Dev = true
	cfg.Server.Port = 8080
	cfg.Auth.Enabled = true
	cfg.Auth.ProtectedPaths = []config.ProtectedPath{{Path: "/dashboard"}}
	cfg.Routes = []config.Route{{Path: "/dashboard", Handler: filepath.Join(handlers, "dash.pars")}}
	cfg.PublicDir = ""

	srv, cookie := newAuthTestServer(t, cfg, dir, "editor")

	req := httptest.NewRequest("GET", "/dashboard", http.NoBody)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("signed-in user: expected 200, got %d (location %q)", rec.Code, rec.Header().Get("Location"))
	}

	req2 := httptest.NewRequest("GET", "/dashboard", http.NoBody)
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Errorf("anonymous user: expected 302, got %d", rec2.Code)
	}
}

// Role checks on protected paths: wrong role gets 403, right role gets 200.
func TestProtectedPathRoleEnforcement(t *testing.T) {
	dir := t.TempDir()
	adminDir := filepath.Join(dir, "site", "admin")
	if err := os.MkdirAll(adminDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "index.pars"), []byte(`<p>"admin only"</p>`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.BaseDir = dir
	cfg.Server.Dev = true
	cfg.Server.Port = 8080
	cfg.Site.Path = filepath.Join(dir, "site")
	cfg.Auth.Enabled = true
	cfg.Auth.ProtectedPaths = []config.ProtectedPath{{Path: "/admin", Roles: []string{"admin"}}}
	cfg.PublicDir = ""

	srv, editorCookie := newAuthTestServer(t, cfg, dir, "editor")

	// Editor: 403
	req := httptest.NewRequest("GET", "/admin/", http.NoBody)
	req.AddCookie(editorCookie)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("editor on admin path: expected 403, got %d", rec.Code)
	}

	// Admin: 200
	adminUser, err := srv.authDB.CreateUserWithRole("Admin", "admin@example.com", "admin")
	if err != nil {
		t.Fatal(err)
	}
	adminSess, err := srv.authDB.CreateSession(adminUser.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest("GET", "/admin/", http.NoBody)
	req2.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: adminSess.ID})
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("admin on admin path: expected 200, got %d", rec2.Code)
	}
}

// Route-level roles in routes mode: `roles:` on a route requires sign-in
// and the matching role, even without a protected_paths entry.
func TestRoutesModeRouteRoles(t *testing.T) {
	dir := t.TempDir()
	handlers := filepath.Join(dir, "handlers")
	if err := os.MkdirAll(handlers, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(handlers, "admin.pars"), []byte(`<p>"admin only"</p>`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.BaseDir = dir
	cfg.Server.Dev = true
	cfg.Server.Port = 8080
	cfg.Auth.Enabled = true
	cfg.Routes = []config.Route{{Path: "/admin", Handler: filepath.Join(handlers, "admin.pars"), Roles: []string{"admin"}}}
	cfg.PublicDir = ""

	srv, editorCookie := newAuthTestServer(t, cfg, dir, "editor")

	// Editor: 403
	req := httptest.NewRequest("GET", "/admin", http.NoBody)
	req.AddCookie(editorCookie)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("editor on admin route: expected 403, got %d", rec.Code)
	}

	// Anonymous: redirect to login
	req2 := httptest.NewRequest("GET", "/admin", http.NoBody)
	rec2 := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusFound {
		t.Errorf("anonymous on admin route: expected 302, got %d", rec2.Code)
	}
}
