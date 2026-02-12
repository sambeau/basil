---
id: FEAT-109-REMAINING-WORK
title: "Tree-sitter Grammar & Editor Extensions — Remaining Work"
date: 2026-02-14
feature: FEAT-109
status: post-alpha
---

# FEAT-109: Remaining Work (Post-Alpha)

## Summary

The Tree-sitter grammar for Parsley is complete and working locally. The Zed dev extension provides syntax highlighting, bracket matching, code outline, auto-indentation, and language injection for CSS/JS in `<style>`/`<script>` tags. The grammar covers all Parsley syntax with 253/253 corpus tests passing and 96% of example files parsing cleanly.

What remains is **publishing and validation** — getting the extension into editor registries and confirming it works in other editors. None of this blocks Alpha.

---

## Remaining Tasks

### 1. Submit Zed Extension to Registry

**Effort:** 30–60 min · **Type:** Human task · **Priority:** High (highest user-facing value)

The extension works locally via Zed's "Install Dev Extension" feature. To make it available to all Zed users:

1. Create standalone repository `github.com/sambeau/parsley-zed`
2. Copy contents of `contrib/zed-extension/` to the new repo
3. Verify `extension.toml` grammar reference points to `https://github.com/sambeau/tree-sitter-parsley`
4. Fork `zed-industries/extensions`
5. Add submodule: `git submodule add https://github.com/sambeau/parsley-zed.git extensions/parsley`
6. Add entry to `extensions.toml`:
   ```toml
   [parsley]
   submodule = "extensions/parsley"
   version = "0.1.0"
   ```
7. Run `pnpm sort-extensions`
8. Submit PR

**Reference:** PLAN-090 Tasks 9–10. Similar extensions to study: Zig, Gleam, Odin in `zed-industries/extensions`.

**Depends on:** The `tree-sitter-parsley` standalone repo must have the latest generated parser files committed (parser.c, grammar.json, node-types.json, scanner.c). The Zed build fetches these from the grammar repo.

---

### 2. Submit to Tree-sitter Grammar Registry

**Effort:** 15 min · **Type:** Human task · **Priority:** Medium

Add Parsley to the [tree-sitter wiki list of parsers](https://github.com/tree-sitter/tree-sitter/wiki/List-of-parsers). This is how editors and tools discover available grammars.

Entry format:
```
| [Parsley](https://github.com/sambeau/tree-sitter-parsley) | `.pars` | Syntax highlighting |
```

---

### 3. Validate in Neovim

**Effort:** 30 min · **Type:** Validation · **Priority:** Low

Test that the grammar works with nvim-treesitter. Configuration:

```lua
local parser_config = require("nvim-treesitter.parsers").get_parser_configs()
parser_config.parsley = {
  install_info = {
    url = "https://github.com/sambeau/tree-sitter-parsley",
    files = {"src/parser.c", "src/scanner.c"},
  },
  filetype = "pars",
}
```

Copy `contrib/tree-sitter-parsley/queries/highlights.scm` to Neovim's queries directory for Parsley. Open a `.pars` file and verify highlighting.

---

### 4. Validate in Helix

**Effort:** 30 min · **Type:** Validation · **Priority:** Low

Test that the grammar works in Helix. Add to `languages.toml`:

```toml
[[language]]
name = "parsley"
scope = "source.parsley"
file-types = ["pars", "part"]
comment-tokens = ["//"]
indent = { tab-width = 2, unit = "  " }
roots = []

[[grammar]]
name = "parsley"
source = { git = "https://github.com/sambeau/tree-sitter-parsley", rev = "main" }
```

Run `hx --grammar fetch && hx --grammar build`, copy highlights.scm to Helix's runtime queries directory, and verify.

---

### 5. GitHub Linguist Merge

**Effort:** None (waiting) · **Type:** External dependency · **Priority:** Medium

A PR has been submitted to `github/linguist` for `.pars` file recognition. Once merged, `.pars` files on GitHub will be:

- Recognized as Parsley in repository language stats
- Syntax highlighted using the Tree-sitter grammar

Nothing to do here except wait for the linguist maintainers to review.

---

## Known Limitations (Backlog)

These are grammar edge cases that affect 5 of 125 example files. They are tracked in `work/BACKLOG.md` and do not block any user-facing functionality.

### Backlog #98: Complex Regex Escape Handling

**Affects:** 2 example files with URL-matching regexes

Regex literals containing escaped forward slashes (e.g., `/^https?:\/\/.+/`) fail to parse because the token-level regex pattern stops at the first unescaped `/`. The lexer correctly handles this in the Go parser, but the Tree-sitter grammar's token rule cannot express the escape logic.

**Fix:** Extend the external scanner (`src/scanner.c`) to handle regex tokenization, tracking `\` escape sequences to skip over `\/`. This is the same approach used for raw text in `<style>`/`<script>` tags.

**Estimated effort:** 2–4 hours

---

### Backlog #99: XML Comment Support

**Affects:** 1 example file (svg_xml_demo.pars)

`<!-- ... -->` inside tag content is not parsed. The grammar has no rule for XML/HTML comments.

**Fix:** Add a grammar rule for XML comments or handle them in the external scanner. Straightforward but low priority since XML comments are rare in Parsley code.

**Estimated effort:** 1 hour

---

### Backlog #100: HTML-like Tags Inside Strings

**Affects:** 2 example files

When `<tag>` appears inside a quoted string (e.g., `"use <code> for inline code"`), Tree-sitter may interpret `<code>` as a tag start rather than string content. This is an inherent ambiguity in the grammar since Parsley allows tags as expressions.

**Fix:** Would require complex lookahead in the external scanner or a redesign of string content rules. Unlikely to be worth the complexity for this edge case.

**Estimated effort:** 4–8 hours (and may not be fully solvable)

---

## File Locations

| Asset | Path |
|-------|------|
| Tree-sitter grammar (dev) | `contrib/tree-sitter-parsley/` |
| Zed extension (dev) | `contrib/zed-extension/` |
| Grammar in Zed extension | `contrib/zed-extension/grammars/parsley/` |
| Standalone grammar repo | `github.com/sambeau/tree-sitter-parsley` |
| Feature spec | `work/specs/FEAT-109.md` |
| Zed investigation | `work/specs/FEAT-109-ZED-INVESTIGATION.md` |
| Zed extension plan | `work/plans/PLAN-090.md` |
| Phase 2 plan | `work/plans/PLAN-092-TREE-SITTER-PHASE-2.md` |

## Current Stats

- **Corpus tests:** 253/253 passing
- **Example file parsing:** 120/125 (96%)
- **Highlight captures:** 40+ capture rules covering keywords, literals, operators, tags, Query DSL, schemas, mutations
- **External scanner:** Handles raw text in `<style>`/`<script>`, `@{}` interpolation
- **Zed extension features:** Syntax highlighting, bracket matching, code outline, auto-indentation, CSS/JS language injection