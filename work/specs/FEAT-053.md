---
id: FEAT-053
title: "String .render() Method for Raw Interpolation"
status: implemented
priority: medium
created: 2025-12-09
implemented: 2025-12-09
author: "@human"
---

# FEAT-053: String .render() Method for Raw Interpolation

## Summary
Add a `.render()` method to strings that performs `@{...}` interpolation, allowing developers to create string templates where `{...}` is literal (for CSS, JavaScript, JSON) but `@{...}` triggers expression evaluation. Optionally accepts a dictionary to provide interpolation values. Also provides `printf()` builtin and `dictionary.render()` method as convenient synonyms.

## User Story
As a Parsley developer, I want to create CSS/JavaScript templates with literal `{...}` braces that only interpolate when I explicitly call `.render()`, so that I can cleanly separate template definition from value substitution without escaping braces. I also want convenient syntax options like `printf()` and `dict.render()` for common use cases.

## Acceptance Criteria
- [x] `string.render()` with no arguments interpolates `@{...}` expressions using current scope
- [x] `string.render(dict)` interpolates using provided dictionary values
- [x] `printf(string, dict)` works as synonym for `string.render(dict)`
- [x] `dict.render(string)` works as synonym for `string.render(dict)`
- [x] `@{...}` can contain full Parsley expressions (variables, functions, math, conditionals, method chains)
- [x] Regular `{...}` braces remain literal in the string
- [x] Nested braces in `@{...}` expressions are handled correctly
- [x] `\@` escape sequence produces literal `@` (prevents interpolation)
- [x] `markdown()` builtin applies `.render()` to markdown content before converting to HTML
- [x] Errors in interpolated expressions return proper Error objects
- [x] Methods added to `stringMethods` and `dictionaryMethods` arrays for fuzzy matching
- [x] Tests cover all use cases (simple vars, math, functions, conditionals, method chains, escaping, all three syntax forms, markdown integration)
- [x] Documentation updated in reference.md and CHEATSHEET.md

## Design Decisions
- **Name: `.render()`** — Most widely recognized term in templating libraries (Mustache, Handlebars, Jinja2, ERB, Liquid all use "render")
- **Syntax: `@{...}`** — Reuses existing raw text interpolation syntax from `<style>`/`<script>` tags, maintaining consistency
- **Optional dictionary parameter** — Allows both implicit (current scope) and explicit (provided values) interpolation contexts
- **Full expression support** — No artificial limitations; `@{...}` evaluates complete Parsley expressions just like `{...}` in template strings
- **`printf()` builtin** — Familiar name from C/Python, convenient for template-first workflows where template is defined inline
- **`dict.render()` method** — Natural for data-first workflows where you have a data object and want to apply it to a template

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context
### Affected Components
- `pkg/parsley/evaluator/methods.go` — Add `render` case to `evalStringMethod()` and `evalDictionaryMethod()`
- `pkg/parsley/evaluator/evaluator.go` — Add `interpolateRawString()` helper function, `printf` builtin, and update `markdown()` builtin
- `pkg/parsley/tests/string_methods_test.go` — Add comprehensive test suite for all syntax forms
- `pkg/parsley/tests/dictionary_methods_test.go` — Add tests for `dict.render()`
- `pkg/parsley/tests/builtins_test.go` — Add tests for `printf()`
- `pkg/parsley/tests/markdown_test.go` — Add tests for markdown with `@{...}` interpolation
- `docs/parsley/reference.md` — Document `.render()` method, `printf()` builtin, `dict.render()`, and markdown integration
- `docs/parsley/CHEATSHEET.md` — Add to string methods, dictionary methods, and builtins sections()`
- `docs/parsley/CHEATSHEET.md` — Add to string methods, dictionary methods, and builtins sections

### Dependencies
- None — Uses existing lexer/parser/evaluator infrastructure

### Implementation Details

#### String Method Signature
```go
case "render":
    // render() or render(dict)
    if len(args) > 1 {
        return newArityErrorRange("render", len(args), 0, 1)
    }
    
    var renderEnv *Environment
    if len(args) == 0 {
        // No args: use current environment
        renderEnv = env
    } else {
        // With dict arg: create new environment with those bindings
        dict, ok := args[0].(*Dictionary)
        if !ok {
            return newTypeError("TYPE-0012", "render", "a dictionary", args[0].Type())
        }
        // Create new environment and populate with dictionary values
        renderEnv = NewEnvironment()
        for key, valExpr := range dict.Pairs {
            val := Eval(valExpr, env)
            if isError(val) {
                return val
            }
            renderEnv.Set(key, val)
        }
    }
    
    return interpolateRawString(str.Value, renderEnv)
```

#### Dictionary Method Signature
```go
case "render":
    // render(string) - render a template string with this dictionary's values
    if len(args) != 1 {
        return newArityError("render", len(args), 1)
    }
    
    templateStr, ok := args[0].(*String)
    if !ok {
        return newTypeError("TYPE-0012", "render", "a string", args[0].Type())
    }
    
    // Create environment from dictionary
    renderEnv := NewEnvironment()
    for key, valExpr := range dict.Pairs {
        val := Eval(valExpr, env)
        if isError(val) {
            return val
        }
        renderEnv.Set(key, val)
    }
    
    return interpolateRawString(templateStr.Value, renderEnv)
```

#### printf() Builtin
```go
"printf": {
    Fn: func(args ...Object) Object {
        if len(args) != 2 {
            return newArityError("printf", len(args), 2)
        }
        
        templateStr, ok := args[0].(*String)
        if !ok {
            return newTypeError("TYPE-0005", "printf", "a string (template)", args[0].Type())
        }
        
        dict, ok := args[1].(*Dictionary)
        if !ok {
            return newTypeError("TYPE-0006", "printf", "a dictionary (values)", args[1].Type())
        }
        
        // Create environment from dictionary
        renderEnv := NewEnvironment()
        for key, valExpr := range dict.Pairs {
            val := Eval(valExpr, NewEnvironment()) // Evaluate in fresh env
            if isError(val) {
                return val
            }
            renderEnv.Set(key, val)
        }
        
        return interpolateRawString(templateStr.Value, renderEnv)
    },
},
```

#### Helper Function
```go
// interpolateRawString evaluates a string with @{...} interpolation
// Similar to evalTemplateLiteral but uses @{ instead of {
// Supports \@ escape sequence for literal @
func interpolateRawString(template string, env *Environment) Object {
    var result strings.Builder
    i := 0
    
    for i < len(template) {
        // Handle escape sequences
        if template[i] == '\\' && i+1 < len(template) {
            if template[i+1] == '@' {
                // \@ becomes literal @
                result.WriteByte('@')
                i += 2
                continue
            }
            // Other escapes pass through as-is
            result.WriteByte(template[i])
            i++
            continue
        }
        
        // Look for @{
        if i < len(template)-1 && template[i] == '@' && template[i+1] == '{' {
            i += 2 // skip @{
            braceCount := 1
            exprStart := i
            
            // Find matching }
            for i < len(template) && braceCount > 0 {
                if template[i] == '{' {
                    braceCount++
                } else if template[i] == '}' {
                    braceCount--
                }
                if braceCount > 0 {
                    i++
                }
            }
            
            if braceCount != 0 {
                return newParseError("PARSE-0009", "raw template", nil)
            }
            
            // Extract and evaluate the expression
            exprStr := template[exprStart:i]
            i++ // skip closing }
            
            // Parse and evaluate the expression
            l := lexer.New(exprStr)
            p := parser.New(l)
            program := p.ParseProgram()
            
            if len(p.Errors()) > 0 {
                return newParseError("PARSE-0011", "raw template", fmt.Errorf("%s", p.Errors()[0]))
            }
            
            // Evaluate the expression
            var evaluated Object
            for _, stmt := range program.Statements {
                evaluated = Eval(stmt, env)
                if isError(evaluated) {
                    return evaluated
                }
            }
            
            // Convert result to string
            if evaluated != nil {
                result.WriteString(objectToTemplateString(evaluated))
            }
        } else {
            // Regular character (including literal { and })
            result.WriteByte(template[i])
            i++
        }
    }
    
    return &String{Value: result.String()}
}
```

### Examples

#### Simple Variable Substitution
```parsley
let css = ".color { background: @{bgColor}; }"
css.render({bgColor: "red"})
// → ".color { background: red; }"
```

#### Math Expressions
```parsley
let css = "width: @{width * 2}px;"
css.render({width: 10})
// → "width: 20px;"
```

#### Function Calls
```parsley
let css = "content: '@{name.toUpper()}';"
css.render({name: "alice"})
// → "content: 'ALICE';"
```

#### Conditionals
```parsley
let css = "display: @{visible ? 'block' : 'none'};"
css.render({visible: true})
// → "display: block;"
```

#### Method Chains
```parsley
let css = "color: @{colors.first().toUpper()};"
css.render({colors: ["red", "blue"]})
// → "color: RED;"
```

#### Literal Braces Preserved
```parsley
let js = "function test() { return @{value}; }"
js.render({value: 42})
// → "function test() { return 42; }"
// Note: { } in function body are literal
```

#### Escaping @ Symbol
```parsley
let email = "Contact us at support\@example.com or @{contactEmail}"
email.render({contactEmail: "help@example.com"})
// → "Contact us at support@example.com or help@example.com"
```

#### Using Current Scope
```parsley
let bgColor = "blue"
let css = ".color { background: @{bgColor}; }"
css.render()  // no args, uses current scope
// → ".color { background: blue; }"
```

#### Using printf() Builtin
```parsley
printf("width: @{w}px; height: @{h}px;", {w: 100, h: 200})
// → "width: 100px; height: 200px;"
```

#### Using dict.render() Method
```parsley
let data = {name: "Alice", age: 30}
data.render("Hello, @{name}! You are @{age} years old.")
// → "Hello, Alice! You are 30 years old."
```

#### All Three Syntax Forms (Equivalent)
```parsley
let template = "Color: @{color}, Size: @{size}"
let data = {color: "red", size: "large"}

// String method
template.render(data)

// printf builtin
printf(template, data)

// Dictionary method
data.render(template)
// All three produce: "Color: red, Size: large"
```

#### Markdown Integration
```parsley
let content = markdown(@(./docs/template.md))
// If template.md contains: "# @{title}\n\n@{description}"
// And current scope has title="Hello" and description="World"
// Result: "<h1>Hello</h1>\n<p>World</p>"
6. **Other escape sequences** — `\n`, `\t`, etc. pass through as-is (already handled by string literal parsing)
7. **printf() vs string.render()** — Functionally identical when dict provided; `printf()` doesn't support no-args form (always requires dict)
8. **dict.render() scope** — Dictionary values evaluated in dictionary's original environment, not the render environment
9. **markdown() interpolation** — Applies `.render()` using current environment before Markdown→HTML conversion; allows dynamic content in markdown files
### Edge Cases & Constraints
1. **Nested braces** — Brace counting handles nested objects/blocks: `@{obj.prop}` where obj contains `{...}`
2. **Escaped @** — `\@` produces literal `@` character, preventing interpolation (useful for email addresses, CSS at-rules)
3. **Missing variables** — Returns standard Parsley "undefined identifier" error
## Implementation Notes
*To be added during implementation*

### Markdown Integration Details
The `markdown()` builtin should call `interpolateRawString()` on the file content before passing to the markdown processor:

```go
// In markdown() builtin, after reading file content:
content := string(fileContent)

// Apply render with current environment
rendered := interpolateRawString(content, env)
if isError(rendered) {
    return rendered
}

renderedStr, ok := rendered.(*String)
if !ok {
    return rendered // propagate error
}

// Now convert markdown to HTML
html := markdownToHTML(renderedStr.Value)
return &String{Value: html}
```

This allows markdown files to use `@{...}` for dynamic content while keeping literal `{...}` (useful for code blocks showing JSON, CSS, etc.).

## Related
- Plan: `docs/plans/FEAT-053-plan.md` (to be created)
- Similar: `evalTemplateLiteral()` in `evaluator.go` (line 9756)
- Context: Raw text mode in `<style>`/`<script>` uses same `@{...}` syntax
- Integration: `markdown()` builtin in `evaluator.go`ided; `printf()` doesn't support no-args form (always requires dict)
8. **dict.render() scope** — Dictionary values evaluated in dictionary's original environment, not the render environment

## Implementation Notes
*To be added during implementation*

## Related
- Plan: `docs/plans/FEAT-053-plan.md` (to be created)
- Similar: `evalTemplateLiteral()` in `evaluator.go` (line 9756)
- Context: Raw text mode in `<style>`/`<script>` uses same `@{...}` syntax
