// Package server tests for Basil web server.
//
// This file tests cookie handling functionality implemented in handler.go
// (cookie setting, parsing, security attributes via basil.http.response.cookies).
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
	"github.com/sambeau/basil/server/config"
)

func TestBuildRequestContext_Cookies(t *testing.T) {
	tests := []struct {
		name     string
		cookies  []*http.Cookie
		expected map[string]string
	}{
		{
			name:     "no cookies",
			cookies:  nil,
			expected: map[string]string{},
		},
		{
			name: "single cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			expected: map[string]string{"session": "abc123"},
		},
		{
			name: "multiple cookies",
			cookies: []*http.Cookie{
				{Name: "theme", Value: "dark"},
				{Name: "lang", Value: "en"},
				{Name: "remember", Value: "true"},
			},
			expected: map[string]string{"theme": "dark", "lang": "en", "remember": "true"},
		},
		{
			name: "cookie with special characters",
			cookies: []*http.Cookie{
				{Name: "data", Value: "hello%20world"},
			},
			expected: map[string]string{"data": "hello%20world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for _, c := range tt.cookies {
				req.AddCookie(c)
			}

			route := config.Route{Path: "/test"}
			ctx := buildRequestContext(req, route)

			cookies, ok := ctx["cookies"].(map[string]any)
			if !ok {
				t.Fatal("cookies should be a map")
			}

			if len(cookies) != len(tt.expected) {
				t.Errorf("expected %d cookies, got %d", len(tt.expected), len(cookies))
			}

			for name, expected := range tt.expected {
				got, ok := cookies[name].(string)
				if !ok {
					t.Errorf("cookie %q should be a string", name)
					continue
				}
				if got != expected {
					t.Errorf("cookie %q: expected %q, got %q", name, expected, got)
				}
			}
		})
	}
}

func TestBuildCookie_SimpleString(t *testing.T) {
	// Test simple string value with dev mode
	cookie := buildCookie("theme", "dark", true)
	if cookie.Name != "theme" {
		t.Errorf("expected name 'theme', got %q", cookie.Name)
	}
	if cookie.Value != "dark" {
		t.Errorf("expected value 'dark', got %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("expected path '/', got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", cookie.SameSite)
	}
	// In dev mode, Secure should default to false
	if cookie.Secure {
		t.Error("expected Secure to be false in dev mode")
	}

	// Test simple string value with prod mode
	cookie = buildCookie("theme", "dark", false)
	if !cookie.Secure {
		t.Error("expected Secure to be true in prod mode")
	}
}

func TestBuildCookie_WithOptions(t *testing.T) {
	opts := map[string]any{
		"value":    "token123",
		"path":     "/admin",
		"domain":   "example.com",
		"secure":   true,
		"httpOnly": false,
		"sameSite": "Strict",
		"maxAge":   int64(3600),
	}

	cookie := buildCookie("session", opts, true)
	if cookie.Value != "token123" {
		t.Errorf("expected value 'token123', got %q", cookie.Value)
	}
	if cookie.Path != "/admin" {
		t.Errorf("expected path '/admin', got %q", cookie.Path)
	}
	if cookie.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got %q", cookie.Domain)
	}
	if !cookie.Secure {
		t.Error("expected Secure to be true")
	}
	if cookie.HttpOnly {
		t.Error("expected HttpOnly to be false")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite Strict, got %v", cookie.SameSite)
	}
	if cookie.MaxAge != 3600 {
		t.Errorf("expected MaxAge 3600, got %d", cookie.MaxAge)
	}
}

func TestBuildCookie_SameSiteNone(t *testing.T) {
	// SameSite=None should force Secure=true
	opts := map[string]any{
		"value":    "cross-site-data",
		"sameSite": "None",
		"secure":   false, // Explicitly set to false
	}

	cookie := buildCookie("xsite", opts, true)
	if cookie.SameSite != http.SameSiteNoneMode {
		t.Errorf("expected SameSite None, got %v", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Error("SameSite=None should force Secure=true")
	}
}

func TestBuildCookie_Duration(t *testing.T) {
	// Test with duration dict (like what Parsley produces)
	durationDict := map[string]any{
		"days":         int64(7),
		"hours":        int64(0),
		"minutes":      int64(0),
		"seconds":      int64(0),
		"totalHours":   int64(168),
		"totalMinutes": int64(10080),
		"totalSeconds": int64(604800),
		"kind":         "duration",
	}

	opts := map[string]any{
		"value":  "remember-token",
		"maxAge": durationDict,
	}

	cookie := buildCookie("remember", opts, true)
	if cookie.MaxAge != 604800 {
		t.Errorf("expected MaxAge 604800 (7 days), got %d", cookie.MaxAge)
	}
}

func TestBuildCookie_Expires(t *testing.T) {
	// Test with datetime dict (like what Parsley produces)
	futureTime := time.Now().Add(24 * time.Hour)
	datetimeDict := map[string]any{
		"year":   int64(futureTime.Year()),
		"month":  int64(futureTime.Month()),
		"day":    int64(futureTime.Day()),
		"hour":   int64(futureTime.Hour()),
		"minute": int64(futureTime.Minute()),
		"second": int64(futureTime.Second()),
		"kind":   "datetime",
		"unix":   futureTime.Unix(),
	}

	opts := map[string]any{
		"value":   "session-token",
		"expires": datetimeDict,
	}

	cookie := buildCookie("session", opts, true)
	// Check that expires is set within a reasonable range
	if cookie.Expires.IsZero() {
		t.Error("expected expires to be set")
	}
	// Allow 1 second tolerance
	if cookie.Expires.Unix() < futureTime.Unix()-1 || cookie.Expires.Unix() > futureTime.Unix()+1 {
		t.Errorf("expected expires near %v, got %v", futureTime, cookie.Expires)
	}
}

func TestBuildCookie_DeleteCookie(t *testing.T) {
	// Setting maxAge to 0 should delete the cookie
	opts := map[string]any{
		"value":  "",
		"maxAge": int64(0),
	}

	cookie := buildCookie("old_cookie", opts, true)
	if cookie.MaxAge != 0 {
		t.Errorf("expected MaxAge 0, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Errorf("expected empty value, got %q", cookie.Value)
	}
}

func TestDurationToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{
			name:     "int64",
			input:    int64(3600),
			expected: 3600,
		},
		{
			name:     "int",
			input:    3600,
			expected: 3600,
		},
		{
			name:     "float64",
			input:    float64(3600.5),
			expected: 3600,
		},
		{
			name: "duration dict with totalSeconds",
			input: map[string]any{
				"days":         int64(7),
				"hours":        int64(0),
				"minutes":      int64(0),
				"seconds":      int64(0),
				"totalSeconds": int64(604800),
				"kind":         "duration",
			},
			expected: 604800,
		},
		{
			name: "duration dict with seconds only",
			input: map[string]any{
				"seconds": int64(3600),
			},
			expected: 3600,
		},
		{
			name: "duration dict with months (approximation)",
			input: map[string]any{
				"months":  int64(1),
				"seconds": int64(0),
			},
			expected: 30 * 24 * 60 * 60, // ~30 days
		},
		{
			name:     "unknown type",
			input:    "not a duration",
			expected: 0,
		},
		{
			name:     "nil",
			input:    nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := durationToSeconds(tt.input)
			if got != tt.expected {
				t.Errorf("durationToSeconds(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractResponseMeta_Cookies(t *testing.T) {
	// Create an environment with basil.http.response.cookies
	env := evaluator.NewEnvironment()

	basilMap := map[string]any{
		"http": map[string]any{
			"request": map[string]any{},
			"response": map[string]any{
				"status":  int64(200),
				"headers": map[string]any{},
				"cookies": map[string]any{
					"theme": "dark",
					"session": map[string]any{
						"value":    "abc123",
						"maxAge":   int64(86400),
						"httpOnly": true,
						"secure":   true,
						"sameSite": "Strict",
					},
				},
			},
		},
	}

	basilObj, err := parsley.ToParsley(basilMap)
	if err != nil {
		t.Fatalf("failed to convert basil map: %v", err)
	}
	env.Set("basil", basilObj)

	meta := extractResponseMeta(env, true)

	if len(meta.cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(meta.cookies))
	}

	// Find the cookies by name
	var themeCookie, sessionCookie *http.Cookie
	for _, c := range meta.cookies {
		switch c.Name {
		case "theme":
			themeCookie = c
		case "session":
			sessionCookie = c
		}
	}

	if themeCookie == nil {
		t.Error("expected theme cookie")
	} else {
		if themeCookie.Value != "dark" {
			t.Errorf("theme cookie value: expected 'dark', got %q", themeCookie.Value)
		}
	}

	if sessionCookie == nil {
		t.Error("expected session cookie")
	} else {
		if sessionCookie.Value != "abc123" {
			t.Errorf("session cookie value: expected 'abc123', got %q", sessionCookie.Value)
		}
		if sessionCookie.MaxAge != 86400 {
			t.Errorf("session cookie MaxAge: expected 86400, got %d", sessionCookie.MaxAge)
		}
		if !sessionCookie.HttpOnly {
			t.Error("session cookie should be HttpOnly")
		}
		if !sessionCookie.Secure {
			t.Error("session cookie should be Secure")
		}
		if sessionCookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("session cookie SameSite: expected Strict, got %v", sessionCookie.SameSite)
		}
	}
}

func TestCookieDefaults_DevVsProd(t *testing.T) {
	// Test that secure defaults differ between dev and prod mode
	opts := map[string]any{
		"value": "test",
	}

	// Dev mode: Secure defaults to false
	devCookie := buildCookie("test", opts, true)
	if devCookie.Secure {
		t.Error("dev mode: expected Secure to default to false")
	}

	// Prod mode: Secure defaults to true
	prodCookie := buildCookie("test", opts, false)
	if !prodCookie.Secure {
		t.Error("prod mode: expected Secure to default to true")
	}

	// Both should default to HttpOnly=true
	if !devCookie.HttpOnly {
		t.Error("dev mode: expected HttpOnly to default to true")
	}
	if !prodCookie.HttpOnly {
		t.Error("prod mode: expected HttpOnly to default to true")
	}

	// Both should default to SameSite=Lax
	if devCookie.SameSite != http.SameSiteLaxMode {
		t.Error("dev mode: expected SameSite to default to Lax")
	}
	if prodCookie.SameSite != http.SameSiteLaxMode {
		t.Error("prod mode: expected SameSite to default to Lax")
	}
}
