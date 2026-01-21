package evaluator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// ============================================================================
// Method Metadata
// ============================================================================

// MethodInfo holds metadata about a method
type MethodInfo struct {
	Name        string
	Arity       string // e.g., "0", "1", "0-1", "1+"
	Description string
}

// BuiltinInfo holds metadata about a builtin function
type BuiltinInfo struct {
	Name        string
	Arity       string // e.g., "1", "1-2", "0+", "1+"
	Description string
	Params      []string // Parameter names, "?" suffix for optional
	Category    string   // Grouping: "file", "time", "conversion", etc.
	Deprecated  string   // If non-empty, deprecation message
}

// PropertyInfo holds metadata about a property
type PropertyInfo struct {
	Name        string
	Type        string // Return type, e.g., "array", "dictionary"
	Description string
}

// TypeProperties maps type names to their available properties
var TypeProperties = map[string][]PropertyInfo{
	"string": {
		// No direct properties on string primitives
	},
	"integer": {
		// No properties on integer primitives
	},
	"float": {
		// No properties on float primitives
	},
	"boolean": {
		// No properties on boolean primitives
	},
	"array": {
		// No direct properties on arrays
	},
	"dictionary": {
		// Dynamic properties based on keys
	},
	"datetime": {
		// Direct properties (stored in dict)
		{Name: "year", Type: "integer", Description: "Year number"},
		{Name: "month", Type: "integer", Description: "Month (1-12)"},
		{Name: "day", Type: "integer", Description: "Day of month (1-31)"},
		{Name: "hour", Type: "integer", Description: "Hour (0-23)"},
		{Name: "minute", Type: "integer", Description: "Minute (0-59)"},
		{Name: "second", Type: "integer", Description: "Second (0-59)"},
		{Name: "weekday", Type: "string", Description: "Day name (Monday, Tuesday, etc.)"},
		{Name: "unix", Type: "integer", Description: "Unix timestamp (seconds since 1970-01-01)"},
		{Name: "iso", Type: "string", Description: "ISO 8601 datetime string"},
		{Name: "kind", Type: "string", Description: "Datetime kind (date, datetime, time, time_seconds)"},
		// Computed properties
		{Name: "date", Type: "string", Description: "Date portion (YYYY-MM-DD)"},
		{Name: "time", Type: "string", Description: "Time portion (HH:MM or HH:MM:SS)"},
		{Name: "dayOfYear", Type: "integer", Description: "Day number within year (1-366)"},
		{Name: "week", Type: "integer", Description: "ISO week number (1-53)"},
		{Name: "timestamp", Type: "integer", Description: "Unix timestamp (alias for .unix)"},
	},
	"money": {
		{Name: "amount", Type: "integer", Description: "Amount in smallest currency unit (e.g., cents)"},
		{Name: "currency", Type: "string", Description: "ISO 4217 currency code (e.g., USD, EUR)"},
		{Name: "scale", Type: "integer", Description: "Number of decimal places for currency"},
	},
	"duration": {
		{Name: "months", Type: "integer", Description: "Month component (years are stored as 12*years)"},
		{Name: "seconds", Type: "integer", Description: "Seconds component (weeks/days/hours/minutes as seconds)"},
		{Name: "totalSeconds", Type: "integer", Description: "Total seconds (only present when months == 0)"},
		{Name: "days", Type: "integer", Description: "Total duration in days (null if months > 0)"},
		{Name: "hours", Type: "integer", Description: "Total duration in hours (null if months > 0)"},
		{Name: "minutes", Type: "integer", Description: "Total duration in minutes (null if months > 0)"},
	},
	"path": {
		{Name: "absolute", Type: "boolean", Description: "Whether path is absolute"},
		{Name: "segments", Type: "array", Description: "Path segments as array of strings"},
		{Name: "extension", Type: "string", Description: "File extension (without dot)"},
		{Name: "filename", Type: "string", Description: "Last segment (file or directory name)"},
		{Name: "parent", Type: "path", Description: "Parent directory path"},
	},
	"url": {
		{Name: "scheme", Type: "string", Description: "URL scheme (http, https, etc.)"},
		{Name: "host", Type: "string", Description: "Hostname"},
		{Name: "port", Type: "integer", Description: "Port number"},
		{Name: "path", Type: "path", Description: "URL path as path object"},
		{Name: "query", Type: "dictionary", Description: "Query parameters as dictionary"},
		{Name: "fragment", Type: "string", Description: "Fragment identifier (after #)"},
	},
	"file": {
		{Name: "path", Type: "path", Description: "File path"},
		{Name: "format", Type: "string", Description: "File format (json, yaml, csv, etc.)"},
		{Name: "exists", Type: "boolean", Description: "Whether file exists"},
		{Name: "size", Type: "integer", Description: "File size in bytes"},
	},
	"dir": {
		{Name: "path", Type: "path", Description: "Directory path"},
		{Name: "exists", Type: "boolean", Description: "Whether directory exists"},
	},
	"table": {
		{Name: "row", Type: "dictionary", Description: "First row (or NULL if empty)"},
		{Name: "rows", Type: "array", Description: "All rows as array of dictionaries"},
		{Name: "columns", Type: "array", Description: "Column names as array of strings"},
	},
	"regex": {
		{Name: "pattern", Type: "string", Description: "Regular expression pattern"},
		{Name: "flags", Type: "string", Description: "Regex flags"},
	},
}

// TypeMethods maps type names to their available methods
var TypeMethods = map[string][]MethodInfo{
	"string": {
		{Name: "toUpper", Arity: "0", Description: "Convert to uppercase"},
		{Name: "toLower", Arity: "0", Description: "Convert to lowercase"},
		{Name: "toTitle", Arity: "0", Description: "Convert to title case (capitalize first letter of each word)"},
		{Name: "trim", Arity: "0", Description: "Remove leading/trailing whitespace"},
		{Name: "split", Arity: "1", Description: "Split by delimiter into array"},
		{Name: "replace", Arity: "2", Description: "Replace all occurrences"},
		{Name: "length", Arity: "0", Description: "Get character count"},
		{Name: "includes", Arity: "1", Description: "Check if contains substring"},
		{Name: "highlight", Arity: "1-2", Description: "Wrap matches in HTML tag"},
		{Name: "paragraphs", Arity: "0", Description: "Convert blank lines to <p> tags"},
		{Name: "render", Arity: "0-1", Description: "Interpolate template with values"},
		{Name: "parseJSON", Arity: "0", Description: "Parse as JSON"},
		{Name: "parseCSV", Arity: "0-1", Description: "Parse as CSV"},
		{Name: "collapse", Arity: "0", Description: "Collapse whitespace to single spaces"},
		{Name: "normalizeSpace", Arity: "0", Description: "Collapse and trim whitespace"},
		{Name: "stripSpace", Arity: "0", Description: "Remove all whitespace"},
		{Name: "stripHtml", Arity: "0", Description: "Remove HTML tags"},
		{Name: "digits", Arity: "0", Description: "Extract only digits"},
		{Name: "slug", Arity: "0", Description: "Convert to URL-safe slug"},
		{Name: "htmlEncode", Arity: "0", Description: "Encode HTML entities (<, >, &, etc.)"},
		{Name: "htmlDecode", Arity: "0", Description: "Decode HTML entities"},
		{Name: "urlEncode", Arity: "0", Description: "URL encode (spaces become +)"},
		{Name: "urlDecode", Arity: "0", Description: "Decode URL-encoded string"},
		{Name: "urlPathEncode", Arity: "0", Description: "Encode URL path segment (/ becomes %2F)"},
		{Name: "urlQueryEncode", Arity: "0", Description: "Encode URL query value (& and = encoded)"},
		{Name: "outdent", Arity: "0", Description: "Remove common leading whitespace from all lines"},
		{Name: "indent", Arity: "1", Description: "Add spaces to beginning of all non-blank lines"},
	},
	"array": {
		{Name: "length", Arity: "0", Description: "Get element count"},
		{Name: "reverse", Arity: "0", Description: "Reverse order"},
		{Name: "sort", Arity: "0", Description: "Sort elements"},
		{Name: "sortBy", Arity: "1", Description: "Sort by key function"},
		{Name: "map", Arity: "1", Description: "Transform each element"},
		{Name: "filter", Arity: "1", Description: "Filter by predicate"},
		{Name: "reduce", Arity: "2", Description: "Reduce to single value with accumulator function"},
		{Name: "format", Arity: "0-2", Description: "Format as list (and/or/unit, locale)"},
		{Name: "join", Arity: "0-1", Description: "Join elements into string"},
		{Name: "toJSON", Arity: "0", Description: "Convert to JSON string"},
		{Name: "toCSV", Arity: "0-1", Description: "Convert to CSV string"},
		{Name: "shuffle", Arity: "0", Description: "Randomly shuffle elements"},
		{Name: "pick", Arity: "0-1", Description: "Pick random element(s)"},
		{Name: "take", Arity: "1", Description: "Take n unique random elements"},
		{Name: "insert", Arity: "2", Description: "Insert at index"},
	},
	"integer": {
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "format", Arity: "0-1", Description: "Format with locale"},
		{Name: "humanize", Arity: "0", Description: "Human-readable format (1K, 1M)"},
	},
	"float": {
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "format", Arity: "0-2", Description: "Format with decimals and locale"},
		{Name: "round", Arity: "0-1", Description: "Round to n decimals"},
		{Name: "floor", Arity: "0", Description: "Round down"},
		{Name: "ceil", Arity: "0", Description: "Round up"},
		{Name: "humanize", Arity: "0", Description: "Human-readable format (1K, 1M)"},
	},
	"dictionary": {
		{Name: "keys", Arity: "0", Description: "Get all keys"},
		{Name: "values", Arity: "0", Description: "Get all values"},
		{Name: "entries", Arity: "0", Description: "Get [key, value] pairs"},
		{Name: "has", Arity: "1", Description: "Check if key exists"},
		{Name: "delete", Arity: "1", Description: "Remove key"},
		{Name: "insertAfter", Arity: "2", Description: "Insert after key"},
		{Name: "insertBefore", Arity: "2", Description: "Insert before key"},
		{Name: "render", Arity: "0-1", Description: "Render template with values"},
		{Name: "toJSON", Arity: "0", Description: "Convert to JSON string"},
	},
	"money": {
		{Name: "format", Arity: "0-1", Description: "Format with locale"},
		{Name: "abs", Arity: "0", Description: "Absolute value"},
		{Name: "negate", Arity: "0", Description: "Negate amount"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"datetime": {
		{Name: "format", Arity: "0-2", Description: "Format with style and locale"},
		{Name: "dayOfYear", Arity: "0", Description: "Day of year (1-366)"},
		{Name: "week", Arity: "0", Description: "ISO week number"},
		{Name: "timestamp", Arity: "0", Description: "Unix timestamp"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"duration": {
		{Name: "format", Arity: "0-1", Description: "Format as relative time"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"path": {
		{Name: "toString", Arity: "0", Description: "Convert to string"},
		{Name: "join", Arity: "1+", Description: "Join path components"},
		{Name: "parent", Arity: "0", Description: "Get parent directory"},
		{Name: "isAbsolute", Arity: "0", Description: "Check if absolute path"},
		{Name: "isRelative", Arity: "0", Description: "Check if relative path"},
		{Name: "public", Arity: "0", Description: "Get public URL"},
		{Name: "toURL", Arity: "1", Description: "Convert to URL with prefix"},
		{Name: "match", Arity: "1", Description: "Match against pattern"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"url": {
		{Name: "origin", Arity: "0", Description: "Get origin (scheme://host:port)"},
		{Name: "pathname", Arity: "0", Description: "Get path component"},
		{Name: "toString", Arity: "0", Description: "Convert to string"},
		{Name: "withPath", Arity: "1", Description: "Create URL with new path"},
		{Name: "withQuery", Arity: "1", Description: "Create URL with query params"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"regex": {
		{Name: "test", Arity: "1", Description: "Test if string matches"},
		{Name: "match", Arity: "1", Description: "Find first match"},
		{Name: "matchAll", Arity: "1", Description: "Find all matches"},
		{Name: "replace", Arity: "2", Description: "Replace matches"},
		{Name: "split", Arity: "1", Description: "Split string by pattern"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"file": {
		{Name: "exists", Arity: "0", Description: "Check if file exists"},
		{Name: "read", Arity: "0", Description: "Read file contents"},
		{Name: "stat", Arity: "0", Description: "Get file metadata"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"directory": {
		{Name: "exists", Arity: "0", Description: "Check if directory exists"},
		{Name: "list", Arity: "0", Description: "List directory contents"},
		{Name: "toDict", Arity: "0", Description: "Convert to dictionary"},
	},
	"table": {
		{Name: "where", Arity: "1", Description: "Filter rows by predicate"},
		{Name: "orderBy", Arity: "1+", Description: "Sort rows by column(s)"},
		{Name: "select", Arity: "1+", Description: "Select specific columns"},
		{Name: "limit", Arity: "1-2", Description: "Limit rows (count, offset?)"},
		{Name: "count", Arity: "0", Description: "Count rows"},
		{Name: "sum", Arity: "1", Description: "Sum column values"},
		{Name: "avg", Arity: "1", Description: "Average column values"},
		{Name: "min", Arity: "1", Description: "Minimum column value"},
		{Name: "max", Arity: "1", Description: "Maximum column value"},
		{Name: "toHTML", Arity: "0-1", Description: "Convert to HTML table (footer: string|dict?)"},
		{Name: "toCSV", Arity: "0", Description: "Convert to CSV string"},
		{Name: "toMarkdown", Arity: "0", Description: "Convert to Markdown table"},
		{Name: "toJSON", Arity: "0", Description: "Convert to JSON array"},
		{Name: "appendRow", Arity: "1", Description: "Add row at end"},
		{Name: "insertRowAt", Arity: "2", Description: "Insert row at index"},
		{Name: "appendCol", Arity: "2", Description: "Add column at end"},
		{Name: "insertColAfter", Arity: "3", Description: "Insert column after another"},
		{Name: "insertColBefore", Arity: "3", Description: "Insert column before another"},
		{Name: "rowCount", Arity: "0", Description: "Get number of rows"},
		{Name: "columnCount", Arity: "0", Description: "Get number of columns"},
		{Name: "column", Arity: "1", Description: "Get array of values from column"},
		// Array-like methods
		{Name: "map", Arity: "1", Description: "Transform each row (fn) - preserves schema if Records returned"},
		{Name: "find", Arity: "1", Description: "Find first row matching predicate (fn) - returns row or null"},
		{Name: "any", Arity: "1", Description: "Check if any row matches predicate (fn) - returns boolean"},
		{Name: "all", Arity: "1", Description: "Check if all rows match predicate (fn) - returns boolean"},
		// Data manipulation methods
		{Name: "unique", Arity: "0-1", Description: "Remove duplicate rows (columns?)"},
		{Name: "renameCol", Arity: "2", Description: "Rename column (oldName, newName)"},
		{Name: "dropCol", Arity: "1+", Description: "Remove columns (col1, col2, ...)"},
		{Name: "groupBy", Arity: "1-2", Description: "Group rows by column(s) (cols, aggregationFn?)"},
	},
	"dbconnection": {
		{Name: "begin", Arity: "0", Description: "Begin transaction"},
		{Name: "commit", Arity: "0", Description: "Commit transaction"},
		{Name: "rollback", Arity: "0", Description: "Rollback transaction"},
		{Name: "close", Arity: "0", Description: "Close connection"},
		{Name: "ping", Arity: "0", Description: "Test connection"},
	},
	"sftpconnection": {
		{Name: "close", Arity: "0", Description: "Close connection"},
	},
	"sftpfile": {
		{Name: "mkdir", Arity: "0-1", Description: "Create directory"},
		{Name: "rmdir", Arity: "0-1", Description: "Remove directory"},
		{Name: "remove", Arity: "0", Description: "Remove file"},
	},
	"session": {
		{Name: "get", Arity: "1-2", Description: "Get session value (key, default?)"},
		{Name: "set", Arity: "2", Description: "Set session value (key, value)"},
		{Name: "delete", Arity: "1", Description: "Delete session key"},
		{Name: "has", Arity: "1", Description: "Check if key exists"},
		{Name: "clear", Arity: "0", Description: "Clear all session data"},
		{Name: "all", Arity: "0", Description: "Get all session data"},
		{Name: "flash", Arity: "2", Description: "Set flash message (key, value)"},
		{Name: "getFlash", Arity: "1", Description: "Get and clear flash message"},
		{Name: "getAllFlash", Arity: "0", Description: "Get all flash messages"},
		{Name: "hasFlash", Arity: "0", Description: "Check if flash messages exist"},
		{Name: "regenerate", Arity: "0", Description: "Regenerate session ID"},
	},
	"dev": {
		{Name: "log", Arity: "1-3", Description: "Log value to dev panel"},
		{Name: "clearLog", Arity: "0", Description: "Clear dev log"},
		{Name: "logPage", Arity: "0-1", Description: "Log page content"},
		{Name: "setLogRoute", Arity: "1", Description: "Set log route pattern"},
		{Name: "clearLogPage", Arity: "0", Description: "Clear page log"},
	},
	"tablemodule": {
		{Name: "fromDict", Arity: "1", Description: "Create table from dictionary"},
	},
	"function": {
		// Functions have no methods but we include them for completeness
	},
	"boolean": {
		// Booleans have no methods
	},
	"null": {
		// Null has no methods
	},
}

// ============================================================================
// Builtin Function Metadata
// ============================================================================

// BuiltinMetadata maps builtin function names to their metadata
var BuiltinMetadata = map[string]BuiltinInfo{
	// === File/Data Loading ===
	"JSON":     {Name: "JSON", Arity: "1-2", Description: "Load JSON from path or URL", Params: []string{"source", "options?"}, Category: "file"},
	"YAML":     {Name: "YAML", Arity: "1-2", Description: "Load YAML from path or URL", Params: []string{"source", "options?"}, Category: "file"},
	"PLN":      {Name: "PLN", Arity: "1-2", Description: "Load PLN (Parsley Literal Notation) from path", Params: []string{"path", "options?"}, Category: "file"},
	"CSV":      {Name: "CSV", Arity: "1-2", Description: "Load CSV from path or URL as table", Params: []string{"source", "options?"}, Category: "file"},
	"MD":       {Name: "MD", Arity: "1-2", Description: "Load markdown file and render to HTML", Params: []string{"path", "options?"}, Category: "file"},
	"markdown": {Name: "markdown", Arity: "1-2", Description: "Load markdown file with frontmatter", Params: []string{"path", "options?"}, Category: "file"},
	"lines":    {Name: "lines", Arity: "1-2", Description: "Load file as array of lines", Params: []string{"source", "options?"}, Category: "file"},
	"text":     {Name: "text", Arity: "1-2", Description: "Load file as text string", Params: []string{"source", "options?"}, Category: "file"},
	"bytes":    {Name: "bytes", Arity: "1", Description: "Load file as byte array", Params: []string{"path"}, Category: "file"},
	"SVG":      {Name: "SVG", Arity: "1-2", Description: "Load SVG file with optional attributes", Params: []string{"path", "attributes?"}, Category: "file"},
	"file":     {Name: "file", Arity: "1-2", Description: "Load file with auto-detected format", Params: []string{"path", "options?"}, Category: "file"},
	"dir":      {Name: "dir", Arity: "1", Description: "List directory contents", Params: []string{"path"}, Category: "file"},
	"fileList": {Name: "fileList", Arity: "1-2", Description: "List files in directory recursively", Params: []string{"path", "pattern?"}, Category: "file"},

	// === Time ===
	"date":     {Name: "date", Arity: "1-2", Description: "Parse date string with locale support", Params: []string{"input", "options?"}, Category: "time"},
	"time":     {Name: "time", Arity: "1", Description: "Parse time-only string (e.g., '3:45 PM')", Params: []string{"input"}, Category: "time"},
	"datetime": {Name: "datetime", Arity: "1-2", Description: "Parse datetime from string, timestamp, or dict with locale support", Params: []string{"input", "options?"}, Category: "time"},
	"now":      {Name: "now", Arity: "0", Description: "Current datetime", Params: []string{}, Category: "time", Deprecated: "Use @now datetime literal instead"},

	// === URLs ===
	"url": {Name: "url", Arity: "1", Description: "Parse URL string into components", Params: []string{"urlString"}, Category: "url"},

	// === Paths ===
	"path": {Name: "path", Arity: "1", Description: "Create path from string", Params: []string{"pathString"}, Category: "path"},

	// === Type Conversion ===
	"toInt":    {Name: "toInt", Arity: "1", Description: "Convert value to integer", Params: []string{"value"}, Category: "conversion"},
	"toFloat":  {Name: "toFloat", Arity: "1", Description: "Convert value to float", Params: []string{"value"}, Category: "conversion"},
	"toNumber": {Name: "toNumber", Arity: "1", Description: "Convert value to number (int or float)", Params: []string{"value"}, Category: "conversion"},
	"toString": {Name: "toString", Arity: "1", Description: "Convert value to string", Params: []string{"value"}, Category: "conversion"},
	"toArray":  {Name: "toArray", Arity: "1", Description: "Convert value to array", Params: []string{"value"}, Category: "conversion"},
	"toDict":   {Name: "toDict", Arity: "1", Description: "Convert array of [key,value] pairs to dictionary", Params: []string{"pairs"}, Category: "conversion"},

	// === Serialization (PLN) ===
	"serialize":   {Name: "serialize", Arity: "1", Description: "Convert value to PLN string", Params: []string{"value"}, Category: "serialization"},
	"deserialize": {Name: "deserialize", Arity: "1", Description: "Parse PLN string to value", Params: []string{"plnString"}, Category: "serialization"},

	// === Type Introspection ===
	"inspect":  {Name: "inspect", Arity: "1", Description: "Get introspection data for value", Params: []string{"value"}, Category: "introspection"},
	"describe": {Name: "describe", Arity: "1", Description: "Get human-readable description of value", Params: []string{"value"}, Category: "introspection"},
	"repr":     {Name: "repr", Arity: "1", Description: "Convert value to Parsley-parseable literal string", Params: []string{"value"}, Category: "conversion"},
	"builtins": {Name: "builtins", Arity: "0-1", Description: "List all builtin functions by category", Params: []string{"category?"}, Category: "introspection"},

	// === Output ===
	"print":   {Name: "print", Arity: "1+", Description: "Print values without newline", Params: []string{"values..."}, Category: "output"},
	"println": {Name: "println", Arity: "0+", Description: "Print values with newline", Params: []string{"values..."}, Category: "output"},
	"printf":  {Name: "printf", Arity: "1+", Description: "Print formatted string", Params: []string{"format", "values..."}, Category: "output"},
	"log":     {Name: "log", Arity: "1+", Description: "Log message", Params: []string{"values..."}, Category: "output"},
	"logLine": {Name: "logLine", Arity: "1+", Description: "Log message with newline", Params: []string{"values..."}, Category: "output"},

	// === Control Flow ===
	"fail": {Name: "fail", Arity: "1", Description: "Throw an error with message", Params: []string{"message"}, Category: "control"},

	// === Formatting ===
	"format": {Name: "format", Arity: "2+", Description: "Format string with placeholders", Params: []string{"template", "values..."}, Category: "format"},
	"tag":    {Name: "tag", Arity: "1-3", Description: "Create HTML tag", Params: []string{"name", "attributes?", "content?"}, Category: "format"},

	// === Regex ===
	"regex": {Name: "regex", Arity: "1-2", Description: "Create regex pattern", Params: []string{"pattern", "flags?"}, Category: "regex"},
	"match": {Name: "match", Arity: "2-3", Description: "Match string against pattern", Params: []string{"string", "pattern", "flags?"}, Category: "regex"},

	// === Money ===
	"money": {Name: "money", Arity: "1-2", Description: "Create money value", Params: []string{"amount", "currency?"}, Category: "money"},

	// === Assets ===
	"asset": {Name: "asset", Arity: "1", Description: "Get asset path with cache busting", Params: []string{"path"}, Category: "asset"},

	// === Connection Literals (internal) ===
	"sqlite":   {Name: "sqlite", Arity: "1", Description: "Create SQLite database connection", Params: []string{"path"}, Category: "connection"},
	"postgres": {Name: "postgres", Arity: "1", Description: "Create PostgreSQL database connection", Params: []string{"connectionString"}, Category: "connection"},
	"mysql":    {Name: "mysql", Arity: "1", Description: "Create MySQL database connection", Params: []string{"connectionString"}, Category: "connection"},
	"sftp":     {Name: "sftp", Arity: "1", Description: "Create SFTP connection", Params: []string{"connectionString"}, Category: "connection"},
	"shell":    {Name: "shell", Arity: "0", Description: "Create shell command executor", Params: []string{}, Category: "connection"},
}

// ============================================================================
// Type Detection
// ============================================================================

// getObjectTypeString returns a user-facing type name string for .type() method
// Returns lowercase semantic type names consistent with pseudo-type __type fields
func getObjectTypeString(obj Object) string {
	switch o := obj.(type) {
	case *String:
		return "string"
	case *Integer:
		return "integer"
	case *Float:
		return "float"
	case *Boolean:
		return "boolean"
	case *Array:
		return "array"
	case *Function:
		return "function"
	case *Builtin:
		return "builtin"
	case *StdlibBuiltin:
		return "builtin"
	case *Money:
		return "money"
	case *Table:
		return "table"
	case *TableBinding:
		return "table"
	case *DBConnection:
		return "database"
	case *SFTPConnection:
		return "sftp"
	case *SFTPFileHandle:
		return "file"
	case *SessionModule:
		return "session"
	case *DevModule:
		return "module"
	case *TableModule:
		return "module"
	case *StdlibRoot:
		return "module"
	case *BasilRoot:
		return "module"
	case *StdlibModuleDict:
		return "module"
	case *MdDoc:
		return "markdown"
	case *DSLSchema:
		return "schema"
	case *Dictionary:
		// For dictionaries with __type field, return that value
		if typeExpr, ok := o.Pairs["__type"]; ok {
			if typeStr, ok := typeExpr.(*ast.StringLiteral); ok {
				return typeStr.Value
			}
		}
		// Check for pseudo-types without evaluating (for efficiency)
		if isDatetimeDict(o) {
			return "datetime"
		}
		if isDurationDict(o) {
			return "duration"
		}
		if isPathDict(o) {
			return "path"
		}
		if isUrlDict(o) {
			return "url"
		}
		if isRegexDict(o) {
			return "regex"
		}
		if isFileDict(o) {
			return "file"
		}
		if isDirDict(o) {
			return "dir"
		}
		if isRequestDict(o) {
			return "request"
		}
		if isResponseDict(o) {
			return "response"
		}
		// Regular dictionary
		return "dictionary"
	case *Null:
		return "null"
	case *Error:
		return "error"
	default:
		// Fallback to ObjectType string
		return strings.ToLower(string(obj.Type()))
	}
}

// getObjectTypeName returns the type name and subtype (for typed dicts) of an object
func getObjectTypeName(obj Object, env *Environment) (typeName string, subType string) {
	switch o := obj.(type) {
	case *String:
		return "string", ""
	case *Integer:
		return "integer", ""
	case *Float:
		return "float", ""
	case *Boolean:
		return "boolean", ""
	case *Array:
		return "array", ""
	case *Function:
		return "function", ""
	case *Builtin:
		return "builtin", ""
	case *Money:
		return "money", ""
	case *Table:
		return "table", ""
	case *DBConnection:
		return "dbconnection", ""
	case *SFTPConnection:
		return "sftpconnection", ""
	case *SFTPFileHandle:
		return "sftpfile", ""
	case *SessionModule:
		return "session", ""
	case *DevModule:
		return "dev", ""
	case *TableModule:
		return "tablemodule", ""
	case *StdlibRoot:
		return "stdlib", ""
	case *BasilRoot:
		return "basil", ""
	case *StdlibModuleDict:
		return "module", ""
	case *Dictionary:
		// Check for typed dictionaries
		if isDatetimeDict(o) {
			return "dictionary", "datetime"
		}
		if isDurationDict(o) {
			return "dictionary", "duration"
		}
		if isPathDict(o) {
			return "dictionary", "path"
		}
		if isUrlDict(o) {
			return "dictionary", "url"
		}
		if isRegexDict(o) {
			return "dictionary", "regex"
		}
		if isFileDict(o) {
			return "dictionary", "file"
		}
		if isDirDict(o) {
			return "dictionary", "directory"
		}
		if isRequestDict(o) {
			return "dictionary", "request"
		}
		if isResponseDict(o) {
			return "dictionary", "response"
		}
		return "dictionary", ""
	case *Null:
		return "null", ""
	case *Error:
		return "error", ""
	default:
		return strings.ToLower(string(obj.Type())), ""
	}
}

// ============================================================================
// Inspect Function
// ============================================================================

// builtinInspect returns introspection data as a dictionary
func builtinInspect(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("inspect", len(args), 1)
	}

	obj := args[0]

	// Special handling for StdlibRoot - show available modules
	if root, ok := obj.(*StdlibRoot); ok {
		return inspectStdlibRoot(root)
	}

	// Special handling for BasilRoot - show available modules
	if root, ok := obj.(*BasilRoot); ok {
		return inspectBasilRoot(root)
	}

	// Special handling for StdlibModuleDict - show exports
	if mod, ok := obj.(*StdlibModuleDict); ok {
		return inspectStdlibModule(mod)
	}

	// Special handling for Builtin functions
	if builtin, ok := obj.(*Builtin); ok {
		return inspectBuiltin(builtin)
	}

	typeName, subType := getObjectTypeName(obj, nil)

	// Determine which method list to use
	methodKey := typeName
	if subType != "" {
		methodKey = subType
	}

	// Build methods array
	methodInfos, ok := TypeMethods[methodKey]
	if !ok {
		methodInfos = []MethodInfo{}
	}

	// Sort methods alphabetically
	sortedMethods := make([]MethodInfo, len(methodInfos))
	copy(sortedMethods, methodInfos)
	sort.Slice(sortedMethods, func(i, j int) bool {
		return sortedMethods[i].Name < sortedMethods[j].Name
	})

	// Build method dictionaries
	methodDicts := make([]Object, len(sortedMethods))
	for i, m := range sortedMethods {
		pairs := map[string]ast.Expression{
			"name":        createLiteralExpression(&String{Value: m.Name}),
			"arity":       createLiteralExpression(&String{Value: m.Arity}),
			"description": createLiteralExpression(&String{Value: m.Description}),
		}
		methodDicts[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build properties array
	propertyInfos, hasProps := TypeProperties[methodKey]
	var propertyDicts []Object
	if hasProps {
		// Sort properties alphabetically
		sortedProps := make([]PropertyInfo, len(propertyInfos))
		copy(sortedProps, propertyInfos)
		sort.Slice(sortedProps, func(i, j int) bool {
			return sortedProps[i].Name < sortedProps[j].Name
		})

		propertyDicts = make([]Object, len(sortedProps))
		for i, p := range sortedProps {
			propPairs := map[string]ast.Expression{
				"name":        createLiteralExpression(&String{Value: p.Name}),
				"type":        createLiteralExpression(&String{Value: p.Type}),
				"description": createLiteralExpression(&String{Value: p.Description}),
			}
			propertyDicts[i] = &Dictionary{Pairs: propPairs, Env: NewEnvironment()}
		}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: typeName}),
		"methods": createLiteralExpression(&Array{Elements: methodDicts}),
	}

	// Add properties if present
	if hasProps {
		pairs["properties"] = createLiteralExpression(&Array{Elements: propertyDicts})
	}

	// Add subtype if present
	if subType != "" {
		pairs["subtype"] = createLiteralExpression(&String{Value: subType})
	}

	// For functions, add parameter info
	if fn, ok := obj.(*Function); ok {
		params := make([]Object, len(fn.Params))
		for i, p := range fn.Params {
			params[i] = &String{Value: p.String()}
		}
		pairs["params"] = createLiteralExpression(&Array{Elements: params})
	}

	// For dictionaries, add keys
	if dict, ok := obj.(*Dictionary); ok && subType == "" {
		keys := dict.Keys()
		keyObjs := make([]Object, len(keys))
		for i, k := range keys {
			keyObjs[i] = &String{Value: k}
		}
		pairs["keys"] = createLiteralExpression(&Array{Elements: keyObjs})
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// inspectStdlibModule returns introspection data for a stdlib module
func inspectStdlibModule(mod *StdlibModuleDict) Object {
	// Build exports array - sorted list of {name, type, description?}
	keys := make([]string, 0, len(mod.Exports))
	for k := range mod.Exports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	exports := make([]Object, len(keys))
	for i, name := range keys {
		obj := mod.Exports[name]
		exportType := strings.ToLower(string(obj.Type()))

		pairs := map[string]ast.Expression{
			"name": createLiteralExpression(&String{Value: name}),
			"type": createLiteralExpression(&String{Value: exportType}),
		}

		// Check if we have metadata for this export
		if info, ok := StdlibExports[name]; ok {
			pairs["arity"] = createLiteralExpression(&String{Value: info.Arity})
			pairs["description"] = createLiteralExpression(&String{Value: info.Description})
		}

		exports[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: "module"}),
		"exports": createLiteralExpression(&Array{Elements: exports}),
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// inspectBuiltin returns introspection data for a builtin function
func inspectBuiltin(builtin *Builtin) Object {
	// Try to find the builtin name by searching getBuiltins()
	// This is a bit indirect but necessary since Builtin doesn't store its name
	builtins := getBuiltins()
	var name string
	for n, b := range builtins {
		if b == builtin {
			name = n
			break
		}
	}

	if name == "" {
		// Builtin not found in metadata
		pairs := map[string]ast.Expression{
			"type": createLiteralExpression(&String{Value: "builtin"}),
			"name": createLiteralExpression(&String{Value: "<unknown>"}),
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Look up metadata
	metadata, hasMetadata := BuiltinMetadata[name]
	if !hasMetadata {
		// No metadata available
		pairs := map[string]ast.Expression{
			"type": createLiteralExpression(&String{Value: "builtin"}),
			"name": createLiteralExpression(&String{Value: name}),
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build params array
	paramObjs := make([]Object, len(metadata.Params))
	for i, p := range metadata.Params {
		paramObjs[i] = &String{Value: p}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":        createLiteralExpression(&String{Value: "builtin"}),
		"name":        createLiteralExpression(&String{Value: metadata.Name}),
		"arity":       createLiteralExpression(&String{Value: metadata.Arity}),
		"description": createLiteralExpression(&String{Value: metadata.Description}),
		"params":      createLiteralExpression(&Array{Elements: paramObjs}),
		"category":    createLiteralExpression(&String{Value: metadata.Category}),
	}

	if metadata.Deprecated != "" {
		pairs["deprecated"] = createLiteralExpression(&String{Value: metadata.Deprecated})
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// inspectStdlibRoot returns introspection data for the stdlib root
func inspectStdlibRoot(root *StdlibRoot) Object {
	// Build modules array
	modules := make([]Object, len(root.Modules))
	for i, name := range root.Modules {
		info, hasInfo := StdlibModuleDescriptions[name]
		pairs := map[string]ast.Expression{
			"name": createLiteralExpression(&String{Value: name}),
		}
		if hasInfo {
			pairs["description"] = createLiteralExpression(&String{Value: info})
		}
		modules[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: "stdlib"}),
		"modules": createLiteralExpression(&Array{Elements: modules}),
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// inspectBasilRoot returns introspection data for the basil root
func inspectBasilRoot(root *BasilRoot) Object {
	modules := make([]Object, len(root.Modules))
	for i, name := range root.Modules {
		info, hasInfo := BasilModuleDescriptions[name]
		pairs := map[string]ast.Expression{
			"name": createLiteralExpression(&String{Value: name}),
		}
		if hasInfo {
			pairs["description"] = createLiteralExpression(&String{Value: info})
		}
		modules[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	}

	pairs := map[string]ast.Expression{
		"type":    createLiteralExpression(&String{Value: "basil"}),
		"modules": createLiteralExpression(&Array{Elements: modules}),
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// describeStdlibRoot pretty prints the stdlib module listing
func describeStdlibRoot(root *StdlibRoot) Object {
	var sb strings.Builder
	sb.WriteString("Parsley Standard Library (@std)\n\n")
	sb.WriteString("Available modules:\n")

	// Find max name length for alignment
	maxNameLen := 0
	for _, name := range root.Modules {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	for _, name := range root.Modules {
		padding := strings.Repeat(" ", maxNameLen-len(name)+2)
		if desc, ok := StdlibModuleDescriptions[name]; ok {
			sb.WriteString(fmt.Sprintf("  @std/%s%s- %s\n", name, padding, desc))
		} else {
			sb.WriteString(fmt.Sprintf("  @std/%s\n", name))
		}
	}

	sb.WriteString("\nUsage: import @std/<module>\n")
	sb.WriteString("Example: let { floor, ceil } = import @std/math\n")

	return &String{Value: sb.String()}
}

// describeBasilRoot pretty prints the basil module listing
func describeBasilRoot(root *BasilRoot) Object {
	var sb strings.Builder
	sb.WriteString("Basil Server Namespace (@basil)\n\n")
	sb.WriteString("Available modules:\n")

	maxNameLen := 0
	for _, name := range root.Modules {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	for _, name := range root.Modules {
		padding := strings.Repeat(" ", maxNameLen-len(name)+2)
		if desc, ok := BasilModuleDescriptions[name]; ok {
			sb.WriteString(fmt.Sprintf("  @basil/%s%s- %s\n", name, padding, desc))
		} else {
			sb.WriteString(fmt.Sprintf("  @basil/%s\n", name))
		}
	}

	sb.WriteString("\nUsage: import @basil/<module>\n")
	sb.WriteString("Example: let { route, method } = import @basil/http\n")

	return &String{Value: sb.String()}
}

// StdlibModuleDescriptions contains descriptions for each stdlib module
var StdlibModuleDescriptions = map[string]string{
	"api":    "HTTP client for API requests",
	"dev":    "Development tools (logging, debugging)",
	"id":     "ID generation (UUID, nanoid, etc.)",
	"math":   "Mathematical functions and constants",
	"schema": "Schema validation and type checking",
	"table":  "Table data structure with query methods",
	"valid":  "Validation functions for strings, numbers, formats",
}

// BasilModuleDescriptions contains descriptions for each basil namespace module
var BasilModuleDescriptions = map[string]string{
	"http": "HTTP request context (request, response, route, method). Use @params for query/form data.",
	"auth": "Auth context, db, session, and user shortcuts",
}

// describeBuiltin returns human-readable documentation for a builtin function
func describeBuiltin(builtin *Builtin) Object {
	// Find the builtin name
	builtins := getBuiltins()
	var name string
	for n, b := range builtins {
		if b == builtin {
			name = n
			break
		}
	}

	if name == "" {
		return &String{Value: "Builtin function (name unknown)"}
	}

	// Look up metadata
	metadata, hasMetadata := BuiltinMetadata[name]
	if !hasMetadata {
		return &String{Value: fmt.Sprintf("Builtin: %s (no documentation available)", name)}
	}

	var sb strings.Builder

	// Function signature
	sb.WriteString(fmt.Sprintf("%s(", metadata.Name))
	sb.WriteString(strings.Join(metadata.Params, ", "))
	sb.WriteString(")\n\n")

	// Description
	sb.WriteString(fmt.Sprintf("%s\n\n", metadata.Description))

	// Arity
	sb.WriteString(fmt.Sprintf("Arity: %s\n", metadata.Arity))

	// Category
	sb.WriteString(fmt.Sprintf("Category: %s\n", metadata.Category))

	// Deprecation warning
	if metadata.Deprecated != "" {
		sb.WriteString(fmt.Sprintf("\n⚠ DEPRECATED: %s\n", metadata.Deprecated))
	}

	return &String{Value: sb.String()}
}

// ============================================================================
// Describe Function (Pretty Print)
// ============================================================================

// builtinDescribe pretty prints introspection data
func builtinDescribe(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("describe", len(args), 1)
	}

	obj := args[0]

	// Special handling for StdlibRoot - show available modules
	if root, ok := obj.(*StdlibRoot); ok {
		return describeStdlibRoot(root)
	}

	// Special handling for BasilRoot - show available modules
	if root, ok := obj.(*BasilRoot); ok {
		return describeBasilRoot(root)
	}

	// Special handling for StdlibModuleDict - show exports
	if mod, ok := obj.(*StdlibModuleDict); ok {
		return describeStdlibModule(mod)
	}

	// Special handling for Builtin functions
	if builtin, ok := obj.(*Builtin); ok {
		return describeBuiltin(builtin)
	}

	typeName, subType := getObjectTypeName(obj, nil)

	var sb strings.Builder

	// Type header
	if subType != "" {
		sb.WriteString(fmt.Sprintf("Type: %s (%s)\n", subType, typeName))
	} else {
		sb.WriteString(fmt.Sprintf("Type: %s\n", typeName))
	}

	// For functions, show parameters
	if fn, ok := obj.(*Function); ok {
		sb.WriteString("Parameters: ")
		if len(fn.Params) == 0 {
			sb.WriteString("(none)")
		} else {
			params := make([]string, len(fn.Params))
			for i, p := range fn.Params {
				params[i] = p.String()
			}
			sb.WriteString(strings.Join(params, ", "))
		}
		sb.WriteString("\n")
	}

	// For dictionaries without subtype, show keys
	if dict, ok := obj.(*Dictionary); ok && subType == "" {
		keys := dict.Keys()
		sb.WriteString(fmt.Sprintf("Keys: %s\n", strings.Join(keys, ", ")))
	}

	// Determine method/property key
	methodKey := typeName
	if subType != "" {
		methodKey = subType
	}

	// Properties
	propertyInfos, hasProps := TypeProperties[methodKey]
	if hasProps && len(propertyInfos) > 0 {
		sb.WriteString("\nProperties:\n")

		// Sort properties alphabetically
		sortedProps := make([]PropertyInfo, len(propertyInfos))
		copy(sortedProps, propertyInfos)
		sort.Slice(sortedProps, func(i, j int) bool {
			return sortedProps[i].Name < sortedProps[j].Name
		})

		// Find max name length for alignment
		maxPropLen := 0
		for _, p := range sortedProps {
			nameWithType := fmt.Sprintf(".%s: %s", p.Name, p.Type)
			if len(nameWithType) > maxPropLen {
				maxPropLen = len(nameWithType)
			}
		}

		for _, p := range sortedProps {
			nameWithType := fmt.Sprintf(".%s: %s", p.Name, p.Type)
			padding := strings.Repeat(" ", maxPropLen-len(nameWithType)+2)
			sb.WriteString(fmt.Sprintf("  %s%s- %s\n", nameWithType, padding, p.Description))
		}
	}

	// Methods
	methodInfos, ok := TypeMethods[methodKey]
	if !ok || len(methodInfos) == 0 {
		sb.WriteString("Methods: (none)\n")
	} else {
		sb.WriteString("\nMethods:\n")

		// Sort methods alphabetically
		sortedMethods := make([]MethodInfo, len(methodInfos))
		copy(sortedMethods, methodInfos)
		sort.Slice(sortedMethods, func(i, j int) bool {
			return sortedMethods[i].Name < sortedMethods[j].Name
		})

		// Find max name length for alignment
		maxNameLen := 0
		for _, m := range sortedMethods {
			nameWithArity := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			if len(nameWithArity) > maxNameLen {
				maxNameLen = len(nameWithArity)
			}
		}

		for _, m := range sortedMethods {
			nameWithArity := fmt.Sprintf(".%s(%s)", m.Name, arityToParams(m.Arity))
			padding := strings.Repeat(" ", maxNameLen-len(nameWithArity)+2)
			sb.WriteString(fmt.Sprintf("  %s%s- %s\n", nameWithArity, padding, m.Description))
		}
	}

	return &String{Value: sb.String()}
}

// arityToParams converts arity string to parameter placeholder
func arityToParams(arity string) string {
	switch arity {
	case "0":
		return ""
	case "1":
		return "arg"
	case "2":
		return "arg1, arg2"
	case "0-1":
		return "arg?"
	case "0-2":
		return "arg1?, arg2?"
	case "1-2":
		return "arg1, arg2?"
	case "1+":
		return "arg1, ..."
	default:
		return "..."
	}
}

// describeStdlibModule pretty prints a stdlib module's exports
func describeStdlibModule(mod *StdlibModuleDict) Object {
	var sb strings.Builder
	sb.WriteString("Type: module\n\nExports:\n")

	// Sort exports alphabetically
	keys := make([]string, 0, len(mod.Exports))
	for k := range mod.Exports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Group by type (constants vs functions)
	var constants []string
	var functions []string
	for _, name := range keys {
		obj := mod.Exports[name]
		switch obj.(type) {
		case *Builtin, *Function:
			functions = append(functions, name)
		default:
			constants = append(constants, name)
		}
	}

	// Find max name length for alignment
	maxNameLen := 0
	for _, name := range keys {
		info, hasInfo := StdlibExports[name]
		var display string
		if hasInfo && info.Arity != "" {
			display = fmt.Sprintf("%s(%s)", name, arityToParams(info.Arity))
		} else {
			display = name
		}
		if len(display) > maxNameLen {
			maxNameLen = len(display)
		}
	}

	// Print constants first
	if len(constants) > 0 {
		sb.WriteString("  Constants:\n")
		for _, name := range constants {
			obj := mod.Exports[name]
			padding := strings.Repeat(" ", maxNameLen-len(name)+2)
			sb.WriteString(fmt.Sprintf("    %s%s= %s\n", name, padding, obj.Inspect()))
		}
		sb.WriteString("\n")
	}

	// Print functions
	if len(functions) > 0 {
		sb.WriteString("  Functions:\n")
		for _, name := range functions {
			info, hasInfo := StdlibExports[name]
			var display string
			var desc string
			if hasInfo {
				display = fmt.Sprintf("%s(%s)", name, arityToParams(info.Arity))
				desc = info.Description
			} else {
				display = name + "(...)"
				desc = ""
			}
			padding := strings.Repeat(" ", maxNameLen-len(display)+2)
			if desc != "" {
				sb.WriteString(fmt.Sprintf("    %s%s- %s\n", display, padding, desc))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", display))
			}
		}
	}

	return &String{Value: sb.String()}
}

// ============================================================================
// Stdlib Export Metadata
// ============================================================================

// StdlibExports contains metadata for stdlib module exports
var StdlibExports = map[string]MethodInfo{
	// math module - Constants
	"PI":  {Arity: "", Description: "Pi (3.14159...)"},
	"E":   {Arity: "", Description: "Euler's number (2.71828...)"},
	"TAU": {Arity: "", Description: "Tau (2*Pi)"},

	// math module - Rounding
	"floor": {Arity: "1", Description: "Round down to integer"},
	"ceil":  {Arity: "1", Description: "Round up to integer"},
	"round": {Arity: "1-2", Description: "Round to nearest (decimals?)"},
	"trunc": {Arity: "1", Description: "Truncate to integer"},

	// math module - Comparison & Clamping
	"abs":   {Arity: "1", Description: "Absolute value"},
	"sign":  {Arity: "1", Description: "Sign (-1, 0, or 1)"},
	"clamp": {Arity: "3", Description: "Clamp value between min and max"},

	// math module - Aggregation
	"min":     {Arity: "1+", Description: "Minimum of values or array"},
	"max":     {Arity: "1+", Description: "Maximum of values or array"},
	"sum":     {Arity: "1+", Description: "Sum of values or array"},
	"avg":     {Arity: "1+", Description: "Average of values or array"},
	"mean":    {Arity: "1+", Description: "Mean (alias for avg)"},
	"product": {Arity: "1+", Description: "Product of values or array"},
	"count":   {Arity: "1", Description: "Count elements in array"},

	// math module - Statistics
	"median":   {Arity: "1", Description: "Median of array"},
	"mode":     {Arity: "1", Description: "Mode of array"},
	"stddev":   {Arity: "1", Description: "Standard deviation"},
	"variance": {Arity: "1", Description: "Variance"},
	"range":    {Arity: "1", Description: "Range (max - min)"},

	// math module - Random
	"random":    {Arity: "0", Description: "Random float 0-1"},
	"randomInt": {Arity: "1-2", Description: "Random int (max) or (min, max)"},
	"seed":      {Arity: "1", Description: "Seed random generator"},

	// math module - Powers & Logarithms
	"sqrt":  {Arity: "1", Description: "Square root"},
	"pow":   {Arity: "2", Description: "Power (base, exponent)"},
	"exp":   {Arity: "1", Description: "e^x"},
	"log":   {Arity: "1", Description: "Natural logarithm"},
	"log10": {Arity: "1", Description: "Base-10 logarithm"},

	// math module - Trigonometry
	"sin":   {Arity: "1", Description: "Sine (radians)"},
	"cos":   {Arity: "1", Description: "Cosine (radians)"},
	"tan":   {Arity: "1", Description: "Tangent (radians)"},
	"asin":  {Arity: "1", Description: "Arc sine"},
	"acos":  {Arity: "1", Description: "Arc cosine"},
	"atan":  {Arity: "1", Description: "Arc tangent"},
	"atan2": {Arity: "2", Description: "Arc tangent of y/x"},

	// math module - Angular Conversion
	"degrees": {Arity: "1", Description: "Radians to degrees"},
	"radians": {Arity: "1", Description: "Degrees to radians"},

	// math module - Geometry & Interpolation
	"hypot": {Arity: "2", Description: "Hypotenuse length"},
	"dist":  {Arity: "4", Description: "Distance between points"},
	"lerp":  {Arity: "3", Description: "Linear interpolation"},
	"map":   {Arity: "5", Description: "Map value from one range to another"},

	// valid module - Type validators
	"string":  {Arity: "1", Description: "Check if value is string"},
	"number":  {Arity: "1", Description: "Check if value is number"},
	"integer": {Arity: "1", Description: "Check if value is integer"},
	"boolean": {Arity: "1", Description: "Check if value is boolean"},
	"array":   {Arity: "1", Description: "Check if value is array"},
	"dict":    {Arity: "1", Description: "Check if value is dictionary"},

	// valid module - String validators
	"empty":        {Arity: "1", Description: "Check if string is empty"},
	"minLen":       {Arity: "2", Description: "Check minimum length"},
	"maxLen":       {Arity: "2", Description: "Check maximum length"},
	"length":       {Arity: "2-3", Description: "Check length (exact or range)"},
	"matches":      {Arity: "2", Description: "Check regex match"},
	"alpha":        {Arity: "1", Description: "Check if only letters"},
	"alphanumeric": {Arity: "1", Description: "Check if only letters/numbers"},
	"numeric":      {Arity: "1", Description: "Check if only digits"},

	// valid module - Number validators
	// "min" and "max" already defined in math
	"between":  {Arity: "3", Description: "Check if number in range"},
	"positive": {Arity: "1", Description: "Check if positive"},
	"negative": {Arity: "1", Description: "Check if negative"},

	// valid module - Format validators
	"email":      {Arity: "1", Description: "Check email format"},
	"url":        {Arity: "1", Description: "Check URL format"},
	"uuid":       {Arity: "1", Description: "Check UUID format"},
	"phone":      {Arity: "1-2", Description: "Check phone format (locale?)"},
	"creditCard": {Arity: "1", Description: "Check credit card format"},
	"date":       {Arity: "1-2", Description: "Check date format"},
	"time":       {Arity: "1", Description: "Check time format"},

	// valid module - Locale-aware validators
	"postalCode": {Arity: "1-2", Description: "Check postal code (locale?)"},
	"parseDate":  {Arity: "1-2", Description: "Parse date string (locale?)"},

	// valid module - Collection validators
	"contains": {Arity: "2", Description: "Check if array contains value"},
	"oneOf":    {Arity: "2", Description: "Check if value is one of array"},
}
