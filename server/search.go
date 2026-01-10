package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/search"
	_ "modernc.org/sqlite"
)

// SearchInstance represents a configured search instance
type SearchInstance struct {
	index       *search.FTS5Index
	options     SearchOptions
	db          *sql.DB
	initMutex   sync.Mutex
	initialized bool
	lastCheck   time.Time // Last time we checked for file changes
	checkMutex  sync.Mutex
}

// SearchOptions contains configuration for a search instance
type SearchOptions struct {
	Backend       string
	Watch         []string
	Extensions    []string
	Weights       search.Weights
	SnippetLen    int
	HighlightTag  string
	ExtractTitle  bool
	ExtractTags   bool
	ExtractDate   bool
	Tokenizer     string
	CheckInterval time.Duration // How often to check for file changes (0 = every query)
}

// Global search instance cache (per-configuration)
var (
	searchCache      = make(map[string]*SearchInstance)
	searchCacheMutex sync.RWMutex
)

// generateCacheKey generates a unique cache key from search options
func generateCacheKey(opts SearchOptions) string {
	h := sha256.New()
	h.Write([]byte(opts.Backend))
	h.Write([]byte(opts.Tokenizer))

	// Sort watch paths for consistent hashing
	watchPaths := make([]string, len(opts.Watch))
	copy(watchPaths, opts.Watch)
	sort.Strings(watchPaths)
	for _, p := range watchPaths {
		h.Write([]byte(p))
	}

	// Sort extensions
	exts := make([]string, len(opts.Extensions))
	copy(exts, opts.Extensions)
	sort.Strings(exts)
	for _, e := range exts {
		h.Write([]byte(e))
	}

	// Include weights
	h.Write([]byte(fmt.Sprintf("%.2f,%.2f,%.2f,%.2f",
		opts.Weights.Title, opts.Weights.Headings, opts.Weights.Tags, opts.Weights.Content)))

	return hex.EncodeToString(h.Sum(nil))
}

// NewSearchBuiltin creates the @SEARCH built-in function factory
// Returns {search, error} tuple following the standard error pattern
func NewSearchBuiltin(env *evaluator.Environment) evaluator.Object {
	return &evaluator.StdlibBuiltin{
		Name: "SEARCH",
		Fn: func(args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
			// Helper to create result tuple
			makeResult := func(searchObj evaluator.Object, errObj evaluator.Object) evaluator.Object {
				result := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
					"search": searchObj,
					"error":  errObj,
				})
				result.Env = env
				return result
			}

			if len(args) != 1 {
				return makeResult(evaluator.NULL, &evaluator.String{
					Value: fmt.Sprintf("@SEARCH takes exactly 1 argument (got %d)", len(args)),
				})
			}

			// Parse options dictionary
			optsDict, ok := args[0].(*evaluator.Dictionary)
			if !ok {
				return makeResult(evaluator.NULL, &evaluator.String{
					Value: "@SEARCH requires a dictionary argument",
				})
			}

			opts, err := parseSearchOptions(optsDict, env)
			if err != nil {
				return makeResult(evaluator.NULL, &evaluator.String{
					Value: fmt.Sprintf("invalid @SEARCH options: %v", err),
				})
			}

			// Check cache
			cacheKey := generateCacheKey(opts)
			searchCacheMutex.RLock()
			if cached, exists := searchCache[cacheKey]; exists {
				searchCacheMutex.RUnlock()
				return makeResult(createSearchObject(cached, env), evaluator.NULL)
			}
			searchCacheMutex.RUnlock()

			// Create new search instance
			searchCacheMutex.Lock()
			defer searchCacheMutex.Unlock()

			// Double-check after acquiring write lock
			if cached, exists := searchCache[cacheKey]; exists {
				return makeResult(createSearchObject(cached, env), evaluator.NULL)
			}

			instance, err := createSearchInstance(opts, env)
			if err != nil {
				return makeResult(evaluator.NULL, &evaluator.String{
					Value: fmt.Sprintf("failed to create search instance: %v", err),
				})
			}

			searchCache[cacheKey] = instance
			return makeResult(createSearchObject(instance, env), evaluator.NULL)
		},
	}
}

// parseSearchOptions parses the options dictionary
func parseSearchOptions(optsDict *evaluator.Dictionary, env *evaluator.Environment) (SearchOptions, error) {
	opts := SearchOptions{
		Extensions:    []string{".md", ".html"},
		Weights:       search.DefaultWeights(),
		SnippetLen:    200,
		HighlightTag:  "mark",
		ExtractTitle:  true,
		ExtractTags:   true,
		ExtractDate:   true,
		Tokenizer:     "porter",
		CheckInterval: 0, // Check on every query by default
	}

	// Parse backend
	if backendExpr, ok := optsDict.Pairs["backend"]; ok {
		backend := evaluator.Eval(backendExpr, optsDict.Env)
		if pathDict, ok := backend.(*evaluator.Dictionary); ok && isPathDict(pathDict) {
			opts.Backend = pathDictToString(pathDict)
		} else if str, ok := backend.(*evaluator.String); ok {
			opts.Backend = str.Value
		} else {
			return opts, fmt.Errorf("backend must be a path or string")
		}
	}

	// Parse watch paths
	if watchExpr, ok := optsDict.Pairs["watch"]; ok {
		watch := evaluator.Eval(watchExpr, optsDict.Env)

		// Single path literal
		if pathDict, ok := watch.(*evaluator.Dictionary); ok && isPathDict(pathDict) {
			pathStr := pathDictToString(pathDict)
			fmt.Printf("[DEBUG] Watch path (single path literal): %q\n", pathStr)
			opts.Watch = []string{pathStr}
		} else if str, ok := watch.(*evaluator.String); ok {
			// String path
			fmt.Printf("[DEBUG] Watch path (string): %q\n", str.Value)
			opts.Watch = []string{str.Value}
		} else if arr, ok := watch.(*evaluator.Array); ok {
			// Array of paths
			for _, elem := range arr.Elements {
				if pathDict, ok := elem.(*evaluator.Dictionary); ok && isPathDict(pathDict) {
					pathStr := pathDictToString(pathDict)
					fmt.Printf("[DEBUG] Watch path from array (path literal): %q\n", pathStr)
					opts.Watch = append(opts.Watch, pathStr)
				} else if str, ok := elem.(*evaluator.String); ok {
					fmt.Printf("[DEBUG] Watch path from array (string): %q\n", str.Value)
					opts.Watch = append(opts.Watch, str.Value)
				} else {
					return opts, fmt.Errorf("watch array must contain paths or strings")
				}
			}
		} else {
			return opts, fmt.Errorf("watch must be a path, string, or array of paths/strings")
		}
	}

	// Auto-generate backend from watch path if not provided
	if opts.Backend == "" && len(opts.Watch) > 0 {
		// Use first watch path as base for database name
		base := filepath.Base(opts.Watch[0])
		opts.Backend = filepath.Join(filepath.Dir(opts.Watch[0]), base+"_search.db")
	}

	// Parse tokenizer
	if tokExpr, ok := optsDict.Pairs["tokenizer"]; ok {
		tok := evaluator.Eval(tokExpr, optsDict.Env)
		if str, ok := tok.(*evaluator.String); ok {
			if str.Value != "porter" && str.Value != "unicode61" {
				return opts, fmt.Errorf("tokenizer must be 'porter' or 'unicode61'")
			}
			opts.Tokenizer = str.Value
		}
	}

	// Parse extensions
	if extExpr, ok := optsDict.Pairs["extensions"]; ok {
		ext := evaluator.Eval(extExpr, optsDict.Env)
		if arr, ok := ext.(*evaluator.Array); ok {
			opts.Extensions = nil
			for _, elem := range arr.Elements {
				if str, ok := elem.(*evaluator.String); ok {
					opts.Extensions = append(opts.Extensions, str.Value)
				}
			}
		}
	}

	// Parse weights
	if weightsExpr, ok := optsDict.Pairs["weights"]; ok {
		weights := evaluator.Eval(weightsExpr, optsDict.Env)
		if dict, ok := weights.(*evaluator.Dictionary); ok {
			if titleExpr, ok := dict.Pairs["title"]; ok {
				if num := evaluator.Eval(titleExpr, dict.Env); num != nil {
					if f, ok := num.(*evaluator.Float); ok {
						opts.Weights.Title = f.Value
					} else if i, ok := num.(*evaluator.Integer); ok {
						opts.Weights.Title = float64(i.Value)
					}
				}
			}
			if headingsExpr, ok := dict.Pairs["headings"]; ok {
				if num := evaluator.Eval(headingsExpr, dict.Env); num != nil {
					if f, ok := num.(*evaluator.Float); ok {
						opts.Weights.Headings = f.Value
					} else if i, ok := num.(*evaluator.Integer); ok {
						opts.Weights.Headings = float64(i.Value)
					}
				}
			}
			if tagsExpr, ok := dict.Pairs["tags"]; ok {
				if num := evaluator.Eval(tagsExpr, dict.Env); num != nil {
					if f, ok := num.(*evaluator.Float); ok {
						opts.Weights.Tags = f.Value
					} else if i, ok := num.(*evaluator.Integer); ok {
						opts.Weights.Tags = float64(i.Value)
					}
				}
			}
			if contentExpr, ok := dict.Pairs["content"]; ok {
				if num := evaluator.Eval(contentExpr, dict.Env); num != nil {
					if f, ok := num.(*evaluator.Float); ok {
						opts.Weights.Content = f.Value
					} else if i, ok := num.(*evaluator.Integer); ok {
						opts.Weights.Content = float64(i.Value)
					}
				}
			}
		}
	}

	return opts, nil
}

// createSearchInstance creates a new search instance with the given options
func createSearchInstance(opts SearchOptions, env *evaluator.Environment) (*SearchInstance, error) {
	// Open or create SQLite database
	var db *sql.DB
	var err error

	if opts.Backend == ":memory:" {
		db, err = sql.Open("sqlite", ":memory:")
	} else {
		// Resolve relative path
		dbPath := opts.Backend
		if !filepath.IsAbs(dbPath) && env.RootPath != "" {
			dbPath = filepath.Join(env.RootPath, dbPath)
		}

		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}

		// Open with WAL mode for better concurrency
		connStr := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
		db, err = sql.Open("sqlite", connStr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create FTS5 index
	index, err := search.NewFTS5Index(db, opts.Tokenizer, opts.Weights)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create FTS5 index: %w", err)
	}

	return &SearchInstance{
		index:   index,
		options: opts,
		db:      db,
	}, nil
}

// createSearchObject creates a Parsley dictionary object representing the search instance
func createSearchObject(instance *SearchInstance, env *evaluator.Environment) evaluator.Object {
	dict := evaluator.NewDictionaryFromObjects(make(map[string]evaluator.Object))
	dict.Env = env

	// Add query method
	dict.SetKey("query", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "query",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchQueryMethod(instance, args, callEnv)
		},
	}})

	// Add add method
	dict.SetKey("add", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "add",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchAddMethod(instance, args, callEnv)
		},
	}})

	// Add update method
	dict.SetKey("update", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "update",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchUpdateMethod(instance, args, callEnv)
		},
	}})

	// Add remove method
	dict.SetKey("remove", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "remove",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchRemoveMethod(instance, args, callEnv)
		},
	}})

	// Add stats method
	dict.SetKey("stats", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "stats",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchStatsMethod(instance, args, callEnv)
		},
	}})

	// Add reindex method
	dict.SetKey("reindex", &ast.ObjectLiteralExpression{Obj: &evaluator.StdlibBuiltin{
		Name: "reindex",
		Fn: func(args []evaluator.Object, callEnv *evaluator.Environment) evaluator.Object {
			return searchReindexMethod(instance, args, callEnv)
		},
	}})

	return dict
}

// Helper functions for path dictionaries
func isPathDict(dict *evaluator.Dictionary) bool {
	if typeVal, ok := dict.Pairs["__type"]; ok {
		// Evaluate the expression to get the object
		if dict.Env != nil {
			typeObj := evaluator.Eval(typeVal, dict.Env)
			if typeStr, ok := typeObj.(*evaluator.String); ok {
				return typeStr.Value == "path"
			}
		}
	}
	return false
}

func pathDictToString(dict *evaluator.Dictionary) string {
	// Check for stdio special paths
	if stdioExpr, ok := dict.Pairs["__stdio"]; ok {
		stdioVal := evaluator.Eval(stdioExpr, dict.Env)
		if stdioStr, ok := stdioVal.(*evaluator.String); ok {
			if stdioStr.Value == "stdio" {
				return "-"
			}
			return stdioStr.Value
		}
	}

	// Get segments
	segmentsExpr, ok := dict.Pairs["segments"]
	if !ok {
		return ""
	}
	segments := evaluator.Eval(segmentsExpr, dict.Env)
	segmentsArr, ok := segments.(*evaluator.Array)
	if !ok {
		return ""
	}

	// Get absolute flag
	isAbsolute := false
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		absVal := evaluator.Eval(absExpr, dict.Env)
		if absBool, ok := absVal.(*evaluator.Boolean); ok {
			isAbsolute = absBool.Value
		}
	}

	// Build path string
	var parts []string
	for _, elem := range segmentsArr.Elements {
		if str, ok := elem.(*evaluator.String); ok {
			parts = append(parts, str.Value)
		}
	}

	pathStr := strings.Join(parts, "/")
	if isAbsolute {
		return "/" + pathStr
	}
	return pathStr
}

// ensureInitialized checks if the search index is initialized and performs auto-indexing if needed
func (si *SearchInstance) ensureInitialized() error {
	si.initMutex.Lock()
	defer si.initMutex.Unlock()

	if si.initialized {
		return nil
	}

	// Check if database has any documents
	stats, err := si.index.Stats()
	if err != nil {
		return fmt.Errorf("failed to check index stats: %w", err)
	}

	docCount, ok := stats["documents"].(int)
	if ok && docCount > 0 {
		// Index already has documents
		si.initialized = true
		return nil
	}

	// Auto-index if watch paths are configured
	if len(si.options.Watch) > 0 {
		return si.autoIndex()
	}

	// No watch paths configured, mark as initialized (manual indexing only)
	si.initialized = true
	return nil
}

// autoIndex scans watched folders and indexes all documents
func (si *SearchInstance) autoIndex() error {
	scanOpts := &search.ScanOptions{
		Extensions: si.options.Extensions,
		Recursive:  true,
	}

	// Scan all watched folders
	docs, err := search.ScanMultipleFolders(si.options.Watch, scanOpts)
	if err != nil {
		return fmt.Errorf("failed to scan folders: %w", err)
	}

	if len(docs) == 0 {
		// No documents found, but that's okay
		si.initialized = true
		return nil
	}

	// Batch index all documents
	if err := si.index.BatchIndex(docs); err != nil {
		return fmt.Errorf("failed to index documents: %w", err)
	}

	si.initialized = true
	si.lastCheck = time.Now() // Update last check time
	return nil
}

// checkForUpdates checks watched folders for changes and updates the index
func (si *SearchInstance) checkForUpdates() error {
	si.checkMutex.Lock()
	defer si.checkMutex.Unlock()

	// Check if enough time has passed since last check
	now := time.Now()
	if !si.lastCheck.IsZero() && now.Sub(si.lastCheck) < si.options.CheckInterval {
		return nil // Too soon, skip check
	}

	// Only check if watch paths are configured
	if len(si.options.Watch) == 0 {
		return nil
	}

	// Check for changes and update
	stats, err := search.CheckAndUpdate(si.index, si.options.Watch, si.options.Extensions)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Update last check time
	si.lastCheck = now

	// Log if any changes were found (optional, for debugging)
	if stats.NewFiles > 0 || stats.ChangedFiles > 0 || stats.DeletedFiles > 0 {
		// TODO: Add proper logging when available
		_ = stats
	}

	return nil
}

// searchQueryMethod implements the .query() method
func searchQueryMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) < 1 || len(args) > 2 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("query() takes 1 or 2 arguments (got %d)", len(args)),
		}
	}

	// Ensure index is initialized (auto-index if needed)
	if err := instance.ensureInitialized(); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to initialize search index: %v", err),
		}
	}

	// Check for file updates (respects CheckInterval throttling)
	if err := instance.checkForUpdates(); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to check for updates: %v", err),
		}
	}

	// First argument: query string
	queryStr, ok := args[0].(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "query() first argument must be a string",
		}
	}

	// Second argument (optional): options
	opts := search.DefaultSearchOptions()
	if len(args) == 2 {
		optsDict, ok := args[1].(*evaluator.Dictionary)
		if !ok {
			return &evaluator.Error{
				Class:   evaluator.ErrorClass("type"),
				Message: "query() second argument must be a dictionary",
			}
		}

		// Parse limit
		if limitExpr, ok := optsDict.Pairs["limit"]; ok {
			limit := evaluator.Eval(limitExpr, optsDict.Env)
			if limitInt, ok := limit.(*evaluator.Integer); ok {
				opts.Limit = int(limitInt.Value)
			}
		}

		// Parse offset
		if offsetExpr, ok := optsDict.Pairs["offset"]; ok {
			offset := evaluator.Eval(offsetExpr, optsDict.Env)
			if offsetInt, ok := offset.(*evaluator.Integer); ok {
				opts.Offset = int(offsetInt.Value)
			}
		}

		// Parse raw
		if rawExpr, ok := optsDict.Pairs["raw"]; ok {
			raw := evaluator.Eval(rawExpr, optsDict.Env)
			if rawBool, ok := raw.(*evaluator.Boolean); ok {
				opts.Raw = rawBool.Value
			}
		}

		// Parse filters
		if filtersExpr, ok := optsDict.Pairs["filters"]; ok {
			filters := evaluator.Eval(filtersExpr, optsDict.Env)
			if filtersDict, ok := filters.(*evaluator.Dictionary); ok {
				// Parse tags filter
				if tagsExpr, ok := filtersDict.Pairs["tags"]; ok {
					tags := evaluator.Eval(tagsExpr, filtersDict.Env)
					if tagsArr, ok := tags.(*evaluator.Array); ok {
						for _, elem := range tagsArr.Elements {
							if tagStr, ok := elem.(*evaluator.String); ok {
								opts.Filters.Tags = append(opts.Filters.Tags, tagStr.Value)
							}
						}
					} else if tagStr, ok := tags.(*evaluator.String); ok {
						// Single tag
						opts.Filters.Tags = []string{tagStr.Value}
					}
				}

				// Parse dateAfter filter
				if dateAfterExpr, ok := filtersDict.Pairs["dateAfter"]; ok {
					dateAfter := evaluator.Eval(dateAfterExpr, filtersDict.Env)
					if dateStr, ok := dateAfter.(*evaluator.String); ok {
						// Try parsing common date formats
						formats := []string{
							"2006-01-02",
							"2006-01-02T15:04:05Z07:00",
							"2006-01-02 15:04:05",
						}
						for _, format := range formats {
							if t, err := time.Parse(format, dateStr.Value); err == nil {
								opts.Filters.DateAfter = t
								break
							}
						}
					}
				}

				// Parse dateBefore filter
				if dateBeforeExpr, ok := filtersDict.Pairs["dateBefore"]; ok {
					dateBefore := evaluator.Eval(dateBeforeExpr, filtersDict.Env)
					if dateStr, ok := dateBefore.(*evaluator.String); ok {
						// Try parsing common date formats
						formats := []string{
							"2006-01-02",
							"2006-01-02T15:04:05Z07:00",
							"2006-01-02 15:04:05",
						}
						for _, format := range formats {
							if t, err := time.Parse(format, dateStr.Value); err == nil {
								opts.Filters.DateBefore = t
								break
							}
						}
					}
				}
			}
		}
	}

	// Execute search
	results, err := instance.index.Search(queryStr.Value, opts)
	if err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("search query failed: %v", err),
		}
	}

	// Convert results to Parsley dictionary
	return searchResultsToDict(results, env)
}

// searchResultsToDict converts search results to a Parsley dictionary
func searchResultsToDict(results *search.SearchResults, env *evaluator.Environment) evaluator.Object {
	pairs := make(map[string]evaluator.Object)
	pairs["query"] = &evaluator.String{Value: results.Query}
	pairs["total"] = &evaluator.Integer{Value: int64(results.Total)}
	pairs["limit"] = &evaluator.Integer{Value: int64(results.Limit)}
	pairs["offset"] = &evaluator.Integer{Value: int64(results.Offset)}

	// Convert results array
	items := &evaluator.Array{Elements: make([]evaluator.Object, len(results.Results))}
	for i, r := range results.Results {
		itemPairs := make(map[string]evaluator.Object)
		itemPairs["url"] = &evaluator.String{Value: r.URL}
		itemPairs["title"] = &evaluator.String{Value: r.Title}
		itemPairs["snippet"] = &evaluator.String{Value: r.Snippet}
		itemPairs["highlight"] = &evaluator.String{Value: r.Highlight}
		itemPairs["score"] = &evaluator.Float{Value: r.Score}
		itemPairs["rank"] = &evaluator.Integer{Value: int64(r.Rank)}

		if !r.Date.IsZero() {
			// Create datetime dictionary with __type and date components
			datePairs := make(map[string]evaluator.Object)
			datePairs["__type"] = &evaluator.String{Value: "datetime"}
			datePairs["kind"] = &evaluator.String{Value: "date"}
			datePairs["year"] = &evaluator.Integer{Value: int64(r.Date.Year())}
			datePairs["month"] = &evaluator.Integer{Value: int64(r.Date.Month())}
			datePairs["day"] = &evaluator.Integer{Value: int64(r.Date.Day())}
			datePairs["hour"] = &evaluator.Integer{Value: int64(r.Date.Hour())}
			datePairs["minute"] = &evaluator.Integer{Value: int64(r.Date.Minute())}
			datePairs["second"] = &evaluator.Integer{Value: int64(r.Date.Second())}
			itemPairs["date"] = evaluator.NewDictionaryFromObjects(datePairs)
		}

		items.Elements[i] = evaluator.NewDictionaryFromObjects(itemPairs)
	}
	pairs["items"] = items

	return evaluator.NewDictionaryFromObjects(pairs)
}

// searchAddMethod adds a document to the index manually
func searchAddMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) != 1 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("add() takes exactly 1 argument (got %d)", len(args)),
			Hints:   []string{"Usage: search.add({url: @/path, title: @Title, content: @Content})"},
		}
	}

	// Parse document dictionary
	docDict, ok := args[0].(*evaluator.Dictionary)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "add() requires a dictionary argument",
			Hints:   []string{"Usage: search.add({url: @/path, title: @Title, content: @Content})"},
		}
	}

	// Extract required fields
	urlExpr, ok := docDict.Pairs["url"]
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("value"),
			Message: "add() requires 'url' field",
			Hints:   []string{"Usage: search.add({url: @/path, title: @Title, content: @Content})"},
		}
	}
	urlObj := evaluator.Eval(urlExpr, docDict.Env)
	urlStr, ok := urlObj.(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "url must be a string",
		}
	}

	titleExpr, ok := docDict.Pairs["title"]
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("value"),
			Message: "add() requires 'title' field",
			Hints:   []string{"Usage: search.add({url: @/path, title: @Title, content: @Content})"},
		}
	}
	titleObj := evaluator.Eval(titleExpr, docDict.Env)
	titleStr, ok := titleObj.(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "title must be a string",
		}
	}

	contentExpr, ok := docDict.Pairs["content"]
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("value"),
			Message: "add() requires 'content' field",
			Hints:   []string{"Usage: search.add({url: @/path, title: @Title, content: @Content})"},
		}
	}
	contentObj := evaluator.Eval(contentExpr, docDict.Env)
	contentStr, ok := contentObj.(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "content must be a string",
		}
	}

	// Create document
	doc := &search.Document{
		URL:     urlStr.Value,
		Title:   titleStr.Value,
		Content: contentStr.Value,
	}

	// Extract optional fields
	if headingsExpr, ok := docDict.Pairs["headings"]; ok {
		headingsObj := evaluator.Eval(headingsExpr, docDict.Env)
		if headingsStr, ok := headingsObj.(*evaluator.String); ok {
			doc.Headings = headingsStr.Value
		}
	}

	if tagsExpr, ok := docDict.Pairs["tags"]; ok {
		tagsObj := evaluator.Eval(tagsExpr, docDict.Env)
		if tagsArr, ok := tagsObj.(*evaluator.Array); ok {
			for _, elem := range tagsArr.Elements {
				if tagStr, ok := elem.(*evaluator.String); ok {
					doc.Tags = append(doc.Tags, tagStr.Value)
				}
			}
		}
	}

	if dateExpr, ok := docDict.Pairs["date"]; ok {
		dateObj := evaluator.Eval(dateExpr, docDict.Env)
		if dateStr, ok := dateObj.(*evaluator.String); ok {
			// Try parsing common date formats
			formats := []string{
				"2006-01-02",
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, dateStr.Value); err == nil {
					doc.Date = t
					break
				}
			}
		}
	}

	// Index the document (Path is empty so it will be marked as source='manual')
	if err := instance.index.IndexDocument(doc); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to index document: %v", err),
		}
	}

	return &evaluator.Boolean{Value: true}
}

// searchUpdateMethod updates a document in the index
func searchUpdateMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) != 1 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("update() takes exactly 1 argument (got %d)", len(args)),
			Hints:   []string{"Usage: search.update({url: @/path, title: @New Title})"},
		}
	}

	// Parse document dictionary
	docDict, ok := args[0].(*evaluator.Dictionary)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "update() requires a dictionary argument",
			Hints:   []string{"Usage: search.update({url: @/path, title: @New Title})"},
		}
	}

	// Extract required URL field
	urlExpr, ok := docDict.Pairs["url"]
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("value"),
			Message: "update() requires 'url' field",
			Hints:   []string{"Usage: search.update({url: @/path, title: @New Title})"},
		}
	}
	urlObj := evaluator.Eval(urlExpr, docDict.Env)
	urlStr, ok := urlObj.(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "url must be a string",
		}
	}

	// Remove existing document
	if err := instance.index.RemoveDocument(urlStr.Value); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to remove existing document: %v", err),
		}
	}

	// Re-add with new fields (reuse add logic)
	return searchAddMethod(instance, args, env)
}

// searchRemoveMethod removes a document from the index by URL
func searchRemoveMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) != 1 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("remove() takes exactly 1 argument (got %d)", len(args)),
			Hints:   []string{"Usage: search.remove(@/path)"},
		}
	}

	// Parse URL
	urlStr, ok := args[0].(*evaluator.String)
	if !ok {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("type"),
			Message: "remove() requires a string argument (url)",
			Hints:   []string{"Usage: search.remove(@/path)"},
		}
	}

	// Remove the document
	if err := instance.index.RemoveDocument(urlStr.Value); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to remove document: %v", err),
		}
	}

	return &evaluator.Boolean{Value: true}
}

func searchStatsMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) != 0 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("stats() takes no arguments (got %d)", len(args)),
		}
	}

	stats, err := instance.index.Stats()
	if err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to get stats: %v", err),
		}
	}

	pairs := make(map[string]evaluator.Object)

	if docs, ok := stats["documents"].(int); ok {
		pairs["documents"] = &evaluator.Integer{Value: int64(docs)}
	}
	if size, ok := stats["size"].(string); ok {
		pairs["size"] = &evaluator.String{Value: size}
	}
	if lastIndexed, ok := stats["last_indexed"].(string); ok {
		pairs["last_indexed"] = &evaluator.String{Value: lastIndexed}
	}

	dict := evaluator.NewDictionaryFromObjects(pairs)
	dict.Env = env

	return dict
}

func searchReindexMethod(instance *SearchInstance, args []evaluator.Object, env *evaluator.Environment) evaluator.Object {
	if len(args) != 0 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("arity"),
			Message: fmt.Sprintf("reindex() takes no arguments (got %d)", len(args)),
		}
	}

	// Only works with watch paths
	if len(instance.options.Watch) == 0 {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: "reindex() requires watch paths to be configured",
			Hints:   []string{"reindex() rebuilds the index from watched folders", "For manual indexing, use add() to re-add documents"},
		}
	}

	// Lock to prevent concurrent reindexing
	instance.initMutex.Lock()
	defer instance.initMutex.Unlock()

	// Drop and recreate tables
	if err := instance.index.Reindex(); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to reindex tables: %v", err),
		}
	}

	// Reset initialization state
	instance.initialized = false
	instance.lastCheck = time.Time{}

	// Re-run auto indexing
	if err := instance.autoIndex(); err != nil {
		return &evaluator.Error{
			Class:   evaluator.ErrorClass("runtime"),
			Message: fmt.Sprintf("failed to reindex documents: %v", err),
		}
	}

	return &evaluator.Boolean{Value: true}
}
