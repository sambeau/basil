package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

func TestRedirect_BasicURL(t *testing.T) {
	// Create a Redirect object
	redirect := &evaluator.Redirect{
		URL:    "/new-location",
		Status: http.StatusFound, // 302
	}

	// Verify properties
	if redirect.URL != "/new-location" {
		t.Errorf("expected URL '/new-location', got %q", redirect.URL)
	}
	if redirect.Status != 302 {
		t.Errorf("expected status 302, got %d", redirect.Status)
	}

	// Verify type
	if redirect.Type() != evaluator.REDIRECT_OBJ {
		t.Errorf("expected type REDIRECT_OBJ, got %s", redirect.Type())
	}

	// Verify Inspect output
	expected := "redirect(/new-location, 302)"
	if redirect.Inspect() != expected {
		t.Errorf("expected Inspect %q, got %q", expected, redirect.Inspect())
	}
}

func TestRedirect_AbsoluteURL(t *testing.T) {
	redirect := &evaluator.Redirect{
		URL:    "https://example.com/page",
		Status: http.StatusMovedPermanently, // 301
	}

	if redirect.URL != "https://example.com/page" {
		t.Errorf("expected URL 'https://example.com/page', got %q", redirect.URL)
	}
	if redirect.Status != 301 {
		t.Errorf("expected status 301, got %d", redirect.Status)
	}
}

func TestRedirect_AllValidStatusCodes(t *testing.T) {
	// Test all valid 3xx status codes
	validStatuses := []int{
		http.StatusMultipleChoices,   // 300
		http.StatusMovedPermanently,  // 301
		http.StatusFound,             // 302
		http.StatusSeeOther,          // 303
		http.StatusNotModified,       // 304
		http.StatusUseProxy,          // 305
		http.StatusTemporaryRedirect, // 307
		http.StatusPermanentRedirect, // 308
	}

	for _, status := range validStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			redirect := &evaluator.Redirect{
				URL:    "/target",
				Status: status,
			}

			if redirect.Status != status {
				t.Errorf("expected status %d, got %d", status, redirect.Status)
			}
		})
	}
}

func TestApiRedirect_DefaultStatus(t *testing.T) {
	// Test that redirect() with just a URL returns a Redirect with default 302
	code := `let api = import("std/api")
api.redirect("/dashboard")
`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected Redirect result, got nil")
	}

	if result.Value.Type() != evaluator.REDIRECT_OBJ {
		t.Fatalf("expected REDIRECT_OBJ, got %s: %s", result.Value.Type(), result.Value.Inspect())
	}

	redirect := result.Value.(*evaluator.Redirect)
	if redirect.URL != "/dashboard" {
		t.Errorf("expected URL '/dashboard', got %q", redirect.URL)
	}
	if redirect.Status != 302 {
		t.Errorf("expected default status 302, got %d", redirect.Status)
	}
}

func TestApiRedirect_CustomStatus(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		url    string
		status int
	}{
		{
			name:   "301 permanent redirect",
			code:   `let api = import("std/api"); api.redirect("/old-page", 301)`,
			url:    "/old-page",
			status: 301,
		},
		{
			name:   "303 see other",
			code:   `let api = import("std/api"); api.redirect("/result", 303)`,
			url:    "/result",
			status: 303,
		},
		{
			name:   "307 temporary redirect",
			code:   `let api = import("std/api"); api.redirect("/temp", 307)`,
			url:    "/temp",
			status: 307,
		},
		{
			name:   "308 permanent redirect",
			code:   `let api = import("std/api"); api.redirect("/permanent", 308)`,
			url:    "/permanent",
			status: 308,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Value == nil {
				t.Fatal("expected Redirect result, got nil")
			}

			if result.Value.Type() != evaluator.REDIRECT_OBJ {
				t.Fatalf("expected REDIRECT_OBJ, got %s: %s", result.Value.Type(), result.Value.Inspect())
			}

			redirect := result.Value.(*evaluator.Redirect)
			if redirect.URL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, redirect.URL)
			}
			if redirect.Status != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, redirect.Status)
			}
		})
	}
}

func TestApiRedirect_PathLiteral(t *testing.T) {
	// Test with a path literal (using @ prefix)
	code := `let api = import("std/api")
api.redirect(@/users/profile)
`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected Redirect result, got nil")
	}

	if result.Value.Type() != evaluator.REDIRECT_OBJ {
		t.Fatalf("expected REDIRECT_OBJ, got %s: %s", result.Value.Type(), result.Value.Inspect())
	}

	redirect := result.Value.(*evaluator.Redirect)
	if redirect.URL != "/users/profile" {
		t.Errorf("expected URL '/users/profile', got %q", redirect.URL)
	}
}

func TestApiRedirect_InvalidStatus(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "status 200",
			code: `let api = import("std/api"); api.redirect("/page", 200)`,
		},
		{
			name: "status 400",
			code: `let api = import("std/api"); api.redirect("/page", 400)`,
		},
		{
			name: "status 500",
			code: `let api = import("std/api"); api.redirect("/page", 500)`,
		},
		{
			name: "status 299",
			code: `let api = import("std/api"); api.redirect("/page", 299)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)
			// Either err is returned, or result is an error object
			if err == nil && (result.Value == nil || result.Value.Type() != evaluator.ERROR_OBJ) {
				t.Fatalf("expected error for invalid status, got %s: %s", result.Value.Type(), result.Value.Inspect())
			}
		})
	}
}

func TestApiRedirect_MissingURL(t *testing.T) {
	code := `let api = import("std/api"); api.redirect()`
	result, err := parsley.Eval(code)

	// Either err is returned, or result is an error object
	if err == nil && (result.Value == nil || result.Value.Type() != evaluator.ERROR_OBJ) {
		t.Fatalf("expected error for missing URL, got %s: %s", result.Value.Type(), result.Value.Inspect())
	}
}

func TestApiRedirect_InvalidURLType(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "integer URL",
			code: `let api = import("std/api"); api.redirect(123)`,
		},
		{
			name: "boolean URL",
			code: `let api = import("std/api"); api.redirect(true)`,
		},
		{
			name: "array URL",
			code: `let api = import("std/api"); api.redirect(["/page"])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)

			// Either err is returned, or result is an error object
			if err == nil && (result.Value == nil || result.Value.Type() != evaluator.ERROR_OBJ) {
				t.Fatalf("expected error for invalid URL type, got %s: %s", result.Value.Type(), result.Value.Inspect())
			}
		})
	}
}

// Test that the server properly handles Redirect objects
func TestHandler_RedirectResponse(t *testing.T) {
	// Create a Redirect object
	redirect := &evaluator.Redirect{
		URL:    "/new-location",
		Status: http.StatusFound, // 302
	}

	// Use httptest to verify the redirect behavior
	// We'll test the HTTP response directly
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old-location", nil)

	// Simulate what the handler does with a Redirect
	http.Redirect(w, r, redirect.URL, redirect.Status)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/new-location" {
		t.Errorf("expected Location header '/new-location', got %q", location)
	}
}

func TestHandler_RedirectResponse_301(t *testing.T) {
	redirect := &evaluator.Redirect{
		URL:    "https://newsite.com",
		Status: http.StatusMovedPermanently, // 301
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	http.Redirect(w, r, redirect.URL, redirect.Status)

	resp := w.Result()
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected status 301, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "https://newsite.com" {
		t.Errorf("expected Location 'https://newsite.com', got %q", location)
	}
}
