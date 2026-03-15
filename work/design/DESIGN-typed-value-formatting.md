# Design: Typed Value Formatting and Form Field Abstraction Layers

**Status:** Approved  
**Date:** 2025-06-15  
**Related:** STANDARD-PRELUDE-REVIEW.md §9.3, §9.5  
**Implements:** Prelude Review Decisions #2 and #3

---

## 1. Overview

This document covers two related areas of Parsley's handling of typed values:

1. **Default formatting for typed values** — Change `objectToString()` and `objectToPrintString()` to produce human-readable output for `money`, `datetime`, `unit`, and `duration` types.

2. **Form field abstraction layers** — A layered approach to form binding, from most terse (`<field/>`) to most flexible (manual props), with `fieldProps()` as a bridge for component library authors.

### 1.1 Design Goals

- **Human-readable by default** — Typed values should render nicely without explicit formatting calls
- **Progressive disclosure** — Simple forms should be simple; complex forms remain possible
- **Layers of abstraction** — Each layer serves a different use case without overlap
- **Preserve existing power** — The `@field` system already works well; enhance, don't replace

### 1.2 Decision Context

These decisions were made during the Standard Prelude Review (2025-06). The goal is "Parsley templates should produce human-readable output by default with zero effort." Since Basil is pre-1.0, backward compatibility is not a constraint.

---

## 2. Part A: Typed Value Formatting

### 2.1 Current Behaviour

When typed values appear in string contexts (tag content, table cells, `.toHTML()` output), they pass through `objectToString()` or `objectToPrintString()`, which fall through to `Inspect()` or raw string conversion:

| Type | Current Output | Context |
|------|---------------|---------|
| `money(499900, "GBP")` | `£4999.00` | No thousands separator |
| `date("2025-03-15")` | `2025-03-15` | ISO format |
| `datetime("2025-03-15T14:30:00")` | `2025-03-15T14:30:00` | ISO format |
| `unit(5, "kg")` | `5kg` | Short format |
| `duration(9000)` | `9000` | Raw seconds |

### 2.2 New Behaviour

Both `objectToString()` and `objectToPrintString()` will call `.medium()` on typed values:

| Type | New Output | Format Method |
|------|-----------|---------------|
| `money(499900, "GBP")` | `£4,999.00` | `.medium()` |
| `date("2025-03-15")` | `Mar 15, 2025` | `.medium()` |
| `datetime("2025-03-15T14:30:00")` | `Mar 15, 2025, 2:30 PM` | `.medium()` |
| `unit(5, "kg")` | `5.00 kg` | `.medium()` |
| `duration(9000)` | `2 hours 30 minutes` | `.medium()` |

### 2.3 Affected Contexts

This change affects all string coercion contexts:

1. **Tag content interpolation:** `<td>(price)</td>` → now renders `£4,999.00`
2. **Table `.toHTML()` cells:** Typed values in table cells format nicely
3. **Array `.toHTML()` items:** List items with typed values format nicely
4. **Dictionary `.toHTML()` values:** Definition list values format nicely
5. **String concatenation:** `"Price: " + price` → `Price: £4,999.00`

### 2.4 Accessing Raw/ISO Formats

Users who need the previous behaviour have clear alternatives:

| Type | Raw/ISO Access |
|------|----------------|
| `datetime` | `.iso` property: `<time datetime={post.date.iso}>` |
| `money` | `.inspect()` or manual: `"£" + (price.amount / 100)` |
| `unit` | `.short()` method: `weight.short()` → `5kg` |
| `duration` | `.short()` method: `dur.short()` → `2h 30m` |

### 2.5 Implementation

**File:** `pkg/parsley/evaluator/eval_string_conversions.go`

```go
// objectToString converts an Object to its string representation for template output
func objectToString(obj Object) string {
    switch v := obj.(type) {
    case *Money:
        // Use .medium() for human-readable output
        result := moneyMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Datetime:
        // Use .medium() for human-readable output
        result := datetimeMedium(v, nil, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Unit:
        // Use .medium() for human-readable output
        result := unitMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    case *Duration:
        // Use .medium() for human-readable output
        result := durationMedium(v, nil)
        if str, ok := result.(*String); ok {
            return str.Value
        }
        return v.Inspect()
    // ... existing cases unchanged
    }
}
```

The same pattern applies to `objectToPrintString()`.

### 2.6 Testing

Add tests to verify:

1. Each typed value formats correctly in tag content
2. Each typed value formats correctly in `.toHTML()` output
3. `.iso` and `.short()` alternatives still work
4. Locale parameter (if present) is respected

---

## 3. Part B: Form Field Abstraction Layers

### 3.1 The Four Layers

Parsley provides four levels of abstraction for form fields, each serving different needs:

```
Level 4: <field name="email"/>                    ← Most terse, least flexible
         Outputs complete field structure automatically

Level 3: <form @record={user}>                    ← Current system (exists today)
           <label @field="email"/>
           <input @field="email"/>
           <error @field="email"/>
         </form>
         Full control over ordering/structure, schema-aware

Level 2: <TextField ...user.fieldProps("email")/>  ← Component library bridge
         Prelude/custom components with schema-derived props

Level 1: <TextField name="email" type="email"      ← Fully manual
           label="Email" value={user.email}
           error={user.error("email")} required/>
         Complete control, no schema magic
```

### 3.2 When to Use Each Layer

| Level | Use Case | Audience |
|-------|----------|----------|
| **4** `<field/>` | Rapid prototyping, admin panels, standard forms | App developers who want speed |
| **3** `@field` | Custom layouts (checkbox before label, inline errors, help text) | App developers who need control |
| **2** `fieldProps()` | Building component libraries, integrating design systems | Library authors, design system implementors |
| **1** Manual | Edge cases, non-schema data, maximum control | Everyone occasionally |

### 3.3 Key Principle: No Overlap

Each layer has a distinct purpose. There is no `<Field>...</Field>` wrapper component because it would add nothing over a plain `<div class="field">` — the value is in `<field/>` (outputs everything) or raw `@field` attributes (full control), not a hybrid.

---

## 4. Level 4: The `<field/>` Tag

### 4.1 Purpose

A new evaluator-handled tag that outputs a complete, accessible field structure in one line. Ideal for standard forms where the default label→input→error ordering is acceptable.

### 4.2 Basic Usage

```parsley
<form @record={user} method="POST" action="/users">
    <field name="name"/>
    <field name="email"/>
    <field name="role"/>
    <button type="submit">"Save"</button>
</form>
```

### 4.3 Output Structure

`<field name="email"/>` outputs:

```html
<div class="field">
    <label for="email">Email</label>
    <input type="email" name="email" id="email" value="..." 
           required aria-required="true" aria-invalid="false" autocomplete="email"/>
    <span id="email-error" class="error" role="alert">Error message here</span>
</div>
```

The error span is only rendered if the field has an error.

### 4.4 Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `name` | string | required | Field name (must exist in schema) |
| `as` | string | auto | Override input type: `"textarea"`, `"select"`, `"checkbox"`, etc. |
| `class` | string | `"field"` | Wrapper div class |
| `id` | string | field name | Override input id |
| `label` | string | from schema | Override label text |
| `placeholder` | string | from schema | Override placeholder |
| `help` | string | from schema | Help text (rendered after input, before error) |

### 4.5 Examples

```parsley
// Basic field
<field name="email"/>

// Override input type
<field name="bio" as="textarea"/>

// Override label
<field name="email" label="Work Email"/>

// Add custom class
<field name="email" class="field field--wide"/>

// With help text (from schema metadata or prop)
<field name="password" help="Must be at least 8 characters"/>
```

### 4.6 Type Derivation

The `<field/>` tag uses the same type derivation as `@field`:

| Schema Type | Rendered As |
|-------------|-------------|
| `string` | `<input type="text">` |
| `email` | `<input type="email">` |
| `url` | `<input type="url">` |
| `phone` | `<input type="tel">` |
| `integer` | `<input type="number">` |
| `date` | `<input type="date">` |
| `datetime` | `<input type="datetime-local">` |
| `bool` | `<input type="checkbox">` (with label after) |
| `enum[...]` | `<select>` with options |
| `string` + `as="textarea"` | `<textarea>` |

### 4.7 Checkbox/Boolean Special Case

For boolean fields, the ordering changes to input→label (checkbox convention):

```parsley
<field name="active"/>
```

Outputs:

```html
<div class="field field--checkbox">
    <input type="checkbox" name="active" id="active" checked/>
    <label for="active">Active</label>
    <span id="active-error" class="error" role="alert">...</span>
</div>
```

### 4.8 Implementation

**File:** `pkg/parsley/evaluator/form_field_tag.go`

The `<field/>` tag is handled by the evaluator (like `<label @field>`, `<error @field>`, etc.), not as a Parsley component. It:

1. Requires `@record` context (error if not inside `<form @record={...}>`)
2. Looks up the field in the schema
3. Generates the complete HTML structure
4. Uses the same attribute-building logic as `buildInputAttributes()`

```go
// evalFieldTag handles <field name="..."/> 
func evalFieldTag(node *ast.TagLiteral, propsStr string, env *Environment) Object {
    // Get field name from name attribute
    fieldName := parseAttrValue(propsStr, "name")
    if fieldName == "" {
        return &Error{
            Code:    "FORM-0010",
            Message: "<field> requires name attribute",
        }
    }
    
    // Get form context
    formCtx := getFormContext(env)
    if formCtx == nil {
        return &Error{
            Code:    "FORM-0002", 
            Message: "<field> must be inside a <form @record={...}> context",
        }
    }
    
    record := formCtx.Record
    field := record.Schema.Fields[fieldName]
    
    // Build complete field structure
    var result strings.Builder
    
    // Wrapper div
    wrapperClass := parseAttrValue(propsStr, "class")
    if wrapperClass == "" {
        wrapperClass = "field"
    }
    if field.Type == "bool" {
        wrapperClass += " field--checkbox"
    }
    fmt.Fprintf(&result, `<div class="%s">`, wrapperClass)
    
    // For checkbox: input before label
    if field.Type == "bool" {
        result.WriteString(buildCheckboxInput(record, fieldName, propsStr))
        result.WriteString(buildLabel(record, fieldName, propsStr))
    } else {
        result.WriteString(buildLabel(record, fieldName, propsStr))
        result.WriteString(buildInput(record, fieldName, propsStr))
    }
    
    // Help text (if present)
    helpText := getHelpText(field, propsStr)
    if helpText != "" {
        fmt.Fprintf(&result, `<span class="help">%s</span>`, escapeHTMLText(helpText))
    }
    
    // Error (if present)
    if record.Errors != nil && record.Errors[fieldName] != nil {
        msg := record.Errors[fieldName].Message
        if msg != "" {
            fmt.Fprintf(&result, `<span id="%s-error" class="error" role="alert">%s</span>`,
                fieldName, escapeHTMLText(msg))
        }
    }
    
    result.WriteString("</div>")
    return &String{Value: result.String()}
}
```

---

## 5. Level 3: The `@field` Attribute System (Existing)

### 5.1 Current Capabilities

The `@field` system already provides full schema-aware form binding:

**For `<input @field="name"/>`:**
- `name`, `value`, `type` (derived), `required`
- `aria-required`, `aria-invalid`, `aria-describedby`
- `minlength`/`maxlength`, `min`/`max`, `pattern`
- `placeholder`, `autocomplete`

**For `<label @field="name"/>`:**
- `for` attribute, label text from schema title

**For `<error @field="name"/>`:**
- `id="{field}-error"`, `class="error"`, `role="alert"`
- Error message text (or nothing if no error)

**For `<select @field="name"/>`:**
- All input attributes plus auto-populated `<option>` elements from enum

### 5.2 When to Use Level 3

Use `@field` when you need:

- **Custom ordering:** Checkbox label after input, error before input, etc.
- **Custom structure:** Extra wrapper divs, help text placement, icons
- **Conditional rendering:** Show/hide parts of the field based on state
- **Mixed content:** Combine schema fields with non-schema content

```parsley
// Custom checkbox layout
<div class="field field--checkbox">
    <input type="checkbox" @field="terms"/>
    <label @field="terms">
        "I agree to the "
        <a href="/terms">"terms and conditions"</a>
    </label>
    <error @field="terms"/>
</div>

// Field with help text and icon
<div class="field">
    <label @field="email"/>
    <div class="input-wrapper">
        <Icon name="mail"/>
        <input @field="email"/>
    </div>
    <span class="help">"We'll never share your email"</span>
    <error @field="email"/>
</div>
```

### 5.3 No Changes Required

Level 3 is complete and working. This design adds Level 4 above it and Level 2 below it, but Level 3 remains unchanged.

---

## 6. Level 2: The `fieldProps()` Method

### 6.1 Purpose

The `record.fieldProps(name, options?)` method is a bridge for **component library authors**. It extracts schema metadata into a props dictionary that can be spread into custom components.

This is most valuable *inside* component implementations, not at the call site.

### 6.2 Primary Use Case: Component Libraries

```parsley
// Inside a component library's TextField implementation
export TextField = fn({record, field, ...overrides}) {
    // If record+field provided, derive props from schema
    let props = if (record && field) {
        record.fieldProps(field) ++ overrides
    } else {
        overrides  // Manual mode
    }
    
    <div class={"field" ++ (props.error && " field--error")}>
        <label for={props.name}>{props.label}</label>
        <input 
            type={props.type ?? "text"} 
            name={props.name}
            value={props.value}
            required={props.required}
            placeholder={props.placeholder}
            aria-invalid={props.error ? "true" : "false"}
            aria-describedby={props.error && props.name ++ "-error"}
        />
        if (props.error) {
            <span id={props.name ++ "-error"} class="error" role="alert">
                {props.error}
            </span>
        }
    </div>
}

// User code becomes clean
<TextField record={user} field="email"/>
<TextField record={user} field="email" label="Work Email"/>  // Override
```

### 6.3 API

```parsley
// Basic usage
let props = user.fieldProps("email")
// → {name: "email", type: "email", label: "Email", value: "sam@example.com", 
//    required: true, autocomplete: "email"}

// With overrides
let props = user.fieldProps("email", {label: "Work Email", class: "wide"})
// Overrides merge in, user values win
```

### 6.4 Return Value

| Key | Type | Source | Description |
|-----|------|--------|-------------|
| `name` | string | field name | HTML `name` attribute |
| `type` | string | schema type → mapping | HTML `type` or `"select"` for enums |
| `label` | string | `.title()` or field name | Display label |
| `placeholder` | string | `.placeholder()` | Input placeholder |
| `value` | any | record data | Current value (formatted for input) |
| `required` | boolean | schema constraint | Whether required |
| `error` | string | `.error()` | Error message if present |
| `autocomplete` | string | type/metadata | HTML `autocomplete` hint |
| `inputmode` | string | type/metadata | HTML `inputmode` hint |
| `pattern` | string | metadata | HTML `pattern` attribute |
| `min`, `max`, `step` | number | metadata | Numeric constraints |
| `options` | array | `.enumValues()` | For enums, the allowed values |

### 6.5 Type Mappings

Same mappings as Level 4 and Level 3:

| Schema Type | `type` | `inputmode` | `autocomplete` |
|-------------|--------|-------------|----------------|
| `string` | `text` | — | — |
| `email` | `email` | `email` | `email` |
| `url` | `url` | `url` | `url` |
| `phone` | `tel` | `tel` | `tel` |
| `integer` | `number` | `numeric` | — |
| `number`/`float` | `text` | `decimal` | — |
| `boolean` | `checkbox` | — | — |
| `money` | `text` | `decimal` | — |
| `date` | `date` | — | — |
| `datetime` | `datetime-local` | — | — |
| `unit` | `text` | `decimal` | — |
| `enum(...)` | `select` | — | — |

### 6.6 Value Formatting for Inputs

Values are formatted for HTML input elements, not display:

| Type | Formatting |
|------|------------|
| `money` | Decimal string: `"49.99"` not `4999` |
| `date` | ISO: `"2025-03-15"` |
| `datetime` | ISO local: `"2025-03-15T14:30"` |
| `unit` | Numeric value only |
| Others | As-is |

### 6.7 Implementation

**File:** `pkg/parsley/evaluator/methods_record.go`

```go
"fieldProps": {
    Fn:          recordMethodFieldProps,
    Arity:       "1-2",
    Description: "Get form field props for a field (field, options?)",
},
```

```go
func recordFieldProps(record *Record, args []Object, env *Environment) Object {
    if len(args) < 1 || len(args) > 2 {
        return newArityError("fieldProps", len(args), "1-2")
    }
    
    fieldName, ok := args[0].(*String)
    if !ok {
        return newTypeError("TYPE-0001", "Record.fieldProps", "string", args[0].Type())
    }
    
    result := make(map[string]Object)
    result["name"] = fieldName
    
    if record.Schema != nil {
        if field, exists := record.Schema.Fields[fieldName.Value]; exists {
            applyTypeMapping(result, field)
            applyMetadataOverrides(result, field)
            result["required"] = &Boolean{Value: field.Required}
            
            if len(field.EnumValues) > 0 {
                result["type"] = &String{Value: "select"}
                opts := make([]Object, len(field.EnumValues))
                for i, v := range field.EnumValues {
                    opts[i] = &String{Value: v}
                }
                result["options"] = &Array{Elements: opts}
            }
        }
    }
    
    // Label from title() or titlecased field name
    if title := recordTitle(record, []Object{fieldName}, env); title != NULL {
        result["label"] = title
    } else {
        result["label"] = &String{Value: titleCase(fieldName.Value)}
    }
    
    // Placeholder
    if ph := recordPlaceholder(record, []Object{fieldName}, env); ph != NULL {
        result["placeholder"] = ph
    }
    
    // Value (formatted for input)
    if value := getFieldValueForInput(record, fieldName.Value, env); value != NULL {
        result["value"] = value
    }
    
    // Error
    if err := recordError(record, []Object{fieldName}, env); err != NULL {
        result["error"] = err
    }
    
    // Merge user overrides (second argument wins)
    if len(args) == 2 {
        if opts, ok := args[1].(*Dictionary); ok {
            for k, v := range opts.Pairs {
                result[k] = v
            }
        }
    }
    
    return &Dictionary{Pairs: result}
}
```

---

## 7. Level 1: Manual Props (Existing)

### 7.1 When to Use

- Non-schema data (search forms, filters, one-off inputs)
- Maximum control over every attribute
- Edge cases where schema doesn't fit
- Learning/understanding what the higher levels do

```parsley
// No schema, fully manual
<div class="field">
    <label for="search">"Search"</label>
    <input type="search" name="q" id="search" 
           placeholder="Search products..." autocomplete="off"/>
</div>
```

### 7.2 No Changes Required

Level 1 is just normal Parsley/HTML. No special handling.

---

## 8. Implementation Order

### Phase 1: Typed Value Formatting (2-3 hours)

1. Modify `objectToString()` in `eval_string_conversions.go`
2. Modify `objectToPrintString()`
3. Add tests for string coercion contexts
4. Update documentation showing raw output examples

### Phase 2: `fieldProps()` Method (3-4 hours)

1. Implement `recordFieldProps()` in `methods_record.go`
2. Add type mapping helper functions
3. Register method
4. Add comprehensive tests
5. Update `pars describe record`

### Phase 3: `<field/>` Tag (3-4 hours)

1. Create `form_field_tag.go`
2. Implement `evalFieldTag()`
3. Wire into tag evaluation in `eval_tags.go`
4. Add tests for all field types
5. Add tests for props (as, class, label, etc.)
6. Add tests for checkbox special case

### Phase 4: Documentation (1-2 hours)

1. Update form binding documentation with all four levels
2. Add migration guide (when to use which level)
3. Update prelude component docs for `fieldProps()` pattern

**Total: 9-13 hours**

---

## 9. Testing Strategy

### 9.1 Typed Value Formatting Tests

```go
func TestTypedValueFormatting(t *testing.T) {
    tests := []struct{
        input    string
        expected string
    }{
        {`<span>{money(499900, "GBP")}</span>`, `<span>£4,999.00</span>`},
        {`<span>{date("2025-03-15")}</span>`, `<span>Mar 15, 2025</span>`},
        {`<span>{unit(5, "kg")}</span>`, `<span>5.00 kg</span>`},
    }
    // ...
}
```

### 9.2 `<field/>` Tag Tests

```go
func TestFieldTag(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        contains []string
    }{
        {
            name: "basic text field",
            input: `
                @schema User { name: string | {title: "Full Name"} }
                let user = User({name: "Alice"})
                <form @record={user}><field name="name"/></form>
            `,
            contains: []string{
                `<div class="field">`,
                `<label for="name">Full Name</label>`,
                `<input type="text" name="name"`,
                `value="Alice"`,
            },
        },
        {
            name: "checkbox field has input before label",
            input: `
                @schema User { active: bool }
                let user = User({active: true})
                <form @record={user}><field name="active"/></form>
            `,
            contains: []string{
                `class="field field--checkbox"`,
                `<input type="checkbox"`,  // Should come before label
            },
        },
        // ... more tests
    }
}
```

### 9.3 `fieldProps()` Tests

```go
func TestRecordFieldProps(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        expected map[string]interface{}
    }{
        {
            name: "email field props",
            input: `
                @schema User { email: email | {title: "Email Address"} }
                let user = User({email: "test@example.com"})
                user.fieldProps("email")
            `,
            expected: map[string]interface{}{
                "name":         "email",
                "type":         "email",
                "label":        "Email Address",
                "value":        "test@example.com",
                "autocomplete": "email",
            },
        },
        // ... more tests
    }
}
```

---

## 10. Migration Notes

### For Users

**Typed value formatting:**
- If relying on ISO format in output, use `.iso`: `<time datetime={date.iso}>`
- If need raw money, use `.inspect()` or explicit formatting

**Form fields:**
- Existing `@field` code continues to work unchanged
- New `<field/>` tag available for simpler forms
- `fieldProps()` available for component library integration

### For Library Authors

- Use `fieldProps()` inside component implementations
- Accept `record` + `field` props for schema-aware mode
- Fall back to manual props for non-schema usage

---

## 11. Future Considerations

### Not in Scope

1. **`<field/>` custom ordering** — Use Level 3 (`@field`) for custom ordering
2. **Nested records** — `fieldProps("address.street")` syntax not supported
3. **Array fields** — Need different UI patterns

### Potential Enhancements

- `schema.fieldProps(name)` — Props without record instance (no value/error)
- `record.allFieldProps()` — All visible fields at once
- `<field/>` `template` prop — Override entire structure with a component