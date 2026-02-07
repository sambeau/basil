---
applyTo: "docs/**/*.md"
---
# Documentation Style Guide

These rules apply when writing or editing Parsley and Basil manual pages.

## Templates

- **Builtin types:** Follow structure in `.github/templates/DOC_MAN_BUILTIN.md`
- **Stdlib modules:** Follow structure in `.github/templates/DOC_MAN_STD.md`
- **Fundamentals/features:** Adapt the builtin template — omit sections that don't apply

Templates define **page structure**. This file defines **voice and style**.

## Voice & Density

Write for experienced programmers learning a new language. They already know what booleans, arrays, and functions are — they need to know how Parsley's versions **differ**.

- **Be concise.** One clear sentence beats three cautious ones.
- **Lead with what's different.** If a feature works exactly like every other language, say so briefly and move on. Spend space on the parts that are surprising, unusual, or easy to get wrong.
- **Use callout boxes (> ⚠️) for genuine gotchas** — things that will bite someone coming from JavaScript, Python, or Go. Don't use them for standard behaviour.

## Examples

- **Amalgamate small examples.** If several cases are tiny (e.g. `!true`, `!false`, `!null`), combine them into one code block with inline comments rather than giving each its own heading and `**Result:**` block:
  ```parsley
  !true               // false
  !null               // true
  not ""              // true
  ```
- **Use separate blocks for distinct concepts.** When an example needs setup code, has a multi-line result, or demonstrates a genuinely different idea, give it its own block.
- **Inline results with `//` comments** for short, obvious outputs. Use `**Result:**` blocks only when the output is multi-line, complex, or needs explanation.
- **Every example must be valid Parsley.** Test with the `pars` CLI before committing.

## Structure

- **Omit sections with nothing to say.** If a type has no operators, don't include an empty Operators heading.
- **Use tables for lists of small items** (methods, attributes) rather than giving each one its own H3 with a code block — unless the item needs extended explanation.
- **"Key Differences from Other Languages"** belongs near the bottom. Keep it as tight bullet points.
- **"See Also"** is the final section on every page. Link to related manual pages even if they don't exist yet.

## Tone

- Second person ("you can", "use this when") is fine but not required.
- Don't hedge excessively ("it should be noted that perhaps..."). Be direct.
- British or American spelling are both fine — just be consistent within a page.