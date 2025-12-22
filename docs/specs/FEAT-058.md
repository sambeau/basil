---
id: FEAT-058
title: "HTML Components in Prelude"
status: implemented
priority: medium
created: 2025-12-09
implemented: 2025-12-20
author: "@copilot"
depends-on: FEAT-056
part-of: FEAT-051
related: FEAT-046
---

# FEAT-058: HTML Components in Prelude

## Summary

Move HTML component implementations from Go code to Parsley files in the prelude. The `std/html` module loads components from `prelude/components/`, making them human-editable and easy to extend without Go changes.

## User Story

As a Basil maintainer, I want HTML components written in Parsley so that I can iterate on their implementation quickly and maintain consistency with user-written components.

## Acceptance Criteria

### Component Loading
- [x] `std/html` module loads components from `prelude/components/`
- [x] Components are pre-parsed at startup (via FEAT-056 infrastructure)
- [x] `import @std/html` provides `TextField`, `SelectField`, `Button`, etc.
- [x] Components use standard Parsley function syntax

### Components to Implement
- [x] `TextField` - text input with label, hint, error, accessibility
- [x] `SelectField` - select dropdown with options
- [x] `Button` - styled button with variants
- [x] `Form` - form wrapper with CSRF, confirmation, etc.
- [x] `DataTable` - sortable, paginated table
- [x] `CheckboxField` - checkbox with label (implemented as `Checkbox`)
- [x] `RadioGroup` - group of radio buttons
- [x] `TextAreaField` - multi-line text input

### Additional Components (beyond original spec)
- [x] `CheckboxGroup` - group of checkboxes
- [x] `Nav` - navigation landmark with ARIA label
- [x] `Breadcrumb` - breadcrumb navigation with Schema.org markup
- [x] `SkipLink` - accessibility skip link
- [x] `Img` - image with required alt and lazy loading
- [x] `Iframe` - iframe with required title and lazy loading
- [x] `Figure` - figure with caption
- [x] `Blockquote` - blockquote with citation
- [x] `SrOnly` - screen reader only text
- [x] `Abbr` - abbreviation with expansion
- [x] `A` - link with external link safety
- [x] `Icon` - accessible icon wrapper
- [x] `Time` - time element with datetime
- [x] `LocalTime` - client-side localized time
- [x] `TimeRange` - datetime span display
- [x] `RelativeTime` - relative time display

### JavaScript Integration
- [x] Components that need JS use data attributes (e.g., `data-confirm`)
- [ ] JavaScript in site bundle (via FEAT-063 `<Javascript/>`) handles enhancement
- [x] No per-component script injection needed

## Design Decisions

- **Parsley-native**: Components are regular Parsley functions using `fn({props})`
- **Pre-parsed**: No runtime parsing overhead
- **Accessible by default**: All components include proper ARIA attributes
- **Progressive enhancement**: JS enhances but isn't required
- **Unstyled**: Components emit semantic HTML with classes but no styles; users provide CSS
- **Contents prop**: Child content passed via `{contents}` prop (standard Parsley pattern)

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Directory Structure

```
prelude/
├── components/
│   ├── text_field.pars
│   ├── select_field.pars
│   ├── checkbox_field.pars
│   ├── radio_group.pars
│   ├── textarea_field.pars
│   ├── button.pars
│   ├── form.pars
│   └── data_table.pars
└── ...
```

### std/html Module

```go
// pkg/parsley/evaluator/stdlib_html.go

func loadHTMLModule(env *Environment) *Dictionary {
    components := make(map[string]Object)
    
    // Load each component from prelude
    componentFiles := []string{
        "components/text_field.pars",
        "components/select_field.pars",
        "components/button.pars",
        "components/form.pars",
        "components/data_table.pars",
        // ...
    }
    
    for _, file := range componentFiles {
        if ast, ok := preludeASTs[file]; ok {
            // Extract exported function
            // Component name derived from filename: text_field.pars -> TextField
            name := fileToComponentName(file)
            components[name] = extractExportedFunction(ast, name)
        }
    }
    
    return &Dictionary{Pairs: components}
}
```

### Example Component (text_field.pars)

```parsley
export TextField = fn({name, label, type, value, hint, error, required}) {
    let inputId = "field-" ++ name
    let hintId = if (hint) { inputId ++ "-hint" } else { null }
    let errorId = if (error) { inputId ++ "-error" } else { null }
    
    let describedBy = [hintId, errorId]
        .filter(fn(x) { x != null })
        .join(" ")
    
    <div class="field">
        <label for={inputId}>
            {label}
            if (required) {
                <span class="field-required" aria-hidden="true">*</span>
            }
        </label>
        <input 
            type={type ?? "text"}
            id={inputId}
            name={name}
            value={value ?? ""}
            required={required}
            aria-required={required}
            aria-describedby={if (describedBy != "") { describedBy } else { null }}
            aria-invalid={error != null}
        />
        if (hint) {
            <p id={hintId} class="field-hint">{hint}</p>
        }
        if (error) {
            <p id={errorId} class="field-error" role="alert">{error}</p>
        }
    </div>
}
```

### Example Component (form.pars)

```parsley
export Form = fn({action, method, confirm, autosubmit, contents}) {
    <form 
        action={action}
        method={method ?? "POST"}
        data-confirm={confirm}
    >
        {contents}
    </form>
}
```

### Usage in User Code

```parsley
{TextField, Button, Form} = import @std/html

ContactForm = fn() {
    <Form action="/contact" confirm="Submit this form?">
        <TextField name="email" label="Email" type="email" required={true}/>
        <TextField name="message" label="Message" hint="Max 500 characters"/>
        <Button type="submit">Send</Button>
    </Form>
}
```

### Affected Files

- `pkg/parsley/evaluator/stdlib_html.go` — New file: load components from prelude
- `pkg/parsley/evaluator/stdlib.go` — Register `std/html` module
- `prelude/components/*.pars` — Component implementations

## Related

- **Depends on**: FEAT-056 (Prelude Infrastructure), FEAT-063 (CSS/JS Auto-Bundle for JavaScript enhancement)
- **Part of**: FEAT-051 (Standard Prelude)
- **Completes**: FEAT-046 (HTML Components)
