package search

import (
	"database/sql"
	"fmt"
)

// FTS5Index represents a full-text search index using SQLite FTS5
type FTS5Index struct {
	db        *sql.DB
	tokenizer string
	weights   Weights
}

// Weights defines the ranking weights for different document fields
type Weights struct {
	Title    float64
	Headings float64
	Tags     float64
	Content  float64
}

// DefaultWeights returns the default ranking weights
func DefaultWeights() Weights {
	return Weights{
		Title:    10.0,
		Headings: 5.0,
		Tags:     3.0,
		Content:  1.0,
	}
}

// NewFTS5Index creates a new FTS5 index with the given database connection
func NewFTS5Index(db *sql.DB, tokenizer string, weights Weights) (*FTS5Index, error) {
	if tokenizer != "porter" && tokenizer != "unicode61" {
		return nil, fmt.Errorf("invalid tokenizer: %s (must be 'porter' or 'unicode61')", tokenizer)
	}

	idx := &FTS5Index{
		db:        db,
		tokenizer: tokenizer,
		weights:   weights,
	}

	if err := idx.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create FTS5 tables: %w", err)
	}

	return idx, nil
}

// createTables creates the FTS5 virtual table and metadata table
func (idx *FTS5Index) createTables() error {
	// Create FTS5 virtual table for document search
	ftsSQL := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			title,
			headings,
			tags,
			content,
			url UNINDEXED,
			date UNINDEXED,
			tokenize='%s'
		)
	`, idx.tokenizer)

	if _, err := idx.db.Exec(ftsSQL); err != nil {
		return fmt.Errorf("failed to create FTS5 table: %w", err)
	}

	// Create metadata table for tracking file mtimes and sources
	metadataSQL := `
		CREATE TABLE IF NOT EXISTS search_metadata (
			url TEXT PRIMARY KEY,
			path TEXT,
			mtime INTEGER,
			indexed_at INTEGER,
			source TEXT
		)
	`

	if _, err := idx.db.Exec(metadataSQL); err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create index on path for efficient lookups
	indexSQL := `
		CREATE INDEX IF NOT EXISTS idx_search_metadata_path 
		ON search_metadata(path)
	`

	if _, err := idx.db.Exec(indexSQL); err != nil {
		return fmt.Errorf("failed to create metadata index: %w", err)
	}

	return nil
}

// DropTables drops the FTS5 and metadata tables (for reindexing)
func (idx *FTS5Index) DropTables() error {
	queries := []string{
		"DROP TABLE IF EXISTS documents_fts",
		"DROP TABLE IF EXISTS search_metadata",
		"DROP INDEX IF EXISTS idx_search_metadata_path",
	}

	for _, query := range queries {
		if _, err := idx.db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}
	}

	return nil
}

// DB returns the underlying database connection
func (idx *FTS5Index) DB() *sql.DB {
	return idx.db
}

// Weights returns the ranking weights
func (idx *FTS5Index) Weights() Weights {
	return idx.weights
}
