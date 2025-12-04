package evaluator

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/goodsign/monday"
	"github.com/pkg/sftp"
	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/locale"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Database connection cache
var (
	dbConnectionsMu sync.RWMutex
	dbConnections   = make(map[string]*sql.DB)
)

// SFTP connection cache
var (
	sftpConnectionsMu sync.RWMutex
	sftpConnections   = make(map[string]*SFTPConnection)
)

// ObjectType represents the type of objects in our language
type ObjectType string

const (
	INTEGER_OBJ          = "INTEGER"
	FLOAT_OBJ            = "FLOAT"
	BOOLEAN_OBJ          = "BOOLEAN"
	STRING_OBJ           = "STRING"
	NULL_OBJ             = "NULL"
	RETURN_OBJ           = "RETURN_VALUE"
	ERROR_OBJ            = "ERROR"
	FUNCTION_OBJ         = "FUNCTION"
	BUILTIN_OBJ          = "BUILTIN"
	ARRAY_OBJ            = "ARRAY"
	DICTIONARY_OBJ       = "DICTIONARY"
	DB_CONNECTION_OBJ    = "DB_CONNECTION"
	SFTP_CONNECTION_OBJ  = "SFTP_CONNECTION"
	SFTP_FILE_HANDLE_OBJ = "SFTP_FILE_HANDLE"
	TABLE_OBJ            = "TABLE"
)

// Object represents all values in our language
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer represents integer objects
type Integer struct {
	Value int64
}

func (i *Integer) Inspect() string  { return strconv.FormatInt(i.Value, 10) }
func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

// Float represents floating-point objects
type Float struct {
	Value float64
}

func (f *Float) Inspect() string  { return fmt.Sprintf("%g", f.Value) }
func (f *Float) Type() ObjectType { return FLOAT_OBJ }

// Boolean represents boolean objects
type Boolean struct {
	Value bool
}

func (b *Boolean) Inspect() string  { return strconv.FormatBool(b.Value) }
func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

// String represents string objects
type String struct {
	Value string
}

func (s *String) Inspect() string  { return s.Value }
func (s *String) Type() ObjectType { return STRING_OBJ }

// Null represents null/nil objects
type Null struct{}

func (n *Null) Inspect() string  { return "null" }
func (n *Null) Type() ObjectType { return NULL_OBJ }

// ReturnValue wraps other objects when returned
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// Error represents error objects with structured error information.
// It maintains backward compatibility while supporting the new structured error system.
type Error struct {
	Message string
	Line    int
	Column  int
	// New structured error fields
	Class ErrorClass     // Error category (default: ClassType)
	Code  string         // Error code (e.g., "TYPE-0001")
	Hints []string       // Suggestions for fixing the error
	File  string         // File path (if known)
	Data  map[string]any // Template variables for custom rendering
}

// ErrorClass categorizes errors for filtering and templating.
type ErrorClass = perrors.ErrorClass

// Error class constants
const (
	ClassParse     = perrors.ClassParse
	ClassType      = perrors.ClassType
	ClassArity     = perrors.ClassArity
	ClassUndefined = perrors.ClassUndefined
	ClassIO        = perrors.ClassIO
	ClassDatabase  = perrors.ClassDatabase
	ClassNetwork   = perrors.ClassNetwork
	ClassSecurity  = perrors.ClassSecurity
	ClassIndex     = perrors.ClassIndex
	ClassFormat    = perrors.ClassFormat
	ClassOperator  = perrors.ClassOperator
	ClassState     = perrors.ClassState
	ClassImport    = perrors.ClassImport
)

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return "ERROR: " + e.Message
}

// ToParsleyError converts this Error to a ParsleyError for structured error handling.
func (e *Error) ToParsleyError() *perrors.ParsleyError {
	class := e.Class
	if class == "" {
		class = perrors.ClassType // Default class
	}
	return &perrors.ParsleyError{
		Class:   class,
		Code:    e.Code,
		Message: e.Message,
		Hints:   e.Hints,
		Line:    e.Line,
		Column:  e.Column,
		File:    e.File,
		Data:    e.Data,
	}
}

// Function represents function objects
type Function struct {
	Params []*ast.FunctionParameter // parameter list with destructuring support
	Body   *ast.BlockStatement
	Env    *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	return fmt.Sprintf("fn(%v) {\n%s\n}", f.Params, f.Body.String())
}

// ParamCount returns the number of parameters for this function
func (f *Function) ParamCount() int {
	return len(f.Params)
}

// BuiltinFunction represents a built-in function
type BuiltinFunction func(args ...Object) Object

// Builtin represents built-in function objects
type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Array represents array objects
type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out strings.Builder
	elements := []string{}
	for _, e := range a.Elements {
		if e != nil {
			elements = append(elements, e.Inspect())
		} else {
			elements = append(elements, "nil")
		}
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// Dictionary represents dictionary objects with lazy evaluation
type Dictionary struct {
	Pairs map[string]ast.Expression // Store expressions for lazy evaluation
	Env   *Environment              // Environment for evaluation (for 'this' binding)
}

func (d *Dictionary) Type() ObjectType { return DICTIONARY_OBJ }
func (d *Dictionary) Inspect() string {
	var out strings.Builder
	pairs := []string{}

	// Sort keys for consistent output
	keys := make([]string, 0, len(d.Pairs))
	for key := range d.Pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		expr := d.Pairs[key]
		// For inspection, we show the expression, not the evaluated value
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, expr.String()))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// Table represents a tabular data structure wrapping an array of dictionaries.
// Provides SQL-like operations (where, orderBy, select, etc.) with immutable semantics.
type Table struct {
	Rows    []*Dictionary // Array of dictionaries (each row is a dict)
	Columns []string      // Column order (from first row or select())
}

func (t *Table) Type() ObjectType { return TABLE_OBJ }
func (t *Table) Inspect() string {
	return fmt.Sprintf("Table(%d rows)", len(t.Rows))
}

// Copy creates a deep copy of the Table for immutability
func (t *Table) Copy() *Table {
	newRows := make([]*Dictionary, len(t.Rows))
	copy(newRows, t.Rows) // Shallow copy of slice - rows themselves are immutable dicts
	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{Rows: newRows, Columns: newColumns}
}

// DBConnection represents a database connection
type DBConnection struct {
	DB            *sql.DB
	Driver        string // "sqlite", "postgres", "mysql"
	DSN           string // Data Source Name
	InTransaction bool
	LastError     string
	Managed       bool // If true, connection is managed by host application (won't be closed by Parsley)
}

func (dbc *DBConnection) Type() ObjectType { return DB_CONNECTION_OBJ }
func (dbc *DBConnection) Inspect() string {
	return fmt.Sprintf("<DBConnection driver=%s>", dbc.Driver)
}

// NewManagedDBConnection creates a DBConnection that is managed by the host application.
// Managed connections cannot be closed by Parsley scripts - the host is responsible
// for managing the connection lifecycle.
func NewManagedDBConnection(db *sql.DB, driver string) *DBConnection {
	return &DBConnection{
		DB:            db,
		Driver:        driver,
		DSN:           "", // Not applicable for managed connections
		InTransaction: false,
		LastError:     "",
		Managed:       true,
	}
}

// SFTPConnection represents an SFTP connection
type SFTPConnection struct {
	Client    *sftp.Client
	SSHClient *ssh.Client
	Host      string
	Port      int
	User      string
	Connected bool
	LastError string
}

func (sc *SFTPConnection) Type() ObjectType { return SFTP_CONNECTION_OBJ }
func (sc *SFTPConnection) Inspect() string {
	status := "connected"
	if !sc.Connected {
		status = "disconnected"
	}
	return fmt.Sprintf("SFTP(%s@%s:%d) [%s]", sc.User, sc.Host, sc.Port, status)
}

// SFTPFileHandle represents a remote file handle via SFTP
type SFTPFileHandle struct {
	Connection *SFTPConnection
	Path       string
	Format     string // "json", "csv", "text", "lines", "bytes", "" (defaults to "text")
	Options    *Dictionary
}

func (sfh *SFTPFileHandle) Type() ObjectType { return SFTP_FILE_HANDLE_OBJ }
func (sfh *SFTPFileHandle) Inspect() string {
	format := sfh.Format
	if format == "" {
		format = "text"
	}
	return fmt.Sprintf("SFTPFileHandle(%s@%s:%s).%s",
		sfh.Connection.User, sfh.Connection.Host, sfh.Path, format)
}

// SecurityPolicy defines file system access restrictions
type SecurityPolicy struct {
	RestrictRead    []string // Denied read directories (blacklist)
	NoRead          bool     // Deny all reads
	AllowWrite      []string // Allowed write directories (whitelist)
	AllowWriteAll   bool     // Allow all writes
	AllowExecute    []string // Allowed execute directories (whitelist)
	AllowExecuteAll bool     // Allow all executes
}

// Logger interface for log()/logLine() output
type Logger interface {
	Log(values ...interface{})
	LogLine(values ...interface{})
}

// defaultStdoutLogger is the default logger that writes to stdout
type defaultStdoutLogger struct{}

func (l *defaultStdoutLogger) Log(values ...interface{}) {
	for i, v := range values {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(v)
	}
}

func (l *defaultStdoutLogger) LogLine(values ...interface{}) {
	for i, v := range values {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(v)
	}
	fmt.Println()
}

// DefaultLogger is the default stdout logger
var DefaultLogger Logger = &defaultStdoutLogger{}

// Environment represents the environment for variable bindings
type Environment struct {
	store       map[string]Object
	outer       *Environment
	Filename    string
	RootPath    string // Handler root directory for @~/ path resolution
	LastToken   *lexer.Token
	letBindings map[string]bool // tracks which variables were declared with 'let'
	exports     map[string]bool // tracks which variables were explicitly exported
	protected   map[string]bool // tracks which variables cannot be reassigned
	Security    *SecurityPolicy // File system security policy
	Logger      Logger          // Logger for log()/logLine() output
	importStack map[string]bool // tracks modules being imported (for circular dep detection)
	DevLog      DevLogWriter    // Dev log writer (nil in production mode)
	BasilCtx    Object          // Basil server context (request, db, auth, etc.)
}

// NewEnvironment creates a new environment
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	l := make(map[string]bool)
	x := make(map[string]bool)
	p := make(map[string]bool)
	i := make(map[string]bool)
	return &Environment{store: s, outer: nil, letBindings: l, exports: x, protected: p, importStack: i, Logger: DefaultLogger}
}

// NewEnclosedEnvironment creates a new environment with outer reference
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	// Preserve filename, token, logger, devlog, basilctx, and root path from outer environment
	if outer != nil {
		env.Filename = outer.Filename
		env.RootPath = outer.RootPath
		env.LastToken = outer.LastToken
		env.Logger = outer.Logger
		env.DevLog = outer.DevLog
		env.BasilCtx = outer.BasilCtx
	}
	return env
}

// Get retrieves a value from the environment
func (e *Environment) Get(name string) (Object, bool) {
	value, ok := e.store[name]
	if !ok && e.outer != nil {
		value, ok = e.outer.Get(name)
	}
	return value, ok
}

// Set stores a value in the environment
func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// SetLet stores a value in the environment and marks it as a let binding
func (e *Environment) SetLet(name string, val Object) Object {
	e.store[name] = val
	e.letBindings[name] = true
	return val
}

// SetExport stores a value in the environment and marks it as explicitly exported
func (e *Environment) SetExport(name string, val Object) Object {
	e.store[name] = val
	e.exports[name] = true
	return val
}

// SetLetExport stores a value in the environment, marks it as a let binding AND exported
func (e *Environment) SetLetExport(name string, val Object) Object {
	e.store[name] = val
	e.letBindings[name] = true
	e.exports[name] = true
	return val
}

// IsLetBinding checks if a variable was declared with let
func (e *Environment) IsLetBinding(name string) bool {
	// Check current environment
	if e.letBindings[name] {
		return true
	}
	// Don't check outer environments - each module has its own scope
	return false
}

// IsExported checks if a variable is exported (either via explicit export or via let - backward compat)
func (e *Environment) IsExported(name string) bool {
	// Check for explicit export first
	if e.exports[name] {
		return true
	}
	// Backward compatibility: let bindings are also exported
	if e.letBindings[name] {
		return true
	}
	return false
}

// SetProtected stores a value and marks it as protected (cannot be reassigned)
func (e *Environment) SetProtected(name string, val Object) Object {
	e.store[name] = val
	e.protected[name] = true
	return val
}

// IsProtected checks if a variable is protected from reassignment
func (e *Environment) IsProtected(name string) bool {
	if e.protected[name] {
		return true
	}
	if e.outer != nil {
		return e.outer.IsProtected(name)
	}
	return false
}

// AllIdentifiers returns all identifiers available in this environment and its outer scopes.
// This is used for fuzzy matching in error messages.
func (e *Environment) AllIdentifiers() []string {
	seen := make(map[string]bool)
	var result []string

	// Walk through all scopes
	env := e
	for env != nil {
		for name := range env.store {
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
		env = env.outer
	}

	// Add builtins
	for name := range getBuiltins() {
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}

	return result
}

// Update stores a value in the environment where it's defined (current or outer)
// If the variable doesn't exist anywhere, it creates it in the current scope
// Returns an error if trying to reassign a protected variable
func (e *Environment) Update(name string, val Object) Object {
	// Check if variable is protected
	if e.IsProtected(name) {
		return &Error{Message: fmt.Sprintf("cannot reassign protected variable '%s'", name)}
	}

	// Check if variable exists in current scope
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return val
	}

	// Check if it exists in outer scope
	if e.outer != nil {
		if _, ok := e.outer.Get(name); ok {
			return e.outer.Update(name, val)
		}
	}

	// Variable doesn't exist anywhere, create it in current scope
	e.store[name] = val
	return val
}

// NewDictionaryFromObjects creates a Dictionary from a map of Objects
// This is useful for programmatically creating dictionaries without AST expressions
func NewDictionaryFromObjects(pairs map[string]Object) *Dictionary {
	dict := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   NewEnvironment(),
	}
	for k, v := range pairs {
		dict.Pairs[k] = &ast.ObjectLiteralExpression{Obj: v}
	}
	return dict
}

// checkPathAccess validates file system access based on security policy
func (e *Environment) checkPathAccess(path string, operation string) error {
	if e.Security == nil {
		// No policy = default behavior
		// Read: allowed
		// Write: denied
		// Execute: denied
		if operation == "write" {
			return fmt.Errorf("write access denied (use --allow-write or -w)")
		}
		if operation == "execute" {
			return fmt.Errorf("execute access denied (use --allow-execute or -x)")
		}
		return nil
	}

	// Convert to absolute path and resolve symlinks for consistent comparison
	// This handles macOS /var -> /private/var symlinks and similar
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %s", err)
	}
	absPath = filepath.Clean(absPath)

	// Try to resolve symlinks. If the file doesn't exist (e.g., for write operations),
	// resolve the parent directory and append the filename.
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	} else {
		// File doesn't exist - try resolving parent directory
		dir := filepath.Dir(absPath)
		base := filepath.Base(absPath)
		if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
			absPath = filepath.Join(resolvedDir, base)
		}
	}

	switch operation {
	case "read":
		if e.Security.NoRead {
			return fmt.Errorf("file read access denied: %s", path)
		}
		// Check blacklist
		if isPathRestricted(absPath, e.Security.RestrictRead) {
			return fmt.Errorf("file read restricted: %s", path)
		}

	case "write":
		if e.Security.AllowWriteAll {
			return nil // Unrestricted
		}
		if !isPathAllowed(absPath, e.Security.AllowWrite) {
			return fmt.Errorf("file write not allowed: %s (use --allow-write or -w)", path)
		}

	case "execute":
		if e.Security.AllowExecuteAll {
			return nil // Unrestricted
		}
		if !isPathAllowed(absPath, e.Security.AllowExecute) {
			// Include helpful debug info in error message
			if len(e.Security.AllowExecute) > 0 {
				allowedStr := strings.Join(e.Security.AllowExecute, ", ")
				return fmt.Errorf("script execution not allowed: %s (resolved to: %s, allowed: %s)", path, absPath, allowedStr)
			}
			return fmt.Errorf("script execution not allowed: %s (no directories allowed)", path)
		}
	}

	return nil
}

// isPathAllowed checks if a path is within any allowed directory
func isPathAllowed(path string, allowList []string) bool {
	// Empty allow list means nothing is allowed
	if len(allowList) == 0 {
		return false
	}

	// Check if path is within any allowed directory
	for _, allowed := range allowList {
		// Resolve symlinks in allowed path for consistent comparison
		resolvedAllowed := allowed
		if resolved, err := filepath.EvalSymlinks(allowed); err == nil {
			resolvedAllowed = resolved
		}
		if path == resolvedAllowed || strings.HasPrefix(path, resolvedAllowed+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// isPathRestricted checks if a path is within any restricted directory
func isPathRestricted(path string, restrictList []string) bool {
	// Empty restrict list = no restrictions
	if len(restrictList) == 0 {
		return false
	}

	// Check if path is within any restricted directory
	for _, restricted := range restrictList {
		// Resolve symlinks in restricted path for consistent comparison
		resolvedRestricted := restricted
		if resolved, err := filepath.EvalSymlinks(restricted); err == nil {
			resolvedRestricted = resolved
		}
		if path == resolvedRestricted || strings.HasPrefix(path, resolvedRestricted+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// Global constants
var (
	NULL  = &Null{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

// ModuleCache caches imported modules
type ModuleCache struct {
	mu      sync.RWMutex
	modules map[string]*Dictionary // absolute path -> module dictionary
}

// Global module cache
var moduleCache = &ModuleCache{
	modules: make(map[string]*Dictionary),
}

// ClearModuleCache clears all cached modules
// This should be called before each request in Basil to ensure modules
// see fresh basil.* values (request data, auth, etc.)
func ClearModuleCache() {
	moduleCache.mu.Lock()
	defer moduleCache.mu.Unlock()
	moduleCache.modules = make(map[string]*Dictionary)
}

// naturalCompare compares two objects using natural sort order
// Returns true if a < b in natural sort order
func naturalCompare(a, b Object) bool {
	// Type-based ordering: numbers < strings
	aType := getTypeOrder(a)
	bType := getTypeOrder(b)

	if aType != bType {
		return aType < bType
	}

	// Both are numbers
	if aType == 0 {
		return compareNumbers(a, b)
	}

	// Both are strings - use natural string comparison
	if aType == 1 {
		aStr := a.(*String).Value
		bStr := b.(*String).Value
		return naturalStringCompare(aStr, bStr)
	}

	// Other types (shouldn't happen with current implementation)
	return false
}

// getTypeOrder returns a sort order for types
// 0 = numbers (Integer, Float)
// 1 = strings
// 2 = other
func getTypeOrder(obj Object) int {
	switch obj.Type() {
	case INTEGER_OBJ, FLOAT_OBJ:
		return 0
	case STRING_OBJ:
		return 1
	default:
		return 2
	}
}

// compareNumbers compares two numeric objects
func compareNumbers(a, b Object) bool {
	aVal := getNumericValue(a)
	bVal := getNumericValue(b)
	return aVal < bVal
}

// getNumericValue extracts numeric value as float64
func getNumericValue(obj Object) float64 {
	switch obj := obj.(type) {
	case *Integer:
		return float64(obj.Value)
	case *Float:
		return obj.Value
	default:
		return 0
	}
}

// naturalStringCompare compares strings using natural sort order
// It treats consecutive digits as numbers and compares them numerically
func naturalStringCompare(a, b string) bool {
	aRunes := []rune(a)
	bRunes := []rune(b)

	i, j := 0, 0

	for i < len(aRunes) && j < len(bRunes) {
		aChar := aRunes[i]
		bChar := bRunes[j]

		// Both are digits - compare numerically
		if unicode.IsDigit(aChar) && unicode.IsDigit(bChar) {
			// Extract the full number from both strings
			aNum, aEnd := extractNumber(aRunes, i)
			bNum, bEnd := extractNumber(bRunes, j)

			if aNum != bNum {
				return aNum < bNum
			}

			i = aEnd
			j = bEnd
			continue
		}

		// Character comparison
		if aChar != bChar {
			return aChar < bChar
		}

		i++
		j++
	}

	// If we've exhausted one string, the shorter one comes first
	return len(aRunes) < len(bRunes)
}

// extractNumber extracts a number from a rune slice starting at the given position
// Returns the number and the position after the last digit
func extractNumber(runes []rune, start int) (int64, int) {
	end := start
	for end < len(runes) && unicode.IsDigit(runes[end]) {
		end++
	}

	numStr := string(runes[start:end])
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, end
	}

	return num, end
}

// objectsEqual compares two objects for equality
func objectsEqual(a, b Object) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch a := a.(type) {
	case *Integer:
		return a.Value == b.(*Integer).Value
	case *Float:
		return a.Value == b.(*Float).Value
	case *String:
		return a.Value == b.(*String).Value
	case *Boolean:
		return a.Value == b.(*Boolean).Value
	case *Null:
		return true
	default:
		return false
	}
}

// timeToDictWithKind converts a Go time.Time to a Parsley Dictionary with a specified kind
func timeToDictWithKind(t time.Time, kind string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Mark this as a datetime dictionary for special operator handling
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "datetime"},
		Value: "datetime",
	}

	// Store the kind (datetime, date, or time)
	pairs["kind"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: kind},
		Value: kind,
	}

	// Create integer literals for numeric values with proper tokens
	pairs["year"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Year())},
		Value: int64(t.Year()),
	}
	pairs["month"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Month())},
		Value: int64(t.Month()),
	}
	pairs["day"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Day())},
		Value: int64(t.Day()),
	}
	pairs["hour"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Hour())},
		Value: int64(t.Hour()),
	}
	pairs["minute"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Minute())},
		Value: int64(t.Minute()),
	}
	pairs["second"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Second())},
		Value: int64(t.Second()),
	}
	pairs["unix"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Unix())},
		Value: t.Unix(),
	}

	// Create string literals for string values with proper tokens
	weekday := t.Weekday().String()
	pairs["weekday"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: weekday},
		Value: weekday,
	}
	iso := t.Format(time.RFC3339)
	pairs["iso"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: iso},
		Value: iso,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// timeToDict converts a Go time.Time to a Parsley Dictionary (defaults to kind: "datetime")
func timeToDict(t time.Time, env *Environment) *Dictionary {
	return timeToDictWithKind(t, "datetime", env)
}

// dictToTime converts a Parsley Dictionary to a Go time.Time
func dictToTime(dict *Dictionary, env *Environment) (time.Time, error) {
	// Evaluate the year field
	yearExpr, ok := dict.Pairs["year"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'year' field")
	}
	yearObj := Eval(yearExpr, env)
	yearInt, ok := yearObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'year' must be an integer")
	}

	// Evaluate the month field
	monthExpr, ok := dict.Pairs["month"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'month' field")
	}
	monthObj := Eval(monthExpr, env)
	monthInt, ok := monthObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'month' must be an integer")
	}

	// Evaluate the day field
	dayExpr, ok := dict.Pairs["day"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'day' field")
	}
	dayObj := Eval(dayExpr, env)
	dayInt, ok := dayObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'day' must be an integer")
	}

	// Hour, minute, second are optional (default to 0)
	var hour, minute, second int64

	if hExpr, ok := dict.Pairs["hour"]; ok {
		hObj := Eval(hExpr, env)
		if hInt, ok := hObj.(*Integer); ok {
			hour = hInt.Value
		}
	}

	if mExpr, ok := dict.Pairs["minute"]; ok {
		mObj := Eval(mExpr, env)
		if mInt, ok := mObj.(*Integer); ok {
			minute = mInt.Value
		}
	}

	if sExpr, ok := dict.Pairs["second"]; ok {
		sObj := Eval(sExpr, env)
		if sInt, ok := sObj.(*Integer); ok {
			second = sInt.Value
		}
	}

	return time.Date(
		int(yearInt.Value),
		time.Month(monthInt.Value),
		int(dayInt.Value),
		int(hour),
		int(minute),
		int(second),
		0,
		time.UTC,
	), nil
}

// isDatetimeDict checks if a dictionary is a datetime by looking for __type field
func isDatetimeDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "datetime"
		}
	}
	return false
}

// isDurationDict checks if a dictionary is a duration by looking for __type field
func isDurationDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "duration"
		}
	}
	return false
}

// getDurationComponents extracts months and seconds from a duration dictionary
func getDurationComponents(dict *Dictionary, env *Environment) (int64, int64, error) {
	monthsExpr, ok := dict.Pairs["months"]
	if !ok {
		return 0, 0, fmt.Errorf("duration dictionary missing months field")
	}
	monthsObj := Eval(monthsExpr, env)
	monthsInt, ok := monthsObj.(*Integer)
	if !ok {
		return 0, 0, fmt.Errorf("months must be an integer")
	}

	secondsExpr, ok := dict.Pairs["seconds"]
	if !ok {
		return 0, 0, fmt.Errorf("duration dictionary missing seconds field")
	}
	secondsObj := Eval(secondsExpr, env)
	secondsInt, ok := secondsObj.(*Integer)
	if !ok {
		return 0, 0, fmt.Errorf("seconds must be an integer")
	}

	return monthsInt.Value, secondsInt.Value, nil
}

// getDatetimeKind extracts the kind from a datetime dictionary (defaults to "datetime")
func getDatetimeKind(dict *Dictionary, env *Environment) string {
	if kindExpr, ok := dict.Pairs["kind"]; ok {
		kindObj := Eval(kindExpr, env)
		if kindStr, ok := kindObj.(*String); ok {
			return kindStr.Value
		}
	}
	return "datetime"
}

// getDatetimeUnix extracts the unix timestamp from a datetime dictionary
func getDatetimeUnix(dict *Dictionary, env *Environment) (int64, error) {
	unixExpr, ok := dict.Pairs["unix"]
	if !ok {
		return 0, fmt.Errorf("datetime dictionary missing unix field")
	}
	unixObj := Eval(unixExpr, env)
	unixInt, ok := unixObj.(*Integer)
	if !ok {
		return 0, fmt.Errorf("unix field is not an integer")
	}
	return unixInt.Value, nil
}

// getMondayLocale maps a BCP 47 locale string to monday.Locale
func getMondayLocale(locale string) monday.Locale {
	// Normalize locale string
	locale = strings.ToLower(strings.ReplaceAll(locale, "-", "_"))

	localeMap := map[string]monday.Locale{
		"en":    monday.LocaleEnUS,
		"en_us": monday.LocaleEnUS,
		"en_gb": monday.LocaleEnGB,
		"en_au": monday.LocaleEnUS, // Fallback to US
		"de":    monday.LocaleDeDE,
		"de_de": monday.LocaleDeDE,
		"de_at": monday.LocaleDeDE,
		"de_ch": monday.LocaleDeDE,
		"fr":    monday.LocaleFrFR,
		"fr_fr": monday.LocaleFrFR,
		"fr_ca": monday.LocaleFrCA,
		"fr_be": monday.LocaleFrFR,
		"es":    monday.LocaleEsES,
		"es_es": monday.LocaleEsES,
		"es_mx": monday.LocaleEsES,
		"it":    monday.LocaleItIT,
		"it_it": monday.LocaleItIT,
		"pt":    monday.LocalePtPT,
		"pt_pt": monday.LocalePtPT,
		"pt_br": monday.LocalePtBR,
		"nl":    monday.LocaleNlNL,
		"nl_nl": monday.LocaleNlNL,
		"nl_be": monday.LocaleNlBE,
		"ru":    monday.LocaleRuRU,
		"ru_ru": monday.LocaleRuRU,
		"pl":    monday.LocalePlPL,
		"pl_pl": monday.LocalePlPL,
		"cs":    monday.LocaleCsCZ,
		"cs_cz": monday.LocaleCsCZ,
		"da":    monday.LocaleDaDK,
		"da_dk": monday.LocaleDaDK,
		"fi":    monday.LocaleFiFI,
		"fi_fi": monday.LocaleFiFI,
		"sv":    monday.LocaleSvSE,
		"sv_se": monday.LocaleSvSE,
		"nb":    monday.LocaleNbNO,
		"nb_no": monday.LocaleNbNO,
		"nn":    monday.LocaleNnNO,
		"nn_no": monday.LocaleNnNO,
		"ja":    monday.LocaleJaJP,
		"ja_jp": monday.LocaleJaJP,
		"zh":    monday.LocaleZhCN,
		"zh_cn": monday.LocaleZhCN,
		"zh_tw": monday.LocaleZhTW,
		"ko":    monday.LocaleKoKR,
		"ko_kr": monday.LocaleKoKR,
		"tr":    monday.LocaleTrTR,
		"tr_tr": monday.LocaleTrTR,
		"uk":    monday.LocaleUkUA,
		"uk_ua": monday.LocaleUkUA,
		"el":    monday.LocaleElGR,
		"el_gr": monday.LocaleElGR,
		"ro":    monday.LocaleRoRO,
		"ro_ro": monday.LocaleRoRO,
		"hu":    monday.LocaleHuHU,
		"hu_hu": monday.LocaleHuHU,
		"bg":    monday.LocaleBgBG,
		"bg_bg": monday.LocaleBgBG,
		"id":    monday.LocaleIdID,
		"id_id": monday.LocaleIdID,
		"th":    monday.LocaleThTH,
		"th_th": monday.LocaleThTH,
	}

	if loc, ok := localeMap[locale]; ok {
		return loc
	}

	// Try just the language part
	parts := strings.Split(locale, "_")
	if len(parts) > 1 {
		if loc, ok := localeMap[parts[0]]; ok {
			return loc
		}
	}

	return monday.LocaleEnUS // Default fallback
}

// getDateFormatForStyle returns the Go time format string for a given style and locale
func getDateFormatForStyle(style string, locale monday.Locale) string {
	switch style {
	case "short":
		// Numeric format - varies by locale
		switch locale {
		case monday.LocaleEnUS:
			return "1/2/06"
		case monday.LocaleEnGB:
			return "02/01/06"
		case monday.LocaleDeDE:
			return "02.01.06"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "02/01/06"
		case monday.LocaleJaJP:
			return "06/01/02"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "06/1/2"
		case monday.LocaleKoKR:
			return "06. 1. 2."
		default:
			return "02/01/06"
		}
	case "medium":
		// Abbreviated month - locale-aware order
		switch locale {
		case monday.LocaleEnUS:
			return "Jan 2, 2006"
		case monday.LocaleEnGB:
			return "2 Jan 2006"
		case monday.LocaleDeDE:
			return "2. Jan. 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "2 Jan 2006"
		case monday.LocaleEsES:
			return "2 Jan 2006"
		case monday.LocaleItIT:
			return "2 Jan 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日"
		case monday.LocaleKoKR:
			return "2006년 1월 2일"
		case monday.LocalePtBR:
			return "2 Jan 2006"
		case monday.LocaleRuRU:
			return "2 Jan 2006"
		case monday.LocaleNlNL, monday.LocaleNlBE:
			return "2 Jan 2006"
		default:
			return "2 Jan 2006"
		}
	case "long":
		// Full month name - locale-aware order
		switch locale {
		case monday.LocaleEnUS:
			return "January 2, 2006"
		case monday.LocaleEnGB:
			return "2 January 2006"
		case monday.LocaleDeDE:
			return "2. January 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "2 January 2006"
		case monday.LocaleEsES:
			return "2 de January de 2006"
		case monday.LocaleItIT:
			return "2 January 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日"
		case monday.LocaleKoKR:
			return "2006년 1월 2일"
		case monday.LocaleRuRU:
			return "2 January 2006"
		default:
			return "2 January 2006"
		}
	case "full":
		// With weekday - locale-aware
		switch locale {
		case monday.LocaleEnUS:
			return "Monday, January 2, 2006"
		case monday.LocaleEnGB:
			return "Monday, 2 January 2006"
		case monday.LocaleDeDE:
			return "Monday, 2. January 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "Monday 2 January 2006"
		case monday.LocaleEsES:
			return "Monday, 2 de January de 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日 Monday"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日 Monday"
		case monday.LocaleKoKR:
			return "2006년 1월 2일 Monday"
		default:
			return "Monday, 2 January 2006"
		}
	default:
		return "January 2, 2006" // Default to long English
	}
}

// datetimeDictToString converts a datetime dictionary to a human-friendly ISO 8601 string
// Uses the "kind" field to determine output format: "datetime", "date", or "time"
func datetimeDictToString(dict *Dictionary) string {
	// Check for kind field to determine output format
	kind := "datetime" // default
	if kindExpr, ok := dict.Pairs["kind"]; ok {
		if kindLit, ok := kindExpr.(*ast.StringLiteral); ok {
			kind = kindLit.Value
		}
	}

	// Extract time components
	var hour, minute, second int64
	if hExpr, ok := dict.Pairs["hour"]; ok {
		if hLit, ok := hExpr.(*ast.IntegerLiteral); ok {
			hour = hLit.Value
		}
	}
	if minExpr, ok := dict.Pairs["minute"]; ok {
		if minLit, ok := minExpr.(*ast.IntegerLiteral); ok {
			minute = minLit.Value
		}
	}
	if sExpr, ok := dict.Pairs["second"]; ok {
		if sLit, ok := sExpr.(*ast.IntegerLiteral); ok {
			second = sLit.Value
		}
	}

	// Extract date components
	var year, month, day int64
	if yearExpr, ok := dict.Pairs["year"]; ok {
		if yLit, ok := yearExpr.(*ast.IntegerLiteral); ok {
			year = yLit.Value
		}
	}
	if mExpr, ok := dict.Pairs["month"]; ok {
		if mLit, ok := mExpr.(*ast.IntegerLiteral); ok {
			month = mLit.Value
		}
	}
	if dExpr, ok := dict.Pairs["day"]; ok {
		if dLit, ok := dExpr.(*ast.IntegerLiteral); ok {
			day = dLit.Value
		}
	}

	// Format based on kind
	switch kind {
	case "time":
		// Time only without seconds: HH:MM
		return fmt.Sprintf("%02d:%02d", hour, minute)

	case "time_seconds":
		// Time with seconds: HH:MM:SS
		return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)

	case "date":
		// Date only: YYYY-MM-DD
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)

	default:
		// Full datetime: YYYY-MM-DDTHH:MM:SSZ
		// If time is all zeros, still include it for datetime kind
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", year, month, day, hour, minute, second)
	}
}

// durationDictToString converts a duration dictionary to a human-readable string
func durationDictToString(dict *Dictionary) string {
	var months, seconds int64

	// Get months
	if monthsExpr, ok := dict.Pairs["months"]; ok {
		monthsObj := Eval(monthsExpr, dict.Env)
		if i, ok := monthsObj.(*Integer); ok {
			months = i.Value
		}
	}

	// Get seconds
	if secondsExpr, ok := dict.Pairs["seconds"]; ok {
		secondsObj := Eval(secondsExpr, dict.Env)
		if i, ok := secondsObj.(*Integer); ok {
			seconds = i.Value
		}
	}

	// Handle zero duration
	if months == 0 && seconds == 0 {
		return "0 seconds"
	}

	var parts []string
	isNegative := months < 0 || seconds < 0

	// Handle negative values
	if months < 0 {
		months = -months
	}
	if seconds < 0 {
		seconds = -seconds
	}

	// Convert months to years and months
	years := months / 12
	months = months % 12

	if years > 0 {
		if years == 1 {
			parts = append(parts, "1 year")
		} else {
			parts = append(parts, fmt.Sprintf("%d years", years))
		}
	}
	if months > 0 {
		if months == 1 {
			parts = append(parts, "1 month")
		} else {
			parts = append(parts, fmt.Sprintf("%d months", months))
		}
	}

	// Convert seconds to days, hours, minutes, seconds
	days := seconds / 86400
	seconds = seconds % 86400
	hours := seconds / 3600
	seconds = seconds % 3600
	minutes := seconds / 60
	seconds = seconds % 60

	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
	}
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}
	if seconds > 0 {
		if seconds == 1 {
			parts = append(parts, "1 second")
		} else {
			parts = append(parts, fmt.Sprintf("%d seconds", seconds))
		}
	}

	result := strings.Join(parts, " ")
	if isNegative {
		return "-" + result
	}
	return result
}

// regexDictToString converts a regex dictionary to its literal form /pattern/flags
func regexDictToString(dict *Dictionary) string {
	var pattern, flags string

	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		patternObj := Eval(patternExpr, dict.Env)
		if str, ok := patternObj.(*String); ok {
			pattern = str.Value
		}
	}

	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		flagsObj := Eval(flagsExpr, dict.Env)
		if str, ok := flagsObj.(*String); ok {
			flags = str.Value
		}
	}

	return "/" + pattern + "/" + flags
}

// fileDictToString converts a file dictionary to its path string
func fileDictToString(dict *Dictionary) string {
	// Extract path components from the file dict
	var components []string
	var isAbsolute bool

	if compExpr, ok := dict.Pairs["_pathComponents"]; ok {
		compObj := Eval(compExpr, dict.Env)
		if arr, ok := compObj.(*Array); ok {
			for _, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					components = append(components, str.Value)
				}
			}
		}
	}

	if absExpr, ok := dict.Pairs["_pathAbsolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string - use same logic as pathDictToString
	if len(components) == 0 {
		if isAbsolute {
			return "/"
		}
		return "."
	}

	result := strings.Join(components, "/")
	if isAbsolute {
		return "/" + result
	}
	return result
}

// dirDictToString converts a directory dictionary to its path string (with trailing slash)
func dirDictToString(dict *Dictionary) string {
	// Extract path components from the dir dict
	var components []string
	var isAbsolute bool

	if compExpr, ok := dict.Pairs["_pathComponents"]; ok {
		compObj := Eval(compExpr, dict.Env)
		if arr, ok := compObj.(*Array); ok {
			for _, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					components = append(components, str.Value)
				}
			}
		}
	}

	if absExpr, ok := dict.Pairs["_pathAbsolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string - use same logic as pathDictToString
	var pathStr string
	if len(components) == 0 {
		if isAbsolute {
			pathStr = "/"
		} else {
			pathStr = "./"
		}
	} else {
		result := strings.Join(components, "/")
		if isAbsolute {
			pathStr = "/" + result
		} else {
			pathStr = result
		}
	}

	// Add trailing slash for directories
	if !strings.HasSuffix(pathStr, "/") {
		pathStr += "/"
	}

	return pathStr
}

// requestDictToString converts a request dictionary to METHOD URL format
func requestDictToString(dict *Dictionary) string {
	var method, urlStr string

	// Get method (default to GET)
	method = "GET"
	if methodExpr, ok := dict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, dict.Env)
		if str, ok := methodObj.(*String); ok {
			method = str.Value
		}
	}

	// Reconstruct URL from _url_* fields
	var result strings.Builder

	// Scheme
	if schemeExpr, ok := dict.Pairs["_url_scheme"]; ok {
		schemeObj := Eval(schemeExpr, dict.Env)
		if str, ok := schemeObj.(*String); ok {
			result.WriteString(str.Value)
			result.WriteString("://")
		}
	}

	// Host
	if hostExpr, ok := dict.Pairs["_url_host"]; ok {
		hostObj := Eval(hostExpr, dict.Env)
		if str, ok := hostObj.(*String); ok {
			result.WriteString(str.Value)
		}
	}

	// Port
	if portExpr, ok := dict.Pairs["_url_port"]; ok {
		portObj := Eval(portExpr, dict.Env)
		if i, ok := portObj.(*Integer); ok && i.Value != 0 {
			result.WriteString(":")
			result.WriteString(strconv.FormatInt(i.Value, 10))
		}
	}

	// Path
	if pathExpr, ok := dict.Pairs["_url_path"]; ok {
		pathObj := Eval(pathExpr, dict.Env)
		if arr, ok := pathObj.(*Array); ok && len(arr.Elements) > 0 {
			startIdx := 0
			if str, ok := arr.Elements[0].(*String); ok && str.Value == "" {
				result.WriteString("/")
				startIdx = 1
			} else if len(arr.Elements) > 0 {
				result.WriteString("/")
			}
			for i := startIdx; i < len(arr.Elements); i++ {
				if str, ok := arr.Elements[i].(*String); ok && str.Value != "" {
					if i > startIdx {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Query
	if queryExpr, ok := dict.Pairs["_url_query"]; ok {
		queryObj := Eval(queryExpr, dict.Env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, expr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				result.WriteString(key)
				result.WriteString("=")
				valObj := Eval(expr, dict.Env)
				if str, ok := valObj.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	urlStr = result.String()
	return method + " " + urlStr
}

// applyDelta applies time deltas to a time.Time
func applyDelta(t time.Time, delta *Dictionary, env *Environment) time.Time {
	// Apply date-based deltas first (years, months, days)
	if yearsExpr, ok := delta.Pairs["years"]; ok {
		yearsObj := Eval(yearsExpr, env)
		if yearsInt, ok := yearsObj.(*Integer); ok {
			t = t.AddDate(int(yearsInt.Value), 0, 0)
		}
	}

	if monthsExpr, ok := delta.Pairs["months"]; ok {
		monthsObj := Eval(monthsExpr, env)
		if monthsInt, ok := monthsObj.(*Integer); ok {
			t = t.AddDate(0, int(monthsInt.Value), 0)
		}
	}

	if daysExpr, ok := delta.Pairs["days"]; ok {
		daysObj := Eval(daysExpr, env)
		if daysInt, ok := daysObj.(*Integer); ok {
			t = t.AddDate(0, 0, int(daysInt.Value))
		}
	}

	// Apply time-based deltas (hours, minutes, seconds)
	if hoursExpr, ok := delta.Pairs["hours"]; ok {
		hoursObj := Eval(hoursExpr, env)
		if hoursInt, ok := hoursObj.(*Integer); ok {
			t = t.Add(time.Duration(hoursInt.Value) * time.Hour)
		}
	}

	if minutesExpr, ok := delta.Pairs["minutes"]; ok {
		minutesObj := Eval(minutesExpr, env)
		if minutesInt, ok := minutesObj.(*Integer); ok {
			t = t.Add(time.Duration(minutesInt.Value) * time.Minute)
		}
	}

	if secondsExpr, ok := delta.Pairs["seconds"]; ok {
		secondsObj := Eval(secondsExpr, env)
		if secondsInt, ok := secondsObj.(*Integer); ok {
			t = t.Add(time.Duration(secondsInt.Value) * time.Second)
		}
	}

	return t
}

// evalRegexLiteral evaluates a regex literal and returns a Dictionary with __type: "regex"
func evalRegexLiteral(node *ast.RegexLiteral, env *Environment) Object {
	pairs := make(map[string]ast.Expression)

	// Mark this as a regex dictionary
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "regex"},
		Value: "regex",
	}
	pairs["pattern"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: node.Pattern},
		Value: node.Pattern,
	}
	pairs["flags"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: node.Flags},
		Value: node.Flags,
	}

	// Try to compile the regex to validate it
	_, err := compileRegex(node.Pattern, node.Flags)
	if err != nil {
		return newFormatError("FMT-0002", err)
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// evalDatetimeLiteral evaluates a datetime literal like @2024-12-25T14:30:00Z or @12:30
func evalDatetimeLiteral(node *ast.DatetimeLiteral, env *Environment) Object {
	// Parse the ISO-8601 datetime string
	var t time.Time
	var err error
	kind := node.Kind
	if kind == "" {
		kind = "datetime" // default for backwards compatibility
	}

	if kind == "time" || kind == "time_seconds" {
		// Time-only literal: HH:MM or HH:MM:SS
		// Use current UTC date as the base
		now := time.Now().UTC()

		// Try parsing with seconds first
		t, err = time.Parse("15:04:05", node.Value)
		if err != nil {
			// Try without seconds
			t, err = time.Parse("15:04", node.Value)
			if err != nil {
				return newError("invalid time literal: %s", node.Value)
			}
		}

		// Combine with current UTC date
		t = time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	} else {
		// Date or datetime literal
		// Try parsing as RFC3339 first (most complete format with timezone)
		t, err = time.Parse(time.RFC3339, node.Value)
		if err != nil {
			// Try date-only format (2024-12-25) - interpret as UTC
			t, err = time.ParseInLocation("2006-01-02", node.Value, time.UTC)
			if err != nil {
				// Try datetime without timezone (2024-12-25T14:30:05) - interpret as UTC
				t, err = time.ParseInLocation("2006-01-02T15:04:05", node.Value, time.UTC)
				if err != nil {
					return newFormatError("FMT-0004", fmt.Errorf("cannot parse %q", node.Value))
				}
			}
		}
	}

	// Convert to dictionary using the new function with kind
	return timeToDictWithKind(t, kind, env)
}

// evalDurationLiteral parses a duration literal like @2h30m, @7d, @1y6mo
func evalDurationLiteral(node *ast.DurationLiteral, env *Environment) Object {
	// Parse the duration string into months and seconds
	months, seconds, err := parseDurationString(node.Value)
	if err != nil {
		return newFormatError("FMT-0009", err)
	}

	return durationToDict(months, seconds, env)
}

// evalPathLiteral parses a path literal like @/usr/local/bin or @./config.json
// Also handles special stdio paths: @-, @stdin, @stdout, @stderr
func evalPathLiteral(node *ast.PathLiteral, env *Environment) Object {
	// Check for stdio special paths
	switch node.Value {
	case "-":
		// @- is context-dependent: stdin for reads, stdout for writes
		return stdioToDict("stdio", env)
	case "stdin":
		return stdioToDict("stdin", env)
	case "stdout":
		return stdioToDict("stdout", env)
	case "stderr":
		return stdioToDict("stderr", env)
	}

	// Parse the path string into components
	components, isAbsolute := parsePathString(node.Value)

	// Create path dictionary
	return pathToDict(components, isAbsolute, env)
}

// evalUrlLiteral parses a URL literal like @https://example.com/api
func evalUrlLiteral(node *ast.UrlLiteral, env *Environment) Object {
	// Parse the URL string
	urlDict, err := parseUrlString(node.Value, env)
	if err != nil {
		return newFormatError("FMT-0003", err)
	}

	return urlDict
}

// evalPathTemplateLiteral evaluates an interpolated path template like @(./path/{name}/file)
func evalPathTemplateLiteral(node *ast.PathTemplateLiteral, env *Environment) Object {
	// First, interpolate the template
	interpolated := interpolatePathUrlTemplate(node.Value, env)
	if isError(interpolated) {
		return interpolated
	}

	// Get the interpolated string
	pathStr := interpolated.(*String).Value

	// Parse the path string into components
	components, isAbsolute := parsePathString(pathStr)

	// Create path dictionary
	return pathToDict(components, isAbsolute, env)
}

// evalUrlTemplateLiteral evaluates an interpolated URL template like @(https://api.com/{version}/users)
func evalUrlTemplateLiteral(node *ast.UrlTemplateLiteral, env *Environment) Object {
	// First, interpolate the template
	interpolated := interpolatePathUrlTemplate(node.Value, env)
	if isError(interpolated) {
		return interpolated
	}

	// Get the interpolated string
	urlStr := interpolated.(*String).Value

	// Parse the URL string
	urlDict, err := parseUrlString(urlStr, env)
	if err != nil {
		return newFormatError("FMT-0003", err)
	}

	return urlDict
}

// evalDatetimeTemplateLiteral evaluates an interpolated datetime template like @(2024-{month}-{day})
func evalDatetimeTemplateLiteral(node *ast.DatetimeTemplateLiteral, env *Environment) Object {
	// First, interpolate the template
	interpolated := interpolatePathUrlTemplate(node.Value, env)
	if isError(interpolated) {
		return interpolated
	}

	// Get the interpolated string
	datetimeStr := interpolated.(*String).Value

	// Determine the kind and parse the datetime
	var t time.Time
	var err error
	var kind string

	// Check if it's a time-only pattern (starts with digit and contains :)
	// Time patterns: HH:MM or HH:MM:SS
	if len(datetimeStr) >= 4 && datetimeStr[2] == ':' {
		// Looks like a time pattern (e.g., "12:30" or "12:30:45")
		kind = "time"
		now := time.Now().UTC()

		// Try parsing with seconds first
		t, err = time.Parse("15:04:05", datetimeStr)
		if err != nil {
			// Try without seconds
			t, err = time.Parse("15:04", datetimeStr)
			if err != nil {
				return newError("invalid time in datetime template: %s", datetimeStr)
			}
		}

		// Combine with current UTC date
		t = time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	} else {
		// Check for date-only (YYYY-MM-DD) vs full datetime (YYYY-MM-DDTHH:MM:SS)
		if len(datetimeStr) == 10 && datetimeStr[4] == '-' && datetimeStr[7] == '-' {
			kind = "date"
		} else {
			kind = "datetime"
		}

		// Try parsing as RFC3339 first (most complete format with timezone)
		t, err = time.Parse(time.RFC3339, datetimeStr)
		if err != nil {
			// Try date-only format (2024-12-25) - interpret as UTC
			t, err = time.ParseInLocation("2006-01-02", datetimeStr, time.UTC)
			if err != nil {
				// Try datetime without timezone (2024-12-25T14:30:05) - interpret as UTC
				t, err = time.ParseInLocation("2006-01-02T15:04:05", datetimeStr, time.UTC)
				if err != nil {
					return newFormatError("FMT-0004", fmt.Errorf("cannot parse %q", datetimeStr))
				}
			}
		}
	}

	// Convert to dictionary using the function with kind
	return timeToDictWithKind(t, kind, env)
}

// interpolatePathUrlTemplate processes {expr} interpolations in path/URL templates
// This is similar to evalTemplateLiteral but returns a String object
func interpolatePathUrlTemplate(template string, env *Environment) Object {
	var result strings.Builder

	i := 0
	for i < len(template) {
		// Look for {
		if template[i] == '{' {
			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(template) && braceCount > 0 {
				if template[i] == '{' {
					braceCount++
				} else if template[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "path/URL template", nil)
			}

			// Extract and evaluate the expression
			exprStr := template[exprStart:i]
			i++ // skip closing }

			// Handle empty interpolation
			if strings.TrimSpace(exprStr) == "" {
				return newParseError("PARSE-0010", "path/URL template", nil)
			}

			// Parse and evaluate the expression
			l := lexer.New(exprStr)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				return newParseError("PARSE-0011", "template", fmt.Errorf("%s", p.Errors()[0]))
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					return evaluated
				}
			}

			// Convert result to string
			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
		} else {
			// Regular character
			result.WriteByte(template[i])
			i++
		}
	}

	return &String{Value: result.String()}
}

// parseDurationString parses a duration string like "2h30m" or "1y6mo" or "-1d" into months and seconds
// Returns (months, seconds, error)
// Negative durations (e.g., "-1d") return negative values
func parseDurationString(s string) (int64, int64, error) {
	var months int64
	var seconds int64
	negative := false

	i := 0

	// Check for leading minus sign (negative duration)
	if i < len(s) && s[i] == '-' {
		negative = true
		i++
	}

	for i < len(s) {
		// Read number
		if !isDigit(rune(s[i])) {
			return 0, 0, fmt.Errorf("expected digit at position %d", i)
		}

		numStart := i
		for i < len(s) && isDigit(rune(s[i])) {
			i++
		}

		num, err := strconv.ParseInt(s[numStart:i], 10, 64)
		if err != nil {
			return 0, 0, err
		}

		// Read unit
		if i >= len(s) {
			return 0, 0, fmt.Errorf("missing unit after number at position %d", i)
		}

		var unit string
		// Check for "mo" (months)
		if i+1 < len(s) && s[i:i+2] == "mo" {
			unit = "mo"
			i += 2
		} else {
			// Single letter unit
			unit = string(s[i])
			i++
		}

		// Convert to months or seconds
		switch unit {
		case "y": // years = 12 months
			months += num * 12
		case "mo": // months
			months += num
		case "w": // weeks = 7 days = 7 * 24 * 60 * 60 seconds
			seconds += num * 7 * 24 * 60 * 60
		case "d": // days = 24 * 60 * 60 seconds
			seconds += num * 24 * 60 * 60
		case "h": // hours = 60 * 60 seconds
			seconds += num * 60 * 60
		case "m": // minutes = 60 seconds
			seconds += num * 60
		case "s": // seconds
			seconds += num
		default:
			return 0, 0, fmt.Errorf("unknown unit: %s", unit)
		}
	}

	// Apply negative sign if present
	if negative {
		months = -months
		seconds = -seconds
	}

	return months, seconds, nil
}

// durationToDict converts months and seconds into a duration dictionary
func durationToDict(months, seconds int64, env *Environment) *Dictionary {
	dict := &Dictionary{Pairs: make(map[string]ast.Expression)}

	// Add __type field
	dict.Pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "duration"},
		Value: "duration",
	}

	// Add months field
	dict.Pairs["months"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", months)},
		Value: months,
	}

	// Add seconds field
	dict.Pairs["seconds"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", seconds)},
		Value: seconds,
	}

	// Add totalSeconds field (only present if no months)
	if months == 0 {
		dict.Pairs["totalSeconds"] = &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", seconds)},
			Value: seconds,
		}
	}

	return dict
}

// isDigit checks if a rune is a digit
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// isRegexDict checks if a dictionary is a regex by looking for __type field
func isRegexDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "regex"
		}
	}
	return false
}

// isPathDict checks if a dictionary is a path by looking for __type field
func isPathDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "path"
		}
	}
	return false
}

// isUrlDict checks if a dictionary is a URL by looking for __type field
func isUrlDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "url"
		}
	}
	return false
}

// isFileDict checks if a dictionary is a file handle by looking for __type field
func isFileDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "file"
		}
	}
	return false
}

// isTagDict checks if a dictionary is a tag by looking for __type field
func isTagDict(dict *Dictionary) bool {
	if typeExpr, ok := dict.Pairs["__type"]; ok {
		if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
			return strLit.Value == "tag"
		}
	}
	return false
}

// tagDictToString converts a tag dictionary back to an HTML string
func tagDictToString(dict *Dictionary) string {
	var result strings.Builder

	// Get tag name
	nameExpr, ok := dict.Pairs["name"]
	if !ok {
		return dict.Inspect() // Fallback if not a proper tag dict
	}
	nameObj := Eval(nameExpr, dict.Env)
	nameStr, ok := nameObj.(*String)
	if !ok {
		return dict.Inspect()
	}

	// Get contents
	contentsExpr, hasContents := dict.Pairs["contents"]
	var contentsObj Object
	if hasContents {
		contentsObj = Eval(contentsExpr, dict.Env)
	}

	// Get attributes
	attrsExpr, hasAttrs := dict.Pairs["attrs"]
	var attrsDict *Dictionary
	if hasAttrs {
		attrsObj := Eval(attrsExpr, dict.Env)
		if d, ok := attrsObj.(*Dictionary); ok {
			attrsDict = d
		}
	}

	// Check if self-closing (no contents)
	isSelfClosing := contentsObj == nil || contentsObj == NULL

	// Build the opening tag
	result.WriteByte('<')
	result.WriteString(nameStr.Value)

	// Add attributes
	if attrsDict != nil && len(attrsDict.Pairs) > 0 {
		for key, expr := range attrsDict.Pairs {
			result.WriteByte(' ')
			result.WriteString(key)
			result.WriteString(`="`)
			val := Eval(expr, attrsDict.Env)
			result.WriteString(objectToPrintString(val))
			result.WriteByte('"')
		}
	}

	if isSelfClosing {
		result.WriteString(" />")
	} else {
		result.WriteByte('>')

		// Add contents
		switch c := contentsObj.(type) {
		case *String:
			result.WriteString(c.Value)
		case *Array:
			for _, elem := range c.Elements {
				result.WriteString(objectToPrintString(elem))
			}
		default:
			result.WriteString(objectToPrintString(contentsObj))
		}

		// Closing tag
		result.WriteString("</")
		result.WriteString(nameStr.Value)
		result.WriteByte('>')
	}

	return result.String()
}

// compileRegex compiles a regex pattern with optional flags
// Go's regexp doesn't support all Perl flags, so we map what we can
func compileRegex(pattern, flags string) (*regexp.Regexp, error) {
	// Process flags - Go regexp supports (?flags) syntax
	prefix := ""
	for _, flag := range flags {
		switch flag {
		case 'i': // case-insensitive
			prefix += "(?i)"
		case 'm': // multi-line (^ and $ match line boundaries)
			prefix += "(?m)"
		case 's': // dot matches newline
			prefix += "(?s)"
			// 'g' (global) is handled by match operator, not compilation
			// Other flags like 'x' (verbose) could be added
		}
	}

	fullPattern := prefix + pattern
	return regexp.Compile(fullPattern)
}

// evalMatchExpression handles string ~ regex matching
// Returns an array of matches (with captures) or null if no match
func evalMatchExpression(tok lexer.Token, text string, regexDict *Dictionary, env *Environment) Object {
	// Extract pattern and flags from regex dictionary
	patternExpr, ok := regexDict.Pairs["pattern"]
	if !ok {
		return newErrorWithPos(tok, "regex dictionary missing pattern field")
	}
	patternObj := Eval(patternExpr, env)
	patternStr, ok := patternObj.(*String)
	if !ok {
		return newErrorWithPos(tok, "regex pattern must be a string")
	}

	flagsExpr, ok := regexDict.Pairs["flags"]
	var flags string
	if ok {
		flagsObj := Eval(flagsExpr, env)
		if flagsStr, ok := flagsObj.(*String); ok {
			flags = flagsStr.Value
		}
	}

	// Compile the regex
	re, err := compileRegex(patternStr.Value, flags)
	if err != nil {
		return newErrorWithPos(tok, "invalid regex: %s", err.Error())
	}

	// Find matches
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		return NULL // No match - returns null (falsy)
	}

	// Convert matches to array of strings
	elements := make([]Object, len(matches))
	for i, match := range matches {
		elements[i] = &String{Value: match}
	}

	return &Array{Elements: elements}
}

// cleanPathComponents implements Rob Pike's cleanname algorithm from Plan 9
// to canonicalize path components. This ensures paths always present clean file names.
// See: https://9p.io/sys/doc/lexnames.html
//
// Rules:
// 1. Reduce multiple slashes to a single slash (handled by parsePathString)
// 2. Eliminate . path name elements (the current directory)
// 3. Eliminate .. elements and the non-. non-.. element that precedes them
// 4. Eliminate .. elements that begin a rooted path (replace /.. by /)
// 5. Leave intact .. elements that begin a non-rooted path
//
// Note: For absolute paths, we prepend an empty string to represent the root.
// This is the traditional Unix convention: /usr/local → ["", "usr", "local"]
func cleanPathComponents(components []string, isAbsolute bool) []string {
	var result []string

	for _, comp := range components {
		switch comp {
		case "":
			// Skip empty components (multiple slashes already handled)
			continue
		case ".":
			// Rule 2: Eliminate . (current directory)
			continue
		case "..":
			if len(result) > 0 && result[len(result)-1] != ".." {
				// Rule 3: Eliminate .. and the preceding element
				result = result[:len(result)-1]
			} else if isAbsolute {
				// Rule 4: Eliminate .. at the beginning of rooted paths
				// (do nothing, effectively replacing /.. with /)
			} else {
				// Rule 5: Leave .. intact at the beginning of non-rooted paths
				result = append(result, comp)
			}
		default:
			result = append(result, comp)
		}
	}

	// If result is empty, return current directory for relative paths
	// For absolute paths with no components (just "/"), return empty slice
	// The absolute flag will be used during reconstruction to add leading "/"
	if len(result) == 0 && !isAbsolute {
		return []string{"."} // Current directory
	}

	return result
}

// parsePathString parses a file path string into components
// Returns components array and whether path is absolute
// The path is cleaned using Rob Pike's cleanname algorithm
func parsePathString(pathStr string) ([]string, bool) {
	if pathStr == "" {
		return []string{"."}, false
	}

	// Detect absolute vs relative
	isAbsolute := false
	hasLeadingDot := false
	if pathStr[0] == '/' {
		isAbsolute = true
	} else if len(pathStr) >= 2 && pathStr[1] == ':' {
		// Windows drive letter (C:, D:, etc.)
		isAbsolute = true
	} else if pathStr[0] == '.' && (len(pathStr) == 1 || pathStr[1] == '/') {
		// Starts with ./ - remember this for output
		hasLeadingDot = true
	} else if pathStr[0] == '~' {
		// Home directory reference - treat specially
		hasLeadingDot = false
	}

	// Split on forward slashes (handle both Unix and Windows)
	pathStr = strings.ReplaceAll(pathStr, "\\", "/")
	parts := strings.Split(pathStr, "/")

	// Collect raw components
	components := []string{}
	for _, part := range parts {
		if part != "" {
			components = append(components, part)
		}
	}

	// Clean the path components
	cleaned := cleanPathComponents(components, isAbsolute)

	// For relative paths that originally started with ./, preserve that style
	// unless the cleaned result already starts with . or ..
	if hasLeadingDot && len(cleaned) > 0 && cleaned[0] != "." && cleaned[0] != ".." {
		cleaned = append([]string{"."}, cleaned...)
	}

	return cleaned, isAbsolute
}

// pathToDict creates a path dictionary from components
func pathToDict(components []string, isAbsolute bool, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "path"},
		Value: "path",
	}

	// Add components as array literal
	componentExprs := make([]ast.Expression, len(components))
	for i, comp := range components {
		componentExprs[i] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: comp},
			Value: comp,
		}
	}
	pairs["components"] = &ast.ArrayLiteral{
		Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: componentExprs,
	}

	// Add absolute flag
	tokenType := lexer.FALSE
	tokenLiteral := "false"
	if isAbsolute {
		tokenType = lexer.TRUE
		tokenLiteral = "true"
	}
	pairs["absolute"] = &ast.Boolean{
		Token: lexer.Token{Type: tokenType, Literal: tokenLiteral},
		Value: isAbsolute,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// stdioToDict creates a path dictionary for stdin/stdout/stderr
func stdioToDict(stream string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "path"},
		Value: "path",
	}

	// Add __stdio field to mark this as a stdio path
	pairs["__stdio"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stream},
		Value: stream,
	}

	// Add components as array with just "-"
	pairs["components"] = &ast.ArrayLiteral{
		Token: lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: []ast.Expression{
			&ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: "-"},
				Value: "-",
			},
		},
	}

	// Not absolute
	pairs["absolute"] = &ast.Boolean{
		Token: lexer.Token{Type: lexer.FALSE, Literal: "false"},
		Value: false,
	}

	// Add path property as "-" for display
	pairs["path"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "-"},
		Value: "-",
	}

	// Add name property
	pairs["name"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stream},
		Value: stream,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// parseUrlString parses a URL string into components
// Supports: scheme://[user:pass@]host[:port]/path?query#fragment
func parseUrlString(urlStr string, env *Environment) (*Dictionary, error) {
	// Simple URL parsing (not using net/url to keep it simple)
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "url"},
		Value: "url",
	}

	// Parse scheme
	schemeEnd := strings.Index(urlStr, "://")
	if schemeEnd == -1 {
		return nil, fmt.Errorf("invalid URL: missing scheme (expected scheme://...)")
	}
	scheme := urlStr[:schemeEnd]
	rest := urlStr[schemeEnd+3:]

	pairs["scheme"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: scheme},
		Value: scheme,
	}

	// Parse fragment (if present)
	var fragment string
	if fragIdx := strings.Index(rest, "#"); fragIdx != -1 {
		fragment = rest[fragIdx+1:]
		rest = rest[:fragIdx]
	}

	// Parse query (if present)
	queryPairs := make(map[string]ast.Expression)
	if queryIdx := strings.Index(rest, "?"); queryIdx != -1 {
		queryStr := rest[queryIdx+1:]
		rest = rest[:queryIdx]

		// Parse query parameters
		for _, param := range strings.Split(queryStr, "&") {
			if param == "" {
				continue
			}
			parts := strings.SplitN(param, "=", 2)
			key := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}
			queryPairs[key] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: value},
				Value: value,
			}
		}
	}
	pairs["query"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: queryPairs,
	}

	// Parse path (if present)
	pathComponents := []string{}
	var pathStr string
	if pathIdx := strings.Index(rest, "/"); pathIdx != -1 {
		pathStr = rest[pathIdx:]
		rest = rest[:pathIdx]
		pathComponents, _ = parsePathString(pathStr)
	}

	pathExprs := make([]ast.Expression, len(pathComponents))
	for i, comp := range pathComponents {
		pathExprs[i] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: comp},
			Value: comp,
		}
	}
	pairs["path"] = &ast.ArrayLiteral{
		Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: pathExprs,
	}

	// Parse authority (user:pass@host:port)
	var username, password, host string
	var port int64 = 0

	// Check for userinfo (user:pass@)
	if atIdx := strings.Index(rest, "@"); atIdx != -1 {
		userinfo := rest[:atIdx]
		rest = rest[atIdx+1:]

		if colonIdx := strings.Index(userinfo, ":"); colonIdx != -1 {
			username = userinfo[:colonIdx]
			password = userinfo[colonIdx+1:]
		} else {
			username = userinfo
		}
	}

	// Parse host:port
	if colonIdx := strings.Index(rest, ":"); colonIdx != -1 {
		host = rest[:colonIdx]
		portStr := rest[colonIdx+1:]
		if p, err := strconv.ParseInt(portStr, 10, 64); err == nil {
			port = p
		}
	} else {
		host = rest
	}

	pairs["host"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: host},
		Value: host,
	}

	pairs["port"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", port)},
		Value: port,
	}

	if username != "" {
		pairs["username"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: username},
			Value: username,
		}
	} else {
		pairs["username"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	if password != "" {
		pairs["password"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: password},
			Value: password,
		}
	} else {
		pairs["password"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	if fragment != "" {
		pairs["fragment"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: fragment},
			Value: fragment,
		}
	} else {
		pairs["fragment"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}, nil
}

// evalPathComputedProperty returns computed properties for path dictionaries
// Returns nil if the property doesn't exist
func evalPathComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "basename":
		// Get last component
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		return arr.Elements[len(arr.Elements)-1]

	case "dirname", "parent":
		// Get all but last component, return as path dict
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}

		// Get absolute flag
		absoluteExpr, ok := dict.Pairs["absolute"]
		isAbsolute := false
		if ok {
			absoluteObj := Eval(absoluteExpr, env)
			if b, ok := absoluteObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		// Create new components array (all but last)
		parentComponents := []string{}
		for i := 0; i < len(arr.Elements)-1; i++ {
			if str, ok := arr.Elements[i].(*String); ok {
				parentComponents = append(parentComponents, str.Value)
			}
		}

		return pathToDict(parentComponents, isAbsolute, env)

	case "extension", "ext":
		// Get extension from basename
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return NULL
		}

		// Find last dot
		lastDot := strings.LastIndex(basename.Value, ".")
		if lastDot == -1 || lastDot == 0 {
			return &String{Value: ""}
		}
		return &String{Value: basename.Value[lastDot+1:]}

	case "stem":
		// Get filename without extension
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return NULL
		}

		// Find last dot
		lastDot := strings.LastIndex(basename.Value, ".")
		if lastDot == -1 || lastDot == 0 {
			return basename
		}
		return &String{Value: basename.Value[:lastDot]}

	case "name":
		// Alias for basename
		return evalPathComputedProperty(dict, "basename", env)

	case "suffix":
		// Alias for extension
		return evalPathComputedProperty(dict, "extension", env)

	case "suffixes":
		// Get all extensions as array (e.g., ["tar", "gz"] from file.tar.gz)
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return &Array{Elements: []Object{}}
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return &Array{Elements: []Object{}}
		}

		// Find all dots and extract suffixes
		var suffixes []Object
		parts := strings.Split(basename.Value, ".")
		if len(parts) > 1 {
			// Skip the first part (filename), collect rest as suffixes
			for i := 1; i < len(parts); i++ {
				if parts[i] != "" {
					suffixes = append(suffixes, &String{Value: parts[i]})
				}
			}
		}
		return &Array{Elements: suffixes}

	case "parts":
		// Alias for components
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return NULL
		}
		return Eval(componentsExpr, env)

	case "isAbsolute":
		// Boolean indicating if path is absolute
		absoluteExpr, ok := dict.Pairs["absolute"]
		if !ok {
			return FALSE
		}
		return Eval(absoluteExpr, env)

	case "isRelative":
		// Boolean indicating if path is relative (opposite of absolute)
		absoluteExpr, ok := dict.Pairs["absolute"]
		if !ok {
			return TRUE
		}
		absoluteObj := Eval(absoluteExpr, env)
		if b, ok := absoluteObj.(*Boolean); ok {
			return nativeBoolToParsBoolean(!b.Value)
		}
		return TRUE

	case "string":
		// Full path as string
		return &String{Value: pathDictToString(dict)}

	case "dir":
		// Directory path as string (all but the last component)
		componentsExpr, ok := dict.Pairs["components"]
		if !ok {
			return &String{Value: ""}
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) <= 1 {
			// If only one component (or empty), dir is empty or root
			absoluteExpr, ok := dict.Pairs["absolute"]
			isAbsolute := false
			if ok {
				absoluteObj := Eval(absoluteExpr, env)
				if b, ok := absoluteObj.(*Boolean); ok {
					isAbsolute = b.Value
				}
			}
			if isAbsolute {
				return &String{Value: "/"}
			}
			return &String{Value: "."}
		}

		// Get absolute flag
		absoluteExpr, ok := dict.Pairs["absolute"]
		isAbsolute := false
		if ok {
			absoluteObj := Eval(absoluteExpr, env)
			if b, ok := absoluteObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		// Build directory path (all but last component)
		var result strings.Builder
		for i := 0; i < len(arr.Elements)-1; i++ {
			if str, ok := arr.Elements[i].(*String); ok {
				if str.Value == "" && i == 0 && isAbsolute {
					result.WriteString("/")
				} else {
					if i > 0 && (i > 1 || !isAbsolute) {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
		return &String{Value: result.String()}
	}

	return nil // Property doesn't exist
}

// evalUrlComputedProperty returns computed properties for URL dictionaries
// Returns nil if the property doesn't exist
func evalUrlComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "origin":
		// scheme://host[:port]
		var result strings.Builder

		if schemeExpr, ok := dict.Pairs["scheme"]; ok {
			schemeObj := Eval(schemeExpr, env)
			if str, ok := schemeObj.(*String); ok {
				result.WriteString(str.Value)
				result.WriteString("://")
			}
		}

		if hostExpr, ok := dict.Pairs["host"]; ok {
			hostObj := Eval(hostExpr, env)
			if str, ok := hostObj.(*String); ok {
				result.WriteString(str.Value)
			}
		}

		if portExpr, ok := dict.Pairs["port"]; ok {
			portObj := Eval(portExpr, env)
			if i, ok := portObj.(*Integer); ok && i.Value != 0 {
				result.WriteString(":")
				result.WriteString(strconv.FormatInt(i.Value, 10))
			}
		}

		return &String{Value: result.String()}

	case "pathname":
		// Just the path part as a string (always with leading /)
		if pathExpr, ok := dict.Pairs["path"]; ok {
			pathObj := Eval(pathExpr, env)
			if arr, ok := pathObj.(*Array); ok {
				var parts []string
				for _, elem := range arr.Elements {
					if str, ok := elem.(*String); ok && str.Value != "" {
						parts = append(parts, str.Value)
					}
				}
				// URL paths always start with /
				return &String{Value: "/" + strings.Join(parts, "/")}
			}
		}
		return &String{Value: "/"}

	case "hostname":
		// Alias for host
		if hostExpr, ok := dict.Pairs["host"]; ok {
			return Eval(hostExpr, env)
		}
		return &String{Value: ""}

	case "protocol":
		// Scheme with colon suffix (e.g., "https:")
		if schemeExpr, ok := dict.Pairs["scheme"]; ok {
			schemeObj := Eval(schemeExpr, env)
			if str, ok := schemeObj.(*String); ok {
				return &String{Value: str.Value + ":"}
			}
		}
		return &String{Value: ""}

	case "search":
		// Query string with ? prefix (e.g., "?key=value&foo=bar")
		if queryExpr, ok := dict.Pairs["query"]; ok {
			queryObj := Eval(queryExpr, env)
			if queryDict, ok := queryObj.(*Dictionary); ok {
				if len(queryDict.Pairs) == 0 {
					return &String{Value: ""}
				}
				var result strings.Builder
				result.WriteString("?")
				first := true
				for key, expr := range queryDict.Pairs {
					val := Eval(expr, env)
					if str, ok := val.(*String); ok {
						if !first {
							result.WriteString("&")
						}
						result.WriteString(key)
						result.WriteString("=")
						result.WriteString(str.Value)
						first = false
					}
				}
				return &String{Value: result.String()}
			}
		}
		return &String{Value: ""}

	case "href":
		// Full URL as string (alias for toString)
		return &String{Value: urlDictToString(dict)}

	case "string":
		// Full URL as string (alias for href)
		return &String{Value: urlDictToString(dict)}
	}

	return nil // Property doesn't exist
}

// fileToDict creates a file dictionary from a path and format
// format can be: "json", "csv", "lines", "text", "bytes", or "" for auto-detect
func fileToDict(pathDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "file"},
		Value: "file",
	}

	// Add path field (the original path dictionary)
	// Store the path components and absolute flag from the path dict
	if compExpr, ok := pathDict.Pairs["components"]; ok {
		pairs["_pathComponents"] = compExpr
	}
	if absExpr, ok := pathDict.Pairs["absolute"]; ok {
		pairs["_pathAbsolute"] = absExpr
	}

	// Propagate __stdio marker from path dict (for stdin/stdout/stderr)
	if stdioExpr, ok := pathDict.Pairs["__stdio"]; ok {
		pairs["__stdio"] = stdioExpr
	}

	// Add format field
	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Add options field (if provided)
	if options != nil {
		// Copy options to ast expressions
		optPairs := make(map[string]ast.Expression)
		for k, v := range options.Pairs {
			optPairs[k] = v
		}
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: optPairs,
		}
	} else {
		// Empty options
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// dirToDict creates a directory dictionary from a path dictionary
// Directory dictionaries have __type: "dir" and can be read to list contents
func dirToDict(pathDict *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "dir"},
		Value: "dir",
	}

	// Store the path components and absolute flag from the path dict
	if compExpr, ok := pathDict.Pairs["components"]; ok {
		pairs["_pathComponents"] = compExpr
	}
	if absExpr, ok := pathDict.Pairs["absolute"]; ok {
		pairs["_pathAbsolute"] = absExpr
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// isDirDict checks if a dictionary is a directory handle
func isDirDict(dict *Dictionary) bool {
	typeExpr, ok := dict.Pairs["__type"]
	if !ok {
		return false
	}
	if lit, ok := typeExpr.(*ast.StringLiteral); ok {
		return lit.Value == "dir"
	}
	if ident, ok := typeExpr.(*ast.Identifier); ok {
		return ident.Value == "dir"
	}
	return false
}

// fileDictToPathDict converts a file/dir dictionary to a path dictionary
// File dicts use _pathComponents/_pathAbsolute, path dicts use components/absolute
func fileDictToPathDict(dict *Dictionary) *Dictionary {
	compExpr, ok := dict.Pairs["_pathComponents"]
	if !ok {
		return nil
	}
	absExpr := dict.Pairs["_pathAbsolute"]
	if absExpr == nil {
		absExpr = &ast.Boolean{Value: false}
	}

	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"components": compExpr,
			"absolute":   absExpr,
		},
		Env: dict.Env,
	}
}

// evalDirComputedProperty returns computed properties for directory dictionaries
func evalDirComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	pathStr := getFilePathString(dict, env)

	switch key {
	case "path":
		// Return the underlying path dictionary
		compExpr, ok := dict.Pairs["_pathComponents"]
		if !ok {
			return NULL
		}
		compObj := Eval(compExpr, env)
		arr, ok := compObj.(*Array)
		if !ok {
			return NULL
		}

		absExpr, ok := dict.Pairs["_pathAbsolute"]
		isAbsolute := false
		if ok {
			absObj := Eval(absExpr, env)
			if b, ok := absObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		components := []string{}
		for _, elem := range arr.Elements {
			if str, ok := elem.(*String); ok {
				components = append(components, str.Value)
			}
		}

		return pathToDict(components, isAbsolute, env)

	case "exists":
		info, err := os.Stat(pathStr)
		return nativeBoolToParsBoolean(err == nil && info.IsDir())

	case "isDir":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(info.IsDir())

	case "isFile":
		return FALSE // Directories are not files

	case "name", "basename":
		return &String{Value: filepath.Base(pathStr)}

	case "parent", "dirname":
		dir := filepath.Dir(pathStr)
		components, isAbsolute := parsePathString(dir)
		return pathToDict(components, isAbsolute, env)

	case "mode":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &String{Value: ""}
		}
		return &String{Value: info.Mode().String()}

	case "modified":
		info, err := os.Stat(pathStr)
		if err != nil {
			return NULL
		}
		return timeToDatetimeDict(info.ModTime(), env)

	case "files":
		// Return array of file handles in directory
		return readDirContents(pathStr, env)

	case "count":
		// Return count of items in directory
		entries, err := os.ReadDir(pathStr)
		if err != nil {
			return &Integer{Value: 0}
		}
		return &Integer{Value: int64(len(entries))}
	}

	return nil // Property doesn't exist
}

// readDirContents reads directory contents and returns array of file/dir handles
func readDirContents(dirPath string, env *Environment) Object {
	// Security check
	if err := env.checkPathAccess(dirPath, "read"); err != nil {
		return newSecurityError("read", err)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return newIOError("IO-0003", dirPath, err)
	}

	elements := make([]Object, 0, len(entries))
	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		components, isAbsolute := parsePathString(entryPath)
		pathDict := pathToDict(components, isAbsolute, env)

		var handle *Dictionary
		if entry.IsDir() {
			handle = dirToDict(pathDict, env)
		} else {
			format := inferFormatFromExtension(entryPath)
			handle = fileToDict(pathDict, format, nil, env)
		}
		elements = append(elements, handle)
	}

	return &Array{Elements: elements}
}

// getFilePathString extracts the filesystem path string from a file dictionary
func getFilePathString(dict *Dictionary, env *Environment) string {
	// Get path components
	compExpr, ok := dict.Pairs["_pathComponents"]
	if !ok {
		return ""
	}
	if compExpr == nil {
		return ""
	}
	compObj := Eval(compExpr, env)
	arr, ok := compObj.(*Array)
	if !ok {
		return ""
	}

	// Get absolute flag
	absExpr, ok := dict.Pairs["_pathAbsolute"]
	isAbsolute := false
	if ok && absExpr != nil {
		absObj := Eval(absExpr, env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string
	var result strings.Builder

	// Add leading / for absolute paths
	if isAbsolute {
		result.WriteString("/")
	}

	for i, elem := range arr.Elements {
		if str, ok := elem.(*String); ok {
			if str.Value == "." && i == 0 && !isAbsolute {
				result.WriteString(".")
			} else if str.Value == "~" && i == 0 {
				// Keep ~ unexpanded - resolveModulePath will handle it
				// This allows ~/ to mean "handler root" in Basil context
				result.WriteString("~")
			} else if str.Value != "" {
				if i > 0 || (isAbsolute && result.Len() > 1) {
					result.WriteString("/")
				}
				result.WriteString(str.Value)
			}
		}
	}

	// Handle empty result
	if result.Len() == 0 {
		return "."
	}

	return result.String()
}

// inferFormatFromExtension guesses the file format from its extension
func inferFormatFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return "json"
	case ".csv":
		return "csv"
	case ".txt", ".md", ".html", ".xml", ".pars":
		return "text"
	case ".log":
		return "lines"
	default:
		return "text" // Default to text
	}
}

// evalFileComputedProperty returns computed properties for file dictionaries
// Returns nil if the property doesn't exist
func evalFileComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	pathStr := getFilePathString(dict, env)

	switch key {
	case "path":
		// Return the underlying path dictionary
		compExpr, ok := dict.Pairs["_pathComponents"]
		if !ok {
			return NULL
		}
		compObj := Eval(compExpr, env)
		arr, ok := compObj.(*Array)
		if !ok {
			return NULL
		}

		absExpr, ok := dict.Pairs["_pathAbsolute"]
		isAbsolute := false
		if ok {
			absObj := Eval(absExpr, env)
			if b, ok := absObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		components := []string{}
		for _, elem := range arr.Elements {
			if str, ok := elem.(*String); ok {
				components = append(components, str.Value)
			}
		}

		return pathToDict(components, isAbsolute, env)

	case "exists":
		_, err := os.Stat(pathStr)
		return nativeBoolToParsBoolean(err == nil)

	case "size":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &Integer{Value: 0}
		}
		return &Integer{Value: info.Size()}

	case "modified":
		info, err := os.Stat(pathStr)
		if err != nil {
			return NULL
		}
		return timeToDatetimeDict(info.ModTime(), env)

	case "isDir":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(info.IsDir())

	case "isFile":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(!info.IsDir())

	case "mode":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &String{Value: ""}
		}
		return &String{Value: info.Mode().String()}

	case "ext", "extension":
		ext := filepath.Ext(pathStr)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		return &String{Value: ext}

	case "basename", "name":
		return &String{Value: filepath.Base(pathStr)}

	case "dirname", "parent":
		dir := filepath.Dir(pathStr)
		components, isAbsolute := parsePathString(dir)
		return pathToDict(components, isAbsolute, env)

	case "stem":
		base := filepath.Base(pathStr)
		ext := filepath.Ext(base)
		return &String{Value: strings.TrimSuffix(base, ext)}
	}

	return nil // Property doesn't exist
}

// timeToDatetimeDict converts a time.Time to a datetime dictionary
func timeToDatetimeDict(t time.Time, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "datetime"},
		Value: "datetime",
	}

	pairs["year"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(t.Year())},
		Value: int64(t.Year()),
	}

	pairs["month"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(int(t.Month()))},
		Value: int64(t.Month()),
	}

	pairs["day"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(t.Day())},
		Value: int64(t.Day()),
	}

	pairs["hour"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(t.Hour())},
		Value: int64(t.Hour()),
	}

	pairs["minute"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(t.Minute())},
		Value: int64(t.Minute()),
	}

	pairs["second"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(t.Second())},
		Value: int64(t.Second()),
	}

	pairs["unix"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(t.Unix(), 10)},
		Value: t.Unix(),
	}

	weekday := t.Weekday().String()
	pairs["weekday"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: weekday},
		Value: weekday,
	}

	iso := t.UTC().Format(time.RFC3339)
	// Simplify to the format we use
	if strings.HasSuffix(iso, "+00:00") || strings.HasSuffix(iso, "-00:00") {
		iso = strings.TrimSuffix(iso, "+00:00")
		iso = strings.TrimSuffix(iso, "-00:00")
		iso = iso + "Z"
	}
	pairs["iso"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: iso},
		Value: iso,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// evalDatetimeComputedProperty returns computed properties for datetime dictionaries
// Returns nil if the property doesn't exist
func evalDatetimeComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "date":
		// Just the date part as string (YYYY-MM-DD)
		if yearExpr, ok := dict.Pairs["year"]; ok {
			if monthExpr, ok := dict.Pairs["month"]; ok {
				if dayExpr, ok := dict.Pairs["day"]; ok {
					year := Eval(yearExpr, env)
					month := Eval(monthExpr, env)
					day := Eval(dayExpr, env)
					if yInt, ok := year.(*Integer); ok {
						if mInt, ok := month.(*Integer); ok {
							if dInt, ok := day.(*Integer); ok {
								return &String{Value: fmt.Sprintf("%04d-%02d-%02d", yInt.Value, mInt.Value, dInt.Value)}
							}
						}
					}
				}
			}
		}
		return NULL

	case "time":
		// Just the time part as string (HH:MM:SS or HH:MM if seconds are zero)
		if hourExpr, ok := dict.Pairs["hour"]; ok {
			if minExpr, ok := dict.Pairs["minute"]; ok {
				if secExpr, ok := dict.Pairs["second"]; ok {
					hour := Eval(hourExpr, env)
					minute := Eval(minExpr, env)
					second := Eval(secExpr, env)
					if hInt, ok := hour.(*Integer); ok {
						if mInt, ok := minute.(*Integer); ok {
							if sInt, ok := second.(*Integer); ok {
								if sInt.Value == 0 {
									return &String{Value: fmt.Sprintf("%02d:%02d", hInt.Value, mInt.Value)}
								}
								return &String{Value: fmt.Sprintf("%02d:%02d:%02d", hInt.Value, mInt.Value, sInt.Value)}
							}
						}
					}
				}
			}
		}
		return NULL

	case "format":
		// Human-readable format: "Month DD, YYYY" or "Month DD, YYYY at HH:MM"
		//
		// Note: THIS IS A SIMPLE IMPLEMENTATION
		// as it does not handle localization.
		//
		if yearExpr, ok := dict.Pairs["year"]; ok {
			if monthExpr, ok := dict.Pairs["month"]; ok {
				if dayExpr, ok := dict.Pairs["day"]; ok {
					year := Eval(yearExpr, env)
					month := Eval(monthExpr, env)
					day := Eval(dayExpr, env)
					if yInt, ok := year.(*Integer); ok {
						if mInt, ok := month.(*Integer); ok {
							if dInt, ok := day.(*Integer); ok {
								monthNames := []string{
									"January", "February", "March", "April", "May", "June",
									"July", "August", "September", "October", "November", "December",
								}
								monthName := "Invalid"
								if mInt.Value >= 1 && mInt.Value <= 12 {
									monthName = monthNames[mInt.Value-1]
								}

								// Check if time is set (not all zeros)
								hasTime := false
								if hourExpr, ok := dict.Pairs["hour"]; ok {
									if minExpr, ok := dict.Pairs["minute"]; ok {
										hour := Eval(hourExpr, env)
										minute := Eval(minExpr, env)
										if hInt, ok := hour.(*Integer); ok {
											if mInt, ok := minute.(*Integer); ok {
												if hInt.Value != 0 || mInt.Value != 0 {
													hasTime = true
													timeStr := fmt.Sprintf("%02d:%02d", hInt.Value, mInt.Value)
													return &String{Value: fmt.Sprintf("%s %d, %d at %s", monthName, dInt.Value, yInt.Value, timeStr)}
												}
											}
										}
									}
								}

								if !hasTime {
									return &String{Value: fmt.Sprintf("%s %d, %d", monthName, dInt.Value, yInt.Value)}
								}
							}
						}
					}
				}
			}
		}
		return NULL

	case "timestamp":
		// Alias for unix field - more intuitive name
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			return Eval(unixExpr, env)
		}
		return NULL

	case "dayOfYear":
		// Calculate day of year (1-366)
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			unixObj := Eval(unixExpr, env)
			if unixInt, ok := unixObj.(*Integer); ok {
				t := time.Unix(unixInt.Value, 0).UTC()
				return &Integer{Value: int64(t.YearDay())}
			}
		}
		return NULL

	case "week":
		// ISO week number (1-53)
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			unixObj := Eval(unixExpr, env)
			if unixInt, ok := unixObj.(*Integer); ok {
				t := time.Unix(unixInt.Value, 0).UTC()
				_, week := t.ISOWeek()
				return &Integer{Value: int64(week)}
			}
		}
		return NULL
	}

	return nil // Property doesn't exist
}

// getPublicDirComponents extracts public_dir components from basil config in environment
// Returns nil if basil.public_dir is not set or path is outside public_dir
func getPublicDirComponents(env *Environment) []string {
	if env == nil {
		return nil
	}

	// Get basil object from environment
	basilObj, ok := env.Get("basil")
	if !ok || basilObj == nil {
		return nil
	}

	// Extract basil.public_dir
	basilDict, ok := basilObj.(*Dictionary)
	if !ok {
		return nil
	}

	publicDirExpr, ok := basilDict.Pairs["public_dir"]
	if !ok {
		return nil
	}

	publicDirObj := Eval(publicDirExpr, env)
	publicDirStr, ok := publicDirObj.(*String)
	if !ok || publicDirStr.Value == "" {
		return nil
	}

	// Parse public_dir into components (e.g., "./public" → ["public"])
	publicDir := publicDirStr.Value

	// Clean the path and split into components
	// Handle "./public", "public", "./public/assets" etc.
	publicDir = strings.TrimPrefix(publicDir, "./")
	publicDir = strings.TrimPrefix(publicDir, "/")
	publicDir = strings.TrimSuffix(publicDir, "/")

	if publicDir == "" {
		return nil
	}

	return strings.Split(publicDir, "/")
}

// pathDictToString converts a path dictionary back to a string
func pathDictToString(dict *Dictionary) string {
	// Get components array
	componentsExpr, ok := dict.Pairs["components"]
	if !ok {
		return ""
	}

	// Evaluate the array expression
	componentsObj := Eval(componentsExpr, dict.Env)
	arr, ok := componentsObj.(*Array)
	if !ok {
		return ""
	}

	// Get absolute flag
	isAbsolute := false
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		absObj := Eval(absExpr, dict.Env)
		if b, ok := absObj.(*Boolean); ok {
			isAbsolute = b.Value
		}
	}

	// Build path string from components
	var parts []string
	for _, elem := range arr.Elements {
		if str, ok := elem.(*String); ok {
			parts = append(parts, str.Value)
		}
	}

	if len(parts) == 0 {
		if isAbsolute {
			return "/"
		}
		return "."
	}

	// Join components and add leading / for absolute paths
	result := strings.Join(parts, "/")
	if isAbsolute {
		return "/" + result
	}
	return result
}

// pathToWebURL transforms a path under public_dir to a web URL
// e.g., ./public/images/foo.png -> /images/foo.png (when public_dir is ./public)
// Returns the original path if not under public_dir or if public_dir is not set
func pathToWebURL(dict *Dictionary) string {
	if dict.Env == nil {
		return pathDictToString(dict)
	}

	// Get the file path as a string first
	filePath := pathDictToString(dict)
	if filePath == "" {
		return ""
	}

	// Get public_dir from environment
	publicDirStr := getPublicDir(dict.Env)
	if publicDirStr == "" {
		return filePath
	}

	// Resolve both paths to absolute for comparison
	var absFilePath, absPublicDir string

	if filepath.IsAbs(filePath) {
		absFilePath = filepath.Clean(filePath)
	} else {
		// Relative to root path if available
		if dict.Env.RootPath != "" {
			absFilePath = filepath.Clean(filepath.Join(dict.Env.RootPath, filePath))
		} else {
			absFilePath = filepath.Clean(filePath)
		}
	}

	if filepath.IsAbs(publicDirStr) {
		absPublicDir = filepath.Clean(publicDirStr)
	} else {
		// Relative to root path if available
		if dict.Env.RootPath != "" {
			absPublicDir = filepath.Clean(filepath.Join(dict.Env.RootPath, publicDirStr))
		} else {
			absPublicDir = filepath.Clean(publicDirStr)
		}
	}

	// Check if file path is under public_dir
	// Use HasPrefix on cleaned absolute paths
	if strings.HasPrefix(absFilePath, absPublicDir+"/") {
		// Strip public_dir prefix and return as web-root-relative path
		webPath := strings.TrimPrefix(absFilePath, absPublicDir)
		if webPath == "" || webPath == "/" {
			return "/"
		}
		// Ensure it starts with /
		if !strings.HasPrefix(webPath, "/") {
			webPath = "/" + webPath
		}
		return webPath
	}

	// Exact match (file IS the public dir root)
	if absFilePath == absPublicDir {
		return "/"
	}

	// Not under public_dir, return as-is
	return filePath
}

// getPublicDir returns the public_dir string from the environment
func getPublicDir(env *Environment) string {
	if env == nil {
		return ""
	}

	// Get basil object from environment
	basilObj, ok := env.Get("basil")
	if !ok || basilObj == nil {
		return ""
	}

	// Extract basil.public_dir
	basilDict, ok := basilObj.(*Dictionary)
	if !ok {
		return ""
	}

	publicDirExpr, ok := basilDict.Pairs["public_dir"]
	if !ok {
		return ""
	}

	publicDirObj := Eval(publicDirExpr, env)
	publicDirStr, ok := publicDirObj.(*String)
	if !ok {
		return ""
	}

	return publicDirStr.Value
}

// urlDictToString converts a URL dictionary back to a string
func urlDictToString(dict *Dictionary) string {
	var result strings.Builder

	// Scheme
	if schemeExpr, ok := dict.Pairs["scheme"]; ok {
		schemeObj := Eval(schemeExpr, dict.Env)
		if str, ok := schemeObj.(*String); ok {
			result.WriteString(str.Value)
			result.WriteString("://")
		}
	}

	// Username and password
	if usernameExpr, ok := dict.Pairs["username"]; ok {
		usernameObj := Eval(usernameExpr, dict.Env)
		if str, ok := usernameObj.(*String); ok && str.Value != "" {
			result.WriteString(str.Value)

			if passwordExpr, ok := dict.Pairs["password"]; ok {
				passwordObj := Eval(passwordExpr, dict.Env)
				if pstr, ok := passwordObj.(*String); ok && pstr.Value != "" {
					result.WriteString(":")
					result.WriteString(pstr.Value)
				}
			}
			result.WriteString("@")
		}
	}

	// Host
	if hostExpr, ok := dict.Pairs["host"]; ok {
		hostObj := Eval(hostExpr, dict.Env)
		if str, ok := hostObj.(*String); ok {
			result.WriteString(str.Value)
		}
	}

	// Port (if non-zero)
	if portExpr, ok := dict.Pairs["port"]; ok {
		portObj := Eval(portExpr, dict.Env)
		if i, ok := portObj.(*Integer); ok && i.Value != 0 {
			result.WriteString(":")
			result.WriteString(strconv.FormatInt(i.Value, 10))
		}
	}

	// Path
	if pathExpr, ok := dict.Pairs["path"]; ok {
		pathObj := Eval(pathExpr, dict.Env)
		if arr, ok := pathObj.(*Array); ok && len(arr.Elements) > 0 {
			// Check if first element is empty string (indicates leading slash)
			startIdx := 0
			if str, ok := arr.Elements[0].(*String); ok && str.Value == "" {
				// Leading empty string means path starts with /
				result.WriteString("/")
				startIdx = 1
			} else if len(arr.Elements) > 0 {
				// No leading empty, but we have segments, so add /
				result.WriteString("/")
			}

			// Add remaining path segments
			for i := startIdx; i < len(arr.Elements); i++ {
				if str, ok := arr.Elements[i].(*String); ok && str.Value != "" {
					if i > startIdx {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Query
	if queryExpr, ok := dict.Pairs["query"]; ok {
		queryObj := Eval(queryExpr, dict.Env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, expr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				result.WriteString(key)
				result.WriteString("=")
				valObj := Eval(expr, dict.Env)
				if str, ok := valObj.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Fragment
	if fragmentExpr, ok := dict.Pairs["fragment"]; ok {
		fragmentObj := Eval(fragmentExpr, dict.Env)
		if str, ok := fragmentObj.(*String); ok && str.Value != "" {
			result.WriteString("#")
			result.WriteString(str.Value)
		}
	}

	return result.String()
}

// getBuiltins returns the map of built-in functions
func getBuiltins() map[string]*Builtin {
	return map[string]*Builtin{
		"SQLITE": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("SQLITE", len(args), 1, 2)
				}

				// First arg: path literal
				pathStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "SQLITE", "a path", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "SQLITE", "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				// Create DSN (SQLite just uses the path, with special handling for :memory:)
				dsn := pathStr.Value

				// Check cache
				cacheKey := "sqlite:" + dsn
				dbConnectionsMu.RLock()
				db, exists := dbConnections[cacheKey]
				dbConnectionsMu.RUnlock()

				if !exists {
					var err error
					db, err = sql.Open("sqlite", dsn)
					if err != nil {
						return newDatabaseErrorWithDriver("DB-0003", "SQLite", err)
					}

					// Apply connection options if provided
					if options != nil {
						if maxOpen, ok := options["maxOpenConns"]; ok {
							if maxOpenInt, ok := maxOpen.(*Integer); ok {
								db.SetMaxOpenConns(int(maxOpenInt.Value))
							}
						}
						if maxIdle, ok := options["maxIdleConns"]; ok {
							if maxIdleInt, ok := maxIdle.(*Integer); ok {
								db.SetMaxIdleConns(int(maxIdleInt.Value))
							}
						}
					}

					// Test connection
					if err := db.Ping(); err != nil {
						db.Close()
						return newDatabaseErrorWithDriver("DB-0005", "SQLite", err)
					}

					// Cache connection
					dbConnectionsMu.Lock()
					dbConnections[cacheKey] = db
					dbConnectionsMu.Unlock()
				}

				return &DBConnection{
					DB:            db,
					Driver:        "sqlite",
					DSN:           dsn,
					InTransaction: false,
					LastError:     "",
				}
			},
		},
		"POSTGRES": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("POSTGRES", len(args), 1, 2)
				}

				// First arg: URL literal
				urlStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "POSTGRES", "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "POSTGRES", "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				dsn := urlStr.Value

				// Check cache
				cacheKey := "postgres:" + dsn
				dbConnectionsMu.RLock()
				db, exists := dbConnections[cacheKey]
				dbConnectionsMu.RUnlock()

				if !exists {
					var err error
					db, err = sql.Open("postgres", dsn)
					if err != nil {
						return newDatabaseErrorWithDriver("DB-0003", "PostgreSQL", err)
					}

					// Apply connection options if provided
					if options != nil {
						if maxOpen, ok := options["maxOpenConns"]; ok {
							if maxOpenInt, ok := maxOpen.(*Integer); ok {
								db.SetMaxOpenConns(int(maxOpenInt.Value))
							}
						}
						if maxIdle, ok := options["maxIdleConns"]; ok {
							if maxIdleInt, ok := maxIdle.(*Integer); ok {
								db.SetMaxIdleConns(int(maxIdleInt.Value))
							}
						}
					}

					// Test connection
					if err := db.Ping(); err != nil {
						db.Close()
						return newDatabaseErrorWithDriver("DB-0005", "PostgreSQL", err)
					}

					// Cache connection
					dbConnectionsMu.Lock()
					dbConnections[cacheKey] = db
					dbConnectionsMu.Unlock()
				}

				return &DBConnection{
					DB:            db,
					Driver:        "postgres",
					DSN:           dsn,
					InTransaction: false,
					LastError:     "",
				}
			},
		},
		"MYSQL": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("MYSQL", len(args), 1, 2)
				}

				// First arg: URL literal
				urlStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "MYSQL", "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "MYSQL", "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				dsn := urlStr.Value

				// Check cache
				cacheKey := "mysql:" + dsn
				dbConnectionsMu.RLock()
				db, exists := dbConnections[cacheKey]
				dbConnectionsMu.RUnlock()

				if !exists {
					var err error
					db, err = sql.Open("mysql", dsn)
					if err != nil {
						return newDatabaseErrorWithDriver("DB-0003", "MySQL", err)
					}

					// Apply connection options if provided
					if options != nil {
						if maxOpen, ok := options["maxOpenConns"]; ok {
							if maxOpenInt, ok := maxOpen.(*Integer); ok {
								db.SetMaxOpenConns(int(maxOpenInt.Value))
							}
						}
						if maxIdle, ok := options["maxIdleConns"]; ok {
							if maxIdleInt, ok := maxIdle.(*Integer); ok {
								db.SetMaxIdleConns(int(maxIdleInt.Value))
							}
						}
					}

					// Test connection
					if err := db.Ping(); err != nil {
						db.Close()
						return newDatabaseErrorWithDriver("DB-0005", "MySQL", err)
					}

					// Cache connection
					dbConnectionsMu.Lock()
					dbConnections[cacheKey] = db
					dbConnectionsMu.Unlock()
				}

				return &DBConnection{
					DB:            db,
					Driver:        "mysql",
					DSN:           dsn,
					InTransaction: false,
					LastError:     "",
				}
			},
		},
		"SFTP": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("SFTP", len(args), 1, 2)
				}

				// First arg: URL (can be dictionary or string)
				var urlStr string
				switch arg := args[0].(type) {
				case *Dictionary:
					if !isUrlDict(arg) {
						return newTypeError("TYPE-0005", "SFTP", "a URL", DICTIONARY_OBJ)
					}
					// Extract URL string from dictionary
					if schemeExpr, ok := arg.Pairs["scheme"]; ok {
						scheme := Eval(schemeExpr, arg.Env)
						if schemeVal, ok := scheme.(*String); ok && schemeVal.Value != "sftp" {
							return newFormatError("FMT-0003", fmt.Errorf("SFTP requires sftp:// URL scheme, got %s://", schemeVal.Value))
						}
					}
					urlStr = urlDictToString(arg)
				case *String:
					urlStr = arg.Value
				default:
					return newTypeError("TYPE-0005", "SFTP", "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "SFTP", "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				// Parse SFTP URL
				if !strings.HasPrefix(urlStr, "sftp://") {
					return newFormatError("FMT-0003", fmt.Errorf("SFTP URL must start with sftp://"))
				}

				// Parse URL components
				parsedURL := urlStr[7:] // Remove "sftp://"
				var user, password, host string
				port := 22

				// Extract user@host:port
				atIndex := strings.Index(parsedURL, "@")
				if atIndex >= 0 {
					userPass := parsedURL[:atIndex]
					parsedURL = parsedURL[atIndex+1:]

					// Check for password in user:pass format
					colonIndex := strings.Index(userPass, ":")
					if colonIndex >= 0 {
						user = userPass[:colonIndex]
						password = userPass[colonIndex+1:]
					} else {
						user = userPass
					}
				} else {
					user = "anonymous"
				}

				// Extract host and port
				slashIndex := strings.Index(parsedURL, "/")
				hostPort := parsedURL
				if slashIndex >= 0 {
					hostPort = parsedURL[:slashIndex]
				}

				colonIndex := strings.LastIndex(hostPort, ":")
				if colonIndex >= 0 {
					host = hostPort[:colonIndex]
					portStr := hostPort[colonIndex+1:]
					if p, err := strconv.Atoi(portStr); err == nil {
						port = p
					}
				} else {
					host = hostPort
				}

				// Check cache
				cacheKey := fmt.Sprintf("sftp:%s@%s:%d", user, host, port)
				sftpConnectionsMu.RLock()
				conn, exists := sftpConnections[cacheKey]
				sftpConnectionsMu.RUnlock()

				if exists && conn.Connected {
					return conn
				}

				// Create new SFTP connection
				var authMethods []ssh.AuthMethod

				// Check for SSH key authentication
				if options != nil {
					if keyFileObj, ok := options["keyFile"]; ok {
						var keyPath string
						if keyDict, ok := keyFileObj.(*Dictionary); ok && isPathDict(keyDict) {
							keyPath = pathDictToString(keyDict)
						} else if keyStr, ok := keyFileObj.(*String); ok {
							keyPath = keyStr.Value
						}

						if keyPath != "" {
							keyData, err := os.ReadFile(keyPath)
							if err != nil {
								return newNetworkError("NET-0006", err)
							}

							var signer ssh.Signer
							var signerErr error

							// Check if key has passphrase
							if passphraseObj, ok := options["passphrase"]; ok {
								if passphraseStr, ok := passphraseObj.(*String); ok {
									signer, signerErr = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphraseStr.Value))
								}
							} else {
								signer, signerErr = ssh.ParsePrivateKey(keyData)
							}

							if signerErr != nil {
								return newNetworkError("NET-0007", signerErr)
							}

							authMethods = append(authMethods, ssh.PublicKeys(signer))
						}
					}

					// Check for password from options
					if passwordObj, ok := options["password"]; ok {
						if passwordStr, ok := passwordObj.(*String); ok {
							password = passwordStr.Value
						}
					}
				}

				// Add password auth if password provided
				if password != "" {
					authMethods = append(authMethods, ssh.Password(password))
				}

				if len(authMethods) == 0 {
					perr := perrors.New("SEC-0006", nil)
					return &Error{
						Class:   ErrorClass(perr.Class),
						Code:    perr.Code,
						Message: perr.Message,
						Hints:   perr.Hints,
						Data:    perr.Data,
					}
				}

				// Configure SSH client
				config := &ssh.ClientConfig{
					User:            user,
					Auth:            authMethods,
					HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Default to accept any (user can override)
					Timeout:         30 * time.Second,
				}

				// Check for known_hosts file
				if options != nil {
					if knownHostsObj, ok := options["knownHostsFile"]; ok {
						var knownHostsPath string
						if khDict, ok := knownHostsObj.(*Dictionary); ok && isPathDict(khDict) {
							knownHostsPath = pathDictToString(khDict)
						} else if khStr, ok := knownHostsObj.(*String); ok {
							knownHostsPath = khStr.Value
						}

						if knownHostsPath != "" {
							callback, err := knownhosts.New(knownHostsPath)
							if err != nil {
								return newError("failed to load known_hosts: %s", err.Error())
							}
							config.HostKeyCallback = callback
						}
					}

					// Check for timeout
					if timeoutObj, ok := options["timeout"]; ok {
						if timeoutDict, ok := timeoutObj.(*Dictionary); ok && isDurationDict(timeoutDict) {
							tempEnv := NewEnvironment()
							_, seconds, err := getDurationComponents(timeoutDict, tempEnv)
							if err == nil {
								config.Timeout = time.Duration(seconds) * time.Second
							}
						}
					}
				}

				// Connect to SSH server
				sshClient, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), config)
				if err != nil {
					return newNetworkError("NET-0003", err)
				}

				// Create SFTP client
				sftpClient, err := sftp.NewClient(sshClient)
				if err != nil {
					sshClient.Close()
					return newNetworkError("NET-0009", err)
				}

				// Create connection object
				newConn := &SFTPConnection{
					Client:    sftpClient,
					SSHClient: sshClient,
					Host:      host,
					Port:      port,
					User:      user,
					Connected: true,
					LastError: "",
				}

				// Cache connection
				sftpConnectionsMu.Lock()
				sftpConnections[cacheKey] = newConn
				sftpConnectionsMu.Unlock()

				return newConn
			},
		},
		"import": {
			Fn: func(args ...Object) Object {
				// This is a placeholder - actual implementation happens in CallExpression
				// where we have access to the environment for path resolution
				return newError("import() requires environment context")
			},
		},
		"sin": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("sin", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Sin(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Sin(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "sin", "", arg.Type())
				}
			},
		},
		"cos": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("cos", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Cos(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Cos(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "cos", "", arg.Type())
				}
			},
		},
		"tan": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("tan", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Tan(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Tan(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "tan", "", arg.Type())
				}
			},
		},
		"asin": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("asin", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Asin(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Asin(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "asin", "", arg.Type())
				}
			},
		},
		"acos": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("acos", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Acos(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Acos(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "acos", "", arg.Type())
				}
			},
		},
		"atan": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("atan", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Atan(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Atan(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "atan", "", arg.Type())
				}
			},
		},
		"sqrt": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("sqrt", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return &Float{Value: math.Sqrt(float64(arg.Value))}
				case *Float:
					return &Float{Value: math.Sqrt(arg.Value)}
				default:
					return newTypeError("TYPE-0002", "sqrt", "", arg.Type())
				}
			},
		},
		"round": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("round", len(args), 1)
				}

				arg := args[0]
				switch arg := arg.(type) {
				case *Integer:
					return arg // already an integer
				case *Float:
					return &Integer{Value: int64(math.Round(arg.Value))}
				default:
					return newTypeError("TYPE-0002", "round", "", arg.Type())
				}
			},
		},
		"pow": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("pow", len(args), 2)
				}

				base := args[0]
				exp := args[1]

				var baseVal, expVal float64

				switch base := base.(type) {
				case *Integer:
					baseVal = float64(base.Value)
				case *Float:
					baseVal = base.Value
				default:
					return newTypeError("TYPE-0005", "pow", "a number", base.Type())
				}

				switch exp := exp.(type) {
				case *Integer:
					expVal = float64(exp.Value)
				case *Float:
					expVal = exp.Value
				default:
					return newTypeError("TYPE-0006", "pow", "a number", exp.Type())
				}

				return &Float{Value: math.Pow(baseVal, expVal)}
			},
		},
		"pi": {
			Fn: func(args ...Object) Object {
				if len(args) != 0 {
					return newArityError("pi", len(args), 0)
				}
				return &Float{Value: math.Pi}
			},
		},
		"now": {
			Fn: func(args ...Object) Object {
				if len(args) != 0 {
					return newArityError("now", len(args), 0)
				}
				// Get current environment from context (we'll pass it through the Builtin)
				// For now, create a new environment for the dictionary
				env := NewEnvironment()
				return timeToDict(time.Now(), env)
			},
		},
		"time": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("time", len(args), 1, 2)
				}

				env := NewEnvironment()
				var t time.Time
				var err error

				switch arg := args[0].(type) {
				case *String:
					// Try parsing as ISO 8601 first, then fall back to date-only format
					t, err = time.Parse(time.RFC3339, arg.Value)
					if err != nil {
						t, err = time.Parse("2006-01-02", arg.Value)
					}
					if err != nil {
						t, err = time.Parse("2006-01-02T15:04:05", arg.Value)
					}
					if err != nil {
						return newFormatError("FMT-0004", fmt.Errorf("cannot parse %q", arg.Value))
					}
				case *Integer:
					// Unix timestamp
					t = time.Unix(arg.Value, 0).UTC()
				case *Dictionary:
					// From dictionary
					t, err = dictToTime(arg, env)
					if err != nil {
						return newFormatError("FMT-0004", err)
					}
				default:
					return newTypeError("TYPE-0012", "time", "a string, integer, or dictionary", args[0].Type())
				}

				// Apply delta if provided
				if len(args) == 2 {
					delta, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "time", "a dictionary", args[1].Type())
					}
					t = applyDelta(t, delta, env)
				}

				return timeToDict(t, env)
			},
		},
		"url": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("url", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "url", "a string", args[0].Type())
				}

				env := NewEnvironment()
				urlDict, err := parseUrlString(str.Value, env)
				if err != nil {
					return newFormatError("FMT-0003", err)
				}

				return urlDict
			},
		},
		// File handle factories
		"file": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("file", len(args), 1, 2)
				}

				// First argument must be a path dictionary or string
				var pathDict *Dictionary
				env := NewEnvironment()

				switch arg := args[0].(type) {
				case *Dictionary:
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "file", "a path", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "file", "a path or string", args[0].Type())
				}

				// Get the path string for format inference
				pathStr := getFilePathString(&Dictionary{Pairs: map[string]ast.Expression{
					"_pathComponents": pathDict.Pairs["components"],
					"_pathAbsolute":   pathDict.Pairs["absolute"],
				}, Env: env}, env)

				// Auto-detect format from extension
				format := inferFormatFromExtension(pathStr)

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				return fileToDict(pathDict, format, options, env)
			},
		},
		"JSON": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("JSON", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// First argument can be a path, URL, or string
				switch arg := args[0].(type) {
				case *Dictionary:
					if isUrlDict(arg) {
						// URL dictionary - create request handle for fetch
						return requestToDict(arg, "json", options, env)
					}
					if isPathDict(arg) {
						// Path dictionary - create file handle
						return fileToDict(arg, "json", options, env)
					}
					return newTypeError("TYPE-0005", "JSON", "a path or URL", DICTIONARY_OBJ)
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict := pathToDict(components, isAbsolute, env)
					return fileToDict(pathDict, "json", options, env)
				default:
					return newTypeError("TYPE-0005", "JSON", "a path, URL, or string", args[0].Type())
				}
			},
		},
		"YAML": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("YAML", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "yaml", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "YAML", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "YAML", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "yaml", options, env)
			},
		},
		"CSV": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("CSV", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict (e.g., {header: true})
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "csv", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "CSV", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "CSV", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "csv", options, env)
			},
		},
		"lines": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("lines", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "lines", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "lines", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "lines", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "lines", options, env)
			},
		},
		"text": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("text", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict (e.g., {encoding: "latin1"})
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "text", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "text", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "text", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "text", options, env)
			},
		},
		"bytes": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("bytes", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "bytes", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "bytes", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "bytes", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "bytes", options, env)
			},
		},
		// SVG file format - reads SVG files and strips XML prolog for use as components
		"SVG": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("SVG", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "svg", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "SVG", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "SVG", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "svg", options, env)
			},
		},
		// Markdown file format - reads MD files with frontmatter support
		"MD": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("MD", len(args), 1, 2)
				}

				// First argument must be a path dictionary, URL dictionary, or string
				var pathDict *Dictionary
				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					// Check if it's a URL dict first
					if isUrlDict(arg) {
						// Create request dictionary for URL
						return requestToDict(arg, "md", options, env)
					}
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "MD", "a path or URL", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "MD", "a path, URL, or string", args[0].Type())
				}

				return fileToDict(pathDict, "md", options, env)
			},
		},
		// Directory handle factory
		"dir": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 1 {
					return newArityError("dir", len(args), 1)
				}

				// First argument must be a path dictionary or string
				var pathDict *Dictionary
				env := NewEnvironment()

				switch arg := args[0].(type) {
				case *Dictionary:
					if !isPathDict(arg) {
						return newTypeError("TYPE-0012", "dir", "a path", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0012", "dir", "a path or string", args[0].Type())
				}

				return dirToDict(pathDict, env)
			},
		},
		// File pattern matching (glob patterns)
		"files": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 1 {
					return newArityError("files", len(args), 1)
				}

				var pattern string
				var env *Environment

				switch arg := args[0].(type) {
				case *Dictionary:
					if isPathDict(arg) {
						// Use the path dict's environment to preserve basil context
						if arg.Env != nil {
							env = arg.Env
						} else {
							env = NewEnvironment()
						}
						pattern = pathDictToString(arg)
					} else {
						return newTypeError("TYPE-0012", "files", "a path or string pattern", DICTIONARY_OBJ)
					}
				case *String:
					pattern = arg.Value
					env = NewEnvironment()
				default:
					return newTypeError("TYPE-0012", "files", "a path or string pattern", args[0].Type())
				}

				// Expand ~/ paths - in Parsley/Basil, ~/ means project root, not user home
				if strings.HasPrefix(pattern, "~/") {
					if env != nil && env.RootPath != "" {
						pattern = filepath.Join(env.RootPath, pattern[2:])
					} else {
						// Fallback to user home directory if no root path set
						home, err := os.UserHomeDir()
						if err == nil {
							pattern = filepath.Join(home, pattern[2:])
						}
					}
				}

				// Track if original pattern was explicitly relative (./ prefix)
				// Go's filepath.Glob strips this, so we need to restore it
				wasExplicitlyRelative := strings.HasPrefix(pattern, "./")

				// Use doublestar for ** glob patterns, fallback to filepath.Glob for simple patterns
				matches, err := filepath.Glob(pattern)
				if err != nil {
					return newError("invalid file pattern '%s': %s", pattern, err.Error())
				}

				// Convert matches to array of file handles
				elements := make([]Object, 0, len(matches))
				for _, match := range matches {
					info, statErr := os.Stat(match)
					if statErr != nil {
						continue
					}

					// Restore ./ prefix if the original pattern had it
					// filepath.Glob strips ./ but we want to preserve relative path semantics
					if wasExplicitlyRelative && !strings.HasPrefix(match, "./") && !strings.HasPrefix(match, "/") {
						match = "./" + match
					}

					components, isAbsolute := parsePathString(match)
					pathDict := pathToDict(components, isAbsolute, env)

					var fileHandle *Dictionary
					if info.IsDir() {
						fileHandle = dirToDict(pathDict, env)
					} else {
						format := inferFormatFromExtension(match)
						fileHandle = fileToDict(pathDict, format, nil, env)
					}
					elements = append(elements, fileHandle)
				}

				return &Array{Elements: elements}
			},
		},
		// Locale-aware formatting functions
		"formatNumber": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("formatNumber", len(args), 1, 2)
				}

				var value float64
				switch arg := args[0].(type) {
				case *Integer:
					value = float64(arg.Value)
				case *Float:
					value = arg.Value
				default:
					return newTypeError("TYPE-0005", "formatNumber", "an integer or float", args[0].Type())
				}

				locale := "en"
				if len(args) == 2 {
					locStr, ok := args[1].(*String)
					if !ok {
						return newTypeError("TYPE-0006", "formatNumber", "a string", args[1].Type())
					}
					locale = locStr.Value
				}

				tag, err := language.Parse(locale)
				if err != nil {
					return newLocaleError(locale)
				}

				p := message.NewPrinter(tag)
				return &String{Value: p.Sprintf("%v", number.Decimal(value))}
			},
		},
		"formatCurrency": {
			Fn: func(args ...Object) Object {
				if len(args) < 2 || len(args) > 3 {
					return newArityErrorRange("formatCurrency", len(args), 2, 3)
				}

				var value float64
				switch arg := args[0].(type) {
				case *Integer:
					value = float64(arg.Value)
				case *Float:
					value = arg.Value
				default:
					return newTypeError("TYPE-0005", "formatCurrency", "an integer or float", args[0].Type())
				}

				currStr, ok := args[1].(*String)
				if !ok {
					return newTypeError("TYPE-0006", "formatCurrency", "a string (currency code)", args[1].Type())
				}

				cur, err := currency.ParseISO(currStr.Value)
				if err != nil {
					return newError("invalid currency code: %s", currStr.Value)
				}

				locale := "en"
				if len(args) == 3 {
					locStr, ok := args[2].(*String)
					if !ok {
						return newTypeError("TYPE-0011", "formatCurrency", "a string", args[2].Type())
					}
					locale = locStr.Value
				}

				tag, err := language.Parse(locale)
				if err != nil {
					return newLocaleError(locale)
				}

				p := message.NewPrinter(tag)
				amount := cur.Amount(value)
				return &String{Value: p.Sprintf("%v", currency.Symbol(amount))}
			},
		},
		"formatPercent": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("formatPercent", len(args), 1, 2)
				}

				var value float64
				switch arg := args[0].(type) {
				case *Integer:
					value = float64(arg.Value)
				case *Float:
					value = arg.Value
				default:
					return newTypeError("TYPE-0005", "formatPercent", "an integer or float", args[0].Type())
				}

				locale := "en"
				if len(args) == 2 {
					locStr, ok := args[1].(*String)
					if !ok {
						return newTypeError("TYPE-0006", "formatPercent", "a string", args[1].Type())
					}
					locale = locStr.Value
				}

				tag, err := language.Parse(locale)
				if err != nil {
					return newLocaleError(locale)
				}

				p := message.NewPrinter(tag)
				return &String{Value: p.Sprintf("%v", number.Percent(value))}
			},
		},
		"formatDate": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 3 {
					return newArityErrorRange("formatDate", len(args), 1, 3)
				}

				// First argument must be a datetime dictionary
				dict, ok := args[0].(*Dictionary)
				if !ok || !isDatetimeDict(dict) {
					return newTypeError("TYPE-0005", "formatDate", "a datetime", args[0].Type())
				}

				// Extract time from datetime dictionary
				var t time.Time
				if unixExpr, ok := dict.Pairs["unix"]; ok {
					unixObj := Eval(unixExpr, NewEnvironment())
					if unixInt, ok := unixObj.(*Integer); ok {
						t = time.Unix(unixInt.Value, 0).UTC()
					}
				}

				// Default style and locale
				style := "long"
				locale := "en-US"

				if len(args) >= 2 {
					styleStr, ok := args[1].(*String)
					if !ok {
						return newTypeError("TYPE-0006", "formatDate", "a string", args[1].Type())
					}
					style = styleStr.Value
					// Validate style
					validStyles := map[string]bool{"short": true, "medium": true, "long": true, "full": true}
					if !validStyles[style] {
						return newError("style must be one of: short, medium, long, full, got %s", style)
					}
				}

				if len(args) == 3 {
					locStr, ok := args[2].(*String)
					if !ok {
						return newTypeError("TYPE-0011", "formatDate", "a string", args[2].Type())
					}
					locale = locStr.Value
				}

				// Map locale string to monday.Locale
				mondayLocale := getMondayLocale(locale)

				// Get format pattern for style
				format := getDateFormatForStyle(style, mondayLocale)

				return &String{Value: monday.Format(t, format, mondayLocale)}
			},
		},
		"format": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 3 {
					return newArityErrorRange("format", len(args), 1, 3)
				}

				// Handle arrays (list formatting)
				if arr, ok := args[0].(*Array); ok {
					// Convert array elements to strings
					items := make([]string, len(arr.Elements))
					for i, elem := range arr.Elements {
						// Use Inspect() for all types (String.Inspect() returns just the value)
						items[i] = elem.Inspect()
					}

					// Get style (default to "and")
					style := locale.ListStyleAnd
					localeStr := "en-US"

					if len(args) >= 2 {
						styleStr, ok := args[1].(*String)
						if !ok {
							return newTypeError("TYPE-0006", "format", "a string (style)", args[1].Type())
						}
						switch styleStr.Value {
						case "and":
							style = locale.ListStyleAnd
						case "or":
							style = locale.ListStyleOr
						case "unit":
							style = locale.ListStyleUnit
						default:
							return newError("invalid style %q for `format`, use 'and', 'or', or 'unit'", styleStr.Value)
						}
					}

					if len(args) == 3 {
						locStr, ok := args[2].(*String)
						if !ok {
							return newTypeError("TYPE-0011", "format", "a string (locale)", args[2].Type())
						}
						localeStr = locStr.Value
					}

					result := locale.FormatList(items, style, localeStr)
					return &String{Value: result}
				}

				// Handle duration dictionaries
				dict, ok := args[0].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0005", "format", "a duration or array", args[0].Type())
				}

				if !isDurationDict(dict) {
					return newTypeError("TYPE-0005", "format", "a duration", DICTIONARY_OBJ)
				}

				// Extract months and seconds from duration
				months, seconds, err := getDurationComponents(dict, NewEnvironment())
				if err != nil {
					return newFormatError("FMT-0009", err)
				}

				// Get locale (default to en-US)
				localeStr := "en-US"
				if len(args) == 2 {
					locStr, ok := args[1].(*String)
					if !ok {
						return newTypeError("TYPE-0006", "format", "a string", args[1].Type())
					}
					localeStr = locStr.Value
				}

				// Format the duration as relative time
				result := locale.DurationToRelativeTime(months, seconds, localeStr)
				return &String{Value: result}
			},
		},
		"map": {
			Fn: func(args ...Object) Object {
				if len(args) < 2 {
					return newArityErrorMin("map", len(args), 2)
				}

				fn, ok := args[0].(*Function)
				if !ok {
					return newTypeError("TYPE-0005", "map", "a function", args[0].Type())
				}

				// If second argument is an array, use it; otherwise create array from remaining args
				var arr *Array
				if a, ok := args[1].(*Array); ok && len(args) == 2 {
					arr = a
				} else {
					// Create array from all arguments after the function
					arr = &Array{Elements: args[1:]}
				}

				// Validate function parameter count
				if fn.ParamCount() != 1 {
					return newError("function passed to `map` must take exactly 1 parameter, got %d", fn.ParamCount())
				}

				result := []Object{}
				for _, elem := range arr.Elements {
					// Apply function to each element
					extendedEnv := extendFunctionEnv(fn, []Object{elem})

					// Evaluate the function body
					var evaluated Object
					for _, stmt := range fn.Body.Statements {
						evaluated = evalStatement(stmt, extendedEnv)
						if returnValue, ok := evaluated.(*ReturnValue); ok {
							evaluated = returnValue.Value
							break
						}
						if isError(evaluated) {
							return evaluated
						}
					}

					// Skip null values (filter behavior)
					if evaluated != NULL {
						result = append(result, evaluated)
					}
				}

				return &Array{Elements: result}
			},
		},
		"toUpper": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toUpper", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "toUpper", "a string", args[0].Type())
				}

				return &String{Value: strings.ToUpper(str.Value)}
			},
		},
		"toLower": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toLower", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "toLower", "a string", args[0].Type())
				}

				return &String{Value: strings.ToLower(str.Value)}
			},
		},
		"regex": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("regex", len(args), 1, 2)
				}

				pattern, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "regex", "a string", args[0].Type())
				}

				flags := ""
				if len(args) == 2 {
					flagsStr, ok := args[1].(*String)
					if !ok {
						return newTypeError("TYPE-0006", "regex", "a string", args[1].Type())
					}
					flags = flagsStr.Value
				}

				// Validate the regex
				_, err := compileRegex(pattern.Value, flags)
				if err != nil {
					return newFormatError("FMT-0002", err)
				}

				// Create regex dictionary
				pairs := make(map[string]ast.Expression)
				pairs["__type"] = &ast.StringLiteral{Value: "regex"}
				pairs["pattern"] = &ast.StringLiteral{Value: pattern.Value}
				pairs["flags"] = &ast.StringLiteral{Value: flags}

				return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
			},
		},
		"replace": {
			Fn: func(args ...Object) Object {
				if len(args) != 3 {
					return newArityError("replace", len(args), 3)
				}

				text, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "replace", "a string", args[0].Type())
				}

				// Second arg can be string or regex
				var pattern string
				var flags string
				if str, ok := args[1].(*String); ok {
					// String pattern - use literal replacement
					replacement, ok := args[2].(*String)
					if !ok {
						return newTypeError("TYPE-0011", "replace", "a string", args[2].Type())
					}
					return &String{Value: strings.Replace(text.Value, str.Value, replacement.Value, -1)}
				} else if dict, ok := args[1].(*Dictionary); ok && isRegexDict(dict) {
					// Regex pattern
					patternExpr, _ := dict.Pairs["pattern"]
					patternObj := Eval(patternExpr, NewEnvironment())
					patternStr := patternObj.(*String)
					pattern = patternStr.Value

					flagsExpr, ok := dict.Pairs["flags"]
					if ok {
						flagsObj := Eval(flagsExpr, NewEnvironment())
						if flagsStr, ok := flagsObj.(*String); ok {
							flags = flagsStr.Value
						}
					}
				} else {
					return newTypeError("TYPE-0006", "replace", "a string or regex", args[1].Type())
				}

				replacement, ok := args[2].(*String)
				if !ok {
					return newTypeError("TYPE-0011", "replace", "a string", args[2].Type())
				}

				re, err := compileRegex(pattern, flags)
				if err != nil {
					return newFormatError("FMT-0002", err)
				}

				result := re.ReplaceAllString(text.Value, replacement.Value)
				return &String{Value: result}
			},
		},
		"split": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("split", len(args), 2)
				}

				text, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "split", "a string", args[0].Type())
				}

				// Second arg can be string or regex
				var parts []string
				if str, ok := args[1].(*String); ok {
					// String delimiter
					parts = strings.Split(text.Value, str.Value)
				} else if dict, ok := args[1].(*Dictionary); ok && isRegexDict(dict) {
					// Regex pattern
					patternExpr, _ := dict.Pairs["pattern"]
					patternObj := Eval(patternExpr, NewEnvironment())
					patternStr := patternObj.(*String)
					pattern := patternStr.Value

					flags := ""
					flagsExpr, ok := dict.Pairs["flags"]
					if ok {
						flagsObj := Eval(flagsExpr, NewEnvironment())
						if flagsStr, ok := flagsObj.(*String); ok {
							flags = flagsStr.Value
						}
					}

					re, err := compileRegex(pattern, flags)
					if err != nil {
						return newFormatError("FMT-0002", err)
					}

					parts = re.Split(text.Value, -1)
				} else {
					return newTypeError("TYPE-0006", "split", "a string or regex", args[1].Type())
				}

				elements := make([]Object, len(parts))
				for i, part := range parts {
					elements[i] = &String{Value: part}
				}

				return &Array{Elements: elements}
			},
		},
		"tag": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 3 {
					return newArityErrorRange("tag", len(args), 1, 3)
				}

				// First arg: tag name (required)
				nameStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "tag", "a string (tag name)", args[0].Type())
				}

				// Create the tag dictionary
				pairs := make(map[string]ast.Expression)
				pairs["__type"] = createLiteralExpression(&String{Value: "tag"})
				pairs["name"] = createLiteralExpression(nameStr)

				// Second arg: attributes (optional dictionary)
				if len(args) >= 2 && args[1] != nil && args[1] != NULL {
					switch attrArg := args[1].(type) {
					case *Dictionary:
						// Copy attributes from the provided dictionary
						attrs := make(map[string]ast.Expression)
						for key, expr := range attrArg.Pairs {
							attrs[key] = expr
						}
						// Store as nested dictionary for attributes
						attrDict := &Dictionary{Pairs: attrs, Env: NewEnvironment()}
						pairs["attrs"] = createLiteralExpression(attrDict)
					case *Null:
						// No attributes, use empty dict
						pairs["attrs"] = createLiteralExpression(&Dictionary{Pairs: map[string]ast.Expression{}, Env: NewEnvironment()})
					default:
						return newTypeError("TYPE-0006", "tag", "a dictionary (attributes)", args[1].Type())
					}
				} else {
					pairs["attrs"] = createLiteralExpression(&Dictionary{Pairs: map[string]ast.Expression{}, Env: NewEnvironment()})
				}

				// Third arg: contents (optional string or array)
				if len(args) >= 3 && args[2] != nil && args[2] != NULL {
					switch contentArg := args[2].(type) {
					case *String:
						pairs["contents"] = createLiteralExpression(contentArg)
					case *Array:
						pairs["contents"] = createLiteralExpression(contentArg)
					case *Null:
						pairs["contents"] = createLiteralExpression(NULL)
					default:
						return newTypeError("TYPE-0011", "tag", "a string or array (contents)", args[2].Type())
					}
				} else {
					pairs["contents"] = createLiteralExpression(NULL)
				}

				return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
			},
		},
		"len": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("len", len(args), 1)
				}

				arg := args[0]

				// Handle response typed dictionary - unwrap __data for length
				if dict, ok := arg.(*Dictionary); ok && isResponseDict(dict) {
					if dataExpr, ok := dict.Pairs["__data"]; ok {
						arg = Eval(dataExpr, dict.Env)
					}
				}

				switch a := arg.(type) {
				case *String:
					return &Integer{Value: int64(len(a.Value))}
				case *Array:
					return &Integer{Value: int64(len(a.Elements))}
				default:
					return newTypeError("TYPE-0002", "len", "", args[0].Type())
				}
			},
		},
		// asset() - converts a path under public_dir to a web URL
		// e.g., asset(@./public/images/foo.png) -> "/images/foo.png"
		// Also accepts file dictionaries from files() and extracts their path
		"asset": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("asset", len(args), 1)
				}

				switch arg := args[0].(type) {
				case *Dictionary:
					if isPathDict(arg) {
						return &String{Value: pathToWebURL(arg)}
					}
					// Check if it's a file/dir dictionary - extract path and convert
					if isFileDict(arg) || isDirDict(arg) {
						// Convert file dict to path dict for pathToWebURL
						pathDict := fileDictToPathDict(arg)
						if pathDict != nil {
							return &String{Value: pathToWebURL(pathDict)}
						}
						return newError("could not extract path from file")
					}
					return newTypeError("TYPE-0012", "asset", "a path or file", DICTIONARY_OBJ)
				case *String:
					// If it's already a string, just return it
					return arg
				default:
					return newTypeError("TYPE-0012", "asset", "a path or file", args[0].Type())
				}
			},
		},
		"repr": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("repr", len(args), 1)
				}

				// Return the debug/dictionary representation of any value
				// For dictionaries (including pseudo-types), returns the dict's Inspect()
				// For other types, returns their string representation
				arg := args[0]
				if arg == nil {
					return &String{Value: "null"}
				}

				switch obj := arg.(type) {
				case *Dictionary:
					// For all dictionaries (including pseudo-types), return the raw dict representation
					return &String{Value: obj.Inspect()}
				case *Array:
					return &String{Value: obj.Inspect()}
				case *String:
					// For strings, include quotes in repr
					return &String{Value: "\"" + obj.Value + "\""}
				case *Integer:
					return &String{Value: obj.Inspect()}
				case *Float:
					return &String{Value: obj.Inspect()}
				case *Boolean:
					return &String{Value: obj.Inspect()}
				case *Null:
					return &String{Value: "null"}
				case *Function:
					return &String{Value: obj.Inspect()}
				case *Error:
					return &String{Value: "error: " + obj.Message}
				default:
					return &String{Value: obj.Inspect()}
				}
			},
		},
		"toInt": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toInt", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "toInt", "a string", args[0].Type())
				}

				var val int64
				_, err := fmt.Sscanf(str.Value, "%d", &val)
				if err != nil {
					return newConversionError("TYPE-0015", str.Value)
				}

				return &Integer{Value: val}
			},
		},
		"toFloat": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toFloat", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "toFloat", "a string", args[0].Type())
				}

				var val float64
				_, err := fmt.Sscanf(str.Value, "%f", &val)
				if err != nil {
					return newConversionError("TYPE-0016", str.Value)
				}

				return &Float{Value: val}
			},
		},
		"toNumber": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toNumber", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "toNumber", "a string", args[0].Type())
				}

				// Try to parse as integer first
				var intVal int64
				if _, err := fmt.Sscanf(str.Value, "%d", &intVal); err == nil {
					// Check if the string has a decimal point - if so, it's a float
					if !strings.Contains(str.Value, ".") {
						return &Integer{Value: intVal}
					}
				}

				// Parse as float
				var floatVal float64
				if _, err := fmt.Sscanf(str.Value, "%f", &floatVal); err == nil {
					return &Float{Value: floatVal}
				}

				return newConversionError("TYPE-0017", str.Value)
			},
		},
		"toString": {
			Fn: func(args ...Object) Object {
				var result strings.Builder

				for _, arg := range args {
					result.WriteString(objectToPrintString(arg))
				}

				return &String{Value: result.String()}
			},
		},
		"toDebug": {
			Fn: func(args ...Object) Object {
				var result strings.Builder

				for i, arg := range args {
					if i > 0 {
						result.WriteString(", ")
					}
					result.WriteString(objectToDebugString(arg))
				}

				return &String{Value: result.String()}
			},
		},
		"log": {
			Fn: func(args ...Object) Object {
				var result strings.Builder

				for i, arg := range args {
					if i == 0 {
						// First argument: if it's a string, show without quotes
						if str, ok := arg.(*String); ok {
							result.WriteString(str.Value)
						} else {
							result.WriteString(objectToDebugString(arg))
						}
					} else {
						// Subsequent arguments: add separator and debug format
						if i == 1 {
							// After first string, no comma - just space
							if _, firstWasString := args[0].(*String); firstWasString {
								result.WriteString(" ")
							} else {
								result.WriteString(", ")
							}
						} else {
							result.WriteString(", ")
						}
						result.WriteString(objectToDebugString(arg))
					}
				}

				// Write immediately to stdout
				fmt.Fprintln(os.Stdout, result.String())

				// Return null
				return NULL
			},
		},
		"logLine": {
			Fn: func(args ...Object) Object {
				// This is a placeholder - will be replaced with actual implementation
				// that has access to environment
				return NULL
			},
		},
		"sort": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("sort", len(args), 1)
				}

				arr, ok := args[0].(*Array)
				if !ok {
					return newTypeError("TYPE-0012", "sort", "an array", args[0].Type())
				}

				// Create a copy to avoid modifying the original
				sortedElements := make([]Object, len(arr.Elements))
				copy(sortedElements, arr.Elements)

				// Sort using natural sort comparison
				sort.Slice(sortedElements, func(i, j int) bool {
					return naturalCompare(sortedElements[i], sortedElements[j])
				})

				return &Array{Elements: sortedElements}
			},
		},
		"reverse": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("reverse", len(args), 1)
				}

				arr, ok := args[0].(*Array)
				if !ok {
					return newTypeError("TYPE-0012", "reverse", "an array", args[0].Type())
				}

				// Create a reversed copy
				reversed := make([]Object, len(arr.Elements))
				for i, elem := range arr.Elements {
					reversed[len(arr.Elements)-1-i] = elem
				}

				return &Array{Elements: reversed}
			},
		},
		"sortBy": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("sortBy", len(args), 2)
				}

				arr, ok := args[0].(*Array)
				if !ok {
					return newTypeError("TYPE-0005", "sortBy", "an array", args[0].Type())
				}

				compareFn := args[1]

				// Verify it's a function
				fn, ok := compareFn.(*Function)
				if !ok {
					return newTypeError("TYPE-0006", "sortBy", "a function", compareFn.Type())
				}

				// Verify the function takes exactly 2 parameters
				if fn.ParamCount() != 2 {
					return newError("comparison function must take exactly 2 parameters, got %d", fn.ParamCount())
				}

				// Create a copy to avoid modifying the original
				sortedElements := make([]Object, len(arr.Elements))
				copy(sortedElements, arr.Elements)

				// Sort using the custom comparison function
				sort.Slice(sortedElements, func(i, j int) bool {
					// Call the comparison function with the two elements
					result := applyFunction(fn, []Object{sortedElements[i], sortedElements[j]})

					// The function should return a 2-element array
					resultArr, ok := result.(*Array)
					if !ok || len(resultArr.Elements) != 2 {
						return false
					}

					// Check if the first element equals sortedElements[i]
					// If so, it means i comes before j (ascending order)
					return objectsEqual(resultArr.Elements[0], sortedElements[i])
				})

				return &Array{Elements: sortedElements}
			},
		},
		"keys": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("keys", len(args), 1)
				}

				dict, ok := args[0].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0012", "keys", "a dictionary", args[0].Type())
				}

				keys := make([]Object, 0, len(dict.Pairs))
				for key := range dict.Pairs {
					keys = append(keys, &String{Value: key})
				}
				return &Array{Elements: keys}
			},
		},
		"values": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("values", len(args), 1)
				}

				dict, ok := args[0].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0012", "values", "a dictionary", args[0].Type())
				}

				// Create environment for evaluation with 'this'
				dictEnv := NewEnclosedEnvironment(dict.Env)
				dictEnv.Set("this", dict)

				values := make([]Object, 0, len(dict.Pairs))
				for _, expr := range dict.Pairs {
					val := Eval(expr, dictEnv)
					values = append(values, val)
				}
				return &Array{Elements: values}
			},
		},
		"has": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("has", len(args), 2)
				}

				dict, ok := args[0].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0005", "has", "a dictionary", args[0].Type())
				}

				key, ok := args[1].(*String)
				if !ok {
					return newTypeError("TYPE-0006", "has", "a string", args[1].Type())
				}

				_, exists := dict.Pairs[key.Value]
				return nativeBoolToParsBoolean(exists)
			},
		},
		"toArray": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toArray", len(args), 1)
				}

				dict, ok := args[0].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0012", "toArray", "a dictionary", args[0].Type())
				}

				// Create environment for evaluation with 'this'
				dictEnv := NewEnclosedEnvironment(dict.Env)
				dictEnv.Set("this", dict)

				pairs := make([]Object, 0, len(dict.Pairs))
				for key, expr := range dict.Pairs {
					val := Eval(expr, dictEnv)

					// Skip functions with parameters (they can't be called without args)
					if fn, ok := val.(*Function); ok && fn.ParamCount() > 0 {
						continue
					}

					// If it's a function with no parameters, call it
					if fn, ok := val.(*Function); ok && fn.ParamCount() == 0 {
						val = applyFunction(fn, []Object{})
					}

					// Create [key, value] pair
					pair := &Array{Elements: []Object{
						&String{Value: key},
						val,
					}}
					pairs = append(pairs, pair)
				}
				return &Array{Elements: pairs}
			},
		},
		"toDict": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("toDict", len(args), 1)
				}

				arr, ok := args[0].(*Array)
				if !ok {
					return newTypeError("TYPE-0012", "toDict", "an array", args[0].Type())
				}

				dict := &Dictionary{
					Pairs: make(map[string]ast.Expression),
					Env:   NewEnvironment(),
				}

				for _, elem := range arr.Elements {
					pair, ok := elem.(*Array)
					if !ok || len(pair.Elements) != 2 {
						return newError("toDict requires array of [key, value] pairs")
					}

					keyObj, ok := pair.Elements[0].(*String)
					if !ok {
						return newError("dictionary keys must be strings, got %s", pair.Elements[0].Type())
					}

					// Create a literal expression from the value
					valueObj := pair.Elements[1]
					var expr ast.Expression

					switch v := valueObj.(type) {
					case *Integer:
						expr = &ast.IntegerLiteral{Value: v.Value}
					case *Float:
						expr = &ast.FloatLiteral{Value: v.Value}
					case *String:
						expr = &ast.StringLiteral{Value: v.Value}
					case *Boolean:
						expr = &ast.Boolean{Value: v.Value}
					case *Array:
						// For arrays, we'll store a reference that evaluates to the array
						// This is a workaround - store in environment and reference it
						tempKey := "__toDict_temp_" + keyObj.Value
						dict.Env.Set(tempKey, v)
						expr = &ast.Identifier{Value: tempKey}
					default:
						return newError("toDict: unsupported value type %s", valueObj.Type())
					}

					dict.Pairs[keyObj.Value] = expr
				}

				return dict
			},
		},
		"COMMAND": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 3 {
					return newArityErrorRange("COMMAND", len(args), 1, 3)
				}

				env := NewEnvironment()

				// First argument: binary name/path (string)
				binary, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "COMMAND", "a string", args[0].Type())
				}

				// Second argument (optional): args array
				var cmdArgs []string
				if len(args) >= 2 {
					if argsArray, ok := args[1].(*Array); ok {
						cmdArgs = make([]string, len(argsArray.Elements))
						for i, arg := range argsArray.Elements {
							if str, ok := arg.(*String); ok {
								cmdArgs[i] = str.Value
							} else {
								return newError("COMMAND arguments must be strings, got %s at index %d", arg.Type(), i)
							}
						}
					} else {
						return newTypeError("TYPE-0006", "COMMAND", "an array", args[1].Type())
					}
				}

				// Third argument (optional): options dict
				var options *Dictionary
				if len(args) >= 3 {
					if optDict, ok := args[2].(*Dictionary); ok {
						options = optDict
					} else {
						return newTypeError("TYPE-0011", "COMMAND", "a dictionary", args[2].Type())
					}
				}

				return createCommandHandle(binary.Value, cmdArgs, options, env)
			},
		},
		"parseJSON": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("parseJSON", len(args), 1)
				}
				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "parseJSON", "a string", args[0].Type())
				}

				var result interface{}
				if err := json.Unmarshal([]byte(str.Value), &result); err != nil {
					return newFormatError("FMT-0005", err)
				}

				return jsonToObject(result)
			},
		},
		"stringifyJSON": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("stringifyJSON", len(args), 1)
				}

				jsonData := objectToGo(args[0])
				jsonBytes, err := json.Marshal(jsonData)
				if err != nil {
					return newFormatError("FMT-0005", err)
				}

				return &String{Value: string(jsonBytes)}
			},
		},
		"parseCSV": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("parseCSV", len(args), 1, 2)
				}
				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "parseCSV", "a string", args[0].Type())
				}

				// Parse options if provided (default: header=true)
				hasHeader := true
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						if headerExpr, exists := optDict.Pairs["header"]; exists {
							headerObj := Eval(headerExpr, optDict.Env)
							if headerBool, ok := headerObj.(*Boolean); ok {
								hasHeader = headerBool.Value
							}
						}
					}
				}

				reader := csv.NewReader(strings.NewReader(str.Value))
				records, err := reader.ReadAll()
				if err != nil {
					return newFormatError("FMT-0007", err)
				}

				if hasHeader && len(records) > 0 {
					// Return array of dicts with headers as keys
					headers := records[0]
					rows := make([]Object, len(records)-1)
					for i, record := range records[1:] {
						dict := &Dictionary{
							Pairs: make(map[string]ast.Expression),
							Env:   NewEnvironment(),
						}
						for j, value := range record {
							if j < len(headers) {
								dict.Pairs[headers[j]] = &ast.ObjectLiteralExpression{Obj: parseCSVValue(value)}
							}
						}
						rows[i] = dict
					}
					return &Array{Elements: rows}
				}

				// Return array of arrays
				rows := make([]Object, len(records))
				for i, record := range records {
					row := make([]Object, len(record))
					for j, value := range record {
						row[j] = parseCSVValue(value)
					}
					rows[i] = &Array{Elements: row}
				}
				return &Array{Elements: rows}
			},
		},
		"stringifyCSV": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("stringifyCSV", len(args), 1)
				}

				arr, ok := args[0].(*Array)
				if !ok {
					return newTypeError("TYPE-0012", "stringifyCSV", "an array", args[0].Type())
				}

				var buf bytes.Buffer
				writer := csv.NewWriter(&buf)

				for _, elem := range arr.Elements {
					if row, ok := elem.(*Array); ok {
						record := make([]string, len(row.Elements))
						for i, cell := range row.Elements {
							record[i] = cell.Inspect()
						}
						if err := writer.Write(record); err != nil {
							return newFormatError("FMT-0007", err)
						}
					} else {
						return newTypeError("TYPE-0012", "stringifyCSV", "an array of arrays", elem.Type())
					}
				}

				writer.Flush()
				if err := writer.Error(); err != nil {
					return newFormatError("FMT-0007", err)
				}

				return &String{Value: buf.String()}
			},
		},
	}
}

// createCommandHandle creates a command handle dictionary
func createCommandHandle(binary string, args []string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Add __type field
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "command"},
		Value: "command",
	}

	// Add binary field
	pairs["binary"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: binary},
		Value: binary,
	}

	// Add args field
	argElements := make([]ast.Expression, len(args))
	for i, arg := range args {
		argElements[i] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: arg},
			Value: arg,
		}
	}
	pairs["args"] = &ast.ArrayLiteral{
		Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
		Elements: argElements,
	}

	// Add options field
	if options != nil {
		// Copy options to ast expressions
		optPairs := make(map[string]ast.Expression)
		for k, v := range options.Pairs {
			optPairs[k] = v
		}
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: optPairs,
		}
	} else {
		// Empty options
		pairs["options"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// isCommandHandle checks if a dictionary is a command handle
func isCommandHandle(dict *Dictionary) bool {
	typeExpr, ok := dict.Pairs["__type"]
	if !ok {
		return false
	}
	typeLit, ok := typeExpr.(*ast.StringLiteral)
	if !ok {
		return false
	}
	return typeLit.Value == "command"
}

// executeCommand executes a command handle with input and returns result dictionary
func executeCommand(cmdDict *Dictionary, input Object, env *Environment) Object {
	// Extract binary
	binaryExpr, ok := cmdDict.Pairs["binary"]
	if !ok {
		return newError("command handle missing binary field")
	}
	binaryLit, ok := binaryExpr.(*ast.StringLiteral)
	if !ok {
		return newError("command binary must be a string")
	}
	binary := binaryLit.Value

	// Resolve command path
	var resolvedPath string
	if strings.Contains(binary, "/") {
		// Relative or absolute path
		resolvedPath = binary
	} else {
		// Look in PATH
		path, err := exec.LookPath(binary)
		if err != nil {
			return createErrorResult("command not found: "+binary, -1)
		}
		resolvedPath = path
	}

	// Security check
	if env.Security != nil {
		if err := env.checkPathAccess(resolvedPath, "execute"); err != nil {
			return createErrorResult("security: "+err.Error(), -1)
		}
	}

	// Extract args
	argsExpr, ok := cmdDict.Pairs["args"]
	if !ok {
		return newError("command handle missing args field")
	}
	argsLit, ok := argsExpr.(*ast.ArrayLiteral)
	if !ok {
		return newError("command args must be an array")
	}

	args := make([]string, len(argsLit.Elements))
	for i, argExpr := range argsLit.Elements {
		argLit, ok := argExpr.(*ast.StringLiteral)
		if !ok {
			return newError("command arguments must be strings")
		}
		args[i] = argLit.Value
	}

	// Extract options
	optsExpr, ok := cmdDict.Pairs["options"]
	if !ok {
		return newError("command handle missing options field")
	}
	optsLit, ok := optsExpr.(*ast.DictionaryLiteral)
	if !ok {
		return newError("command options must be a dictionary")
	}

	// Build exec.Command
	cmd := exec.Command(resolvedPath, args...)

	// Apply options
	applyCommandOptions(cmd, optsLit, env)

	// Set stdin if provided
	if input != nil && input.Type() != NULL_OBJ {
		if str, ok := input.(*String); ok {
			cmd.Stdin = strings.NewReader(str.Value)
		} else {
			return newError("command input must be a string or null, got %s", input.Type())
		}
	}

	// Execute and capture
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Build result dict
	return createResultDict(stdout.String(), stderr.String(), err)
}

// applyCommandOptions applies options to the exec.Cmd
func applyCommandOptions(cmd *exec.Cmd, optsLit *ast.DictionaryLiteral, env *Environment) {
	// env option
	if envExpr, ok := optsLit.Pairs["env"]; ok {
		envObj := Eval(envExpr, env)
		if envDict, ok := envObj.(*Dictionary); ok {
			var envVars []string
			for key, valExpr := range envDict.Pairs {
				valObj := Eval(valExpr, env)
				if str, ok := valObj.(*String); ok {
					envVars = append(envVars, key+"="+str.Value)
				}
			}
			cmd.Env = envVars
		}
	}

	// dir option
	if dirExpr, ok := optsLit.Pairs["dir"]; ok {
		dirObj := Eval(dirExpr, env)
		if pathDict, ok := dirObj.(*Dictionary); ok {
			if isPathDict(pathDict) {
				pathStr := pathDictToString(pathDict)
				cmd.Dir = pathStr
			}
		}
	}

	// timeout option
	if timeoutExpr, ok := optsLit.Pairs["timeout"]; ok {
		timeoutObj := Eval(timeoutExpr, env)
		if durDict, ok := timeoutObj.(*Dictionary); ok {
			if isDurationDict(durDict) {
				_, seconds, err := getDurationComponents(durDict, env)
				if err == nil {
					timeout := time.Duration(seconds) * time.Second
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					// Replace cmd with CommandContext
					*cmd = *exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
				}
			}
		}
	}
}

// createResultDict creates a result dictionary from command output
func createResultDict(stdout, stderr string, err error) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// stdout
	pairs["stdout"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stdout},
		Value: stdout,
	}

	// stderr
	pairs["stderr"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: stderr},
		Value: stderr,
	}

	// exitCode and error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Non-zero exit
			pairs["exitCode"] = &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.Itoa(exitErr.ExitCode())},
				Value: int64(exitErr.ExitCode()),
			}
			pairs["error"] = &ast.Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "null"}, Value: "null"}
		} else {
			// Execution failed
			pairs["exitCode"] = &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: "-1"},
				Value: -1,
			}
			pairs["error"] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: err.Error()},
				Value: err.Error(),
			}
		}
	} else {
		// Success
		pairs["exitCode"] = &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: "0"},
			Value: 0,
		}
		pairs["error"] = &ast.Identifier{Token: lexer.Token{Type: lexer.IDENT, Literal: "null"}, Value: "null"}
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// createErrorResult creates a result dictionary for errors
func createErrorResult(errMsg string, exitCode int64) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["stdout"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: ""},
		Value: "",
	}
	pairs["stderr"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: ""},
		Value: "",
	}
	pairs["exitCode"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(exitCode, 10)},
		Value: exitCode,
	}
	pairs["error"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: errMsg},
		Value: errMsg,
	}

	return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
}

// Helper function to evaluate a statement
func evalStatement(stmt ast.Statement, env *Environment) Object {
	switch stmt := stmt.(type) {
	case *ast.ExpressionStatement:
		return Eval(stmt.Expression, env)
	case *ast.ReturnStatement:
		val := Eval(stmt.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}
	default:
		return Eval(stmt, env)
	}
}

// evalDBConnectionMethod handles method calls on database connections
func evalDBConnectionMethod(conn *DBConnection, method string, args []Object, env *Environment) Object {
	switch method {
	case "begin":
		if len(args) != 0 {
			return newArityError("begin", len(args), 0)
		}
		if conn.InTransaction {
			return newDatabaseStateError("DB-0007")
		}
		conn.InTransaction = true
		return &Boolean{Value: true}

	case "commit":
		if len(args) != 0 {
			return newArityError("commit", len(args), 0)
		}
		if !conn.InTransaction {
			return newDatabaseStateError("DB-0006")
		}
		// For now, just mark transaction as complete
		// Real transaction support will be added with actual query execution
		conn.InTransaction = false
		return &Boolean{Value: true}

	case "rollback":
		if len(args) != 0 {
			return newArityError("rollback", len(args), 0)
		}
		if !conn.InTransaction {
			return newDatabaseStateError("DB-0006")
		}
		conn.InTransaction = false
		return &Boolean{Value: true}

	case "close":
		if len(args) != 0 {
			return newArityError("close", len(args), 0)
		}
		// Managed connections cannot be closed by Parsley scripts
		if conn.Managed {
			return newError("cannot close server-managed database connection")
		}
		// Remove from cache and close
		cacheKey := conn.Driver + ":" + conn.DSN
		dbConnectionsMu.Lock()
		delete(dbConnections, cacheKey)
		dbConnectionsMu.Unlock()

		if err := conn.DB.Close(); err != nil {
			conn.LastError = err.Error()
			return newError("failed to close connection: %s", err.Error())
		}
		return NULL

	case "ping":
		if len(args) != 0 {
			return newArityError("ping", len(args), 0)
		}
		if err := conn.DB.Ping(); err != nil {
			conn.LastError = err.Error()
			return &Boolean{Value: false}
		}
		return &Boolean{Value: true}

	default:
		return newUndefinedMethodError(method, "database connection")
	}
}

// evalSFTPConnectionMethod handles method calls on SFTP connections
func evalSFTPConnectionMethod(conn *SFTPConnection, method string, args []Object, env *Environment) Object {
	switch method {
	case "close":
		if len(args) != 0 {
			return newArityError("close", len(args), 0)
		}

		// Remove from cache
		cacheKey := fmt.Sprintf("sftp:%s@%s:%d", conn.User, conn.Host, conn.Port)
		sftpConnectionsMu.Lock()
		delete(sftpConnections, cacheKey)
		sftpConnectionsMu.Unlock()

		// Close SFTP and SSH clients
		if conn.Client != nil {
			conn.Client.Close()
		}
		if conn.SSHClient != nil {
			conn.SSHClient.Close()
		}
		conn.Connected = false
		return NULL

	default:
		return newUndefinedMethodError(method, "SFTP connection")
	}
}

// evalSFTPFileHandleMethod handles method calls on SFTP file handles
func evalSFTPFileHandleMethod(handle *SFTPFileHandle, method string, args []Object, env *Environment) Object {
	switch method {
	case "mkdir":
		// Create directory
		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if parentsExpr, ok := optDict.Pairs["parents"]; ok {
					if parentsVal := Eval(parentsExpr, optDict.Env); parentsVal != nil {
						if boolVal, ok := parentsVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		var err error
		if recursive {
			err = handle.Connection.Client.MkdirAll(handle.Path)
		} else {
			err = handle.Connection.Client.Mkdir(handle.Path)
		}

		if err != nil {
			return newError("failed to create directory: %s", err.Error())
		}
		return NULL

	case "rmdir":
		// Remove directory
		var recursive bool
		if len(args) > 0 {
			if optDict, ok := args[0].(*Dictionary); ok {
				if recursiveExpr, ok := optDict.Pairs["recursive"]; ok {
					if recursiveVal := Eval(recursiveExpr, optDict.Env); recursiveVal != nil {
						if boolVal, ok := recursiveVal.(*Boolean); ok {
							recursive = boolVal.Value
						}
					}
				}
			}
		}

		var err error
		if recursive {
			// Recursively remove directory and contents
			err = handle.Connection.Client.RemoveDirectory(handle.Path)
		} else {
			// Remove empty directory only
			err = handle.Connection.Client.RemoveDirectory(handle.Path)
		}

		if err != nil {
			return newError("failed to remove directory: %s", err.Error())
		}
		return NULL

	case "remove":
		// Remove file
		if len(args) != 0 {
			return newArityError("remove", len(args), 0)
		}

		if err := handle.Connection.Client.Remove(handle.Path); err != nil {
			return newIOError("IO-0005", handle.Path, err)
		}
		return NULL

	default:
		return newUndefinedMethodError(method, "SFTP file handle")
	}
}

// Eval evaluates AST nodes and returns objects
func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(node.Statements, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}

		// Handle dictionary destructuring
		if node.DictPattern != nil {
			return evalDictDestructuringAssignment(node.DictPattern, val, env, true, node.Export)
		}

		// Handle array destructuring assignment
		if len(node.Names) > 0 {
			return evalDestructuringAssignment(node.Names, val, env, true, node.Export)
		}

		// Single assignment
		// Special handling for '_' - don't store it
		if node.Name.Value != "_" {
			if node.Export {
				env.SetLetExport(node.Name.Value, val)
			} else {
				env.SetLet(node.Name.Value, val)
			}
		}
		// Declarations return NULL (excluded from block concatenation)
		return NULL

	case *ast.AssignmentStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}

		// Handle dictionary destructuring
		if node.DictPattern != nil {
			return evalDictDestructuringAssignment(node.DictPattern, val, env, false, node.Export)
		}

		// Handle array destructuring assignment
		if len(node.Names) > 0 {
			return evalDestructuringAssignment(node.Names, val, env, false, node.Export)
		}

		// Single assignment
		// Special handling for '_' - don't store it
		if node.Name.Value != "_" {
			if node.Export {
				env.SetExport(node.Name.Value, val)
			} else {
				env.Update(node.Name.Value, val)
			}
		}
		// Assignments return NULL (excluded from block concatenation)
		return NULL

	case *ast.ReadStatement:
		return evalReadStatement(node, env)

	case *ast.FetchStatement:
		return evalFetchStatement(node, env)

	case *ast.WriteStatement:
		return evalWriteStatement(node, env)

	case *ast.QueryOneStatement:
		return evalQueryOneStatement(node, env)

	case *ast.QueryManyStatement:
		return evalQueryManyStatement(node, env)

	case *ast.ExecuteStatement:
		return evalExecuteStatement(node, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &ReturnValue{Value: val}

	// Expressions
	case *ast.IntegerLiteral:
		return &Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &Float{Value: node.Value}

	case *ast.StringLiteral:
		return &String{Value: node.Value}

	case *ast.TemplateLiteral:
		return evalTemplateLiteral(node, env)

	case *ast.RegexLiteral:
		return evalRegexLiteral(node, env)

	case *ast.DatetimeLiteral:
		return evalDatetimeLiteral(node, env)

	case *ast.DurationLiteral:
		return evalDurationLiteral(node, env)

	case *ast.PathLiteral:
		return evalPathLiteral(node, env)

	case *ast.UrlLiteral:
		return evalUrlLiteral(node, env)

	case *ast.StdlibPathLiteral:
		// Standard library paths evaluate to a simple string (e.g., "std/table")
		return &String{Value: node.Value}

	case *ast.PathTemplateLiteral:
		return evalPathTemplateLiteral(node, env)

	case *ast.UrlTemplateLiteral:
		return evalUrlTemplateLiteral(node, env)

	case *ast.DatetimeTemplateLiteral:
		return evalDatetimeTemplateLiteral(node, env)

	case *ast.TagLiteral:
		return evalTagLiteral(node, env)

	case *ast.TagPairExpression:
		return evalTagPair(node, env)

	case *ast.TextNode:
		return &String{Value: node.Value}

	case *ast.InterpolationBlock:
		return evalInterpolationBlock(node, env)

	case *ast.Boolean:
		return nativeBoolToParsBoolean(node.Value)

	case *ast.ObjectLiteralExpression:
		return node.Obj.(Object)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Token, node.Operator, right)

	case *ast.InfixExpression:
		// Special handling for database operators
		if node.Operator == "<=?=>" || node.Operator == "<=??=>" || node.Operator == "<=!=>" {
			connection := Eval(node.Left, env)
			if isError(connection) {
				return connection
			}
			query := Eval(node.Right, env)
			if isError(query) {
				return query
			}

			switch node.Operator {
			case "<=?=>":
				return evalDatabaseQueryOne(connection, query, env)
			case "<=??=>":
				return evalDatabaseQueryMany(connection, query, env)
			case "<=!=>":
				return evalDatabaseExecute(connection, query, env)
			}
		}

		// Special handling for nullish coalescing operator (??)
		// It's short-circuit: only evaluate right if left is NULL
		if node.Operator == "??" {
			left := Eval(node.Left, env)
			if isError(left) {
				return left
			}
			// If left is not NULL, return it (short-circuit)
			if left != NULL {
				return left
			}
			// Left is NULL, evaluate and return right
			return Eval(node.Right, env)
		}

		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Token, node.Operator, left, right)

	case *ast.ExecuteExpression:
		// Evaluate command handle
		cmdObj := Eval(node.Command, env)
		if isError(cmdObj) {
			return cmdObj
		}

		// Verify it's a command handle
		cmdDict, ok := cmdObj.(*Dictionary)
		if !ok {
			return newError("left operand of <=#=> must be command handle, got %s", cmdObj.Type())
		}

		if !isCommandHandle(cmdDict) {
			return newError("left operand of <=#=> must be command handle")
		}

		// Evaluate input
		inputObj := Eval(node.Input, env)
		if isError(inputObj) {
			return inputObj
		}

		// Execute the command
		return executeCommand(cmdDict, inputObj, env)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		body := node.Body
		// Use new-style params
		return &Function{Params: node.Params, Env: env, Body: body}

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &Array{Elements: elements}

	case *ast.DictionaryLiteral:
		return evalDictionaryLiteral(node, env)

	case *ast.DotExpression:
		return evalDotExpression(node, env)

	case *ast.CallExpression:
		// Store current token in environment for logLine
		env.LastToken = &node.Token

		// Check if this is a call to import
		if ident, ok := node.Function.(*ast.Identifier); ok && ident.Value == "import" {
			args := evalExpressions(node.Arguments, env)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}
			return evalImport(args, env)
		}

		// Check if this is a call to log (needs env for Logger)
		if ident, ok := node.Function.(*ast.Identifier); ok && ident.Value == "log" {
			args := evalExpressions(node.Arguments, env)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}
			return evalLog(args, env)
		}

		// Check if this is a call to logLine
		if ident, ok := node.Function.(*ast.Identifier); ok && ident.Value == "logLine" {
			args := evalExpressions(node.Arguments, env)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}
			return evalLogLine(args, env)
		}

		// Check if this is a method call (DotExpression as function)
		if dotExpr, ok := node.Function.(*ast.DotExpression); ok {
			left := Eval(dotExpr.Left, env)
			if isError(left) {
				return left
			}

			// Null propagation: method calls on null return null
			if left == NULL || left == nil {
				return NULL
			}

			// Evaluate arguments
			args := evalExpressions(node.Arguments, env)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}

			method := dotExpr.Key

			// Dispatch based on receiver type
			switch receiver := left.(type) {
			case *DevModule:
				return evalDevModuleMethod(receiver, method, args, env)
			case *TableModule:
				return evalTableModuleMethod(receiver, method, args, env)
			case *Table:
				return EvalTableMethod(receiver, method, args, env)
			case *DBConnection:
				return evalDBConnectionMethod(receiver, method, args, env)
			case *SFTPConnection:
				return evalSFTPConnectionMethod(receiver, method, args, env)
			case *SFTPFileHandle:
				return evalSFTPFileHandleMethod(receiver, method, args, env)
			case *String:
				return evalStringMethod(receiver, method, args)
			case *Array:
				return evalArrayMethod(receiver, method, args, env)
			case *Integer:
				return evalIntegerMethod(receiver, method, args)
			case *Float:
				return evalFloatMethod(receiver, method, args)
			case *Dictionary:
				// Check for special dictionary types first
				if isDatetimeDict(receiver) {
					result := evalDatetimeMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// Fall through to check dictionary methods if datetime method failed
					if result != nil && isError(result) {
						// Check if it's "unknown method" error - try dictionary method
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							// Try dictionary methods
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isDurationDict(receiver) {
					result := evalDurationMethod(receiver, method, args, env)
					if result != nil {
						return result
					}
				}
				if isPathDict(receiver) {
					result := evalPathMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isUrlDict(receiver) {
					result := evalUrlMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isRegexDict(receiver) {
					result := evalRegexMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isFileDict(receiver) {
					result := evalFileMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isDirDict(receiver) {
					result := evalDirMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isRequestDict(receiver) {
					result := evalRequestMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				if isResponseDict(receiver) {
					result := evalResponseMethod(receiver, method, args, env)
					if result != nil && !isError(result) {
						return result
					}
					// If unknown method, fall through to dictionary methods
					if result != nil && isError(result) {
						if errObj, ok := result.(*Error); ok && errObj.Code == "UNDEF-0002" {
							dictResult := evalDictionaryMethod(receiver, method, args, env)
							if dictResult != nil {
								return dictResult
							}
						}
						return result
					}
				}
				// Regular dictionary methods (keys, values, has)
				result := evalDictionaryMethod(receiver, method, args, env)
				if result != nil {
					return result
				}
				// Check if the dictionary has a user-defined function at this key
				if fnExpr, ok := receiver.Pairs[method]; ok {
					fnObj := Eval(fnExpr, receiver.Env)
					if fn, ok := fnObj.(*Function); ok {
						// Call the function with 'this' bound to the dictionary
						return applyMethodWithThis(fn, args, receiver)
					}
					// If it's not a function, return error
					if !isError(fnObj) {
						return newErrorWithPos(node.Token, "'%s' is not a function", method)
					}
				}
				// Fall through to normal property/function evaluation
			}
		}

		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		// Better error for calling null as a function
		if function == NULL || function == nil {
			funcName := node.Function.String()
			// Check if this looks like it came from an import destructuring
			if ident, ok := node.Function.(*ast.Identifier); ok {
				return newError("cannot call '%s' because it is null\n   💡 Hint: '%s' may not be exported from the imported module. Check the export name matches.", ident.Value, ident.Value)
			}
			return newError("cannot call null as a function: %s", funcName)
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunctionWithEnv(function, args, env)

	case *ast.ForExpression:
		return evalForExpression(node, env)

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(node.Token, left, index, node.Optional)

	case *ast.SliceExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		var start, end Object
		if node.Start != nil {
			start = Eval(node.Start, env)
			if isError(start) {
				return start
			}
		}
		if node.End != nil {
			end = Eval(node.End, env)
			if isError(end) {
				return end
			}
		}
		return evalSliceExpression(left, start, end)
	}

	return newError("unknown node type: %T", node)
}

// Helper functions
func evalProgram(stmts []ast.Statement, env *Environment) Object {
	var result Object

	for _, statement := range stmts {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *ReturnValue:
			return result.Value
		case *Error:
			return result
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
	var results []Object

	for _, statement := range block.Statements {
		result := Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_OBJ || rt == ERROR_OBJ {
				return result
			}

			// Collect non-NULL results
			if rt != NULL_OBJ {
				results = append(results, result)
			}
		}
	}

	// Return based on number of results
	switch len(results) {
	case 0:
		return NULL
	case 1:
		return results[0] // Single result: return directly (preserves type)
	default:
		return &Array{Elements: results} // Multiple results: return as array
	}
}

// evalInterpolationBlock evaluates an interpolation block inside tag contents
// Collects non-NULL results; returns single value, array, or NULL
func evalInterpolationBlock(block *ast.InterpolationBlock, env *Environment) Object {
	var results []Object

	for _, statement := range block.Statements {
		result := Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_OBJ || rt == ERROR_OBJ {
				return result
			}

			// Collect non-NULL results
			if rt != NULL_OBJ {
				results = append(results, result)
			}
		}
	}

	// Return based on number of results
	switch len(results) {
	case 0:
		return NULL
	case 1:
		return results[0] // Single result: return directly
	default:
		return &Array{Elements: results} // Multiple results: return as array
	}
}

func nativeBoolToParsBoolean(input bool) *Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(tok lexer.Token, operator string, right Object) Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "not":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(tok, right)
	default:
		return newErrorWithPos(tok, "unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right Object) Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(tok lexer.Token, right Object) Object {
	if right.Type() != INTEGER_OBJ {
		return newErrorWithPos(tok, "unknown operator: -%s", right.Type())
	}

	value := right.(*Integer).Value
	return &Integer{Value: -value}
}

func evalInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	switch {
	case operator == "&" || operator == "&&" || operator == "and":
		// Array intersection
		if left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ {
			return evalArrayIntersection(left.(*Array), right.(*Array))
		}
		// Datetime intersection (must come before general dictionary intersection)
		if left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ {
			leftDict := left.(*Dictionary)
			rightDict := right.(*Dictionary)
			if isDatetimeDict(leftDict) && isDatetimeDict(rightDict) {
				return evalDatetimeIntersection(tok, leftDict, rightDict, NewEnvironment())
			}
			// Regular dictionary intersection
			return evalDictionaryIntersection(leftDict, rightDict)
		}
		// Boolean and
		return nativeBoolToParsBoolean(isTruthy(left) && isTruthy(right))
	case operator == "|" || operator == "||" || operator == "or":
		// Array union
		if left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ {
			return evalArrayUnion(left.(*Array), right.(*Array))
		}
		// Boolean or
		return nativeBoolToParsBoolean(isTruthy(left) || isTruthy(right))
	case operator == "++":
		return evalConcatExpression(left, right)
	case operator == "in":
		return evalInExpression(tok, left, right)
	case operator == "..":
		return evalRangeExpression(tok, left, right)
	// Path and URL operators with strings (must come before general string concatenation)
	case left.Type() == DICTIONARY_OBJ && right.Type() == STRING_OBJ:
		if dict := left.(*Dictionary); isPathDict(dict) {
			return evalPathStringInfixExpression(tok, operator, dict, right.(*String))
		}
		if dict := left.(*Dictionary); isUrlDict(dict) {
			return evalUrlStringInfixExpression(tok, operator, dict, right.(*String))
		}
		// Fall through to string concatenation if not path/url
		if operator == "+" {
			return evalStringConcatExpression(left, right)
		}
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	case operator == "+" && (left.Type() == STRING_OBJ || right.Type() == STRING_OBJ):
		// String concatenation with automatic type conversion
		return evalStringConcatExpression(left, right)
	// Regex match operators
	case operator == "~" || operator == "!~":
		if left.Type() != STRING_OBJ {
			return newErrorWithPos(tok, "left operand of %s must be a string, got %s", operator, left.Type())
		}
		if right.Type() != DICTIONARY_OBJ {
			return newErrorWithPos(tok, "right operand of %s must be a regex, got %s", operator, right.Type())
		}
		rightDict := right.(*Dictionary)
		if !isRegexDict(rightDict) {
			return newErrorWithPos(tok, "right operand of %s must be a regex dictionary", operator)
		}
		result := evalMatchExpression(tok, left.(*String).Value, rightDict, NewEnvironment())
		if operator == "!~" {
			// !~ returns boolean: true if no match, false if match
			return nativeBoolToParsBoolean(result == NULL)
		}
		return result // ~ returns array or null
	// Datetime dictionary operations
	case left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ:
		leftDict := left.(*Dictionary)
		rightDict := right.(*Dictionary)
		if isDatetimeDict(leftDict) && isDatetimeDict(rightDict) {
			return evalDatetimeInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDurationDict(leftDict) && isDurationDict(rightDict) {
			return evalDurationInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDatetimeDict(leftDict) && isDurationDict(rightDict) {
			return evalDatetimeDurationInfixExpression(tok, operator, leftDict, rightDict)
		}
		if isDurationDict(leftDict) && isDatetimeDict(rightDict) {
			// duration + datetime not allowed, only datetime + duration
			return newErrorWithPos(tok, "cannot add datetime to duration (use datetime + duration instead)")
		}
		// Path dictionary operations
		if isPathDict(leftDict) && isPathDict(rightDict) {
			return evalPathInfixExpression(tok, operator, leftDict, rightDict)
		}
		// URL dictionary operations
		if isUrlDict(leftDict) && isUrlDict(rightDict) {
			return evalUrlInfixExpression(tok, operator, leftDict, rightDict)
		}
		// Dictionary subtraction for regular dicts
		if operator == "-" {
			return evalDictionarySubtraction(leftDict, rightDict)
		}
		// Fall through to default comparison for non-datetime dicts
		if operator == "==" {
			return nativeBoolToParsBoolean(left == right)
		} else if operator == "!=" {
			return nativeBoolToParsBoolean(left != right)
		}
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	case left.Type() == DICTIONARY_OBJ && right.Type() == INTEGER_OBJ:
		if dict := left.(*Dictionary); isDatetimeDict(dict) {
			return evalDatetimeIntegerInfixExpression(tok, operator, dict, right.(*Integer))
		}
		if dict := left.(*Dictionary); isDurationDict(dict) {
			return evalDurationIntegerInfixExpression(tok, operator, dict, right.(*Integer))
		}
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	case left.Type() == INTEGER_OBJ && right.Type() == DICTIONARY_OBJ:
		if dict := right.(*Dictionary); isDatetimeDict(dict) {
			return evalIntegerDatetimeInfixExpression(tok, operator, left.(*Integer), dict)
		}
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	// Array subtraction
	case operator == "-" && left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ:
		return evalArraySubtraction(left.(*Array), right.(*Array))
	// Array chunking
	case operator == "/" && left.Type() == ARRAY_OBJ && right.Type() == INTEGER_OBJ:
		return evalArrayChunking(tok, left.(*Array), right.(*Integer))
	// String repetition
	case operator == "*" && left.Type() == STRING_OBJ && right.Type() == INTEGER_OBJ:
		return evalStringRepetition(left.(*String), right.(*Integer))
	// Array repetition
	case operator == "*" && left.Type() == ARRAY_OBJ && right.Type() == INTEGER_OBJ:
		return evalArrayRepetition(left.(*Array), right.(*Integer))
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(tok, operator, left, right)
	case left.Type() == FLOAT_OBJ && right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(tok, operator, left, right)
	case left.Type() == INTEGER_OBJ && right.Type() == FLOAT_OBJ:
		return evalMixedInfixExpression(tok, operator, left, right)
	case left.Type() == FLOAT_OBJ && right.Type() == INTEGER_OBJ:
		return evalMixedInfixExpression(tok, operator, left, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(tok, operator, left, right)
	case operator == "==":
		return nativeBoolToParsBoolean(left == right)
	case operator == "!=":
		return nativeBoolToParsBoolean(left != right)
	case left.Type() != right.Type():
		return newErrorWithPos(tok, "type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*Integer).Value
	rightVal := right.(*Integer).Value

	switch operator {
	case "+":
		return &Integer{Value: leftVal + rightVal}
	case "-":
		return &Integer{Value: leftVal - rightVal}
	case "*":
		return &Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newErrorWithPos(tok, "division by zero")
		}
		return &Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newErrorWithPos(tok, "modulo by zero")
		}
		return &Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newErrorWithPos(tok, "unknown operator: %s", operator)
	}
}

func evalFloatInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*Float).Value
	rightVal := right.(*Float).Value

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newErrorWithPos(tok, "division by zero")
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newErrorWithPos(tok, "unknown operator: %s", operator)
	}
}

func evalMixedInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	var leftVal, rightVal float64

	// Convert both operands to float64
	switch left := left.(type) {
	case *Integer:
		leftVal = float64(left.Value)
	case *Float:
		leftVal = left.Value
	default:
		return newErrorWithPos(tok, "unsupported type for mixed arithmetic: %s", left.Type())
	}

	switch right := right.(type) {
	case *Integer:
		rightVal = float64(right.Value)
	case *Float:
		rightVal = right.Value
	default:
		return newErrorWithPos(tok, "unsupported type for mixed arithmetic: %s", right.Type())
	}

	switch operator {
	case "+":
		return &Float{Value: leftVal + rightVal}
	case "-":
		return &Float{Value: leftVal - rightVal}
	case "*":
		return &Float{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newErrorWithPos(tok, "division by zero")
		}
		return &Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToParsBoolean(leftVal < rightVal)
	case ">":
		return nativeBoolToParsBoolean(leftVal > rightVal)
	case "<=":
		return nativeBoolToParsBoolean(leftVal <= rightVal)
	case ">=":
		return nativeBoolToParsBoolean(leftVal >= rightVal)
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newErrorWithPos(tok, "unknown operator: %s", operator)
	}
}

func evalStringInfixExpression(tok lexer.Token, operator string, left, right Object) Object {
	leftVal := left.(*String).Value
	rightVal := right.(*String).Value

	switch operator {
	case "+":
		return &String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToParsBoolean(leftVal == rightVal)
	case "!=":
		return nativeBoolToParsBoolean(leftVal != rightVal)
	default:
		return newErrorWithPos(tok, "unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

// evalDatetimeInfixExpression handles operations between two datetime dictionaries
func evalDatetimeInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	env := NewEnvironment()

	// Handle && operator for combining date and time components
	if operator == "&" || operator == "&&" || operator == "and" {
		return evalDatetimeIntersection(tok, left, right, env)
	}

	leftUnix, err := getDatetimeUnix(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}
	rightUnix, err := getDatetimeUnix(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	switch operator {
	case "<":
		return nativeBoolToParsBoolean(leftUnix < rightUnix)
	case ">":
		return nativeBoolToParsBoolean(leftUnix > rightUnix)
	case "<=":
		return nativeBoolToParsBoolean(leftUnix <= rightUnix)
	case ">=":
		return nativeBoolToParsBoolean(leftUnix >= rightUnix)
	case "==":
		return nativeBoolToParsBoolean(leftUnix == rightUnix)
	case "!=":
		return nativeBoolToParsBoolean(leftUnix != rightUnix)
	case "-":
		// BREAKING CHANGE: Return Duration instead of Integer
		// Calculate difference in seconds
		diffSeconds := leftUnix - rightUnix
		// Return as duration (0 months, diffSeconds seconds)
		return durationToDict(0, diffSeconds, env)
	default:
		return newErrorWithPos(tok, "unknown operator for datetime: %s", operator)
	}
}

// evalDatetimeIntersection combines date and time components using && operator
// Rules:
// - Date && Time -> DateTime (combine date from left, time from right)
// - Time && Date -> DateTime (combine time from left, date from right)
// - DateTime && Time -> DateTime (replace time component)
// - DateTime && Date -> DateTime (replace date component)
// - Date && Date -> Error (ambiguous)
// - Time && Time -> Error (ambiguous)
// - DateTime && DateTime -> Error (ambiguous)
func evalDatetimeIntersection(tok lexer.Token, left, right *Dictionary, env *Environment) Object {
	leftKind := getDatetimeKind(left, env)
	rightKind := getDatetimeKind(right, env)

	// Get components from both sides
	leftTime, err := dictToTime(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}
	rightTime, err := dictToTime(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	var resultTime time.Time

	switch {
	case leftKind == "date" && rightKind == "time":
		// Date && Time -> combine date from left, time from right
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "time" && rightKind == "date":
		// Time && Date -> combine time from left, date from right
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "datetime" && rightKind == "time":
		// DateTime && Time -> replace time component
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "time" && rightKind == "datetime":
		// Time && DateTime -> replace time component of right
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "datetime" && rightKind == "date":
		// DateTime && Date -> replace date component
		resultTime = time.Date(
			rightTime.Year(), rightTime.Month(), rightTime.Day(),
			leftTime.Hour(), leftTime.Minute(), leftTime.Second(),
			0, time.UTC,
		)
	case leftKind == "date" && rightKind == "datetime":
		// Date && DateTime -> replace date component of right
		resultTime = time.Date(
			leftTime.Year(), leftTime.Month(), leftTime.Day(),
			rightTime.Hour(), rightTime.Minute(), rightTime.Second(),
			0, time.UTC,
		)
	case leftKind == "date" && rightKind == "date":
		return newErrorWithPos(tok, "cannot intersect two dates - use date && time to combine")
	case leftKind == "time" && rightKind == "time":
		return newErrorWithPos(tok, "cannot intersect two times - use date && time to combine")
	case leftKind == "datetime" && rightKind == "datetime":
		return newErrorWithPos(tok, "cannot intersect two datetimes - ambiguous which components to use")
	default:
		return newErrorWithPos(tok, "unknown datetime kinds: %s && %s", leftKind, rightKind)
	}

	return timeToDictWithKind(resultTime, "datetime", env)
}

// evalDatetimeIntegerInfixExpression handles datetime + integer or datetime - integer
func evalDatetimeIntegerInfixExpression(tok lexer.Token, operator string, dt *Dictionary, seconds *Integer) Object {
	env := NewEnvironment()
	unixTime, err := getDatetimeUnix(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add seconds to datetime
		newTime := time.Unix(unixTime+seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	case "-":
		// Subtract seconds from datetime
		newTime := time.Unix(unixTime-seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	default:
		return newErrorWithPos(tok, "unknown operator for datetime and integer: %s", operator)
	}
}

// evalIntegerDatetimeInfixExpression handles integer + datetime
func evalIntegerDatetimeInfixExpression(tok lexer.Token, operator string, seconds *Integer, dt *Dictionary) Object {
	env := NewEnvironment()
	unixTime, err := getDatetimeUnix(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add seconds to datetime (commutative)
		newTime := time.Unix(unixTime+seconds.Value, 0).UTC()
		return timeToDictWithKind(newTime, kind, env)
	default:
		return newErrorWithPos(tok, "unknown operator for integer and datetime: %s", operator)
	}
}

// evalDurationInfixExpression handles duration + duration or duration - duration
func evalDurationInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	env := NewEnvironment()

	leftMonths, leftSeconds, err := getDurationComponents(left, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	rightMonths, rightSeconds, err := getDurationComponents(right, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	switch operator {
	case "+":
		return durationToDict(leftMonths+rightMonths, leftSeconds+rightSeconds, env)
	case "-":
		return durationToDict(leftMonths-rightMonths, leftSeconds-rightSeconds, env)
	case "<", ">", "<=", ">=", "==", "!=":
		// Comparison only allowed for pure-seconds durations (no months)
		if leftMonths != 0 || rightMonths != 0 {
			return newErrorWithPos(tok, "cannot compare durations with month components (months have variable length)")
		}
		switch operator {
		case "<":
			return nativeBoolToParsBoolean(leftSeconds < rightSeconds)
		case ">":
			return nativeBoolToParsBoolean(leftSeconds > rightSeconds)
		case "<=":
			return nativeBoolToParsBoolean(leftSeconds <= rightSeconds)
		case ">=":
			return nativeBoolToParsBoolean(leftSeconds >= rightSeconds)
		case "==":
			return nativeBoolToParsBoolean(leftSeconds == rightSeconds && leftMonths == rightMonths)
		case "!=":
			return nativeBoolToParsBoolean(leftSeconds != rightSeconds || leftMonths != rightMonths)
		}
	}

	return newErrorWithPos(tok, "unknown operator for duration: %s", operator)
}

// evalDurationIntegerInfixExpression handles duration * integer or duration / integer
func evalDurationIntegerInfixExpression(tok lexer.Token, operator string, dur *Dictionary, num *Integer) Object {
	env := NewEnvironment()

	months, seconds, err := getDurationComponents(dur, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	switch operator {
	case "*":
		return durationToDict(months*num.Value, seconds*num.Value, env)
	case "/":
		if num.Value == 0 {
			return newErrorWithPos(tok, "division by zero")
		}
		return durationToDict(months/num.Value, seconds/num.Value, env)
	default:
		return newErrorWithPos(tok, "unknown operator for duration and integer: %s", operator)
	}
}

// evalDatetimeDurationInfixExpression handles datetime + duration or datetime - duration
func evalDatetimeDurationInfixExpression(tok lexer.Token, operator string, dt, dur *Dictionary) Object {
	env := NewEnvironment()

	// Get datetime as time.Time
	t, err := dictToTime(dt, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0004", tok, err)
	}

	// Get duration components
	months, seconds, err := getDurationComponents(dur, env)
	if err != nil {
		return newFormatErrorWithPos("FMT-0009", tok, err)
	}

	// Get the kind from the original datetime
	kind := getDatetimeKind(dt, env)

	switch operator {
	case "+":
		// Add months first (using AddDate for proper month arithmetic)
		if months != 0 {
			t = t.AddDate(0, int(months), 0)
		}
		// Then add seconds
		if seconds != 0 {
			t = t.Add(time.Duration(seconds) * time.Second)
		}
		return timeToDictWithKind(t, kind, env)
	case "-":
		// Subtract months first
		if months != 0 {
			t = t.AddDate(0, -int(months), 0)
		}
		// Then subtract seconds
		if seconds != 0 {
			t = t.Add(-time.Duration(seconds) * time.Second)
		}
		return timeToDictWithKind(t, kind, env)
	default:
		return newErrorWithPos(tok, "unknown operator for datetime and duration: %s", operator)
	}
}

// evalPathInfixExpression handles operations between two path dictionaries
func evalPathInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	switch operator {
	case "==":
		// Compare paths by their filesystem string representation
		leftStr := pathDictToString(left)
		rightStr := pathDictToString(right)
		return nativeBoolToParsBoolean(leftStr == rightStr)
	case "!=":
		leftStr := pathDictToString(left)
		rightStr := pathDictToString(right)
		return nativeBoolToParsBoolean(leftStr != rightStr)
	default:
		return newErrorWithPos(tok, "unknown operator for path: %s (supported: ==, !=)", operator)
	}
}

// evalPathStringInfixExpression handles path + string or path / string
func evalPathStringInfixExpression(tok lexer.Token, operator string, path *Dictionary, str *String) Object {
	env := path.Env
	if env == nil {
		env = NewEnvironment()
	}

	switch operator {
	case "+", "/":
		// Join path with string segment
		// Get current components
		componentsExpr, ok := path.Pairs["components"]
		if !ok {
			return newErrorWithPos(tok, "path dictionary missing components field")
		}
		componentsObj := Eval(componentsExpr, env)
		if componentsObj.Type() != ARRAY_OBJ {
			return newErrorWithPos(tok, "path components is not an array")
		}
		componentsArr := componentsObj.(*Array)

		// Get absolute flag
		absoluteExpr, ok := path.Pairs["absolute"]
		if !ok {
			return newErrorWithPos(tok, "path dictionary missing absolute field")
		}
		absoluteObj := Eval(absoluteExpr, env)
		if absoluteObj.Type() != BOOLEAN_OBJ {
			return newErrorWithPos(tok, "path absolute is not a boolean")
		}
		isAbsolute := absoluteObj.(*Boolean).Value

		// Parse the string to add as new path segments
		newSegments, _ := parsePathString(str.Value)

		// Combine components
		var newComponents []string
		for _, elem := range componentsArr.Elements {
			if strObj, ok := elem.(*String); ok {
				newComponents = append(newComponents, strObj.Value)
			}
		}

		// Append new segments (skip empty leading segment if present)
		for _, seg := range newSegments {
			if seg != "" || len(newComponents) == 0 {
				newComponents = append(newComponents, seg)
			}
		}

		return pathToDict(newComponents, isAbsolute, env)
	default:
		return newErrorWithPos(tok, "unknown operator for path and string: %s (supported: +, /)", operator)
	}
}

// evalUrlInfixExpression handles operations between two URL dictionaries
func evalUrlInfixExpression(tok lexer.Token, operator string, left, right *Dictionary) Object {
	switch operator {
	case "==":
		// Compare URLs by their string representation
		leftStr := urlDictToString(left)
		rightStr := urlDictToString(right)
		return nativeBoolToParsBoolean(leftStr == rightStr)
	case "!=":
		leftStr := urlDictToString(left)
		rightStr := urlDictToString(right)
		return nativeBoolToParsBoolean(leftStr != rightStr)
	default:
		return newErrorWithPos(tok, "unknown operator for url: %s (supported: ==, !=)", operator)
	}
}

// evalUrlStringInfixExpression handles url + string for path joining
func evalUrlStringInfixExpression(tok lexer.Token, operator string, urlDict *Dictionary, str *String) Object {
	env := urlDict.Env
	if env == nil {
		env = NewEnvironment()
	}

	switch operator {
	case "+":
		// Add string to URL path
		// Get current path array
		pathExpr, ok := urlDict.Pairs["path"]
		if !ok {
			return newErrorWithPos(tok, "url dictionary missing path field")
		}
		pathObj := Eval(pathExpr, env)
		if pathObj.Type() != ARRAY_OBJ {
			return newErrorWithPos(tok, "url path is not an array")
		}
		pathArr := pathObj.(*Array)

		// Parse the string as a path to add
		newSegments, _ := parsePathString(str.Value)

		// Combine path segments
		var newPath []string
		for _, elem := range pathArr.Elements {
			if strObj, ok := elem.(*String); ok {
				newPath = append(newPath, strObj.Value)
			}
		}

		// Append new segments (skip empty leading segment)
		for _, seg := range newSegments {
			if seg != "" {
				newPath = append(newPath, seg)
			}
		}

		// Create new URL dict with updated path
		pairs := make(map[string]ast.Expression)
		for k, v := range urlDict.Pairs {
			if k == "path" {
				// Create new path array
				pathElements := make([]ast.Expression, len(newPath))
				for i, seg := range newPath {
					pathElements[i] = &ast.StringLiteral{Value: seg}
				}
				pairs[k] = &ast.ArrayLiteral{Elements: pathElements}
			} else {
				pairs[k] = v
			}
		}

		return &Dictionary{Pairs: pairs, Env: env}
	default:
		return newErrorWithPos(tok, "unknown operator for url and string: %s (supported: +)", operator)
	}
}

// evalStringConcatExpression handles string concatenation with automatic type conversion
func evalStringConcatExpression(left, right Object) Object {
	leftStr := objectToTemplateString(left)
	rightStr := objectToTemplateString(right)
	return &String{Value: leftStr + rightStr}
}

func evalIfExpression(ie *ast.IfExpression, env *Environment) Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func isTruthy(obj Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func evalIdentifier(node *ast.Identifier, env *Environment) Object {
	// Special handling for '_' - always returns null
	if node.Value == "_" {
		return NULL
	}

	// Special handling for 'null' - returns null
	if node.Value == "null" {
		return NULL
	}

	// Special handling for '__null__' - internal null representation
	if node.Value == "__null__" {
		return NULL
	}

	val, ok := env.Get(node.Value)
	if !ok {
		if builtin, ok := getBuiltins()[node.Value]; ok {
			return builtin
		}

		// Create a structured error with fuzzy matching
		parsleyErr := perrors.NewUndefinedIdentifier(node.Value, env.AllIdentifiers())
		parsleyErr.Line = node.Token.Line
		parsleyErr.Column = node.Token.Column

		// Also check for common keywords that might be misspelled
		if len(parsleyErr.Hints) == 0 {
			if suggestion := perrors.FindClosestMatch(node.Value, perrors.ParsleyKeywords); suggestion != "" {
				parsleyErr.Hints = append(parsleyErr.Hints, "Did you mean `"+suggestion+"`?")
			}
		}

		return &Error{
			Message: parsleyErr.Message,
			Class:   parsleyErr.Class,
			Code:    parsleyErr.Code,
			Hints:   parsleyErr.Hints,
			Line:    parsleyErr.Line,
			Column:  parsleyErr.Column,
		}
	}

	return val
}

func evalExpressions(exps []ast.Expression, env *Environment) []Object {
	var result []Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn Object, args []Object) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *Builtin:
		return fn.Fn(args...)
	case *StdlibBuiltin:
		// StdlibBuiltin needs an environment but applyFunction doesn't have one
		// This shouldn't happen as StdlibBuiltin should be called via applyFunctionWithEnv
		return newError("stdlib function called without environment context")
	default:
		if fn == NULL || fn == nil {
			return newError("cannot call null as a function\n   💡 Hint: The value may not be exported from an imported module, or the variable is uninitialized")
		}
		return newError("cannot call %s as a function\n   💡 Hint: Only functions can be called with parentheses", fn.Type())
	}
}

// applyMethodWithThis calls a function with 'this' bound to a dictionary.
// This enables object-oriented style method calls like user.greet() where
// the function can access the dictionary via 'this'.
func applyMethodWithThis(fn *Function, args []Object, thisObj *Dictionary) Object {
	extendedEnv := extendFunctionEnv(fn, args)
	extendedEnv.Set("this", thisObj)
	evaluated := Eval(fn.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

func applyFunctionWithEnv(fn Object, args []Object, env *Environment) Object {
	switch fn := fn.(type) {
	case *Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *Builtin:
		result := fn.Fn(args...)
		// Add position info to builtin errors for better debugging
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *StdlibBuiltin:
		result := fn.Fn(args, env)
		// Add position info to stdlib errors for better debugging
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *TableModule:
		// TableModule is callable: table(arr) creates a Table from an array
		result := TableConstructor(args, env)
		if isError(result) {
			return enrichErrorWithPos(result, env.LastToken)
		}
		return result
	case *DevModule:
		// DevModule is not directly callable, only used as a namespace
		return newError("dev module cannot be called directly, use dev.log() or other methods")
	case *SFTPConnection:
		// SFTP connection is callable: conn(@/path) returns SFTP file handle
		if len(args) != 1 {
			return newArityError("SFTP", len(args), 1)
		}

		// Extract path from argument
		var pathStr string
		switch arg := args[0].(type) {
		case *Dictionary:
			if !isPathDict(arg) {
				return newTypeError("TYPE-0012", "SFTP connection", "a path", DICTIONARY_OBJ)
			}
			pathStr = pathDictToString(arg)
		case *String:
			pathStr = arg.Value
		default:
			return newTypeError("TYPE-0012", "SFTP connection", "a path", arg.Type())
		}

		// Return SFTP file handle
		return &SFTPFileHandle{
			Connection: fn,
			Path:       pathStr,
			Format:     "", // Will default to "text"
			Options:    nil,
		}
	default:
		if fn == NULL || fn == nil {
			return newError("cannot call null as a function\n   💡 Hint: The value may not be exported from an imported module, or the variable is uninitialized")
		}
		return newError("cannot call %s as a function\n   💡 Hint: Only functions can be called with parentheses", fn.Type())
	}
}

// evalImport implements the import(path) builtin
func evalImport(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("import", len(args), 1)
	}

	// Extract path string from argument (handle both path dictionaries and strings)
	var pathStr string
	switch arg := args[0].(type) {
	case *Dictionary:
		// Handle path literal (@/path/to/file.pars)
		if typeExpr, ok := arg.Pairs["__type"]; ok {
			typeVal := Eval(typeExpr, arg.Env)
			if typeStr, ok := typeVal.(*String); ok && typeStr.Value == "path" {
				pathStr = pathDictToString(arg)
			} else {
				return newTypeError("TYPE-0012", "import", "a path or string", DICTIONARY_OBJ)
			}
		} else {
			return newTypeError("TYPE-0012", "import", "a path or string", DICTIONARY_OBJ)
		}
	case *String:
		pathStr = arg.Value
	default:
		return newTypeError("TYPE-0012", "import", "a path or string", arg.Type())
	}

	// Check for standard library imports (@std/modulename)
	if strings.HasPrefix(pathStr, "std/") {
		moduleName := strings.TrimPrefix(pathStr, "std/")
		return loadStdlibModule(moduleName, env)
	}

	// Resolve path relative to current file (or root path for ~/ paths)
	absPath, err := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if err != nil {
		return newError("failed to resolve module path: %s", err.Error())
	}

	// Security check
	if err := env.checkPathAccess(absPath, "execute"); err != nil {
		return newSecurityError("execute", err)
	}

	// Check if module is currently being loaded in THIS request (circular dependency)
	// Use the root environment's importStack to track across nested imports
	rootEnv := env
	for rootEnv.outer != nil {
		rootEnv = rootEnv.outer
	}
	if rootEnv.importStack[absPath] {
		return newError("circular dependency detected when importing: %s", absPath)
	}

	// Check cache first (with lock for thread safety)
	moduleCache.mu.RLock()
	if cached, ok := moduleCache.modules[absPath]; ok {
		moduleCache.mu.RUnlock()
		return cached
	}
	moduleCache.mu.RUnlock()

	// Mark as loading in this request's import stack
	rootEnv.importStack[absPath] = true
	defer delete(rootEnv.importStack, absPath)

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return newIOError("IO-0002", absPath, err)
		}
		return newIOError("IO-0003", absPath, err)
	}

	// Parse the module
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString(fmt.Sprintf("parse errors in module %s:\n", absPath))
		for _, msg := range p.Errors() {
			errMsg.WriteString(fmt.Sprintf("  %s\n", msg))
		}
		return newError("%s", errMsg.String())
	}

	// Create isolated environment for the module
	moduleEnv := NewEnvironment()
	moduleEnv.Filename = absPath
	// Copy root path from parent environment (preserved across imports for ~/ resolution)
	moduleEnv.RootPath = env.RootPath
	// Copy security policy from parent environment
	moduleEnv.Security = env.Security
	// Copy DevLog and BasilCtx for stdlib imports (std/dev, std/basil)
	moduleEnv.DevLog = env.DevLog
	moduleEnv.BasilCtx = env.BasilCtx

	// Copy basil context to module environment (if present)
	// This allows modules to access basil.http, basil.auth, basil.sqlite etc.
	if basil, ok := env.Get("basil"); ok {
		moduleEnv.SetProtected("basil", basil)
	}

	// Evaluate the module
	result := Eval(program, moduleEnv)

	// Check for errors during module evaluation
	if isError(result) {
		errObj := result.(*Error)
		// Include module path in error message for context
		if errObj.Line > 0 {
			return newError("in module %s: line %d, column %d: %s", absPath, errObj.Line, errObj.Column, errObj.Message)
		}
		return newError("in module %s: %s", absPath, errObj.Message)
	}

	// Convert environment to dictionary
	moduleDict := environmentToDict(moduleEnv)

	// Cache the result
	moduleCache.mu.Lock()
	moduleCache.modules[absPath] = moduleDict
	moduleCache.mu.Unlock()

	return moduleDict
}

// evalLogLine implements logLine with filename and line number
func evalLogLine(args []Object, env *Environment) Object {
	var result strings.Builder

	// Add filename and line number prefix
	filename := env.Filename
	if filename == "" {
		filename = "<unknown>"
	}
	line := 1
	if env.LastToken != nil {
		line = env.LastToken.Line
	}
	result.WriteString(fmt.Sprintf("%s:%d: ", filename, line))

	// Process arguments like log()
	for i, arg := range args {
		if i == 0 {
			// First argument: if it's a string, show without quotes
			if str, ok := arg.(*String); ok {
				result.WriteString(str.Value)
			} else {
				result.WriteString(objectToDebugString(arg))
			}
		} else {
			// Subsequent arguments: add separator and debug format
			if i == 1 {
				// After first string, no comma - just space
				if _, firstWasString := args[0].(*String); firstWasString {
					result.WriteString(" ")
				} else {
					result.WriteString(", ")
				}
			} else {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(arg))
		}
	}

	// Use the environment's logger
	if env.Logger != nil {
		env.Logger.LogLine(result.String())
	} else {
		fmt.Fprintln(os.Stdout, result.String())
	}

	// Return null
	return NULL
}

// evalLog implements log() using the environment's logger
func evalLog(args []Object, env *Environment) Object {
	var result strings.Builder

	for i, arg := range args {
		if i == 0 {
			// First argument: if it's a string, show without quotes
			if str, ok := arg.(*String); ok {
				result.WriteString(str.Value)
			} else {
				result.WriteString(objectToDebugString(arg))
			}
		} else {
			// Subsequent arguments: add separator and debug format
			if i == 1 {
				// After first string, no comma - just space
				if _, firstWasString := args[0].(*String); firstWasString {
					result.WriteString(" ")
				} else {
					result.WriteString(", ")
				}
			} else {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(arg))
		}
	}

	// Use the environment's logger
	if env.Logger != nil {
		env.Logger.LogLine(result.String())
	} else {
		fmt.Fprintln(os.Stdout, result.String())
	}

	// Return null
	return NULL
}

func extendFunctionEnv(fn *Function, args []Object) *Environment {
	env := NewEnclosedEnvironment(fn.Env)

	// Use parameter list with destructuring support
	for paramIdx, param := range fn.Params {
		if paramIdx >= len(args) {
			break
		}
		arg := args[paramIdx]

		// Handle different parameter types
		if param.DictPattern != nil {
			// Dictionary destructuring (in function params, never exported)
			evalDictDestructuringAssignment(param.DictPattern, arg, env, true, false)
		} else if len(param.ArrayPattern) > 0 {
			// Array destructuring
			evalArrayDestructuringForParam(param.ArrayPattern, arg, env)
		} else if param.Ident != nil {
			// Simple identifier
			env.Set(param.Ident.Value, arg)
		}
	}

	return env
}

// evalArrayDestructuringForParam handles array destructuring in function parameters
func evalArrayDestructuringForParam(pattern []*ast.Identifier, val Object, env *Environment) {
	// Convert value to array if it isn't already
	var elements []Object

	switch v := val.(type) {
	case *Array:
		elements = v.Elements
	default:
		// Single value becomes single-element array
		elements = []Object{v}
	}

	// Assign each element to corresponding variable
	for i, name := range pattern {
		if i < len(elements) {
			if name.Value != "_" {
				env.Set(name.Value, elements[i])
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				env.Set(name.Value, NULL)
			}
		}
	}

	// If there are more elements than names, assign remaining as array to last variable
	if len(elements) > len(pattern) && len(pattern) > 0 {
		lastIdx := len(pattern) - 1
		lastName := pattern[lastIdx]
		if lastName.Value != "_" {
			// Replace the last assignment with an array of remaining elements
			remaining := &Array{Elements: elements[lastIdx:]}
			env.Set(lastName.Value, remaining)
		}
	}
}

func unwrapReturnValue(obj Object) Object {
	if returnValue, ok := obj.(*ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

// evalForExpression evaluates for expressions
func evalForExpression(node *ast.ForExpression, env *Environment) Object {
	// Evaluate the array/dict expression
	iterableObj := Eval(node.Array, env)
	if isError(iterableObj) {
		return iterableObj
	}

	// Handle response typed dictionary - unwrap __data for iteration
	if dict, ok := iterableObj.(*Dictionary); ok && isResponseDict(dict) {
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			iterableObj = Eval(dataExpr, dict.Env)
			if isError(iterableObj) {
				return iterableObj
			}
		}
	}

	// Handle dictionary iteration
	if dict, ok := iterableObj.(*Dictionary); ok {
		return evalForDictExpression(node, dict, env)
	}

	// Convert to array (handle strings as rune arrays)
	var elements []Object
	switch arr := iterableObj.(type) {
	case *Array:
		elements = arr.Elements
	case *String:
		// Convert string to array of single-character strings
		runes := []rune(arr.Value)
		elements = make([]Object, len(runes))
		for i, r := range runes {
			elements[i] = &String{Value: string(r)}
		}
	default:
		return newError("for expects an array, string, or dictionary, got %s", iterableObj.Type())
	}

	// Determine which function to use
	var fn Object
	if node.Function != nil {
		// Simple form: for(array) func
		fn = Eval(node.Function, env)
		if isError(fn) {
			return fn
		}
		// Accept both functions and builtins
		switch fn.(type) {
		case *Function, *Builtin:
			// OK
		default:
			return newError("for expects a function or builtin, got %s", fn.Type())
		}
	} else if node.Body != nil {
		// 'in' form: for(var in array) body
		// node.Body is already a FunctionLiteral with the variable as parameter
		fn = &Function{
			Params: node.Body.(*ast.FunctionLiteral).Params,
			Body:   node.Body.(*ast.FunctionLiteral).Body,
			Env:    env,
		}
	} else {
		return newError("for expression missing function or body")
	}

	// Map function over array elements
	result := []Object{}
	for idx, elem := range elements {
		var evaluated Object

		switch f := fn.(type) {
		case *Builtin:
			// Call builtin with single element
			evaluated = f.Fn(elem)
		case *Function:
			// Call user function
			paramCount := f.ParamCount()
			if paramCount != 1 && paramCount != 2 {
				return newError("function passed to for must take 1 or 2 parameters, got %d", paramCount)
			}

			// Prepare arguments based on parameter count
			var args []Object
			if paramCount == 2 {
				// Two parameters: index and element
				args = []Object{&Integer{Value: int64(idx)}, elem}
			} else {
				// One parameter: element only (backward compatible)
				args = []Object{elem}
			}

			// Create a new environment and bind the parameters
			extendedEnv := extendFunctionEnv(f, args)

			// Evaluate all statements in the body
			for _, stmt := range f.Body.Statements {
				evaluated = evalStatement(stmt, extendedEnv)
				if returnValue, ok := evaluated.(*ReturnValue); ok {
					evaluated = returnValue.Value
					break
				}
				if isError(evaluated) {
					return evaluated
				}
			}
		}

		// Skip null values (filter behavior)
		if evaluated != NULL {
			result = append(result, evaluated)
		}
	}

	return &Array{Elements: result}
}

// evalForDictExpression handles for loops over dictionaries
func evalForDictExpression(node *ast.ForExpression, dict *Dictionary, env *Environment) Object {
	// Create environment for evaluation with 'this'
	dictEnv := NewEnclosedEnvironment(dict.Env)
	dictEnv.Set("this", dict)

	// Determine which function to use
	var fn *Function
	if node.Body != nil {
		bodyFn := node.Body.(*ast.FunctionLiteral)
		if len(bodyFn.Params) > 0 {
			fn = &Function{
				Params: bodyFn.Params,
				Body:   bodyFn.Body,
				Env:    env,
			}
		} else {
			return newError("for loop over dictionary requires body with key, value parameters")
		}
	} else {
		return newError("for loop over dictionary requires function body")
	}

	// Check parameter count
	if fn.ParamCount() != 2 {
		return newError("for loop over dictionary requires exactly 2 parameters (key, value), got %d", fn.ParamCount())
	}

	// Iterate over dictionary key-value pairs
	result := []Object{}
	for key, expr := range dict.Pairs {
		// Evaluate the value
		value := Eval(expr, dictEnv)
		if isError(value) {
			return value
		}

		// Create environment for loop body with both key and value
		extendedEnv := extendFunctionEnv(fn, []Object{&String{Value: key}, value})

		// Evaluate all statements in the body
		var evaluated Object
		for _, stmt := range fn.Body.Statements {
			evaluated = evalStatement(stmt, extendedEnv)
			if returnValue, ok := evaluated.(*ReturnValue); ok {
				evaluated = returnValue.Value
				break
			}
			if isError(evaluated) {
				return evaluated
			}
		}

		// Skip null values (filter behavior)
		if evaluated != NULL {
			result = append(result, evaluated)
		}
	}

	return &Array{Elements: result}
}

func newError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

// newErrorWithPos creates an error with position information from a token
func newErrorWithPos(tok lexer.Token, format string, a ...interface{}) *Error {
	return &Error{
		Message: fmt.Sprintf(format, a...),
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newErrorWithClass creates an error with a specific class.
func newErrorWithClass(class ErrorClass, format string, a ...interface{}) *Error {
	return &Error{
		Class:   class,
		Message: fmt.Sprintf(format, a...),
	}
}

// newErrorWithClassAndPos creates an error with class and position information.
func newErrorWithClassAndPos(class ErrorClass, tok lexer.Token, format string, a ...interface{}) *Error {
	return &Error{
		Class:   class,
		Message: fmt.Sprintf(format, a...),
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newStructuredError creates a structured error from the catalog.
func newStructuredError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newStructuredErrorWithPos creates a structured error with position information.
func newStructuredErrorWithPos(code string, tok lexer.Token, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Line:    tok.Line,
		Column:  tok.Column,
		Data:    perr.Data,
	}
}

// newSecurityError creates a structured security error from a checkPathAccess error.
// The operation should be "read", "write", or "execute".
// We preserve the original error message for specificity (e.g., "file read restricted: /path")
// but add structured metadata for programmatic handling.
func newSecurityError(operation string, err error) *Error {
	// Map operation to error code
	var code string
	switch operation {
	case "read":
		code = "SEC-0002"
	case "write":
		code = "SEC-0003"
	case "execute":
		code = "SEC-0004"
	default:
		code = "SEC-0001"
	}

	// Get the catalog entry for hints
	perr := perrors.New(code, map[string]any{
		"Operation": operation,
	})

	// Use original error message for specificity, but add structured metadata
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: "security: " + err.Error(), // Preserve original specific message
		Hints:   perr.Hints,
		Data: map[string]any{
			"Operation": operation,
			"GoError":   err.Error(),
		},
	}
}

// newDatabaseError creates a structured database error.
func newDatabaseError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newDatabaseStateError creates a structured database state error (no Go error).
func newDatabaseStateError(code string) *Error {
	perr := perrors.New(code, nil)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
	}
}

// newDatabaseErrorWithDriver creates a structured database error with driver info.
func newDatabaseErrorWithDriver(code, driver string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"Driver":  driver,
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newTypeError creates a structured type error for function arguments.
// code should be TYPE-0001 (general), TYPE-0005 (first arg), or TYPE-0006 (second arg).
func newTypeError(code, function, expected string, got ObjectType) *Error {
	perr := perrors.New(code, map[string]any{
		"Function": function,
		"Expected": expected,
		"Got":      perrors.TypeName(string(got)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIndexTypeError creates a structured error for unsupported index operations.
func newIndexTypeError(tok lexer.Token, left, index ObjectType) *Error {
	perr := perrors.New("TYPE-0013", map[string]any{
		"Left":  perrors.TypeName(string(left)),
		"Right": perrors.TypeName(string(index)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newSliceTypeError creates a structured error for unsupported slice operations.
func newSliceTypeError(left ObjectType) *Error {
	perr := perrors.New("TYPE-0014", map[string]any{
		"Type": perrors.TypeName(string(left)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityError creates a structured error for wrong number of arguments (exact count).
func newArityError(function string, got, want int) *Error {
	perr := perrors.New("ARITY-0001", map[string]any{
		"Function": function,
		"Got":      got,
		"Want":     want,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityErrorRange creates a structured error for wrong number of arguments (range).
func newArityErrorRange(function string, got, min, max int) *Error {
	perr := perrors.New("ARITY-0004", map[string]any{
		"Function": function,
		"Got":      got,
		"Min":      min,
		"Max":      max,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityErrorMin creates a structured error for minimum arguments required.
func newArityErrorMin(function string, got, min int) *Error {
	perr := perrors.New("ARITY-0005", map[string]any{
		"Function": function,
		"Got":      got,
		"Min":      min,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIOError creates a structured error for I/O operations.
func newIOError(code string, path string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"Path":    path,
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newFormatError creates a structured error for format/parsing issues.
func newFormatError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newUndefinedMethodError creates a structured error for unknown methods.
func newUndefinedMethodError(method string, typeName string) *Error {
	perr := perrors.New("UNDEF-0002", map[string]any{
		"Method": method,
		"Type":   typeName,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newStateError creates a structured error for state-related issues.
func newStateError(code string) *Error {
	perr := perrors.New(code, nil)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newUndefinedComponentError creates a structured error for undefined components.
func newUndefinedComponentError(name string) *Error {
	perr := perrors.New("UNDEF-0003", map[string]any{
		"Name": name,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}
// newLocaleError creates a structured error for invalid locale.
func newLocaleError(locale string) *Error {
	perr := perrors.New("FMT-0008", map[string]any{
		"Locale": locale,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newFormatErrorWithPos creates a structured format error with position info.
func newFormatErrorWithPos(code string, tok lexer.Token, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newParseError creates a structured parse error for template syntax issues.
func newParseError(code string, context string, err error) *Error {
	data := map[string]any{
		"Context": context,
	}
	if err != nil {
		data["GoError"] = err.Error()
	}
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newConversionError creates a structured type error for value conversion failures.
func newConversionError(code string, value string) *Error {
	perr := perrors.New(code, map[string]any{
		"Value": value,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newNetworkError creates a structured network error.
func newNetworkError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newSliceIndexTypeError creates a structured type error for slice index type issues.
func newSliceIndexTypeError(position string, got string) *Error {
	perr := perrors.New("TYPE-0018", map[string]any{
		"Position": position,
		"Got":      got,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIndexError creates a structured index error.
func newIndexError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// enrichErrorWithPos adds position info to an error that doesn't have it.
// This is useful for wrapping errors from builtins at the call site.
func enrichErrorWithPos(obj Object, tok *lexer.Token) Object {
	if tok == nil {
		return obj
	}
	if errObj, ok := obj.(*Error); ok && errObj.Line == 0 {
		errObj.Line = tok.Line
		errObj.Column = tok.Column
	}
	return obj
}

func isError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

// evalDestructuringAssignment handles array destructuring assignment
func evalDestructuringAssignment(names []*ast.Identifier, val Object, env *Environment, isLet bool, export bool) Object {
	// Convert value to array if it isn't already
	var elements []Object

	switch v := val.(type) {
	case *Array:
		elements = v.Elements
	default:
		// Single value becomes single-element array
		elements = []Object{v}
	}

	// Assign each element to corresponding variable
	for i, name := range names {
		if i < len(elements) {
			// Direct assignment for elements within bounds
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, elements[i])
				} else if export {
					env.SetExport(name.Value, elements[i])
				} else if isLet {
					env.SetLet(name.Value, elements[i])
				} else {
					env.Update(name.Value, elements[i])
				}
			}
		} else {
			// No more elements, assign null
			if name.Value != "_" {
				if export && isLet {
					env.SetLetExport(name.Value, NULL)
				} else if export {
					env.SetExport(name.Value, NULL)
				} else if isLet {
					env.SetLet(name.Value, NULL)
				} else {
					env.Update(name.Value, NULL)
				}
			}
		}
	}

	// If there are more elements than names, assign remaining as array to last variable
	if len(elements) > len(names) && len(names) > 0 {
		lastIdx := len(names) - 1
		lastName := names[lastIdx]
		if lastName.Value != "_" {
			// Replace the last assignment with an array of remaining elements
			remaining := &Array{Elements: elements[lastIdx:]}
			if export && isLet {
				env.SetLetExport(lastName.Value, remaining)
			} else if export {
				env.SetExport(lastName.Value, remaining)
			} else if isLet {
				env.SetLet(lastName.Value, remaining)
			} else {
				env.Update(lastName.Value, remaining)
			}
		}
	}

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// evalDictDestructuringAssignment evaluates dictionary destructuring patterns
func evalDictDestructuringAssignment(pattern *ast.DictDestructuringPattern, val Object, env *Environment, isLet bool, export bool) Object {
	// Handle StdlibModuleDict (from @std/ imports)
	if stdlibMod, ok := val.(*StdlibModuleDict); ok {
		return evalStdlibModuleDestructuring(pattern, stdlibMod, env, isLet, export)
	}

	// Type check: value must be a dictionary
	dict, ok := val.(*Dictionary)
	if !ok {
		return newError("dictionary destructuring requires a dictionary value, got %s", val.Type())
	}

	// Track which keys we've extracted (for rest operator)
	extractedKeys := make(map[string]bool)

	// Process each key in the pattern
	for _, keyPattern := range pattern.Keys {
		keyName := keyPattern.Key.Value
		extractedKeys[keyName] = true

		// Get expression from dictionary and evaluate it
		var value Object
		if expr, exists := dict.Pairs[keyName]; exists {
			// Evaluate the expression in the dictionary's environment
			value = Eval(expr, dict.Env)
			if isError(value) {
				return value
			}
		} else {
			// If key not found, assign null
			value = NULL
		}

		// Handle nested destructuring
		if keyPattern.Nested != nil {
			if nestedPattern, ok := keyPattern.Nested.(*ast.DictDestructuringPattern); ok {
				result := evalDictDestructuringAssignment(nestedPattern, value, env, isLet, export)
				if isError(result) {
					return result
				}
			} else {
				return newError("unsupported nested destructuring pattern")
			}
		} else {
			// Determine the target variable name (alias or original key)
			targetName := keyName
			if keyPattern.Alias != nil {
				targetName = keyPattern.Alias.Value
			}

			// Assign to environment
			if targetName != "_" {
				if export && isLet {
					env.SetLetExport(targetName, value)
				} else if export {
					env.SetExport(targetName, value)
				} else if isLet {
					env.Set(targetName, value)
				} else {
					env.Update(targetName, value)
				}
			}
		}
	}

	// Handle rest operator
	if pattern.Rest != nil {
		restPairs := make(map[string]ast.Expression)
		for key, expr := range dict.Pairs {
			if !extractedKeys[key] {
				restPairs[key] = expr
			}
		}

		restDict := &Dictionary{Pairs: restPairs, Env: dict.Env}
		if pattern.Rest.Value != "_" {
			if export && isLet {
				env.SetLetExport(pattern.Rest.Value, restDict)
			} else if export {
				env.SetExport(pattern.Rest.Value, restDict)
			} else if isLet {
				env.SetLet(pattern.Rest.Value, restDict)
			} else {
				env.Update(pattern.Rest.Value, restDict)
			}
		}
	}

	// Destructuring assignments return NULL (excluded from block concatenation)
	return NULL
}

// evalTemplateLiteral evaluates a template literal with interpolation
func evalTemplateLiteral(node *ast.TemplateLiteral, env *Environment) Object {
	template := node.Value
	var result strings.Builder

	i := 0
	for i < len(template) {
		// Look for {
		if template[i] == '{' {
			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(template) && braceCount > 0 {
				if template[i] == '{' {
					braceCount++
				} else if template[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "template literal", nil)
			}

			// Extract and evaluate the expression
			exprStr := template[exprStart:i]
			i++ // skip closing }

			// Parse and evaluate the expression
			l := lexer.New(exprStr)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				return newParseError("PARSE-0011", "template", fmt.Errorf("%s", p.Errors()[0]))
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					return evaluated
				}
			}

			// Convert result to string
			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
		} else {
			// Regular character
			result.WriteByte(template[i])
			i++
		}
	}

	return &String{Value: result.String()}
}

// evalTagLiteral evaluates a singleton tag
func evalTagLiteral(node *ast.TagLiteral, env *Environment) Object {
	raw := node.Raw

	// Parse tag name (first word)
	i := 0
	for i < len(raw) && !unicode.IsSpace(rune(raw[i])) {
		i++
	}
	tagName := raw[:i]
	rest := raw[i:]

	// Check if it's a custom tag (starts with uppercase)
	isCustom := len(tagName) > 0 && unicode.IsUpper(rune(tagName[0]))

	if isCustom {
		// Custom tag - call function with props dictionary
		return evalCustomTag(tagName, rest, env)
	} else {
		// Standard tag - return as interpolated string
		return evalStandardTag(tagName, rest, env)
	}
}

// evalTagPair evaluates a paired tag like <div>content</div> or <Component>content</Component>
func evalTagPair(node *ast.TagPairExpression, env *Environment) Object {
	// Empty grouping tag <> just returns its contents
	if node.Name == "" {
		return evalTagContents(node.Contents, env)
	}

	// Check if it's a custom component (starts with uppercase)
	isCustom := len(node.Name) > 0 && unicode.IsUpper(rune(node.Name[0]))

	if isCustom {
		// Custom component - call function with props dictionary including contents
		return evalCustomTagPair(node, env)
	} else {
		// Standard tag - return as HTML string
		return evalStandardTagPair(node, env)
	}
}

// evalStandardTagPair evaluates a standard (lowercase) tag pair as HTML string
func evalStandardTagPair(node *ast.TagPairExpression, env *Environment) Object {
	var result strings.Builder

	result.WriteByte('<')
	result.WriteString(node.Name)

	// Process props with interpolation (similar to singleton tags)
	if node.Props != "" {
		result.WriteByte(' ')
		propsResult := evalTagProps(node.Props, env)
		if isError(propsResult) {
			return propsResult
		}
		result.WriteString(propsResult.(*String).Value)
	}

	result.WriteByte('>')

	// Evaluate and append contents
	contentsObj := evalTagContents(node.Contents, env)
	if isError(contentsObj) {
		return contentsObj
	}
	result.WriteString(contentsObj.(*String).Value)

	result.WriteString("</")
	result.WriteString(node.Name)
	result.WriteByte('>')

	return &String{Value: result.String()}
}

// evalCustomTagPair evaluates a custom (uppercase) tag pair as a function call
func evalCustomTagPair(node *ast.TagPairExpression, env *Environment) Object {
	// Special handling for <SQL> tags
	if node.Name == "SQL" {
		return evalSQLTag(node, env)
	}

	// Look up the component variable/function
	val, ok := env.Get(node.Name)
	if !ok {
		return newUndefinedComponentError(node.Name)
	}

	// If the value is a String (e.g., loaded SVG), return it directly
	// Note: For tag pairs like <Arrow>...</Arrow>, the contents are ignored for string values
	if str, isString := val.(*String); isString {
		return str
	}

	// Parse props into a dictionary and add contents
	propsDict := parseTagProps(node.Props, env)
	if isError(propsDict) {
		return propsDict
	}

	dict := propsDict.(*Dictionary)

	// Evaluate contents and add to props as "contents"
	contentsObj := evalTagContentsAsArray(node.Contents, env)
	if isError(contentsObj) {
		return contentsObj
	}

	// Create a literal expression for the contents array
	// We need to wrap the evaluated contents in an expression
	dict.Pairs["contents"] = &ast.ArrayLiteral{Elements: []ast.Expression{}}

	// Store the evaluated contents directly in the environment temporarily
	contentsEnv := NewEnclosedEnvironment(env)
	contentsEnv.Set("__tag_contents__", contentsObj)

	// Actually, let's simplify - evaluate contents as a single value
	if contentsArray, ok := contentsObj.(*Array); ok && len(contentsArray.Elements) == 1 {
		// Single item - pass directly
		dict.Pairs["contents"] = createLiteralExpression(contentsArray.Elements[0])
	} else {
		// Multiple items or empty - pass as array
		dict.Pairs["contents"] = createLiteralExpression(contentsObj)
	}

	// Check if component is null (common when import destructuring gets wrong name)
	if val == NULL || val == nil {
		return newError("cannot use '<%s/>' because '%s' is null\n   💡 Hint: '%s' may not be exported from the imported module. Check the export name matches.", node.Name, node.Name, node.Name)
	}

	// Call the function with the props dictionary
	result := applyFunction(val, []Object{dict})

	// Improve error message if function call failed
	if err, isErr := result.(*Error); isErr && strings.Contains(err.Message, "cannot call") {
		return newError("cannot use '<%s/>' because '%s' is not a function (got %s)\n   💡 Hint: Components must be functions. Check that '%s' is exported as a function.", node.Name, node.Name, val.Type(), node.Name)
	}

	return result
}

// evalTagContents evaluates tag contents and returns as a concatenated string
func evalTagContents(contents []ast.Node, env *Environment) Object {
	var result strings.Builder

	for _, node := range contents {
		obj := Eval(node, env)
		if isError(obj) {
			return obj
		}
		result.WriteString(objectToTemplateString(obj))
	}

	return &String{Value: result.String()}
}

// evalTagContentsAsArray evaluates tag contents and returns as an array
func evalTagContentsAsArray(contents []ast.Node, env *Environment) Object {
	if len(contents) == 0 {
		return NULL
	}

	elements := make([]Object, 0, len(contents))
	for _, node := range contents {
		obj := Eval(node, env)
		if isError(obj) {
			return obj
		}
		// Convert to string for consistency
		elements = append(elements, &String{Value: objectToTemplateString(obj)})
	}

	return &Array{Elements: elements}
}

// evalSQLTag handles <SQL params={...}>...</SQL> tags
func evalSQLTag(node *ast.TagPairExpression, env *Environment) Object {
	// Parse props to get params
	propsDict := parseTagProps(node.Props, env)
	if isError(propsDict) {
		return propsDict
	}

	// Get the SQL content
	sqlContent := evalTagContents(node.Contents, env)
	if isError(sqlContent) {
		return sqlContent
	}

	sqlStr, ok := sqlContent.(*String)
	if !ok {
		return newError("SQL tag content must be a string")
	}

	// Build result dictionary with sql and params
	resultPairs := map[string]ast.Expression{
		"sql": &ast.StringLiteral{Value: sqlStr.Value},
	}

	// Add params if provided
	if dict, ok := propsDict.(*Dictionary); ok {
		if paramsExpr, hasParams := dict.Pairs["params"]; hasParams {
			resultPairs["params"] = paramsExpr
		}
	}

	return &Dictionary{
		Pairs: resultPairs,
		Env:   env,
	}
}

// evalTagProps evaluates tag props string with interpolations
func evalTagProps(propsStr string, env *Environment) Object {
	var result strings.Builder

	i := 0
	for i < len(propsStr) {
		// Look for {expr}
		if propsStr[i] == '{' {
			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag props", nil)
			}

			// Extract and evaluate the expression
			exprStr := propsStr[exprStart:i]
			i++ // skip closing }

			// Parse and evaluate the expression
			l := lexer.New(exprStr)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				return newParseError("PARSE-0011", "tag prop", fmt.Errorf("%s", p.Errors()[0]))
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					return evaluated
				}
			}

			// Convert result to string
			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
		} else {
			// Regular character
			result.WriteByte(propsStr[i])
			i++
		}
	}

	return &String{Value: result.String()}
}

// createLiteralExpression creates an AST expression from an evaluated object
// This is a helper for passing evaluated values back through the AST
func createLiteralExpression(obj Object) ast.Expression {
	switch obj := obj.(type) {
	case *Integer:
		return &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", obj.Value)},
			Value: obj.Value,
		}
	case *Float:
		return &ast.FloatLiteral{
			Token: lexer.Token{Type: lexer.FLOAT, Literal: fmt.Sprintf("%g", obj.Value)},
			Value: obj.Value,
		}
	case *String:
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: obj.Value},
			Value: obj.Value,
		}
	case *Boolean:
		lit := "false"
		if obj.Value {
			lit = "true"
		}
		return &ast.Boolean{
			Token: lexer.Token{Type: lexer.IDENT, Literal: lit},
			Value: obj.Value,
		}
	case *Null:
		// Use an identifier that will evaluate to the NULL object
		return &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "__null__"},
			Value: "__null__",
		}
	case *Array:
		// For arrays, create array literal with elements
		elements := make([]ast.Expression, len(obj.Elements))
		for i, elem := range obj.Elements {
			elements[i] = createLiteralExpression(elem)
		}
		return &ast.ArrayLiteral{
			Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
			Elements: elements,
		}
	case *Dictionary:
		// For dictionaries, create dictionary literal with pairs
		pairs := make(map[string]ast.Expression)
		for key, expr := range obj.Pairs {
			// Evaluate the expression to get the value
			val := Eval(expr, obj.Env)
			pairs[key] = createLiteralExpression(val)
		}
		return &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: pairs,
		}
	default:
		// For other types, return a string literal
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: obj.Inspect()},
			Value: obj.Inspect(),
		}
	}
}

// evalStandardTag evaluates a standard (lowercase) tag as an interpolated string
func evalStandardTag(tagName string, propsStr string, env *Environment) Object {
	var result strings.Builder
	result.WriteByte('<')
	result.WriteString(tagName)

	// Process props with interpolation
	i := 0
	for i < len(propsStr) {
		// Look for {expr}
		if propsStr[i] == '{' {
			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag", nil)
			}

			// Extract and evaluate the expression
			exprStr := propsStr[exprStart:i]
			i++ // skip closing }

			// Parse and evaluate the expression
			l := lexer.New(exprStr)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				return newParseError("PARSE-0011", "tag", fmt.Errorf("%s", p.Errors()[0]))
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					return evaluated
				}
			}

			// Convert result to string (don't add quotes - they should be in the tag already)
			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
		} else {
			// Regular character
			result.WriteByte(propsStr[i])
			i++
		}
	}

	result.WriteString(" />")
	return &String{Value: result.String()}
}

// evalCustomTag evaluates a custom (uppercase) tag as a function call
func evalCustomTag(tagName string, propsStr string, env *Environment) Object {
	// Look up the variable/function
	val, ok := env.Get(tagName)
	if !ok {
		if builtin, ok := getBuiltins()[tagName]; ok {
			val = builtin
		} else {
			return newUndefinedComponentError(tagName)
		}
	}

	// Check if component is null (common when import destructuring gets wrong name)
	if val == NULL || val == nil {
		return newError("cannot use '<%s/>' because '%s' is null\n   💡 Hint: '%s' may not be exported from the imported module. Check the export name matches.", tagName, tagName, tagName)
	}

	// If the value is a String (e.g., loaded SVG), return it directly
	if str, isString := val.(*String); isString {
		return str
	}

	// Parse props into a dictionary
	props := parseTagProps(propsStr, env)
	if isError(props) {
		return props
	}

	// Call the function with the props dictionary
	result := applyFunction(val, []Object{props})

	// Improve error message if function call failed
	if err, isErr := result.(*Error); isErr && strings.Contains(err.Message, "cannot call") {
		return newError("cannot use '<%s/>' because '%s' is not a function (got %s)\n   💡 Hint: Components must be functions. Check that '%s' is exported as a function.", tagName, tagName, val.Type(), tagName)
	}

	return result
}

// parseTagProps parses tag properties into a dictionary
func parseTagProps(propsStr string, env *Environment) Object {
	pairs := make(map[string]ast.Expression)

	i := 0
	for i < len(propsStr) {
		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}
		if i >= len(propsStr) {
			break
		}

		// Read prop name
		nameStart := i
		for i < len(propsStr) && !unicode.IsSpace(rune(propsStr[i])) && propsStr[i] != '=' {
			i++
		}
		if nameStart == i {
			break
		}
		propName := propsStr[nameStart:i]

		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}

		// Check for = or standalone prop
		if i >= len(propsStr) || propsStr[i] != '=' {
			// Standalone prop (boolean)
			pairs[propName] = &ast.Boolean{Value: true}
			continue
		}

		i++ // skip =

		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}

		if i >= len(propsStr) {
			break
		}

		// Read prop value
		var valueStr string
		if propsStr[i] == '"' {
			// Quoted string - check if it contains interpolation
			i++ // skip opening quote
			valueStart := i
			hasInterpolation := false
			tempI := i
			for tempI < len(propsStr) && propsStr[tempI] != '"' {
				if propsStr[tempI] == '{' {
					hasInterpolation = true
					break
				}
				if propsStr[tempI] == '\\' {
					tempI += 2
				} else {
					tempI++
				}
			}

			if hasInterpolation {
				// The string contains {expr}, treat it as an interpolation
				// Extract content between quotes
				for i < len(propsStr) && propsStr[i] != '"' {
					if propsStr[i] == '\\' {
						i += 2
					} else {
						i++
					}
				}
				valueStr = propsStr[valueStart:i]
				if i < len(propsStr) {
					i++ // skip closing quote
				}

				// Now parse the interpolation - find the {expr}
				j := 0
				for j < len(valueStr) {
					if valueStr[j] == '{' {
						j++ // skip {
						exprStart := j
						braceCount := 1
						for j < len(valueStr) && braceCount > 0 {
							if valueStr[j] == '{' {
								braceCount++
							} else if valueStr[j] == '}' {
								braceCount--
							}
							if braceCount > 0 {
								j++
							}
						}
						exprStr := valueStr[exprStart:j]
						// Parse the expression
						l := lexer.New(exprStr)
						p := parser.New(l)
						program := p.ParseProgram()

						if len(p.Errors()) > 0 {
							return newParseError("PARSE-0011", "tag prop", fmt.Errorf("%s", p.Errors()[0]))
						}

						// Store as expression statement
						if len(program.Statements) > 0 {
							if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
								pairs[propName] = exprStmt.Expression
							}
						}
						break
					}
					j++
				}
			} else {
				// Plain string with no interpolation
				for i < len(propsStr) && propsStr[i] != '"' {
					if propsStr[i] == '\\' {
						i += 2
					} else {
						i++
					}
				}
				valueStr = propsStr[valueStart:i]
				if i < len(propsStr) {
					i++ // skip closing quote
				}
				pairs[propName] = &ast.StringLiteral{Value: valueStr}
			}
		} else if propsStr[i] == '{' {
			// Expression in braces
			i++ // skip {

			// Check for spread operator ...expr
			if i+3 <= len(propsStr) && propsStr[i] == '.' && propsStr[i+1] == '.' && propsStr[i+2] == '.' {
				i += 3 // skip ...
				exprStart := i
				braceCount := 1

				for i < len(propsStr) && braceCount > 0 {
					if propsStr[i] == '{' {
						braceCount++
					} else if propsStr[i] == '}' {
						braceCount--
					}
					if braceCount > 0 {
						i++
					}
				}

				if braceCount != 0 {
					return newParseError("PARSE-0009", "tag spread operator", nil)
				}

				exprStr := propsStr[exprStart:i]
				i++ // skip }

				// Parse and evaluate the spread expression
				l := lexer.New(exprStr)
				p := parser.New(l)
				program := p.ParseProgram()

				if len(p.Errors()) > 0 {
					return newParseError("PARSE-0011", "tag spread", fmt.Errorf("%s", p.Errors()[0]))
				}

				if len(program.Statements) > 0 {
					if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
						// Evaluate the spread expression immediately
						spreadObj := Eval(exprStmt.Expression, env)
						if isError(spreadObj) {
							return spreadObj
						}

						// If it's a dictionary, merge its properties
						if spreadDict, ok := spreadObj.(*Dictionary); ok {
							for key, value := range spreadDict.Pairs {
								pairs[key] = value
							}
						} else {
							return newError("spread operator requires a dictionary, got %s", spreadObj.Type())
						}
					}
				}
				continue
			}

			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag prop", nil)
			}

			exprStr := propsStr[exprStart:i]
			i++ // skip }

			// Parse the expression
			l := lexer.New(exprStr)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				return newParseError("PARSE-0011", "tag prop", fmt.Errorf("%s", p.Errors()[0]))
			}

			// Store as expression statement
			if len(program.Statements) > 0 {
				if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
					pairs[propName] = exprStmt.Expression
				}
			}
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// objectToTemplateString converts an object to its string representation for template interpolation
func objectToTemplateString(obj Object) string {
	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		return obj.Value
	case *Array:
		// Arrays are printed without commas in templates
		var result strings.Builder
		for _, elem := range obj.Elements {
			result.WriteString(objectToTemplateString(elem))
		}
		return result.String()
	case *Dictionary:
		// Check for special dictionary types
		if isPathDict(obj) {
			return pathDictToString(obj)
		}
		if isUrlDict(obj) {
			return urlDictToString(obj)
		}
		if isTagDict(obj) {
			return tagDictToString(obj)
		}
		if isDatetimeDict(obj) {
			return datetimeDictToString(obj)
		}
		if isDurationDict(obj) {
			return durationDictToString(obj)
		}
		if isRegexDict(obj) {
			return regexDictToString(obj)
		}
		if isFileDict(obj) {
			return fileDictToString(obj)
		}
		if isDirDict(obj) {
			return dirDictToString(obj)
		}
		if isRequestDict(obj) {
			return requestDictToString(obj)
		}
		return obj.Inspect()
	case *Null:
		return ""
	default:
		return obj.Inspect()
	}
}

// objectToPrintString converts an object to its string representation for print function
func objectToPrintString(obj Object) string {
	if obj == nil {
		return ""
	}

	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		return obj.Value
	case *Array:
		// Arrays: recursively print each element without any separators
		var result strings.Builder
		for _, elem := range obj.Elements {
			result.WriteString(objectToPrintString(elem))
		}
		return result.String()
	case *Dictionary:
		// Check for special dictionary types
		if isPathDict(obj) {
			// Convert path dictionary back to string
			return pathDictToString(obj)
		}
		if isUrlDict(obj) {
			// Convert URL dictionary back to string
			return urlDictToString(obj)
		}
		if isTagDict(obj) {
			// Convert tag dictionary to HTML string
			return tagDictToString(obj)
		}
		if isDatetimeDict(obj) {
			// Convert datetime dictionary to ISO 8601 string
			return datetimeDictToString(obj)
		}
		if isDurationDict(obj) {
			// Convert duration dictionary to human-readable string
			return durationDictToString(obj)
		}
		if isRegexDict(obj) {
			// Convert regex dictionary to /pattern/flags format
			return regexDictToString(obj)
		}
		if isFileDict(obj) {
			// Convert file dictionary to path string
			return fileDictToString(obj)
		}
		if isDirDict(obj) {
			// Convert dir dictionary to path string with trailing slash
			return dirDictToString(obj)
		}
		if isRequestDict(obj) {
			// Convert request dictionary to METHOD URL format
			return requestDictToString(obj)
		}
		return obj.Inspect()
	case *Null:
		return ""
	default:
		return obj.Inspect()
	}
}

// ObjectToPrintString is the exported version for use outside the package
func ObjectToPrintString(obj Object) string {
	return objectToPrintString(obj)
}

// objectToDebugString converts an object to its debug string representation
func objectToDebugString(obj Object) string {
	switch obj := obj.(type) {
	case *Integer:
		return strconv.FormatInt(obj.Value, 10)
	case *Float:
		return fmt.Sprintf("%g", obj.Value)
	case *Boolean:
		if obj.Value {
			return "true"
		}
		return "false"
	case *String:
		// Strings are wrapped in quotes for debug output
		return fmt.Sprintf("\"%s\"", obj.Value)
	case *Array:
		// Arrays: recursively debug print each element with separators, wrapped in brackets
		var result strings.Builder
		result.WriteString("[")
		for i, elem := range obj.Elements {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(objectToDebugString(elem))
		}
		result.WriteString("]")
		return result.String()
	case *Null:
		return "null"
	default:
		return obj.Inspect()
	}
}

// evalConcatExpression handles the ++ operator for array concatenation
func evalConcatExpression(left, right Object) Object {
	// Handle dictionary concatenation
	if left.Type() == DICTIONARY_OBJ && right.Type() == DICTIONARY_OBJ {
		leftDict := left.(*Dictionary)
		rightDict := right.(*Dictionary)

		// Create new dictionary with merged pairs
		merged := &Dictionary{
			Pairs: make(map[string]ast.Expression),
			Env:   leftDict.Env, // Use left dict's environment
		}

		// Copy left dictionary pairs
		for k, v := range leftDict.Pairs {
			merged.Pairs[k] = v
		}

		// Copy right dictionary pairs (overwrites left if keys match)
		for k, v := range rightDict.Pairs {
			merged.Pairs[k] = v
		}

		return merged
	}

	// Convert single values to arrays
	var leftElements, rightElements []Object

	switch l := left.(type) {
	case *Array:
		leftElements = l.Elements
	default:
		leftElements = []Object{left}
	}

	switch r := right.(type) {
	case *Array:
		rightElements = r.Elements
	default:
		rightElements = []Object{right}
	}

	// Concatenate the arrays
	result := make([]Object, 0, len(leftElements)+len(rightElements))
	result = append(result, leftElements...)
	result = append(result, rightElements...)

	return &Array{Elements: result}
}

// evalInExpression handles the 'in' membership operator
// Returns true if left is contained in right (array, dictionary key, or substring)
func evalInExpression(tok lexer.Token, left, right Object) Object {
	switch r := right.(type) {
	case *Array:
		// Check if left is an element of the array
		for _, elem := range r.Elements {
			if objectsEqual(left, elem) {
				return TRUE
			}
		}
		return FALSE
	case *Dictionary:
		// Check if left is a key in the dictionary
		if left.Type() != STRING_OBJ {
			return newErrorWithPos(tok, "dictionary key must be a string, got %s", left.Type())
		}
		key := left.(*String).Value
		if _, ok := r.Pairs[key]; ok {
			return TRUE
		}
		return FALSE
	case *String:
		// Check if left is a substring of right
		if left.Type() != STRING_OBJ {
			return newErrorWithPos(tok, "substring must be a string, got %s", left.Type())
		}
		substring := left.(*String).Value
		if strings.Contains(r.Value, substring) {
			return TRUE
		}
		return FALSE
	default:
		return newErrorWithPos(tok, "'in' operator requires array, dictionary, or string on right side, got %s", right.Type())
	}
}

// evalIndexExpression handles array and string indexing
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalIndexExpression(tok lexer.Token, left, index Object, optional bool) Object {
	// Handle response typed dictionary - unwrap __data for indexing
	if dict, ok := left.(*Dictionary); ok && isResponseDict(dict) {
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			left = Eval(dataExpr, dict.Env)
			if isError(left) {
				return left
			}
		}
	}

	switch {
	case left.Type() == ARRAY_OBJ && index.Type() == INTEGER_OBJ:
		return evalArrayIndexExpression(tok, left, index, optional)
	case left.Type() == STRING_OBJ && index.Type() == INTEGER_OBJ:
		return evalStringIndexExpression(tok, left, index, optional)
	case left.Type() == DICTIONARY_OBJ && index.Type() == STRING_OBJ:
		return evalDictionaryIndexExpression(left, index, optional)
	default:
		return newIndexTypeError(tok, left.Type(), index.Type())
	}
}

// evalArrayIndexExpression handles array indexing with support for negative indices
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalArrayIndexExpression(tok lexer.Token, array, index Object, optional bool) Object {
	arrayObject := array.(*Array)
	idx := index.(*Integer).Value
	max := int64(len(arrayObject.Elements))

	// Handle negative indices
	if idx < 0 {
		idx = max + idx
	}

	if idx < 0 || idx >= max {
		if optional {
			return NULL
		}
		return newErrorWithPos(tok, "index out of range: %d", index.(*Integer).Value)
	}

	return arrayObject.Elements[idx]
}

// evalStringIndexExpression handles string indexing with support for negative indices
// If optional is true, returns NULL instead of error for out-of-bounds access
func evalStringIndexExpression(tok lexer.Token, str, index Object, optional bool) Object {
	stringObject := str.(*String)
	idx := index.(*Integer).Value
	max := int64(len(stringObject.Value))

	// Handle negative indices
	if idx < 0 {
		idx = max + idx
	}

	if idx < 0 || idx >= max {
		if optional {
			return NULL
		}
		return newErrorWithPos(tok, "index out of range: %d", index.(*Integer).Value)
	}

	return &String{Value: string(stringObject.Value[idx])}
}

// evalSliceExpression handles array and string slicing
func evalSliceExpression(left, start, end Object) Object {
	switch left.Type() {
	case ARRAY_OBJ:
		return evalArraySliceExpression(left, start, end)
	case STRING_OBJ:
		return evalStringSliceExpression(left, start, end)
	default:
		return newSliceTypeError(left.Type())
	}
}

// evalArraySliceExpression handles array slicing
func evalArraySliceExpression(array, start, end Object) Object {
	arrayObject := array.(*Array)
	max := int64(len(arrayObject.Elements))

	var startIdx, endIdx int64

	// Determine start index
	if start == nil {
		startIdx = 0
	} else if start.Type() == INTEGER_OBJ {
		startIdx = start.(*Integer).Value
		if startIdx < 0 {
			startIdx = max + startIdx
		}
	} else {
		return newSliceIndexTypeError("start", string(start.Type()))
	}

	// Determine end index
	if end == nil {
		endIdx = max
	} else if end.Type() == INTEGER_OBJ {
		endIdx = end.(*Integer).Value
		if endIdx < 0 {
			endIdx = max + endIdx
		}
	} else {
		return newSliceIndexTypeError("end", string(end.Type()))
	}

	// Validate and clamp indices
	if startIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": startIdx, "Length": max})
	}
	if endIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": endIdx, "Length": max})
	}
	if startIdx > endIdx {
		return newIndexError("INDEX-0003", map[string]any{"Start": startIdx, "End": endIdx})
	}

	// Clamp to array bounds (allow slicing beyond length)
	if startIdx > max {
		startIdx = max
	}
	if endIdx > max {
		endIdx = max
	}

	// Create the slice
	return &Array{Elements: arrayObject.Elements[startIdx:endIdx]}
}

// evalStringSliceExpression handles string slicing
func evalStringSliceExpression(str, start, end Object) Object {
	stringObject := str.(*String)
	max := int64(len(stringObject.Value))

	var startIdx, endIdx int64

	// Determine start index
	if start == nil {
		startIdx = 0
	} else if start.Type() == INTEGER_OBJ {
		startIdx = start.(*Integer).Value
		if startIdx < 0 {
			startIdx = max + startIdx
		}
	} else {
		return newSliceIndexTypeError("start", string(start.Type()))
	}

	// Determine end index
	if end == nil {
		endIdx = max
	} else if end.Type() == INTEGER_OBJ {
		endIdx = end.(*Integer).Value
		if endIdx < 0 {
			endIdx = max + endIdx
		}
	} else {
		return newSliceIndexTypeError("end", string(end.Type()))
	}

	// Validate and clamp indices
	if startIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": startIdx, "Length": max})
	}
	if endIdx < 0 {
		return newIndexError("INDEX-0001", map[string]any{"Index": endIdx, "Length": max})
	}
	if startIdx > endIdx {
		return newIndexError("INDEX-0003", map[string]any{"Start": startIdx, "End": endIdx})
	}

	// Clamp to string bounds (allow slicing beyond length)
	if startIdx > max {
		startIdx = max
	}
	if endIdx > max {
		endIdx = max
	}

	// Create the slice
	return &String{Value: stringObject.Value[startIdx:endIdx]}
}

// evalDictionaryLiteral evaluates dictionary literals
func evalDictionaryLiteral(node *ast.DictionaryLiteral, env *Environment) Object {
	// Evaluate all values eagerly and store them as ObjectLiteralExpressions
	// This ensures values like method calls (t.count()) are evaluated at creation time
	pairs := make(map[string]ast.Expression)
	for key, expr := range node.Pairs {
		value := Eval(expr, env)
		if isError(value) {
			return value
		}
		// Convert the evaluated value back to an expression for storage
		pairs[key] = objectToExpression(value)
	}

	dict := &Dictionary{
		Pairs: pairs,
		Env:   env,
	}
	return dict
}

// evalDotExpression evaluates dot notation access (dict.key)
func evalDotExpression(node *ast.DotExpression, env *Environment) Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	// Null propagation: property access on null returns null
	if left == NULL || left == nil {
		return NULL
	}

	// Handle Table property access
	if table, ok := left.(*Table); ok {
		return EvalTableProperty(table, node.Key)
	}

	// Handle SFTP file handles for format accessors
	if sftpHandle, ok := left.(*SFTPFileHandle); ok {
		// Format accessors: .json, .text, .csv, .lines, .bytes, .file
		validFormats := map[string]bool{
			"json": true, "text": true, "csv": true,
			"lines": true, "bytes": true, "file": true,
		}
		if validFormats[node.Key] {
			return &SFTPFileHandle{
				Connection: sftpHandle.Connection,
				Path:       sftpHandle.Path,
				Format:     node.Key,
				Options:    sftpHandle.Options,
			}
		}
		// Check for directory accessor
		if node.Key == "dir" {
			// Return a special dict representing dir accessor
			// This will be handled by evalSFTPFileHandleMethod
			return &SFTPFileHandle{
				Connection: sftpHandle.Connection,
				Path:       sftpHandle.Path,
				Format:     "dir",
				Options:    sftpHandle.Options,
			}
		}
		return newErrorWithPos(node.Token, "unknown property for SFTP file handle: %s", node.Key)
	}

	// Handle Dictionary (including special types like datetime, path, url)
	dict, ok := left.(*Dictionary)
	if !ok {
		return newErrorWithPos(node.Token, "dot notation can only be used on dictionaries, got %s", left.Type())
	}

	// Handle HTTP method accessors for request dictionaries
	if isRequestDict(dict) {
		httpMethods := map[string]string{
			"get": "GET", "post": "POST", "put": "PUT",
			"patch": "PATCH", "delete": "DELETE",
		}
		if method, ok := httpMethods[node.Key]; ok {
			return setRequestMethod(dict, method, env)
		}
	}

	// Handle response typed dictionary auto-unwrap for data access
	if isResponseDict(dict) {
		// Auto-unwrap __data for property access
		if dataExpr, ok := dict.Pairs["__data"]; ok {
			dataObj := Eval(dataExpr, dict.Env)
			if dataDict, ok := dataObj.(*Dictionary); ok {
				// Try to get the property from __data
				if expr, ok := dataDict.Pairs[node.Key]; ok {
					return Eval(expr, dataDict.Env)
				}
			}
		}
		// Fall through to normal dict access for __type, __format, etc.
	}

	// Check for computed properties on special dictionary types
	if isPathDict(dict) {
		if computed := evalPathComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}
	if isUrlDict(dict) {
		if computed := evalUrlComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}
	if isFileDict(dict) {
		if computed := evalFileComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}
	if isDirDict(dict) {
		if computed := evalDirComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}
	if isDatetimeDict(dict) {
		if computed := evalDatetimeComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}

	// Get the expression from the dictionary
	expr, ok := dict.Pairs[node.Key]
	if !ok {
		return NULL
	}

	// Create a new environment with 'this' bound to the dictionary
	dictEnv := NewEnclosedEnvironment(dict.Env)
	dictEnv.Set("this", dict)

	// Evaluate the expression in the dictionary's environment
	return Eval(expr, dictEnv)
}

// evalReadStatement evaluates the <== operator to read file content
func evalReadStatement(node *ast.ReadStatement, env *Environment) Object {
	// Check if we're using dict pattern destructuring with error capture pattern
	// Only use {data, error} wrapping if the pattern contains "data" or "error" keys
	useErrorCapture := node.DictPattern != nil && isErrorCapturePattern(node.DictPattern)

	// Evaluate the source expression (should be a file or dir handle)
	source := Eval(node.Source, env)
	if isError(source) {
		if useErrorCapture {
			// Wrap the error in {data: null, error: "message"} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, source.(*Error).Message, env), env, node.IsLet, false)
		}
		return source
	}

	// The source should be a file or directory dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		errMsg := fmt.Sprintf("read operator <== requires a file or directory handle, got %s", source.Type())
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
		}
		return newError("read operator <== requires a file or directory handle, got %s", source.Type())
	}

	var content Object
	var readErr *Error

	if isDirDict(sourceDict) {
		// Read directory contents
		pathStr := getFilePathString(sourceDict, env)
		if pathStr == "" {
			errMsg := "directory handle has no valid path"
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
			}
			return newError("directory handle has no valid path")
		}
		content = readDirContents(pathStr, env)
		if isError(content) {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, content.(*Error).Message, env), env, node.IsLet, false)
			}
			return content
		}
	} else if isFileDict(sourceDict) {
		// Read file content based on format
		content, readErr = readFileContent(sourceDict, env)
		if readErr != nil {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeDataErrorDict(NULL, readErr.Message, env), env, node.IsLet, false)
			}
			return readErr
		}
	} else {
		errMsg := "read operator <== requires a file or directory handle, got dictionary"
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(NULL, errMsg, env), env, node.IsLet, false)
		}
		return newError("read operator <== requires a file or directory handle, got dictionary")
	}

	// Assign to the target variable(s)
	if node.DictPattern != nil {
		if useErrorCapture {
			// Wrap successful result in {data: ..., error: null} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeDataErrorDict(content, "", env), env, node.IsLet, false)
		}
		// Normal dict destructuring - extract keys directly from content
		return evalDictDestructuringAssignment(node.DictPattern, content, env, node.IsLet, false)
	}

	if len(node.Names) > 0 {
		return evalDestructuringAssignment(node.Names, content, env, node.IsLet, false)
	}

	// Single assignment
	if node.Name != nil && node.Name.Value != "_" {
		if node.IsLet {
			env.SetLet(node.Name.Value, content)
		} else {
			env.Update(node.Name.Value, content)
		}
	}

	return content
}

// evalFetchStatement evaluates the <=/= operator to fetch URL content
func evalFetchStatement(node *ast.FetchStatement, env *Environment) Object {
	// Check if we're using dict pattern destructuring with error capture pattern
	useErrorCapture := node.DictPattern != nil && isErrorCapturePattern(node.DictPattern)

	// Evaluate the source expression (should be a request handle, URL, or SFTP file handle)
	source := Eval(node.Source, env)
	if isError(source) {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, source.(*Error).Message, 0, nil, env), env, node.IsLet, false)
		}
		return source
	}

	// Check if it's an SFTP file handle
	if sftpHandle, ok := source.(*SFTPFileHandle); ok {
		content, err := evalSFTPRead(sftpHandle, env)
		if err != nil {
			if useErrorCapture {
				return evalDictDestructuringAssignment(node.DictPattern,
					makeSFTPResponseDict(NULL, err.(*Error).Message, env), env, node.IsLet, false)
			}
			return err
		}

		// Assign to the target variable(s)
		if node.DictPattern != nil {
			if useErrorCapture {
				// Wrap successful result in {data: ..., error: null} format
				return evalDictDestructuringAssignment(node.DictPattern,
					makeSFTPResponseDict(content, "", env), env, node.IsLet, false)
			}
			// Regular dict destructuring
			return evalDictDestructuringAssignment(node.DictPattern, content, env, node.IsLet, false)
		}

		// Simple assignment
		if len(node.Names) > 0 {
			return evalDestructuringAssignment(node.Names, content, env, node.IsLet, false)
		}

		return content
	}

	// The source should be a request dictionary (from JSON(@url), etc.) or a URL dictionary
	sourceDict, ok := source.(*Dictionary)
	if !ok {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, fmt.Sprintf("fetch operator <=/= requires a request or URL handle, got %s", source.Type()), 0, nil, env), env, node.IsLet, false)
		}
		return newError("fetch operator <=/= requires a request or URL handle, got %s", source.Type())
	}

	var reqDict *Dictionary

	if isRequestDict(sourceDict) {
		reqDict = sourceDict
	} else if isUrlDict(sourceDict) {
		// Wrap URL in a request dictionary with default format (text)
		reqDict = urlToRequestDict(sourceDict, "text", nil, env)
	} else {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, "fetch operator <=/= requires a request or URL handle, got dictionary", 0, nil, env), env, node.IsLet, false)
		}
		return newError("fetch operator <=/= requires a request or URL handle, got dictionary")
	}

	// Fetch URL content with full response info
	info := fetchUrlContentFull(reqDict, env)

	// Handle errors with legacy error capture pattern
	if info.Error != "" {
		if useErrorCapture {
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(NULL, info.Error, info.StatusCode, info.Headers, env), env, node.IsLet, false)
		}
		return newError("%s", info.Error)
	}

	// Create response typed dictionary
	responseDict := makeResponseTypedDict(
		info.Content,
		info.Format,
		info.StatusCode,
		info.StatusText,
		info.OK,
		info.FinalURL,
		info.Headers,
		"",
		env,
	)

	// Assign to the target variable(s)
	if node.DictPattern != nil {
		if useErrorCapture {
			// Wrap successful result in {data: ..., error: null, status: ..., headers: ...} format
			return evalDictDestructuringAssignment(node.DictPattern,
				makeFetchResponseDict(info.Content, "", info.StatusCode, info.Headers, env), env, node.IsLet, false)
		}
		// Normal dict destructuring - extract keys directly from __data
		return evalDictDestructuringAssignment(node.DictPattern, info.Content, env, node.IsLet, false)
	}

	if len(node.Names) > 0 {
		return evalDestructuringAssignment(node.Names, responseDict, env, node.IsLet, false)
	}

	// Single assignment
	if node.Name != nil && node.Name.Value != "_" {
		if node.IsLet {
			env.SetLet(node.Name.Value, responseDict)
		} else {
			env.Update(node.Name.Value, responseDict)
		}
	}

	return responseDict
}

// isRequestDict checks if a dictionary is a request handle by looking for __type field
func isRequestDict(dict *Dictionary) bool {
	typeExpr, ok := dict.Pairs["__type"]
	if !ok {
		return false
	}
	if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
		return strLit.Value == "request"
	}
	return false
}

// isResponseDict checks if a dictionary is a response typed dictionary
func isResponseDict(dict *Dictionary) bool {
	typeExpr, ok := dict.Pairs["__type"]
	if !ok {
		return false
	}
	if strLit, ok := typeExpr.(*ast.StringLiteral); ok {
		return strLit.Value == "response"
	}
	return false
}

// setRequestMethod clones a request dict with a new HTTP method
func setRequestMethod(dict *Dictionary, method string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Copy all existing pairs
	for key, expr := range dict.Pairs {
		pairs[key] = expr
	}

	// Set the method
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// parseURLToDict parses a URL string into a URL dictionary, returning nil on error
func parseURLToDict(urlStr string, env *Environment) *Dictionary {
	dict, err := parseUrlString(urlStr, env)
	if err != nil {
		return nil
	}
	return dict
}

// urlToRequestDict wraps a URL dictionary in a request dictionary
func urlToRequestDict(urlDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "request"},
		Value: "request",
	}

	// Copy URL fields
	for key, expr := range urlDict.Pairs {
		pairs["_url_"+key] = expr
	}

	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "GET"},
		Value: "GET",
	}

	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Add empty headers dict
	pairs["headers"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: make(map[string]ast.Expression),
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// requestToDict creates a request dictionary from a URL dictionary with format and options
func requestToDict(urlDict *Dictionary, format string, options *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "request"},
		Value: "request",
	}

	// Copy URL fields with prefix
	for key, expr := range urlDict.Pairs {
		pairs["_url_"+key] = expr
	}

	pairs["format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Default method is GET
	method := "GET"
	if options != nil {
		if methodExpr, ok := options.Pairs["method"]; ok {
			methodObj := Eval(methodExpr, env)
			if methodStr, ok := methodObj.(*String); ok {
				method = strings.ToUpper(methodStr.Value)
			}
		}
	}
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	// Copy headers from options
	if options != nil {
		if headersExpr, ok := options.Pairs["headers"]; ok {
			pairs["headers"] = headersExpr
		} else {
			pairs["headers"] = &ast.DictionaryLiteral{
				Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
				Pairs: make(map[string]ast.Expression),
			}
		}
		// Copy body from options
		if bodyExpr, ok := options.Pairs["body"]; ok {
			pairs["body"] = bodyExpr
		}
		// Copy timeout from options
		if timeoutExpr, ok := options.Pairs["timeout"]; ok {
			pairs["timeout"] = timeoutExpr
		}
	} else {
		pairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// makeResponseTypedDict creates a response typed dictionary with __type, __format, __data, __response
// This is the new response structure that auto-unwraps for iteration/indexing
func makeResponseTypedDict(data Object, format string, statusCode int64, statusText string, ok bool, urlStr string, headers *Dictionary, errorMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set __type
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "response"},
		Value: "response",
	}

	// Set __format
	pairs["__format"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: format},
		Value: format,
	}

	// Set __data (the actual fetched data, or null on error)
	if data != nil {
		pairs["__data"] = &ast.ObjectLiteralExpression{Obj: data}
	} else {
		pairs["__data"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	// Build __response dictionary
	responsePairs := make(map[string]ast.Expression)

	responsePairs["status"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", statusCode)},
		Value: statusCode,
	}

	responsePairs["statusText"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: statusText},
		Value: statusText,
	}

	responsePairs["ok"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: ok}}

	// URL as a URL dictionary
	if urlStr != "" {
		urlDict := parseURLToDict(urlStr, env)
		if urlDict != nil {
			responsePairs["url"] = &ast.ObjectLiteralExpression{Obj: urlDict}
		} else {
			responsePairs["url"] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: urlStr},
				Value: urlStr,
			}
		}
	} else {
		responsePairs["url"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	// Headers
	if headers != nil {
		responsePairs["headers"] = &ast.ObjectLiteralExpression{Obj: headers}
	} else {
		responsePairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	// Error
	if errorMsg == "" {
		responsePairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		responsePairs["error"] = &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: errorMsg},
			Value: errorMsg,
		}
	}

	pairs["__response"] = &ast.DictionaryLiteral{
		Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
		Pairs: responsePairs,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// makeFetchResponseDict creates a {data: ..., error: ..., status: ..., headers: ...} dictionary
// This is the legacy format for error capture pattern
func makeFetchResponseDict(data Object, errorMsg string, status int64, headers *Dictionary, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set data field
	pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}

	// Set error field
	if errorMsg == "" {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errorMsg}}
	}

	// Set status field
	pairs["status"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", status)},
		Value: status,
	}

	// Set headers field
	if headers != nil {
		pairs["headers"] = &ast.ObjectLiteralExpression{Obj: headers}
	} else {
		pairs["headers"] = &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: make(map[string]ast.Expression),
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// getRequestUrlString extracts the URL string from a request dictionary
func getRequestUrlString(dict *Dictionary, env *Environment) string {
	var result strings.Builder

	// Get scheme
	schemeExpr, ok := dict.Pairs["_url_scheme"]
	if !ok {
		return ""
	}
	schemeObj := Eval(schemeExpr, env)
	schemeStr, ok := schemeObj.(*String)
	if !ok {
		return ""
	}
	result.WriteString(schemeStr.Value)
	result.WriteString("://")

	// Get host
	hostExpr, ok := dict.Pairs["_url_host"]
	if !ok {
		return ""
	}
	hostObj := Eval(hostExpr, env)
	hostStr, ok := hostObj.(*String)
	if !ok {
		return ""
	}
	result.WriteString(hostStr.Value)

	// Get port (if non-zero)
	if portExpr, ok := dict.Pairs["_url_port"]; ok {
		portObj := Eval(portExpr, env)
		if portInt, ok := portObj.(*Integer); ok && portInt.Value != 0 {
			result.WriteString(fmt.Sprintf(":%d", portInt.Value))
		}
	}

	// Get path
	if pathExpr, ok := dict.Pairs["_url_path"]; ok {
		pathObj := Eval(pathExpr, env)
		if pathArr, ok := pathObj.(*Array); ok {
			for _, elem := range pathArr.Elements {
				result.WriteString("/")
				if str, ok := elem.(*String); ok {
					result.WriteString(str.Value)
				}
			}
		}
	}

	// Get query
	if queryExpr, ok := dict.Pairs["_url_query"]; ok {
		queryObj := Eval(queryExpr, env)
		if queryDict, ok := queryObj.(*Dictionary); ok && len(queryDict.Pairs) > 0 {
			result.WriteString("?")
			first := true
			for key, valExpr := range queryDict.Pairs {
				if !first {
					result.WriteString("&")
				}
				first = false
				valObj := Eval(valExpr, env)
				result.WriteString(key)
				result.WriteString("=")
				switch v := valObj.(type) {
				case *String:
					result.WriteString(v.Value)
				case *Integer:
					result.WriteString(fmt.Sprintf("%d", v.Value))
				default:
					result.WriteString(valObj.Inspect())
				}
			}
		}
	}

	return result.String()
}

// HTTPResponseInfo holds all information about an HTTP response
type HTTPResponseInfo struct {
	Content    Object
	StatusCode int64
	StatusText string
	OK         bool
	FinalURL   string
	Headers    *Dictionary
	Format     string
	Error      string
}

// fetchUrlContentFull fetches content from a URL and returns full response info
func fetchUrlContentFull(reqDict *Dictionary, env *Environment) *HTTPResponseInfo {
	info := &HTTPResponseInfo{}

	// Get the URL string
	urlStr := getRequestUrlString(reqDict, env)
	if urlStr == "" {
		info.Error = "request handle has no valid URL"
		return info
	}
	info.FinalURL = urlStr

	// Get method
	method := "GET"
	if methodExpr, ok := reqDict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, env)
		if methodStr, ok := methodObj.(*String); ok {
			method = strings.ToUpper(methodStr.Value)
		}
	}

	// Get format
	format := "text"
	if formatExpr, ok := reqDict.Pairs["format"]; ok {
		formatObj := Eval(formatExpr, env)
		if formatStr, ok := formatObj.(*String); ok {
			format = formatStr.Value
		}
	}
	info.Format = format

	// Get timeout (default 30 seconds)
	timeout := 30 * time.Second
	if timeoutExpr, ok := reqDict.Pairs["timeout"]; ok {
		timeoutObj := Eval(timeoutExpr, env)
		if timeoutInt, ok := timeoutObj.(*Integer); ok {
			timeout = time.Duration(timeoutInt.Value) * time.Millisecond
		}
	}

	// Prepare request body
	var bodyReader io.Reader
	if bodyExpr, ok := reqDict.Pairs["body"]; ok {
		bodyObj := Eval(bodyExpr, env)
		if bodyObj != nil && bodyObj != NULL {
			switch v := bodyObj.(type) {
			case *String:
				bodyReader = strings.NewReader(v.Value)
			case *Dictionary, *Array:
				jsonBytes, err := encodeJSON(bodyObj)
				if err != nil {
					info.Error = fmt.Sprintf("failed to encode request body: %s", err.Error())
					return info
				}
				bodyReader = bytes.NewReader(jsonBytes)
			default:
				bodyReader = strings.NewReader(bodyObj.Inspect())
			}
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		info.Error = fmt.Sprintf("failed to create request: %s", err.Error())
		return info
	}

	// Set headers
	if headersExpr, ok := reqDict.Pairs["headers"]; ok {
		headersObj := Eval(headersExpr, env)
		if headersDict, ok := headersObj.(*Dictionary); ok {
			for key, valExpr := range headersDict.Pairs {
				valObj := Eval(valExpr, env)
				if valStr, ok := valObj.(*String); ok {
					req.Header.Set(key, valStr.Value)
				}
			}
		}
	}

	// Set default Content-Type for POST/PUT with body
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		info.Error = fmt.Sprintf("fetch failed: %s", err.Error())
		return info
	}
	defer resp.Body.Close()

	// Capture response info
	info.StatusCode = int64(resp.StatusCode)
	info.StatusText = resp.Status // e.g., "200 OK" or "404 Not Found"
	info.OK = resp.StatusCode >= 200 && resp.StatusCode < 300
	info.FinalURL = resp.Request.URL.String() // Final URL after redirects

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Error = fmt.Sprintf("failed to read response: %s", err.Error())
		return info
	}

	// Convert response headers to dictionary
	respHeaders := &Dictionary{Pairs: make(map[string]ast.Expression), Env: env}
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders.Pairs[strings.ToLower(key)] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: values[0]},
				Value: values[0],
			}
		}
	}
	info.Headers = respHeaders

	// Decode based on format
	var content Object
	var parseErr *Error

	switch format {
	case "text":
		content = &String{Value: string(data)}

	case "json":
		content, parseErr = parseJSON(string(data))
		if parseErr != nil {
			info.Error = parseErr.Message
			return info
		}

	case "yaml":
		content, parseErr = parseYAML(string(data))
		if parseErr != nil {
			info.Error = parseErr.Message
			return info
		}

	case "lines":
		lines := strings.Split(string(data), "\n")
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		content = &Array{Elements: elements}

	case "bytes":
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		content = &Array{Elements: elements}

	default:
		content = &String{Value: string(data)}
	}

	info.Content = content
	return info
}

// fetchUrlContent fetches content from a URL based on the request configuration
// (Legacy function - kept for backward compatibility with error capture pattern)
func fetchUrlContent(reqDict *Dictionary, env *Environment) (Object, int64, *Dictionary, *Error) {
	// Get the URL string
	urlStr := getRequestUrlString(reqDict, env)
	if urlStr == "" {
		return nil, 0, nil, newError("request handle has no valid URL")
	}

	// Get method
	method := "GET"
	if methodExpr, ok := reqDict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, env)
		if methodStr, ok := methodObj.(*String); ok {
			method = strings.ToUpper(methodStr.Value)
		}
	}

	// Get format
	format := "text"
	if formatExpr, ok := reqDict.Pairs["format"]; ok {
		formatObj := Eval(formatExpr, env)
		if formatStr, ok := formatObj.(*String); ok {
			format = formatStr.Value
		}
	}

	// Get timeout (default 30 seconds)
	timeout := 30 * time.Second
	if timeoutExpr, ok := reqDict.Pairs["timeout"]; ok {
		timeoutObj := Eval(timeoutExpr, env)
		if timeoutInt, ok := timeoutObj.(*Integer); ok {
			timeout = time.Duration(timeoutInt.Value) * time.Millisecond
		}
	}

	// Prepare request body
	var bodyReader io.Reader
	if bodyExpr, ok := reqDict.Pairs["body"]; ok {
		bodyObj := Eval(bodyExpr, env)
		if bodyObj != nil && bodyObj != NULL {
			// Encode body based on content type (default to JSON for objects)
			switch v := bodyObj.(type) {
			case *String:
				bodyReader = strings.NewReader(v.Value)
			case *Dictionary, *Array:
				jsonBytes, err := encodeJSON(bodyObj)
				if err != nil {
					return nil, 0, nil, newError("failed to encode request body: %s", err.Error())
				}
				bodyReader = bytes.NewReader(jsonBytes)
			default:
				bodyReader = strings.NewReader(bodyObj.Inspect())
			}
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return nil, 0, nil, newError("failed to create request: %s", err.Error())
	}

	// Set headers
	if headersExpr, ok := reqDict.Pairs["headers"]; ok {
		headersObj := Eval(headersExpr, env)
		if headersDict, ok := headersObj.(*Dictionary); ok {
			for key, valExpr := range headersDict.Pairs {
				valObj := Eval(valExpr, env)
				if valStr, ok := valObj.(*String); ok {
					req.Header.Set(key, valStr.Value)
				}
			}
		}
	}

	// Set default Content-Type for POST/PUT with body
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, newError("fetch failed: %s", err.Error())
	}
	defer resp.Body.Close()

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, int64(resp.StatusCode), nil, newError("failed to read response: %s", err.Error())
	}

	// Convert response headers to dictionary
	respHeaders := &Dictionary{Pairs: make(map[string]ast.Expression), Env: env}
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders.Pairs[key] = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: values[0]},
				Value: values[0],
			}
		}
	}

	// Decode based on format
	var content Object
	var parseErr *Error

	switch format {
	case "text":
		content = &String{Value: string(data)}

	case "json":
		content, parseErr = parseJSON(string(data))
		if parseErr != nil {
			return nil, int64(resp.StatusCode), respHeaders, parseErr
		}

	case "yaml":
		content, parseErr = parseYAML(string(data))
		if parseErr != nil {
			return nil, int64(resp.StatusCode), respHeaders, parseErr
		}

	case "lines":
		lines := strings.Split(string(data), "\n")
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		content = &Array{Elements: elements}

	case "bytes":
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		content = &Array{Elements: elements}

	default:
		// Default to text
		content = &String{Value: string(data)}
	}

	return content, int64(resp.StatusCode), respHeaders, nil
}

// isErrorCapturePattern checks if a dict destructuring pattern contains "data" or "error" keys
// which indicates the user wants to use the error capture pattern
func isErrorCapturePattern(pattern *ast.DictDestructuringPattern) bool {
	for _, key := range pattern.Keys {
		if key.Key != nil {
			keyName := key.Key.Value
			if keyName == "data" || keyName == "error" {
				return true
			}
		}
	}
	return false
}

// makeDataErrorDict creates a {data: ..., error: ...} dictionary for error capture pattern
func makeDataErrorDict(data Object, errorMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Set data field
	pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}

	// Set error field
	if errorMsg == "" {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	} else {
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errorMsg}}
	}

	return &Dictionary{Pairs: pairs}
}

// readFileContent reads the content of a file based on its format
func readFileContent(fileDict *Dictionary, env *Environment) (Object, *Error) {
	// Check if this is a stdio stream
	var data []byte
	var pathStr string

	if stdioExpr, ok := fileDict.Pairs["__stdio"]; ok {
		stdioObj := Eval(stdioExpr, env)
		if stdioStr, ok := stdioObj.(*String); ok {
			switch stdioStr.Value {
			case "stdin", "stdio":
				// Read from stdin (@stdin or @- for reads)
				var readErr error
				data, readErr = io.ReadAll(os.Stdin)
				if readErr != nil {
					return nil, newError("failed to read from stdin: %s", readErr.Error())
				}
				pathStr = "-"
			case "stdout", "stderr":
				return nil, newError("cannot read from %s", stdioStr.Value)
			default:
				return nil, newError("unknown stdio stream: %s", stdioStr.Value)
			}
		}
	} else {
		// Get the path from the file dictionary
		pathStr = getFilePathString(fileDict, env)
		if pathStr == "" {
			return nil, newError("file handle has no valid path")
		}

		// Resolve the path relative to the current file (or root path for ~/ paths)
		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return nil, newIOError("IO-0007", pathStr, pathErr)
		}
		pathStr = absPath

		// Security check
		if err := env.checkPathAccess(pathStr, "read"); err != nil {
			return nil, newSecurityError("read", err)
		}

		// Read the raw file content
		var readErr error
		data, readErr = os.ReadFile(pathStr)
		if readErr != nil {
			return nil, newIOError("IO-0003", pathStr, readErr)
		}
	}

	// Get the format
	formatExpr, hasFormat := fileDict.Pairs["format"]
	if !hasFormat {
		return nil, newError("file handle has no format specified")
	}
	formatObj := Eval(formatExpr, env)
	if isError(formatObj) {
		return nil, formatObj.(*Error)
	}
	formatStr, ok := formatObj.(*String)
	if !ok {
		return nil, newError("file format must be a string, got %s", formatObj.Type())
	}

	// Decode based on format
	switch formatStr.Value {
	case "text":
		return &String{Value: string(data)}, nil

	case "bytes":
		// Return as array of integers
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		return &Array{Elements: elements}, nil

	case "lines":
		// Split into lines
		content := string(data)
		lines := strings.Split(content, "\n")
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		return &Array{Elements: elements}, nil

	case "json":
		// Parse JSON
		content := string(data)
		return parseJSON(content)

	case "yaml":
		// Parse YAML
		content := string(data)
		return parseYAML(content)

	case "csv":
		// Parse CSV with header
		return parseCSV(data, true)

	case "csv-noheader":
		// Parse CSV without header
		return parseCSV(data, false)

	case "svg":
		// Return SVG content with XML prolog stripped
		content := string(data)
		return &String{Value: stripXMLProlog(content)}, nil

	case "md", "markdown":
		// Parse markdown with optional YAML frontmatter
		content := string(data)
		return parseMarkdown(content, env)

	default:
		return nil, newError("unsupported file format: %s", formatStr.Value)
	}
}

// parseJSON parses a JSON string into Parsley objects
func parseJSON(content string) (Object, *Error) {
	var data interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, newError("failed to parse JSON: %s", err.Error())
	}
	return jsonToObject(data), nil
}

// parseYAML parses a YAML string into Parsley objects
func parseYAML(content string) (Object, *Error) {
	var data interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, newError("failed to parse YAML: %s", err.Error())
	}
	return yamlToObject(data), nil
}

// parseMarkdown parses markdown content with optional YAML frontmatter
// Returns a dictionary with: html, raw, and any frontmatter fields
func parseMarkdown(content string, env *Environment) (Object, *Error) {
	pairs := make(map[string]ast.Expression)

	// Check for YAML frontmatter (starts with ---)
	body := content
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		// Find the closing ---
		trimmed := strings.TrimSpace(content)
		rest := trimmed[3:] // Skip opening ---

		endIndex := strings.Index(rest, "\n---")
		if endIndex != -1 {
			// Extract frontmatter YAML
			frontmatterYAML := rest[:endIndex]
			body = strings.TrimSpace(rest[endIndex+4:]) // Skip closing ---\n

			// Parse YAML frontmatter
			var frontmatter map[string]interface{}
			if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
				return nil, newError("failed to parse frontmatter: %s", err.Error())
			}

			// Add frontmatter fields to result
			for key, value := range frontmatter {
				obj := yamlToObject(value)
				pairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
			}
		}
	}

	// Convert markdown to HTML using goldmark
	var htmlBuf bytes.Buffer
	md := goldmark.New()
	if err := md.Convert([]byte(body), &htmlBuf); err != nil {
		return nil, newError("failed to convert markdown: %s", err.Error())
	}

	// Add html and raw fields
	pairs["html"] = &ast.ObjectLiteralExpression{Obj: &String{Value: htmlBuf.String()}}
	pairs["raw"] = &ast.ObjectLiteralExpression{Obj: &String{Value: body}}

	return &Dictionary{Pairs: pairs, Env: env}, nil
}

// yamlToObject converts a YAML value to a Parsley Object
func yamlToObject(value interface{}) Object {
	switch v := value.(type) {
	case nil:
		return NULL
	case bool:
		return nativeBoolToParsBoolean(v)
	case int:
		return &Integer{Value: int64(v)}
	case int64:
		return &Integer{Value: v}
	case float64:
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case time.Time:
		// YAML timestamps are parsed directly by yaml.v3
		return timeToDatetimeDict(v, NewEnvironment())
	case string:
		// Try to parse as date if it looks like ISO format
		if len(v) >= 10 && v[4] == '-' && v[7] == '-' {
			if t, err := time.Parse("2006-01-02", v[:10]); err == nil {
				return timeToDatetimeDict(t, NewEnvironment())
			}
		}
		return &String{Value: v}
	case []interface{}:
		elements := make([]Object, len(v))
		for i, elem := range v {
			elements[i] = yamlToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[string]ast.Expression)
		for key, val := range v {
			obj := yamlToObject(val)
			pairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	default:
		// Handle other YAML types (like timestamps)
		return &String{Value: fmt.Sprintf("%v", v)}
	}
}

// jsonToObject converts a Go interface{} (from JSON) to a Parsley Object
func jsonToObject(data interface{}) Object {
	switch v := data.(type) {
	case nil:
		return NULL
	case bool:
		return nativeBoolToParsBoolean(v)
	case float64:
		// JSON numbers are always float64
		if v == float64(int64(v)) {
			return &Integer{Value: int64(v)}
		}
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case []interface{}:
		elements := make([]Object, len(v))
		for i, elem := range v {
			elements[i] = jsonToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[string]ast.Expression)
		for key, val := range v {
			obj := jsonToObject(val)
			pairs[key] = &ast.ObjectLiteralExpression{Obj: obj}
		}
		return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
	default:
		return NULL
	}
}

// stripXMLProlog removes XML prolog (<?xml ...?>) and DOCTYPE declarations from SVG content
func stripXMLProlog(content string) string {
	result := content

	// Strip XML prolog: <?xml version="1.0" ...?>
	for {
		start := strings.Index(result, "<?")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "?>")
		if end == -1 {
			break
		}
		// Remove the prolog and any following whitespace
		endPos := start + end + 2
		for endPos < len(result) && (result[endPos] == ' ' || result[endPos] == '\t' || result[endPos] == '\n' || result[endPos] == '\r') {
			endPos++
		}
		result = result[:start] + result[endPos:]
	}

	// Strip DOCTYPE: <!DOCTYPE ...>
	for {
		// Case insensitive search for DOCTYPE
		lower := strings.ToLower(result)
		start := strings.Index(lower, "<!doctype")
		if start == -1 {
			break
		}
		// Find the closing >
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		// Remove the DOCTYPE and any following whitespace
		endPos := start + end + 1
		for endPos < len(result) && (result[endPos] == ' ' || result[endPos] == '\t' || result[endPos] == '\n' || result[endPos] == '\r') {
			endPos++
		}
		result = result[:start] + result[endPos:]
	}

	return strings.TrimSpace(result)
}

// parseCSV parses CSV data into an array of dictionaries (if header) or array of arrays
func parseCSV(data []byte, hasHeader bool) (Object, *Error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, newError("failed to parse CSV: %s", err.Error())
	}

	if len(records) == 0 {
		return &Array{Elements: []Object{}}, nil
	}

	if hasHeader {
		// First row is headers
		headers := records[0]
		rows := make([]Object, 0, len(records)-1)

		for _, record := range records[1:] {
			pairs := make(map[string]ast.Expression)
			for i, value := range record {
				if i < len(headers) {
					pairs[headers[i]] = &ast.ObjectLiteralExpression{Obj: parseCSVValue(value)}
				}
			}
			rows = append(rows, &Dictionary{Pairs: pairs, Env: NewEnvironment()})
		}
		return &Array{Elements: rows}, nil
	}

	// No header - return array of arrays
	rows := make([]Object, len(records))
	for i, record := range records {
		elements := make([]Object, len(record))
		for j, value := range record {
			elements[j] = parseCSVValue(value)
		}
		rows[i] = &Array{Elements: elements}
	}
	return &Array{Elements: rows}, nil
}

// parseCSVValue converts a CSV string value to the appropriate type
// Tries integer, float, boolean, then falls back to string
func parseCSVValue(value string) Object {
	// Try integer first (stricter than float)
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &Integer{Value: i}
	}
	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return &Float{Value: f}
	}
	// Try boolean
	lower := strings.ToLower(value)
	if lower == "true" {
		return TRUE
	}
	if lower == "false" {
		return FALSE
	}
	// Keep as string
	return &String{Value: value}
}

// evalWriteStatement evaluates the ==> and ==>> operators to write file content
func evalWriteStatement(node *ast.WriteStatement, env *Environment) Object {
	// Evaluate the value to write
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	// Evaluate the target expression (should be a file handle, SFTP file handle, or HTTP request)
	target := Eval(node.Target, env)
	if isError(target) {
		return target
	}

	// Check if it's an SFTP file handle
	if sftpHandle, ok := target.(*SFTPFileHandle); ok {
		err := evalSFTPWrite(sftpHandle, value, node.Append, env)
		if err != nil {
			return err
		}
		return NULL
	}

	// Check if it's a request dictionary (HTTP request)
	if reqDict, ok := target.(*Dictionary); ok && isRequestDict(reqDict) {
		return evalHTTPWrite(reqDict, value, env)
	}

	// The target should be a file dictionary
	fileDict, ok := target.(*Dictionary)
	if !ok || !isFileDict(fileDict) {
		return newError("write operator requires a file handle or HTTP request, got %s", target.Type())
	}

	// Write the file content based on format
	err := writeFileContent(fileDict, value, node.Append, env)
	if err != nil {
		return err
	}

	return NULL
}

// evalHTTPWrite performs an HTTP write operation (POST/PUT/PATCH)
func evalHTTPWrite(reqDict *Dictionary, value Object, env *Environment) Object {
	// Set the body to the value being written
	pairs := make(map[string]ast.Expression)
	for key, expr := range reqDict.Pairs {
		pairs[key] = expr
	}

	// Encode the value as the request body
	pairs["body"] = &ast.ObjectLiteralExpression{Obj: value}

	// Default method to POST if not already set to PUT or PATCH
	method := "POST"
	if methodExpr, ok := reqDict.Pairs["method"]; ok {
		methodObj := Eval(methodExpr, env)
		if methodStr, ok := methodObj.(*String); ok {
			upperMethod := strings.ToUpper(methodStr.Value)
			// Only keep PUT, PATCH - otherwise default to POST
			if upperMethod == "PUT" || upperMethod == "PATCH" {
				method = upperMethod
			}
		}
	}
	pairs["method"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: method},
		Value: method,
	}

	newReqDict := &Dictionary{Pairs: pairs, Env: env}

	// Fetch URL content with full response info
	info := fetchUrlContentFull(newReqDict, env)

	// Handle errors
	if info.Error != "" {
		return newError("%s", info.Error)
	}

	// Create and return response typed dictionary
	return makeResponseTypedDict(
		info.Content,
		info.Format,
		info.StatusCode,
		info.StatusText,
		info.OK,
		info.FinalURL,
		info.Headers,
		"",
		env,
	)
}

// makeSFTPResponseDict creates a response dictionary for SFTP operations with error capture
func makeSFTPResponseDict(data Object, errMsg string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	if errMsg != "" {
		pairs["data"] = &ast.ObjectLiteralExpression{Obj: NULL}
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: &String{Value: errMsg}}
	} else {
		// Store data directly as an expression
		pairs["data"] = &ast.ObjectLiteralExpression{Obj: data}
		pairs["error"] = &ast.ObjectLiteralExpression{Obj: NULL}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// evalSFTPRead reads content from an SFTP file handle
func evalSFTPRead(handle *SFTPFileHandle, env *Environment) (Object, Object) {
	if !handle.Connection.Connected {
		return nil, newError("SFTP connection is not connected")
	}

	// Handle directory listing
	if handle.Format == "dir" {
		entries, err := handle.Connection.Client.ReadDir(handle.Path)
		if err != nil {
			return nil, newError("failed to list directory: %s", err.Error())
		}

		files := make([]Object, 0, len(entries))
		for _, entry := range entries {
			fileInfo := make(map[string]ast.Expression)
			fileInfo["name"] = &ast.StringLiteral{Value: entry.Name()}
			fileInfo["path"] = &ast.StringLiteral{Value: filepath.Join(handle.Path, entry.Name())}
			fileInfo["size"] = &ast.IntegerLiteral{Value: entry.Size()}
			fileInfo["isDir"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: entry.IsDir()}}
			fileInfo["isFile"] = &ast.ObjectLiteralExpression{Obj: &Boolean{Value: !entry.IsDir()}}
			fileInfo["mode"] = &ast.StringLiteral{Value: entry.Mode().String()}
			fileInfo["modified"] = &ast.ObjectLiteralExpression{Obj: timeToDict(entry.ModTime(), env)}

			files = append(files, &Dictionary{Pairs: fileInfo, Env: env})
		}

		return &Array{Elements: files}, nil
	}

	// Open remote file
	file, err := handle.Connection.Client.Open(handle.Path)
	if err != nil {
		return nil, newError("SFTP read failed: %s", err.Error())
	}
	defer file.Close()

	// Read content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, newError("SFTP read failed: %s", err.Error())
	}

	// Parse based on format
	format := handle.Format
	if format == "" {
		format = "text"
	}

	switch format {
	case "json":
		return parseJSON(string(data))
	case "text":
		return &String{Value: string(data)}, nil
	case "lines":
		lines := strings.Split(string(data), "\n")
		// Remove trailing empty line if present
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		elements := make([]Object, len(lines))
		for i, line := range lines {
			elements[i] = &String{Value: line}
		}
		return &Array{Elements: elements}, nil
	case "csv":
		return parseCSV(data, true) // Assume CSV has headers by default
	case "bytes":
		elements := make([]Object, len(data))
		for i, b := range data {
			elements[i] = &Integer{Value: int64(b)}
		}
		return &Array{Elements: elements}, nil
	case "file":
		// Auto-detect from extension
		ext := filepath.Ext(handle.Path)
		switch ext {
		case ".json":
			return parseJSON(string(data))
		case ".csv":
			return parseCSV(data, true)
		default:
			return &String{Value: string(data)}, nil
		}
	default:
		return nil, newError("unknown format: %s", format)
	}
}

// evalSFTPWrite writes content to an SFTP file handle
func evalSFTPWrite(handle *SFTPFileHandle, value Object, append bool, env *Environment) Object {
	if !handle.Connection.Connected {
		return newError("SFTP connection is not connected")
	}

	// Determine open flags
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND // SSH_FXF_APPEND (0x00000004)
	} else {
		flags |= os.O_TRUNC
	}

	// Encode based on format
	format := handle.Format
	if format == "" {
		format = "text"
	}

	var content string
	switch format {
	case "json":
		jsonBytes, err := encodeJSON(value)
		if err != nil {
			handle.Connection.Client.Close()
			return makeSFTPResponseDict(NULL, fmt.Sprintf("JSON encoding failed: %s", err.Error()), env)
		}
		content = string(jsonBytes)
	case "text":
		if str, ok := value.(*String); ok {
			content = str.Value
		} else {
			return newError("text format requires string value, got %s", value.Type())
		}
	case "lines":
		if arr, ok := value.(*Array); ok {
			lines := make([]string, len(arr.Elements))
			for i, elem := range arr.Elements {
				if str, ok := elem.(*String); ok {
					lines[i] = str.Value
				} else {
					return newError("lines format requires array of strings, got %s at index %d", elem.Type(), i)
				}
			}
			content = strings.Join(lines, "\n") + "\n"
		} else {
			return newError("lines format requires array, got %s", value.Type())
		}
	case "csv":
		return newError("CSV write not yet implemented for SFTP")
	case "bytes":
		if arr, ok := value.(*Array); ok {
			bytes := make([]byte, len(arr.Elements))
			for i, elem := range arr.Elements {
				if intVal, ok := elem.(*Integer); ok {
					bytes[i] = byte(intVal.Value)
				} else {
					return newError("bytes format requires array of integers, got %s at index %d", elem.Type(), i)
				}
			}
			content = string(bytes)
		} else {
			return newError("bytes format requires array, got %s", value.Type())
		}
	default:
		return newError("unknown format: %s", format)
	}

	// Open remote file via SFTP with appropriate flags
	file, err := handle.Connection.Client.OpenFile(handle.Path, flags)
	if err != nil {
		return newError("SFTP write failed: %s", err.Error())
	}
	defer file.Close()

	// Write content
	_, err = file.Write([]byte(content))
	if err != nil {
		return newError("SFTP write failed: %s", err.Error())
	}

	return NULL
}

// evalQueryOneStatement evaluates the <=?=> operator to query a single row
func evalQueryOneStatement(node *ast.QueryOneStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("query operator <=?=> requires a database connection, got %s", connObj.Type())
	}

	// Evaluate the query expression (should return a tag with SQL and params)
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params from the query object
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	// For QueryRow, we need to get column info, so we use Query instead
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Check if there's a row
	if !rows.Next() {
		// No rows - return null
		return assignQueryResult(node.Names, NULL, env, node.IsLet)
	}

	// Scan the row into a map
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
		conn.LastError = scanErr.Error()
		return newDatabaseError("DB-0004", scanErr)
	}

	// Convert to dictionary
	resultDict := rowToDict(columns, values, env)

	return assignQueryResult(node.Names, resultDict, env, node.IsLet)
}

// evalQueryManyStatement evaluates the <=??=> operator to query multiple rows
func evalQueryManyStatement(node *ast.QueryManyStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("query operator <=??=> requires a database connection, got %s", connObj.Type())
	}

	// Evaluate the query expression
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Scan all rows
	var results []Object
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			conn.LastError = scanErr.Error()
			return newDatabaseError("DB-0004", scanErr)
		}

		resultDict := rowToDict(columns, values, env)
		results = append(results, resultDict)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		conn.LastError = rowsErr.Error()
		return newDatabaseError("DB-0002", rowsErr)
	}

	resultArray := &Array{Elements: results}
	return assignQueryResult(node.Names, resultArray, env, node.IsLet)
}

// evalExecuteStatement evaluates the <=!=> operator to execute mutations
func evalExecuteStatement(node *ast.ExecuteStatement, env *Environment) Object {
	// Evaluate the connection
	connObj := Eval(node.Connection, env)
	if isError(connObj) {
		return connObj
	}

	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("execute operator <=!=> requires a database connection, got %s", connObj.Type())
	}

	// Evaluate the query expression
	queryObj := Eval(node.Query, env)
	if isError(queryObj) {
		return queryObj
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the statement
	result, execErr := conn.DB.Exec(sql, params...)
	if execErr != nil {
		conn.LastError = execErr.Error()
		return newError("execute failed: %s", execErr.Error())
	}

	// Get affected rows and last insert ID
	affected, _ := result.RowsAffected()
	lastId, _ := result.LastInsertId()

	// Return result as dictionary
	resultDict := &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
				Value: affected,
			},
			"lastId": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastId, 10)},
				Value: lastId,
			},
		},
		Env: env,
	}

	return assignQueryResult(node.Names, resultDict, env, node.IsLet)
}

// extractSQLAndParams extracts SQL string and parameters from a query object
func extractSQLAndParams(queryObj Object, env *Environment) (string, []interface{}, *Error) {
	// If it's a string, use it directly with no params
	if str, ok := queryObj.(*String); ok {
		return str.Value, nil, nil
	}

	// If it's a dictionary (from <SQL> tag), extract sql and params
	if dict, ok := queryObj.(*Dictionary); ok {
		// Get SQL content
		sqlExpr, hasSql := dict.Pairs["sql"]
		if !hasSql {
			return "", nil, newError("query object missing 'sql' property")
		}
		sqlObj := Eval(sqlExpr, env)
		if isError(sqlObj) {
			return "", nil, sqlObj.(*Error)
		}
		sqlStr, ok := sqlObj.(*String)
		if !ok {
			return "", nil, newError("sql property must be a string, got %s", sqlObj.Type())
		}

		// Get params if present
		var params []interface{}
		if paramsExpr, hasParams := dict.Pairs["params"]; hasParams {
			paramsObj := Eval(paramsExpr, env)
			if isError(paramsObj) {
				return "", nil, paramsObj.(*Error)
			}
			if paramsDict, ok := paramsObj.(*Dictionary); ok {
				params = dictToNamedParams(paramsDict, env)
			}
		}

		return sqlStr.Value, params, nil
	}

	return "", nil, newError("query must be a string or <SQL> tag, got %s", queryObj.Type())
}

// dictToNamedParams converts a dictionary to a slice of named parameters
func dictToNamedParams(dict *Dictionary, env *Environment) []interface{} {
	params := make([]interface{}, 0, len(dict.Pairs))

	// Sort keys for consistent order
	keys := make([]string, 0, len(dict.Pairs))
	for key := range dict.Pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		expr := dict.Pairs[key]
		val := Eval(expr, env)
		params = append(params, objectToGoValue(val))
	}

	return params
}

// objectToGoValue converts a Parsley object to a Go value for database params
func objectToGoValue(obj Object) interface{} {
	switch v := obj.(type) {
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Boolean:
		return v.Value
	case *Null:
		return nil
	default:
		return obj.Inspect()
	}
}

// rowToDict converts a database row to a Parsley dictionary
func rowToDict(columns []string, values []interface{}, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	for i, col := range columns {
		var expr ast.Expression

		switch v := values[i].(type) {
		case int64:
			literal := strconv.FormatInt(v, 10)
			expr = &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: literal},
				Value: v,
			}
		case float64:
			literal := strconv.FormatFloat(v, 'f', -1, 64)
			expr = &ast.FloatLiteral{
				Token: lexer.Token{Type: lexer.FLOAT, Literal: literal},
				Value: v,
			}
		case string:
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: v},
				Value: v,
			}
		case []byte:
			strVal := string(v)
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: strVal},
				Value: strVal,
			}
		case bool:
			var tokenType lexer.TokenType
			var literal string
			if v {
				tokenType = lexer.TRUE
				literal = "true"
			} else {
				tokenType = lexer.FALSE
				literal = "false"
			}
			expr = &ast.Boolean{
				Token: lexer.Token{Type: tokenType, Literal: literal},
				Value: v,
			}
		case nil:
			expr = &ast.Identifier{
				Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
				Value: "null",
			}
		default:
			// For unknown types, convert to string
			strVal := fmt.Sprintf("%v", v)
			expr = &ast.StringLiteral{
				Token: lexer.Token{Type: lexer.STRING, Literal: strVal},
				Value: strVal,
			}
		}

		pairs[col] = expr
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// assignQueryResult assigns query result to variables
func assignQueryResult(names []*ast.Identifier, result Object, env *Environment, isLet bool) Object {
	if len(names) == 0 {
		return result
	}

	if len(names) == 1 {
		name := names[0].Value
		if name != "_" {
			if isLet {
				env.SetLet(name, result)
			} else {
				env.Update(name, result)
			}
		}
		return result
	}

	// Multiple names - destructure array or dict
	return evalDestructuringAssignment(names, result, env, isLet, false)
}

// evalDatabaseQueryOne evaluates database query for single row (infix expression version)
func evalDatabaseQueryOne(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("query operator <=?=> requires a database connection, got %s", connObj.Type())
	}

	// Extract SQL and params from the query object
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Check if there's a row
	if !rows.Next() {
		// No rows - return null
		return NULL
	}

	// Scan the row into a map
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
		conn.LastError = scanErr.Error()
		return newDatabaseError("DB-0004", scanErr)
	}

	// Convert to dictionary
	return rowToDict(columns, values, env)
}

// evalDatabaseQueryMany evaluates database query for multiple rows (infix expression version)
func evalDatabaseQueryMany(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("query operator <=??=> requires a database connection, got %s", connObj.Type())
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the query
	rows, queryErr := conn.DB.Query(sql, params...)
	if queryErr != nil {
		conn.LastError = queryErr.Error()
		return newDatabaseError("DB-0002", queryErr)
	}
	defer rows.Close()

	// Get column names
	columns, colErr := rows.Columns()
	if colErr != nil {
		conn.LastError = colErr.Error()
		return newDatabaseError("DB-0008", colErr)
	}

	// Scan all rows
	var results []Object
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if scanErr := rows.Scan(valuePtrs...); scanErr != nil {
			conn.LastError = scanErr.Error()
			return newDatabaseError("DB-0004", scanErr)
		}

		resultDict := rowToDict(columns, values, env)
		results = append(results, resultDict)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		conn.LastError = rowsErr.Error()
		return newDatabaseError("DB-0002", rowsErr)
	}

	return &Array{Elements: results}
}

// evalDatabaseExecute evaluates database execute statement (infix expression version)
func evalDatabaseExecute(connObj Object, queryObj Object, env *Environment) Object {
	conn, ok := connObj.(*DBConnection)
	if !ok {
		return newError("execute operator <=!=> requires a database connection, got %s", connObj.Type())
	}

	// Extract SQL and params
	sql, params, err := extractSQLAndParams(queryObj, env)
	if err != nil {
		return err
	}

	// Execute the statement
	result, execErr := conn.DB.Exec(sql, params...)
	if execErr != nil {
		conn.LastError = execErr.Error()
		return newError("execute failed: %s", execErr.Error())
	}

	// Get affected rows and last insert ID
	affected, _ := result.RowsAffected()
	lastId, _ := result.LastInsertId()

	// Return result as dictionary
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"affected": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(affected, 10)},
				Value: affected,
			},
			"lastId": &ast.IntegerLiteral{
				Token: lexer.Token{Type: lexer.INT, Literal: strconv.FormatInt(lastId, 10)},
				Value: lastId,
			},
		},
		Env: env,
	}
}

// writeFileContent writes content to a file based on its format
func writeFileContent(fileDict *Dictionary, value Object, appendMode bool, env *Environment) *Error {
	// Check if this is a stdio stream
	var isStdio bool
	var stdioStream string

	if stdioExpr, ok := fileDict.Pairs["__stdio"]; ok {
		stdioObj := Eval(stdioExpr, env)
		if stdioStr, ok := stdioObj.(*String); ok {
			switch stdioStr.Value {
			case "stdin":
				return newError("cannot write to stdin")
			case "stdout", "stderr":
				isStdio = true
				stdioStream = stdioStr.Value
			case "stdio":
				// @- for writes means stdout
				isStdio = true
				stdioStream = "stdout"
			default:
				return newError("unknown stdio stream: %s", stdioStr.Value)
			}
		}
	}

	var pathStr string
	if !isStdio {
		// Get the path from the file dictionary
		pathStr = getFilePathString(fileDict, env)
		if pathStr == "" {
			return newError("file handle has no valid path")
		}

		// Resolve the path relative to the current file (or root path for ~/ paths)
		absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
		if pathErr != nil {
			return newError("failed to resolve path '%s': %s", pathStr, pathErr.Error())
		}
		pathStr = absPath

		// Security check
		if err := env.checkPathAccess(pathStr, "write"); err != nil {
			return newSecurityError("write", err)
		}
	}

	// Get the format
	formatExpr, hasFormat := fileDict.Pairs["format"]
	if !hasFormat {
		return newError("file handle has no format specified")
	}
	formatObj := Eval(formatExpr, env)
	if isError(formatObj) {
		return formatObj.(*Error)
	}
	formatStr, ok := formatObj.(*String)
	if !ok {
		return newError("file format must be a string, got %s", formatObj.Type())
	}

	// Encode the value based on format
	var data []byte
	var encodeErr error

	switch formatStr.Value {
	case "text":
		data, encodeErr = encodeText(value)

	case "bytes":
		data, encodeErr = encodeBytes(value)

	case "lines":
		data, encodeErr = encodeLines(value, appendMode)

	case "json":
		data, encodeErr = encodeJSON(value)

	case "csv", "csv-noheader":
		data, encodeErr = encodeCSV(value, formatStr.Value == "csv")

	case "svg":
		data, encodeErr = encodeSVG(value)

	case "yaml":
		data, encodeErr = encodeYAML(value)

	default:
		return newError("unsupported file format for writing: %s", formatStr.Value)
	}

	if encodeErr != nil {
		return newError("failed to encode data: %s", encodeErr.Error())
	}

	// Write to stdout/stderr or file
	var writeErr error
	if isStdio {
		// Write to stdout or stderr
		var w *os.File
		if stdioStream == "stdout" {
			w = os.Stdout
		} else {
			w = os.Stderr
		}
		_, writeErr = w.Write(data)
	} else if appendMode {
		f, err := os.OpenFile(pathStr, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return newIOError("IO-0004", pathStr, err)
		}
		defer f.Close()
		_, writeErr = f.Write(data)
	} else {
		writeErr = os.WriteFile(pathStr, data, 0644)
	}

	if writeErr != nil {
		if isStdio {
			return newIOError("IO-0004", stdioStream, writeErr)
		}
		return newIOError("IO-0004", pathStr, writeErr)
	}

	return nil
}

// encodeText encodes a value as text
func encodeText(value Object) ([]byte, error) {
	switch v := value.(type) {
	case *String:
		return []byte(v.Value), nil
	default:
		return []byte(value.Inspect()), nil
	}
}

// encodeBytes encodes a value as bytes
func encodeBytes(value Object) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		return nil, fmt.Errorf("bytes format requires an array, got %s", value.Type())
	}

	data := make([]byte, len(arr.Elements))
	for i, elem := range arr.Elements {
		intVal, ok := elem.(*Integer)
		if !ok {
			return nil, fmt.Errorf("bytes array must contain integers, got %s at index %d", elem.Type(), i)
		}
		if intVal.Value < 0 || intVal.Value > 255 {
			return nil, fmt.Errorf("byte value out of range (0-255): %d at index %d", intVal.Value, i)
		}
		data[i] = byte(intVal.Value)
	}
	return data, nil
}

// encodeLines encodes a value as lines
func encodeLines(value Object, appendMode bool) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		// Single value - treat as single line
		if appendMode {
			return []byte(value.Inspect() + "\n"), nil
		}
		return []byte(value.Inspect()), nil
	}

	var builder strings.Builder
	for i, elem := range arr.Elements {
		if i > 0 {
			builder.WriteString("\n")
		}
		switch v := elem.(type) {
		case *String:
			builder.WriteString(v.Value)
		default:
			builder.WriteString(elem.Inspect())
		}
	}
	return []byte(builder.String()), nil
}

// encodeJSON encodes a value as JSON
func encodeJSON(value Object) ([]byte, error) {
	goValue := objectToGo(value)
	return json.MarshalIndent(goValue, "", "  ")
}

// objectToGo converts a Parsley Object to a Go interface{} for JSON encoding
func objectToGo(obj Object) interface{} {
	switch v := obj.(type) {
	case *Null:
		return nil
	case *Boolean:
		return v.Value
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Array:
		result := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			result[i] = objectToGo(elem)
		}
		return result
	case *Dictionary:
		result := make(map[string]interface{})
		for key, expr := range v.Pairs {
			// Skip internal fields
			if strings.HasPrefix(key, "_") {
				continue
			}
			// Evaluate expression if it's an ObjectLiteralExpression
			if ole, ok := expr.(*ast.ObjectLiteralExpression); ok {
				result[key] = objectToGo(ole.Obj.(Object))
			} else {
				// For other expressions, we need to evaluate them
				env := NewEnvironment()
				val := Eval(expr, env)
				result[key] = objectToGo(val)
			}
		}
		return result
	default:
		return obj.Inspect()
	}
}

// encodeSVG encodes a value as SVG (text format, for writing)
func encodeSVG(value Object) ([]byte, error) {
	switch v := value.(type) {
	case *String:
		return []byte(v.Value), nil
	default:
		// Convert to string representation
		return []byte(value.Inspect()), nil
	}
}

// encodeYAML encodes a value as YAML
func encodeYAML(value Object) ([]byte, error) {
	goValue := objectToGo(value)
	return yaml.Marshal(goValue)
}

// encodeCSV encodes a value as CSV
func encodeCSV(value Object, hasHeader bool) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		return nil, fmt.Errorf("CSV format requires an array, got %s", value.Type())
	}

	if len(arr.Elements) == 0 {
		return []byte{}, nil
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Check if first element is a dictionary (has header) or array (no header)
	firstDict, isDict := arr.Elements[0].(*Dictionary)

	if isDict && hasHeader {
		// Write header from dictionary keys
		var headers []string
		for key := range firstDict.Pairs {
			if !strings.HasPrefix(key, "_") {
				headers = append(headers, key)
			}
		}
		sort.Strings(headers) // Consistent ordering
		if err := writer.Write(headers); err != nil {
			return nil, err
		}

		// Write rows
		for _, elem := range arr.Elements {
			dict, ok := elem.(*Dictionary)
			if !ok {
				return nil, fmt.Errorf("CSV with header requires all rows to be dictionaries")
			}
			row := make([]string, len(headers))
			for i, key := range headers {
				if expr, exists := dict.Pairs[key]; exists {
					if ole, ok := expr.(*ast.ObjectLiteralExpression); ok {
						row[i] = ole.Obj.(Object).Inspect()
					} else {
						env := NewEnvironment()
						val := Eval(expr, env)
						row[i] = val.Inspect()
					}
				}
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	} else {
		// Write as array of arrays
		for _, elem := range arr.Elements {
			rowArr, ok := elem.(*Array)
			if !ok {
				// Single-element row
				if err := writer.Write([]string{elem.Inspect()}); err != nil {
					return nil, err
				}
				continue
			}
			row := make([]string, len(rowArr.Elements))
			for i, cell := range rowArr.Elements {
				switch v := cell.(type) {
				case *String:
					row[i] = v.Value
				default:
					row[i] = cell.Inspect()
				}
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// evalFileRemove removes/deletes a file from the filesystem
func evalFileRemove(fileDict *Dictionary, env *Environment) Object {
	// Get the path from the file dictionary
	pathStr := getFilePathString(fileDict, env)
	if pathStr == "" {
		return newError("file handle has no valid path")
	}

	// Resolve the path relative to the current file (or root path for ~/ paths)
	absPath, pathErr := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if pathErr != nil {
		return newIOError("IO-0007", pathStr, pathErr)
	}

	// Security check (treat as write operation)
	if err := env.checkPathAccess(absPath, "write"); err != nil {
		return newSecurityError("write", err)
	}

	// Delete the file
	err := os.Remove(absPath)
	if err != nil {
		return newIOError("IO-0005", absPath, err)
	}

	// Return a new null value instead of the global NULL
	return &Null{}
}

// evalDictionaryIndexExpression handles dictionary access via dict["key"]
// The optional parameter is accepted for API consistency but dictionaries already return NULL for missing keys
func evalDictionaryIndexExpression(dict, index Object, optional bool) Object {
	dictObject := dict.(*Dictionary)
	key := index.(*String).Value

	// Get the expression from the dictionary
	expr, ok := dictObject.Pairs[key]
	if !ok {
		return NULL
	}

	// Create a new environment with 'this' bound to the dictionary
	dictEnv := NewEnclosedEnvironment(dictObject.Env)
	dictEnv.Set("this", dictObject)

	// Evaluate the expression in the dictionary's environment
	return Eval(expr, dictEnv)
}

// environmentToDict converts an environment's store to a Dictionary object
// Only includes variables that are exported (either via explicit 'export' or 'let' for backward compat)
func environmentToDict(env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Only export variables that are explicitly exported or declared with 'let'
	for name, value := range env.store {
		if env.IsExported(name) {
			// Wrap the object as a literal expression
			pairs[name] = objectToExpression(value)
		}
	}

	// Create dictionary with the module's environment for evaluation
	return &Dictionary{Pairs: pairs, Env: env}
}

// objectToExpression wraps an Object as an AST expression
func objectToExpression(obj Object) ast.Expression {
	switch v := obj.(type) {
	case *Integer:
		return &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", v.Value)},
			Value: v.Value,
		}
	case *Float:
		return &ast.FloatLiteral{
			Token: lexer.Token{Type: lexer.FLOAT, Literal: fmt.Sprintf("%g", v.Value)},
			Value: v.Value,
		}
	case *String:
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: v.Value},
			Value: v.Value,
		}
	case *Boolean:
		if v.Value {
			return &ast.Boolean{
				Token: lexer.Token{Type: lexer.TRUE, Literal: "true"},
				Value: v.Value,
			}
		}
		return &ast.Boolean{
			Token: lexer.Token{Type: lexer.FALSE, Literal: "false"},
			Value: v.Value,
		}
	default:
		// For complex types (functions, arrays, dictionaries, null), we create
		// an expression that returns the object directly when evaluated
		return &ast.ObjectLiteralExpression{Obj: obj}
	}
}

// objectLiteralExpression removed - now using ast.ObjectLiteralExpression

// resolveModulePath resolves a module path relative to the current file or root path.
// Paths starting with ~/ are resolved from rootPath (handler root directory in Basil).
// If rootPath is not set, ~/ falls back to the user's home directory.
// Paths starting with / are absolute.
// All other paths are resolved relative to the current file.
func resolveModulePath(pathStr string, currentFile string, rootPath string) (string, error) {
	var absPath string

	// Handle ~/ prefix - resolve from root path or home directory
	if strings.HasPrefix(pathStr, "~/") {
		if rootPath != "" {
			// In Basil context: ~/ means handler root directory
			absPath = filepath.Join(rootPath, pathStr[2:])
		} else {
			// Standalone: ~/ means home directory (traditional behavior)
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("cannot expand ~/: %s", err.Error())
			}
			absPath = filepath.Join(home, pathStr[2:])
		}
	} else if strings.HasPrefix(pathStr, "/") {
		// If path is absolute, use it directly
		absPath = pathStr
	} else {
		// Resolve relative to the current file's directory
		var baseDir string
		if currentFile != "" {
			baseDir = filepath.Dir(currentFile)
		} else {
			// If no current file, use current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			baseDir = cwd
		}

		// Join and clean the path
		absPath = filepath.Join(baseDir, pathStr)
	}

	// Clean the path (resolve . and ..)
	absPath = filepath.Clean(absPath)

	return absPath, nil
}

// ============================================================================
// Enhanced Operator Functions
// ============================================================================

// evalArrayIntersection returns elements present in both arrays
func evalArrayIntersection(left, right *Array) Object {
	// Build hash set of right array elements for O(n) lookup
	rightSet := make(map[string]bool)
	for _, elem := range right.Elements {
		rightSet[elem.Inspect()] = true
	}

	// Keep elements from left that exist in right, deduplicate
	seen := make(map[string]bool)
	result := []Object{}
	for _, elem := range left.Elements {
		key := elem.Inspect()
		if rightSet[key] && !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	return &Array{Elements: result}
}

// evalDictionaryIntersection returns keys present in both dictionaries with values from left
func evalDictionaryIntersection(left, right *Dictionary) Object {
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   left.Env,
	}

	// Keep only keys that exist in both dictionaries
	for k, v := range left.Pairs {
		if _, exists := right.Pairs[k]; exists {
			result.Pairs[k] = v
		}
	}

	return result
}

// evalArrayUnion returns all unique elements from both arrays
func evalArrayUnion(left, right *Array) Object {
	seen := make(map[string]bool)
	result := []Object{}

	// Add elements from left
	for _, elem := range left.Elements {
		key := elem.Inspect()
		if !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	// Add elements from right
	for _, elem := range right.Elements {
		key := elem.Inspect()
		if !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	return &Array{Elements: result}
}

// evalArraySubtraction removes elements present in right from left
func evalArraySubtraction(left, right *Array) Object {
	// Build hash set of elements to remove
	removeSet := make(map[string]bool)
	for _, elem := range right.Elements {
		removeSet[elem.Inspect()] = true
	}

	// Keep elements from left that are not in removeSet
	result := []Object{}
	for _, elem := range left.Elements {
		if !removeSet[elem.Inspect()] {
			result = append(result, elem)
		}
	}

	return &Array{Elements: result}
}

// evalDictionarySubtraction removes keys present in right from left
func evalDictionarySubtraction(left, right *Dictionary) Object {
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   left.Env,
	}

	// Keep keys from left that don't exist in right
	for k, v := range left.Pairs {
		if _, exists := right.Pairs[k]; !exists {
			result.Pairs[k] = v
		}
	}

	return result
}

// evalArrayChunking splits array into chunks of specified size
func evalArrayChunking(tok lexer.Token, array *Array, size *Integer) Object {
	chunkSize := int(size.Value)

	if chunkSize <= 0 {
		return newErrorWithPos(tok, "chunk size must be > 0, got %d", chunkSize)
	}

	result := []Object{}
	for i := 0; i < len(array.Elements); i += chunkSize {
		end := i + chunkSize
		if end > len(array.Elements) {
			end = len(array.Elements)
		}
		chunk := &Array{Elements: array.Elements[i:end]}
		result = append(result, chunk)
	}

	return &Array{Elements: result}
}

// evalStringRepetition repeats a string n times
func evalStringRepetition(str *String, count *Integer) Object {
	n := int(count.Value)

	if n <= 0 {
		return &String{Value: ""}
	}

	var builder strings.Builder
	builder.Grow(len(str.Value) * n)
	for i := 0; i < n; i++ {
		builder.WriteString(str.Value)
	}

	return &String{Value: builder.String()}
}

// evalArrayRepetition repeats an array n times
func evalArrayRepetition(array *Array, count *Integer) Object {
	n := int(count.Value)

	if n <= 0 {
		return &Array{Elements: []Object{}}
	}

	result := make([]Object, 0, len(array.Elements)*n)
	for i := 0; i < n; i++ {
		result = append(result, array.Elements...)
	}

	return &Array{Elements: result}
}

// evalRangeExpression creates an inclusive range from start to end
func evalRangeExpression(tok lexer.Token, left, right Object) Object {
	if left.Type() != INTEGER_OBJ {
		return newErrorWithPos(tok, "range start must be an integer, got %s", left.Type())
	}
	if right.Type() != INTEGER_OBJ {
		return newErrorWithPos(tok, "range end must be an integer, got %s", right.Type())
	}

	start := left.(*Integer).Value
	end := right.(*Integer).Value

	// Calculate size and direction
	var size int64
	var step int64
	if start <= end {
		size = end - start + 1
		step = 1
	} else {
		size = start - end + 1
		step = -1
	}

	// Pre-allocate array
	elements := make([]Object, size)
	val := start
	for i := int64(0); i < size; i++ {
		elements[i] = &Integer{Value: val}
		val += step
	}

	return &Array{Elements: elements}
}

// ============================================================================
// Helper functions for method implementations (used by methods.go)
// ============================================================================

// formatNumberWithLocale formats a number with the given locale
func formatNumberWithLocale(value float64, localeStr string) Object {
	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}
	p := message.NewPrinter(tag)
	return &String{Value: p.Sprintf("%v", number.Decimal(value))}
}

// formatCurrencyWithLocale formats a currency value with the given locale
func formatCurrencyWithLocale(value float64, currencyCode string, localeStr string) Object {
	cur, err := currency.ParseISO(currencyCode)
	if err != nil {
		return newError("invalid currency code: %s", currencyCode)
	}

	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}

	p := message.NewPrinter(tag)
	amount := cur.Amount(value)
	return &String{Value: p.Sprintf("%v", currency.Symbol(amount))}
}

// formatPercentWithLocale formats a percentage with the given locale
func formatPercentWithLocale(value float64, localeStr string) Object {
	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}
	p := message.NewPrinter(tag)
	return &String{Value: p.Sprintf("%v", number.Percent(value))}
}

// formatDateWithStyleAndLocale formats a datetime dictionary with the given style and locale
func formatDateWithStyleAndLocale(dict *Dictionary, style string, localeStr string, env *Environment) Object {
	// Extract time from datetime dictionary
	var t time.Time
	if unixExpr, ok := dict.Pairs["unix"]; ok {
		unixObj := Eval(unixExpr, NewEnvironment())
		if unixInt, ok := unixObj.(*Integer); ok {
			t = time.Unix(unixInt.Value, 0).UTC()
		}
	}

	// Validate style
	validStyles := map[string]bool{"short": true, "medium": true, "long": true, "full": true}
	if !validStyles[style] {
		return newError("style must be one of: short, medium, long, full, got %s", style)
	}

	// Map locale string to monday.Locale
	mondayLocale := getMondayLocale(localeStr)

	// Get format pattern for style
	format := getDateFormatForStyle(style, mondayLocale)

	return &String{Value: monday.Format(t, format, mondayLocale)}
}
