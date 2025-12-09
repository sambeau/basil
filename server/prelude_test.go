package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitPrelude(t *testing.T) {
	tests := []struct {
		name    string
		commit  string
		wantErr bool
	}{
		{
			name:    "with commit hash",
			commit:  "abc1234",
			wantErr: false,
		},
		{
			name:    "with long commit hash",
			commit:  "abc1234567890",
			wantErr: false,
		},
		{
			name:    "without commit",
			commit:  "",
			wantErr: false,
		},
		{
			name:    "with unknown commit",
			commit:  "unknown",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset globals
			preludeASTs = nil
			jsAssetHash = ""

			err := initPrelude(tt.commit)
			if (err != nil) != tt.wantErr {
				t.Errorf("initPrelude() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify preludeASTs was initialized
				if preludeASTs == nil {
					t.Error("preludeASTs was not initialized")
				}

				// Verify jsAssetHash was set
				if jsAssetHash == "" {
					t.Error("jsAssetHash was not set")
				}
			}
		})
	}
}

func TestInitJSHash(t *testing.T) {
	tests := []struct {
		name       string
		commit     string
		wantLength int
	}{
		{
			name:       "7 char commit",
			commit:     "abc1234",
			wantLength: 7,
		},
		{
			name:       "long commit (truncated)",
			commit:     "abc1234567890",
			wantLength: 7,
		},
		{
			name:       "empty commit (content hash)",
			commit:     "",
			wantLength: 7,
		},
		{
			name:       "unknown commit (content hash)",
			commit:     "unknown",
			wantLength: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsAssetHash = ""
			initJSHash(tt.commit)

			if len(jsAssetHash) != tt.wantLength {
				t.Errorf("jsAssetHash length = %d, want %d (hash: %s)", len(jsAssetHash), tt.wantLength, jsAssetHash)
			}

			// Verify it's not empty
			if jsAssetHash == "" {
				t.Error("jsAssetHash should not be empty")
			}

			// If commit is valid and 7+ chars, should match truncated commit
			if tt.commit != "" && tt.commit != "unknown" && len(tt.commit) >= 7 {
				expected := tt.commit[:7]
				if jsAssetHash != expected {
					t.Errorf("jsAssetHash = %s, want %s", jsAssetHash, expected)
				}
			}
		})
	}
}

func TestJSAssetURL(t *testing.T) {
	// Set a known hash
	jsAssetHash = "abc1234"

	url := JSAssetURL()
	expected := "/__/js/basil.abc1234.js"

	if url != expected {
		t.Errorf("JSAssetURL() = %s, want %s", url, expected)
	}
}

func TestGetPreludeAST(t *testing.T) {
	// Initialize prelude
	if err := initPrelude("test123"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	// Test non-existent file
	ast := GetPreludeAST("nonexistent.pars")
	if ast != nil {
		t.Error("GetPreludeAST for non-existent file should return nil")
	}
}

func TestHasPreludeAST(t *testing.T) {
	// Initialize prelude
	if err := initPrelude("test123"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	// Test non-existent file
	if HasPreludeAST("nonexistent.pars") {
		t.Error("HasPreludeAST for non-existent file should return false")
	}
}

func TestHandlePreludeAsset_BasilJS(t *testing.T) {
	// Initialize prelude
	if err := initPrelude("abc1234"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	s := &Server{}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantType   string
		wantCache  string
	}{
		{
			name:       "basil.js with correct hash",
			path:       "/__/js/basil.abc1234.js",
			wantStatus: http.StatusOK,
			wantType:   "application/javascript",
			wantCache:  "public, max-age=31536000, immutable",
		},
		{
			name:       "basil.js with different hash (still works)",
			path:       "/__/js/basil.xyz9999.js",
			wantStatus: http.StatusOK,
			wantType:   "application/javascript",
			wantCache:  "public, max-age=31536000, immutable",
		},
		{
			name:       "basil.js direct (unversioned)",
			path:       "/__/js/basil.js",
			wantStatus: http.StatusOK,
			wantType:   "application/javascript",
			wantCache:  "public, max-age=3600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			s.handlePreludeAsset(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Code == http.StatusOK {
				if ct := w.Header().Get("Content-Type"); ct != tt.wantType {
					t.Errorf("Content-Type = %s, want %s", ct, tt.wantType)
				}

				if cc := w.Header().Get("Cache-Control"); cc != tt.wantCache {
					t.Errorf("Cache-Control = %s, want %s", cc, tt.wantCache)
				}

				// Verify body contains JavaScript
				body := w.Body.String()
				if !strings.Contains(body, "document.querySelectorAll") {
					t.Error("response should contain JavaScript code")
				}
			}
		})
	}
}

func TestHandlePreludeAsset_NotFound(t *testing.T) {
	// Initialize prelude
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	s := &Server{}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "non-existent JS file",
			path: "/__/js/nonexistent.js",
		},
		{
			name: "non-existent CSS file",
			path: "/__/css/nonexistent.css",
		},
		{
			name: "invalid path prefix",
			path: "/__/invalid/file.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			s.handlePreludeAsset(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandlePreludeAsset_DirectoryTraversal(t *testing.T) {
	// Initialize prelude
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude() error = %v", err)
	}

	s := &Server{}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "parent directory traversal",
			path: "/__/js/../../../etc/passwd",
		},
		{
			name: "relative path traversal",
			path: "/__/js/../../server.go",
		},
		{
			name: "encoded traversal",
			path: "/__/js/..%2F..%2Fserver.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			s.handlePreludeAsset(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d (should block directory traversal)", w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestIsVersionedAsset(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "versioned with 7 char hash",
			filename: "basil.abc1234.js",
			want:     true,
		},
		{
			name:     "versioned with long hash",
			filename: "basil.abc1234567890.js",
			want:     true,
		},
		{
			name:     "not versioned (direct)",
			filename: "basil.js",
			want:     false,
		},
		{
			name:     "short hash (< 7 chars)",
			filename: "basil.abc.js",
			want:     false,
		},
		{
			name:     "no extension",
			filename: "basil",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVersionedAsset(tt.filename)
			if got != tt.want {
				t.Errorf("isVersionedAsset(%s) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
