package images

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestRegistry(t *testing.T, devMode bool) (*Registry, string) {
	t.Helper()
	cacheDir := t.TempDir()
	reg := NewRegistry(cacheDir, 2048, 2048, 80, "", devMode, nil)
	return reg, cacheDir
}

// registerTestImage creates a test image, transforms it through the registry,
// and returns the URL path (e.g. "/__img/{hash}.{ext}").
func registerTestImage(t *testing.T, reg *Registry, format string) string {
	t.Helper()
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "test."+format, 100, 80, format)

	url, err := reg.Transform(srcPath, map[string]any{"width": 50})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	return url
}

func TestHandler_ServeImage(t *testing.T) {
	reg, _ := newTestRegistry(t, false)
	handler := NewHandler(reg, false)

	imgURL := registerTestImage(t, reg, "jpeg")

	req := httptest.NewRequest(http.MethodGet, imgURL, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.Len() == 0 {
		t.Error("response body is empty, want non-empty image data")
	}
}

func TestHandler_NotFound_UnknownHash(t *testing.T) {
	reg, _ := newTestRegistry(t, false)
	handler := NewHandler(reg, false)

	req := httptest.NewRequest(http.MethodGet, "/__img/unknownhash1234.jpg", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandler_NotFound_EmptyHash(t *testing.T) {
	reg, _ := newTestRegistry(t, false)
	handler := NewHandler(reg, false)

	tests := []struct {
		name string
		path string
	}{
		{"bare prefix", "/__img/"},
		{"extension only", "/__img/.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHandler_NotFound_ExtensionMismatch(t *testing.T) {
	reg, _ := newTestRegistry(t, false)
	handler := NewHandler(reg, false)

	// Register a JPEG image
	imgURL := registerTestImage(t, reg, "jpeg")

	// Replace .jpg with .png to create an extension mismatch
	mismatchURL := strings.TrimSuffix(imgURL, ".jpg") + ".png"

	req := httptest.NewRequest(http.MethodGet, mismatchURL, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandler_CacheHeaders_DevMode(t *testing.T) {
	reg, _ := newTestRegistry(t, true)
	handler := NewHandler(reg, true)

	imgURL := registerTestImage(t, reg, "jpeg")

	req := httptest.NewRequest(http.MethodGet, imgURL, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	cc := rr.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-cache") {
		t.Errorf("Cache-Control = %q, want it to contain %q", cc, "no-cache")
	}
}

func TestHandler_CacheHeaders_ProdMode(t *testing.T) {
	reg, _ := newTestRegistry(t, false)
	handler := NewHandler(reg, false)

	imgURL := registerTestImage(t, reg, "jpeg")

	req := httptest.NewRequest(http.MethodGet, imgURL, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	cc := rr.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want it to contain %q", cc, "immutable")
	}
}
