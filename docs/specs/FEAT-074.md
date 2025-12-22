---
id: FEAT-074
title: "Dictionary Spreading in HTML Tags"
status: proposed
priority: high
created: 2025-12-22
author: "@copilot"
---

# FEAT-074: Dictionary Spreading in HTML Tags

## Problem

Component props with optional HTML attributes require repetitive conditional logic:

```parsley
<input 
    placeholder={if (placeholder) placeholder else null}
    autocomplete={if (autocomplete) autocomplete else null}
    disabled={if (disabled) "disabled" else null}
    readonly={if (readonly) "readonly" else null}
    minlength={if (minlength) minlength else null}
    maxlength={if (maxlength) maxlength else null}
/>
```

This is verbose, error-prone, and doesn't scale well for components with many optional attributes.

## Proposed Solution

Add dictionary spreading syntax `...attrs` in HTML tags, similar to JSX. Dictionaries are expanded into HTML attributes with smart handling of boolean values and null omission.

### Syntax

```parsley
let attrs = {placeholder: "Enter name", maxlength: 50, disabled: true}
<input type="text" ...attrs/>
```

Renders as:

```html
<input type="text" placeholder="Enter name" maxlength="50" disabled/>
```

### Value Type Handling

| Dictionary Value | HTML Output | Rationale |
|------------------|-------------|-----------|
| `{async: true}` | `async` | Boolean `true` → boolean attribute (no value) |
| `{async: false}` | _(omitted)_ | Boolean `false` → attribute absent |
| `{placeholder: null}` | _(omitted)_ | `null` → attribute absent |
| `{type: "text"}` | `type="text"` | String → quoted value |
| `{maxlength: 50}` | `maxlength="50"` | Number → quoted value |
| `{class: "btn"}` | `class="btn"` | All other types → stringified |

This aligns with HTML semantics where boolean attributes (async, disabled, required, etc.) are present/absent, not true/false.

## Usage Examples

### Basic Spreading

```parsley
let attrs = {
    placeholder: "Enter email",
    type: "email",
    required: true,
    autocomplete: "email"
}

<input ...attrs/>
```

Renders:

```html
<input placeholder="Enter email" type="email" required autocomplete="email"/>
```

### Multiple Spreads (Later Overrides Earlier)

```parsley
let baseAttrs = {type: "text", class: "input"}
let specificAttrs = {class: "input-lg", placeholder: "Name"}

<input ...baseAttrs ...specificAttrs/>
```

Renders:

```html
<input type="text" class="input-lg" placeholder="Name"/>
```

### Spreading with Rest Destructuring

```parsley
export TextField = fn(props) {
    let {name, label, type, value, hint, error, ...inputAttrs} = props
    
    <input 
        type={type ?? "text"}
        name={name}
        value={value ?? ""}
        ...inputAttrs
    />
}

// Usage
<TextField 
    name="email" 
    label="Email"
    placeholder="you@example.com"
    required={true}
    maxlength={100}
/>
```

The component extracts known props (`name`, `label`, etc.) and spreads the rest (`placeholder`, `required`, `maxlength`) directly to the `<input>`.

### Conditional Attributes

```parsley
let optionalAttrs = {
    disabled: user.isBlocked,
    readonly: !user.canEdit,
    placeholder: user.isNew ? "Enter value" : null
}

<input type="text" ...optionalAttrs/>
```

If `user.isBlocked = false`, `user.canEdit = true`, `user.isNew = false`:

```html
<input type="text"/>
```

All attributes with `false` or `null` values are omitted.

### Script/Link Tags with Boolean Attributes

```parsley
let scriptAttrs = {
    async: true,
    defer: false,
    src: "app.js",
    type: "module"
}

<script ...scriptAttrs></script>
```

Renders:

```html
<script async src="app.js" type="module"></script>
```

Note: `defer: false` is omitted entirely.

## Implementation

### 1. Parser Changes

Detect `...identifier` syntax in tag props:

```go
// In parseTagNameAndProps or during tag parsing
// Detect pattern: ...identifier
// Store spread operations alongside raw props string
```

Could extend `TagPairExpression` and `TagLiteral` AST nodes:

```go
type TagPairExpression struct {
    Token    lexer.Token
    Name     string
    Props    string         // Raw props string (existing)
    Spreads  []string       // NEW: identifiers to spread
    Contents []Node
}
```

### 2. Evaluator Changes

In `evalStandardTag` and `evalStandardTagPair`:

```go
// Pseudocode
for each spread identifier:
    1. Evaluate identifier to get dictionary
    2. For each key-value pair:
        - if value is null or false: skip
        - if value is true: write " attrname"
        - else: write " attrname=\"{stringified value}\""
```

Key functions to modify:
- `evalStandardTag` (line ~11405) - singleton tags
- `evalStandardTagPair` (line ~10957) - paired tags  
- May need new helper: `evalDictionarySpread(dict Object) string`

### 3. Order of Operations

Attributes are processed left-to-right:

```parsley
<input class="base" ...attrs class="override"/>
```

1. `class="base"` written
2. Dictionary spreads (may overwrite `class`)
3. `class="override"` written (final value)

Later attributes override earlier ones (last write wins).

## Edge Cases

### Non-Dictionary Spread

```parsley
let notDict = "string"
<input ...notDict/>  // Error: cannot spread non-dictionary
```

Should produce clear error message.

### Reserved Attributes

Some attributes are special:
- `contents` - reserved for tag body
- Potentially others?

Decision: Allow all dictionary keys as attributes. No reservations.

### Numeric Keys

```parsley
let attrs = {0: "value", data-id: 123}
<div ...attrs/>
```

Numeric keys: `0="value"` (valid HTML5)  
Hyphenated keys: `data-id="123"` (valid)

### Duplicate Spreads

```parsley
let attrs1 = {class: "foo"}
let attrs2 = {class: "bar"}
<div ...attrs1 ...attrs2/>  // class="bar" (last wins)
```

## Testing

### Unit Tests

```parsley
// Basic spread
let attrs = {id: "test", class: "box"}
<div ...attrs/> // "<div id=\"test\" class=\"box\" />"

// Boolean true
let attrs = {disabled: true}
<input ...attrs/> // "<input disabled />"

// Boolean false (omitted)
let attrs = {disabled: false}
<input ...attrs/> // "<input />"

// Null (omitted)
let attrs = {placeholder: null}
<input ...attrs/> // "<input />"

// Multiple spreads
let a = {id: "foo"}
let b = {class: "bar"}
<div ...a ...b/> // "<div id=\"foo\" class=\"bar\" />"

// Override
let base = {class: "base"}
let override = {class: "new"}
<div ...base ...override/> // "<div class=\"new\" />"

// Mixed with regular attrs
<div id="x" ...attrs class="y"/> // Later attrs override spread
```

### Integration Tests

Test with actual components:

```parsley
export TextField = fn(props) {
    let {name, label, ...inputAttrs} = props
    <input name={name} ...inputAttrs/>
}

<TextField name="email" placeholder="Email" required={true}/>
```

Expected: `<input name="email" placeholder="Email" required/>`

## Documentation Updates

1. **CHEATSHEET.md** - Add section on spreading:
   ```parsley
   // Spread dictionary into tag attributes
   let attrs = {disabled: true, maxlength: 50}
   <input type="text" ...attrs/>
   ```

2. **reference.md** - Document spreading syntax and value type handling

3. **Component examples** - Update TextField, Button, Form to use spreading

## Migration Path

This is additive - no breaking changes. Existing code continues to work.

Components can be gradually refactored:

```parsley
// Before
<input 
    placeholder={if (placeholder) placeholder else null}
    disabled={if (disabled) "disabled" else null}
/>

// After
<input ...{placeholder, disabled}/>
```

Or with destructuring:

```parsley
// Before
fn({placeholder, disabled, readonly, ...}) {
    <input 
        placeholder={if (placeholder) placeholder else null}
        disabled={if (disabled) "disabled" else null}
        readonly={if (readonly) "readonly" else null}
    />
}

// After
fn({name, label, ...inputAttrs}) {
    <input name={name} ...inputAttrs/>
}
```

## Benefits

1. **Less Boilerplate** - No repetitive `if (x) x else null` patterns
2. **Better Composition** - Easy to merge attribute sets
3. **Type Safety** - Boolean attributes work as expected
4. **Familiar** - Matches JSX/React patterns
5. **Flexible** - Supports multiple spreads, overrides, and mixing with regular attributes

## Related Features

- Complements FEAT-073 (HTML Components) by making components more ergonomic
- Builds on the existing `attr={null}` omission fix (commit 15c4f10)
- Works with destructuring rest patterns (`...rest`)

## Open Questions

1. **Syntax preference**: `...attrs` vs `{attrs}` vs other?
   - Recommendation: `...attrs` (matches JSX, clear intent)

2. **String values for boolean attrs**: Should `{disabled: "disabled"}` render as `disabled="disabled"` or just `disabled`?
   - Recommendation: `disabled="disabled"` (preserve string)

3. **Error handling**: Spread non-dictionary - error or silent ignore?
   - Recommendation: Error with clear message

## Implementation Priority

**High** - This significantly improves component authoring ergonomics and is a natural extension of the existing null attribute omission feature.

Estimated effort: 2-3 days
- Parser: 0.5 day
- Evaluator: 1 day  
- Tests: 0.5 day
- Documentation: 0.5 day
- Component refactoring: 0.5 day
