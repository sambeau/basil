---
id: FEAT-109-ZED-INVESTIGATION
title: "Zed Editor Extension Investigation"
related: FEAT-109
created: 2025-02-11
author: "@ai"
---

# Zed Editor Extension Investigation

## Executive Summary

**Creating a Zed Editor extension for Parsley is highly feasible and straightforward.** Our existing Tree-sitter grammar (`contrib/tree-sitter-parsley/`) has everything needed. Zed extensions are simpler than VS Code extensions because they leverage Tree-sitter directly without custom language server requirements for basic syntax highlighting.

**Recommendation:** Proceed with creating a Zed extension. Estimated effort: 2-4 hours.

---

## What We Have

### Current Assets
1. ✅ **Complete Tree-sitter grammar** at `contrib/tree-sitter-parsley/`
   - Grammar covers all Parsley syntax
   - 129/129 corpus tests passing
   - `highlights.scm` query file ready

2. ✅ **MIT License** in place (required by Zed)

3. ✅ **Repository structure** suitable for publishing

4. ✅ **Highlight queries** that map to Zed's supported captures

### What Zed Needs

According to Zed's documentation, a language extension requires:

```
my-extension/
  extension.toml          # Extension metadata
  languages/
    parsley/
      config.toml         # Language configuration
      highlights.scm      # Syntax highlighting (we have this!)
      brackets.scm        # Optional: bracket matching
      outline.scm         # Optional: code outline
      indents.scm         # Optional: auto-indentation
```

**We already have 80% of what's needed.** The Tree-sitter grammar and highlights.scm exist.

---

## Zed Extension Architecture

### How Zed Extensions Work

1. **Grammar Registration**: Extensions reference Tree-sitter grammar repos via Git URL
2. **Automatic Discovery**: Zed discovers grammars from registered extensions
3. **No Compilation**: Users don't compile anything—Zed handles it
4. **WebAssembly for Advanced Features**: Only needed for language servers or custom behavior

### For Parsley

**We need the simplest extension type:**
- Language metadata (file types, comment syntax)
- Reference to our Tree-sitter grammar
- Query files for highlighting

**We do NOT need:**
- Custom Rust/WASM code
- Language server integration (LSP)
- Complex build pipeline

---

## Implementation Plan

### Step 1: Create Extension Structure

Create a new directory structure (can be in `contrib/` or separate repo):

```
parsley-zed/
├── extension.toml
└── languages/
    └── parsley/
        ├── config.toml
        ├── highlights.scm      # Copy from tree-sitter-parsley
        ├── brackets.scm        # New, simple
        ├── outline.scm         # New, simple
        └── indents.scm         # New, simple
```

### Step 2: Extension Metadata

**`extension.toml`:**
```toml
id = "parsley"
name = "Parsley"
version = "0.1.0"
schema_version = 1
authors = ["Basil Contributors <basil@example.com>"]
description = "Parsley language support for Zed"
repository = "https://github.com/sambeau/parsley-zed"

[grammars.parsley]
repository = "https://github.com/sambeau/tree-sitter-parsley"
rev = "main"  # Or specific commit SHA for stability
```

### Step 3: Language Configuration

**`languages/parsley/config.toml`:**
```toml
name = "Parsley"
grammar = "parsley"
path_suffixes = ["pars", "part"]
line_comments = ["// "]
tab_size = 2
hard_tabs = false

# Optional: Detect shebang lines
# first_line_pattern = "^#!.*\\bparsley\\b"
```

### Step 4: Query Files

**`highlights.scm`:** Copy directly from `contrib/tree-sitter-parsley/queries/highlights.scm`

**`brackets.scm`:** (New file)
```scheme
("[" @open "]" @close)
("{" @open "}" @close)
("(" @open ")" @close)
("<" @open ">" @close)  ; For tags
```

**`outline.scm`:** (New file)
```scheme
; Show function definitions in outline
(function_definition
  name: (identifier) @name) @item

; Show exported values
(export_statement
  name: (identifier) @name) @item

; Show let statements
(let_statement
  name: (identifier) @name) @item
```

**`indents.scm`:** (New file)
```scheme
; Indent within braces, brackets, parens
(array) @indent
(dictionary) @indent
(function_expression) @indent
(tag_expression) @indent

; End markers
("]" @end)
("}" @end)
(")" @end)
```

### Step 5: Repository Setup

**Option A: Standalone Repository** (Recommended for publishing)
1. Create `github.com/sambeau/parsley-zed`
2. Add extension files
3. Add README with installation instructions

**Option B: Monorepo Subdirectory** (For initial development)
1. Create `contrib/zed-extension/`
2. Develop locally using Zed's "Install Dev Extension" feature
3. Move to standalone repo when ready to publish

### Step 6: Local Testing

Zed supports installing dev extensions:

1. Open Zed
2. `Cmd+Shift+P` → "zed: install dev extension"
3. Select the extension directory
4. Open a `.pars` file to test highlighting

### Step 7: Publishing

To publish to Zed's extension registry:

1. Fork `https://github.com/zed-industries/extensions`
2. Add our extension as a Git submodule:
   ```bash
   git submodule add https://github.com/sambeau/parsley-zed.git extensions/parsley
   ```
3. Add entry to `extensions.toml`:
   ```toml
   [parsley]
   submodule = "extensions/parsley"
   version = "0.1.0"
   ```
4. Run `pnpm sort-extensions` (they require sorted order)
5. Submit PR

---

## Highlight Query Compatibility

Our existing `highlights.scm` uses these captures, which are **all supported by Zed**:

| Our Capture | Zed Support | Notes |
|-------------|-------------|-------|
| `@keyword` | ✅ | Fully supported |
| `@keyword.operator` | ✅ | Fully supported |
| `@number` | ✅ | Fully supported |
| `@string` | ✅ | Fully supported |
| `@string.escape` | ✅ | Fully supported |
| `@string.regexp` | ✅ | Fully supported |
| `@string.special.path` | ✅ | Fully supported |
| `@string.special.url` | ✅ | Fully supported |
| `@constant.builtin` | ✅ | Fully supported |
| `@function` | ✅ | Fully supported |
| `@function.builtin` | ✅ | Fully supported |
| `@function.call` | ✅ | Fully supported |
| `@variable` | ✅ | Fully supported |
| `@variable.builtin` | ✅ | Fully supported |
| `@operator` | ✅ | Fully supported |
| `@punctuation.bracket` | ✅ | Fully supported |
| `@punctuation.delimiter` | ✅ | Fully supported |
| `@comment` | ✅ | Fully supported |
| `@type` | ✅ | Fully supported |
| `@tag` | ✅ | Fully supported |
| `@attribute` | ✅ | Fully supported |
| `@module` | ✅ | Fully supported |

**No changes needed to our existing highlights.scm!**

---

## Advanced Features (Future)

Once basic highlighting works, we can add:

1. **Language Server Protocol (LSP):** Would require Rust/WASM extension code
2. **Semantic Highlighting:** If we build an LSP
3. **Code Actions:** Refactorings, quick fixes (requires LSP)
4. **Diagnostics:** Syntax errors, warnings (could integrate with Parsley interpreter)

These are **not needed for initial release.** Syntax highlighting alone provides significant value.

---

## Comparison with Other Editors

| Editor | Status | Effort | Notes |
|--------|--------|--------|-------|
| **Zed** | ⏳ Planned | **Low** | Tree-sitter grammar → extension (2-4 hours) |
| **VS Code** | ✅ Done | Medium | Custom extension with TextMate grammar |
| **Neovim** | ✅ Ready | Low | Users can add Tree-sitter config |
| **Helix** | ✅ Ready | Low | Users can add Tree-sitter config |
| **GitHub** | ⏳ PR Open | Low | Linguist PR submitted, awaiting merge |

**Zed is easier than VS Code** because it uses Tree-sitter natively—no separate TextMate grammar needed.

---

## Risks and Considerations

### Low Risk
- ✅ Tree-sitter grammar is stable (129/129 tests passing)
- ✅ License is compatible (MIT)
- ✅ No compilation required for end users
- ✅ Zed's extension API is stable

### Medium Risk
- ⚠️ **Zed's rapid development:** API may change, but they maintain backward compatibility
- ⚠️ **Extension approval:** Zed maintainers review PRs, may request changes

### Mitigations
- Start with dev extension to validate approach
- Follow existing language extensions as templates (Zig, Gleam, etc.)
- Test thoroughly before submitting PR

---

## Dependencies

### Tree-sitter Grammar Status
- ✅ Grammar implemented: `contrib/tree-sitter-parsley/`
- ✅ Tests passing: 129/129
- ❓ Standalone repository: Not yet published separately (currently in `basil` monorepo)

### Action Required
Before publishing Zed extension:
1. **Publish Tree-sitter grammar as standalone repo** (or ensure monorepo URL works)
2. The grammar must be accessible via Git URL for Zed to fetch it

**Current URL:** `https://github.com/sambeau/basil` (subfolder access works!)
**Zed can reference:** `https://github.com/sambeau/basil` with path `contrib/tree-sitter-parsley`

---

## Recommended Next Steps

1. **Create extension structure** in `contrib/zed-extension/` for local development
2. **Copy highlights.scm** from tree-sitter-parsley
3. **Create basic brackets.scm, outline.scm, indents.scm**
4. **Test locally** using Zed's dev extension feature
5. **Create standalone repository** `parsley-zed` once validated
6. **Submit PR** to `zed-industries/extensions`

---

## Timeline Estimate

| Task | Time | Notes |
|------|------|-------|
| Create extension structure | 30 min | Straightforward directory setup |
| Write config files | 30 min | Minimal configuration needed |
| Create query files | 1 hour | brackets, outline, indents |
| Local testing | 30 min | Install dev extension, test |
| Documentation | 30 min | README with install instructions |
| **Total** | **3 hours** | Could be done in one sitting |
| PR submission | 15 min | Fork, submodule, PR |
| PR review/approval | 1-7 days | Depends on Zed maintainers |

---

## Example Extensions to Reference

Similar language extensions in Zed (for reference):
- **Zig:** `https://github.com/zed-industries/extensions/tree/main/extensions/zig`
- **Gleam:** `https://github.com/zed-industries/extensions/tree/main/extensions/gleam`
- **Odin:** `https://github.com/zed-industries/extensions/tree/main/extensions/odin`

All follow the same simple pattern we'd use for Parsley.

---

## Conclusion

**Creating a Zed extension for Parsley is highly feasible and recommended.** 

We have all the technical assets (Tree-sitter grammar, highlights queries, license). The extension structure is simple and well-documented. Effort is minimal (2-4 hours), and the result would give Parsley excellent editor support in one of the fastest-growing code editors.

**Recommend proceeding with implementation.**