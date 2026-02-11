---
id: FEAT-106
title: "CLI Enhancements: Inline Evaluation and Syntax Check"
status: implemented
priority: high
created: 2026-02-08
author: "@human"
---

# FEAT-106: CLI Enhancements: Inline Evaluation and Syntax Check

## Summary
Add two common CLI features to `pars` that users expect from modern language interpreters: inline code evaluation (`-e`) and syntax checking (`--check`). These features improve developer experience for quick testing, scripting, and CI/CD integration.

## User Story
As a developer, I want to evaluate Parsley code directly from the command line so that I can quickly test expressions without creating a file.

As a developer, I want to check Parsley files for syntax errors without executing them so that I can integrate syntax validation into my CI pipeline and editor tooling.

## Acceptance Criteria
- [x] `pars -e "code"` evaluates the provided code and prints the result
- [x] `pars -e "code"` supports all Parsley syntax including multiline (via shell escaping)
- [x] `pars -e "code"` respects existing flags (`-pp`, `--no-write`, etc.)
- [x] `pars --check file.pars` parses the file and reports any syntax errors
- [x] `pars --check` exits with code 0 on success, non-zero on errors
- [x] `pars --check` produces no output on success (Unix convention)
- [x] `pars --check` can check multiple files: `pars --check file1.pars file2.pars`
- [x] Both flags include help text in `pars --help`

## Design Decisions

- **`-e` not `-c`**: Python uses `-c` but Ruby/Perl use `-e`. We choose `-e` because it's more mnemonic ("evaluate" or "execute") and `-c` could be confused with "check".

- **`--check` not `--syntax-only`**: Shorter and matches Node.js convention. Also commonly understood.

- **No output on `--check` success**: Following Unix convention (like `go fmt -l`), successful checks produce no output. Errors go to stderr.

- **Exit codes for `--check`**: 0 = all files valid, 1 = syntax errors found, 2 = file not found or other error. This follows common conventions.

- **`-e` with `-pp`**: The `-pp` (pretty-print) flag should work with `-e` for HTML output formatting.

---

## Technical Context

### Affected Components
- `cmd/pars/main.go` — Add flag definitions, dispatch logic, and implementation

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints

1. **Empty `-e` argument** — `pars -e ""` should produce no output (evaluates to null)

2. **`-e` with file argument** — `pars -e "code" file.pars` is ambiguous. Decision: `-e` takes precedence, file argument is ignored with a warning, OR treat remaining args as `@args` for the evaluated code. Recommendation: treat as `@args`.

3. **`--check` with stdin** — Future consideration: `pars --check -` to read from stdin. Out of scope for initial implementation.

4. **`--check` with glob patterns** — Shell handles glob expansion, so `pars --check *.pars` works naturally.

5. **Multiple `-e` flags** — `pars -e "let x = 1" -e "x + 1"` could concatenate code. Recommendation: not supported initially; use semicolons or newlines within single `-e` argument.

6. **`-e` multiline code** — Shell handles this: `pars -e $'let x = 1\nx + 1'` (bash) or heredoc. No special handling needed in pars.

### API Design

```
Usage:
  pars [options] [file] [args...]
  pars -e "code" [args...]
  pars --check <file>...
  pars fmt [options] <file>...

Evaluation Options:
  -e, --eval <code>     Evaluate code string instead of file
  --check               Check syntax without executing (can specify multiple files)
```

### Implementation Sketch

```go
// In cmd/pars/main.go

var (
    // ... existing flags ...
    evalFlag      = flag.String("e", "", "Evaluate code string")
    evalLongFlag  = flag.String("eval", "", "Evaluate code string")
    checkFlag     = flag.Bool("check", false, "Check syntax without executing")
)

func main() {
    // Check for subcommands first
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "fmt":
            fmtCommand(os.Args[2:])
            return
        }
    }
    
    flag.Parse()
    
    // Get eval code (prefer -e over --eval if both somehow set)
    evalCode := *evalFlag
    if evalCode == "" {
        evalCode = *evalLongFlag
    }
    
    // Mode dispatch
    if evalCode != "" {
        // Inline evaluation mode
        executeInline(evalCode, flag.Args(), *prettyPrintFlag || *prettyLongFlag)
    } else if *checkFlag {
        // Syntax check mode
        files := flag.Args()
        if len(files) == 0 {
            fmt.Fprintln(os.Stderr, "Error: --check requires at least one file")
            os.Exit(2)
        }
        os.Exit(checkFiles(files))
    } else if len(flag.Args()) > 0 {
        // File execution mode
        filename := flag.Args()[0]
        scriptArgs := flag.Args()[1:]
        executeFile(filename, scriptArgs, *prettyPrintFlag || *prettyLongFlag)
    } else {
        // REPL mode
        repl.Start(os.Stdin, os.Stdout, Version)
    }
}

func executeInline(code string, args []string, prettyPrint bool) {
    policy, err := buildSecurityPolicy()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %s\n", err)
        os.Exit(1)
    }
    
    l := lexer.New(code)
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
    
    if evaluated != nil && evaluated.Type() == evaluator.ERROR_OBJ {
        errObj, ok := evaluated.(*evaluator.Error)
        if ok {
            printRuntimeError("<eval>", code, errObj)
        } else {
            fmt.Fprintf(os.Stderr, "%s\n", evaluated.Inspect())
        }
        os.Exit(1)
    }
    
    if evaluated != nil && evaluated.Type() != evaluator.ERROR_OBJ && evaluated.Type() != evaluator.NULL_OBJ {
        output := evaluator.ObjectToPrintString(evaluated)
        if prettyPrint {
            output = formatter.FormatHTML(output)
        }
        fmt.Println(output)
    }
}

func checkFiles(files []string) int {
    hasErrors := false
    
    for _, filename := range files {
        content, err := os.ReadFile(filename)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
            return 2  // File error
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
        return 1  // Syntax errors
    }
    return 0  // Success
}
```

## Test Plan

### `-e` Flag Tests

| Test Case | Command | Expected |
|-----------|---------|----------|
| Simple expression | `pars -e "1 + 2"` | Output: `3`, exit 0 |
| String output | `pars -e '"hello"'` | Output: `hello`, exit 0 |
| Println | `pars -e 'println("hi")'` | Output: `hi`, exit 0 |
| Syntax error | `pars -e "1 +"` | Error to stderr, exit 1 |
| Runtime error | `pars -e "1 / 0"` | Error to stderr, exit 1 |
| With args | `pars -e '@args' foo bar` | Output: `["foo", "bar"]`, exit 0 |
| With pretty-print | `pars -e '"<div>hi</div>"' -pp` | Formatted HTML output |
| Null result | `pars -e 'let x = 1'` | No output (null), exit 0 |
| Long flag | `pars --eval "1 + 2"` | Output: `3`, exit 0 |

### `--check` Flag Tests

| Test Case | Command | Expected |
|-----------|---------|----------|
| Valid file | `pars --check valid.pars` | No output, exit 0 |
| Invalid file | `pars --check invalid.pars` | Errors to stderr, exit 1 |
| File not found | `pars --check missing.pars` | Error to stderr, exit 2 |
| Multiple valid | `pars --check a.pars b.pars` | No output, exit 0 |
| One invalid | `pars --check valid.pars invalid.pars` | Errors for invalid, exit 1 |
| No files | `pars --check` | Error message, exit 2 |

## Implementation Notes

### Implementation Summary
Implementation completed on 2026-02-09. All acceptance criteria met.

### Changes Made
- Added `-e/--eval <code>` flags to `cmd/pars/main.go` for inline code evaluation
- Added `--check` flag to `cmd/pars/main.go` for syntax-only checking
- Implemented `executeInline()` function that:
  - Parses and evaluates inline code with `<eval>` as filename
  - Supports all security flags (`--no-read`, `--no-write`, etc.)
  - Passes remaining CLI args as `@args` to the evaluated code
  - Respects `-pp/--pretty` flag for HTML formatting
  - Uses same error reporting as file execution
- Implemented `checkFiles()` function that:
  - Parses multiple files without executing them
  - Returns exit code 0 (success), 1 (syntax errors), or 2 (file errors)
  - Produces no output on success (Unix convention)
  - Prints structured errors on failure
- Updated `main()` dispatch logic with switch statement for clarity
- Updated `printHelp()` with new flags, usage patterns, and examples

### Test Results
All manual tests passed:
- `-e` flag: expressions, strings, println, null results, args pass-through
- `--eval` long flag works correctly
- `--check` flag: single/multiple files, valid/invalid syntax, missing files
- Exit codes correct in all scenarios
- Security flags work with `-e` mode
- Pretty-print works with `-e` mode
- Help text displays correctly

### Deferred Items
- Multiple `-e` flags support (can use semicolons for now)
- `--check` with stdin input (`pars --check -`)
- JSON output format for `--check` (for tooling integration)

### Code Quality
- All parsley package tests pass
- No new linter issues in modified code
- Refactored dispatch logic to switch statement per gocritic recommendation

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md`
- Similar: Python `-c`, Ruby `-e`, Perl `-e`, Node `-e`/`--check`
