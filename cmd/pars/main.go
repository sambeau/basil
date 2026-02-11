package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/format"
	"github.com/sambeau/basil/pkg/parsley/formatter"
	"github.com/sambeau/basil/pkg/parsley/help"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/sambeau/basil/pkg/parsley/repl"

	// Import pln for init() to register serialize/deserialize functions
	_ "github.com/sambeau/basil/pkg/parsley/pln"
)

// Version is set at compile time via -ldflags
var Version = "0.15.3"

var (
	// Display flags
	helpFlag        = flag.Bool("h", false, "Show help message")
	helpLongFlag    = flag.Bool("help", false, "Show help message")
	versionFlag     = flag.Bool("V", false, "Show version information")
	versionLongFlag = flag.Bool("version", false, "Show version information")
	prettyPrintFlag = flag.Bool("pp", false, "Pretty-print HTML output")
	prettyLongFlag  = flag.Bool("pretty", false, "Pretty-print HTML output")

	// Evaluation flags
	evalFlag     = flag.String("e", "", "Evaluate code string")
	evalLongFlag = flag.String("eval", "", "Evaluate code string")
	rawFlag      = flag.Bool("r", false, "Output raw print string instead of PLN")
	rawLongFlag  = flag.Bool("raw", false, "Output raw print string instead of PLN")
	checkFlag    = flag.Bool("check", false, "Check syntax without executing")

	// Security flags
	restrictReadFlag     = flag.String("restrict-read", "", "Comma-separated read blacklist paths")
	noReadFlag           = flag.Bool("no-read", false, "Deny all file reads")
	restrictWriteFlag    = flag.String("restrict-write", "", "Comma-separated write blacklist paths")
	noWriteFlag          = flag.Bool("no-write", false, "Deny all file writes")
	allowExecuteFlag     = flag.String("allow-execute", "", "Comma-separated execute whitelist paths")
	allowExecuteAllFlag  = flag.Bool("allow-execute-all", false, "Allow unrestricted executes")
	allowExecuteAllShort = flag.Bool("x", false, "Shorthand for --allow-execute-all")
)

func main() {
	// Check for subcommands first (before flag parsing)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "fmt":
			fmtCommand(os.Args[2:])
			return
		case "describe":
			describeCommand(os.Args[2:])
			return
		}
	}

	// Customize flag usage message
	flag.Usage = printHelp
	flag.Parse()

	// Check for help flag
	if *helpFlag || *helpLongFlag {
		printHelp()
		os.Exit(0)
	}

	// Check for version flag
	if *versionFlag || *versionLongFlag {
		fmt.Printf("pars version %s\n", Version)
		os.Exit(0)
	}

	// Determine pretty print setting
	prettyPrint := *prettyPrintFlag || *prettyLongFlag

	// Determine raw output setting
	raw := *rawFlag || *rawLongFlag

	// Get eval code (prefer -e over --eval if both set)
	evalCode := *evalFlag
	if evalCode == "" {
		evalCode = *evalLongFlag
	}

	// Mode dispatch
	switch {
	case evalCode != "":
		// Inline evaluation mode
		executeInline(evalCode, flag.Args(), prettyPrint, raw)
	case *checkFlag:
		// Syntax check mode
		files := flag.Args()
		if len(files) == 0 {
			fmt.Fprintln(os.Stderr, "Error: --check requires at least one file")
			os.Exit(2)
		}
		os.Exit(checkFiles(files))
	case len(flag.Args()) > 0:
		// File execution mode
		filename := flag.Args()[0]
		scriptArgs := flag.Args()[1:]
		executeFile(filename, scriptArgs, prettyPrint)
	default:
		// REPL mode
		repl.Start(os.Stdin, os.Stdout, Version)
	}
}

func printHelp() {
	fmt.Printf(`pars - Parsley language interpreter version %s

Usage:
  pars [options] [file] [args...]
  pars -e "code" [args...]
  pars --check <file>...
  pars fmt [options] <file>...
  pars describe <topic>

Commands:
  fmt                   Format Parsley source files
  describe <topic>      Show help for a type, builtin, module, or operator

Display Options:
  -h, --help            Show this help message
  -V, --version         Show version information
  -pp, --pretty         Pretty-print HTML output with proper indentation

Evaluation Options:
  -e, --eval <code>     Evaluate code string (outputs PLN representation)
  -r, --raw             Output raw print string instead of PLN (with -e)
  --check               Check syntax without executing (can specify multiple files)

Security Options:
  --restrict-read=PATHS     Deny reading from comma-separated paths
  --no-read                 Deny all file reads
  --restrict-write=PATHS    Deny writing to comma-separated paths
  --no-write                Deny all file writes
  --allow-execute=PATHS     Allow executing scripts from paths
  --allow-execute-all, -x   Allow unrestricted script execution

Security Examples:
  pars script.pars                              # Allow all reads/writes (default)
  pars --no-write script.pars                   # Deny all writes
  pars --restrict-write=/etc script.pars        # Deny writes to /etc only
  pars -x script.pars                           # Allow all reads/writes/executes
  pars --restrict-read=/etc script.pars         # Deny reads from /etc

Examples:
  pars                      Start interactive REPL
  pars script.pars          Execute a Parsley script
  pars -pp page.pars        Execute and pretty-print HTML output
  pars -e "1 + 2"           Evaluate inline code (outputs: 3)
  pars -e "[1, 2, 3]"       Evaluate array (outputs: [1, 2, 3])
  pars -e "[1,2,3]" --raw   Raw output for scripting (outputs: 123)
  pars -e '@args' foo bar   Evaluate code with arguments
  pars --check script.pars  Check syntax without executing
  pars --check *.pars       Check multiple files
  pars fmt script.pars      Format a Parsley file (print to stdout)
  pars fmt -w script.pars   Format a Parsley file in place
  pars describe string      Show help for string type
  pars describe builtins    List all builtin functions
  pars describe operators   List all operators
  pars describe @std/math   Show help for a module

For more information, visit: https://github.com/sambeau/parsley
`, Version)
}

// describeCommand implements the 'pars describe <topic>' subcommand
func describeCommand(args []string) {
	// Check for --json flag
	jsonOutput := false
	var topic string

	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else if !strings.HasPrefix(arg, "-") {
			topic = arg
		}
	}

	if topic == "" {
		fmt.Fprintln(os.Stderr, `Usage: pars describe [--json] <topic>

Topics:
  types              List all available types
  builtins           List all builtin functions by category
  operators          List all operators
  <type>             Help for a specific type (string, array, dictionary, ...)
  <builtin>          Help for a specific builtin (JSON, CSV, now, ...)
  @std/<module>      Help for a stdlib module (@std/math, @std/table, ...)
  @basil/<module>    Help for a basil module (@basil/http, @basil/auth)

Examples:
  pars describe string
  pars describe array
  pars describe @std/math
  pars describe builtins
  pars describe operators
  pars describe JSON
  pars describe --json string`)
		os.Exit(1)
	}

	result, err := help.DescribeTopic(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		data, err := help.FormatJSON(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		fmt.Print(help.FormatText(result, 80))
	}
}

// executeInline evaluates inline code provided via -e flag
func executeInline(code string, args []string, prettyPrint, raw bool) {
	policy, err := buildSecurityPolicy()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	l := lexer.NewWithFilename(code, "<eval>")
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.StructuredErrors(); len(errs) != 0 {
		printStructuredErrors("<eval>", code, errs)
		os.Exit(1)
	}

	env := evaluator.NewEnvironmentWithArgs(args)
	env.Filename = "<eval>"
	env.Security = policy
	evaluated := evaluator.Eval(program, env)

	// Handle nil evaluation result
	if evaluated == nil {
		if !raw {
			fmt.Println("null")
		}
		return
	}

	// Handle errors
	if evaluated.Type() == evaluator.ERROR_OBJ {
		errObj, ok := evaluated.(*evaluator.Error)
		if ok {
			printRuntimeError("<eval>", code, errObj)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", evaluated.Inspect())
		}
		os.Exit(1)
	}

	// Handle output based on mode
	if raw {
		// File-like behavior (current)
		if evaluated.Type() != evaluator.NULL_OBJ {
			output := evaluator.ObjectToPrintString(evaluated)
			if prettyPrint {
				output = formatter.FormatHTML(output)
			}
			fmt.Println(output)
		}
	} else {
		// REPL-like behavior (new default)
		if evaluated.Type() == evaluator.NULL_OBJ {
			fmt.Println("null")
		} else {
			fmt.Println(evaluator.ObjectToFormattedReprString(evaluated))
		}
	}
}

// checkFiles checks the syntax of one or more files without executing them
func checkFiles(files []string) int {
	hasErrors := false

	for _, filename := range files {
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
			return 2 // File error
		}

		l := lexer.NewWithFilename(string(content), filename)
		p := parser.New(l)
		_ = p.ParseProgram()

		if errs := p.StructuredErrors(); len(errs) != 0 {
			printStructuredErrors(filename, string(content), errs)
			hasErrors = true
		}
	}

	if hasErrors {
		return 1 // Syntax errors
	}
	return 0 // Success
}

// executeFile reads and executes a pars source file
func executeFile(filename string, scriptArgs []string, prettyPrint bool) {
	// Build security policy (always create one to enable default restrictions)
	policy, err := buildSecurityPolicy()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file '%s': %v\n", filename, err)
		os.Exit(1)
	}

	// Create lexer and parser with filename
	l := lexer.NewWithFilename(string(content), filename)
	p := parser.New(l)

	// Parse the program
	program := p.ParseProgram()
	if errs := p.StructuredErrors(); len(errs) != 0 {
		printStructuredErrors(filename, string(content), errs)
		os.Exit(1)
	}

	// Evaluate the program with @env and @args populated
	env := evaluator.NewEnvironmentWithArgs(scriptArgs)
	env.Filename = filename
	env.Security = policy
	evaluated := evaluator.Eval(program, env)

	// Check for evaluation errors
	if evaluated != nil && evaluated.Type() == evaluator.ERROR_OBJ {
		errObj, ok := evaluated.(*evaluator.Error)
		if ok {
			printRuntimeError(filename, string(content), errObj)
		} else {
			// Legacy error format (shouldn't happen anymore)
			fmt.Fprintf(os.Stderr, "%s: %s\n", filename, evaluated.Inspect())
		}
		os.Exit(1)
	}

	// Print result if not null and not an error
	if evaluated != nil && evaluated.Type() != evaluator.ERROR_OBJ && evaluated.Type() != evaluator.NULL_OBJ {
		output := evaluator.ObjectToPrintString(evaluated)

		// Apply HTML formatting if --pp flag is set
		if prettyPrint {
			output = formatter.FormatHTML(output)
		}

		fmt.Println(output)
	}
}

// printStructuredErrors prints parser errors with source context
func printStructuredErrors(filename string, source string, errs []*errors.ParsleyError) {
	lines := strings.Split(source, "\n")

	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err.PrettyString())
		printSourceContext(lines, err.Line, err.Column)
	}
}

// printRuntimeError prints a runtime error with source context
func printRuntimeError(filename string, source string, err *evaluator.Error) {
	// Use the file from the error if available (for errors in imported modules)
	displayFile := filename
	displaySource := source
	if err.File != "" && err.File != filename {
		displayFile = err.File
		// Try to load the actual source file for context
		if content, readErr := os.ReadFile(err.File); readErr == nil {
			displaySource = string(content)
		}
	}
	lines := strings.Split(displaySource, "\n")

	fmt.Fprint(os.Stderr, "Runtime error")
	if err.Line > 0 {
		fmt.Fprintf(os.Stderr, " in %s: line %d, column %d\n", displayFile, err.Line, err.Column)
	} else if displayFile != "" {
		fmt.Fprintf(os.Stderr, " in %s\n", displayFile)
	} else {
		fmt.Fprintln(os.Stderr)
	}
	fmt.Fprintf(os.Stderr, "  %s\n", err.Message)

	// Hints
	for _, hint := range err.Hints {
		fmt.Fprintf(os.Stderr, "  hint: %s\n", hint)
	}

	// Source context
	if err.Line > 0 {
		printSourceContext(lines, err.Line, err.Column)
	}
}

// printSourceContext prints the source line and error pointer
func printSourceContext(lines []string, lineNum, colNum int) {
	if lineNum <= 0 || lineNum > len(lines) {
		return
	}

	sourceLine := lines[lineNum-1]

	// Calculate how many columns to trim from the left
	trimCount := 0
	for i := 0; i < len(sourceLine); i++ {
		if sourceLine[i] == ' ' || sourceLine[i] == '\t' {
			if sourceLine[i] == '\t' {
				trimCount += 8
			} else {
				trimCount++
			}
		} else {
			break
		}
	}

	// Trim left whitespace from the source line
	trimmedLine := strings.TrimLeft(sourceLine, " \t")

	// Show the trimmed line with slight indentation
	fmt.Fprintf(os.Stderr, "    %s\n", trimmedLine)

	// Show pointer to the error position
	if colNum > 0 {
		// Calculate visual column accounting for tabs (8 spaces each) up to error position
		visualCol := 0
		for i := 0; i < colNum-1 && i < len(sourceLine); i++ {
			if sourceLine[i] == '\t' {
				visualCol += 8
			} else {
				visualCol++
			}
		}

		// Adjust pointer position by subtracting trimmed columns
		adjustedCol := max(visualCol-trimCount, 0)

		pointer := strings.Repeat(" ", adjustedCol) + "^"
		fmt.Fprintf(os.Stderr, "    %s\n", pointer)
	}
}

// buildSecurityPolicy creates a SecurityPolicy from command-line flags
func buildSecurityPolicy() (*evaluator.SecurityPolicy, error) {
	policy := &evaluator.SecurityPolicy{
		NoRead:          *noReadFlag,
		NoWrite:         *noWriteFlag,
		AllowWriteAll:   !*noWriteFlag, // Default to allowing writes unless --no-write is set
		AllowExecuteAll: *allowExecuteAllFlag || *allowExecuteAllShort,
	}

	// Parse restrict lists
	if *restrictReadFlag != "" {
		paths, err := parseAndResolvePaths(*restrictReadFlag)
		if err != nil {
			return nil, fmt.Errorf("invalid --restrict-read: %s", err)
		}
		policy.RestrictRead = paths
	}

	if *restrictWriteFlag != "" {
		paths, err := parseAndResolvePaths(*restrictWriteFlag)
		if err != nil {
			return nil, fmt.Errorf("invalid --restrict-write: %s", err)
		}
		policy.RestrictWrite = paths
	}

	if *allowExecuteFlag != "" {
		paths, err := parseAndResolvePaths(*allowExecuteFlag)
		if err != nil {
			return nil, fmt.Errorf("invalid --allow-execute: %s", err)
		}
		policy.AllowExecute = paths
	}

	return policy, nil
}

// parseAndResolvePaths parses comma-separated paths and resolves them to absolute paths
func parseAndResolvePaths(pathList string) ([]string, error) {
	parts := strings.Split(pathList, ",")
	resolved := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Expand home directory
		if strings.HasPrefix(p, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("cannot expand ~: %s", err)
			}
			p = filepath.Join(home, p[2:])
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("invalid path %s: %s", p, err)
		}

		// Clean path
		absPath = filepath.Clean(absPath)

		resolved = append(resolved, absPath)
	}

	return resolved, nil
}

// fmtCommand handles the 'pars fmt' subcommand
func fmtCommand(args []string) {
	fmtFlags := flag.NewFlagSet("fmt", flag.ExitOnError)
	writeFlag := fmtFlags.Bool("w", false, "Write result to source file instead of stdout")
	diffFlag := fmtFlags.Bool("d", false, "Display diffs instead of rewriting files (implies -w behavior check)")
	listFlag := fmtFlags.Bool("l", false, "List files whose formatting differs from pars fmt's")

	fmtFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, `pars fmt - format Parsley source files

Usage:
  pars fmt [options] <file>...

Options:
  -w    Write result to source file instead of stdout
  -d    Display diffs instead of rewriting files
  -l    List files whose formatting differs from pars fmt's

Examples:
  pars fmt script.pars           Print formatted output to stdout
  pars fmt -w script.pars        Format file in place
  pars fmt -l *.pars             List files that need formatting
  pars fmt -d script.pars        Show what would change
`)
	}

	if err := fmtFlags.Parse(args); err != nil {
		os.Exit(1)
	}

	files := fmtFlags.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no files specified")
		fmtFlags.Usage()
		os.Exit(1)
	}

	exitCode := 0
	for _, filename := range files {
		if err := formatFile(filename, *writeFlag, *diffFlag, *listFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", filename, err)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

// formatFile formats a single Parsley file
func formatFile(filename string, write, diff, list bool) error {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	source := string(content)

	// Create lexer and parser
	l := lexer.NewWithFilename(source, filename)
	p := parser.New(l)

	// Parse the program
	program := p.ParseProgram()
	if errs := p.StructuredErrors(); len(errs) != 0 {
		// Print parse errors
		lines := strings.Split(source, "\n")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err.PrettyString())
			printSourceContext(lines, err.Line, err.Column)
		}
		return fmt.Errorf("parse errors")
	}

	// Format the AST
	formatted := format.FormatProgram(program)

	// Ensure file ends with newline
	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}

	// Check if formatting changed anything
	changed := formatted != source

	if list {
		// List mode: just print filename if it would change
		if changed {
			fmt.Println(filename)
		}
		return nil
	}

	if diff {
		// Diff mode: show what would change
		if changed {
			showDiff(filename, source, formatted)
		}
		return nil
	}

	if write {
		// Write mode: update file in place
		if changed {
			if err := os.WriteFile(filename, []byte(formatted), 0644); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}
		}
		return nil
	}

	// Default: print to stdout
	fmt.Print(formatted)
	return nil
}

// showDiff displays a simple diff between original and formatted content
func showDiff(filename, original, formatted string) {
	fmt.Printf("diff %s\n", filename)

	origLines := strings.Split(original, "\n")
	fmtLines := strings.Split(formatted, "\n")

	// Simple line-by-line diff (not a full unified diff, but useful)
	maxLines := max(len(fmtLines), len(origLines))

	for i := range maxLines {
		origLine := ""
		fmtLine := ""
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(fmtLines) {
			fmtLine = fmtLines[i]
		}

		if origLine != fmtLine {
			if origLine != "" {
				fmt.Printf("-%d: %s\n", i+1, origLine)
			}
			if fmtLine != "" {
				fmt.Printf("+%d: %s\n", i+1, fmtLine)
			}
		}
	}
}
