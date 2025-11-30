package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// requestLogger is middleware that logs HTTP requests
type requestLogger struct {
	handler http.Handler
	output  io.Writer
	format  string // "json" or "text"
}

// RequestLogEntry represents a single request log entry
type RequestLogEntry struct {
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Duration   string `json:"duration"`
	DurationMs int64  `json:"duration_ms"`
	ClientIP   string `json:"client_ip"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// responseCapture wraps http.ResponseWriter to capture status code
type responseCapture struct {
	http.ResponseWriter
	status int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if rc.status == 0 {
		rc.status = http.StatusOK
	}
	return rc.ResponseWriter.Write(b)
}

// newRequestLogger creates request logging middleware
func newRequestLogger(handler http.Handler, output io.Writer, format string) *requestLogger {
	if format == "" {
		format = "text"
	}
	return &requestLogger{
		handler: handler,
		output:  output,
		format:  format,
	}
}

func (rl *requestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap response writer to capture status
	rc := &responseCapture{ResponseWriter: w, status: 0}

	// Serve the request
	rl.handler.ServeHTTP(rc, r)

	// Calculate duration
	duration := time.Since(start)

	// Get client IP (respecting X-Forwarded-For if present)
	clientIP := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = xff
	}

	// Build log entry
	entry := RequestLogEntry{
		Timestamp:  start.Format(time.RFC3339),
		Method:     r.Method,
		Path:       r.URL.Path,
		Status:     rc.status,
		Duration:   duration.String(),
		DurationMs: duration.Milliseconds(),
		ClientIP:   clientIP,
		UserAgent:  r.UserAgent(),
	}

	// Write log
	if rl.format == "json" {
		rl.writeJSON(entry)
	} else {
		rl.writeText(entry)
	}
}

func (rl *requestLogger) writeJSON(entry RequestLogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	fmt.Fprintf(rl.output, "%s\n", data)
}

func (rl *requestLogger) writeText(entry RequestLogEntry) {
	fmt.Fprintf(rl.output, "%s %s %s %d %s\n",
		entry.Timestamp,
		entry.Method,
		entry.Path,
		entry.Status,
		entry.Duration,
	)
}
