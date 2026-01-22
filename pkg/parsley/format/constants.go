// Package format provides pretty-printing for Parsley code.
// All formatting thresholds are configurable via these constants.
package format

// Line width - the target maximum line length
// Using 92 leaves headroom before the common 80-char limit
const MaxLineWidth = 92

// Threshold percentages (of MaxLineWidth)
// These control when to switch from inline to multiline formatting
// Lower = more conservative (more multiline), Higher = more aggressive (more inline)
const (
	ThresholdSmallPercent  = 50 // Arrays, dicts, chains, function args (was 60)
	ThresholdIfElsePercent = 40 // Single-line if/else expressions (was 50)
)

// Computed thresholds - derived from MaxLineWidth and percentages
// Change MaxLineWidth to adjust all thresholds proportionally
var (
	ArrayThreshold    = MaxLineWidth * ThresholdSmallPercent / 100  // 46 chars
	DictThreshold     = MaxLineWidth * ThresholdSmallPercent / 100  // 46 chars
	ChainThreshold    = MaxLineWidth * ThresholdSmallPercent / 100  // 46 chars
	FuncArgsThreshold = MaxLineWidth * ThresholdSmallPercent / 100  // 46 chars
	IfElseThreshold   = MaxLineWidth * ThresholdIfElsePercent / 100 // 37 chars
)

// Query DSL thresholds
const (
	QueryMaxInlineClauses = 2  // Max clauses for inline query format
	QueryInlineThreshold  = 50 // Max chars for inline query (was 60)
)

// Indentation - gofmt style: tabs for indentation, spaces for alignment
const (
	TabWidth     = 4    // Display width of a tab character
	IndentWidth  = TabWidth // Each indent level is one tab = 4 display chars
	IndentString = "\t" // tab character
)

// Structure
const (
	BlankLinesBetweenDefs = 2 // Blank lines between top-level definitions
)

// Trailing commas - whether to add trailing comma on multiline structures
const TrailingCommaMultiline = true      // For arrays and objects
const TrailingCommaFuncCalls = false     // For function call arguments
const TrailingCommaFuncParams = false    // For function parameter lists
