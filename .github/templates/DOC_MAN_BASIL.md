<!-- Style guide: .github/instructions/docs.instructions.md -->

---
id: man-bas-(feature-name) e.g. (man-bas-images)
title: "Manual Page Title" e.g. "Images"
system: basil
type: builtin | feature | tutorial e.g. (builtin)
name: name of feature in code (e.g. images)
created: YYYY-MM-DD
version: xx.xx.xx (version of basil when this was generated - without git id)
author: "@sam"
keywords: a, few, keywords
---

<!--
  Basil server manual pages live in: docs/basil/manual/
  These document features that require the Basil server environment
  and will error or no-op in standalone `pars` or the REPL.

  For Parsley language builtins (types like array, string, etc.),
  use DOC_MAN_BUILTIN.md instead → docs/parsley/manual/builtins/

  For Parsley stdlib modules (@std/math, etc.),
  use DOC_MAN_STD.md instead → docs/parsley/manual/stdlib/
-->

# Manual Page Title (e.g. "Images")

One paragraph overview: what this feature does, why you'd use it, and that it's Basil-only.

> **Basil-only.** This feature requires the Basil server environment. It will error in `pars` or the REPL.

```parsley
A small, tested, piece of code showing the most typical usage.
```

---

## function_name()

```parsley
function_name(args) → return_type
function_name(args, options) → return_type
```

One paragraph describing what this function does and when to use it.

**Options:**

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `option` | type | default | What it does |

```parsley
// Minimal example
let result = function_name(@./file.ext)

// Example with options
let result = function_name(@./file.ext, {option: "value"})
```

**Good to know:**

- Bullet points for non-obvious behaviour, caveats, defaults.
- Mention caching, performance, or security implications if relevant.

**Errors:**

| Condition | Error class |
|-----------|-------------|
| Describe what went wrong | `class` |

---

## Configuration

```yaml
feature:
  setting: value        # What this controls
```

Brief note on defaults (e.g. "Zero-config defaults work for most projects.").

---

## Security

Describe any security boundaries: path restrictions, access controls, input validation.

---

## See Also

- [Related Page](relative/path.md) — brief description