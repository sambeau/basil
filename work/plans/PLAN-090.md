---
id: PLAN-090
feature: FEAT-109
title: "Implementation Plan for Zed Editor Extension"
status: draft
created: 2026-02-11
---

# Implementation Plan: Zed Editor Extension for Parsley

## Overview
Create a Zed Editor extension for Parsley language support. This extension will provide syntax highlighting, bracket matching, code outline, and auto-indentation for `.pars` and `.part` files using our existing Tree-sitter grammar.

**Based on:** Investigation in `work/specs/FEAT-109-ZED-INVESTIGATION.md`

## Prerequisites
- [x] Tree-sitter grammar complete (`contrib/tree-sitter-parsley/`)
- [x] Grammar tests passing (129/129)
- [x] `highlights.scm` query file exists
- [x] MIT license in place
- [ ] Zed Editor installed for local testing

## Tasks

### Task 1: Create Extension Structure
**Location**: `contrib/zed-extension/`
**Estimated effort**: Small (30 min)

Steps:
1. Create directory structure:
   ```
   contrib/zed-extension/
   ├── extension.toml
   └── languages/
       └── parsley/
           ├── config.toml
           ├── highlights.scm
           ├── brackets.scm
           ├── outline.scm
           └── indents.scm
   ```
2. Create `extension.toml` with metadata
3. Configure grammar reference to point to `contrib/tree-sitter-parsley`

Tests:
- Directory structure matches Zed extension conventions
- All required files are present

---

### Task 2: Configure Language Metadata
**Files**: `contrib/zed-extension/extension.toml`, `contrib/zed-extension/languages/parsley/config.toml`
**Estimated effort**: Small (30 min)

Steps:
1. Write `extension.toml`:
   - Set extension ID to "parsley"
   - Set version to "0.1.0"
   - Reference Tree-sitter grammar at `https://github.com/sambeau/basil` (path: `contrib/tree-sitter-parsley`)
   - Add author information
   - Add description

2. Write `languages/parsley/config.toml`:
   - Set `name = "Parsley"`
   - Set `grammar = "parsley"`
   - Set `path_suffixes = ["pars", "part"]`
   - Set `line_comments = ["// "]`
   - Set `tab_size = 2`
   - Set `hard_tabs = false`

Tests:
- TOML files parse correctly
- Grammar reference is valid URL
- File extension matching works

---

### Task 3: Copy and Verify Highlights Query
**Files**: `contrib/zed-extension/languages/parsley/highlights.scm`
**Estimated effort**: Small (15 min)

Steps:
1. Copy `contrib/tree-sitter-parsley/queries/highlights.scm` to extension
2. Verify all captures are Zed-compatible (they are, per investigation)
3. No modifications needed

Tests:
- File copied successfully
- Syntax is valid Tree-sitter query format

---

### Task 4: Create Brackets Query
**Files**: `contrib/zed-extension/languages/parsley/brackets.scm`
**Estimated effort**: Small (15 min)

Steps:
1. Create `brackets.scm` with bracket pairs:
   - `[ ]` - Arrays
   - `{ }` - Dictionaries, blocks
   - `( )` - Function calls, grouping
   - `< >` - Tags (without rainbow coloring)
   - `" "` - Strings (without rainbow coloring)

2. Add `rainbow.exclude` for strings and tags

Tests:
- Query syntax is valid
- All Parsley bracket types are covered

---

### Task 5: Create Outline Query
**Files**: `contrib/zed-extension/languages/parsley/outline.scm`
**Estimated effort**: Small (20 min)

Steps:
1. Create `outline.scm` to capture:
   - Function definitions (`@name` and `@item`)
   - Export statements (`@name` and `@item`)
   - Let statements at top level (`@name` and `@item`)

2. Test patterns against sample Parsley code

Tests:
- Query matches function definitions
- Query matches export statements
- Query matches let statements
- No false positives

---

### Task 6: Create Indents Query
**Files**: `contrib/zed-extension/languages/parsley/indents.scm`
**Estimated effort**: Small (20 min)

Steps:
1. Create `indents.scm` with indent rules:
   - Mark `@indent` for: arrays, dictionaries, function_expression, tag_expression, for_statement, if_statement, try_statement
   - Mark `@end` for: `]`, `}`, `)`

2. Test against sample Parsley code with various nesting levels

Tests:
- Indentation increases inside blocks
- Indentation decreases at closing brackets
- Multi-level nesting works correctly

---

### Task 7: Local Testing
**Files**: All extension files
**Estimated effort**: Small (30 min)

Steps:
1. Install Zed Editor if not already installed
2. In Zed: `Cmd+Shift+P` → "zed: install dev extension"
3. Select `contrib/zed-extension/` directory
4. Open test Parsley files:
   - Simple expressions
   - Functions
   - Tags
   - Nested structures
5. Verify:
   - Syntax highlighting works
   - Bracket matching works
   - Outline populates correctly
   - Auto-indentation works
6. Test file type detection (`.pars` and `.part`)

Tests:
- Extension loads without errors
- `.pars` files are recognized as Parsley
- `.part` files are recognized as Parsley
- Syntax highlighting matches expectations
- Brackets highlight when cursor is inside them
- Outline shows all functions and exports
- Auto-indentation works on new lines

---

### Task 8: Documentation
**Files**: `contrib/zed-extension/README.md`
**Estimated effort**: Small (20 min)

Steps:
1. Create README with:
   - Description of extension
   - Features list
   - Installation instructions (for dev extension)
   - Usage notes
   - Link to Parsley documentation
   - License information
   - Contributing guidelines

Tests:
- README is clear and accurate
- Installation instructions are testable

---

### Task 9: Prepare Standalone Repository
**Location**: New repo `github.com/sambeau/parsley-zed`
**Estimated effort**: Small (30 min)

Steps:
1. Create new GitHub repository `parsley-zed`
2. Copy extension files from `contrib/zed-extension/`
3. Update grammar reference in `extension.toml` if needed
4. Add LICENSE (MIT)
5. Add README
6. Add `.gitignore`
7. Create initial commit and push

Tests:
- Repository is public
- All files are present
- README renders correctly on GitHub
- License is visible

---

### Task 10: Submit to Zed Extensions Registry
**Location**: Fork of `github.com/zed-industries/extensions`
**Estimated effort**: Small (30 min)

Steps:
1. Fork `zed-industries/extensions`
2. Clone fork locally
3. Add Parsley extension as submodule:
   ```bash
   git submodule add https://github.com/sambeau/parsley-zed.git extensions/parsley
   ```
4. Add entry to `extensions.toml`:
   ```toml
   [parsley]
   submodule = "extensions/parsley"
   version = "0.1.0"
   ```
5. Run `pnpm sort-extensions` to sort files
6. Commit changes
7. Push to fork
8. Create PR with description:
   - What is Parsley
   - Link to language documentation
   - Link to Tree-sitter grammar
   - Screenshots of syntax highlighting (optional)

Tests:
- Submodule added correctly
- `extensions.toml` is valid TOML
- Files are sorted
- PR description is clear and complete

---

## Validation Checklist
- [ ] Extension structure matches Zed conventions
- [ ] All query files have valid syntax
- [ ] Extension loads in Zed without errors
- [ ] Syntax highlighting works for all Parsley constructs
- [ ] Bracket matching works
- [ ] Code outline populates correctly
- [ ] Auto-indentation works
- [ ] File type detection works (`.pars`, `.part`)
- [ ] Documentation is complete and accurate
- [ ] Standalone repository is created
- [ ] PR submitted to Zed extensions registry
- [ ] Update FEAT-109.md acceptance criteria

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-02-11 | Plan created | ✅ Complete | — |
| 2026-02-11 | Task 1: Extension structure | ✅ Complete | Created in contrib/zed-extension/ |
| 2026-02-11 | Task 2: Language metadata | ✅ Complete | extension.toml and config.toml written |
| 2026-02-11 | Task 3: Highlights query | ✅ Complete | Copied from tree-sitter-parsley |
| 2026-02-11 | Task 4: Brackets query | ✅ Complete | All bracket types covered |
| 2026-02-11 | Task 5: Outline query | ✅ Complete | Functions and exports captured |
| 2026-02-11 | Task 6: Indents query | ✅ Complete | Indent rules for all blocks |
| 2026-02-11 | Task 7: Local testing | ⏸️ Pending | Ready for user to test in Zed |
| 2026-02-11 | Task 8: Documentation | ✅ Complete | README, LICENSE, .gitignore created |
| — | Task 9: Standalone repo | ⏸️ Pending | Awaiting user decision |
| — | Task 10: Submit to registry | ⏸️ Pending | Requires standalone repo first |

## Deferred Items
No items deferred. Extension implemented as planned.

## Implementation Notes

### Completed Structure
```
contrib/zed-extension/
├── extension.toml              ✅ Created
├── LICENSE                     ✅ Created (MIT)
├── README.md                   ✅ Created
├── .gitignore                  ✅ Created
├── languages/
│   └── parsley/
│       ├── config.toml         ✅ Created
│       ├── highlights.scm      ✅ Copied from tree-sitter-parsley
│       ├── brackets.scm        ✅ Created
│       ├── outline.scm         ✅ Created
│       └── indents.scm         ✅ Created
└── test/
    └── sample.pars             ✅ Created for testing
```

### Ready for Testing
The extension is ready for local testing in Zed:
1. Open Zed Editor
2. `Cmd+Shift+P` → "zed: install dev extension"
3. Select `contrib/zed-extension/` directory
4. Open `test/sample.pars` to verify highlighting
5. Test all features: highlighting, brackets, outline, indentation

### Next Steps
Once validated:
1. Create standalone repository `github.com/sambeau/parsley-zed`
2. Copy extension files to new repo
3. Submit PR to `zed-industries/extensions`

## Notes
- **Total estimated effort:** 3.5 hours
- **Critical path:** Tasks 1-7 must be completed before Tasks 9-10
- **Optional enhancements for future:**
  - Language server integration (requires Rust/WASM)
  - Semantic highlighting (requires LSP)
  - Code actions/refactorings (requires LSP)
  - Snippets
  - Textobjects for Vim mode
- **Testing sample files location:** `pkg/parsley/tests/` contains comprehensive test cases
- **Reference extensions:** Zig, Gleam, Odin extensions in `zed-industries/extensions`

## Success Criteria
Extension is considered successful when:
1. Available in Zed's extension marketplace
2. Parsley files are automatically recognized and highlighted
3. Basic editor features (brackets, outline, indentation) work correctly
4. No errors or warnings in Zed's log when using the extension