package server

import (
	"database/sql"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// devToolsLiveRefreshJS is the JavaScript for live-refresh polling with pause/play.
// Use in templates via: <script>devtools.live_refresh_js</script>
// Toggle pause with: window.basilLiveRefresh.toggle() or .pause() / .resume()
const devToolsLiveRefreshJS = `
(function() {
  let lastSeq = -1;
  let paused = false;
  const pollInterval = 1000;
  const pollURL = '/__/logs/poll';

  async function checkForChanges() {
    if (paused) {
      setTimeout(checkForChanges, pollInterval);
      return;
    }
    try {
      const resp = await fetch(pollURL);
      const data = await resp.json();
      if (lastSeq === -1) {
        lastSeq = data.seq;
      } else if (data.seq !== lastSeq) {
        console.log('[DevTools] Log change detected, refreshing...');
        location.reload();
      }
    } catch (e) {
      // Server might be restarting, retry
    }
    setTimeout(checkForChanges, pollInterval);
  }

  // Public API for pause/resume
  window.basilLiveRefresh = {
    pause: function() { paused = true; console.log('[DevTools] Live refresh paused'); },
    resume: function() { paused = false; console.log('[DevTools] Live refresh resumed'); },
    toggle: function() { paused = !paused; console.log('[DevTools] Live refresh ' + (paused ? 'paused' : 'resumed')); return !paused; },
    isPaused: function() { return paused; }
  };

  if (document.readyState === 'complete') {
    checkForChanges();
    console.log('[DevTools] Live refresh connected');
  } else {
    window.addEventListener('load', function() {
      checkForChanges();
      console.log('[DevTools] Live refresh connected');
    });
  }
})();
`

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
		h.handleDevToolsWithPrelude(w, r, "index.pars")
	case path == "/__/env" || path == "/__/env/":
		h.handleDevToolsWithPrelude(w, r, "env.pars")
	case path == "/__/logs/poll":
		h.serveLogsPoll(w, r)
	case path == "/__/logs" || path == "/__/logs/":
		h.serveLogs(w, r, "")
	case strings.HasPrefix(path, "/__/logs/"):
		route := strings.TrimPrefix(path, "/__/logs/")
		route = strings.TrimSuffix(route, "/")
		h.serveLogs(w, r, route)
	case path == "/__/db" || path == "/__/db/":
		h.handleDevToolsWithPrelude(w, r, "db.pars")
	case path == "/__/db/download" || path == "/__/db/download/":
		h.handleDevDBFileDownload(w, r)
	case path == "/__/db/upload" || path == "/__/db/upload/":
		h.handleDevDBFileUpload(w, r)
	case strings.HasPrefix(path, "/__/db/view/"):
		h.handleDevToolsWithPrelude(w, r, "db_table.pars")
	case strings.HasPrefix(path, "/__/db/download/"):
		tableName := strings.TrimPrefix(path, "/__/db/download/")
		tableName = strings.TrimSuffix(tableName, "/")
		h.serveDBDownload(w, r, tableName)
	case strings.HasPrefix(path, "/__/db/upload/"):
		tableName := strings.TrimPrefix(path, "/__/db/upload/")
		tableName = strings.TrimSuffix(tableName, "/")
		h.serveDBUpload(w, r, tableName)
	case path == "/__/db/create":
		h.serveDBCreate(w, r)
	case strings.HasPrefix(path, "/__/db/delete/"):
		tableName := strings.TrimPrefix(path, "/__/db/delete/")
		tableName = strings.TrimSuffix(tableName, "/")
		h.serveDBDelete(w, r, tableName)
	default:
		http.NotFound(w, r)
	}
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

	// Check for ?text query param
	var entries []LogEntry
	if h.server.devLog != nil {
		var err error
		entries, err = h.server.devLog.GetLogs(route, 500)
		if err != nil {
			h.server.logError("failed to get logs: %v", err)
		}
	}

	if r.URL.Query().Has("text") {
		h.serveLogsText(w, entries, route)
		return
	}

	h.handleDevToolsWithPrelude(w, r, "logs.pars")
}

// serveLogsPoll serves the logs polling endpoint for live refresh.
// Returns JSON with the current log sequence number.
func (h *devToolsHandler) serveLogsPoll(w http.ResponseWriter, r *http.Request) {
	seq := uint64(0)
	if h.server.devLog != nil {
		seq = h.server.devLog.GetSeq()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprintf(w, `{"seq":%d}`, seq)
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

// openAppDB opens the application's SQLite database.
func (h *devToolsHandler) openAppDB() (*sql.DB, error) {
	dbPath := h.server.config.SQLite
	if dbPath == "" {
		return nil, fmt.Errorf("no database configured (set sqlite in config)")
	}

	// Resolve relative path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(h.server.config.BaseDir, dbPath)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return db, nil
}

// serveDB serves the database overview page.
func (h *devToolsHandler) serveDB(w http.ResponseWriter, r *http.Request) {
	db, err := h.openAppDB()
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}
	defer db.Close()

	// Get table list
	tables, err := getTableList(db)
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}

	// Get info for each table
	var tableInfos []*TableInfo
	for _, name := range tables {
		info, err := getTableInfo(db, name)
		if err != nil {
			h.server.logError("failed to get table info for %s: %v", name, err)
			continue
		}
		tableInfos = append(tableInfos, info)
	}

	// Build tables HTML
	var tablesHTML strings.Builder
	if len(tableInfos) == 0 {
		tablesHTML.WriteString(`<div class="empty-state">No tables in database. Create one below.</div>`)
	} else {
		for _, t := range tableInfos {
			tablesHTML.WriteString(h.renderTableCard(t))
		}
	}

	// Get database filename for display
	dbPath := h.server.config.SQLite

	htmlOut := fmt.Sprintf(devToolsDBHTML,
		html.EscapeString(filepath.Base(dbPath)),
		len(tableInfos),
		tablesHTML.String())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlOut)
}

// renderTableCard renders a single table's card HTML.
func (h *devToolsHandler) renderTableCard(t *TableInfo) string {
	var columnsHTML strings.Builder
	for _, col := range t.Columns {
		constraints := ""
		if col.PK {
			constraints = " <span class=\"constraint\">PK</span>"
		}
		if col.NotNull {
			constraints += " <span class=\"constraint\">NOT NULL</span>"
		}
		columnsHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td class="col-name">%s</td>
				<td class="col-type">%s%s</td>
			</tr>`,
			html.EscapeString(col.Name),
			html.EscapeString(col.Type),
			constraints))
	}

	return fmt.Sprintf(`
		<div class="table-card">
			<div class="table-header">
				<span class="table-name">📊 %s</span>
				<a href="/__/db/view/%s" class="row-count-link">%d rows →</a>
			</div>
			<table class="columns-table">
				<thead>
					<tr><th>Column</th><th>Type</th></tr>
				</thead>
				<tbody>%s</tbody>
			</table>
			<div class="table-actions">
				<a href="/__/db/download/%s" class="btn btn-download">⬇️ Download CSV</a>
				<form action="/__/db/upload/%s" method="POST" enctype="multipart/form-data" class="upload-form">
					<label class="btn btn-upload">
						⬆️ Replace Table from CSV
						<input type="file" name="file" accept=".csv" onchange="this.form.submit()" hidden>
					</label>
				</form>
				<div class="spacer"></div>
				<form action="/__/db/delete/%s" method="POST" class="delete-form" onsubmit="return confirm('Delete table %s? This cannot be undone.')">
					<button type="submit" class="btn btn-delete">🗑️ Delete</button>
				</form>
			</div>
		</div>`,
		html.EscapeString(t.Name),
		html.EscapeString(t.Name),
		t.RowCount,
		columnsHTML.String(),
		html.EscapeString(t.Name),
		html.EscapeString(t.Name),
		html.EscapeString(t.Name),
		html.EscapeString(t.Name))
}

// serveDBDownload serves a table as CSV download.
func (h *devToolsHandler) serveDBDownload(w http.ResponseWriter, r *http.Request, tableName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := h.openAppDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, tableName))

	if err := exportTableCSV(db, tableName, w); err != nil {
		h.server.logError("failed to export table %s: %v", tableName, err)
		// Can't change response at this point if we've started writing
	}
}

// serveDBUpload handles CSV upload to replace a table.
func (h *devToolsHandler) serveDBUpload(w http.ResponseWriter, r *http.Request, tableName string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.serveDBError(w, "Upload Error", "Failed to parse form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.serveDBError(w, "Upload Error", "No file provided: "+err.Error())
		return
	}
	defer file.Close()

	db, err := h.openAppDB()
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}
	defer db.Close()

	if err := replaceTableFromCSV(db, tableName, file); err != nil {
		h.serveDBError(w, "Import Error", err.Error())
		return
	}

	// Re-render the database page directly (avoids Safari redirect bug)
	h.serveDB(w, r)
}

// serveDBCreate handles creating a new table.
func (h *devToolsHandler) serveDBCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tableName := strings.TrimSpace(r.FormValue("name"))
	if tableName == "" {
		h.serveDBError(w, "Create Error", "Table name is required")
		return
	}

	db, err := h.openAppDB()
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}
	defer db.Close()

	if err := createEmptyTable(db, tableName); err != nil {
		h.serveDBError(w, "Create Error", err.Error())
		return
	}

	// Re-render the database page directly (avoids Safari redirect bug)
	h.serveDB(w, r)
}

// serveDBDelete handles deleting a table.
func (h *devToolsHandler) serveDBDelete(w http.ResponseWriter, r *http.Request, tableName string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := h.openAppDB()
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}
	defer db.Close()

	if err := dropTable(db, tableName); err != nil {
		h.serveDBError(w, "Delete Error", err.Error())
		return
	}

	// Re-render the database page directly
	h.serveDB(w, r)
}

// handleDevDBFileDownload downloads the entire database file.
func (h *devToolsHandler) handleDevDBFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbPath := h.server.config.SQLite
	if dbPath == "" {
		http.Error(w, "No database configured", http.StatusInternalServerError)
		return
	}

	// Resolve relative path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(h.server.config.BaseDir, dbPath)
	}

	// Get base filename for download
	basename := filepath.Base(dbPath)

	// Set headers for database file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, basename))

	http.ServeFile(w, r, dbPath)
}

// handleDevDBFileUpload handles uploading a database file to replace the current one.
func (h *devToolsHandler) handleDevDBFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("database")
	if err != nil {
		http.Error(w, "No file provided: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 16 bytes to validate SQLite magic bytes
	magic := make([]byte, 16)
	n, err := file.Read(magic)
	if err != nil || n != 16 {
		http.Error(w, "Failed to read file header", http.StatusBadRequest)
		return
	}

	// Check SQLite magic bytes: "SQLite format 3\x00"
	expectedMagic := []byte{0x53, 0x51, 0x4c, 0x69, 0x74, 0x65, 0x20, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x20, 0x33, 0x00}
	if !bytesEqual(magic, expectedMagic) {
		http.Error(w, "Invalid file: not a SQLite database", http.StatusBadRequest)
		return
	}

	// Reset file pointer to beginning
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Failed to reset file pointer", http.StatusInternalServerError)
			return
		}
	}

	dbPath := h.server.config.SQLite
	if dbPath == "" {
		http.Error(w, "No database configured", http.StatusInternalServerError)
		return
	}

	// Resolve relative path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(h.server.config.BaseDir, dbPath)
	}

	// Create timestamped backup
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s.backup", dbPath, timestamp)
	if err := copyFile(dbPath, backupPath); err != nil {
		http.Error(w, "Failed to create backup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Write uploaded file to database path
	outFile, err := os.Create(dbPath)
	if err != nil {
		http.Error(w, "Failed to create database file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		// Try to restore from backup
		_ = copyFile(backupPath, dbPath)
		http.Error(w, "Failed to write database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate caches
	h.server.ReloadScripts()

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Database uploaded successfully", "backup": "%s"}`, filepath.Base(backupPath))
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return destFile.Sync()
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// serveDBError renders an error page for database operations.
func (h *devToolsHandler) serveDBError(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, devToolsDBErrorHTML, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
}

const devToolsDBHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Basil Database</title>
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
      align-items: center;
      justify-content: space-between;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      padding: 5rem 2rem 2rem 2rem;
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
    .back-link {
      color: #61afef;
      text-decoration: none;
      font-size: 0.9rem;
    }
    .back-link:hover {
      text-decoration: underline;
    }
    .info-box {
      background: #252542;
      border-radius: 8px;
      padding: 1rem 1.5rem;
      margin-bottom: 1.5rem;
      display: flex;
      gap: 2rem;
      align-items: center;
    }
    .info-item {
      display: flex;
      gap: 0.5rem;
      align-items: center;
    }
    .info-label {
      color: #5c6370;
      font-size: 0.85rem;
    }
    .info-value {
      color: #e5c07b;
      font-family: 'SF Mono', Monaco, monospace;
      font-size: 0.85rem;
    }
    .create-section {
      background: #252542;
      border-radius: 8px;
      padding: 1rem 1.5rem;
      margin-bottom: 1.5rem;
    }
    .create-section h2 {
      font-size: 1rem;
      color: #98c379;
      margin-bottom: 0.75rem;
    }
    .create-form {
      display: flex;
      gap: 0.75rem;
      align-items: center;
    }
    .create-form input[type="text"] {
      flex: 1;
      padding: 0.5rem 0.75rem;
      border: 1px solid #3d3d5c;
      border-radius: 4px;
      background: #1a1a2e;
      color: #eee;
      font-size: 0.9rem;
      font-family: 'SF Mono', Monaco, monospace;
    }
    .create-form input[type="text"]:focus {
      outline: none;
      border-color: #98c379;
    }
    .btn {
      display: inline-block;
      padding: 0.5rem 1rem;
      border-radius: 4px;
      font-size: 0.85rem;
      text-decoration: none;
      cursor: pointer;
      border: none;
      transition: background 0.2s;
    }
    .btn-create {
      background: #98c379;
      color: #1a1a2e;
      font-weight: 500;
    }
    .btn-create:hover {
      background: #7cb668;
    }
    .btn-download {
      background: #61afef;
      color: #1a1a2e;
    }
    .btn-download:hover {
      background: #4d9fe6;
    }
    .btn-upload {
      background: #e5c07b;
      color: #1a1a2e;
    }
    .btn-upload:hover {
      background: #d4af6a;
    }
    .empty-state {
      text-align: center;
      padding: 3rem;
      color: #5c6370;
      font-size: 0.95rem;
    }
    .table-card {
      background: #252542;
      border-radius: 8px;
      padding: 1rem 1.5rem;
      margin-bottom: 1rem;
    }
    .table-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.75rem;
      padding-bottom: 0.75rem;
      border-bottom: 1px solid #3d3d5c;
    }
    .table-name {
      font-size: 1.1rem;
      font-weight: 500;
      color: #61afef;
    }
    .row-count-link {
      font-size: 0.85rem;
      color: #61afef;
      text-decoration: none;
    }
    .row-count-link:hover {
      text-decoration: underline;
    }
    .columns-table {
      width: 100%%;
      border-collapse: collapse;
      margin-bottom: 1rem;
      font-size: 0.85rem;
    }
    .columns-table th {
      text-align: left;
      padding: 0.4rem 0.75rem;
      background: #1a1a2e;
      color: #61afef;
      font-weight: 500;
      font-size: 0.75rem;
      text-transform: uppercase;
    }
    .columns-table td {
      padding: 0.4rem 0.75rem;
      border-bottom: 1px solid #3d3d5c;
    }
    .columns-table tr:last-child td {
      border-bottom: none;
    }
    .col-name {
      font-family: 'SF Mono', Monaco, monospace;
      color: #eee;
    }
    .col-type {
      color: #e5c07b;
      font-family: 'SF Mono', Monaco, monospace;
    }
    .constraint {
      font-size: 0.7rem;
      background: #3d3d5c;
      color: #98c379;
      padding: 0.1rem 0.3rem;
      border-radius: 3px;
      margin-left: 0.3rem;
    }
    .table-actions {
      display: flex;
      gap: 0.75rem;
      padding-top: 0.75rem;
      border-top: 1px solid #3d3d5c;
      align-items: center;
    }
    .upload-form, .delete-form {
      display: inline-block;
    }
    .spacer {
      flex: 1;
    }
    .btn-delete {
      background: transparent;
      color: #e06c75;
      border: 1px solid #e06c75;
    }
    .btn-delete:hover {
      background: #e06c75;
      color: #1a1a2e;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="header-inner">
        <h1><span class="brand">🌿 DEV</span> Database</h1>
        <a href="/__" class="back-link">← Dev Tools</a>
      </div>
    </div>
    <div class="info-box">
      <div class="info-item">
        <span class="info-label">File:</span>
        <span class="info-value">%s</span>
      </div>
      <div class="info-item">
        <span class="info-label">Tables:</span>
        <span class="info-value">%d</span>
      </div>
    </div>
    <div class="create-section">
      <h2>➕ Create New Table</h2>
      <form action="/__/db/create" method="POST" class="create-form">
        <input type="text" name="name" placeholder="table_name" pattern="[a-zA-Z_][a-zA-Z0-9_]*" required>
        <button type="submit" class="btn btn-create">Create</button>
      </form>
    </div>
    %s
  </div>
</body>
</html>
`

const devToolsDBErrorHTML = `<!DOCTYPE html>
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
      padding: 2rem;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .error-box {
      background: #252542;
      border-radius: 8px;
      padding: 2rem;
      max-width: 500px;
      text-align: center;
    }
    .error-icon {
      font-size: 3rem;
      margin-bottom: 1rem;
    }
    h1 {
      color: #e06c75;
      font-size: 1.25rem;
      margin-bottom: 1rem;
    }
    .error-message {
      color: #abb2bf;
      font-size: 0.95rem;
      margin-bottom: 1.5rem;
      font-family: 'SF Mono', Monaco, monospace;
      background: #1a1a2e;
      padding: 1rem;
      border-radius: 4px;
      text-align: left;
      word-break: break-word;
    }
    .back-link {
      display: inline-block;
      color: #61afef;
      text-decoration: none;
      font-size: 0.9rem;
    }
    .back-link:hover {
      text-decoration: underline;
    }
  </style>
</head>
<body>
  <div class="error-box">
    <div class="error-icon">⚠️</div>
    <h1>%s</h1>
    <div class="error-message">%s</div>
    <a href="/__/db" class="back-link">← Back to Database</a>
  </div>
</body>
</html>
`

// devToolsComponents lists the shared component files to load into the environment
var devToolsComponents = []string{
	"page.pars",
	"panel.pars",
	"header.pars",
	"logo.pars",
	"info_grid.pars",
	"empty_state.pars",
	"stats.pars",
	"error_state.pars",
}

// loadDevToolsComponents loads shared component definitions into the environment
func loadDevToolsComponents(env *evaluator.Environment) {
	for _, file := range devToolsComponents {
		program := GetPreludeAST("devtools/components/" + file)
		if program != nil {
			// Evaluate to define the component function in env
			evaluator.Eval(program, env)
		}
	}
}

// createDevToolsEnv creates an environment for rendering DevTools pages
func (h *devToolsHandler) createDevToolsEnv(path string, r *http.Request) *evaluator.Environment {
	env := evaluator.NewEnvironment()

	// Load shared components
	loadDevToolsComponents(env)

	// Add Basil metadata
	version := h.server.version
	if version == "" {
		version = "dev"
	}
	basilMap := map[string]interface{}{
		"version":     version,
		"commit":      "unknown", // commit hash not stored in Server struct
		"dev":         h.server.config.Server.Dev,
		"go_version":  runtime.Version(),
		"route_count": len(h.server.config.Routes),
	}
	basilObj, _ := parsley.ToParsley(basilMap)
	env.BasilCtx = basilObj.(*evaluator.Dictionary)
	env.Set("basil", env.BasilCtx)

	// Add DevTools-specific data
	devtoolsMap := map[string]interface{}{}

	// Determine which page we're rendering
	switch {
	case path == "/__" || path == "/__/":
		// Index page
		devtoolsMap["has_db"] = h.server.config.SQLite != ""

	case strings.HasPrefix(path, "/__/logs"):
		// Logs page
		route := ""
		if strings.HasPrefix(path, "/__/logs/") {
			route = strings.TrimPrefix(path, "/__/logs/")
			route = strings.TrimSuffix(route, "/")
		}
		devtoolsMap["route"] = route

		// Get logs
		var entries []LogEntry
		if h.server.devLog != nil {
			var err error
			entries, err = h.server.devLog.GetLogs(route, 500)
			if err != nil {
				h.server.logError("failed to get logs: %v", err)
			}
		}

		// Convert to Parsley-friendly format
		logsArray := make([]interface{}, len(entries))
		for i, e := range entries {
			logsArray[i] = map[string]interface{}{
				"level":     e.Level,
				"filename":  filepath.Base(e.Filename),
				"line":      e.Line,
				"timestamp": e.Timestamp.Format("2006-01-02 15:04:05"),
				"call":      e.CallRepr,
				"value":     e.ValueRepr,
			}
		}
		devtoolsMap["logs"] = logsArray
		devtoolsMap["log_count"] = len(entries)

		clearURL := "/__/logs?clear"
		if route != "" {
			clearURL = fmt.Sprintf("/__/logs/%s?clear", route)
		}
		devtoolsMap["clear_url"] = clearURL

		// Add live refresh JavaScript
		devtoolsMap["live_refresh_js"] = devToolsLiveRefreshJS

	case path == "/__/db" || path == "/__/db/":
		// Database overview page
		// Add database file info
		dbPath := h.server.config.SQLite
		if dbPath != "" {
			devtoolsMap["db_filename"] = filepath.Base(dbPath)
			// Resolve to absolute path for stat
			absPath := dbPath
			if !filepath.IsAbs(dbPath) {
				absPath = filepath.Join(h.server.config.BaseDir, dbPath)
			}
			if stat, err := os.Stat(absPath); err == nil {
				devtoolsMap["db_size"] = stat.Size()
			} else {
				devtoolsMap["db_size"] = 0
			}
		} else {
			devtoolsMap["db_filename"] = ""
			devtoolsMap["db_size"] = 0
		}

		db, err := h.openAppDB()
		if err != nil {
			devtoolsMap["error"] = err.Error()
		} else {
			defer db.Close()

			tables, err := getTableList(db)
			if err != nil {
				devtoolsMap["error"] = err.Error()
			} else {
				// Get info for each table
				tablesArray := make([]interface{}, 0, len(tables))
				for _, name := range tables {
					info, err := getTableInfo(db, name)
					if err != nil {
						h.server.logError("failed to get table info for %s: %v", name, err)
						continue
					}
					// Build columns array for this table
					columnsArray := make([]interface{}, 0, len(info.Columns))
					for _, col := range info.Columns {
						colMap := map[string]interface{}{
							"name":     col.Name,
							"type":     col.Type,
							"not_null": col.NotNull,
							"pk":       col.PK,
						}
						if col.DefaultValue.Valid {
							colMap["default"] = col.DefaultValue.String
						} else {
							colMap["default"] = nil
						}
						columnsArray = append(columnsArray, colMap)
					}

					tablesArray = append(tablesArray, map[string]interface{}{
						"name":         info.Name,
						"row_count":    info.RowCount,
						"columns":      columnsArray,
						"column_count": len(info.Columns),
					})
				}
				devtoolsMap["tables"] = tablesArray
				devtoolsMap["table_count"] = len(tablesArray)
			}
		}

	case strings.HasPrefix(path, "/__/db/view/"):
		// Table view page
		tableName := strings.TrimPrefix(path, "/__/db/view/")
		tableName = strings.TrimSuffix(tableName, "/")
		devtoolsMap["table_name"] = tableName

		db, err := h.openAppDB()
		if err != nil {
			devtoolsMap["error"] = err.Error()
		} else {
			defer db.Close()

			columns, rows, err := getTableData(db, tableName)
			if err != nil {
				devtoolsMap["error"] = err.Error()
			} else {
				// Convert to Parsley-friendly format
				rowsArray := make([]interface{}, len(rows))
				for i, row := range rows {
					cellsArray := make([]interface{}, len(row))
					for j, val := range row {
						if val == nil {
							cellsArray[j] = nil
						} else {
							cellsArray[j] = fmt.Sprintf("%v", val)
						}
					}
					rowsArray[i] = cellsArray
				}

				columnsArray := make([]interface{}, len(columns))
				for i, col := range columns {
					columnsArray[i] = col
				}

				devtoolsMap["columns"] = columnsArray
				devtoolsMap["rows"] = rowsArray
				devtoolsMap["row_count"] = len(rows)
				devtoolsMap["column_count"] = len(columns)
			}
		}

	case path == "/__/env" || path == "/__/env/":
		// Environment info page - organized by section with descriptions
		cfg := h.server.config

		// Helper to create a setting entry
		setting := func(name, value, help string) map[string]interface{} {
			return map[string]interface{}{"name": name, "value": value, "help": help}
		}

		// Helper to format boolean
		boolStr := func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		}

		// Helper to format optional string
		optStr := func(s string) string {
			if s == "" {
				return "(empty)"
			}
			return s
		}

		// Helper to format duration
		durStr := func(d time.Duration) string {
			if d == 0 {
				return "(not set)"
			}
			return d.String()
		}

		// Build grouped config sections
		configGroups := []interface{}{}

		// Server section
		serverSettings := []interface{}{
			setting("Host", optStr(cfg.Server.Host), "Bind address"),
			setting("Port", fmt.Sprintf("%d", cfg.Server.Port), "Listen port"),
			setting("Dev Mode", boolStr(cfg.Server.Dev), "Development mode enabled"),
			setting("Base Dir", cfg.BaseDir, "Configuration base directory"),
		}
		if cfg.Server.HTTPS.Auto || cfg.Server.HTTPS.Cert != "" {
			serverSettings = append(serverSettings,
				setting("HTTPS Auto", boolStr(cfg.Server.HTTPS.Auto), "Let's Encrypt auto-certificates"),
				setting("HTTPS Cert", optStr(cfg.Server.HTTPS.Cert), "Manual certificate path"),
			)
		}
		if cfg.Server.Proxy.Trusted {
			serverSettings = append(serverSettings,
				setting("Proxy Trusted", "true", "Trust X-Forwarded-* headers"),
			)
		}
		configGroups = append(configGroups, map[string]interface{}{
			"name":        "Server",
			"description": "Core server settings",
			"settings":    serverSettings,
		})

		// Session section
		sessionSecret := "●●●●●●●●"
		if cfg.Session.Secret.IsAuto() {
			sessionSecret = "(auto-generated)"
		}
		sessionSettings := []interface{}{
			setting("Store", cfg.Session.Store, "Session storage backend"),
			setting("Secret", sessionSecret, "Encryption secret"),
			setting("Max Age", durStr(cfg.Session.MaxAge), "Session lifetime"),
			setting("Cookie Name", cfg.Session.CookieName, "Session cookie name"),
			setting("SameSite", cfg.Session.SameSite, "Cookie SameSite policy"),
		}
		if cfg.Session.Store == "sqlite" {
			sessionSettings = append(sessionSettings,
				setting("Table", cfg.Session.Table, "SQLite table name"),
				setting("Cleanup", durStr(cfg.Session.Cleanup), "Expired session cleanup interval"),
			)
		}
		configGroups = append(configGroups, map[string]interface{}{
			"name":        "Session",
			"description": "Session storage and cookie settings",
			"settings":    sessionSettings,
		})

		// Database section (if configured)
		if cfg.SQLite != "" {
			configGroups = append(configGroups, map[string]interface{}{
				"name":        "Database",
				"description": "SQLite database settings",
				"settings": []interface{}{
					setting("SQLite", cfg.SQLite, "Database file path"),
				},
			})
		}

		// Security section
		securitySettings := []interface{}{
			setting("Content-Type-Options", optStr(cfg.Security.ContentTypeOptions), "X-Content-Type-Options header"),
			setting("Frame-Options", optStr(cfg.Security.FrameOptions), "X-Frame-Options header"),
			setting("XSS-Protection", optStr(cfg.Security.XSSProtection), "X-XSS-Protection header"),
			setting("Referrer-Policy", optStr(cfg.Security.ReferrerPolicy), "Referrer-Policy header"),
		}
		if cfg.Security.CSP != "" {
			securitySettings = append(securitySettings,
				setting("CSP", cfg.Security.CSP, "Content-Security-Policy header"),
			)
		}
		if cfg.Security.HSTS.Enabled {
			securitySettings = append(securitySettings,
				setting("HSTS Enabled", "true", "HTTP Strict Transport Security"),
				setting("HSTS Max-Age", cfg.Security.HSTS.MaxAge, "HSTS max-age directive"),
			)
		}
		configGroups = append(configGroups, map[string]interface{}{
			"name":        "Security",
			"description": "Security headers and policies",
			"settings":    securitySettings,
		})

		// CORS section (if configured)
		if len(cfg.CORS.Origins) > 0 {
			corsOrigins := "(none)"
			if len(cfg.CORS.Origins) > 0 {
				corsOrigins = strings.Join(cfg.CORS.Origins, ", ")
			}
			configGroups = append(configGroups, map[string]interface{}{
				"name":        "CORS",
				"description": "Cross-Origin Resource Sharing",
				"settings": []interface{}{
					setting("Origins", corsOrigins, "Allowed origins"),
					setting("Credentials", boolStr(cfg.CORS.Credentials), "Allow credentials"),
				},
			})
		}

		// Compression section
		configGroups = append(configGroups, map[string]interface{}{
			"name":        "Compression",
			"description": "Response compression settings",
			"settings": []interface{}{
				setting("Enabled", boolStr(cfg.Compression.Enabled), "Compression enabled"),
				setting("Level", cfg.Compression.Level, "Compression level"),
				setting("Min Size", fmt.Sprintf("%d bytes", cfg.Compression.MinSize), "Minimum response size"),
				setting("Zstd", boolStr(cfg.Compression.Zstd), "Zstd compression support"),
			},
		})

		// Auth section (if enabled)
		if cfg.Auth.Enabled {
			authSettings := []interface{}{
				setting("Enabled", "true", "Authentication enabled"),
				setting("Registration", cfg.Auth.Registration, "Registration mode"),
				setting("Session TTL", durStr(cfg.Auth.SessionTTL), "Auth session lifetime"),
				setting("Login Path", cfg.Auth.LoginPath, "Login redirect path"),
			}
			if cfg.Auth.EmailVerification.Enabled {
				authSettings = append(authSettings,
					setting("Email Verification", "true", "Email verification enabled"),
					setting("Email Provider", cfg.Auth.EmailVerification.Provider, "Email service provider"),
				)
			}
			configGroups = append(configGroups, map[string]interface{}{
				"name":        "Authentication",
				"description": "User authentication settings",
				"settings":    authSettings,
			})
		}

		// Git section (if enabled)
		if cfg.Git.Enabled {
			configGroups = append(configGroups, map[string]interface{}{
				"name":        "Git",
				"description": "Git HTTP server settings",
				"settings": []interface{}{
					setting("Enabled", "true", "Git server enabled"),
					setting("Require Auth", boolStr(cfg.Git.RequireAuth), "Require API key"),
				},
			})
		}

		// Routing section
		routingSettings := []interface{}{}
		if cfg.Site != "" {
			routingSettings = append(routingSettings,
				setting("Site", cfg.Site, "Filesystem-based routing directory"),
			)
			if cfg.SiteCache > 0 {
				routingSettings = append(routingSettings,
					setting("Site Cache", durStr(cfg.SiteCache), "Response cache TTL"),
				)
			}
		}
		if cfg.PublicDir != "" {
			routingSettings = append(routingSettings,
				setting("Public Dir", cfg.PublicDir, "Static files directory"),
			)
		}
		if len(cfg.Routes) > 0 {
			routingSettings = append(routingSettings,
				setting("Routes", fmt.Sprintf("%d configured", len(cfg.Routes)), "Explicit route definitions"),
			)
		}
		if len(cfg.Static) > 0 {
			routingSettings = append(routingSettings,
				setting("Static Routes", fmt.Sprintf("%d configured", len(cfg.Static)), "Static file mappings"),
			)
		}
		if len(routingSettings) > 0 {
			configGroups = append(configGroups, map[string]interface{}{
				"name":        "Routing",
				"description": "URL routing configuration",
				"settings":    routingSettings,
			})
		}

		// Logging section
		loggingSettings := []interface{}{
			setting("Level", cfg.Logging.Level, "Log verbosity"),
			setting("Format", cfg.Logging.Format, "Log output format"),
		}
		configGroups = append(configGroups, map[string]interface{}{
			"name":        "Logging",
			"description": "Log output settings",
			"settings":    loggingSettings,
		})

		devtoolsMap["config"] = configGroups

		// Add meta if configured
		if cfg.Meta != nil && len(cfg.Meta) > 0 {
			devtoolsMap["has_meta"] = true
			devtoolsMap["meta"] = cfg.Meta
		}
	}

	devtoolsObj, _ := parsley.ToParsley(devtoolsMap)
	env.Set("devtools", devtoolsObj)

	return env
}

// handleDevToolsWithPrelude renders DevTools pages using Parsley templates from prelude
func (h *devToolsHandler) handleDevToolsWithPrelude(w http.ResponseWriter, r *http.Request, templateName string) {
	// Get the prelude AST with error context
	ast, parseErr := GetPreludeASTWithError("devtools/" + templateName)
	if parseErr != nil {
		// Try to render using dev_error.pars, fall back to plain text with both errors
		wrappedErr := fmt.Errorf("%s", parseErr.Error())
		if !h.server.renderPreludeError(w, r, 500, wrappedErr) {
			// dev_error.pars also failed - show both errors
			h.renderDoubleError(w, "DevTools Template Error", parseErr.Error())
		}
		return
	}

	// Create environment
	env := h.createDevToolsEnv(r.URL.Path, r)

	// Evaluate
	result := evaluator.Eval(ast, env)
	if err, ok := result.(*evaluator.Error); ok {
		h.renderDevToolsError(w, r, templateName, err)
		return
	}

	// Convert result to Go value
	val := parsley.FromParsley(result)

	// Handle array results (join like error pages)
	var output string
	if arr, ok := val.([]interface{}); ok {
		parts := make([]string, len(arr))
		for i, item := range arr {
			parts[i] = fmt.Sprintf("%v", item)
		}
		output = strings.Join(parts, "")
	} else {
		output = fmt.Sprintf("%v", val)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, output)
}

// renderDevToolsError renders an error that occurred in a devtools template.
// It tries to use dev_error.pars for nice formatting, but falls back to a raw <pre>
// block if that's not possible (e.g., if dev_error.pars itself has the error).
func (h *devToolsHandler) renderDevToolsError(w http.ResponseWriter, r *http.Request, templateName string, err *evaluator.Error) {
	// Build error message with context
	errDetails := fmt.Sprintf("DevTools template error in: devtools/%s\n\n%s", templateName, err.Message)
	if err.File != "" {
		errDetails += fmt.Sprintf("\n\nFile: %s", err.File)
		if err.Line > 0 {
			errDetails += fmt.Sprintf(":%d", err.Line)
			if err.Column > 0 {
				errDetails += fmt.Sprintf(":%d", err.Column)
			}
		}
	}
	for _, hint := range err.Hints {
		errDetails += fmt.Sprintf("\nHint: %s", hint)
	}

	// Don't try to use dev_error.pars to render its own errors (infinite recursion)
	if templateName == "errors/dev_error.pars" || strings.HasSuffix(templateName, "/dev_error.pars") {
		h.renderRawDevToolsError(w, r, errDetails)
		return
	}

	// Try to render using the standard error page system
	wrappedErr := fmt.Errorf("%s at %s:%d:%d", err.Message, err.File, err.Line, err.Column)
	if h.server.renderPreludeError(w, r, 500, wrappedErr) {
		return
	}

	// Fallback to raw <pre> block
	h.renderRawDevToolsError(w, r, errDetails)
}

// renderRawDevToolsError renders a minimal error page with just a <pre> block.
// Used when dev_error.pars cannot be used (e.g., when it's the one with the error).
func (h *devToolsHandler) renderRawDevToolsError(w http.ResponseWriter, r *http.Request, errDetails string) {
	// Try to render using dev_error.pars first
	wrappedErr := fmt.Errorf("%s", errDetails)
	if h.server.renderPreludeError(w, r, 500, wrappedErr) {
		return
	}
	// dev_error.pars also failed - show both errors
	h.renderDoubleError(w, "DevTools Template Error", errDetails)
}

// renderDoubleError renders both the original error and the dev_error.pars error.
// Used when dev_error.pars itself has errors.
func (h *devToolsHandler) renderDoubleError(w http.ResponseWriter, errorType string, originalErr string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(500)

	// Check what went wrong with dev_error.pars
	_, devErrorParseErr := GetPreludeASTWithError("errors/dev_error.pars")
	if devErrorParseErr != nil {
		fmt.Fprintf(w, "%s:\n%s\n\n---\n\nAdditionally, dev_error.pars failed to render:\n%v\n", errorType, originalErr, devErrorParseErr)
	} else {
		// dev_error.pars parsed OK but evaluation failed - just show original error
		fmt.Fprintf(w, "%s:\n%s\n", errorType, originalErr)
	}
}

// renderPlainTextError renders a plain text error response.
// Used as ultimate fallback when all templates fail.
func (h *devToolsHandler) renderPlainTextError(w http.ResponseWriter, errDetails string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(500)
	fmt.Fprintf(w, "DevTools Error\n\n%s\n", errDetails)
}
