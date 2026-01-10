package search

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// SearchOptions contains options for search queries
type SearchOptions struct {
	Limit      int
	Offset     int
	Raw        bool
	Filters    SearchFilters
	Extensions []string
}

// SearchFilters contains metadata filters
type SearchFilters struct {
	Tags       []string
	DateAfter  time.Time
	DateBefore time.Time
}

// SearchResult represents a single search result
type SearchResult struct {
	URL       string
	Path      string    // Source file path (empty for manual docs)
	Title     string
	Snippet   string
	Score     float64
	Rank      int
	Date      time.Time
	Highlight string // Snippet with <mark> tags
}

// SearchResults contains all search results and metadata
type SearchResults struct {
	Query   string
	Total   int
	Limit   int
	Offset  int
	Results []SearchResult
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit:  10,
		Offset: 0,
		Raw:    false,
	}
}

// Search executes a full-text search query
func (idx *FTS5Index) Search(query string, opts SearchOptions) (*SearchResults, error) {
	if query == "" {
		return &SearchResults{
			Query:   query,
			Total:   0,
			Limit:   opts.Limit,
			Offset:  opts.Offset,
			Results: []SearchResult{},
		}, nil
	}

	// Sanitize query
	ftsQuery := SanitizeQuery(query, opts.Raw)
	if ftsQuery == "" {
		return &SearchResults{
			Query:   query,
			Total:   0,
			Limit:   opts.Limit,
			Offset:  opts.Offset,
			Results: []SearchResult{},
		}, nil
	}

	// Build SQL query with BM25 ranking
	sqlQuery, args := idx.buildSearchSQL(ftsQuery, opts)

	// Execute query
	rows, err := idx.db.Query(sqlQuery, args...)
	if err != nil {
		// Handle FTS5 syntax errors gracefully
		if strings.Contains(err.Error(), "fts5:") || strings.Contains(err.Error(), "syntax error") {
			return &SearchResults{
				Query:   query,
				Total:   0,
				Limit:   opts.Limit,
				Offset:  opts.Offset,
				Results: []SearchResult{},
			}, nil
		}
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	// Parse results
	var results []SearchResult
	var minScore, maxScore float64
	firstResult := true

	for rows.Next() {
		var r SearchResult
		var dateStr string
		var snippet string
		var pathNull *string // Path may be null for manual docs

		err := rows.Scan(&r.URL, &r.Title, &snippet, &r.Score, &dateStr, &pathNull)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		if pathNull != nil {
			r.Path = *pathNull
		}

		// Parse date
		if dateStr != "" {
			if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
				r.Date = parsed
			}
		}

		// Wrap snippet in <mark> tags
		r.Highlight = wrapSnippet(snippet)
		r.Snippet = snippet

		results = append(results, r)

		// Track min/max scores for normalization
		if firstResult {
			minScore = r.Score
			maxScore = r.Score
			firstResult = false
		} else {
			if r.Score < minScore {
				minScore = r.Score
			}
			if r.Score > maxScore {
				maxScore = r.Score
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	// Normalize scores to 0-1 range and add rank
	scoreRange := maxScore - minScore
	for i := range results {
		results[i].Rank = opts.Offset + i + 1
		if scoreRange > 0 {
			results[i].Score = (results[i].Score - minScore) / scoreRange
		} else {
			results[i].Score = 1.0 // All scores the same
		}
	}

	// Get total count (without limit/offset)
	total := len(results) + opts.Offset
	if len(results) >= opts.Limit {
		// There might be more results
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM documents_fts WHERE documents_fts MATCH ?")
		countArgs := []interface{}{ftsQuery}
		if err := idx.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
			// Non-fatal, use estimate
			total = len(results) + opts.Offset
		}
	}

	return &SearchResults{
		Query:   query,
		Total:   total,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
		Results: results,
	}, nil
}

// buildSearchSQL builds the SQL query with filters and ranking
func (idx *FTS5Index) buildSearchSQL(ftsQuery string, opts SearchOptions) (string, []interface{}) {
	w := idx.weights

	// BM25 with custom weights
	bm25Expr := fmt.Sprintf("bm25(documents_fts, %.1f, %.1f, %.1f, %.1f)",
		w.Title, w.Headings, w.Tags, w.Content)

	// Build WHERE clauses
	whereClauses := []string{"documents_fts MATCH ?"}
	args := []interface{}{ftsQuery}

	// Add tag filter
	if len(opts.Filters.Tags) > 0 {
		// Tags are stored as newline-separated strings
		// Check if any of the requested tags are present
		tagConditions := make([]string, len(opts.Filters.Tags))
		for i, tag := range opts.Filters.Tags {
			tagConditions[i] = "tags LIKE ?"
			args = append(args, "%"+tag+"%")
		}
		whereClauses = append(whereClauses, "("+strings.Join(tagConditions, " OR ")+")")
	}

	// Add date range filters
	if !opts.Filters.DateAfter.IsZero() {
		whereClauses = append(whereClauses, "date >= ?")
		args = append(args, opts.Filters.DateAfter.Unix())
	}
	if !opts.Filters.DateBefore.IsZero() {
		whereClauses = append(whereClauses, "date <= ?")
		args = append(args, opts.Filters.DateBefore.Unix())
	}

	whereClause := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			documents_fts.url,
			title,
			snippet(documents_fts, 3, '<mark>', '</mark>', '...', 64) as snippet,
			%s as score,
			date,
			search_metadata.path
		FROM documents_fts
		LEFT JOIN search_metadata ON documents_fts.url = search_metadata.url
		WHERE %s
		ORDER BY score
		LIMIT ? OFFSET ?
	`, bm25Expr, whereClause)

	args = append(args, opts.Limit, opts.Offset)

	return query, args
}

// wrapSnippet wraps the FTS5 snippet with proper <mark> tags
// FTS5 snippet() function already adds <mark> tags, so we just return it
func wrapSnippet(snippet string) string {
	return snippet
}

// Stats returns search index statistics
func (idx *FTS5Index) Stats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count documents
	var count int
	err := idx.db.QueryRow("SELECT COUNT(*) FROM documents_fts").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}
	stats["documents"] = count

	// Get last indexed timestamp
	var lastIndexed int64
	err = idx.db.QueryRow("SELECT MAX(indexed_at) FROM search_metadata").Scan(&lastIndexed)
	if err != nil {
		lastIndexed = 0
	}
	if lastIndexed > 0 {
		stats["last_indexed"] = time.Unix(lastIndexed, 0).Format(time.RFC3339)
	} else {
		stats["last_indexed"] = nil
	}

	// Get database size (approximate via page count)
	var pageCount, pageSize int
	err = idx.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err == nil {
		err = idx.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
		if err == nil {
			sizeBytes := pageCount * pageSize
			stats["size"] = formatBytes(sizeBytes)
			stats["size_bytes"] = sizeBytes
		}
	}

	return stats, nil
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// normalize normalizes a score to 0-1 range
func normalize(score, min, max float64) float64 {
	if max == min {
		return 1.0
	}
	normalized := (score - min) / (max - min)
	return math.Max(0, math.Min(1, normalized))
}
