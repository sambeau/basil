package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/server/config"
)

// asset() maps a path under the public directory to the web-root URL the file
// is actually served at. That contract has two halves and BUG-049 broke the
// first: the URL asset() returns must be one the server serves the file at.
// In site mode the synthesized route's PublicDir field carries the handler
// root (so @~/ resolves there - BUG-011), and exporting that overloaded value
// as basil.public_dir made asset() strip the wrong prefix:
// asset(@./public/images/logo.png) returned "/public/images/logo.png", a URL
// the site catch-all serves as HTML. In routes mode without a per-route
// public_dir, basil.public_dir was empty and asset() was a silent no-op.
//
// These tests assert the contract end to end: the handler's output is the
// URL, and a GET of that URL through the same mux returns the file's bytes.

// assetServer builds a server over a temp project. siteMode selects
// filesystem routing (the handler at site/index.pars) or an explicit route
// (the handler at the project root, with no per-route public_dir).
func assetServer(t *testing.T, siteMode bool) *Server {
	t.Helper()
	evaluator.ClearModuleCache()

	dir := t.TempDir()
	publicDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(filepath.Join(publicDir, "images"), 0o755); err != nil {
		t.Fatalf("mkdir public/images: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "images", "logo.png"), []byte("PNGBYTES"), 0o644); err != nil {
		t.Fatalf("write logo: %v", err)
	}

	handler := "asset(@./public/images/logo.png)\n"

	cfg := &config.Config{
		ReleaseDir: dir,
		DataDir:    dir,
		Server:     config.ServerConfig{Host: "localhost", Port: 8080, Dev: true},
		PublicDir:  publicDir,
		Logging:    config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}

	if siteMode {
		siteDir := filepath.Join(dir, "site")
		if err := os.MkdirAll(siteDir, 0o755); err != nil {
			t.Fatalf("mkdir site: %v", err)
		}
		if err := os.WriteFile(filepath.Join(siteDir, "index.pars"), []byte(handler), 0o644); err != nil {
			t.Fatalf("write handler: %v", err)
		}
		cfg.Site = config.SiteConfig{Path: siteDir}
	} else {
		handlerPath := filepath.Join(dir, "page.pars")
		if err := os.WriteFile(handlerPath, []byte(handler), 0o644); err != nil {
			t.Fatalf("write handler: %v", err)
		}
		rootPath := filepath.Join(dir, "root.pars")
		if err := os.WriteFile(rootPath, []byte("\"root\"\n"), 0o644); err != nil {
			t.Fatalf("write root handler: %v", err)
		}
		// config.Load's ResolvePaths gives the "/" route the global
		// public_dir, and that route's static fallback is what serves the
		// public files at the web root. Mirror that shape here.
		cfg.Routes = []config.Route{
			{Path: "/page", Handler: handlerPath},
			{Path: "/", Handler: rootPath, PublicDir: publicDir},
		}
	}

	srv, err := New(cfg, "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return srv
}

// assetGet issues a request through the server's mux and returns the body.
func assetGet(t *testing.T, srv *Server, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	srv.mux.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func assertAssetURLServesFile(t *testing.T, srv *Server, page string) {
	t.Helper()

	code, body := assetGet(t, srv, page)
	if code != http.StatusOK {
		t.Fatalf("handler: expected 200, got %d: %s", code, body)
	}
	// Dev mode may append live-reload markup, so contain rather than equal.
	if !strings.Contains(body, "/images/logo.png") {
		t.Fatalf("expected body to contain %q, got %q", "/images/logo.png", body)
	}
	if strings.Contains(body, "/public/images/logo.png") {
		t.Fatalf("asset() kept the public prefix (BUG-049): %q", body)
	}

	code, body = assetGet(t, srv, "/images/logo.png")
	if code != http.StatusOK {
		t.Fatalf("the URL asset() returned does not serve: %d", code)
	}
	if body != "PNGBYTES" {
		t.Fatalf("expected the file's bytes at the returned URL, got %q", body)
	}
}

// TestAssetURLSiteMode is the regression test for BUG-049: in site mode
// basil.public_dir carried the handler root, so asset() stripped the wrong
// prefix and returned a URL the catch-all serves as HTML.
func TestAssetURLSiteMode(t *testing.T) {
	assertAssetURLServesFile(t, assetServer(t, true), "/")
}

// TestAssetURLRouteModeGlobalPublicDir: a route with no per-route public_dir
// falls back to the global one. Before the fix basil.public_dir was empty
// here and asset() returned its argument unchanged.
func TestAssetURLRouteModeGlobalPublicDir(t *testing.T) {
	assertAssetURLServesFile(t, assetServer(t, false), "/page")
}
