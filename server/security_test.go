package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sambeau/basil/config"
)

func TestSecurityHeaders_Default(t *testing.T) {
	cfg := config.Defaults().Security

	handler := newSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), cfg, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := rec.Header().Get(tt.header)
			if got != tt.want {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestSecurityHeaders_DevMode(t *testing.T) {
	cfg := config.Defaults().Security

	handler := newSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), cfg, true) // dev mode = true

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// HSTS should be absent in dev mode
	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS should be empty in dev mode, got %q", hsts)
	}

	// Other headers should still be present
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want 'nosniff'", got)
	}
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	cfg := config.SecurityConfig{
		CSP: "default-src 'self'; script-src 'self' 'unsafe-inline'",
	}

	handler := newSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), cfg, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := "default-src 'self'; script-src 'self' 'unsafe-inline'"
	if got := rec.Header().Get("Content-Security-Policy"); got != want {
		t.Errorf("CSP = %q, want %q", got, want)
	}
}

func TestSecurityHeaders_HSTSPreload(t *testing.T) {
	cfg := config.SecurityConfig{
		HSTS: config.HSTSConfig{
			Enabled:           true,
			MaxAge:            "63072000",
			IncludeSubDomains: true,
			Preload:           true,
		},
	}

	handler := newSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), cfg, false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := "max-age=63072000; includeSubDomains; preload"
	if got := rec.Header().Get("Strict-Transport-Security"); got != want {
		t.Errorf("HSTS = %q, want %q", got, want)
	}
}

func TestProxyAware_NotTrusted(t *testing.T) {
	cfg := config.ProxyConfig{
		Trusted: false,
	}

	var gotRemoteAddr string
	handler := newProxyAware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRemoteAddr = r.RemoteAddr
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should NOT trust X-Forwarded-For
	if gotRemoteAddr != "192.168.1.1:54321" {
		t.Errorf("RemoteAddr = %q, want unchanged address", gotRemoteAddr)
	}
}

func TestProxyAware_Trusted(t *testing.T) {
	cfg := config.ProxyConfig{
		Trusted: true,
	}

	var gotRemoteAddr string
	handler := newProxyAware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRemoteAddr = r.RemoteAddr
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should trust X-Forwarded-For and use first IP
	if gotRemoteAddr != "203.0.113.195" {
		t.Errorf("RemoteAddr = %q, want '203.0.113.195'", gotRemoteAddr)
	}
}

func TestProxyAware_TrustedIPsAllowed(t *testing.T) {
	cfg := config.ProxyConfig{
		Trusted:    true,
		TrustedIPs: []string{"10.0.0.1"},
	}

	var gotRemoteAddr string
	handler := newProxyAware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRemoteAddr = r.RemoteAddr
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:8080" // Trusted proxy IP
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should trust because proxy IP is in allowed list
	if gotRemoteAddr != "203.0.113.195" {
		t.Errorf("RemoteAddr = %q, want '203.0.113.195'", gotRemoteAddr)
	}
}

func TestProxyAware_TrustedIPsNotAllowed(t *testing.T) {
	cfg := config.ProxyConfig{
		Trusted:    true,
		TrustedIPs: []string{"10.0.0.1"},
	}

	var gotRemoteAddr string
	handler := newProxyAware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRemoteAddr = r.RemoteAddr
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321" // NOT a trusted proxy IP
	req.Header.Set("X-Forwarded-For", "203.0.113.195")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should NOT trust because proxy IP is not in allowed list
	if gotRemoteAddr != "192.168.1.1:54321" {
		t.Errorf("RemoteAddr = %q, want unchanged '192.168.1.1:54321'", gotRemoteAddr)
	}
}

func TestProxyAware_XRealIP(t *testing.T) {
	cfg := config.ProxyConfig{
		Trusted: true,
	}

	var gotRemoteAddr string
	handler := newProxyAware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRemoteAddr = r.RemoteAddr
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("X-Real-IP", "203.0.113.195") // nginx style
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should trust X-Real-IP
	if gotRemoteAddr != "203.0.113.195" {
		t.Errorf("RemoteAddr = %q, want '203.0.113.195'", gotRemoteAddr)
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		cfg        config.ProxyConfig
		want       string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.1:54321",
			cfg:        config.ProxyConfig{Trusted: false},
			want:       "192.168.1.1",
		},
		{
			name:       "trusted proxy",
			remoteAddr: "10.0.0.1:8080",
			xff:        "203.0.113.195",
			cfg:        config.ProxyConfig{Trusted: true},
			want:       "203.0.113.195",
		},
		{
			name:       "multiple proxies",
			remoteAddr: "10.0.0.1:8080",
			xff:        "203.0.113.195, 70.41.3.18, 10.0.0.1",
			cfg:        config.ProxyConfig{Trusted: true},
			want:       "203.0.113.195",
		},
		{
			name:       "x-real-ip fallback",
			remoteAddr: "10.0.0.1:8080",
			xri:        "203.0.113.195",
			cfg:        config.ProxyConfig{Trusted: true},
			want:       "203.0.113.195",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			got := ClientIP(req, tt.cfg)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"192.168.1.1", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"::1", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := extractIP(tt.addr)
			if got != tt.want {
				t.Errorf("extractIP(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}
