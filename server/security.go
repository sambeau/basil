package server

import (
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/sambeau/basil/server/config"
)

// securityHeaders wraps an http.Handler to add security headers to all responses.
type securityHeaders struct {
	handler http.Handler
	cfg     config.SecurityConfig
	devMode bool
}

// newSecurityHeaders creates a middleware that adds security headers.
func newSecurityHeaders(handler http.Handler, cfg config.SecurityConfig, devMode bool) http.Handler {
	return &securityHeaders{
		handler: handler,
		cfg:     cfg,
		devMode: devMode,
	}
}

// ServeHTTP implements http.Handler, adding security headers before delegating.
func (s *securityHeaders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()

	// In dev mode, disable browser caching to ensure fresh content on every request
	if s.devMode {
		h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")
	}

	// HSTS - tell browsers to always use HTTPS (skip in dev mode)
	if !s.devMode && s.cfg.HSTS.Enabled {
		hstsValue := "max-age=" + s.cfg.HSTS.MaxAge
		if s.cfg.HSTS.IncludeSubDomains {
			hstsValue += "; includeSubDomains"
		}
		if s.cfg.HSTS.Preload {
			hstsValue += "; preload"
		}
		h.Set("Strict-Transport-Security", hstsValue)
	}

	// Content-Type-Options - prevent MIME-sniffing
	if s.cfg.ContentTypeOptions != "" {
		h.Set("X-Content-Type-Options", s.cfg.ContentTypeOptions)
	}

	// Frame-Options - clickjacking protection
	if s.cfg.FrameOptions != "" {
		h.Set("X-Frame-Options", s.cfg.FrameOptions)
	}

	// XSS-Protection - legacy XSS filter (for older browsers)
	if s.cfg.XSSProtection != "" {
		h.Set("X-XSS-Protection", s.cfg.XSSProtection)
	}

	// Referrer-Policy - control referrer information
	if s.cfg.ReferrerPolicy != "" {
		h.Set("Referrer-Policy", s.cfg.ReferrerPolicy)
	}

	// Content-Security-Policy
	if s.cfg.CSP != "" {
		h.Set("Content-Security-Policy", s.cfg.CSP)
	}

	// Permissions-Policy (formerly Feature-Policy)
	if s.cfg.PermissionsPolicy != "" {
		h.Set("Permissions-Policy", s.cfg.PermissionsPolicy)
	}

	s.handler.ServeHTTP(w, r)
}

// proxyAware wraps an http.Handler to extract the real client IP from proxy headers.
type proxyAware struct {
	handler    http.Handler
	trusted    bool
	trustedIPs map[string]bool
}

// newProxyAware creates a middleware that handles proxy headers.
func newProxyAware(handler http.Handler, cfg config.ProxyConfig) http.Handler {
	if !cfg.Trusted {
		return handler // No proxy handling needed
	}

	trustedIPs := make(map[string]bool)
	for _, ip := range cfg.TrustedIPs {
		trustedIPs[ip] = true
	}

	return &proxyAware{
		handler:    handler,
		trusted:    cfg.Trusted,
		trustedIPs: trustedIPs,
	}
}

// ServeHTTP implements http.Handler, extracting real client IP from proxy headers.
func (p *proxyAware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If we have a trusted IPs list, verify the direct connection is from one
	if len(p.trustedIPs) > 0 {
		remoteIP := extractIP(r.RemoteAddr)
		if !p.trustedIPs[remoteIP] {
			// Not from a trusted proxy, don't trust forwarded headers
			p.handler.ServeHTTP(w, r)
			return
		}
	}

	// Extract real IP from X-Forwarded-For or X-Real-IP
	realIP := p.getRealIP(r)
	if realIP != "" {
		// Store original RemoteAddr and replace with real IP
		r.Header.Set("X-Original-Remote-Addr", r.RemoteAddr)
		r.RemoteAddr = realIP
	}

	p.handler.ServeHTTP(w, r)
}

// getRealIP extracts the real client IP from proxy headers.
func (p *proxyAware) getRealIP(r *http.Request) string {
	// X-Forwarded-For is a comma-separated list of IPs, leftmost is original client
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			// Return first IP (the original client)
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP is a single IP set by some proxies
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	return ""
}

// extractIP extracts just the IP address from an address:port string.
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr // Return as-is if not host:port format
	}
	return host
}

// ClientIP extracts the client IP from a request, considering proxy headers if configured.
// This is useful for logging and rate limiting.
func ClientIP(r *http.Request, proxyCfg config.ProxyConfig) string {
	if proxyCfg.Trusted {
		// Check if request came through trusted proxy
		if len(proxyCfg.TrustedIPs) > 0 {
			remoteIP := extractIP(r.RemoteAddr)
			trusted := slices.Contains(proxyCfg.TrustedIPs, remoteIP)
			if !trusted {
				return extractIP(r.RemoteAddr)
			}
		}

		// Trust X-Forwarded-For
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}

		// Trust X-Real-IP
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	return extractIP(r.RemoteAddr)
}
