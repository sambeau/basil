package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/config"
)

// TestSiteHandler_WalkBack tests the walk-back algorithm for finding handlers.
func TestSiteHandler_WalkBack(t *testing.T) {
	// Create a temp site structure
	dir := t.TempDir()

	// Create site structure:
	// site/
	//   index.pars              -> handles /
	//   reports/
	//     index.pars            -> handles /reports/ and /reports/2025/Q4/
	//   admin/
	//     index.pars            -> handles /admin/
	//     users/
	//       index.pars          -> handles /admin/users/

	siteDir := filepath.Join(dir, "site")
	must(os.MkdirAll(filepath.Join(siteDir, "reports"), 0755))
	must(os.MkdirAll(filepath.Join(siteDir, "admin", "users"), 0755))

	// Root handler
	must(os.WriteFile(filepath.Join(siteDir, "index.pars"), []byte(`"root"`), 0644))
	// Reports handler
	must(os.WriteFile(filepath.Join(siteDir, "reports", "index.pars"), []byte(`"reports"`), 0644))
	// Admin handler
	must(os.WriteFile(filepath.Join(siteDir, "admin", "index.pars"), []byte(`"admin"`), 0644))
	// Admin users handler
	must(os.WriteFile(filepath.Join(siteDir, "admin", "users", "index.pars"), []byte(`"users"`), 0644))

	handler := newSiteHandler(nil, siteDir, nil)

	tests := []struct {
		name              string
		urlPath           string
		wantHandlerSuffix string // Suffix of expected handler path
		wantSubpath       string
	}{
		{
			name:              "root path",
			urlPath:           "/",
			wantHandlerSuffix: "/index.pars",
			wantSubpath:       "",
		},
		{
			name:              "reports root",
			urlPath:           "/reports/",
			wantHandlerSuffix: "/reports/index.pars",
			wantSubpath:       "",
		},
		{
			name:              "reports with subpath",
			urlPath:           "/reports/2025/",
			wantHandlerSuffix: "/reports/index.pars",
			wantSubpath:       "/2025",
		},
		{
			name:              "reports with deep subpath",
			urlPath:           "/reports/2025/Q4/",
			wantHandlerSuffix: "/reports/index.pars",
			wantSubpath:       "/2025/Q4",
		},
		{
			name:              "admin root",
			urlPath:           "/admin/",
			wantHandlerSuffix: "/admin/index.pars",
			wantSubpath:       "",
		},
		{
			name:              "admin users (specific handler)",
			urlPath:           "/admin/users/",
			wantHandlerSuffix: "/admin/users/index.pars",
			wantSubpath:       "",
		},
		{
			name:              "admin users with subpath",
			urlPath:           "/admin/users/123/",
			wantHandlerSuffix: "/admin/users/index.pars",
			wantSubpath:       "/123",
		},
		{
			name:              "nonexistent path falls back to parent",
			urlPath:           "/admin/settings/",
			wantHandlerSuffix: "/admin/index.pars",
			wantSubpath:       "/settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerPath, subpath := handler.findHandler(tt.urlPath)

			if !strings.HasSuffix(handlerPath, tt.wantHandlerSuffix) {
				t.Errorf("findHandler(%q) handler = %q, want suffix %q", tt.urlPath, handlerPath, tt.wantHandlerSuffix)
			}
			if subpath != tt.wantSubpath {
				t.Errorf("findHandler(%q) subpath = %q, want %q", tt.urlPath, subpath, tt.wantSubpath)
			}
		})
	}
}

// TestSiteHandler_NoHandler tests 404 when no handler is found.
func TestSiteHandler_NoHandler(t *testing.T) {
	dir := t.TempDir()
	siteDir := filepath.Join(dir, "site")
	must(os.MkdirAll(siteDir, 0755))
	// No index.pars files

	handler := newSiteHandler(nil, siteDir, nil)

	handlerPath, _ := handler.findHandler("/any/path/")
	if handlerPath != "" {
		t.Errorf("expected empty handler path for site with no index.pars, got %q", handlerPath)
	}
}

// TestSiteHandler_PathTraversal tests rejection of path traversal attempts.
func TestSiteHandler_PathTraversal(t *testing.T) {
	tests := []struct {
		path        string
		wantBlocked bool
	}{
		{"/normal/path/", false},
		{"/../etc/passwd", true},
		{"/reports/../../../etc/passwd", true},
		{"/reports/..%2F..%2Fetc/passwd", false}, // URL encoded, not actual ..
		{"/reports/2025/../2024/", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blocked := containsPathTraversal(tt.path)
			if blocked != tt.wantBlocked {
				t.Errorf("containsPathTraversal(%q) = %v, want %v", tt.path, blocked, tt.wantBlocked)
			}
		})
	}
}

// TestSiteHandler_Dotfiles tests rejection of dotfile access.
func TestSiteHandler_Dotfiles(t *testing.T) {
	tests := []struct {
		path        string
		wantBlocked bool
	}{
		{"/normal/path/", false},
		{"/.git/config", true},
		{"/reports/.hidden/file", true},
		{"/.env", true},
		{"/reports/2025/", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			blocked := containsDotfile(tt.path)
			if blocked != tt.wantBlocked {
				t.Errorf("containsDotfile(%q) = %v, want %v", tt.path, blocked, tt.wantBlocked)
			}
		})
	}
}

// TestSiteHandler_TrailingSlashRedirect tests the trailing slash redirect behavior.
func TestSiteHandler_TrailingSlashRedirect(t *testing.T) {
	dir := t.TempDir()
	siteDir := filepath.Join(dir, "site")
	must(os.MkdirAll(filepath.Join(siteDir, "reports"), 0755))
	must(os.WriteFile(filepath.Join(siteDir, "reports", "index.pars"), []byte(`"reports"`), 0644))

	cfg := &config.Config{
		Server: config.ServerConfig{Dev: true},
		Site:   siteDir,
	}
	s := &Server{config: cfg}
	handler := newSiteHandler(s, siteDir, nil)

	// Request without trailing slash should redirect
	req := httptest.NewRequest("GET", "/reports", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/reports/" {
		t.Errorf("expected redirect to /reports/, got %q", loc)
	}
}

// TestSiteHandler_StaticFiles tests serving static files from public_dir.
func TestSiteHandler_StaticFiles(t *testing.T) {
	dir := t.TempDir()
	siteDir := filepath.Join(dir, "site")
	publicDir := filepath.Join(dir, "public")

	must(os.MkdirAll(siteDir, 0755))
	must(os.MkdirAll(publicDir, 0755))

	// Create a static file
	must(os.WriteFile(filepath.Join(publicDir, "style.css"), []byte("body {}"), 0644))
	// Create site index
	must(os.WriteFile(filepath.Join(siteDir, "index.pars"), []byte(`"root"`), 0644))

	cfg := &config.Config{
		Server:    config.ServerConfig{Dev: true},
		Site:      siteDir,
		PublicDir: publicDir,
	}
	s := &Server{config: cfg}
	handler := newSiteHandler(s, siteDir, nil)

	req := httptest.NewRequest("GET", "/style.css", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "body {}" {
		t.Errorf("expected static file content, got %q", body)
	}
}

// TestBuildSubpathObject tests the subpath Path object construction.
func TestBuildSubpathObject(t *testing.T) {
	tests := []struct {
		subpath  string
		wantSegs []interface{}
	}{
		{"", []interface{}{}},
		{"/", []interface{}{}},
		{"/2025", []interface{}{"2025"}},
		{"/2025/Q4", []interface{}{"2025", "Q4"}},
		{"/a/b/c", []interface{}{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.subpath, func(t *testing.T) {
			obj := buildSubpathObject(tt.subpath)

			if obj["__type"] != "path" {
				t.Errorf("expected __type='path', got %v", obj["__type"])
			}
			if obj["absolute"] != false {
				t.Errorf("expected absolute=false, got %v", obj["absolute"])
			}

			segs, ok := obj["segments"].([]interface{})
			if !ok {
				t.Fatalf("expected segments to be []interface{}, got %T", obj["segments"])
			}

			if len(segs) != len(tt.wantSegs) {
				t.Errorf("expected %d segments, got %d", len(tt.wantSegs), len(segs))
				return
			}

			for i, want := range tt.wantSegs {
				if segs[i] != want {
					t.Errorf("segment[%d] = %v, want %v", i, segs[i], want)
				}
			}
		})
	}
}

// TestSplitPath tests the path splitting helper function.
func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/", nil},
		{"/a", []string{"a"}},
		{"/a/b/c", []string{"a", "b", "c"}},
		{"/a/b/c/", []string{"a", "b", "c"}},
		{"a/b", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// must panics if err is not nil (test helper)
func must(err error) {
	if err != nil {
		panic(err)
	}
}
