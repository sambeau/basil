package search

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// FileChange represents a change to a file that needs index updates.
type FileChange struct {
	Type     string    // "new", "changed", or "deleted"
	Document *Document // Document for new/changed files (nil for deleted)
	URL      string    // URL for deleted files
}

// ChangeSet contains all changes detected during a check.
type ChangeSet struct {
	New     []Document
	Changed []Document
	Deleted []string // URLs of deleted documents
}

// CheckForChanges checks watched folders for file changes.
// Returns a ChangeSet with new, changed, and deleted files.
func CheckForChanges(db *sql.DB, watchFolders []string, extensions []string) (*ChangeSet, error) {
	// Get current files from filesystem
	currentFiles := make(map[string]*Document)
	for _, folder := range watchFolders {
		docs, err := ScanFolder(folder, &ScanOptions{
			Extensions: extensions,
			Recursive:  true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan folder %s: %w", folder, err)
		}
		for _, doc := range docs {
			currentFiles[doc.URL] = doc
		}
	}

	// Get all metadata from database
	allMeta, err := GetAllMetadata(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	// Build metadata map by URL
	metaMap := make(map[string]*FileMetadata)
	for i := range allMeta {
		// Only consider auto-indexed files for change detection
		if allMeta[i].Source == "auto" {
			metaMap[allMeta[i].URL] = &allMeta[i]
		}
	}

	changeset := &ChangeSet{}

	// Check for new and changed files
	for url, doc := range currentFiles {
		meta, exists := metaMap[url]
		if !exists {
			// New file
			changeset.New = append(changeset.New, *doc)
		} else {
			// Check if mtime has changed
			if doc.Mtime != meta.Mtime {
				changeset.Changed = append(changeset.Changed, *doc)
			}
		}
	}

	// Check for deleted files
	for url, meta := range metaMap {
		if _, exists := currentFiles[url]; !exists {
			// File was in index but not on filesystem
			// Verify the file actually doesn't exist (not just in wrong folder)
			if _, err := os.Stat(meta.Path); os.IsNotExist(err) {
				changeset.Deleted = append(changeset.Deleted, url)
			}
		}
	}

	return changeset, nil
}

// UpdateIndex updates the search index with the given changes.
// Each document operation runs in its own transaction for consistency.
func UpdateIndex(index *FTS5Index, changes *ChangeSet) error {
	db := index.DB()
	now := time.Now().Unix()

	// Index new files
	for _, doc := range changes.New {
		err := index.IndexDocument(&doc)
		if err != nil {
			return fmt.Errorf("failed to index new document %s: %w", doc.URL, err)
		}

		// Store metadata
		meta := FileMetadata{
			URL:       doc.URL,
			Path:      doc.Path, // Use actual file path
			Mtime:     doc.Mtime,
			IndexedAt: now,
			Source:    "auto",
		}
		err = StoreMetadata(db, meta)
		if err != nil {
			return fmt.Errorf("failed to store metadata for %s: %w", doc.URL, err)
		}
	}

	// Update changed files
	for _, doc := range changes.Changed {
		// Remove old version
		err := index.RemoveDocument(doc.URL)
		if err != nil {
			return fmt.Errorf("failed to remove old document %s: %w", doc.URL, err)
		}

		// Re-index with new content
		err = index.IndexDocument(&doc)
		if err != nil {
			return fmt.Errorf("failed to reindex document %s: %w", doc.URL, err)
		}

		// Update metadata
		meta := FileMetadata{
			URL:       doc.URL,
			Path:      doc.Path, // Use actual file path
			Mtime:     doc.Mtime,
			IndexedAt: now,
			Source:    "auto",
		}
		err = StoreMetadata(db, meta)
		if err != nil {
			return fmt.Errorf("failed to update metadata for %s: %w", doc.URL, err)
		}
	}

	// Remove deleted files
	for _, url := range changes.Deleted {
		err := index.RemoveDocument(url)
		if err != nil {
			return fmt.Errorf("failed to remove document %s: %w", url, err)
		}

		err = RemoveMetadata(db, url)
		if err != nil {
			return fmt.Errorf("failed to remove metadata for %s: %w", url, err)
		}
	}

	return nil
}

// UpdateStats returns statistics about an update operation.
type UpdateStats struct {
	NewFiles     int
	ChangedFiles int
	DeletedFiles int
	Duration     time.Duration
}

// String formats the stats for logging.
func (s UpdateStats) String() string {
	return fmt.Sprintf("new=%d changed=%d deleted=%d duration=%v",
		s.NewFiles, s.ChangedFiles, s.DeletedFiles, s.Duration)
}

// CheckAndUpdate checks for changes and updates the index if needed.
// Returns statistics about the update.
func CheckAndUpdate(index *FTS5Index, watchFolders []string, extensions []string) (*UpdateStats, error) {
	db := index.DB()
	start := time.Now()

	// Check for changes
	changes, err := CheckForChanges(db, watchFolders, extensions)
	if err != nil {
		return nil, err
	}

	stats := &UpdateStats{
		NewFiles:     len(changes.New),
		ChangedFiles: len(changes.Changed),
		DeletedFiles: len(changes.Deleted),
	}

	// Skip update if no changes
	if stats.NewFiles == 0 && stats.ChangedFiles == 0 && stats.DeletedFiles == 0 {
		stats.Duration = time.Since(start)
		return stats, nil
	}

	// Update index
	err = UpdateIndex(index, changes)
	if err != nil {
		return nil, err
	}

	stats.Duration = time.Since(start)
	return stats, nil
}
