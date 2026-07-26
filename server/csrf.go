package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	// CSRFCookieName is the name of the cookie storing the CSRF token
	CSRFCookieName = "_csrf"

	// CSRFFormField is the form field name for the CSRF token
	CSRFFormField = "_csrf"

	// CSRFHeaderName is the header name for CSRF tokens in AJAX requests
	CSRFHeaderName = "X-CSRF-Token"

	// csrfTokenLength is the number of bytes of randomness (32 bytes = 64 hex chars)
	csrfTokenLength = 32
)

// GenerateCSRFToken generates a new random CSRF token.
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetCSRFToken retrieves the CSRF token from the request cookie, or generates a new one.
// Returns the token and a boolean indicating if a new token was generated.
func GetCSRFToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(CSRFCookieName)
	if err == nil && cookie.Value != "" && len(cookie.Value) == csrfTokenLength*2 {
		return cookie.Value, false
	}

	// Generate new token
	token, err := GenerateCSRFToken()
	if err != nil {
		// Fallback to empty token on error (will fail validation)
		return "", true
	}
	return token, true
}

// SetCSRFCookie sets the CSRF token cookie on the response.
func SetCSRFCookie(w http.ResponseWriter, token string, devMode bool) {
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	// Secure cookie in production
	if !devMode {
		cookie.Secure = true
	}

	http.SetCookie(w, cookie)
}

// ValidateCSRF checks that the CSRF token from the cookie matches the one in the form/header.
// Returns true if valid, false otherwise.
func ValidateCSRF(r *http.Request) bool {
	// Get token from cookie
	cookieToken, err := r.Cookie(CSRFCookieName)
	if err != nil || cookieToken.Value == "" {
		return false
	}

	// Get token from form field or header
	submittedToken := r.FormValue(CSRFFormField)
	if submittedToken == "" {
		submittedToken = r.Header.Get(CSRFHeaderName)
	}

	if submittedToken == "" {
		return false
	}

	// Constant-time comparison to prevent timing attacks
	return secureCompare(cookieToken.Value, submittedToken)
}

// secureCompare performs a constant-time string comparison to prevent timing attacks.
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// CSRFMiddleware wraps an http.Handler to validate CSRF tokens on mutating requests.
// It only validates for POST, PUT, PATCH, DELETE requests.
type CSRFMiddleware struct {
	devMode bool
	// server is optional. When set, production CSRF failures render the 403
	// error page (including a custom one from error_pages) instead of plain
	// text. Nil in unit tests that exercise the middleware in isolation.
	server *Server
}

// NewCSRFMiddleware creates a new CSRF middleware.
func NewCSRFMiddleware(devMode bool) *CSRFMiddleware {
	return &CSRFMiddleware{devMode: devMode}
}

// Validate returns middleware that validates CSRF tokens for mutating requests.
func (m *CSRFMiddleware) Validate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate mutating methods
		if isMutatingMethod(r.Method) {
			if !ValidateCSRF(r) {
				m.handleCSRFError(w, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isMutatingMethod returns true for HTTP methods that modify state.
func isMutatingMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

// csrfTokenSummary returns truncated descriptions of the tokens Basil saw, for
// dev-mode diagnostics. Either may be "(missing)".
func csrfTokenSummary(r *http.Request) (cookieToken, submittedToken string) {
	if cookie, err := r.Cookie(CSRFCookieName); err == nil {
		cookieToken = cookie.Value
		if len(cookieToken) > 16 {
			cookieToken = cookieToken[:16] + "..."
		}
	} else {
		cookieToken = "(missing)"
	}

	submittedToken = r.FormValue(CSRFFormField)
	if submittedToken == "" {
		submittedToken = r.Header.Get(CSRFHeaderName)
	}
	if submittedToken == "" {
		submittedToken = "(missing)"
	} else if len(submittedToken) > 16 {
		submittedToken = submittedToken[:16] + "..."
	}

	return cookieToken, submittedToken
}

// handleCSRFError sends a 403 Forbidden response for CSRF validation failures.
// API clients get JSON; browsers get the detailed diagnostic in dev mode and
// the 403 error page in production.
func (m *CSRFMiddleware) handleCSRFError(w http.ResponseWriter, r *http.Request) {
	// API clients get JSON, not an HTML page
	if isAPIRequest(r) {
		errorObj := map[string]any{
			"code":    "HTTP-403",
			"message": "Forbidden: CSRF token validation failed",
		}
		if m.devMode {
			cookieToken, submittedToken := csrfTokenSummary(r)
			errorObj["details"] = "token from cookie: " + cookieToken +
				"; token from form/header: " + submittedToken +
				"; send the token as the _csrf field or the " + CSRFHeaderName + " header"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{"error": errorObj})
		return
	}

	// Dev mode: detailed diagnostic naming which token went missing
	if m.devMode {
		cookieToken, submittedToken := csrfTokenSummary(r)
		html := `<!DOCTYPE html>
<html>
<head><title>403 Forbidden</title></head>
<body>
<h1>403 Forbidden</h1>
<p>CSRF token validation failed.</p>
<ul>
  <li>Token from cookie: ` + cookieToken + `</li>
  <li>Token from form/header: ` + submittedToken + `</li>
</ul>
<p>Make sure your form includes:</p>
<pre>&lt;input type=hidden name=_csrf value={basil.csrf.token}/&gt;</pre>
<p>Or for AJAX requests, include the header:</p>
<pre>X-CSRF-Token: {token}</pre>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(html))
		return
	}

	// Production: the same 403 page a role failure gets (custom page honoured)
	if m.server != nil && m.server.renderPreludeError(w, r, http.StatusForbidden, nil) {
		return
	}

	// Fallback when no error page is available
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("403 Forbidden"))
}
