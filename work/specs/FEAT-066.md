---
id: FEAT-066
title: "Database Download/Upload for Dev Tools"
status: implemented
priority: medium
created: 2025-12-11
updated: 2025-12-11
author: "@human + AI"
---

# FEAT-066: Database Download/Upload for Dev Tools

## Summary
Add download and upload buttons to the database management page (`/__/db`) to enable remote database backup and management with SQLite CLI tools. Upload automatically invalidates all caches.

## User Story
As a developer managing a remote Basil site, I want to download the SQLite database for backup and local management, and upload a modified database back to the server, so that I can manage the database using standard SQLite tools without SSH access.

## Acceptance Criteria

### Download Functionality
- [ ] "Download Database" button on `/__/db` page
- [ ] Downloads the SQLite database file as `database.db` (or configured filename)
- [ ] Sets appropriate `Content-Disposition` header for browser download
- [ ] Sets `Content-Type: application/x-sqlite3`
- [ ] Only available in dev mode

### Upload Functionality
- [ ] "Upload Database" button with file picker on `/__/db` page
- [ ] Accepts SQLite database files (`.db`, `.sqlite`, `.sqlite3` extensions)
- [ ] Validates uploaded file is a valid SQLite database
- [ ] Creates backup of existing database before replacing (`.backup` suffix)
- [ ] Replaces current database with uploaded file
- [ ] Clears all caches after successful upload (script, response, fragment)
- [ ] 100MB file size limit
- [ ] Only available in dev mode

### Error Handling
- [ ] Clear error if no database configured
- [ ] Clear error if upload file is not SQLite format
- [ ] Clear error if upload exceeds size limit
- [ ] Error if backup creation fails
- [ ] Rollback on upload failure

### User Experience
- [ ] Buttons prominently placed at top of database page
- [ ] Success message after upload with backup location
- [ ] Upload shows progress indicator (if possible)
- [ ] Download works in all modern browsers

## Design Decisions

- **Dev mode only**: Uses existing `/__/` dev tools security model
- **Automatic backup**: Safety net - always backup before replace with timestamp
- **Cache invalidation**: Uploaded database might have different data, so invalidate everything
- **Size limit**: 100MB prevents abuse and memory issues
- **Validation**: Prevents accidental upload of wrong file type
- **Backup naming**: `<original>.YYYYMMDD-HHMMSS.backup` (e.g., `data.db.20251211-143022.backup`) - timestamped for multiple backups

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `server/dev_tools.go` — Add download and upload HTTP handlers
- `server/dev_tools.html` (embedded) — Add download/upload buttons to database page

### HTTP Endpoints

**Download:**
```
GET /__/db/download
```

**Upload:**
```
POST /__/db/upload
Content-Type: multipart/form-data
```

### Download Implementation

```go
func (h *devToolsHandler) handleDatabaseDownload(w http.ResponseWriter, r *http.Request) {
    // Get database path from server config
    dbPath := h.server.config.SQLite
    
    // Read file
    data, err := os.ReadFile(dbPath)
    if err != nil {
        http.Error(w, "Failed to read database", 500)
        return
    }
    
    // Set headers
    filename := filepath.Base(dbPath)
    w.Header().Set("Content-Type", "application/x-sqlite3")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
    w.Header().Set("Content-Length", strconv.Itoa(len(data)))
    
    w.Write(data)
}
```

### Upload Implementation

```go
func (h *devToolsHandler) handleDatabaseUpload(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form (100MB limit)
    r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)
    if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil {
        http.Error(w, "File too large (max 100MB)", 413)
        return
    }
    
    // Get uploaded file
    file, header, err := r.FormFile("database")
    if err != nil {
        http.Error(w, "No file uploaded", 400)
        return
    }
    defer file.Close()
    
    // Read file content
    data, err := io.ReadAll(file)
    if err != nil {
        http.Error(w, "Failed to read uploaded file", 500)
        return
    }
    
    // Validate SQLite format (magic bytes)
    if !isSQLiteDatabase(data) {
        http.Error(w, "Uploaded file is not a valid SQLite database", 400)
        return
    }
    
    // Backup existing database
    dbPath := h.server.config.SQLite
    timestamp := time.Now().Format("20060102-150405")
    backupPath := fmt.Sprintf("%s.%s.backup", dbPath, timestamp)
    if err := copyFile(dbPath, backupPath); err != nil {
        http.Error(w, "Failed to create backup", 500)
        return
    }
    
    // Write new database
    if err := os.WriteFile(dbPath, data, 0644); err != nil {
        // Try to restore backup
        copyFile(backupPath, dbPath)
        http.Error(w, "Failed to write database", 500)
        return
    }
    
    // Clear all caches
    h.server.ReloadScripts() // This already clears script, response, fragment caches
    
    http.Redirect(w, r, "/__/db?uploaded=1", http.StatusSeeOther)
}

func isSQLiteDatabase(data []byte) bool {
    // SQLite magic: "SQLite format 3\x00"
    return len(data) >= 16 && string(data[0:15]) == "SQLite format 3"
}
```

### UI Changes

Add to database page HTML:

```html
<div style="margin-bottom: 1rem; display: flex; gap: 0.5rem;">
    <a href="/__/db/download" class="button" download>⬇️ Download Database</a>
    <form method="POST" action="/__/db/upload" enctype="multipart/form-data" style="display: inline;">
        <input type="file" name="database" accept=".db,.sqlite,.sqlite3" required style="display: none;" id="db-upload">
        <label for="db-upload" class="button">⬆️ Upload Database</label>
    </form>
</div>

<script>
document.getElementById('db-upload').addEventListener('change', function(e) {
    if (this.files.length > 0) {
        if (confirm('Upload will replace the database and clear all caches. Continue?')) {
            this.form.submit();
        } else {
            this.value = '';
        }
    }
});
</script>
```

### Edge Cases & Constraints

1. **No database configured**: Show message "No database configured" instead of buttons
2. **Database locked**: SQLite might be locked during active transactions - handle gracefully
3. **File permissions**: Ensure uploaded file has correct permissions (0644)
4. **Concurrent uploads**: Not a concern in dev mode (single developer)
5. **Large files**: 100MB limit prevents memory issues, stream if needed later
6. **Browser compatibility**: Standard HTML5 download/upload works everywhere

### Security Considerations

- Only available in dev mode (`/__/` routes already protected)
- No authentication needed beyond dev mode check
- Validate file format before writing
- Create backup before any destructive operation
- Size limit prevents DoS

## Implementation Notes

**Implemented**: 2025-12-11

### Key Implementation Details

1. **Download Handler** (`handleDevDBFileDownload`):
   - Uses `http.ServeFile` for efficient streaming
   - Sets proper Content-Type and Content-Disposition headers
   - Extracts filename from configured database path

2. **Upload Handler** (`handleDevDBFileUpload`):
   - Validates SQLite magic bytes: `53 51 4C 69 74 65 20 66 6F 72 6D 61 74 20 33 00`
   - Creates timestamped backup: `<dbPath>.YYYYMMDD-HHMMSS.backup`
   - Replaces database file atomically
   - Calls `server.ReloadScripts()` to invalidate all caches
   - Returns JSON response with success status and backup filename

3. **Helper Functions**:
   - `copyFile(src, dst)`: Safe file copy with error handling
   - `bytesEqual(a, b)`: Byte slice comparison for magic bytes validation

4. **UI Implementation** (`server/prelude/devtools/db.pars`):
   - Download button linking to `/__/db/download`
   - Upload form with file input and submit button
   - JavaScript for file selection UX and AJAX upload
   - Success/error message display
   - Auto-reload after successful upload

5. **Routes** (added to `devtools.go` ServeHTTP):
   - `GET /__/db/download` → `handleDevDBFileDownload`
   - `POST /__/db/upload` → `handleDevDBFileUpload`

### Test Coverage

Added 6 comprehensive tests in `server/devtools_test.go`:
- `TestDevToolsDBFileDownload`: Verifies download headers and content
- `TestDevToolsDBFileUpload`: Tests valid upload with backup creation
- `TestDevToolsDBFileUploadInvalidFile`: Validates rejection of non-SQLite files
- `TestCopyFile`: Tests file copy utility
- `TestCopyFileNonExistent`: Tests error handling
- `TestBytesEqual`: Tests byte comparison utility

All tests pass successfully.

### Files Modified

- `server/devtools.go`: Added download/upload handlers, helper functions
- `server/devtools_test.go`: Added comprehensive test coverage
- `server/prelude/devtools/db.pars`: Updated UI with download/upload controls
- `docs/specs/FEAT-066.md`: This specification
- `docs/plans/FEAT-066-plan.md`: Implementation plan

### Commit

```
feat: add database download/upload to dev tools (FEAT-066)

- Add download button to export entire SQLite database
- Add upload form with drag-and-drop file selection
- Validate uploaded files with SQLite magic bytes check
- Automatically create timestamped backups before upload
- Invalidate caches after database replacement
- Add comprehensive tests for download, upload, validation
```

## Related
- Plan: `docs/plans/FEAT-066-plan.md`
- Similar: Existing CSV import/export on `/__/db` page (per-table operations)
