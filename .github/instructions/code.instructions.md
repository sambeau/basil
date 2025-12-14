# Code Standards

These rules apply to all code changes in this repository.

## Go Code Style

### Formatting
- Run `go fmt` before committing
- Use `goimports` for import organization
- Maximum line length: 100 characters (soft limit)

### Naming
- Use camelCase for unexported identifiers
- Use PascalCase for exported identifiers
- Acronyms should be consistent case: `URL`, `HTTP`, `ID` (not `Url`, `Http`, `Id`)
- Package names: lowercase, single word, no underscores

### Error Handling
- Always check errors; never use `_` to ignore them
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Return errors, don't panic (except for truly unrecoverable situations)

### Error Messages (Parsley)
After implementing any feature that produces error messages, verify:
- **Capitalization**: Messages start with a capital letter (unless starting with a code like `PARSE-0001`)
- **Line numbers**: All runtime errors include line and column information
- **Hints**: Complex errors include actionable hints
- **Consistency**: Use existing error catalog in `pkg/parsley/errors/errors.go` when possible
- **Testing**: Error message tests should use case-insensitive matching (`strings.Contains(strings.ToLower(...))`)

### Comments
- Exported functions must have doc comments
- Doc comments start with the function name: `// FunctionName does...`
- Use `// TODO:` for planned improvements
- Use `// FIXME:` for known issues

### Testing
- Test files: `*_test.go` in the same package
- Test functions: `TestFunctionName_Scenario`
- Use table-driven tests for multiple cases
- Aim for meaningful coverage, not 100%

## File Organization
```
cmd/          # Main applications (if multiple)
internal/     # Private packages
pkg/          # Public packages (if any)
*.go          # Root-level for simple CLIs
```

## Dependencies
- Prefer standard library when reasonable
- Run `go mod tidy` after adding/removing dependencies
- Commit `go.sum` with `go.mod`

## Testing
- All code changes must include tests
- Run tests frequently during implementation
- Update test files in `pkg/parsley/tests/` for Parsley language features
- Bug fixes must include regression tests

## Builtin Function Introspection
When adding, modifying, or removing builtin functions:

1. **Update `BuiltinMetadata` map** in `pkg/parsley/evaluator/introspect.go`:
   - Add entries for new builtins
   - Update descriptions for modified builtins
   - Remove entries for deprecated/removed builtins
   - Ensure `Arity`, `Params`, and `Category` are accurate

2. **Audit checklist** (run periodically or when touching builtins):
   - [ ] Every function in `getBuiltins()` has a `BuiltinMetadata` entry
   - [ ] Every `BuiltinMetadata` entry matches an actual builtin
   - [ ] All parameter names end with `?` for optional params
   - [ ] Arity strings match actual function behavior (e.g., `"1"`, `"1-2"`, `"0+"`, `"1+"`)
   - [ ] Categories are consistent across similar functions
   - [ ] Deprecated builtins have non-empty `Deprecated` messages

3. **Test coverage**:
   - `inspect(builtin_name)` returns expected metadata
   - `describe(builtin_name)` produces readable output
   - No missing or outdated introspection data

**Location reference**: See FEAT-069 implementation plan for full details.

