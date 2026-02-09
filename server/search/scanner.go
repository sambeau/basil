package search

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanOptions configures file scanning behavior
type ScanOptions struct {
	Extensions     []string // File extensions to include (e.g., [".md", ".html"])
	Recursive      bool     // Recursively scan subdirectories
	FollowSymlinks bool     // Follow symbolic links
}

// DefaultScanOptions returns default scanning options
func DefaultScanOptions() *ScanOptions {
	return &ScanOptions{
		Extensions:     []string{".md", ".markdown", ".html"},
		Recursive:      true,
		FollowSymlinks: false,
	}
}

// ScanFolder recursively scans a directory for markdown/HTML files and returns documents.
// It processes each file with ProcessMarkdown() and returns an array of documents ready for indexing.
func ScanFolder(folderPath string, opts *ScanOptions) ([]*Document, error) {
	if opts == nil {
		opts = DefaultScanOptions()
	}

	// Check if folder exists
	info, err := os.Stat(folderPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access folder %s: %w", folderPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", folderPath)
	}

	var documents []*Document
	var scanErrors []error

	// Walk the directory tree
	err = filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log error but continue scanning
			scanErrors = append(scanErrors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil
		}

		// Skip directories (unless it's the root)
		if d.IsDir() {
			// Skip hidden directories (starting with .)
			if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file has a valid extension
		ext := strings.ToLower(filepath.Ext(path))
		validExt := false
		for _, allowedExt := range opts.Extensions {
			if ext == strings.ToLower(allowedExt) {
				validExt = true
				break
			}
		}
		if !validExt {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Get file modification time
		fileInfo, err := d.Info()
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("error getting file info for %s: %w", path, err))
			return nil
		}
		mtime := fileInfo.ModTime()

		// Process file based on extension
		var doc *Document
		if IsDOCX(path) {
			// Process DOCX file (binary format)
			doc, err = ProcessDOCX(path, mtime)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("error processing DOCX %s: %w", path, err))
				return nil
			}
		} else if IsPDF(path) {
			// Process PDF file (binary format)
			doc, err = ProcessPDF(path, mtime)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("error processing PDF %s: %w", path, err))
				return nil
			}
		} else {
			// Process text-based files (markdown, HTML, etc.)
			content, err := os.ReadFile(path)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("error reading %s: %w", path, err))
				return nil
			}
			doc, err = ProcessMarkdown(string(content), path, mtime)
			if err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("error processing %s: %w", path, err))
				return nil
			}
		}

		documents = append(documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	// Log scan errors but don't fail the whole operation
	if len(scanErrors) > 0 { //nolint:staticcheck // TODO: Add proper logging when available
	}

	return documents, nil
}

// ScanMultipleFolders scans multiple directories and combines the results.
// This is useful for the watch parameter which can accept multiple paths.
func ScanMultipleFolders(folders []string, opts *ScanOptions) ([]*Document, error) {
	var allDocuments []*Document
	var errors []error

	for _, folder := range folders {
		docs, err := ScanFolder(folder, opts)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		allDocuments = append(allDocuments, docs...)
	}

	if len(errors) > 0 && len(allDocuments) == 0 {
		// All folders failed
		return nil, fmt.Errorf("failed to scan any folders: %d errors", len(errors))
	}

	return allDocuments, nil
}

// CountFiles counts the number of indexable files in a folder without processing them.
// Useful for progress reporting during indexing.
func CountFiles(folderPath string, opts *ScanOptions) (int, error) {
	if opts == nil {
		opts = DefaultScanOptions()
	}

	count := 0
	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		for _, allowedExt := range opts.Extensions {
			if ext == strings.ToLower(allowedExt) {
				count++
				break
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return count, nil
}
