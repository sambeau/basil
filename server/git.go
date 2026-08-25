package server

// The Git transport (FEAT-154): Smart HTTP over the site's bare repository,
// served at /.git. Basil speaks the stateless-RPC protocol itself — two
// endpoints, four exec invocations — rather than through go-git-http, for one
// load-bearing reason: the authenticated account name must reach the receive
// hooks as BASIL_PUBLISHER *per request* (two people pushing concurrently
// must each be recorded as themselves), and go-git-http offers no way to set
// environment on the git it execs. Once receive-pack is served by hand,
// serving upload-pack through the same two helpers is smaller than keeping
// the dependency — and drops its hardcoded /usr/bin/git and its dumb-protocol
// raw-file serving of the repository, which nothing needs.

import (
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/sambeau/basil/server/auth"
	"github.com/sambeau/basil/server/config"
)

// maxDecompressedPush bounds how far a gzipped RPC body may inflate before it
// reaches git's stdin. A gzip bomb is a tiny POST body that decompresses
// without limit; unbounded, it drives arbitrary CPU/IO in the git child. The
// cap is deliberately generous — far larger than any real push's smart-protocol
// stream (the pack it carries is already compressed, so gzip barely shrinks it)
// — so it never trips a legitimate push. This is only the decompression bound;
// a total-push-size limit is a separate, deferred concern (spec Open Q#2).
const maxDecompressedPush = 1 << 30 // 1 GiB

// errPushTooLarge is the refusal when a decompressed RPC body passes
// maxDecompressedPush.
var errPushTooLarge = errors.New("git RPC body exceeded the decompressed size limit")

// limitReader wraps r so that reading more than limit bytes in total fails with
// errPushTooLarge. It is deliberately NOT io.LimitReader: that reports a clean
// EOF at the cap, silently truncating the stream — a git stream cut at an
// arbitrary point must be an error, never a short read git could mistake for a
// complete (tiny) request.
type limitReader struct {
	r     io.Reader
	limit int64
	read  int64
}

func (l *limitReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.read += int64(n)
	if l.read > l.limit {
		return n, errPushTooLarge
	}
	return n, err
}

// publisherEnv is the environment variable the receive hooks read to record
// who pushed (cmd/basil/fromhook.go, DESIGN-git-deploy D20).
const publisherEnv = "BASIL_PUBLISHER"

// GitHandler serves the bare repository over Smart HTTP with authentication.
type GitHandler struct {
	repoDir string // the bare repository (<site root>/site.git)
	gitPath string // resolved git binary
	authDB  *auth.DB
	config  *config.Config
	stdout  io.Writer
	stderr  io.Writer
}

// NewGitHandler creates a Git Smart HTTP handler for the bare repository at
// repoDir.
func NewGitHandler(repoDir string, authDB *auth.DB, cfg *config.Config, stdout, stderr io.Writer) (*GitHandler, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("the git binary is required to serve /.git: %w", err)
	}
	return &GitHandler{
		repoDir: repoDir,
		gitPath: gitPath,
		authDB:  authDB,
		config:  cfg,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// ServeHTTP handles Git Smart HTTP requests with authentication.
func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Guard against path traversal. The router below only matches exact
	// paths, so this is defence in depth, not the only wall.
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	dev := h.isDevLocalhost(r)

	// Refuse Git over plain HTTP: Basic auth carries the API key in an
	// easily-decoded header, so a plain-HTTP request means a plaintext
	// credential with push rights on the wire. A warning in a log nobody
	// reads is not a control (the old behaviour); refusal is. The sole
	// exception is a dev-mode localhost bind, decided in code.
	if r.TLS == nil && !dev {
		http.Error(w, "Git over plain HTTP is refused: HTTP Basic authentication would send the API key unencrypted. Use https:// (Basil obtains its own certificate), or a --dev server on localhost.", http.StatusForbidden)
		return
	}

	// The requested git service, decided ONCE from the decoded request. The
	// role gate and the advertisement/RPC routing below both read this single
	// value, so they cannot disagree under percent-encoding.
	service, isGit := h.gitService(r)

	// publisher is the authenticated account name, exported to the receive
	// hooks as BASIL_PUBLISHER so the deploy record names who pushed.
	var publisher string

	if !dev {
		// No auth database, no Git — ever. Startup refuses to serve Git in
		// this state (initGit); this is the belt to that brace.
		if h.authDB == nil {
			http.Error(w, "Git is unavailable: this server has no auth database, and Git access always requires an API key (auth.enabled: true).", http.StatusForbidden)
			return
		}

		user, ok := h.authenticate(w, r)
		if !ok {
			return // response already sent
		}
		// The write/push gate applies whenever the decoded service is
		// receive-pack — on the info/refs advertisement as well as the RPC.
		if service == "receive-pack" && user.Role != auth.RoleAdmin && user.Role != auth.RoleEditor {
			http.Error(w, fmt.Sprintf("Forbidden: pushing requires the editor or admin role (%s has the %s role)", user.Name, user.Role), http.StatusForbidden)
			return
		}
		publisher = user.Name
		fmt.Fprintf(h.stdout, "[git] %s %s by %s (%s)\n", r.Method, r.URL.Path, user.Name, user.Role)
	} else {
		// Dev-mode localhost skips authentication, but credentials offered
		// anyway still name the publisher in the deploy record.
		if h.authDB != nil {
			if user := h.authenticateOptional(r); user != nil {
				publisher = user.Name
			}
		}
		fmt.Fprintf(h.stdout, "[git] %s %s (dev mode, localhost)\n", r.Method, r.URL.Path)
	}

	// Route. The mux mounts us at /.git/; everything the Smart HTTP
	// protocol needs is exactly these paths, so anything else is 404 — the
	// repository's files themselves are never served. Both branches route on
	// the same decoded service the gate used above.
	path := strings.TrimPrefix(r.URL.Path, "/.git")
	switch {
	case r.Method == http.MethodGet && path == "/info/refs":
		if !isGit {
			// The dumb HTTP protocol (no/unknown service) is deliberately not
			// served: every git since 2010 speaks Smart HTTP.
			http.Error(w, "smart HTTP is required: ?service=git-upload-pack or ?service=git-receive-pack", http.StatusForbidden)
			return
		}
		h.advertiseRefs(w, r, service)
	case r.Method == http.MethodPost && path == "/git-upload-pack":
		h.serviceRPC(w, r, "upload-pack", "")
	case r.Method == http.MethodPost && path == "/git-receive-pack":
		h.serviceRPC(w, r, "receive-pack", publisher)
	default:
		http.NotFound(w, r)
	}
}

// advertiseRefs answers GET /info/refs?service=git-{upload,receive}-pack —
// the ref advertisement that starts every Smart HTTP exchange.
func (h *GitHandler) advertiseRefs(w http.ResponseWriter, r *http.Request, service string) {
	// service is the decoded, validated value computed in ServeHTTP — the
	// same one the role gate read, so advertisement and gate never diverge.
	cmd := exec.CommandContext(r.Context(), h.gitPath, service, "--stateless-rpc", "--advertise-refs", ".")
	cmd.Dir = h.repoDir
	cmd.Stderr = h.stderr
	refs, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(h.stderr, "[git] %s --advertise-refs: %v\n", service, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hdrNoCache(w)
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	w.Write(pktLine("# service=git-" + service + "\n"))
	w.Write([]byte("0000")) // pkt-line flush
	w.Write(refs)
}

// serviceRPC answers POST /git-{upload,receive}-pack: the request body is the
// client's protocol stream, piped to `git <service> --stateless-rpc`, whose
// stdout streams back as the response. For receive-pack, publisher (when
// known) is exported as BASIL_PUBLISHER in this invocation's environment
// only, so concurrent pushes by different accounts each record their own
// name.
func (h *GitHandler) serviceRPC(w http.ResponseWriter, r *http.Request, service, publisher string) {
	if ct := r.Header.Get("Content-Type"); ct != fmt.Sprintf("application/x-git-%s-request", service) {
		http.Error(w, fmt.Sprintf("expected Content-Type application/x-git-%s-request", service), http.StatusBadRequest)
		return
	}

	body := io.Reader(r.Body)
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		defer gz.Close()
		// Bound the DECOMPRESSED stream (not the compressed body): a gzip bomb
		// is small on the wire and huge once inflated.
		body = &limitReader{r: gz, limit: maxDecompressedPush}
	}

	// The hooks inherit this environment. Any BASIL_PUBLISHER the server
	// process itself carries is stripped first: identity comes from this
	// request's credentials or not at all.
	env := environWithout(publisherEnv + "=")
	if service == "receive-pack" && publisher != "" {
		env = append(env, publisherEnv+"="+publisher)
	}

	cmd := exec.CommandContext(r.Context(), h.gitPath, service, "--stateless-rpc", ".")
	cmd.Dir = h.repoDir
	cmd.Env = env
	cmd.Stdin = body
	cmd.Stdout = flushingWriter(w)
	cmd.Stderr = h.stderr

	hdrNoCache(w)
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

	if err := cmd.Run(); err != nil {
		// The response is (usually) already streaming, so the status cannot
		// change; a push rejection travels inside the protocol stream, not
		// here. This is real failure — git itself dying.
		fmt.Fprintf(h.stderr, "[git] %s: %v\n", service, err)
	}
}

// pktLine encodes one pkt-line: 4 hex digits of total length, then the data.
func pktLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}

// hdrNoCache marks a response uncacheable, as the Smart HTTP spec requires
// for the advertisement and RPC endpoints.
func hdrNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
}

// environWithout is os.Environ() minus any variable with the given
// "NAME=" prefix.
func environWithout(prefix string) []string {
	all := os.Environ()
	env := make([]string, 0, len(all)+1)
	for _, kv := range all {
		if strings.HasPrefix(kv, prefix) {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// flushingWriter flushes after every write when the ResponseWriter supports
// it, so hook output ("remote:" lines) reaches the developer's terminal while
// the push is still running, not in one burst at the end.
func flushingWriter(w http.ResponseWriter) io.Writer {
	f, ok := w.(http.Flusher)
	if !ok {
		return w
	}
	return &flushWriter{w: w, f: f}
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if n > 0 {
		fw.f.Flush()
	}
	return n, err
}

// authenticate extracts and validates HTTP Basic Auth credentials.
// Returns the authenticated user and true if successful, or sends an error
// response and returns false.
func (h *GitHandler) authenticate(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	apiKey, ok := basicAuthPassword(r)
	if !ok {
		h.sendAuthChallenge(w, "Authentication required")
		return nil, false
	}

	// The password field carries the API key; the username selects a stored
	// credential on the client and is ignored here.
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

// authenticateOptional validates credentials if the request carries any,
// without ever challenging: dev-localhost requests are served either way,
// but a valid key still names the publisher.
func (h *GitHandler) authenticateOptional(r *http.Request) *auth.User {
	apiKey, ok := basicAuthPassword(r)
	if !ok {
		return nil
	}
	user, err := h.authDB.ValidateAPIKey(apiKey)
	if err != nil {
		return nil
	}
	return user
}

// basicAuthPassword extracts the password from an HTTP Basic Authorization
// header, if one is present and well-formed.
func basicAuthPassword(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return "", false
	}
	_, password, found := strings.Cut(string(decoded), ":")
	if !found {
		return "", false
	}
	return password, true
}

// sendAuthChallenge sends a 401 response with WWW-Authenticate header.
func (h *GitHandler) sendAuthChallenge(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Basil Git"`)
	http.Error(w, message, http.StatusUnauthorized)
}

// gitService identifies which git service a request drives, from the DECODED
// request — the query is parsed by net/url, not substring-matched on the raw
// bytes. This is the single source of truth for BOTH the role gate and the
// advertisement/RPC routing: deriving them separately let the two disagree
// under percent-encoding (?service=git-receive%2dpack decodes to receive-pack
// for the advertisement but slips a raw substring match, skipping the gate).
//
// It returns the service ("upload-pack" or "receive-pack") and whether the
// request is a recognised smart-HTTP git request at all; anything else (an
// unknown or missing service) returns ok=false and is refused by the caller.
func (h *GitHandler) gitService(r *http.Request) (service string, ok bool) {
	path := strings.TrimPrefix(r.URL.Path, "/.git")
	switch {
	case r.Method == http.MethodGet && path == "/info/refs":
		service = strings.TrimPrefix(r.URL.Query().Get("service"), "git-")
	case r.Method == http.MethodPost && path == "/git-upload-pack":
		service = "upload-pack"
	case r.Method == http.MethodPost && path == "/git-receive-pack":
		service = "receive-pack"
	default:
		return "", false
	}
	if service != "upload-pack" && service != "receive-pack" {
		return "", false
	}
	return service, true
}

// isDevLocalhost returns true if we're in dev mode and the request is from
// localhost. This is the one relaxation of transport and auth requirements,
// and it is decided here in code, never from configuration.
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
