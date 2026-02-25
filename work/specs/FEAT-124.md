---
id: FEAT-124
title: pars CLI Improvements
status: complete
created: 2025-02-25
updated: 2025-02-25
---

# FEAT-124: pars CLI Improvements

## Summary

Improve the `pars` CLI to be more consistent with Unix conventions, add helpful hints for common mistakes, and improve AI/machine friendliness.

## Motivation

An audit of the `pars` CLI revealed several gaps:

1. **Missing shebang support** - Scripts with `#!/usr/bin/env pars` fail to parse
2. **Missing conventional flags** - `-c` for check, `-v` for version
3. **Confusing stdio display** - `@stdin`, `@stdout`, `@stderr` all display as `@-`
4. **Spurious output** - `null` printed after stdout writes
5. **No hints for common mistakes** - Users try `JSON(@-)` expecting data, get a handle
6. **Limited machine-readable output** - No consistent way to get JSON output everywhere

## Detailed Design

### 1. Shebang Support (P0)

**Problem:** `#!` at start of file causes parse error.

**Solution:** In the lexer, skip the first line if it starts with `#!`.

**Test cases:**
```sh
#!/usr/bin/env pars
"Hello, World!"
```

### 2. Add `-c` Alias for `--check` (P1)

**Problem:** Every scripting language uses `-c` for syntax check (Ruby, Node).

**Solution:** Add flag alias.

**Behavior:** `pars -c file.pars` checks syntax without executing.

### 3. Add `-v` Alias for `--version` (P1)

**Problem:** Most tools use `-v` for version.

**Solution:** Add flag alias.

**Note:** Some tools use `-v` for verbose. Since Parsley doesn't have a verbose mode, `-v` for version is safe.

### 4. Fix Stdio Path Display (P1)

**Problem:** `@stdin`, `@stdout`, `@stderr` all display as `@-`.

**Expected:**
```sh
$ pars -e '@stdin'
@stdin
$ pars -e '@stdout'
@stdout
```

**Solution:** In `pathDictToString()`, check the `__stdio` field and display the specific stream name.

### 5. Add Hints for File Handle Confusion (P1)

**Problem:** Users expect `JSON(@-)` to return data, but it returns a handle.

**Solution:** In `executeInline()`, after successful evaluation, check if the result is a file dict and print a hint to stderr.

**Example:**
```sh
$ echo '{}' | pars -e 'JSON(@-)'
@-
hint: Result is a file handle. Use '<== ...' to read contents.
```

Note: The hint uses `<== ...` rather than a specific function name because it applies to all file handle types (`JSON`, `CSV`, `text`, `lines`, etc.).

### 6. Add `--format` Flag (P2)

**Problem:** No consistent way to control output format across commands.

**Solution:** Add global `--format` flag with values: `text` (default), `json`, `pln`.

**Behavior by command:**

| Command | `--format text` | `--format json` |
|---------|-----------------|-----------------|
| `pars -e "expr"` | PLN repr | `{"ok": true, "value": ..., "type": "..."}` |
| `pars describe X` | Current default | Current `--json` behavior |

### 7. Add `--machine` Flag (P2)

**Problem:** AI/scripts need predictable, parseable output.

**Solution:** Add `--machine` flag that enables all machine-friendly behaviors.

**Equivalent to:** `--format json --quiet` (suppresses hints) plus structured exit codes.

**Exit codes in machine mode:**

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error or IO error |
| 2 | Parse/syntax error or usage error |

Note: Go's `flag` package defaults to exit code 2 for usage errors, so we follow that convention.

### 8. Add `pars describe all --json` (P2)

**Problem:** AI has to query each topic separately.

**Solution:** Add `all` as a valid topic that returns complete schema.

```sh
$ pars describe all --json
{
  "types": [...],
  "builtins": [...],
  "operators": [...],
  "modules": [...]
}
```

### 9. Structured Exit Codes (P3)

**Problem:** All errors return exit code 1, making it hard to distinguish error types.

**Solution:** Use distinct exit codes:
- Exit 0: Success
- Exit 1: Runtime error or IO error  
- Exit 2: Parse/syntax error or usage error (follows Go's `flag` package convention)

**Implementation:** Parse errors now exit with code 2, distinguishing them from runtime errors (exit 1).

## Implementation Plan

### Phase 1: Critical & Convention Fixes
- [x] 1. Shebang support in lexer
- [x] 2. `-c` alias for `--check`
- [x] 3. `-v` alias for `--version`
- [x] 4. Fix stdio path display

### Phase 2: Hints & Usability
- [x] 5. File handle hints in `-e` mode

### Phase 3: Machine Friendliness
- [x] 6. `--format` flag
- [x] 7. `--machine` flag
- [x] 8. `pars describe all --json`
- [x] 9. Structured exit codes (exit 2 for parse errors, exit 1 for runtime errors)

## Test Plan

### Shebang
```sh
echo '#!/usr/bin/env pars
"Hello"' > /tmp/test.pars
pars /tmp/test.pars  # Should output: Hello
```

### Flag Aliases
```sh
pars -c syntax_error.pars  # Should report syntax error
pars -v                     # Should print version
```

### Stdio Display
```sh
pars -e '@stdin'   # Should output: @stdin
pars -e '@stdout'  # Should output: @stdout
pars -e '@stderr'  # Should output: @stderr
pars -e '@-'       # Should output: @-
```

### File Handle Hints
```sh
echo '{}' | pars -e 'JSON(@-)'
# Should output:
# @-
# hint: Result is a file handle. Use '<== ...' to read contents.
```

### Machine Mode
```sh
pars --machine -e '1 + 2'
# Should output: {"ok": true, "value": 3, "type": "integer"}

pars --machine -e 'x'
# Should output: {"ok": false, "error": {"code": "...", "message": "..."}}
# Exit code: 1
```

## Compatibility

All changes are backwards compatible:
- New flags are opt-in
- Hints go to stderr, not stdout
- Default behavior unchanged

## Open Questions (Resolved)

1. Should `--quiet` suppress hints? **Yes** - implemented.
2. Should hints also appear in REPL? Deferred - would require a warning system in the evaluator.
3. Should `--machine` imply `--quiet`? **Yes** - implemented.

## Related

- [File I/O Documentation](../../docs/parsley/manual/features/file-io.md)
- [Error Handling](../../pkg/parsley/errors/)