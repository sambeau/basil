package server

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
)

// recoverMiddleware wraps an http.Handler so that a panic in any downstream
// handler (or inner middleware) is caught, logged, and turned into a 500 response
// instead of tearing down the connection — which, for the client, appears as a
// dropped request with no status.
//
// Note: this cannot catch fatal runtime conditions such as a stack overflow —
// those are not recoverable in Go. The evaluator's own call-depth guard
// (evaluator.MaxCallDepth) is what prevents runaway Parsley recursion from ever
// reaching that fatal state; this middleware handles the ordinary panics
// (nil dereference, out-of-range, etc.) that can still surface from the large
// request-handling surface.
type recoverMiddleware struct {
	handler http.Handler
	stderr  io.Writer
	devMode bool
}

// newRecoverMiddleware creates panic-recovery middleware. Install it as the
// outermost layer so it also guards the other middleware.
func newRecoverMiddleware(handler http.Handler, stderr io.Writer, devMode bool) http.Handler {
	return &recoverMiddleware{handler: handler, stderr: stderr, devMode: devMode}
}

func (m *recoverMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &recoverWriter{ResponseWriter: w}

	defer func() {
		rec := recover()
		if rec == nil {
			return
		}

		// http.ErrAbortHandler is the sanctioned way for a handler to abort a
		// response; re-panic so net/http handles it as intended rather than
		// treating it as an application error.
		if rec == http.ErrAbortHandler {
			panic(rec)
		}

		stack := debug.Stack()
		_, _ = fmt.Fprintf(m.stderr, "[ERROR] panic recovered: %s %s -> %v\n%s\n",
			r.Method, r.URL.Path, rec, stack)

		// If the response was already started we cannot change the status; the
		// connection will be closed by net/http. Just stop here.
		if rw.wroteHeader {
			return
		}

		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(http.StatusInternalServerError)
		// Best-effort write of the error body; the client may already be gone.
		if m.devMode {
			_, _ = fmt.Fprintf(rw, "500 Internal Server Error\n\npanic: %v\n\n%s", rec, stack)
		} else {
			_, _ = io.WriteString(rw, "500 Internal Server Error")
		}
	}()

	m.handler.ServeHTTP(rw, r)
}

// recoverWriter records whether the response has been started, so the recovery
// path knows whether it can still emit a 500. It implements Unwrap so that
// http.ResponseController (used for flushing, deadlines, hijacking, etc.) reaches
// the underlying writer, keeping streaming responses like SSE working.
type recoverWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *recoverWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *recoverWriter) Write(b []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(b)
}

func (w *recoverWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
