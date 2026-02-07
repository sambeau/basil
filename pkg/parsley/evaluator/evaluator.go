package evaluator

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/locale"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	_ "modernc.org/sqlite"
)

// FragmentCacher is the interface for fragment caching in the evaluator.
// This allows the server package to provide cache implementation without
// creating a circular dependency.
type FragmentCacher interface {
	// Get returns cached HTML fragment and true on hit, empty string and false on miss
	Get(key string) (string, bool)
	// Set stores a fragment in the cache with the given TTL
	Set(key string, html string, maxAge time.Duration)
	// Invalidate removes a specific cache entry
	Invalidate(key string)
}

// AssetRegistrar is the interface for public asset registration in the evaluator.
// This allows the server package to provide asset registry implementation without
// creating a circular dependency.
type AssetRegistrar interface {
	// Register registers a file and returns its public URL.
	// Returns error if file doesn't exist or exceeds size limits.
	Register(filepath string) (string, error)
}

// AssetBundler provides site-wide CSS/JS bundle URLs for <Css/> and <Script/> tags.
type AssetBundler interface {
	CSSUrl() string // Returns URL for CSS bundle, or empty string if no CSS files
	JSUrl() string  // Returns URL for JS bundle, or empty string if no JS files
}

// Connection caches are now managed in connection_cache.go with TTL and health checks

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
	RECORD_OBJ           = "RECORD" // Schema-bound data with validation
	DB_CONNECTION_OBJ    = "DB_CONNECTION"
	SFTP_CONNECTION_OBJ  = "SFTP_CONNECTION"
	SFTP_FILE_HANDLE_OBJ = "SFTP_FILE_HANDLE"
	TABLE_OBJ            = "TABLE"
	TABLE_BINDING_OBJ    = "TABLE_BINDING"
	MDDOC_OBJ            = "MDDOC"
	PRINT_VALUE_OBJ      = "PRINT_VALUE"
	MONEY_OBJ            = "MONEY"
	API_ERROR_OBJ        = "API_ERROR" // API errors (not runtime errors)
	REDIRECT_OBJ         = "REDIRECT"  // HTTP redirect response
	STOP_SIGNAL_OBJ      = "STOP_SIGNAL"
	SKIP_SIGNAL_OBJ      = "SKIP_SIGNAL"
	CHECK_EXIT_OBJ       = "CHECK_EXIT"
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

// Money represents money/currency objects with exact arithmetic
type Money struct {
	Amount   int64  // Amount in smallest unit (e.g., cents)
	Currency string // Currency code (e.g., "USD", "GBP", "EUR")
	Scale    int8   // Decimal places (2 for USD, 0 for JPY)
}

func (m *Money) Type() ObjectType { return MONEY_OBJ }

func (m *Money) Inspect() string {
	// Use symbol shortcuts for common currencies
	symbol := currencyToSymbol(m.Currency)
	if symbol != "" {
		return symbol + m.formatAmount()
	}
	// Use CODE#amount format for others
	return m.Currency + "#" + m.formatAmount()
}

// formatAmount returns the formatted amount without currency prefix
func (m *Money) formatAmount() string {
	if m.Scale == 0 {
		return strconv.FormatInt(m.Amount, 10)
	}

	divisor := int64(1)
	for i := int8(0); i < m.Scale; i++ {
		divisor *= 10
	}

	negative := m.Amount < 0
	amount := m.Amount
	if negative {
		amount = -amount
	}

	whole := amount / divisor
	frac := amount % divisor

	// Format with leading zeros in fractional part
	format := fmt.Sprintf("%%d.%%0%dd", m.Scale)
	result := fmt.Sprintf(format, whole, frac)

	if negative {
		return "-" + result
	}
	return result
}

// Currency helpers moved to eval_helpers.go

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

// PrintValue represents values to be added to the result stream by print()/println()
type PrintValue struct {
	Values []Object
}

func (pv *PrintValue) Type() ObjectType { return PRINT_VALUE_OBJ }
func (pv *PrintValue) Inspect() string  { return "<print>" }

// StopSignal signals early exit from a for loop
type StopSignal struct{}

func (s *StopSignal) Type() ObjectType { return STOP_SIGNAL_OBJ }
func (s *StopSignal) Inspect() string  { return "<stop>" }

// SkipSignal signals skipping the current iteration in a for loop
type SkipSignal struct{}

func (s *SkipSignal) Type() ObjectType { return SKIP_SIGNAL_OBJ }
func (s *SkipSignal) Inspect() string  { return "<skip>" }

// CheckExit signals early exit from a check statement
type CheckExit struct {
	Value Object
}

func (c *CheckExit) Type() ObjectType { return CHECK_EXIT_OBJ }
func (c *CheckExit) Inspect() string  { return c.Value.Inspect() }

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
	ClassValue     = perrors.ClassValue
)

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	var sb strings.Builder
	if e.File != "" {
		sb.WriteString("in ")
		sb.WriteString(e.File)
		sb.WriteString(": ")
	}
	if e.Line > 0 {
		sb.WriteString(fmt.Sprintf("line %d, column %d: ", e.Line, e.Column))
	}
	sb.WriteString(e.Message)
	return sb.String()
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
	// Format parameters as comma-separated list
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	paramStr := strings.Join(params, ", ")

	// Get body string
	body := f.Body.String()

	// For single-line bodies, keep it compact
	if !strings.Contains(body, "\n") && len(body) < 60 {
		return fmt.Sprintf("fn(%s) { %s }", paramStr, body)
	}

	// For multi-line bodies, indent each line
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}
	indentedBody := strings.Join(lines, "\n")
	return fmt.Sprintf("fn(%s) {\n%s\n}", paramStr, indentedBody)
}

// ParamCount returns the number of parameters for this function
func (f *Function) ParamCount() int {
	return len(f.Params)
}

// BuiltinFunction represents a built-in function
type BuiltinFunction func(args ...Object) Object

// Builtin represents built-in function objects
type Builtin struct {
	Fn        BuiltinFunction
	FnWithEnv func(env *Environment, args ...Object) Object
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
	Pairs    map[string]ast.Expression // Store expressions for lazy evaluation
	KeyOrder []string                  // Insertion order of keys
	Env      *Environment              // Environment for evaluation (for 'this' binding)
}

func (d *Dictionary) Type() ObjectType { return DICTIONARY_OBJ }
func (d *Dictionary) Inspect() string {
	var out strings.Builder
	pairs := []string{}

	// Use KeyOrder if available, otherwise fall back to sorted keys
	keys := d.KeyOrder
	if len(keys) == 0 && len(d.Pairs) > 0 {
		keys = make([]string, 0, len(d.Pairs))
		for key := range d.Pairs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}

	for _, key := range keys {
		expr, ok := d.Pairs[key]
		if !ok {
			continue // Skip keys that were deleted
		}
		// For inspection, we show the expression with proper formatting
		// Empty string literals and empty string objects need quotes to be visible
		var valueStr string
		if strLit, isStrLit := expr.(*ast.StringLiteral); isStrLit && strLit.Value == "" {
			valueStr = `""`
		} else if objLit, isObjLit := expr.(*ast.ObjectLiteralExpression); isObjLit {
			// Check if it's an empty String object
			if strObj, isStr := objLit.Obj.(*String); isStr && strObj.Value == "" {
				valueStr = `""`
			} else if objLit.Obj == nil {
				valueStr = "null"
			} else {
				valueStr = expr.String()
			}
		} else {
			valueStr = expr.String()
		}
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, valueStr))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// Keys returns the keys in insertion order (or sorted if KeyOrder not set)
func (d *Dictionary) Keys() []string {
	if len(d.KeyOrder) > 0 {
		// Filter to only keys that still exist in Pairs
		keys := make([]string, 0, len(d.KeyOrder))
		for _, k := range d.KeyOrder {
			if _, ok := d.Pairs[k]; ok {
				keys = append(keys, k)
			}
		}
		return keys
	}
	// Fallback: sorted keys for backward compatibility
	keys := make([]string, 0, len(d.Pairs))
	for k := range d.Pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SetKey sets a key-value pair, appending to KeyOrder if new
func (d *Dictionary) SetKey(key string, expr ast.Expression) {
	if _, exists := d.Pairs[key]; !exists {
		d.KeyOrder = append(d.KeyOrder, key)
	}
	d.Pairs[key] = expr
}

// DeleteKey removes a key from both Pairs and KeyOrder
func (d *Dictionary) DeleteKey(key string) {
	delete(d.Pairs, key)
	// Remove from KeyOrder (lazy - we filter in Keys() instead for efficiency)
	// KeyOrder cleanup happens in Keys() method
}

// Table represents a tabular data structure wrapping an array of dictionaries.
// Provides SQL-like operations (where, orderBy, select, etc.) with immutable semantics.
type Table struct {
	Rows        []*Dictionary // Array of dictionaries (each row is a dict)
	Columns     []string      // Column order (from first row or select())
	Schema      *DSLSchema    // Optional: attached schema for typed tables
	FromDB      bool          // True if data came from a database query (records are auto-validated)
	isChainCopy bool          // Internal: true if this is a copy created for method chaining
}

func (t *Table) Type() ObjectType { return TABLE_OBJ }
func (t *Table) Inspect() string {
	if t.Schema != nil {
		return fmt.Sprintf("Table<%s>(%d rows)", t.Schema.Name, len(t.Rows))
	}
	return fmt.Sprintf("Table(%d rows)", len(t.Rows))
}

// Copy creates a deep copy of the Table for immutability
func (t *Table) Copy() *Table {
	newRows := make([]*Dictionary, len(t.Rows))
	copy(newRows, t.Rows) // Shallow copy of slice - rows themselves are immutable dicts
	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{
		Rows:        newRows,
		Columns:     newColumns,
		Schema:      t.Schema, // Schema is shared (immutable)
		isChainCopy: false,    // New copy starts fresh chain
	}
}

// ensureChainCopy returns a copy for method chaining.
// If this table is already a chain copy, returns itself (avoiding redundant copies).
// Otherwise, creates a new copy marked as a chain copy.
// This enables efficient chaining: table.where(...).orderBy(...).limit(...)
// creates only ONE copy regardless of chain length.
func (t *Table) ensureChainCopy() *Table {
	if t.isChainCopy {
		return t // Already a chain copy, reuse it
	}
	// Create new copy for the chain
	newRows := make([]*Dictionary, len(t.Rows))
	copy(newRows, t.Rows)
	newColumns := make([]string, len(t.Columns))
	copy(newColumns, t.Columns)
	return &Table{
		Rows:        newRows,
		Columns:     newColumns,
		Schema:      t.Schema,
		isChainCopy: true, // Mark as chain copy
	}
}

// endChain returns a table with the chain flag cleared.
// Called when a table is assigned to a variable, passed as argument, or iterated.
// This ensures subsequent operations on the result create new copies.
func (t *Table) endChain() *Table {
	if !t.isChainCopy {
		return t // Not a chain copy, nothing to do
	}
	t.isChainCopy = false
	return t
}

// endTableChain ends any active chain on a Table.
// If obj is a Table with isChainCopy=true, clears the flag.
// Returns the (possibly modified) object unchanged for non-Tables.
// Call this when storing a table (assignment) or passing as argument.
func endTableChain(obj Object) Object {
	if t, ok := obj.(*Table); ok {
		return t.endChain()
	}
	return obj
}

// DBConnection represents a database connection
type DBConnection struct {
	DB            *sql.DB
	Tx            *sql.Tx // Active transaction, nil if not in transaction
	Driver        string  // "sqlite", "postgres", "mysql"
	DSN           string  // Data Source Name
	InTransaction bool
	LastError     string
	Managed       bool   // If true, connection is managed by host application (won't be closed by Parsley)
	SQLiteVersion string // SQLite version string (e.g., "3.45.0"), empty for non-SQLite
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
	RestrictWrite   []string // Denied write directories (blacklist)
	NoWrite         bool     // Deny all writes
	AllowWrite      []string // Allowed write directories (whitelist, used when AllowWriteAll is false)
	AllowWriteAll   bool     // Allow all writes (default true for pars)
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
	store         map[string]Object
	outer         *Environment
	Filename      string
	RootPath      string // Handler root directory for @~/ path resolution
	LastToken     *lexer.Token
	letBindings   map[string]bool // tracks which variables were declared with 'let'
	exports       map[string]bool // tracks which variables were explicitly exported
	protected     map[string]bool // tracks which variables cannot be reassigned
	Security      *SecurityPolicy // File system security policy
	Logger        Logger          // Logger for log()/logLine() output
	importStack   map[string]bool // tracks modules being imported (for circular dep detection)
	DevLog        DevLogWriter    // Dev log writer (nil in production mode)
	BasilCtx      Object          // Basil server context (request, db, auth, etc.)
	ServerDB      *DBConnection   // Server-level database connection (set at startup, available to modules)
	FragmentCache FragmentCacher  // Fragment cache for <basil.cache.Cache> (nil if not available)
	AssetRegistry AssetRegistrar  // Asset registry for publicUrl() (nil if not available)
	AssetBundle   AssetBundler    // Asset bundle for <Css/> and <Script/> tags (nil if not available)
	BasilJSURL    string          // URL for basil.js prelude script (for <BasilJS/> tag)
	HandlerPath   string          // Current handler path for cache key namespacing
	DevMode       bool            // Whether dev mode is enabled (affects caching)
	ContainsParts bool            // Whether the response contains <Part/> components (for JS injection)
	FormContext   *FormContext    // Current form context for @record/@field binding (FEAT-091)
	PLNSecret     string          // Secret for HMAC signing PLN in Part props (FEAT-098)
}

// NewEnvironment creates a new environment
func NewEnvironment() *Environment {
	return NewEnvironmentWithArgs(nil)
}

// NewEnvironmentWithArgs creates a new environment with @env and @args globals populated.
// This is the primary constructor for creating Parsley environments.
// - @env: dictionary of environment variables (read from os.Environ)
// - @args: array of command-line arguments (passed in, or empty if nil)
func NewEnvironmentWithArgs(args []string) *Environment {
	s := make(map[string]Object)
	l := make(map[string]bool)
	x := make(map[string]bool)
	p := make(map[string]bool)
	i := make(map[string]bool)
	env := &Environment{store: s, outer: nil, letBindings: l, exports: x, protected: p, importStack: i, Logger: DefaultLogger}

	// Populate @env from environment variables
	envPairs := make(map[string]ast.Expression)
	envKeys := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if key, value, ok := strings.Cut(e, "="); ok {
			envPairs[key] = &ast.ObjectLiteralExpression{Obj: &String{Value: value}}
			envKeys = append(envKeys, key)
		}
	}
	// Sort keys for deterministic iteration order
	sort.Strings(envKeys)
	env.store["@env"] = &Dictionary{Pairs: envPairs, KeyOrder: envKeys, Env: env}

	// Populate @args from provided arguments (or empty array)
	var argElements []Object
	if args != nil {
		argElements = make([]Object, len(args))
		for i, arg := range args {
			argElements[i] = &String{Value: arg}
		}
	} else {
		argElements = []Object{}
	}
	env.store["@args"] = &Array{Elements: argElements}

	return env
}

// NewEnclosedEnvironment creates a new environment with outer reference
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	// Preserve filename, token, logger, devlog, basilctx, serverdb, caches, and root path from outer environment
	if outer != nil {
		env.Filename = outer.Filename
		env.RootPath = outer.RootPath
		env.LastToken = outer.LastToken
		env.Logger = outer.Logger
		env.DevLog = outer.DevLog
		env.BasilCtx = outer.BasilCtx
		env.ServerDB = outer.ServerDB
		env.FragmentCache = outer.FragmentCache
		env.AssetRegistry = outer.AssetRegistry
		env.AssetBundle = outer.AssetBundle
		env.BasilJSURL = outer.BasilJSURL
		env.HandlerPath = outer.HandlerPath
		env.DevMode = outer.DevMode
		env.ContainsParts = outer.ContainsParts
		env.FormContext = outer.FormContext // Propagate form context (FEAT-091)
		env.PLNSecret = outer.PLNSecret     // Propagate PLN secret for Record serialization
	}
	return env
}

func logDeprecation(env *Environment, callRepr, suggestion string) {
	if env == nil || env.DevLog == nil {
		return
	}

	filename := env.Filename
	line := 0
	if env.LastToken != nil {
		line = env.LastToken.Line
	}

	message := fmt.Sprintf("%s is deprecated; use %s", callRepr, suggestion)
	if err := env.DevLog.LogFromEvaluator(env.HandlerPath, "warn", filename, line, callRepr, message); err != nil {
		fmt.Printf("[WARN] deprecation log failed: %v\n", err)
	}
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

// IsExported checks if a variable is explicitly exported
func (e *Environment) IsExported(name string) bool {
	return e.exports[name]
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

// UserVariables returns a map of user-defined variables (excluding builtins).
// This is used by the REPL to show what's in scope.
func (e *Environment) UserVariables() map[string]Object {
	result := make(map[string]Object)
	builtins := getBuiltins()

	// Walk through all scopes
	env := e
	for env != nil {
		for name, val := range env.store {
			// Skip if already seen (inner scope shadows outer)
			if _, exists := result[name]; exists {
				continue
			}
			// Skip builtins
			if _, isBuiltin := builtins[name]; isBuiltin {
				continue
			}
			result[name] = val
		}
		env = env.outer
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
// Keys are sorted for deterministic output when no order is specified
func NewDictionaryFromObjects(pairs map[string]Object) *Dictionary {
	// Sort keys for deterministic behavior
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return NewDictionaryFromObjectsWithOrder(pairs, keys)
}

// NewDictionaryFromObjectsWithOrder creates a Dictionary with specific key order
func NewDictionaryFromObjectsWithOrder(pairs map[string]Object, keyOrder []string) *Dictionary {
	dict := &Dictionary{
		Pairs:    make(map[string]ast.Expression),
		KeyOrder: keyOrder,
		Env:      NewEnvironment(),
	}
	for k, v := range pairs {
		dict.Pairs[k] = &ast.ObjectLiteralExpression{Obj: v}
	}
	return dict
}

// Path security helpers moved to eval_helpers.go (checkPathAccess, isPathAllowed, isPathRestricted)

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

// InvalidateModule removes a specific module from the cache.
// The path can be absolute or relative - both will be tried.
// This also invalidates any modules that might have imported the changed file.
func InvalidateModule(path string) {
	moduleCache.mu.Lock()
	defer moduleCache.mu.Unlock()

	// Try to find and remove the module by various path forms
	absPath, err := filepath.Abs(path)
	if err == nil {
		delete(moduleCache.modules, absPath)
	}
	delete(moduleCache.modules, path)

	// Also invalidate all modules that might have imported this one
	// This is a conservative approach - we clear all modules that could
	// transitively depend on the changed file
	// For now, just clear all modules on any file change to ensure correctness
	// TODO: Track import dependencies for selective invalidation
	moduleCache.modules = make(map[string]*Dictionary)
}

// ClearDBConnections closes and clears all cached database connections.
// This is primarily used in tests to ensure isolation between test cases.
func ClearDBConnections() {
	dbCache.close()
	// Recreate the cache
	dbCache = newConnectionCache[*sql.DB](
		100,            // max 100 database connections
		30*time.Minute, // 30 minute TTL
		func(db *sql.DB) error {
			return db.Ping()
		},
		func(db *sql.DB) error {
			return db.Close()
		},
	)
}

// GetDictValue is an exported helper for tests to get a value from a Dictionary.
func GetDictValue(dict *Dictionary, key string) Object {
	expr, ok := dict.Pairs[key]
	if !ok {
		return nil
	}
	return Eval(expr, dict.Env)
}

// BuildTestBasilContext creates a BasilCtx dictionary for testing.
// queryParams: query string parameters (?name=value)
// route: route path segments (or nil for null route)
// session: session data
func BuildTestBasilContext(queryParams map[string]string, route []string, session map[string]string) Object {
	// Build query dictionary
	queryPairs := make(map[string]ast.Expression)
	for k, v := range queryParams {
		queryPairs[k] = &ast.ObjectLiteralExpression{Obj: &String{Value: v}}
	}
	queryDict := &Dictionary{Pairs: queryPairs}

	// Build route (as string if provided, null otherwise)
	var routeObj Object = NULL
	if route != nil && len(route) > 0 {
		routeObj = &String{Value: "/" + strings.Join(route, "/")}
	}

	// Build request dictionary
	requestPairs := map[string]ast.Expression{
		"query":  &ast.ObjectLiteralExpression{Obj: queryDict},
		"route":  &ast.ObjectLiteralExpression{Obj: routeObj},
		"method": &ast.ObjectLiteralExpression{Obj: &String{Value: "GET"}},
	}
	requestDict := &Dictionary{Pairs: requestPairs}

	// Build http dictionary
	httpPairs := map[string]ast.Expression{
		"request":  &ast.ObjectLiteralExpression{Obj: requestDict},
		"response": &ast.ObjectLiteralExpression{Obj: NULL},
	}
	httpDict := &Dictionary{Pairs: httpPairs}

	// Build session dictionary
	sessionPairs := make(map[string]ast.Expression)
	for k, v := range session {
		sessionPairs[k] = &ast.ObjectLiteralExpression{Obj: &String{Value: v}}
	}
	sessionDict := &Dictionary{Pairs: sessionPairs}

	// Build basil root context
	basilPairs := map[string]ast.Expression{
		"http":    &ast.ObjectLiteralExpression{Obj: httpDict},
		"session": &ast.ObjectLiteralExpression{Obj: sessionDict},
		"auth":    &ast.ObjectLiteralExpression{Obj: NULL},
		"sqlite":  &ast.ObjectLiteralExpression{Obj: NULL},
	}
	return &Dictionary{Pairs: basilPairs}
}

// Natural sorting functions have been moved to eval_helpers.go

// objectsEqual compares two objects for equality (defined in eval_helpers.go)

// Datetime/duration conversion functions moved to eval_datetime.go (timeToDictWithKind, timeToDict, dictToTime, durationToDict, etc.)

// Type checking helpers moved to eval_helpers.go (isDatetimeDict, isDurationDict, etc.)

// Datetime/duration conversion helpers moved to eval_datetime.go (getDurationComponents, getDatetimeKind, getDatetimeUnix, datetimeDictToString, durationDictToString, etc.)

// Locale helpers moved to eval_locale.go (getMondayLocale, getDateFormatForStyle)

// Formatting helpers moved to eval_locale.go (formatNumberWithLocale, formatCurrencyWithLocale, formatPercentWithLocale, formatDateWithStyleAndLocale)

// Dictionary-to-string conversion functions moved to eval_dict_to_string.go (regexDictToString, fileDictToString, dirDictToString, requestDictToString, tagDictToString, pathDictToString, urlDictToString)

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

func evalDatetimeNowLiteral(node *ast.DatetimeNowLiteral, env *Environment) Object {
	kind := node.Kind
	if kind == "" {
		kind = "datetime"
	}

	now := time.Now()
	return timeToDictWithKind(now, kind, env)
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
				return newFormatError("FMT-0004", fmt.Errorf("invalid time literal: %s", node.Value))
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
				// Try date with single-digit month/day (2024-3-18) - interpret as UTC
				t, err = time.ParseInLocation("2006-1-2", node.Value, time.UTC)
				if err != nil {
					// Try datetime without timezone (2024-12-25T14:30:05) - interpret as UTC
					t, err = time.ParseInLocation("2006-01-02T15:04:05", node.Value, time.UTC)
					if err != nil {
						// Try datetime with single-digit month/day (2024-3-18T14:30:05) - interpret as UTC
						t, err = time.ParseInLocation("2006-1-2T15:04:05", node.Value, time.UTC)
						if err != nil {
							return newFormatError("FMT-0004", fmt.Errorf("cannot parse %q", node.Value))
						}
					}
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

func evalConnectionLiteral(node *ast.ConnectionLiteral, env *Environment) Object {
	if node.Kind == "db" {
		return resolveDBLiteral(env)
	}

	if node.Kind == "search" {
		return resolveSearchLiteral(env)
	}

	builtin := connectionBuiltins()[node.Kind]
	if builtin == nil {
		return newInternalError("INTERNAL-0002", map[string]any{"Type": "connection literal"})
	}
	return builtin
}

// resolveSearchLiteral returns the @SEARCH builtin factory or an error when unavailable.
func resolveSearchLiteral(env *Environment) Object {
	// @SEARCH is only available in Basil server context
	// It's registered as a protected variable in the environment
	if env != nil {
		if searchBuiltin, ok := env.Get("SEARCH"); ok {
			return searchBuiltin
		}
	}

	return &Error{
		Class:   ErrorClass("state"),
		Message: "@SEARCH is only available in Basil server context",
		Hints:   []string{"Run inside a Basil handler", "@SEARCH requires the Basil server environment"},
	}
}

// resolveDBLiteral returns the Basil-managed database connection or an error when unavailable.
func resolveDBLiteral(env *Environment) Object {
	// 1. Try server-level database first (available at module load time)
	if env != nil && env.ServerDB != nil {
		return env.ServerDB
	}

	// 2. Fall back to BasilCtx["sqlite"] (backward compatibility)
	var basilObj Object
	if env != nil {
		basilObj = env.BasilCtx
		if basilObj == nil {
			if candidate, ok := env.Get("basil"); ok {
				basilObj = candidate
			}
		}
	}

	basilDict, ok := basilObj.(*Dictionary)
	if !ok || basilDict == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "@DB is only available in Basil server context",
			Hints:   []string{"Run inside a Basil handler or module with a configured database"},
		}
	}

	sqliteExpr, ok := basilDict.Pairs["sqlite"]
	if !ok {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "@DB is only available in Basil server context",
			Hints:   []string{"Ensure the server has a configured database connection"},
		}
	}

	evalEnv := basilDict.Env
	if evalEnv == nil {
		evalEnv = env
	}

	connObj := Eval(sqliteExpr, evalEnv)
	conn, ok := connObj.(*DBConnection)
	if !ok {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "@DB is only available in Basil server context",
			Hints:   []string{"Ensure the server has a configured database connection"},
		}
	}

	return conn
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
	// Position offset: token points to @, content starts after @(, so column + 2
	interpolated := interpolatePathUrlTemplate(node.Value, env, node.Token.Line, node.Token.Column+2)
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
	// Position offset: token points to @, content starts after @(, so column + 2
	interpolated := interpolatePathUrlTemplate(node.Value, env, node.Token.Line, node.Token.Column+2)
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
	// Position offset: token points to @, content starts after @(, so column + 2
	interpolated := interpolatePathUrlTemplate(node.Value, env, node.Token.Line, node.Token.Column+2)
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
				return newFormatError("FMT-0004", fmt.Errorf("invalid time in datetime template: %s", datetimeStr))
			}
		}

		// Combine with current UTC date
		t = time.Date(now.Year(), now.Month(), now.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	} else {
		// Check for date-only vs full datetime by looking for 'T' separator
		// Date-only: YYYY-MM-DD or YYYY-M-D (no 'T')
		// DateTime: YYYY-MM-DDTHH:MM:SS (has 'T')
		if !strings.Contains(datetimeStr, "T") {
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
				// Try date with single-digit month/day (2024-3-18) - interpret as UTC
				t, err = time.ParseInLocation("2006-1-2", datetimeStr, time.UTC)
				if err != nil {
					// Try datetime without timezone (2024-12-25T14:30:05) - interpret as UTC
					t, err = time.ParseInLocation("2006-01-02T15:04:05", datetimeStr, time.UTC)
					if err != nil {
						// Try datetime with single-digit month/day (2024-3-18T14:30:05) - interpret as UTC
						t, err = time.ParseInLocation("2006-1-2T15:04:05", datetimeStr, time.UTC)
						if err != nil {
							return newFormatError("FMT-0004", fmt.Errorf("cannot parse %q", datetimeStr))
						}
					}
				}
			}
		}
	}

	// Convert to dictionary using the function with kind
	return timeToDictWithKind(t, kind, env)
}

// interpolatePathUrlTemplate processes {expr} interpolations in path/URL templates
// This is similar to evalTemplateLiteral but returns a String object.
// baseLine and baseCol specify the position of the template content start (after @()
// so that errors within interpolations can report correct source positions.
func interpolatePathUrlTemplate(template string, env *Environment, baseLine, baseCol int) Object {
	var result strings.Builder

	i := 0
	for i < len(template) {
		// Look for {
		if template[i] == '{' {
			// Record position of the opening brace for error reporting
			// The expression starts after the {, so add 1 to the offset
			exprOffset := i + 1

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

			// Parse and evaluate the expression (with filename for error reporting)
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				// Adjust position: add template offset to error position
				// The error's line/column are relative to the expression string (starting at 1,1)
				// We need to adjust based on where in the template this expression appears
				adjustedCol := baseCol + exprOffset + (perr.Column - 1)
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    baseLine,
					Column:  adjustedCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors too
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = baseLine
						errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
						if errObj.Column < baseCol+exprOffset {
							errObj.Column = baseCol + exprOffset
						}
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
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
// Duration parsing moved to eval_parsing.go (parseDurationString, isDigit)

// Type checking helpers moved to eval_helpers.go (typeExprEquals, isRegexDict, isPathDict, isUrlDict, isFileDict, isTagDict)

// Regex functions moved to eval_regex.go (compileRegex, evalMatchExpression)

// Path-related functions moved to eval_paths.go (cleanPathComponents, parsePathString, pathToDict, stdioToDict, fileToDict, dirToDict)

// parseUrlString parses a URL string into components
// Supports: scheme://[user:pass@]host[:port]/path?query#fragment
// URL parsing functions are in eval_urls.go:
// - parseUrlString
// - parseURLToDict
// - urlToRequestDict
// - requestToDict

// Computed property evaluators moved to eval_computed_properties.go:
// - evalPathComputedProperty
// - evalDirComputedProperty
// - evalFileComputedProperty
// - evalUrlComputedProperty

// isDirDict moved to eval_helpers.go

// fileDictToPathDict converts a file/dir dictionary to a path dictionary
// File dicts use _pathComponents/_pathAbsolute, path dicts use components/absolute
// File/path helper functions moved to eval_paths.go:
// - fileDictToPathDict
// - coerceToPathDict
// - getFilePathString
// - readDirContents
// - inferFormatFromExtension

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

// Datetime and duration computed property evaluators are in eval_computed_properties.go

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

// getSQLiteVersionFromDB queries the SQLite version from a database connection
func getSQLiteVersionFromDB(db *sql.DB) string {
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return "" // Non-fatal - return empty string
	}
	return version
}

// sqliteSupportsReturning checks if the SQLite version supports RETURNING clause (3.35.0+)
func sqliteSupportsReturning(version string) bool {
	if version == "" {
		return false // Unknown version, assume no support
	}
	// Parse version string (e.g., "3.45.0" or "3.35.0")
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	// RETURNING support added in 3.35.0
	if major > 3 {
		return true
	}
	if major == 3 && minor >= 35 {
		return true
	}
	return false
}

// connectionBuiltins defines callable constructors for connection literals like @sqlite and @shell.
func connectionBuiltins() map[string]*Builtin {
	return map[string]*Builtin{
		"sqlite": {
			Fn: func(args ...Object) Object {
				callName := "@sqlite"

				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange(callName, len(args), 1, 2)
				}

				// First arg: path literal
				pathStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", callName, "a path", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", callName, "a dictionary", args[1].Type())
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
				db, exists := dbCache.get(cacheKey)

				if !exists {
					var err error
					// Open with WAL mode for better concurrency and busy timeout for locking
					// Skip pragmas for :memory: databases as WAL doesn't work with them
					connStr := dsn
					if dsn != ":memory:" {
						connStr = dsn + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
					}
					db, err = sql.Open("sqlite", connStr)
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

					// Cache connection (TTL and health checks handled by cache)
					dbCache.put(cacheKey, db)
				}

				// Detect SQLite version for this connection
				version := getSQLiteVersionFromDB(db)

				return &DBConnection{
					DB:            db,
					Driver:        "sqlite",
					DSN:           dsn,
					InTransaction: false,
					LastError:     "",
					SQLiteVersion: version,
				}
			},
		},
		"postgres": {
			Fn: func(args ...Object) Object {
				callName := "@postgres"

				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange(callName, len(args), 1, 2)
				}

				// First arg: URL literal
				urlStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", callName, "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", callName, "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				dsn := urlStr.Value

				// Check cache
				cacheKey := "postgres:" + dsn
				db, exists := dbCache.get(cacheKey)

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

					// Cache connection (TTL and health checks handled by cache)
					dbCache.put(cacheKey, db)
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
		"mysql": {
			Fn: func(args ...Object) Object {
				callName := "@mysql"

				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange(callName, len(args), 1, 2)
				}

				// First arg: URL literal
				urlStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", callName, "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", callName, "a dictionary", args[1].Type())
					}
					options = make(map[string]Object)
					for key := range dict.Pairs {
						options[key] = Eval(dict.Pairs[key], dict.Env)
					}
				}

				dsn := urlStr.Value

				// Check cache
				cacheKey := "mysql:" + dsn
				db, exists := dbCache.get(cacheKey)

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

					// Cache connection (TTL and health checks handled by cache)
					dbCache.put(cacheKey, db)
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
		"sftp": {
			Fn: func(args ...Object) Object {
				callName := "@sftp"

				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange(callName, len(args), 1, 2)
				}

				// First arg: URL (can be dictionary or string)
				var urlStr string
				switch arg := args[0].(type) {
				case *Dictionary:
					if !isUrlDict(arg) {
						return newTypeError("TYPE-0005", callName, "a URL", DICTIONARY_OBJ)
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
					return newTypeError("TYPE-0005", callName, "a URL", args[0].Type())
				}

				// Optional second arg: options dictionary
				var options map[string]Object
				if len(args) == 2 {
					dict, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", callName, "a dictionary", args[1].Type())
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
				conn, exists := sftpCache.get(cacheKey)

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
								return newNetworkError("NET-0008", err)
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

				// Cache connection (TTL and health checks handled by cache)
				sftpCache.put(cacheKey, newConn)

				return newConn
			},
		},
		"shell": {
			FnWithEnv: func(env *Environment, args ...Object) Object {
				callName := "@shell"

				if len(args) < 1 || len(args) > 3 {
					return newArityErrorRange(callName, len(args), 1, 3)
				}

				effectiveEnv := env
				if effectiveEnv == nil {
					effectiveEnv = NewEnvironment()
				}

				// First argument: binary name/path (string)
				binary, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", callName, "a string", args[0].Type())
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
								return newCommandError("CMD-0003", nil)
							}
						}
					} else {
						return newTypeError("TYPE-0006", callName, "an array", args[1].Type())
					}
				}

				// Third argument (optional): options dict
				var options *Dictionary
				if len(args) >= 3 {
					if optDict, ok := args[2].(*Dictionary); ok {
						options = optDict
					} else {
						return newTypeError("TYPE-0011", callName, "a dictionary", args[2].Type())
					}
				}

				return createCommandHandle(binary.Value, cmdArgs, options, effectiveEnv)
			},
		},
	}
}

// getBuiltins returns the map of built-in functions
//
// IMPORTANT: When adding, modifying, or removing builtins, update the BuiltinMetadata
// map in introspect.go to keep introspection data in sync. See .github/instructions/code.instructions.md
// for maintenance checklist.
func getBuiltins() map[string]*Builtin {
	return map[string]*Builtin{
		"import": {
			Fn: func(args ...Object) Object {
				// This is a placeholder - actual implementation happens in CallExpression
				// where we have access to the environment for path resolution
				return newInternalError("INTERNAL-0001", map[string]any{"Context": "import()"})
			},
		},
		// date() - Parse a date string with flexible format support
		// date("22 April 2005") - Natural language date
		// date("2005-04-22") - ISO format
		// date("04/22/2005") - US format (default)
		// date("22/04/2005", {locale: "en-GB"}) - UK format
		"date": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("date", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Parse input string
				input, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "date", "a string", args[0].Type())
				}

				// Extract options
				locale := "en-US"
				strict := false
				timezone := "UTC"
				if len(args) == 2 {
					opts, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "date", "a dictionary", args[1].Type())
					}
					locale, strict, timezone = extractParseOptions(opts, env)
					_ = strict // strict mode handled in parseFlexibleDateTime
				}

				// Parse with locale awareness
				localeConfig := getLocaleConfig(locale)
				t, _, err := parseFlexibleDateTime(input.Value, localeConfig, timezone, strict)
				if err != nil {
					return newStructuredError("FMT-0011", map[string]any{
						"Input":   input.Value,
						"GoError": err.Error(),
					})
				}

				// Return as date (strip time component to midnight)
				dateOnly := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				return timeToDictWithKind(dateOnly, "date", env)
			},
		},
		// time() - Parse a time-only string
		// time("3:45 PM") - 12-hour format
		// time("15:45") - 24-hour format
		// time("15:45:30.123") - With seconds and milliseconds
		"time": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("time", len(args), 1)
				}

				env := NewEnvironment()

				input, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "time", "a string", args[0].Type())
				}

				// Parse time-only
				t, err := parseTimeOnly(input.Value)
				if err != nil {
					return newStructuredError("FMT-0011", map[string]any{
						"Input":   input.Value,
						"GoError": err.Error(),
					})
				}

				return timeToDictWithKind(t, "time", env)
			},
		},
		// datetime() - Parse a full datetime with flexible format support
		// datetime("April 22, 2005 3:45 PM") - Natural language
		// datetime("2005-04-22T15:45:00Z") - ISO 8601
		// datetime(1682157900) - Unix timestamp
		// datetime({year: 2005, month: 4, day: 22}) - From dict
		// datetime("01/02/2005 3pm", {locale: "en-GB"}) - With locale
		"datetime": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("datetime", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Extract options if provided
				locale := "en-US"
				strict := false
				timezone := "UTC"
				var delta *Dictionary

				if len(args) == 2 {
					opts, ok := args[1].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0006", "datetime", "a dictionary", args[1].Type())
					}

					// Check if it's options or a delta dict
					// Options have: locale, strict, timezone
					// Delta has: years, months, days, hours, minutes, seconds
					_, hasLocale := opts.Pairs["locale"]
					_, hasStrict := opts.Pairs["strict"]
					_, hasTimezone := opts.Pairs["timezone"]
					_, hasYears := opts.Pairs["years"]
					_, hasMonths := opts.Pairs["months"]
					_, hasDays := opts.Pairs["days"]
					_, hasHours := opts.Pairs["hours"]
					_, hasMinutes := opts.Pairs["minutes"]
					_, hasSeconds := opts.Pairs["seconds"]

					if hasLocale || hasStrict || hasTimezone {
						locale, strict, timezone = extractParseOptions(opts, env)
					} else if hasYears || hasMonths || hasDays || hasHours || hasMinutes || hasSeconds {
						delta = opts
					}
				}

				var t time.Time
				var err error

				switch arg := args[0].(type) {
				case *String:
					// Parse with locale awareness
					localeConfig := getLocaleConfig(locale)
					t, _, err = parseFlexibleDateTime(arg.Value, localeConfig, timezone, strict)
					if err != nil {
						return newStructuredError("FMT-0011", map[string]any{
							"Input":   arg.Value,
							"GoError": err.Error(),
						})
					}
				case *Integer:
					// Unix timestamp
					t = time.Unix(arg.Value, 0).UTC()
				case *Dictionary:
					// From dictionary components
					t, err = dictToTime(arg, env)
					if err != nil {
						return newFormatError("FMT-0004", err)
					}
				default:
					return newTypeError("TYPE-0012", "datetime", "a string, integer, or dictionary", args[0].Type())
				}

				// Apply delta if provided
				if delta != nil {
					t = applyDelta(t, delta, env)
				}

				return timeToDictWithKind(t, "datetime", env)
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
		// path() - create a Path from a string
		// path("./relative/path") - relative path
		// path("/absolute/path") - absolute path
		"path": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("path", len(args), 1)
				}

				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "path", "a string", args[0].Type())
				}

				env := NewEnvironment()
				components, isAbsolute := parsePathString(str.Value)
				return pathToDict(components, isAbsolute, env)
			},
		},
		// duration() - create a Duration from a string or dictionary
		// duration("1d2h30m") - parse duration string
		// duration({days: 1, hours: 2}) - from components
		// duration({months: 6, seconds: 3600}) - raw months/seconds
		"duration": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("duration", len(args), 1)
				}

				env := NewEnvironment()

				switch arg := args[0].(type) {
				case *String:
					// Parse duration string like "1d2h30m"
					months, seconds, err := parseDurationString(arg.Value)
					if err != nil {
						return newFormatError("FMT-0009", err)
					}
					return durationToDict(months, seconds, env)

				case *Dictionary:
					// From dictionary with components
					var months int64
					var seconds int64

					// Check for raw months/seconds first
					if monthsExpr, ok := arg.Pairs["months"]; ok {
						monthsObj := Eval(monthsExpr, arg.Env)
						if monthsInt, ok := monthsObj.(*Integer); ok {
							months = monthsInt.Value
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for months", monthsObj.Type())
						}
					}
					if secondsExpr, ok := arg.Pairs["seconds"]; ok {
						secondsObj := Eval(secondsExpr, arg.Env)
						if secondsInt, ok := secondsObj.(*Integer); ok {
							seconds = secondsInt.Value
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for seconds", secondsObj.Type())
						}
					}

					// Handle named components: years, months, weeks, days, hours, minutes, seconds
					if yearsExpr, ok := arg.Pairs["years"]; ok {
						yearsObj := Eval(yearsExpr, arg.Env)
						if yearsInt, ok := yearsObj.(*Integer); ok {
							months += yearsInt.Value * 12
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for years", yearsObj.Type())
						}
					}
					// Note: "months" already handled above for raw value

					if weeksExpr, ok := arg.Pairs["weeks"]; ok {
						weeksObj := Eval(weeksExpr, arg.Env)
						if weeksInt, ok := weeksObj.(*Integer); ok {
							seconds += weeksInt.Value * 7 * 24 * 60 * 60
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for weeks", weeksObj.Type())
						}
					}
					if daysExpr, ok := arg.Pairs["days"]; ok {
						daysObj := Eval(daysExpr, arg.Env)
						if daysInt, ok := daysObj.(*Integer); ok {
							seconds += daysInt.Value * 24 * 60 * 60
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for days", daysObj.Type())
						}
					}
					if hoursExpr, ok := arg.Pairs["hours"]; ok {
						hoursObj := Eval(hoursExpr, arg.Env)
						if hoursInt, ok := hoursObj.(*Integer); ok {
							seconds += hoursInt.Value * 60 * 60
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for hours", hoursObj.Type())
						}
					}
					if minutesExpr, ok := arg.Pairs["minutes"]; ok {
						minutesObj := Eval(minutesExpr, arg.Env)
						if minutesInt, ok := minutesObj.(*Integer); ok {
							seconds += minutesInt.Value * 60
						} else {
							return newTypeError("TYPE-0012", "duration", "an integer for minutes", minutesObj.Type())
						}
					}
					// Note: "seconds" already handled above for raw value

					return durationToDict(months, seconds, env)

				default:
					return newTypeError("TYPE-0012", "duration", "a string or dictionary", args[0].Type())
				}
			},
		},
		// File handle factories
		"file": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("file", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "file", "a path, file, or string", args[0].Type())
				}

				// Get the path string for format inference
				pathStr := getFilePathString(&Dictionary{Pairs: map[string]ast.Expression{
					"_pathComponents": pathDict.Pairs["segments"],
					"_pathAbsolute":   pathDict.Pairs["absolute"],
				}, Env: pathEnv}, pathEnv)

				// Auto-detect format from extension
				format := inferFormatFromExtension(pathStr)

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				return fileToDict(pathDict, format, options, pathEnv)
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

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "json", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "JSON", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "json", options, pathEnv)
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
		"PLN": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("PLN", len(args), 1, 2)
				}

				// First argument must be a path dictionary or string
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
					if !isPathDict(arg) {
						return newTypeError("TYPE-0005", "PLN", "a path", DICTIONARY_OBJ)
					}
					pathDict = arg
				case *String:
					components, isAbsolute := parsePathString(arg.Value)
					pathDict = pathToDict(components, isAbsolute, env)
				default:
					return newTypeError("TYPE-0005", "PLN", "a path or string", args[0].Type())
				}

				return fileToDict(pathDict, "pln", options, env)
			},
		},
		"CSV": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("CSV", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict (e.g., {header: true})
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "csv", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "CSV", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "csv", options, pathEnv)
			},
		},
		"lines": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("lines", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "lines", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "lines", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "lines", options, pathEnv)
			},
		},
		"text": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("text", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict (e.g., {encoding: "latin1"})
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "text", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "text", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "text", options, pathEnv)
			},
		},
		"bytes": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("bytes", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "bytes", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "bytes", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "bytes", options, pathEnv)
			},
		},
		// SVG file format - reads SVG files and strips XML prolog for use as components
		"SVG": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("SVG", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					return requestToDict(dict, "svg", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "SVG", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "svg", options, pathEnv)
			},
		},
		// MD file format - reads MD files with frontmatter support
		"MD": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("MD", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// First argument can be a path, file dict, or string
				// Check for URL dict first
				if dict, ok := args[0].(*Dictionary); ok && isUrlDict(dict) {
					// Create request dictionary for URL
					return requestToDict(dict, "md", options, env)
				}

				// Coerce to path dict (handles path, file, dir, string)
				pathDict, pathEnv := coerceToPathDict(args[0], env)
				if pathDict == nil {
					return newTypeError("TYPE-0005", "MD", "a path, file, or string", args[0].Type())
				}

				return fileToDict(pathDict, "markdown", options, pathEnv)
			},
		},
		// markdown function - parses markdown strings
		"markdown": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 2 {
					return newArityErrorRange("markdown", len(args), 1, 2)
				}

				env := NewEnvironment()

				// Second argument is optional options dict
				var options *Dictionary
				if len(args) == 2 {
					if optDict, ok := args[1].(*Dictionary); ok {
						options = optDict
					}
				}

				// First argument must be a string
				str, ok := args[0].(*String)
				if !ok {
					// Check if they passed a path and suggest MD() instead
					if _, isDict := args[0].(*Dictionary); isDict {
						return newTypeError("TYPE-0012", "markdown", "a string (use MD(@path) for files)", args[0].Type())
					}
					return newTypeError("TYPE-0012", "markdown", "a string", args[0].Type())
				}

				// Parse the markdown string
				result, err := parseMarkdown(str.Value, options, env)
				if err != nil {
					return err
				}
				return result
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
		"fileList": {
			Fn: func(args ...Object) Object {
				if len(args) < 1 || len(args) > 1 {
					return newArityError("fileList", len(args), 1)
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
						return newTypeError("TYPE-0012", "fileList", "a path or string pattern", DICTIONARY_OBJ)
					}
				case *String:
					pattern = arg.Value
					env = NewEnvironment()
				default:
					return newTypeError("TYPE-0012", "fileList", "a path or string pattern", args[0].Type())
				}

				// Track if original pattern was explicitly relative (./ or ../ prefix) BEFORE resolving
				wasExplicitlyRelative := strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../")

				// Resolve path based on prefix
				if strings.HasPrefix(pattern, "~/") {
					// Expand ~/ paths - in Parsley/Basil, ~/ means project root, not user home
					if env != nil && env.RootPath != "" {
						pattern = filepath.Join(env.RootPath, pattern[2:])
					} else {
						// Fallback to user home directory if no root path set
						home, err := os.UserHomeDir()
						if err == nil {
							pattern = filepath.Join(home, pattern[2:])
						}
					}
				} else if strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../") {
					// Resolve relative paths based on current file's directory (like import does)
					var baseDir string
					if env != nil && env.Filename != "" {
						baseDir = filepath.Dir(env.Filename)
					} else {
						// If no current file, use current working directory
						cwd, err := os.Getwd()
						if err != nil {
							return newIOError("IO-0003", ".", err)
						}
						baseDir = cwd
					}
					pattern = filepath.Join(baseDir, pattern)
				}

				// Use doublestar for ** glob patterns, fallback to filepath.Glob for simple patterns
				matches, err := filepath.Glob(pattern)
				if err != nil {
					return newValidationError("VAL-0003", map[string]any{"Pattern": pattern, "GoError": err.Error()})
				}

				// Convert matches to array of file handles
				elements := make([]Object, 0, len(matches))
				for _, match := range matches {
					info, statErr := os.Stat(match)
					if statErr != nil {
						continue
					}

					// If the original pattern was relative (./ or ../), convert absolute matches back to relative
					if wasExplicitlyRelative && filepath.IsAbs(match) {
						// Get the base directory we used for resolution
						var baseDir string
						if env != nil && env.Filename != "" {
							baseDir = filepath.Dir(env.Filename)
						} else {
							cwd, _ := os.Getwd()
							baseDir = cwd
						}
						// Convert to relative path
						relPath, err := filepath.Rel(baseDir, match)
						if err == nil {
							match = "./" + relPath
						}
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
							perr := perrors.New("VAL-0002", map[string]any{"Style": styleStr.Value, "Context": "`format`", "ValidOptions": "'and', 'or', or 'unit'"})
							return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
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
		// NOTE: map removed - use array method: [1,2,3].map(fn(x) { x * 2 })
		// NOTE: toUpper, toLower removed - use string methods: "text".toUpper(), "text".toLower()
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
		// NOTE: replace, split removed - use string methods: "text".replace(old, new), "text".split(delim)
		// match(path, pattern) - extract named parameters from URL paths
		// Returns dictionary on match, null on no match
		// Supports :name for single segment capture, *name for rest/glob capture
		"match": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("match", len(args), 2)
				}

				// First arg: path (string or path dict)
				var path string
				switch p := args[0].(type) {
				case *String:
					path = p.Value
				case *Dictionary:
					if isPathDict(p) {
						path = pathDictToString(p)
					} else {
						return newTypeError("TYPE-0005", "match", "a string or path", args[0].Type())
					}
				default:
					return newTypeError("TYPE-0005", "match", "a string or path", args[0].Type())
				}

				// Second arg: pattern (string)
				pattern, ok := args[1].(*String)
				if !ok {
					return newTypeError("TYPE-0006", "match", "a string", args[1].Type())
				}

				// Match the path against pattern
				result := matchPathPattern(path, pattern.Value)
				if result == nil {
					return NULL
				}

				// Convert result to Dictionary
				pairs := make(map[string]ast.Expression)
				for key, val := range result {
					switch v := val.(type) {
					case string:
						pairs[key] = createLiteralExpression(&String{Value: v})
					case []string:
						elements := make([]Object, len(v))
						for i, s := range v {
							elements[i] = &String{Value: s}
						}
						pairs[key] = createLiteralExpression(&Array{Elements: elements})
					}
				}

				return &Dictionary{Pairs: pairs, Env: NewEnvironment()}
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
		// asset() - converts a path under public_dir to a web URL
		// e.g., asset(@./public/images/foo.png) -> "/images/foo.png"
		// Also accepts file dictionaries from fileList() and extracts their path
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
						return newFileOpError("FILEOP-0002", nil)
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
				return &String{Value: objectToReprString(args[0])}
			},
		},
		// inspect() - returns introspection data as a dictionary
		"inspect": {
			Fn: builtinInspect,
		},
		// describe() - pretty prints introspection data
		"describe": {
			Fn: builtinDescribe,
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
		"print": {
			Fn: func(args ...Object) Object {
				if len(args) == 0 {
					return newArityError("print", 0, 1)
				}
				return &PrintValue{Values: args}
			},
		},
		"println": {
			Fn: func(args ...Object) Object {
				// println with no args just returns a newline
				if len(args) == 0 {
					return &PrintValue{Values: []Object{&String{Value: "\n"}}}
				}
				// Append newline after all values
				values := make([]Object, len(args)+1)
				copy(values, args)
				values[len(args)] = &String{Value: "\n"}
				return &PrintValue{Values: values}
			},
		},
		"printf": {
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return newArityError("printf", len(args), 2)
				}

				templateStr, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "printf", "a string (template)", args[0].Type())
				}

				dict, ok := args[1].(*Dictionary)
				if !ok {
					return newTypeError("TYPE-0006", "printf", "a dictionary (values)", args[1].Type())
				}

				renderEnv, errObj := buildRenderEnv(dict.Env, dict)
				if errObj != nil {
					return errObj
				}

				return interpolateRawString(templateStr.Value, renderEnv)
			},
		},
		"fail": {
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return newArityError("fail", len(args), 1)
				}

				msg, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0005", "fail", "a string", args[0].Type())
				}

				// Create a Value-class catchable error
				return &Error{
					Class:   ClassValue,
					Code:    "USER-0001",
					Message: msg.Value,
				}
			},
		},
		// NOTE: sort, reverse, sortBy removed - use array methods: arr.sort(), arr.reverse(), arr.sortBy(fn)
		// NOTE: keys, values, has removed - use dict methods: dict.keys(), dict.values(), dict.has(key)
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

				orderedKeys := dict.Keys()
				pairs := make([]Object, 0, len(orderedKeys))
				for _, key := range orderedKeys {
					val := Eval(dict.Pairs[key], dictEnv)

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
					Pairs:    make(map[string]ast.Expression),
					KeyOrder: []string{},
					Env:      NewEnvironment(),
				}

				for _, elem := range arr.Elements {
					pair, ok := elem.(*Array)
					if !ok || len(pair.Elements) != 2 {
						perr := perrors.New("TODICT-0001", nil)
						return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
					}

					keyObj, ok := pair.Elements[0].(*String)
					if !ok {
						perr := perrors.New("TODICT-0002", map[string]any{"Got": string(pair.Elements[0].Type())})
						return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
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
						perr := perrors.New("TODICT-0003", map[string]any{"Got": string(valueObj.Type())})
						return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
					}

					dict.SetKey(keyObj.Value, expr)
				}

				return dict
			},
		},
		// serialize() - convert value to PLN string
		"serialize": {
			FnWithEnv: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return newArityError("serialize", len(args), 1)
				}
				return SerializeToPLN(args[0], env)
			},
		},
		// deserialize() - parse PLN string to value
		"deserialize": {
			FnWithEnv: func(env *Environment, args ...Object) Object {
				if len(args) != 1 {
					return newArityError("deserialize", len(args), 1)
				}
				str, ok := args[0].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "deserialize", "a string", args[0].Type())
				}
				return DeserializeFromPLN(str.Value, env)
			},
		},
		// table() - create a Table from an array of dictionaries
		// table() - empty table
		// table(array) - table from array of dictionaries with rectangular validation
		"table": {
			Fn: func(args ...Object) Object {
				env := NewEnvironment()
				return TableConstructor(args, env)
			},
		},
		// money() - create a Money value from amount and currency
		// money(amount, currency) - amount is integer/float, currency is 3-letter code
		// money(amount, currency, scale) - explicit scale (decimal places)
		// money(dict) - create from dictionary with "amount" and "currency" keys
		"money": {
			Fn: func(args ...Object) Object {
				// Handle dictionary form: money({amount: 50.00, currency: "USD"})
				if len(args) == 1 {
					dict, ok := args[0].(*Dictionary)
					if !ok {
						return newTypeError("TYPE-0012", "money", "a dictionary or (amount, currency)", args[0].Type())
					}

					// Extract amount
					amountExpr, hasAmount := dict.Pairs["amount"]
					if !hasAmount {
						return newValidationError("VAL-0008", map[string]any{"Type": "money dictionary missing 'amount' key"})
					}
					var amountObj Object
					if obj, isObj := amountExpr.(Object); isObj {
						amountObj = obj
					} else {
						// Need to evaluate the expression
						amountObj = Eval(amountExpr, dict.Env)
						if isError(amountObj) {
							return amountObj
						}
					}

					// Extract currency
					currencyExpr, hasCurrency := dict.Pairs["currency"]
					if !hasCurrency {
						return newValidationError("VAL-0008", map[string]any{"Type": "money dictionary missing 'currency' key"})
					}
					var currencyObj Object
					if obj, isObj := currencyExpr.(Object); isObj {
						currencyObj = obj
					} else {
						currencyObj = Eval(currencyExpr, dict.Env)
						if isError(currencyObj) {
							return currencyObj
						}
					}

					// Parse the extracted values directly (inline, don't recurse)
					var amountCents int64
					var inferredScale int8 = 2

					switch a := amountObj.(type) {
					case *Integer:
						amountCents = a.Value * 100 // Convert to cents
					case *Float:
						floatStr := fmt.Sprintf("%g", a.Value)
						if dotIdx := strings.Index(floatStr, "."); dotIdx >= 0 {
							inferredScale = int8(len(floatStr) - dotIdx - 1)
						}
						multiplier := math.Pow10(int(inferredScale))
						amountCents = int64(math.Round(a.Value * multiplier))
					default:
						return newTypeError("TYPE-0012", "money", "a number for amount", amountObj.Type())
					}

					currencyStr, ok := currencyObj.(*String)
					if !ok {
						return newTypeError("TYPE-0012", "money", "a currency code string", currencyObj.Type())
					}
					currency := strings.ToUpper(currencyStr.Value)
					if len(currency) != 3 {
						return newStructuredError("VAL-0019", map[string]any{"Got": currencyStr.Value})
					}

					// Use known currency scale if available
					scale := inferredScale
					if knownScale, ok := lexer.CurrencyScales[currency]; ok {
						if inferredScale != knownScale {
							diff := int(knownScale - inferredScale)
							if diff > 0 {
								amountCents *= int64(math.Pow10(diff))
							} else {
								amountCents /= int64(math.Pow10(-diff))
							}
						}
						scale = knownScale
					}

					return &Money{
						Amount:   amountCents,
						Currency: currency,
						Scale:    scale,
					}
				}

				if len(args) < 2 || len(args) > 3 {
					return newArityErrorRange("money", len(args), 2, 3)
				}

				// Parse amount (integer or float)
				var amountCents int64
				var inferredScale int8 = 2 // Default to 2 decimal places

				switch a := args[0].(type) {
				case *Integer:
					// Integer amount - assume it's already in minor units if scale provided,
					// otherwise treat as major units
					if len(args) == 3 {
						amountCents = a.Value
					} else {
						amountCents = a.Value * 100 // Convert to cents
					}
				case *Float:
					// Float amount - convert to minor units
					// Count decimal places in the float for scale inference
					floatStr := fmt.Sprintf("%g", a.Value)
					if dotIdx := strings.Index(floatStr, "."); dotIdx >= 0 {
						inferredScale = int8(len(floatStr) - dotIdx - 1)
					}
					// Convert to minor units: multiply by 10^scale
					multiplier := math.Pow10(int(inferredScale))
					amountCents = int64(math.Round(a.Value * multiplier))
				default:
					return newTypeError("TYPE-0012", "money", "a number", args[0].Type())
				}

				// Parse currency code
				currencyStr, ok := args[1].(*String)
				if !ok {
					return newTypeError("TYPE-0012", "money", "a currency code string", args[1].Type())
				}

				currency := strings.ToUpper(currencyStr.Value)
				if len(currency) != 3 {
					return newStructuredError("VAL-0019", map[string]any{"Got": currencyStr.Value})
				}

				// Parse optional scale
				scale := inferredScale
				if len(args) == 3 {
					scaleInt, ok := args[2].(*Integer)
					if !ok {
						return newTypeError("TYPE-0012", "money", "a scale integer", args[2].Type())
					}
					scale = int8(scaleInt.Value)
					if scale < 0 || scale > 10 {
						return newStructuredError("VAL-0020", map[string]any{"Got": scaleInt.Value})
					}
				} else {
					// Use known currency scale if available
					if knownScale, ok := lexer.CurrencyScales[currency]; ok {
						// Need to adjust amount if inferredScale differs from knownScale
						if inferredScale != knownScale {
							diff := int(knownScale - inferredScale)
							if diff > 0 {
								amountCents *= int64(math.Pow10(diff))
							} else {
								amountCents /= int64(math.Pow10(-diff))
							}
						}
						scale = knownScale
					}
				}

				return &Money{
					Amount:   amountCents,
					Currency: currency,
					Scale:    scale,
				}
			},
		},
		// builtins() - list all builtin functions by category
		"builtins": {
			Fn: func(args ...Object) Object {
				if len(args) > 1 {
					return newArityErrorRange("builtins", len(args), 0, 1)
				}

				// Optional category filter
				var categoryFilter string
				if len(args) == 1 {
					cat, ok := args[0].(*String)
					if !ok {
						return newTypeError("TYPE-0012", "builtins", "a category string", args[0].Type())
					}
					categoryFilter = cat.Value
				}

				// Group builtins by category
				categories := make(map[string][]BuiltinInfo)
				for _, metadata := range BuiltinMetadata {
					if categoryFilter != "" && metadata.Category != categoryFilter {
						continue
					}
					categories[metadata.Category] = append(categories[metadata.Category], metadata)
				}

				// Sort categories
				catNames := make([]string, 0, len(categories))
				for cat := range categories {
					catNames = append(catNames, cat)
				}
				sort.Strings(catNames)

				// Build result dictionary
				resultPairs := make(map[string]ast.Expression)
				for _, cat := range catNames {
					funcs := categories[cat]

					// Sort functions within category
					sort.Slice(funcs, func(i, j int) bool {
						return funcs[i].Name < funcs[j].Name
					})

					// Convert to array of dictionaries
					funcObjs := make([]Object, len(funcs))
					for i, f := range funcs {
						paramObjs := make([]Object, len(f.Params))
						for j, p := range f.Params {
							paramObjs[j] = &String{Value: p}
						}

						pairs := map[string]ast.Expression{
							"name":        createLiteralExpression(&String{Value: f.Name}),
							"arity":       createLiteralExpression(&String{Value: f.Arity}),
							"description": createLiteralExpression(&String{Value: f.Description}),
							"params":      createLiteralExpression(&Array{Elements: paramObjs}),
						}
						if f.Deprecated != "" {
							pairs["deprecated"] = createLiteralExpression(&String{Value: f.Deprecated})
						}
						funcObjs[i] = &Dictionary{Pairs: pairs, Env: NewEnvironment()}
					}

					resultPairs[cat] = createLiteralExpression(&Array{Elements: funcObjs})
				}

				return &Dictionary{Pairs: resultPairs, Env: NewEnvironment()}
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

// isCommandHandle moved to eval_helpers.go

// executeCommand executes a command handle with input and returns result dictionary
//
// SECURITY CRITICAL FUNCTION
// ===========================
//
// This function executes external commands with user-provided input. Security considerations:
//
// 1. SECURITY POLICY ENFORCEMENT:
//   - env.Security MUST be set for untrusted input (production mode)
//   - nil security policy = unrestricted access (development only)
//   - Security check happens AFTER path resolution to catch binary location
//
// 2. ARGUMENT HANDLING:
//   - Arguments are passed directly to exec.Command (NO shell interpretation)
//   - Shell metacharacters in args are treated as literals (safe)
//   - Example: arg "file; rm -rf /" is passed as single argument, NOT executed as shell
//
// 3. BINARY PATH RESOLUTION:
//   - Absolute/relative paths: used as-is (subject to security check)
//   - Simple names: resolved via PATH lookup using exec.LookPath
//   - PATH lookup can be exploited if:
//     a) Binary name is user-controlled (attacker can reference any binary in PATH)
//     b) PATH environment is manipulated (security policy should prevent this)
//
// 4. TIMEOUT HANDLING:
//   - Optional timeout prevents indefinite hangs
//   - Requires proper context propagation
//   - Timeout kills process tree (SIGKILL on Unix)
//
// 5. ENVIRONMENT VARIABLES:
//   - Custom env vars can be set via options.env
//   - Empty env = inherit from parent process
//   - Security risk: modified PATH, LD_PRELOAD, etc.
//
// AI MAINTENANCE GUIDE:
// ---------------------
// - Never construct binary name or args from unsanitized user input
// - Always document new command features in docs/parsley/security.md
// - Test with malicious inputs: "../../../usr/bin/evil", "arg; injection"
// - Consider: should execute() exist at all in sandboxed/production mode?
// - Review security policy before adding new command capabilities
//
// ATTACK SURFACE ANALYSIS:
// -------------------------
//
//  1. Binary path traversal:
//     execute(cmd("../../../usr/bin/evil"))
//     Mitigation: Security policy checks resolved path
//
//  2. Argument injection attempts (SAFE due to exec.Command):
//     execute(cmd("ls", "-la; rm -rf /"))
//     Result: ls receives literal argument "-la; rm -rf /", semicolon NOT interpreted
//
//  3. Environment manipulation:
//     execute(cmd("gcc"), {env: {("LD_PRELOAD"): "/tmp/evil.so"}})
//     Mitigation: Security policy should block untrusted commands entirely
//
//  4. PATH manipulation (if allowed to set custom env):
//     execute(cmd("python"), {env: {("PATH"): "/tmp/evil/path"}})
//     Mitigation: Resolve path BEFORE applying custom env
//
//  5. Working directory escape:
//     execute(cmd("cat", "flag.txt"), {dir: path("../../../etc")})
//     Mitigation: Security policy checks dir path
//
// RECOMMENDED HARDENING (future):
// --------------------------------
// - Add allowlist of permitted binaries (e.g., only git, make, node)
// - Block dangerous env vars (LD_PRELOAD, DYLD_*, etc.)
// - Add per-command argument validators
// - Consider requiring explicit permission per binary in security policy
func executeCommand(cmdDict *Dictionary, input Object, env *Environment) Object {
	// Extract binary
	binaryExpr, ok := cmdDict.Pairs["binary"]
	if !ok {
		return newCommandError("CMD-0001", map[string]any{"Field": "binary"})
	}
	binaryLit, ok := binaryExpr.(*ast.StringLiteral)
	if !ok {
		return newCommandError("CMD-0002", map[string]any{"Field": "binary", "Expected": "a string", "Actual": "non-string expression"})
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
		return newCommandError("CMD-0001", map[string]any{"Field": "args"})
	}
	argsLit, ok := argsExpr.(*ast.ArrayLiteral)
	if !ok {
		return newCommandError("CMD-0002", map[string]any{"Field": "args", "Expected": "an array", "Actual": "non-array expression"})
	}

	args := make([]string, len(argsLit.Elements))
	for i, argExpr := range argsLit.Elements {
		argLit, ok := argExpr.(*ast.StringLiteral)
		if !ok {
			return newCommandError("CMD-0003", nil)
		}
		args[i] = argLit.Value
	}

	// Extract options
	optsExpr, ok := cmdDict.Pairs["options"]
	if !ok {
		return newCommandError("CMD-0001", map[string]any{"Field": "options"})
	}
	optsLit, ok := optsExpr.(*ast.DictionaryLiteral)
	if !ok {
		return newCommandError("CMD-0002", map[string]any{"Field": "options", "Expected": "a dictionary", "Actual": "non-dictionary expression"})
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
			return newCommandError("CMD-0004", map[string]any{"Type": string(input.Type())})
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
		// Special case: standalone import statements (not in let/assignment)
		// These auto-bind and return NULL to avoid appearing in program output
		if importExpr, ok := stmt.Expression.(*ast.ImportExpression); ok {
			module := Eval(importExpr, env)
			if isError(module) {
				return module
			}
			// Auto-bind ONLY for standalone imports (not in let/assignment)
			// The bind name is derived from the path (e.g., "markdown" from "@std/markdown")
			// or from an explicit alias (e.g., "MD" from "import @std/markdown as MD")
			if importExpr.BindName != "" {
				env.Set(importExpr.BindName, module)
			}
			// Return NULL so imports don't appear in concatenated output
			return NULL
		}
		return Eval(stmt.Expression, env)
	case *ast.ReturnStatement:
		val := Eval(stmt.ReturnValue, env)
		if isError(val) {
			return val
		}
		// Propagate stop/skip signals, but unwrap CheckExit (its value becomes the return value)
		if val != nil {
			rt := val.Type()
			if rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ {
				return val
			}
			// Unwrap CheckExit - check exits the expression, its value is returned
			if checkExit, ok := val.(*CheckExit); ok {
				val = checkExit.Value
			}
		}
		return &ReturnValue{Value: val}
	case *ast.CheckStatement:
		return evalCheckStatement(stmt, env)
	case *ast.StopStatement:
		return &StopSignal{}
	case *ast.SkipStatement:
		return &SkipSignal{}
	default:
		return Eval(stmt, env)
	}
}

// Method dispatch operations (evalDBConnectionMethod, dispatchMethodCall) are in eval_method_dispatch.go

// Eval evaluates AST nodes and returns objects
func Eval(node ast.Node, env *Environment) Object {
	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(node.Statements, env)

	case *ast.ExpressionStatement:
		// Special case: standalone import statements (not in let/assignment)
		// These auto-bind and return NULL to avoid appearing in program output
		if importExpr, ok := node.Expression.(*ast.ImportExpression); ok {
			module := Eval(importExpr, env)
			if isError(module) {
				return module
			}
			// Auto-bind ONLY for standalone imports (not in let/assignment)
			// The bind name is derived from the path (e.g., "markdown" from "@std/markdown")
			// or from an explicit alias (e.g., "MD" from "import @std/markdown as MD")
			if importExpr.BindName != "" {
				env.Set(importExpr.BindName, module)
			}
			// Return NULL so imports don't appear in concatenated output
			return NULL
		}
		return Eval(node.Expression, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		// Propagate return/stop/skip signals (but NOT CheckExit - that gets unwrapped)
		if val != nil {
			rt := val.Type()
			if rt == RETURN_OBJ || rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ {
				return val
			}
			// Unwrap CheckExit to its value - check exits the block, value is stored
			if checkExit, ok := val.(*CheckExit); ok {
				val = checkExit.Value
			}
		}

		// End any active table chain when storing
		val = endTableChain(val)

		// Handle dictionary destructuring
		if node.DictPattern != nil {
			return evalDictDestructuringAssignment(node.DictPattern, val, env, true, node.Export)
		}

		// Handle array destructuring assignment
		if node.ArrayPattern != nil {
			return evalArrayPatternAssignment(node.ArrayPattern, val, env, true, node.Export)
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
		// Propagate return/stop/skip signals (but NOT CheckExit - that gets unwrapped)
		if val != nil {
			rt := val.Type()
			if rt == RETURN_OBJ || rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ {
				return val
			}
			// Unwrap CheckExit to its value - check exits the block, value is stored
			if checkExit, ok := val.(*CheckExit); ok {
				val = checkExit.Value
			}
		}

		// End any active table chain when storing
		val = endTableChain(val)

		// Handle dictionary destructuring
		if node.DictPattern != nil {
			return evalDictDestructuringAssignment(node.DictPattern, val, env, false, node.Export)
		}

		// Handle array destructuring assignment
		if node.ArrayPattern != nil {
			return evalArrayPatternAssignment(node.ArrayPattern, val, env, false, node.Export)
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

	case *ast.ExportNameStatement:
		// Export an already-defined binding: 'export Name'
		val, ok := env.Get(node.Name.Value)
		if !ok {
			return &Error{Message: fmt.Sprintf("undefined identifier for export: %s", node.Name.Value)}
		}
		// Mark as exported (value already in environment)
		env.SetExport(node.Name.Value, val)
		return NULL

	case *ast.ComputedExportStatement:
		// Create a DynamicAccessor that evaluates the body on each access
		// Capture the module environment for the body evaluation
		moduleEnv := env
		accessor := &DynamicAccessor{
			Name: node.Name.Value,
			Resolver: func(accessEnv *Environment) Object {
				// Create a new environment for evaluation that:
				// 1. Uses the module's scope for variable lookups
				// 2. Inherits BasilCtx from the access environment (for @basil/http etc.)
				evalEnv := NewEnclosedEnvironment(moduleEnv)
				if accessEnv != nil {
					evalEnv.BasilCtx = accessEnv.BasilCtx
					evalEnv.DevLog = accessEnv.DevLog
					evalEnv.ServerDB = accessEnv.ServerDB
				}
				return Eval(node.Body, evalEnv)
			},
		}
		env.SetExport(node.Name.Value, accessor)
		return NULL

	case *ast.IndexAssignmentStatement:
		return evalIndexAssignment(node, env)

	case *ast.ReadStatement:
		return evalReadStatement(node, env)

	case *ast.ReadExpression:
		return evalReadExpression(node, env)

	case *ast.FetchStatement:
		return evalFetchStatement(node, env)

	case *ast.WriteStatement:
		return evalWriteStatement(node, env)

	case *ast.RemoteWriteStatement:
		return evalRemoteWriteStatement(node, env)

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
		// Propagate stop/skip signals, but unwrap CheckExit (its value becomes the return value)
		if val != nil {
			rt := val.Type()
			if rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ {
				return val
			}
			// Unwrap CheckExit - check exits the expression, its value is returned
			if checkExit, ok := val.(*CheckExit); ok {
				val = checkExit.Value
			}
		}
		return &ReturnValue{Value: val}

	case *ast.CheckStatement:
		return evalCheckStatement(node, env)

	case *ast.StopStatement:
		return &StopSignal{}

	case *ast.SkipStatement:
		return &SkipSignal{}

	// Expressions
	case *ast.IntegerLiteral:
		return &Integer{Value: node.Value}

	case *ast.FloatLiteral:
		return &Float{Value: node.Value}

	case *ast.StringLiteral:
		return &String{Value: node.Value}

	case *ast.TemplateLiteral:
		return evalTemplateLiteral(node, env)

	case *ast.RawTemplateLiteral:
		return evalRawTemplateLiteral(node, env)

	case *ast.RegexLiteral:
		return evalRegexLiteral(node, env)

	case *ast.DatetimeNowLiteral:
		return evalDatetimeNowLiteral(node, env)

	case *ast.DatetimeLiteral:
		return evalDatetimeLiteral(node, env)

	case *ast.DurationLiteral:
		return evalDurationLiteral(node, env)

	case *ast.ConnectionLiteral:
		return evalConnectionLiteral(node, env)

	case *ast.SchemaDeclaration:
		return evalSchemaDeclaration(node, env)

	case *ast.TableLiteral:
		return evalTableLiteral(node, env)

	case *ast.QueryExpression:
		return evalQueryExpression(node, env)

	case *ast.InsertExpression:
		return evalInsertExpression(node, env)

	case *ast.UpdateExpression:
		return evalUpdateExpression(node, env)

	case *ast.DeleteExpression:
		return evalDeleteExpression(node, env)

	case *ast.TransactionExpression:
		return evalTransactionExpression(node, env)

	case *ast.MoneyLiteral:
		return &Money{
			Amount:   node.Amount,
			Currency: node.Currency,
			Scale:    node.Scale,
		}

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

	case *ast.GroupedExpression:
		// Simply evaluate the inner expression
		return Eval(node.Inner, env)

	case *ast.IsExpression:
		return evalIsExpression(node, env)

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
			perr := perrors.New("CMD-0005", map[string]any{"Got": string(cmdObj.Type())})
			return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: node.Token.Line, Column: node.Token.Column}
		}

		if !isCommandHandle(cmdDict) {
			perr := perrors.New("CMD-0006", nil)
			return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: node.Token.Line, Column: node.Token.Column}
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
			result := evalImport(args, env)
			if isError(result) {
				return withPosition(result, node.Token, env)
			}
			return result
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

			method := dotExpr.Key

			// Null propagation: method calls on null return null
			// EXCEPT for .type() which works on all objects including null
			if (left == NULL || left == nil) && method != "type" {
				return NULL
			}

			// Evaluate arguments
			args := evalExpressions(node.Arguments, env)
			if len(args) == 1 && isError(args[0]) {
				return args[0]
			}

			// Dispatch based on receiver type and enrich errors with position
			result := dispatchMethodCall(left, method, args, env)
			if result != nil {
				return withPosition(result, dotExpr.Token, env)
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
				return withPosition(newCallError("CALL-0004", map[string]any{"Name": ident.Value}), node.Token, env)
			}
			return withPosition(newCallError("CALL-0005", map[string]any{"Context": funcName}), node.Token, env)
		}

		// Save the call token before evaluating arguments (which may modify env.LastToken)
		callToken := node.Token

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		result := ApplyFunctionWithEnv(function, args, env)
		// Enrich errors from function application with call site position
		return withPosition(result, callToken, env)

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

	case *ast.TryExpression:
		return evalTryExpression(node, env)

	case *ast.ImportExpression:
		return evalImportExpression(node, env)
	}

	perr := perrors.New("INTERNAL-0002", map[string]any{"Type": fmt.Sprintf("%T", node)})
	return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
}

// Helper functions
func evalProgram(stmts []ast.Statement, env *Environment) Object {
	var results []Object

	for _, statement := range stmts {
		result := Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_OBJ {
				// Immediate return with value
				return result.(*ReturnValue).Value
			}
			if rt == ERROR_OBJ {
				return result
			}

			// Stop/skip signals at program level are errors
			if rt == STOP_SIGNAL_OBJ {
				return &Error{
					Class:   ClassType,
					Code:    "LOOP-0008",
					Message: "'stop' can only be used inside a for loop",
				}
			}
			if rt == SKIP_SIGNAL_OBJ {
				return &Error{
					Class:   ClassType,
					Code:    "LOOP-0009",
					Message: "'skip' can only be used inside a for loop",
				}
			}

			// CheckExit at program level - return the value
			if rt == CHECK_EXIT_OBJ {
				return result.(*CheckExit).Value
			}

			// Handle PrintValue - expand into results as strings
			if rt == PRINT_VALUE_OBJ {
				pv := result.(*PrintValue)
				for _, v := range pv.Values {
					str := objectToUserString(v)
					if str != "" { // Skip empty (null produces "")
						results = append(results, &String{Value: str})
					}
				}
				continue
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

func evalBlockStatement(block *ast.BlockStatement, env *Environment) Object {
	var results []Object

	for _, statement := range block.Statements {
		result := Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == RETURN_OBJ || rt == ERROR_OBJ {
				return result
			}

			// Bubble up control flow signals (stop, skip, check exit)
			if rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ || rt == CHECK_EXIT_OBJ {
				return result
			}

			// Handle PrintValue - expand into results as strings
			if rt == PRINT_VALUE_OBJ {
				pv := result.(*PrintValue)
				for _, v := range pv.Values {
					str := objectToUserString(v)
					if str != "" { // Skip empty (null produces "")
						results = append(results, &String{Value: str})
					}
				}
				continue
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

			// Bubble up control flow signals (stop, skip, check exit)
			if rt == STOP_SIGNAL_OBJ || rt == SKIP_SIGNAL_OBJ || rt == CHECK_EXIT_OBJ {
				return result
			}

			// Handle PrintValue - expand into results as strings
			if rt == PRINT_VALUE_OBJ {
				pv := result.(*PrintValue)
				for _, v := range pv.Values {
					str := objectToUserString(v)
					if str != "" { // Skip empty (null produces "")
						results = append(results, &String{Value: str})
					}
				}
				continue
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

// nativeBoolToParsBoolean moved to eval_helpers.go
// Prefix operator evaluators moved to eval_operators.go:
// - evalPrefixExpression
// - evalBangOperatorExpression
// - evalMinusPrefixOperatorExpression

// Infix expression evaluators moved to eval_infix.go:
// - evalInfixExpression (main dispatcher)
// - evalIntegerInfixExpression
// - evalFloatInfixExpression
// - evalMixedInfixExpression
// - evalStringInfixExpression
// - evalDatetimeInfixExpression
// - evalDatetimeIntegerInfixExpression
// - evalIntegerDatetimeInfixExpression
// - evalDurationInfixExpression
// - evalDurationIntegerInfixExpression
// - evalDatetimeDurationInfixExpression
// - evalPathInfixExpression
// - evalPathStringInfixExpression
// - evalUrlInfixExpression
// - evalUrlStringInfixExpression

// Function application and assignment operations (applyFunction, extendFunctionEnv,
// evalArrayPatternAssignment, evalDestructuringAssignment, etc.) are in eval_expressions.go

// evalTemplateLiteral evaluates a template literal with interpolation
func evalTemplateLiteral(node *ast.TemplateLiteral, env *Environment) Object {
	template := node.Value
	var result strings.Builder

	// Base position for error reporting - template content starts after the opening backtick
	baseLine := node.Token.Line
	baseCol := node.Token.Column + 1

	i := 0
	for i < len(template) {
		// Look for {
		if template[i] == '{' {
			// Record position for error reporting
			exprOffset := i + 1

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

			// Parse and evaluate the expression (with filename for error reporting)
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				adjustedCol := baseCol + exprOffset + (perr.Column - 1)
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    baseLine,
					Column:  adjustedCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors too
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = baseLine
						errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
						if errObj.Column < baseCol+exprOffset {
							errObj.Column = baseCol + exprOffset
						}
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
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

// evalRawTemplateLiteral evaluates a raw template literal (single-quoted string with @{} interpolation)
func evalRawTemplateLiteral(node *ast.RawTemplateLiteral, env *Environment) Object {
	template := node.Value

	if env == nil {
		env = NewEnvironment()
	}

	// Base position for error reporting - content starts after the opening quote
	baseLine := node.Token.Line
	baseCol := node.Token.Column + 1

	var result strings.Builder
	i := 0
	for i < len(template) {
		// Handle escaped @
		if template[i] == '\\' && i+1 < len(template) && template[i+1] == '@' {
			result.WriteByte('@')
			i += 2
			continue
		}

		// Look for @{
		if i < len(template)-1 && template[i] == '@' && template[i+1] == '{' {
			// Record position for error reporting - expression starts after @{
			exprOffset := i + 2

			i += 2 // skip @{
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
				return newParseError("PARSE-0009", "raw template", nil)
			}

			exprStr := template[exprStart:i]
			i++ // skip closing }

			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				adjustedCol := baseCol + exprOffset + (perr.Column - 1)
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    baseLine,
					Column:  adjustedCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors too
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = baseLine
						errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
						if errObj.Column < baseCol+exprOffset {
							errObj.Column = baseCol + exprOffset
						}
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
					return evaluated
				}
			}

			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
			continue
		}

		result.WriteByte(template[i])
		i++
	}

	return &String{Value: result.String()}
}

// interpolateRawString evaluates a string containing @{...} interpolations.
// Similar to evalTemplateLiteral but uses @{} delimiters and supports \@ for literal @.
func interpolateRawString(template string, env *Environment) Object {
	if env == nil {
		env = NewEnvironment()
	}

	var result strings.Builder
	i := 0
	for i < len(template) {
		// Handle escaped @
		if template[i] == '\\' && i+1 < len(template) && template[i+1] == '@' {
			result.WriteByte('@')
			i += 2
			continue
		}

		// Look for @{
		if i < len(template)-1 && template[i] == '@' && template[i+1] == '{' {
			i += 2 // skip @{
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
				return newParseError("PARSE-0009", "raw template", nil)
			}

			exprStr := template[exprStart:i]
			i++ // skip closing }

			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with file info preserved
				perr := errs[0]
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    perr.Line,
					Column:  perr.Column,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					return evaluated
				}
			}

			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
			continue
		}

		result.WriteByte(template[i])
		i++
	}

	return &String{Value: result.String()}
}

// evalTagLiteral evaluates a singleton tag

// Tag evaluation functions (evalTagLiteral, evalTagPair, evalCacheTag,
// evalPartTag, evalStandardTagPair, evalCustomTagPair, evalTagContents,
// evalTagContentsAsArray, evalSQLTag, evalTagProps, evalStandardTag,
// evalCustomTag) are in eval_tags.go
// Operator evaluation functions moved to eval_operators.go:
// - evalConcatExpression
// - evalInExpression
// - evalIndexExpression
// - evalArrayIndexExpression
// - evalStringIndexExpression
// - evalSliceExpression
// - evalArraySliceExpression
// - evalStringSliceExpression

// evalTryExpression evaluates a try expression.
// It catches "user errors" (IO, Network, Database, Format, Value, Security) and
// returns them in a {result, error} dictionary instead of halting execution.
// "Developer errors" (Type, Arity, Undefined, etc.) are propagated unchanged.

// Try expression evaluation (evalTryExpression) is in eval_control_flow.go

// evalDictionaryLiteral evaluates dictionary literals
func evalDictionaryLiteral(node *ast.DictionaryLiteral, env *Environment) Object {
	// Evaluate all values eagerly and store them as ObjectLiteralExpressions
	// This ensures values like method calls (t.count()) are evaluated at creation time
	pairs := make(map[string]ast.Expression)
	keyOrder := []string{}

	// Use KeyOrder from AST if available, otherwise iterate map (for backward compat)
	keys := node.KeyOrder
	if len(keys) == 0 {
		for key := range node.Pairs {
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		expr := node.Pairs[key]
		value := Eval(expr, env)
		if isError(value) {
			return value
		}
		// Convert the evaluated value back to an expression for storage
		pairs[key] = objectToExpression(value)
		keyOrder = append(keyOrder, key)
	}

	// Evaluate computed key-value pairs
	for _, cp := range node.ComputedPairs {
		// Evaluate the key expression
		keyObj := Eval(cp.Key, env)
		if isError(keyObj) {
			return keyObj
		}

		// Convert key to string
		var keyStr string
		switch k := keyObj.(type) {
		case *String:
			keyStr = k.Value
		case *Integer:
			keyStr = fmt.Sprintf("%d", k.Value)
		case *Float:
			keyStr = fmt.Sprintf("%g", k.Value)
		case *Boolean:
			keyStr = fmt.Sprintf("%t", k.Value)
		default:
			return &Error{Message: fmt.Sprintf("computed dictionary key must be a string, integer, float, or boolean, got %s", keyObj.Type())}
		}

		// Evaluate the value expression
		value := Eval(cp.Value, env)
		if isError(value) {
			return value
		}

		pairs[keyStr] = objectToExpression(value)
		keyOrder = append(keyOrder, keyStr)
	}

	dict := &Dictionary{
		Pairs:    pairs,
		KeyOrder: keyOrder,
		Env:      env,
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
		return newUndefinedError("UNDEF-0004", map[string]any{"Property": node.Key, "Type": "SFTP file handle"})
	}

	// Handle Money property access
	if money, ok := left.(*Money); ok {
		return evalMoneyProperty(money, node.Key)
	}

	// Handle StdlibModuleDict property access (e.g., math.PI)
	if stdlibMod, ok := left.(*StdlibModuleDict); ok {
		if val, exists := stdlibMod.Exports[node.Key]; exists {
			// Resolve DynamicAccessor to current value
			if accessor, ok := val.(*DynamicAccessor); ok {
				return accessor.Resolve(env)
			}
			return val
		}
		return newUndefinedError("UNDEF-0004", map[string]any{"Property": node.Key, "Type": "stdlib module"})
	}

	// Handle DSLSchema property access (e.g., User.Name, User.Fields)
	if schema, ok := left.(*DSLSchema); ok {
		return evalDSLSchemaProperty(schema, node.Key)
	}

	// Handle Record property access (direct data field access)
	if record, ok := left.(*Record); ok {
		return evalRecordProperty(record, node.Key, env)
	}

	// Handle Dictionary (including special types like datetime, path, url)
	dict, ok := left.(*Dictionary)
	if !ok {
		return newStructuredErrorWithPosAndFile("TYPE-0022", node.Token, env, map[string]any{"Got": left.Type()})
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
	if isDurationDict(dict) {
		if computed := evalDurationComputedProperty(dict, node.Key, env); computed != nil {
			return computed
		}
	}

	// Get the expression from the dictionary
	expr, ok := dict.Pairs[node.Key]
	if !ok {
		// Check if it's a dictionary method name - provide helpful error
		for _, m := range dictionaryMethods {
			if m == node.Key {
				return methodAsPropertyError(node.Key, "Dictionary")
			}
		}
		return NULL
	}

	// Create a new environment with 'this' bound to the dictionary
	dictEnv := NewEnclosedEnvironment(dict.Env)
	dictEnv.Set("this", dict)

	// Evaluate the expression in the dictionary's environment
	val := Eval(expr, dictEnv)

	// Resolve DynamicAccessor to current value (for computed exports)
	if accessor, ok := val.(*DynamicAccessor); ok {
		return accessor.Resolve(env)
	}

	return val
}

// evalReadStatement evaluates the <== operator to read file content

// File I/O read operations (evalReadStatement, evalReadExpression) are in eval_file_io.go
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
					return nil, newStdioError("STDIO-0003", map[string]any{"GoError": readErr.Error()})
				}
				pathStr = "-"
			case "stdout", "stderr":
				return nil, newStdioError("STDIO-0004", map[string]any{"Stream": stdioStr.Value})
			default:
				return nil, newStdioError("STDIO-0002", map[string]any{"Name": stdioStr.Value})
			}
		}
	} else {
		// Get the path from the file dictionary
		pathStr = getFilePathString(fileDict, env)
		if pathStr == "" {
			return nil, newFileOpError("FILEOP-0002", nil)
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
		return nil, newFileOpError("FILEOP-0003", nil)
	}
	formatObj := Eval(formatExpr, env)
	if isError(formatObj) {
		return nil, formatObj.(*Error)
	}
	formatStr, ok := formatObj.(*String)
	if !ok {
		return nil, newFileOpError("FILEOP-0004", map[string]any{"Got": string(formatObj.Type())})
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

	case "pln":
		// Parse PLN (Parsley Literal Notation)
		content := string(data)
		return parsePLN(content, env)

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

		// Extract options from fileDict if present
		var options *Dictionary
		if optionsExpr, ok := fileDict.Pairs["options"]; ok {
			optionsObj := Eval(optionsExpr, env)
			if optDict, ok := optionsObj.(*Dictionary); ok {
				options = optDict
			}
		}

		return parseMarkdown(content, options, env)

	default:
		return nil, newFileOpError("FILEOP-0005", map[string]any{"Operation": "reading", "Format": formatStr.Value})
	}
}

// parseJSON parses a JSON string into Parsley objects

// Goldmark Parsley Interpolation Extension moved to eval_parsing.go (KindParsleyInterpolation, ParsleyInterpolationNode, parsleyInterpolationParser, parsleyInterpolationRenderer, ParsleyInterpolationExtension, NewParsleyInterpolation, findMatchingBraceInBytes)

// Markdown/YAML/JSON/CSV parsing moved to eval_parsing.go:
// - parseMarkdown
// - yamlToObject
// - jsonToObject
// - parseCSV
// - parseCSVValue

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

// evalIndexAssignment evaluates index/property assignment statements like dict["key"] = value or obj.prop = value
func evalIndexAssignment(node *ast.IndexAssignmentStatement, env *Environment) Object {
	// Evaluate the value to assign
	value := Eval(node.Value, env)
	if isError(value) {
		return value
	}

	// Handle IndexExpression assignment: dict["key"] = value or arr[0] = value
	if indexExpr, ok := node.Target.(*ast.IndexExpression); ok {
		// Evaluate the object being indexed
		left := Eval(indexExpr.Left, env)
		if isError(left) {
			return left
		}

		// Evaluate the index
		index := Eval(indexExpr.Index, env)
		if isError(index) {
			return index
		}

		switch obj := left.(type) {
		case *Dictionary:
			// Dictionary assignment: dict["key"] = value
			key, ok := index.(*String)
			if !ok {
				return &Error{
					Message: fmt.Sprintf("dictionary key must be a string, got %s", index.Type()),
					Line:    indexExpr.Token.Line,
					Column:  indexExpr.Token.Column,
				}
			}
			// Convert Object to ast.Expression for storage
			obj.Pairs[key.Value] = objectToExpression(value)
			// Add to order if new key
			found := false
			for _, k := range obj.KeyOrder {
				if k == key.Value {
					found = true
					break
				}
			}
			if !found {
				obj.KeyOrder = append(obj.KeyOrder, key.Value)
			}
			return NULL

		case *Array:
			// Array assignment: arr[0] = value
			idx, ok := index.(*Integer)
			if !ok {
				return &Error{
					Message: fmt.Sprintf("array index must be an integer, got %s", index.Type()),
					Line:    indexExpr.Token.Line,
					Column:  indexExpr.Token.Column,
				}
			}
			i := int(idx.Value)
			// Handle negative indices
			if i < 0 {
				i = len(obj.Elements) + i
			}
			if i < 0 || i >= len(obj.Elements) {
				return &Error{
					Message: fmt.Sprintf("array index out of bounds: %d (length %d)", idx.Value, len(obj.Elements)),
					Line:    indexExpr.Token.Line,
					Column:  indexExpr.Token.Column,
				}
			}
			obj.Elements[i] = value
			return NULL

		default:
			return &Error{
				Message: fmt.Sprintf("cannot assign to index of %s", left.Type()),
				Line:    indexExpr.Token.Line,
				Column:  indexExpr.Token.Column,
			}
		}
	}

	// Handle DotExpression assignment: obj.prop = value
	if dotExpr, ok := node.Target.(*ast.DotExpression); ok {
		// Evaluate the object
		left := Eval(dotExpr.Left, env)
		if isError(left) {
			return left
		}

		switch obj := left.(type) {
		case *Dictionary:
			// Dictionary property assignment: dict.key = value
			// Convert Object to ast.Expression for storage
			obj.Pairs[dotExpr.Key] = objectToExpression(value)
			// Add to order if new key
			found := false
			for _, k := range obj.KeyOrder {
				if k == dotExpr.Key {
					found = true
					break
				}
			}
			if !found {
				obj.KeyOrder = append(obj.KeyOrder, dotExpr.Key)
			}
			return NULL

		default:
			return &Error{
				Message: fmt.Sprintf("cannot assign to property of %s", left.Type()),
				Line:    dotExpr.Token.Line,
				Column:  dotExpr.Token.Column,
			}
		}
	}

	return &Error{
		Message: "invalid assignment target",
		Line:    node.Token.Line,
		Column:  node.Token.Column,
	}
}

// evalWriteStatement evaluates the ==> and ==>> operators to write file content

// File I/O write operations (evalWriteStatement, writeFileContent,
// evalFileRemove) are in eval_file_io.go

// evalDictionaryIndexExpression moved to eval_operators.go

// environmentToDict, ExportsToDict, and objectToExpression are in eval_conversions.go

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

// Collection operations (evalArrayIntersection, evalDictionaryIntersection,
// evalArrayUnion, evalArraySubtraction, evalDictionarySubtraction,
// evalArrayChunking, evalStringRepetition, evalArrayRepetition,
// evalRangeExpression) are in eval_collections.go

// ============================================================================
// Helper functions for method implementations (used by methods.go)
// ============================================================================

// Locale formatting helpers moved to eval_locale.go

// ============================================================================
// Money arithmetic
// ============================================================================

// Money infix operations (evalMoneyInfixExpression, evalMoneyScalarExpression,
// evalScalarMoneyExpression, promoteMoneyScale) are in eval_infix.go

// bankersRound implements banker's rounding (round half to even)
func bankersRound(x float64) int64 {
	// Get the integer and fractional parts
	whole := math.Floor(x)
	frac := x - whole

	if frac < 0.5 {
		return int64(whole)
	} else if frac > 0.5 {
		return int64(whole) + 1
	} else {
		// Exactly 0.5 - round to even
		wholeInt := int64(whole)
		if wholeInt%2 == 0 {
			return wholeInt
		}
		return wholeInt + 1
	}
}

// matchPathPattern matches a URL path against a pattern with :param and *glob segments
// Returns map of captured values on match, nil on no match
// :name captures a single segment, *name captures remaining segments as []string
func matchPathPattern(path, pattern string) map[string]interface{} {
	// Normalize: trim trailing slashes for comparison
	path = strings.TrimSuffix(path, "/")
	pattern = strings.TrimSuffix(pattern, "/")

	// Handle empty paths
	if path == "" {
		path = "/"
	}
	if pattern == "" {
		pattern = "/"
	}

	// Split into segments
	pathSegs := strings.Split(path, "/")
	patternSegs := strings.Split(pattern, "/")

	// Remove empty first segment from leading /
	if len(pathSegs) > 0 && pathSegs[0] == "" {
		pathSegs = pathSegs[1:]
	}
	if len(patternSegs) > 0 && patternSegs[0] == "" {
		patternSegs = patternSegs[1:]
	}

	result := make(map[string]interface{})

	pi := 0 // pattern index
	for i := 0; i < len(pathSegs); i++ {
		if pi >= len(patternSegs) {
			// Path has extra segments with no pattern to match
			return nil
		}

		seg := patternSegs[pi]

		if strings.HasPrefix(seg, "*") {
			// Glob: capture rest of path as array
			name := seg[1:]
			if name == "" {
				name = "rest" // default name if just "*"
			}
			result[name] = pathSegs[i:]
			return result
		}

		if strings.HasPrefix(seg, ":") {
			// Parameter: capture single segment
			name := seg[1:]
			result[name] = pathSegs[i]
		} else if seg != pathSegs[i] {
			// Literal: must match exactly (case sensitive)
			return nil
		}

		pi++
	}

	// Check all pattern segments consumed (unless we had a glob)
	if pi < len(patternSegs) {
		// Remaining pattern segments - check if they're all optional (glob at end)
		remaining := patternSegs[pi:]
		if len(remaining) == 1 && strings.HasPrefix(remaining[0], "*") {
			// Single glob at end with no path segments - empty array
			name := remaining[0][1:]
			if name == "" {
				name = "rest"
			}
			result[name] = []string{}
			return result
		}
		return nil
	}

	return result
}
