# Testing Guide for Parsley Zed Extension

This guide walks you through testing the Parsley extension in Zed Editor.

## Prerequisites

- Zed Editor installed ([download here](https://zed.dev))
- This repository cloned locally
- **IMPORTANT**: The `tree-sitter-parsley` repository must have generated parser files committed

### Fixing tree-sitter-parsley Repository

Before testing, ensure the tree-sitter-parsley repository includes the generated parser files:

1. Clone or navigate to the tree-sitter-parsley repo:
   ```bash
   git clone https://github.com/sambeau/tree-sitter-parsley.git
   cd tree-sitter-parsley
   ```

2. Edit `.gitignore` to remove these lines:
   ```
   # Remove or comment out:
   /src/parser.c
   /src/tree_sitter/
   /src/node-types.json
   /src/grammar.json
   ```

3. Generate the parser:
   ```bash
   npm install
   tree-sitter generate
   ```

4. Commit and push the generated files:
   ```bash
   git add src/
   git commit -m "Include generated parser files for Zed extension"
   git push
   ```

Without these files in the repository, Zed cannot compile the grammar.

## Installation for Testing

1. **Open Zed Editor**

2. **Install as Dev Extension**
   - Press `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Linux/Windows)
   - Type "install dev extension"
   - Select **"zed: install dev extension"**
   - Navigate to and select the `contrib/zed-extension/` directory
   - Click "Select"

3. **Verify Installation**
   - Press `Cmd+Shift+P` → "zed: extensions"
   - You should see "Parsley" listed with "Overridden by dev extension"

## Test Cases

### 1. File Type Recognition

**Test**: Open a Parsley file
- [ ] Open `contrib/zed-extension/test/sample.pars`
- [ ] Verify the language indicator in the bottom-right shows "Parsley"
- [ ] Create a new file with `.part` extension
- [ ] Verify it's also recognized as Parsley

**Expected**: Both `.pars` and `.part` files are automatically detected.

---

### 2. Syntax Highlighting

**Test**: Verify highlighting for different token types

Open `test/sample.pars` and check:

- [ ] **Keywords** (`let`, `export`, `fn`, `for`, `if`, `else`, `return`, etc.) are highlighted
- [ ] **Operators** (`+`, `-`, `*`, `/`, `==`, `=>`, `|>`, etc.) are distinct
- [ ] **Strings** (double quotes, backticks, single quotes) are highlighted
- [ ] **Numbers** (integers, floats, money like `$100.50`) are highlighted
- [ ] **Booleans** (`true`, `false`) and **null** are highlighted
- [ ] **Comments** (`//`) are dimmed/italicized
- [ ] **At-literals** (`@sqlite`, `@now`, `@std/fs`, etc.) are highlighted
- [ ] **Functions** in definitions and calls are distinguishable
- [ ] **Tag names** (`<button>`, `<div>`) are highlighted
- [ ] **Attribute names** in tags are highlighted

**Expected**: All syntax elements have appropriate colors based on your theme.

---

### 3. Bracket Matching

**Test**: Verify bracket pairs highlight correctly

In `test/sample.pars`:

- [ ] Place cursor inside `[1, 2, 3]` → brackets should highlight
- [ ] Place cursor inside `{name: "Alice"}` → braces should highlight
- [ ] Place cursor inside `fn(x)` → parentheses should highlight
- [ ] Place cursor inside `<div>...</div>` → angle brackets should highlight
- [ ] Verify rainbow brackets work for arrays and dictionaries (not tags/strings)

**Expected**: Matching brackets highlight when cursor is positioned inside them.

---

### 4. Code Outline

**Test**: Verify outline populates with functions and exports

1. Open `test/sample.pars`
2. Open the Outline panel (if not visible, try `Cmd+Shift+O` or View menu)

Check that outline shows:
- [ ] `greet` function
- [ ] `calculate` function
- [ ] `sum` function
- [ ] `message` export
- [ ] `result` export
- [ ] `process_data` function
- [ ] Other top-level let statements

**Expected**: Functions and exports appear in the outline for quick navigation.

---

### 5. Auto-Indentation

**Test**: Verify smart indentation

1. Open a new `.pars` file or edit `test/sample.pars`
2. Type the following and press Enter after each line:

```parsley
let data = {
  users: [
    {name: "Alice"},
```

- [ ] After typing `{` and pressing Enter, cursor indents
- [ ] After typing `[` and pressing Enter, cursor indents further
- [ ] After typing `{name: "Alice"}` and closing brackets, dedent occurs

**Expected**: Automatic indentation and dedentation based on brackets.

---

### 6. Comment Toggling

**Test**: Verify line comment behavior

1. Select a line of code
2. Press `Cmd+/` (Mac) or `Ctrl+/` (Linux/Windows)

- [ ] Line is commented with `//`
- [ ] Press again to uncomment

**Expected**: Toggle line comments with keyboard shortcut.

---

### 7. Tag Folding

**Test**: Verify code folding works

In `test/sample.pars`:

- [ ] Click the fold indicator next to a multi-line tag (e.g., `<div>`)
- [ ] Verify the tag content collapses
- [ ] Click again to expand

**Expected**: Multi-line structures can be folded/unfolded.

---

### 8. Performance

**Test**: Verify extension doesn't slow down Zed

- [ ] Open `test/sample.pars` (200+ lines)
- [ ] Scroll through the file quickly
- [ ] Type new code
- [ ] Verify no lag or stuttering

**Expected**: Smooth performance even with syntax highlighting active.

---

## Making Changes

If you need to modify the extension:

1. Edit query files in `contrib/zed-extension/languages/parsley/`
   - `highlights.scm` - Syntax highlighting rules
   - `brackets.scm` - Bracket matching
   - `outline.scm` - Code structure
   - `indents.scm` - Indentation rules

2. Reload the extension:
   - Press `Cmd+Shift+P` → "zed: reload extensions"
   - Or restart Zed

3. Re-test the affected features

## Common Issues

### Extension Not Loading

**Symptom**: Parsley files aren't recognized
**Solution**:
- Verify dev extension is installed: `Cmd+Shift+P` → "zed: extensions"
- Check Zed log: `Cmd+Shift+P` → "zed: open log"
- Look for errors related to "parsley" extension

### Syntax Not Highlighting

**Symptom**: File is recognized but no colors
**Solution**:
- Check that `highlights.scm` exists and has content
- Verify Tree-sitter grammar reference in `extension.toml` is correct
- Reload extensions: `Cmd+Shift+P` → "zed: reload extensions"

### Brackets Not Matching

**Symptom**: Bracket highlighting doesn't work
**Solution**:
- Verify `brackets.scm` exists
- Check for syntax errors in the query file
- Reload extensions

### Outline Empty

**Symptom**: Outline panel doesn't show items
**Solution**:
- Verify `outline.scm` exists and matches your code structure
- Check that functions/exports use correct Tree-sitter node names
- Test with `test/sample.pars` which is known to work

## Reporting Issues

If you find bugs:

1. Check the Zed log for errors: `Cmd+Shift+P` → "zed: open log"
2. Note the specific file and line where the issue occurs
3. Create a minimal reproduction case
4. Report at [github.com/sambeau/parsley-zed/issues](https://github.com/sambeau/parsley-zed/issues)

## Debug Mode

To see more verbose output:

1. Close Zed
2. Launch from terminal with: `zed --foreground`
3. Extension debug output will appear in the terminal

## Next Steps

Once testing is complete and all tests pass:

1. Document any issues found in the Progress Log
2. Fix issues if needed
3. Proceed to create standalone repository
4. Submit to Zed extensions registry

## Success Criteria

Extension is ready for release when:
- ✅ All 8 test cases pass
- ✅ No errors in Zed log
- ✅ Performance is acceptable
- ✅ Works with your theme(s)
- ✅ Documentation is accurate