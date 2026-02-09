package search

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents parsed YAML frontmatter from a markdown file
type Frontmatter struct {
	Title   string    // Document title
	Tags    []string  // Document tags
	Date    time.Time // Document date
	Authors []string  // Document authors (optional)
	Draft   bool      // Draft status (optional)
}

// ParseFrontmatter extracts and parses YAML frontmatter from markdown content.
// Frontmatter must be between --- delimiters at the start of the file.
// Returns parsed metadata and remaining content.
func ParseFrontmatter(content string) (*Frontmatter, string, error) {
	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		// No frontmatter found
		return &Frontmatter{}, content, nil
	}

	// Find the closing delimiter
	lines := strings.Split(content, "\n")
	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			closingIndex = i
			break
		}
	}

	if closingIndex == -1 {
		// No closing delimiter found, treat as no frontmatter
		return &Frontmatter{}, content, nil
	}

	// Extract YAML content (between delimiters)
	yamlLines := lines[1:closingIndex]
	yamlContent := strings.Join(yamlLines, "\n")

	// Parse YAML
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		// Invalid YAML - log but don't fail
		// Return empty frontmatter and full content
		return &Frontmatter{}, content, nil
	}

	// Extract frontmatter fields
	fm := &Frontmatter{}

	// Extract title
	if title, ok := raw["title"].(string); ok {
		fm.Title = title
	}

	// Extract tags (can be array or comma-separated string)
	if tags, ok := raw["tags"].([]any); ok {
		fm.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				fm.Tags = append(fm.Tags, tagStr)
			}
		}
	} else if tagsStr, ok := raw["tags"].(string); ok {
		// Handle comma-separated tags
		parts := strings.Split(tagsStr, ",")
		fm.Tags = make([]string, 0, len(parts))
		for _, part := range parts {
			tag := strings.TrimSpace(part)
			if tag != "" {
				fm.Tags = append(fm.Tags, tag)
			}
		}
	}

	// Extract date (try multiple formats)
	if dateVal, ok := raw["date"]; ok {
		switch v := dateVal.(type) {
		case string:
			// Try parsing common date formats
			formats := []string{
				time.RFC3339,          // 2006-01-02T15:04:05Z07:00
				"2006-01-02",          // YYYY-MM-DD
				"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
				time.RFC3339Nano,      // with nanoseconds
			}
			for _, format := range formats {
				if t, err := time.Parse(format, v); err == nil {
					fm.Date = t
					break
				}
			}
		case time.Time:
			fm.Date = v
		}
	}

	// Extract authors (optional)
	if authors, ok := raw["authors"].([]any); ok {
		fm.Authors = make([]string, 0, len(authors))
		for _, author := range authors {
			if authorStr, ok := author.(string); ok {
				fm.Authors = append(fm.Authors, authorStr)
			}
		}
	} else if authorStr, ok := raw["author"].(string); ok {
		// Single author field
		fm.Authors = []string{authorStr}
	}

	// Extract draft status (optional)
	if draft, ok := raw["draft"].(bool); ok {
		fm.Draft = draft
	}

	// Return frontmatter and remaining content (after closing delimiter)
	remainingContent := strings.Join(lines[closingIndex+1:], "\n")
	return fm, remainingContent, nil
}
