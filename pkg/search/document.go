package search

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Document represents a searchable document
type Document struct {
	URL      string
	Title    string
	Headings string
	Tags     []string
	Content  string
	Date     time.Time
	// Internal fields
	Path  string // File path (empty for manual docs)
	Mtime int64  // Modification time (0 for manual docs)
}

// Validate checks if the document has all required fields
func (d *Document) Validate() error {
	if d.URL == "" {
		return fmt.Errorf("document URL is required")
	}
	if d.Title == "" {
		return fmt.Errorf("document title is required")
	}
	if d.Content == "" {
		return fmt.Errorf("document content is required")
	}
	return nil
}

// htmlTagRegex matches HTML tags
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// markdownFormattingRegex matches common markdown formatting
var markdownFormattingRegex = regexp.MustCompile(`(\*\*|__|\*|_|` + "`" + `|~~|\[|\]\(.*?\))`)

// StripHTML removes HTML tags from content
func StripHTML(content string) string {
	return htmlTagRegex.ReplaceAllString(content, "")
}

// StripMarkdown removes markdown formatting from content
func StripMarkdown(content string) string {
	// Remove markdown formatting characters
	cleaned := markdownFormattingRegex.ReplaceAllString(content, "")
	// Clean up extra whitespace
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

// IndexDocument inserts a document into the FTS5 index
func (idx *FTS5Index) IndexDocument(doc *Document) error {
	if err := doc.Validate(); err != nil {
		return err
	}

	// Strip HTML and markdown from content
	cleanContent := StripMarkdown(StripHTML(doc.Content))
	cleanTitle := StripMarkdown(StripHTML(doc.Title))
	cleanHeadings := StripMarkdown(StripHTML(doc.Headings))

	// Start transaction
	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert into FTS5 table
	tagsStr := strings.Join(doc.Tags, " ")
	dateStr := ""
	if !doc.Date.IsZero() {
		dateStr = doc.Date.Format(time.RFC3339)
	}

	insertFTS := `
		INSERT INTO documents_fts(title, headings, tags, content, url, date)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	if _, err := tx.Exec(insertFTS, cleanTitle, cleanHeadings, tagsStr, cleanContent, doc.URL, dateStr); err != nil {
		return fmt.Errorf("failed to insert document into FTS5: %w", err)
	}

	// Insert or update metadata
	source := "file"
	if doc.Path == "" {
		source = "manual"
	}

	insertMeta := `
		INSERT OR REPLACE INTO search_metadata(url, path, mtime, indexed_at, source)
		VALUES (?, ?, ?, ?, ?)
	`
	if _, err := tx.Exec(insertMeta, doc.URL, doc.Path, doc.Mtime, time.Now().Unix(), source); err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return tx.Commit()
}

// RemoveDocument removes a document from the index by URL
func (idx *FTS5Index) RemoveDocument(url string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete from FTS5 table
	if _, err := tx.Exec("DELETE FROM documents_fts WHERE url = ?", url); err != nil {
		return fmt.Errorf("failed to delete from FTS5: %w", err)
	}

	// Delete from metadata table
	if _, err := tx.Exec("DELETE FROM search_metadata WHERE url = ?", url); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return tx.Commit()
}

// UpdateDocument updates specific fields of a document
func (idx *FTS5Index) UpdateDocument(url string, updates map[string]interface{}) error {
	// First check if document exists
	var exists bool
	err := idx.db.QueryRow("SELECT 1 FROM documents_fts WHERE url = ?", url).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("document not found: %s", url)
	} else if err != nil {
		return fmt.Errorf("failed to check document existence: %w", err)
	}

	// Build UPDATE query dynamically based on provided fields
	var setClauses []string
	var values []interface{}

	if title, ok := updates["title"]; ok {
		if titleStr, ok := title.(string); ok {
			setClauses = append(setClauses, "title = ?")
			values = append(values, StripMarkdown(StripHTML(titleStr)))
		}
	}

	if content, ok := updates["content"]; ok {
		if contentStr, ok := content.(string); ok {
			setClauses = append(setClauses, "content = ?")
			values = append(values, StripMarkdown(StripHTML(contentStr)))
		}
	}

	if tags, ok := updates["tags"]; ok {
		if tagSlice, ok := tags.([]string); ok {
			setClauses = append(setClauses, "tags = ?")
			values = append(values, strings.Join(tagSlice, " "))
		}
	}

	if headings, ok := updates["headings"]; ok {
		if headingsStr, ok := headings.(string); ok {
			setClauses = append(setClauses, "headings = ?")
			values = append(values, StripMarkdown(StripHTML(headingsStr)))
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	// Add URL to values for WHERE clause
	values = append(values, url)

	updateSQL := fmt.Sprintf("UPDATE documents_fts SET %s WHERE url = ?", strings.Join(setClauses, ", "))

	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(updateSQL, values...); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	// Update metadata indexed_at timestamp
	if _, err := tx.Exec("UPDATE search_metadata SET indexed_at = ? WHERE url = ?", time.Now().Unix(), url); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return tx.Commit()
}

// BatchIndex indexes multiple documents in a single transaction
func (idx *FTS5Index) BatchIndex(docs []*Document) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	ftsStmt, err := tx.Prepare(`
		INSERT INTO documents_fts(title, headings, tags, content, url, date)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare FTS statement: %w", err)
	}
	defer ftsStmt.Close()

	metaStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO search_metadata(url, path, mtime, indexed_at, source)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare metadata statement: %w", err)
	}
	defer metaStmt.Close()

	indexedAt := time.Now().Unix()

	for _, doc := range docs {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("invalid document %s: %w", doc.URL, err)
		}

		// Clean content
		cleanContent := StripMarkdown(StripHTML(doc.Content))
		cleanTitle := StripMarkdown(StripHTML(doc.Title))
		cleanHeadings := StripMarkdown(StripHTML(doc.Headings))
		tagsStr := strings.Join(doc.Tags, " ")
		dateStr := ""
		if !doc.Date.IsZero() {
			dateStr = doc.Date.Format(time.RFC3339)
		}

		// Insert into FTS5
		if _, err := ftsStmt.Exec(cleanTitle, cleanHeadings, tagsStr, cleanContent, doc.URL, dateStr); err != nil {
			return fmt.Errorf("failed to insert document %s: %w", doc.URL, err)
		}

		// Insert metadata
		source := "file"
		if doc.Path == "" {
			source = "manual"
		}
		if _, err := metaStmt.Exec(doc.URL, doc.Path, doc.Mtime, indexedAt, source); err != nil {
			return fmt.Errorf("failed to insert metadata for %s: %w", doc.URL, err)
		}
	}

	return tx.Commit()
}
