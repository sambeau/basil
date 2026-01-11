package search

import (
	"database/sql"
	"fmt"
)

// FileMetadata represents metadata about an indexed file.
type FileMetadata struct {
	URL       string // Document URL (e.g., "/docs/guide")
	Path      string // Filesystem path (e.g., "./docs/guide.md")
	Mtime     int64  // File modification time (Unix timestamp)
	IndexedAt int64  // When the file was indexed (Unix timestamp)
	Source    string // Source: "auto" or "manual"
}

// CreateMetadataTable creates the search_metadata table if it doesn't exist.
func CreateMetadataTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS search_metadata (
			url TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			mtime INTEGER NOT NULL,
			indexed_at INTEGER NOT NULL,
			source TEXT NOT NULL
		);
		
		CREATE INDEX IF NOT EXISTS idx_search_metadata_path ON search_metadata(path);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}
	return nil
}

// StoreMetadata stores or updates file metadata in the database.
func StoreMetadata(db *sql.DB, meta FileMetadata) error {
	query := `
		INSERT INTO search_metadata (url, path, mtime, indexed_at, source)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			path = excluded.path,
			mtime = excluded.mtime,
			indexed_at = excluded.indexed_at,
			source = excluded.source
	`
	_, err := db.Exec(query, meta.URL, meta.Path, meta.Mtime, meta.IndexedAt, meta.Source)
	if err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}
	return nil
}

// GetMetadata retrieves metadata for a specific URL.
func GetMetadata(db *sql.DB, url string) (*FileMetadata, error) {
	query := `SELECT url, path, mtime, indexed_at, source FROM search_metadata WHERE url = ?`
	row := db.QueryRow(query, url)

	var meta FileMetadata
	err := row.Scan(&meta.URL, &meta.Path, &meta.Mtime, &meta.IndexedAt, &meta.Source)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	return &meta, nil
}

// GetAllMetadata retrieves all file metadata from the database.
func GetAllMetadata(db *sql.DB) ([]FileMetadata, error) {
	query := `SELECT url, path, mtime, indexed_at, source FROM search_metadata ORDER BY url`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all metadata: %w", err)
	}
	defer rows.Close()

	var results []FileMetadata
	for rows.Next() {
		var meta FileMetadata
		err := rows.Scan(&meta.URL, &meta.Path, &meta.Mtime, &meta.IndexedAt, &meta.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		results = append(results, meta)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metadata rows: %w", err)
	}

	return results, nil
}

// RemoveMetadata removes metadata for a specific URL.
func RemoveMetadata(db *sql.DB, url string) error {
	query := `DELETE FROM search_metadata WHERE url = ?`
	_, err := db.Exec(query, url)
	if err != nil {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}
	return nil
}

// GetMetadataByPath retrieves all metadata entries with a specific path prefix.
// Useful for finding all files in a watched folder.
func GetMetadataByPath(db *sql.DB, pathPrefix string) ([]FileMetadata, error) {
	query := `SELECT url, path, mtime, indexed_at, source FROM search_metadata WHERE path LIKE ? ORDER BY path`
	rows, err := db.Query(query, pathPrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata by path: %w", err)
	}
	defer rows.Close()

	var results []FileMetadata
	for rows.Next() {
		var meta FileMetadata
		err := rows.Scan(&meta.URL, &meta.Path, &meta.Mtime, &meta.IndexedAt, &meta.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		results = append(results, meta)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metadata rows: %w", err)
	}

	return results, nil
}
