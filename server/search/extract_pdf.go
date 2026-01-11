package search

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

const (
	// MaxPDFSize is the maximum file size we'll attempt to process (50MB)
	MaxPDFSize = 50 * 1024 * 1024
)

// ProcessPDF extracts text content from a PDF file and returns a Document for indexing.
// It extracts all plain text content from the PDF. Title is derived from the filename
// since PDF metadata extraction is not reliably supported.
//
// Note: This only works for text-based PDFs. Scanned documents (image-based PDFs)
// will return empty or minimal content since OCR is not performed.
func ProcessPDF(filePath string, mtime time.Time) (*Document, error) {
	// Check file size first
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}
	if info.Size() > MaxPDFSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), MaxPDFSize)
	}

	// Open the PDF file
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open PDF file: %w", err)
	}
	defer f.Close()

	// Extract plain text from all pages
	var buf bytes.Buffer
	plainText, err := r.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("error extracting text from PDF: %w", err)
	}
	buf.ReadFrom(plainText)
	content := buf.String()

	// Clean up the extracted text
	content = cleanPDFText(content)

	// Generate URL from file path
	url := GenerateURL(filePath)

	// Derive title from filename (PDF metadata is unreliable)
	base := filepath.Base(filePath)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Try to extract headings from content (lines that look like headers)
	headings := extractPDFHeadings(content)

	doc := &Document{
		URL:      url,
		Title:    title,
		Content:  content,
		Tags:     nil, // No reliable way to get tags from PDF
		Headings: strings.Join(headings, "\n"),
		Date:     time.Time{}, // PDF date metadata is unreliable
		Path:     filePath,
		Mtime:    mtime.Unix(),
	}

	return doc, nil
}

// cleanPDFText cleans up extracted PDF text
func cleanPDFText(text string) string {
	// Replace multiple consecutive newlines with double newline
	lines := strings.Split(text, "\n")
	var cleaned []string
	prevEmpty := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		isEmpty := line == ""

		if isEmpty {
			if !prevEmpty {
				cleaned = append(cleaned, "")
			}
			prevEmpty = true
		} else {
			cleaned = append(cleaned, line)
			prevEmpty = false
		}
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// extractPDFHeadings attempts to identify heading-like lines from PDF content.
// This is heuristic-based since PDF doesn't have semantic structure like HTML.
// We look for short lines that might be titles/headings.
func extractPDFHeadings(content string) []string {
	var headings []string
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Heuristics for heading detection:
		// 1. Short lines (under 80 chars) that don't end with typical sentence punctuation
		// 2. All caps lines (common heading style)
		// 3. Lines followed by an empty line (common heading pattern)
		// 4. First non-empty line is likely the title

		isShort := len(line) < 80
		noSentenceEnd := !strings.HasSuffix(line, ".") &&
			!strings.HasSuffix(line, ",") &&
			!strings.HasSuffix(line, ";")
		isAllCaps := line == strings.ToUpper(line) && len(line) > 3
		followedByEmpty := i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == ""

		// First non-empty line is likely the title
		if len(headings) == 0 && isShort {
			headings = append(headings, line)
			continue
		}

		// All caps short lines are likely headings
		if isAllCaps && isShort && noSentenceEnd {
			headings = append(headings, line)
			continue
		}

		// Short lines followed by empty line might be headings
		if isShort && noSentenceEnd && followedByEmpty && len(line) > 3 {
			// Avoid adding too many headings
			if len(headings) < 10 {
				headings = append(headings, line)
			}
		}
	}

	return headings
}

// IsPDF checks if a file path has a PDF extension
func IsPDF(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".pdf"
}
