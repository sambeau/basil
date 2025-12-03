package server

import (
	"fmt"
	"html"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

// devToolsHandler serves dev tool pages at /__/* routes.
type devToolsHandler struct {
	server *Server
}

// newDevToolsHandler creates a new dev tools handler.
func newDevToolsHandler(s *Server) *devToolsHandler {
	return &devToolsHandler{server: s}
}

// ServeHTTP handles requests to /__/* routes.
func (h *devToolsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Not in dev mode - return 404
	if !h.server.config.Server.Dev {
		http.NotFound(w, r)
		return
	}

	path := r.URL.Path

	switch {
	case path == "/__" || path == "/__/":
		h.serveIndex(w, r)
	case path == "/__/env" || path == "/__/env/":
		h.serveEnv(w, r)
	case path == "/__/logs" || path == "/__/logs/":
		h.serveLogs(w, r, "")
	case strings.HasPrefix(path, "/__/logs/"):
		route := strings.TrimPrefix(path, "/__/logs/")
		route = strings.TrimSuffix(route, "/")
		h.serveLogs(w, r, route)
	default:
		http.NotFound(w, r)
	}
}

// serveIndex serves the dev tools index page.
func (h *devToolsHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, devToolsIndexHTML)
}

// serveEnv serves the environment info page.
func (h *devToolsHandler) serveEnv(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	version := h.server.version
	if version == "" {
		version = "dev"
	}
	goVersion := runtime.Version()
	handlerCount := len(h.server.config.Routes)

	// Build config info (sanitized - no secrets or full paths)
	configInfo := []struct{ Name, Value string }{
		{"Port", fmt.Sprintf("%d", h.server.config.Server.Port)},
		{"Dev Mode", fmt.Sprintf("%v", h.server.config.Server.Dev)},
	}

	var infoHTML strings.Builder
	for _, info := range configInfo {
		infoHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%s</td>
			</tr>
		`, html.EscapeString(info.Name), html.EscapeString(info.Value)))
	}

	htmlOut := fmt.Sprintf(devToolsEnvHTML,
		html.EscapeString(version),
		html.EscapeString(goVersion),
		handlerCount,
		infoHTML.String())
	fmt.Fprint(w, htmlOut)
}

// serveLogs serves the logs page.
func (h *devToolsHandler) serveLogs(w http.ResponseWriter, r *http.Request, route string) {
	// Check for ?clear query param
	if r.URL.Query().Has("clear") {
		if h.server.devLog != nil {
			h.server.devLog.ClearLogs(route)
		}
		// Redirect back to logs page without ?clear
		redirectURL := "/__/logs"
		if route != "" {
			redirectURL = "/__/logs/" + route
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}

	// Get logs
	var entries []LogEntry
	if h.server.devLog != nil {
		var err error
		entries, err = h.server.devLog.GetLogs(route, 500)
		if err != nil {
			h.server.logError("failed to get logs: %v", err)
		}
	}

	// Check for ?text query param
	if r.URL.Query().Has("text") {
		h.serveLogsText(w, entries, route)
		return
	}

	h.serveLogsHTML(w, entries, route)
}

// serveLogsText serves logs in plain text format.
func (h *devToolsHandler) serveLogsText(w http.ResponseWriter, entries []LogEntry, route string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if len(entries) == 0 {
		fmt.Fprintln(w, "No logs")
		return
	}

	// Reverse order (oldest first for text output)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		level := "INFO"
		if e.Level == "warn" {
			level = "WARN"
		}
		fmt.Fprintf(w, "[%s] %s %s:%d\n", e.Timestamp.Format("15:04:05"), level, filepath.Base(e.Filename), e.Line)
		fmt.Fprintf(w, "  %s\n", e.CallRepr)
		fmt.Fprintf(w, "  → %s\n\n", e.ValueRepr)
	}
}

// serveLogsHTML serves logs in HTML format.
func (h *devToolsHandler) serveLogsHTML(w http.ResponseWriter, entries []LogEntry, route string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	title := "Basil Logs"
	if route != "" {
		title = fmt.Sprintf("Basil Logs: %s", route)
	}

	clearURL := "/__/logs?clear"
	if route != "" {
		clearURL = fmt.Sprintf("/__/logs/%s?clear", route)
	}

	var logsHTML strings.Builder
	if len(entries) == 0 {
		logsHTML.WriteString(`<div class="empty-state">No logs yet. Use <code>dev.log(value)</code> in your handlers.</div>`)
	} else {
		// Entries are already newest-first, keep that for HTML display
		for _, e := range entries {
			icon := "ℹ️"
			levelClass := "info"
			if e.Level == "warn" {
				icon = "⚠️"
				levelClass = "warn"
			}

			logsHTML.WriteString(fmt.Sprintf(`
				<div class="log-entry %s">
					<div class="log-header">
						<span class="log-icon">%s</span>
						<span class="log-file">📁 %s:%d</span>
						<span class="log-time">🕐 %s</span>
					</div>
					<div class="log-call">💻 %s</div>
					<div class="log-value">%s</div>
				</div>
			`, levelClass, icon, html.EscapeString(filepath.Base(e.Filename)), e.Line,
				e.Timestamp.Format("2006-01-02 15:04:05"),
				html.EscapeString(e.CallRepr),
				html.EscapeString(e.ValueRepr)))
		}
	}

	html := fmt.Sprintf(devToolsLogsHTML, html.EscapeString(title), html.EscapeString(title), len(entries), clearURL, logsHTML.String())
	fmt.Fprint(w, html)
}

// HTML templates

const devToolsIndexHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Basil Dev Tools</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #1a1a2e;
      color: #eee;
      min-height: 100vh;
      padding: 2rem;
    }
    .container {
      max-width: 700px;
      margin: 0 auto;
    }
    h1 {
      font-size: 1.5rem;
      margin-bottom: 1.5rem;
      color: #98c379;
    }
    .brand {
      display: inline-block;
      background: #98c379;
      color: #1a1a2e;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      font-size: 0.75rem;
      font-weight: 600;
      margin-right: 0.5rem;
    }
    .tools-list {
      list-style: none;
    }
    .tools-list li {
      margin-bottom: 0.5rem;
    }
    .tools-list a {
      display: block;
      background: #16213e;
      border-radius: 8px;
      padding: 1rem 1.25rem;
      color: #61afef;
      text-decoration: none;
      border-left: 4px solid #61afef;
      transition: background 0.2s;
    }
    .tools-list a:hover {
      background: #1e2a47;
    }
    .tool-desc {
      color: #7f8c8d;
      font-size: 0.875rem;
      margin-top: 0.25rem;
    }
    .footer {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #2d2d44;
      font-size: 0.8rem;
      color: #5c6370;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1><span class="brand">🌿 DEV</span> Basil Dev Tools</h1>
    <ul class="tools-list">
      <li>
        <a href="/__/logs">
          📋 Logs
          <div class="tool-desc">View dev.log() output from your handlers</div>
        </a>
      </li>
      <li>
        <a href="/__/env">
          ⚙️ Environment
          <div class="tool-desc">View server info and configuration</div>
        </a>
      </li>
    </ul>
    <div class="footer">
      Dev tools are only available in dev mode.
    </div>
  </div>
</body>
</html>
`

const devToolsLogsHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #1a1a2e;
      color: #eee;
      min-height: 100vh;
    }
    .header {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      background: #1a1a2e;
      border-bottom: 1px solid #2d2d44;
      padding: 1rem 2rem;
      z-index: 100;
    }
    .header-inner {
      max-width: 900px;
      margin: 0 auto;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .logs-container {
      max-width: 900px;
      margin: 0 auto;
      padding: 5rem 2rem 2rem 2rem; /* top padding for fixed header */
    }
    h1 {
      font-size: 1.5rem;
      color: #98c379;
    }
    .brand {
      display: inline-block;
      background: #98c379;
      color: #1a1a2e;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      font-size: 0.75rem;
      font-weight: 600;
      margin-right: 0.5rem;
    }
    .actions {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }
    .count {
      color: #7f8c8d;
      font-size: 0.875rem;
    }
    .btn {
      background: #16213e;
      color: #ff6b6b;
      border: 1px solid #2d2d44;
      padding: 0.5rem 1rem;
      border-radius: 4px;
      cursor: pointer;
      text-decoration: none;
      font-size: 0.875rem;
    }
    .btn:hover {
      background: #1e2a47;
    }
    .back-link {
      color: #61afef;
      text-decoration: none;
      font-size: 0.875rem;
    }
    .back-link:hover {
      text-decoration: underline;
    }
    .log-entry {
      background: #16213e;
      border-radius: 8px;
      padding: 1rem 1.25rem;
      margin-bottom: 1rem;
      border-left: 4px solid #61afef;
    }
    .log-entry.warn {
      border-left-color: #f39c12;
    }
    .log-entry.warn .log-icon {
      color: #f39c12;
    }
    .log-header {
      display: flex;
      gap: 1rem;
      margin-bottom: 0.5rem;
      font-size: 0.8rem;
      color: #7f8c8d;
    }
    .log-icon {
      color: #61afef;
    }
    .log-call {
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
      font-size: 0.85rem;
      color: #98c379;
      margin-bottom: 0.5rem;
    }
    .log-value {
      font-family: 'SF Mono', Monaco, 'Courier New', monospace;
      font-size: 0.9rem;
      color: #eee;
      background: #0f0f23;
      padding: 0.75rem 1rem;
      border-radius: 4px;
      white-space: pre-wrap;
      word-break: break-word;
      overflow-x: auto;
    }
    .empty-state {
      background: #16213e;
      border-radius: 8px;
      padding: 2rem;
      text-align: center;
      color: #7f8c8d;
    }
    .empty-state code {
      background: #0f0f23;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      color: #98c379;
    }
    .footer {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #2d2d44;
      font-size: 0.8rem;
      color: #5c6370;
    }
  </style>
</head>
<body>
  <div class="header">
    <div class="header-inner">
      <h1><span class="brand">🌿 DEV</span> %s</h1>
      <div class="actions">
        <a href="/__" class="back-link">← Dev Tools</a>
        <span class="count">%d entries</span>
        <a href="%s" class="btn">🗑️ Clear</a>
      </div>
    </div>
  </div>
  <div class="logs-container">
    %s
    <div class="footer">
      Add <code>?text</code> to URL for plain text output.
    </div>
  </div>
  <script>
    // Auto-scroll to bottom on load
    window.scrollTo(0, document.body.scrollHeight);
  </script>
</body>
</html>
`

const devToolsEnvHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Basil Environment</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #1a1a2e;
      color: #eee;
      min-height: 100vh;
      padding: 2rem;
    }
    .container {
      max-width: 700px;
      margin: 0 auto;
    }
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 1.5rem;
      flex-wrap: wrap;
      gap: 1rem;
    }
    h1 {
      font-size: 1.25rem;
      color: #eee;
    }
    .brand {
      display: inline-block;
      background: #98c379;
      color: #1a1a2e;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      font-size: 0.75rem;
      font-weight: 600;
      margin-right: 0.5rem;
    }
    .back-link {
      color: #61afef;
      text-decoration: none;
      font-size: 0.875rem;
    }
    .back-link:hover {
      text-decoration: underline;
    }
    .info-section {
      background: #16213e;
      border-radius: 8px;
      padding: 1.5rem;
      margin-bottom: 1rem;
    }
    .info-section h2 {
      font-size: 1rem;
      color: #98c379;
      margin-bottom: 1rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .info-grid {
      display: grid;
      grid-template-columns: 1fr 2fr;
      gap: 0.5rem;
    }
    .info-label {
      color: #7f8c8d;
    }
    .info-value {
      color: #e5c07b;
      font-family: 'SF Mono', Monaco, monospace;
    }
    table {
      width: 100%%;
      border-collapse: collapse;
    }
    table td {
      padding: 0.5rem 0;
      border-bottom: 1px solid #2d2d44;
    }
    table td:first-child {
      color: #7f8c8d;
      width: 40%%;
    }
    table td:last-child {
      color: #e5c07b;
      font-family: 'SF Mono', Monaco, monospace;
    }
    table tr:last-child td {
      border-bottom: none;
    }
    .footer {
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #2d2d44;
      font-size: 0.8rem;
      color: #5c6370;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1><span class="brand">🌿 DEV</span> Environment</h1>
      <a href="/__" class="back-link">← Dev Tools</a>
    </div>
    <div class="info-section">
      <h2>🌿 Server</h2>
      <div class="info-grid">
        <span class="info-label">Basil Version</span>
        <span class="info-value">%s</span>
        <span class="info-label">Go Version</span>
        <span class="info-value">%s</span>
        <span class="info-label">Handlers</span>
        <span class="info-value">%d</span>
      </div>
    </div>
    <div class="info-section">
      <h2>⚙️ Configuration</h2>
      <table>
        %s
      </table>
    </div>
    <div class="footer">
      Sensitive information (secrets, full paths) is hidden for security.
    </div>
  </div>
</body>
</html>
`
