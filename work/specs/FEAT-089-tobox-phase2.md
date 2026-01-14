---
id: FEAT-089
title: "toBox() Phase 2: Style, Title, Width, and Color Options"
status: draft
priority: low
created: 2026-01-14
author: "@human"
related: FEAT-088
---

# FEAT-089: toBox() Phase 2 Options

## Summary
Extend the `toBox()` method with additional formatting options: alternate box styles, titled boxes, width control with truncation, and optional ANSI color output. These are polish features building on the core toBox() implementation from FEAT-088.

## User Story
As a Parsley developer building CLI tools, I want to customize box appearance with different styles, titles, and width limits so that I can create polished, readable terminal output that fits my application's visual design.

## Acceptance Criteria

### Style Option
- [ ] `{style: "single"}` renders with single-line box characters (default, current behavior)
- [ ] `{style: "double"}` renders with double-line box characters (╔═╗║╚╝)
- [ ] `{style: "ascii"}` renders with ASCII-only characters (+|-) for maximum compatibility
- [ ] `{style: "rounded"}` renders with rounded corners (╭╮╰╯)
- [ ] Style option works with all value types (scalar, array, dictionary, table)

### Title Option
- [ ] `{title: "My Data"}` renders a title row at the top of the box
- [ ] Title is centered within the box width
- [ ] Title row uses appropriate separators based on style
- [ ] Works with all toBox variants (vertical, horizontal, grid, key-value)
- [ ] Empty string title `{title: ""}` is ignored (no title row)

### Width Control
- [ ] `{maxWidth: N}` limits column width to N characters
- [ ] Values exceeding maxWidth are truncated with ellipsis (`...`)
- [ ] Truncation preserves box structure (doesn't break borders)
- [ ] Works with all value types
- [ ] `maxWidth` of 0 or negative is ignored (no limit)

### Color Support
- [ ] `{color: true}` enables ANSI color output
- [ ] Type-based coloring: strings (cyan), numbers (yellow), booleans (green/red), null (dim gray)
- [ ] Box borders remain uncolored (default terminal color)
- [ ] Color codes are properly escaped/stripped when output is piped (not a TTY)
- [ ] `{color: false}` (default) produces plain text output

## Design Decisions

### Style Infrastructure Already Exists
The BoxStyle struct and presets (Single, Double, ASCII, Rounded) were created in FEAT-088. This feature exposes them via the options API.

### Title Placement
Title appears in the first row of the box, centered, with a separator line below it. For tables, the title is above the header row.

```
┌─────────────────┐
│   User Data     │
├─────────────────┤
│ name │ Alice    │
├──────┼──────────┤
│ age  │ 30       │
└──────┴──────────┘
```

### Truncation Strategy
Truncation applies per-cell, not per-box. The ellipsis (`...`) counts toward the width limit. Minimum useful maxWidth is 4 (one char + `...`).

```parsley
["hello world", "short"].toBox({maxWidth: 8})
```
```
┌──────────┐
│ hello... │
├──────────┤
│ short    │
└──────────┘
```

### Color Detection
Use `os.Stdout.Fd()` and `term.IsTerminal()` to detect TTY. When not a TTY (piped output), color codes are stripped even if `{color: true}` is set. This prevents garbage in redirected output.

### Color Scheme
Conservative, high-contrast colors that work on both light and dark terminals:
- Strings: Cyan (36)
- Numbers: Yellow (33)
- Booleans: Green (32) for true, Red (31) for false
- Null: Dim/Gray (90)
- Keys: Bold (1)

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/eval_box.go` — Extend BoxRenderer with new options
- `pkg/parsley/evaluator/methods.go` — Parse new options in toBox calls

### Options Schema (Complete)

```parsley
toBox({
    // Existing (from FEAT-088)
    direction: "vertical" | "horizontal" | "grid",
    align: "left" | "right" | "center",
    keys: boolean,
    
    // New (this feature)
    style: "single" | "double" | "ascii" | "rounded",
    title: string,
    maxWidth: number,
    color: boolean
})
```

### Box Style Characters

Already defined in eval_box.go:

| Style | TL | TR | BL | BR | H | V | Cross | T-down | T-up | T-right | T-left |
|-------|----|----|----|----|---|---|-------|--------|------|---------|--------|
| single | ┌ | ┐ | └ | ┘ | ─ | │ | ┼ | ┬ | ┴ | ├ | ┤ |
| double | ╔ | ╗ | ╚ | ╝ | ═ | ║ | ╬ | ╦ | ╩ | ╠ | ╣ |
| rounded | ╭ | ╮ | ╰ | ╯ | ─ | │ | ┼ | ┬ | ┴ | ├ | ┤ |
| ascii | + | + | + | + | - | \| | + | + | + | + | + |

### Title Rendering Logic

```go
// In BoxRenderer
func (br *BoxRenderer) RenderWithTitle(title string, content func() string) string {
    // 1. Render content to get box width
    // 2. If title provided, prepend title row + separator
    // 3. Title centered: padding = (boxWidth - titleLen) / 2
}
```

### Truncation Logic

```go
func truncateToWidth(s string, maxWidth int) string {
    if maxWidth < 4 {
        return s // Too small to truncate meaningfully
    }
    if displayWidth(s) <= maxWidth {
        return s
    }
    // Account for "..." (3 chars)
    return s[:maxWidth-3] + "..."
}
```

Note: Must handle unicode properly—truncate by runes, not bytes.

### ANSI Color Codes

```go
const (
    colorReset  = "\033[0m"
    colorBold   = "\033[1m"
    colorDim    = "\033[2m"
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorCyan   = "\033[36m"
    colorGray   = "\033[90m"
)

func colorize(s string, color string) string {
    return color + s + colorReset
}
```

### TTY Detection

```go
import "golang.org/x/term"

func isColorSupported() bool {
    return term.IsTerminal(int(os.Stdout.Fd()))
}
```

Add `golang.org/x/term` dependency if not already present.

### Implementation Order

1. **Style option** — Simplest, just wire up existing BoxStyle presets
2. **maxWidth truncation** — Independent of other options
3. **Title option** — Requires modifying all render methods
4. **Color support** — Most complex, needs TTY detection and per-type coloring

### Edge Cases

1. **Title longer than content** — Box expands to fit title
2. **Title with unicode** — Use displayWidth() for centering
3. **maxWidth < 4** — Ignore, cannot show meaningful truncated content
4. **Color in nested values** — Don't colorize inline dict/array summaries (too noisy)
5. **Color + redirect** — Strip colors when not TTY

### Dependencies
- Depends on: FEAT-088 (complete)
- Blocks: None
- New dependency: `golang.org/x/term` (if not already present)

### Testing Strategy

Unit tests for each option:
- Style: verify correct box characters used
- Title: verify title row rendered and centered
- maxWidth: verify truncation with ellipsis
- Color: verify ANSI codes present (mock TTY) and absent (non-TTY)

Integration tests:
- Combination of options: `{style: "double", title: "Data", maxWidth: 20, color: true}`
- All value types with each option

## Deferred Ideas

### Per-Column Width Limits
```parsley
table.toBox({maxWidth: [20, 10, 30]})  // Different width per column
```
Could be added later if needed.

### Custom Color Schemes
```parsley
{color: {string: "blue", number: "magenta"}}
```
Overkill for current use cases.

### Syntax Highlighting for Code
Special handling for strings that look like code. Too complex, out of scope.

## Related
- FEAT-088: Universal toBox() Method (Phase 1, complete)
- Backlog items: #61, #62, #63, #64
