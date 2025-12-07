package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	githttp "github.com/AaronO/go-git-http"
	"github.com/sambeau/basil/auth"
	"github.com/sambeau/basil/config"
)

// GitHandler wraps the go-git-http handler with authentication.
type GitHandler struct {
	git        *githttp.GitHttp
	authDB     *auth.DB
	config     *config.Config
	onPush     func() // Callback for post-push reload
	stdout     io.Writer
	stderr     io.Writer
	warnedHTTP bool // Track if we've warned about non-TLS
}

// NewGitHandler creates a new Git HTTP handler.
func NewGitHandler(siteDir string, authDB *auth.DB, cfg *config.Config, onPush func(), stdout, stderr io.Writer) (*GitHandler, error) {
	git := githttp.New(siteDir)

	h := &GitHandler{
		git:    git,
		authDB: authDB,
		config: cfg,
		onPush: onPush,
		stdout: stdout,
		stderr: stderr,
	}

	// Set up event handler for post-push reload
	git.EventHandler = func(ev githttp.Event) {
		if ev.Type == githttp.PUSH {
			fmt.Fprintf(stdout, "[git] Push received from %s: %s\n", ev.Request.RemoteAddr, ev.Commit)
			if onPush != nil {
				onPush()
			}
		}
	}

	return h, nil
}

// ServeHTTP handles Git HTTP requests with authentication.
func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Warn once about non-TLS when auth is enabled (credentials exposed in plain text)
	if h.config.Git.RequireAuth && r.TLS == nil && !h.isDevLocalhost(r) && !h.warnedHTTP {
		fmt.Fprintf(h.stderr, "[git] ⚠ WARNING: Git request received over HTTP (not HTTPS). API keys are being sent in plain text!\n")
		h.warnedHTTP = true
	}

	// Check if auth is required
	if h.config.Git.RequireAuth && !h.isDevLocalhost(r) {
		user, ok := h.authenticate(w, r)
		if !ok {
			return // Response already sent
		}

		// Check role for push operations
		if h.isPushRequest(r) {
			if user.Role != auth.RoleAdmin && user.Role != auth.RoleEditor {
				http.Error(w, "Forbidden: editor or admin role required for push", http.StatusForbidden)
				return
			}
		}

		fmt.Fprintf(h.stdout, "[git] %s %s by %s (%s)\n", r.Method, r.URL.Path, user.Name, user.Role)
	} else if h.isDevLocalhost(r) {
		fmt.Fprintf(h.stdout, "[git] %s %s (dev mode, unauthenticated)\n", r.Method, r.URL.Path)
	}

	// Strip /.git prefix before passing to git handler
	// go-git-http expects paths like /info/refs, not /.git/info/refs
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/.git")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}

	h.git.ServeHTTP(w, r)
}

// authenticate extracts and validates HTTP Basic Auth credentials.
// Returns the authenticated user and true if successful, or sends an error response and returns false.
func (h *GitHandler) authenticate(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.sendAuthChallenge(w, "Authentication required")
		return nil, false
	}

	// Parse Basic auth
	if !strings.HasPrefix(authHeader, "Basic ") {
		h.sendAuthChallenge(w, "Basic authentication required")
		return nil, false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		h.sendAuthChallenge(w, "Invalid authorization header")
		return nil, false
	}

	// Split username:password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		h.sendAuthChallenge(w, "Invalid credentials format")
		return nil, false
	}

	// Password field contains the API key
	apiKey := parts[1]

	// Validate API key
	user, err := h.authDB.ValidateAPIKey(apiKey)
	if err != nil {
		fmt.Fprintf(h.stderr, "[git] API key validation error: %v\n", err)
		h.sendAuthChallenge(w, "Authentication failed")
		return nil, false
	}
	if user == nil {
		h.sendAuthChallenge(w, "Invalid API key")
		return nil, false
	}

	return user, true
}

// sendAuthChallenge sends a 401 response with WWW-Authenticate header.
func (h *GitHandler) sendAuthChallenge(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Basil Git"`)
	http.Error(w, message, http.StatusUnauthorized)
}

// isPushRequest returns true if this is a Git push operation.
func (h *GitHandler) isPushRequest(r *http.Request) bool {
	// Push requests go to git-receive-pack
	if strings.Contains(r.URL.Path, "git-receive-pack") {
		return true
	}
	// Also check the service parameter for refs requests
	if strings.Contains(r.URL.RawQuery, "service=git-receive-pack") {
		return true
	}
	return false
}

// isDevLocalhost returns true if we're in dev mode and the request is from localhost.
func (h *GitHandler) isDevLocalhost(r *http.Request) bool {
	if !h.config.Server.Dev {
		return false
	}

	// Get the remote host
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	// Check for localhost addresses
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}
