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
		logsHTML.WriteString(`<div class="empty-state">No logs yet. Use <code>let {dev} = import @std/dev</code> then <code>dev.log(value)</code> in your handlers.</div>`)
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

// serveDBView serves the table data view page.
func (h *devToolsHandler) serveDBView(w http.ResponseWriter, r *http.Request, tableName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := h.openAppDB()
	if err != nil {
		h.serveDBError(w, "Database Error", err.Error())
		return
	}
	defer db.Close()

	// Get table data
	columns, rows, err := getTableData(db, tableName)
	if err != nil {
		h.serveDBError(w, "Query Error", err.Error())
		return
	}

	// Build table HTML
	var tableHTML strings.Builder
	if len(rows) == 0 {
		tableHTML.WriteString(`<div class="empty-state">No data in this table</div>`)
	} else {
		tableHTML.WriteString(`<div class="data-table-wrapper"><table class="data-table"><thead><tr><th class="row-number">#</th>`)
		for _, col := range columns {
			tableHTML.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(col)))
		}
		tableHTML.WriteString(`</tr></thead><tbody>`)
		for i, row := range rows {
			tableHTML.WriteString(fmt.Sprintf(`<tr><td class="row-number">%d</td>`, i+1))
			for _, val := range row {
				if val == nil {
					tableHTML.WriteString(`<td class="null-value">NULL</td>`)
				} else {
					tableHTML.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(fmt.Sprintf("%v", val))))
				}
			}
			tableHTML.WriteString(`</tr>`)
		}
		tableHTML.WriteString(`</tbody></table></div>`)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, devToolsDBViewHTML,
		html.EscapeString(tableName),
		html.EscapeString(tableName),
		len(rows),
		len(columns),
		tableHTML.String())
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
        <a href="/__/db">
          🗄️ Database
          <div class="tool-desc">View tables, export/import CSV data</div>
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

const devToolsDBViewHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s - Basil Database</title>
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
      max-width: 1200px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .container {
      max-width: 1200px;
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
    .data-table-wrapper {
      background: #252542;
      border-radius: 8px;
      overflow: hidden;
    }
    .data-table {
      width: 100%%;
      border-collapse: collapse;
      font-size: 0.85rem;
    }
    .data-table th {
      text-align: left;
      padding: 0.75rem 1rem;
      background: #1a1a2e;
      color: #61afef;
      font-weight: 500;
      font-size: 0.8rem;
      border-bottom: 1px solid #3d3d5c;
      position: sticky;
      top: 0;
    }
    .data-table td {
      padding: 0.6rem 1rem;
      border-bottom: 1px solid #3d3d5c;
      font-family: 'SF Mono', Monaco, monospace;
      color: #eee;
      max-width: 300px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .data-table tr:last-child td {
      border-bottom: none;
    }
    .data-table tr:hover td {
      background: #2a2a4a;
    }
    .null-value {
      color: #5c6370;
      font-style: italic;
    }
    .empty-state {
      text-align: center;
      padding: 3rem;
      color: #5c6370;
      font-size: 0.95rem;
    }
    .row-number {
      color: #5c6370;
      font-size: 0.75rem;
      text-align: right;
      width: 50px;
    }
  </style>
</head>
<body>
  <div class="header">
    <div class="header-inner">
      <h1><span class="brand">🌿 DEV</span> %s</h1>
      <a href="/__/db" class="back-link">← Database</a>
    </div>
  </div>
  <div class="container">
    <div class="info-box">
      <div class="info-item">
        <span class="info-label">Rows:</span>
        <span class="info-value">%d</span>
      </div>
      <div class="info-item">
        <span class="info-label">Columns:</span>
        <span class="info-value">%d</span>
      </div>
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

// createDevToolsEnv creates an environment for rendering DevTools pages
func (h *devToolsHandler) createDevToolsEnv(path string, r *http.Request) *evaluator.Environment {
	env := evaluator.NewEnvironment()

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

	case path == "/__/db" || path == "/__/db/":
		// Database overview page
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
					tablesArray = append(tablesArray, map[string]interface{}{
						"name":      info.Name,
						"row_count": info.RowCount,
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
		// Environment info page
		configArray := []interface{}{
			map[string]interface{}{"name": "Port", "value": fmt.Sprintf("%d", h.server.config.Server.Port)},
			map[string]interface{}{"name": "Dev Mode", "value": fmt.Sprintf("%v", h.server.config.Server.Dev)},
		}
		devtoolsMap["config"] = configArray
	}

	devtoolsObj, _ := parsley.ToParsley(devtoolsMap)
	env.Set("devtools", devtoolsObj)

	return env
}

// handleDevToolsWithPrelude renders DevTools pages using Parsley templates from prelude
func (h *devToolsHandler) handleDevToolsWithPrelude(w http.ResponseWriter, r *http.Request, templateName string) {
	// Get the prelude AST
	ast := GetPreludeAST("devtools/" + templateName)
	if ast == nil {
		http.Error(w, fmt.Sprintf("Template not found: %s", templateName), http.StatusInternalServerError)
		return
	}

	// Create environment
	env := h.createDevToolsEnv(r.URL.Path, r)

	// Evaluate
	result := evaluator.Eval(ast, env)
	if err, ok := result.(*evaluator.Error); ok {
		http.Error(w, fmt.Sprintf("Template error: %s", err.Message), http.StatusInternalServerError)
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
