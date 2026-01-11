package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sambeau/basil/server/config"
)

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com"},
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// No Origin header means no CORS headers should be added
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin header for same-origin request")
	}
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com", "https://app.example.com"},
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should have CORS headers
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: https://example.com, got %s", got)
	}
	if got := rr.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Expected Vary: Origin, got %s", got)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com"},
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Origin not in allowed list - no CORS headers
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin header for disallowed origin")
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"*"},
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Wildcard should allow any origin
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", got)
	}
}

func TestCORSMiddleware_Credentials(t *testing.T) {
	cfg := config.CORSConfig{
		Origins:     config.StringOrSlice{"https://example.com"},
		Credentials: true,
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should include credentials header
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials: true, got %s", got)
	}
	// With credentials, must use specific origin not wildcard
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: https://example.com, got %s", got)
	}
}

func TestCORSMiddleware_ExposeHeaders(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com"},
		Expose:  []string{"X-Total-Count", "X-Page-Count"},
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should expose specified headers
	if got := rr.Header().Get("Access-Control-Expose-Headers"); got != "X-Total-Count, X-Page-Count" {
		t.Errorf("Expected Access-Control-Expose-Headers: X-Total-Count, X-Page-Count, got %s", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com"},
		Methods: []string{"GET", "POST", "DELETE"},
		Headers: []string{"Content-Type", "Authorization"},
		MaxAge:  86400,
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return 204
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}

	// Should have all preflight headers
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, DELETE" {
		t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, DELETE, got %s", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Errorf("Expected Access-Control-Allow-Headers: Content-Type, Authorization, got %s", got)
	}
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Expected Access-Control-Max-Age: 86400, got %s", got)
	}
}

func TestCORSMiddleware_PreflightEchoHeaders(t *testing.T) {
	cfg := config.CORSConfig{
		Origins: config.StringOrSlice{"https://example.com"},
		Methods: []string{"GET", "POST"},
		// No Headers specified - should echo requested headers
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Headers", "X-Custom-Header, X-Another-Header")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should echo back the requested headers
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "X-Custom-Header, X-Another-Header" {
		t.Errorf("Expected echoed headers, got %s", got)
	}
}

func TestCORSMiddleware_Disabled(t *testing.T) {
	cfg := config.CORSConfig{
		// No origins configured - CORS disabled
	}
	mw := NewCORSMiddleware(cfg)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// No CORS headers when disabled
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS headers when CORS is disabled")
	}
}
