# Parsley Language Reference

> This document is composed from hand-written fragments and auto-generated API tables.
> Regenerate with: `pars reference --template docs/parsley/reference.tmpl.md`

{{template "toc"}}

---

{{include "01-literals.md"}}

---

{{include "02-operators.md"}}

---

{{include "03-control-flow.md"}}

---

{{include "04-statements.md"}}

---

## 5. Type Methods

Methods are called on values using dot notation: `value.method(args)`.

**Return Value Convention**: Most methods return a new value and do not modify the original. You must assign the result to use it. Exception: `delete()` on dictionaries mutates in place.

```parsley
let name = "alice"
name.toUpper()                  // Returns "ALICE", but name is still "alice"
let upper = name.toUpper()      // Assign to use the result
```

{{include "05-type-methods-intro.md"}}

{{generate "type-methods"}}

---

## 6. Builtin Functions

{{include "06-builtins-intro.md"}}

{{generate "builtins"}}

---

## 7. Standard Library

{{include "07-stdlib-intro.md"}}

{{generate "modules"}}

---

{{include "08-tags.md"}}

---

{{include "09-comments.md"}}

---

{{include "10-error-handling.md"}}

---

## Reserved Keywords

{{generate "keywords"}}

---

## Appendix A: Type Summary

{{generate "type-summary"}}

---

## Appendix B: Method Reference

> For the authoritative, up-to-date method list, run `pars describe <type>`.

{{generate "method-reference"}}