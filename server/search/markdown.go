package search

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// Regex patterns for markdown processing
	h1Pattern         = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	headingPattern    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	codeBlockPattern  = regexp.MustCompile("(?s)```.*?```")
	inlineCodePattern = regexp.MustCompile("`[^`]+`")
)

// ProcessMarkdown processes a markdown file and returns a Document for indexing.
// It parses frontmatter, extracts headings, strips formatting, and generates metadata.
func ProcessMarkdown(content string, filePath string, mtime time.Time) (*Document, error) {
	// Parse frontmatter
	fm, remaining, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Generate URL from file path
	url := GenerateURL(filePath)

	// Extract title (priority: frontmatter > first H1 > filename)
	title := fm.Title
	if title == "" {
		// Try to extract first H1
		if matches := h1Pattern.FindStringSubmatch(remaining); len(matches) > 1 {
			title = strings.TrimSpace(matches[1])
		}
	}
	if title == "" {
		// Fall back to filename without extension
		base := filepath.Base(filePath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
		// Convert dashes/underscores to spaces and title case
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")
		title = cases.Title(language.English).String(title)
	}

	// Extract all headings
	headings := ExtractHeadings(remaining)

	// Strip markdown formatting for content
	plainContent := StripMarkdown(remaining)

	// Create document
	doc := &Document{
		URL:      url,
		Title:    title,
		Content:  plainContent,
		Tags:     fm.Tags,
		Headings: strings.Join(headings, "\n"),
		Date:     fm.Date,
		Path:     filePath,
		Mtime:    mtime.Unix(),
	}

	return doc, nil
}

// GenerateURL converts a file path to a URL.
// Example: "./docs/guide/getting-started.md" → "/docs/guide/getting-started"
func GenerateURL(filePath string) string {
	// Clean the path
	path := filepath.Clean(filePath)

	// Remove leading ./ or ../
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "../")

	// Remove file extension
	path = strings.TrimSuffix(path, filepath.Ext(path))

	// Convert to forward slashes (for Windows compatibility)
	path = filepath.ToSlash(path)

	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// ExtractHeadings extracts all heading text from markdown content.
// Returns an array of heading titles (without the # markers).
func ExtractHeadings(content string) []string {
	matches := headingPattern.FindAllStringSubmatch(content, -1)
	headings := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			heading := strings.TrimSpace(match[1])
			// Remove inline code and formatting
			heading = inlineCodePattern.ReplaceAllString(heading, "")
			heading = strings.ReplaceAll(heading, "**", "")
			heading = strings.ReplaceAll(heading, "*", "")
			heading = strings.ReplaceAll(heading, "__", "")
			heading = strings.ReplaceAll(heading, "_", "")
			heading = strings.ReplaceAll(heading, "~~", "")
			if heading != "" {
				headings = append(headings, heading)
			}
		}
	}

	return headings
}

// StripMarkdownForIndexing strips markdown formatting and code blocks for indexing.
// This is more aggressive than the basic StripMarkdown() for better search results.
func StripMarkdownForIndexing(content string) string {
	// Remove code blocks first (they're often not useful for search)
	text := codeBlockPattern.ReplaceAllString(content, " ")

	// Remove inline code
	text = inlineCodePattern.ReplaceAllString(text, " ")

	// Use the existing StripMarkdown for other formatting
	text = StripMarkdown(text)

	// Collapse multiple spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
