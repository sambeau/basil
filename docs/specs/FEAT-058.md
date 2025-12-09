---
id: FEAT-058
title: "HTML Components in Prelude"
status: draft
priority: medium
created: 2025-12-09
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
- [ ] `std/html` module loads components from `prelude/components/`
- [ ] Components are pre-parsed at startup (via FEAT-056 infrastructure)
- [ ] `import @std/html` provides `TextField`, `SelectField`, `Button`, etc.
- [ ] Components use standard Parsley function syntax

### Components to Implement
- [ ] `TextField` - text input with label, hint, error, accessibility
- [ ] `SelectField` - select dropdown with options
- [ ] `Button` - styled button with variants
- [ ] `Form` - form wrapper with CSRF, confirmation, etc.
- [ ] `DataTable` - sortable, paginated table
- [ ] `CheckboxField` - checkbox with label
- [ ] `RadioGroup` - group of radio buttons
- [ ] `TextAreaField` - multi-line text input

### JavaScript Integration
- [ ] Components that need JS emit `<script type="module" src={basil.js.url}/>`
- [ ] `basil.js.url` available in prelude environment
- [ ] Script tag uses `type="module"` for deduplication

## Design Decisions

- **Parsley-native**: Components are regular Parsley functions
- **Pre-parsed**: No runtime parsing overhead
- **Accessible by default**: All components include proper ARIA attributes
- **Progressive enhancement**: JS enhances but isn't required

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
export fn TextField(props) {
    let {name, label, type, value, hint, error, required} = props
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
export fn Form(props, children) {
    let {action, method, confirm, autosubmit} = props
    
    let attrs = {
        action: action,
        method: method ?? "POST"
    }
    
    if (confirm) {
        attrs["data-confirm"] = confirm
    }
    
    <form {...attrs}>
        {children}
        if (confirm || autosubmit) {
            <script type="module" src={basil.js.url}/>
        }
    </form>
}
```

### Usage in User Code

```parsley
import {TextField, Button, Form} from @std/html

fn ContactForm() {
    <Form action="/contact" confirm="Submit this form?">
        <TextField name="email" label="Email" type="email" required={true}/>
        <TextField name="message" label="Message" hint="Max 500 characters"/>
        <Button type="submit">Send</Button>
    </Form>
}
```

### Prelude Environment for Components

Components receive a special environment with:

```go
env.Set("basil", map[string]interface{}{
    "js": map[string]interface{}{
        "url": JSAssetURL(),
    },
})
```

### Affected Files

- `pkg/parsley/evaluator/stdlib_html.go` — New file: load components from prelude
- `pkg/parsley/evaluator/stdlib.go` — Register `std/html` module
- `prelude/components/*.pars` — Component implementations

## Related

- **Depends on**: FEAT-056 (Prelude Infrastructure)
- **Part of**: FEAT-051 (Standard Prelude)
- **Completes**: FEAT-046 (HTML Components)
