package testenv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// startHTTPS starts a fake TLS server using httptest.NewTLSServer.
// It populates env.HTTPSURL and env.HTTPSClient and registers cleanup.
func startHTTPS(t testing.TB, env *Env) {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)

	env.HTTPSURL = server.URL
	env.HTTPSClient = server.Client()
	env.httpsMux = mux
}

// ServeJSON registers a handler at path that responds with v marshalled as JSON.
func (e *Env) ServeJSON(path string, v any) {
	mux := e.httpsMux.(*http.ServeMux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
		}
	})
}

// ServeText registers a handler at path that responds with text as plain text.
func (e *Env) ServeText(path, text string) {
	mux := e.httpsMux.(*http.ServeMux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(text))
	})
}

// ServeRedirect registers a handler at path that redirects to target with the given HTTP status code.
// Use 301 for permanent, 302 for temporary.
func (e *Env) ServeRedirect(path, target string, code int) {
	mux := e.httpsMux.(*http.ServeMux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, code)
	})
}

// ServeError registers a handler at path that responds with the given HTTP status code.
func (e *Env) ServeError(path string, code int) {
	mux := e.httpsMux.(*http.ServeMux)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	})
}
