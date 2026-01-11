---
id: PLAN-047
feature: FEAT-074
title: "Implementation Plan for Dictionary Spreading in HTML Tags"
status: draft
created: 2025-12-22
---

# Implementation Plan: FEAT-074 Dictionary Spreading in HTML Tags

## Overview

Add support for spreading dictionaries into HTML tag attributes using `...attrs` syntax, enabling cleaner component APIs and eliminating repetitive conditional attribute logic.

**Key features:**
- Spread dictionaries into tag attributes: `<input ...attrs/>`
- Smart boolean handling: `{async: true}` → `async`, `{async: false}` → omitted
- Null omission: `{placeholder: null}` → omitted
- Multiple spreads with override semantics
- Works with rest destructuring: `let {name, ...rest} = props`

## Prerequisites

- [x] FEAT-074 spec approved
- [x] Existing null attribute omission (commit 15c4f10) provides foundation
- [ ] Test cases defined
- [ ] Documentation structure planned

## Architecture

### Modified Components

```
pkg/parsley/
├── ast/ast.go               # Extend TagPairExpression and TagLiteral
├── lexer/lexer.go           # Tokenize ... as SPREAD
├── parser/parser.go         # Parse ...identifier in tag props
├── evaluator/evaluator.go   # Expand dictionaries to attributes
└── tests/
    ├── tags_test.go         # Add spread syntax tests
    └── components_test.go   # Test with components
```

### New Helper Functions

```go
// In evaluator.go
func evalDictionarySpread(dict *Dictionary, builder *strings.Builder) error
func isDictionarySpreadable(obj Object) bool
```

## Tasks

### Phase 1: Lexer & AST Changes

#### Task 1.1: Add SPREAD Token Type
**Files**: `pkg/parsley/lexer/lexer.go`
**Estimated effort**: Small (1-2 hours)

Add token type for the spread operator `...`.

**Changes:**

```go
// Add to token types
const (
    // ... existing tokens
    SPREAD = "SPREAD"  // ...
)

// In readToken() or readOperator()
if l.ch == '.' && l.peekChar() == '.' && l.peekCharN(2) == '.' {
    l.readChar()
    l.readChar()
    tok = Token{Type: SPREAD, Literal: "..."}
    return tok
}
```

**Tests:**
```go
func TestSpreadOperator(t *testing.T) {
    input := "...attrs"
    l := New(input)
    
    tok := l.NextToken()
    assert.Equal(t, SPREAD, tok.Type)
    assert.Equal(t, "...", tok.Literal)
    
    tok = l.NextToken()
    assert.Equal(t, IDENT, tok.Type)
    assert.Equal(t, "attrs", tok.Literal)
}
```

---

#### Task 1.2: Extend AST Node Structures
**Files**: `pkg/parsley/ast/ast.go`
**Estimated effort**: Small (2-3 hours)

Add `Spreads` field to tag AST nodes to track spread operations.

**Changes:**

```go
// TagPairExpression - paired tags like <div>...</div>
type TagPairExpression struct {
    Token    lexer.Token
    Name     string
    Props    string           // Raw props string (existing)
    Spreads  []*SpreadExpr    // NEW: spread expressions
    Contents []Node
}

// TagLiteral - singleton tags like <input/>
type TagLiteral struct {
    Token   lexer.Token
    Raw     string            // Raw tag content (existing)
    Spreads []*SpreadExpr     // NEW: spread expressions
}

// SpreadExpr - represents ...identifier
type SpreadExpr struct {
    Token      lexer.Token  // The ... token
    Expression Expression   // The identifier/expression to spread
}
```

**Tests:**
- Verify AST node creation with spreads
- Verify String() methods work correctly

---

### Phase 2: Parser Changes

#### Task 2.1: Parse Spread in Tag Props (Singleton Tags)
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium (4-6 hours)

Detect and parse `...identifier` in singleton tag syntax.

**Changes:**

```go
func (p *Parser) parseTagLiteral() ast.Expression {
    tag := &ast.TagLiteral{
        Token:   p.curToken,
        Raw:     p.curToken.Literal,
        Spreads: []*ast.SpreadExpr{},
    }
    
    // Scan for spreads in the raw string
    raw := tag.Raw
    spreads := p.extractSpreadsFromTagString(raw)
    tag.Spreads = spreads
    
    return tag
}

func (p *Parser) extractSpreadsFromTagString(raw string) []*ast.SpreadExpr {
    spreads := []*ast.SpreadExpr{}
    
    // Simple approach: scan for "..." followed by identifier
    // More robust: proper tokenization of tag content
    i := 0
    for i < len(raw) {
        if i+3 < len(raw) && raw[i:i+3] == "..." {
            // Found spread, extract identifier
            start := i + 3
            end := start
            for end < len(raw) && isIdentChar(raw[end]) {
                end++
            }
            if end > start {
                identName := raw[start:end]
                spreads = append(spreads, &ast.SpreadExpr{
                    Token: lexer.Token{Type: lexer.SPREAD, Literal: "..."},
                    Expression: &ast.Identifier{
                        Token: lexer.Token{Type: lexer.IDENT, Literal: identName},
                        Value: identName,
                    },
                })
            }
            i = end
        } else {
            i++
        }
    }
    
    return spreads
}
```

**Alternative approach (cleaner):**

Rewrite tag prop parsing to tokenize props properly instead of treating as raw string. This would:
1. Parse props into structured attributes
2. Identify spread syntax naturally
3. Avoid string scanning

This is more work but creates better architecture.

**Tests:**
```go
func TestParseTagWithSpread(t *testing.T) {
    input := `<input type="text" ...attrs/>`
    l := lexer.New(input)
    p := New(l)
    program := p.ParseProgram()
    
    tag := program.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.TagLiteral)
    assert.Equal(t, 1, len(tag.Spreads))
    assert.Equal(t, "attrs", tag.Spreads[0].Expression.(*ast.Identifier).Value)
}

func TestParseTagWithMultipleSpreads(t *testing.T) {
    input := `<input ...base ...override/>`
    // Similar assertions for 2 spreads
}
```

---

#### Task 2.2: Parse Spread in Tag Props (Paired Tags)
**Files**: `pkg/parsley/parser/parser.go`
**Estimated effort**: Medium (3-4 hours)

Same as Task 2.1 but for `TagPairExpression`.

**Changes:**

```go
func (p *Parser) parseTagPair() ast.Expression {
    tagExpr := &ast.TagPairExpression{
        Token:    p.curToken,
        Contents: []ast.Node{},
        Spreads:  []*ast.SpreadExpr{},
    }
    
    raw := p.curToken.Literal
    tagExpr.Name, tagExpr.Props = parseTagNameAndProps(raw)
    
    // Extract spreads from Props string
    tagExpr.Spreads = p.extractSpreadsFromTagString(tagExpr.Props)
    
    // ... rest of parsing
    return tagExpr
}
```

**Tests:**
- Same as Task 2.1 but for paired tags

---

### Phase 3: Evaluator Changes

#### Task 3.1: Implement Dictionary Spread Helper
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (4-6 hours)

Create helper function to expand a dictionary into HTML attributes.

**Implementation:**

```go
// evalDictionarySpread evaluates a dictionary and writes its key-value pairs
// as HTML attributes to the builder.
func evalDictionarySpread(dict *Dictionary, builder *strings.Builder) error {
    // Get all keys and sort them for deterministic output
    keys := make([]string, 0, len(dict.Pairs))
    for key := range dict.Pairs {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    
    for _, key := range keys {
        value := dict.Pairs[key]
        
        // Skip null and false values
        if isNullOrFalse(value) {
            continue
        }
        
        builder.WriteByte(' ')
        builder.WriteString(key)
        
        // Check if it's a boolean true (render as boolean attribute)
        if boolVal, ok := value.(*Boolean); ok && boolVal.Value {
            // Boolean attribute - no value
            continue
        }
        
        // Render as quoted attribute value
        builder.WriteString("=\"")
        strVal := objectToTemplateString(value)
        // Escape quotes in value
        for _, c := range strVal {
            if c == '"' {
                builder.WriteString("\\\"")
            } else {
                builder.WriteRune(c)
            }
        }
        builder.WriteByte('"')
    }
    
    return nil
}

// isNullOrFalse checks if an object is null or boolean false
func isNullOrFalse(obj Object) bool {
    switch v := obj.(type) {
    case *Null:
        return true
    case *Boolean:
        return !v.Value
    default:
        return false
    }
}
```

**Tests:**
```go
func TestEvalDictionarySpread(t *testing.T) {
    tests := []struct {
        name     string
        dict     *Dictionary
        expected string
    }{
        {
            name: "basic attributes",
            dict: &Dictionary{Pairs: map[string]Object{
                "id": &String{Value: "test"},
                "class": &String{Value: "box"},
            }},
            expected: ` class="box" id="test"`, // sorted
        },
        {
            name: "boolean true",
            dict: &Dictionary{Pairs: map[string]Object{
                "disabled": &Boolean{Value: true},
            }},
            expected: ` disabled`,
        },
        {
            name: "boolean false (omitted)",
            dict: &Dictionary{Pairs: map[string]Object{
                "disabled": &Boolean{Value: false},
            }},
            expected: ``,
        },
        {
            name: "null (omitted)",
            dict: &Dictionary{Pairs: map[string]Object{
                "placeholder": NULL,
            }},
            expected: ``,
        },
        {
            name: "numbers",
            dict: &Dictionary{Pairs: map[string]Object{
                "maxlength": &Integer{Value: 50},
            }},
            expected: ` maxlength="50"`,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var builder strings.Builder
            err := evalDictionarySpread(tt.dict, &builder)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, builder.String())
        })
    }
}
```

---

#### Task 3.2: Integrate Spreads in evalStandardTag
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (4-6 hours)

Modify `evalStandardTag` to process spreads before writing closing `/>`.

**Current structure:**
```go
func evalStandardTag(tagName string, propsStr string, env *Environment) Object {
    var result strings.Builder
    result.WriteByte('<')
    result.WriteString(tagName)
    
    // Process props with interpolation...
    
    result.WriteString(" />")
    return &String{Value: result.String()}
}
```

**New structure:**
```go
func evalStandardTag(node *ast.TagLiteral, env *Environment) Object {
    var result strings.Builder
    
    // Parse tag name from raw
    tagName := extractTagName(node.Raw)
    result.WriteByte('<')
    result.WriteString(tagName)
    
    // 1. Process regular props (existing logic)
    propsStr := extractPropsString(node.Raw)
    // ... existing interpolation logic
    
    // 2. Process spreads
    for _, spread := range node.Spreads {
        spreadObj := Eval(spread.Expression, env)
        if isError(spreadObj) {
            return spreadObj
        }
        
        // Verify it's a dictionary
        dict, ok := spreadObj.(*Dictionary)
        if !ok {
            return newError("cannot spread non-dictionary value in tag attributes")
        }
        
        // Expand dictionary to attributes
        if err := evalDictionarySpread(dict, &result); err != nil {
            return newError("error spreading attributes: %s", err)
        }
    }
    
    result.WriteString(" />")
    return &String{Value: result.String()}
}
```

**Key changes:**
- Change function signature to accept `*ast.TagLiteral` instead of raw strings
- Add spread processing after regular props
- Evaluate spread expressions and expand dictionaries

**Tests:**
```go
func TestEvalTagWithSpread(t *testing.T) {
    input := `
        let attrs = {placeholder: "Name", maxlength: 50}
        <input type="text" ...attrs/>
    `
    result := testEval(input)
    expected := `<input type="text" maxlength="50" placeholder="Name" />`
    assert.Equal(t, expected, result.(*String).Value)
}

func TestEvalTagWithMultipleSpreads(t *testing.T) {
    input := `
        let base = {type: "text", class: "input"}
        let override = {class: "input-lg"}
        <input ...base ...override/>
    `
    result := testEval(input)
    // class should be "input-lg" (override wins)
    assert.Contains(t, result.(*String).Value, `class="input-lg"`)
}

func TestEvalTagSpreadNonDict(t *testing.T) {
    input := `<input ..."not a dict"/>`
    result := testEval(input)
    assert.True(t, isError(result))
    assert.Contains(t, result.(*Error).Message, "cannot spread non-dictionary")
}
```

---

#### Task 3.3: Integrate Spreads in evalStandardTagPair
**Files**: `pkg/parsley/evaluator/evaluator.go`
**Estimated effort**: Medium (4-6 hours)

Same as Task 3.2 but for paired tags.

**Changes:**
```go
func evalStandardTagPair(node *ast.TagPairExpression, env *Environment) Object {
    var result strings.Builder
    result.WriteByte('<')
    result.WriteString(node.Name)
    
    // 1. Process regular props via evalTagProps
    if node.Props != "" {
        propsResult := evalTagProps(node.Props, env)
        if isError(propsResult) {
            return propsResult
        }
        result.WriteString(propsResult.(*String).Value)
    }
    
    // 2. Process spreads (NEW)
    for _, spread := range node.Spreads {
        spreadObj := Eval(spread.Expression, env)
        if isError(spreadObj) {
            return spreadObj
        }
        
        dict, ok := spreadObj.(*Dictionary)
        if !ok {
            return newError("cannot spread non-dictionary value in tag attributes")
        }
        
        if err := evalDictionarySpread(dict, &result); err != nil {
            return newError("error spreading attributes: %s", err)
        }
    }
    
    result.WriteByte('>')
    
    // ... rest of paired tag logic
}
```

**Tests:**
- Similar to Task 3.2 but with paired tags

---

### Phase 4: Integration & Testing

#### Task 4.1: Component Integration Tests
**Files**: `pkg/parsley/tests/components_test.go`
**Estimated effort**: Medium (3-4 hours)

Test spreading with actual component patterns.

**Test cases:**

```go
func TestComponentWithSpread(t *testing.T) {
    input := `
        let TextField = fn(props) {
            let {name, label, ...inputAttrs} = props
            <input name={name} ...inputAttrs/>
        }
        
        <TextField 
            name="email" 
            label="Email"
            placeholder="you@example.com"
            required={true}
            maxlength={100}
        />
    `
    
    result := testEval(input)
    html := result.(*String).Value
    
    // Should have name from explicit prop
    assert.Contains(t, html, `name="email"`)
    // Should have spread attributes
    assert.Contains(t, html, `placeholder="you@example.com"`)
    assert.Contains(t, html, `required`)
    assert.Contains(t, html, `maxlength="100"`)
    // Should NOT have label (not in inputAttrs)
    assert.NotContains(t, html, `label`)
}

func TestComponentConditionalSpreads(t *testing.T) {
    input := `
        let Button = fn(props) {
            let {text, ...attrs} = props
            <button ...attrs>text</button>
        }
        
        <Button 
            text="Click" 
            disabled={false} 
            class="btn"
        />
    `
    
    result := testEval(input)
    html := result.(*String).Value
    
    // disabled=false should be omitted
    assert.NotContains(t, html, "disabled")
    assert.Contains(t, html, `class="btn"`)
}
```

---

#### Task 4.2: Update Existing Components
**Files**: `server/prelude/components/*.pars`
**Estimated effort**: Medium (4-6 hours)

Refactor existing components to use spread syntax.

**Example - TextField:**

Before:
```parsley
export TextField = fn({name, label, type, value, hint, error, required, id, class, placeholder, autocomplete, disabled, readonly, minlength, maxlength, pattern}) {
    <input 
        type={type ?? "text"}
        name={name}
        value={value ?? ""}
        placeholder={if (placeholder) placeholder else null}
        autocomplete={if (autocomplete) autocomplete else null}
        disabled={if (disabled) "disabled" else null}
        readonly={if (readonly) "readonly" else null}
        minlength={if (minlength) minlength else null}
        maxlength={if (maxlength) maxlength else null}
        pattern={if (pattern) pattern else null}
        required={if (required) "required" else null}
    />
}
```

After:
```parsley
export TextField = fn(props) {
    let {name, label, type, value, hint, error, id, class: className, ...inputAttrs} = props
    
    let fieldId = id ?? "field-" ++ name
    let inputId = fieldId ++ "-input"
    
    <div class={"field" ++ if (className) { " " ++ className } else { "" }} id={fieldId}>
        <label for={inputId}>
            label
            if (required) {
                <span class="field-required" aria-hidden="true">" *"</span>
            }
        </label>
        <input 
            type={type ?? "text"}
            id={inputId}
            name={name}
            value={value ?? ""}
            ...inputAttrs
        />
        if (hint) {
            <p id={hintId} class="field-hint">hint</p>
        }
        if (error) {
            <p id={errorId} class="field-error" role="alert">error</p>
        }
    </div>
}
```

Components to update:
- `text_field.pars` ✓
- `textarea_field.pars`
- `select_field.pars`
- `button.pars`
- `checkbox.pars`
- `radio_group.pars`
- `img.pars`
- `iframe.pars`
- `a.pars`

---

### Phase 5: Documentation

#### Task 5.1: Update CHEATSHEET.md
**Files**: `docs/parsley/CHEATSHEET.md`
**Estimated effort**: Small (1-2 hours)

Add spreading syntax to the cheatsheet.

**Addition:**

```markdown
### Dictionary Spreading in Tags

Spread dictionaries into HTML attributes:

```parsley
// Basic spreading
let attrs = {placeholder: "Name", maxlength: 50, disabled: true}
<input type="text" ...attrs/>
// → <input type="text" placeholder="Name" maxlength="50" disabled/>

// Boolean handling
{async: true}         // → async (boolean attribute)
{disabled: false}     // → (omitted)
{placeholder: null}   // → (omitted)

// Multiple spreads (last wins)
let base = {class: "input"}
let override = {class: "input-lg"}
<input ...base ...override/>  // → class="input-lg"

// With rest destructuring
let TextField = fn(props) {
    let {name, label, ...inputAttrs} = props
    <input name={name} ...inputAttrs/>
}
```
```

---

#### Task 5.2: Update reference.md
**Files**: `docs/parsley/reference.md`
**Estimated effort**: Medium (2-3 hours)

Add formal documentation of spreading syntax.

**Sections to add:**
- Syntax: `...identifier` in tag props
- Evaluation semantics
- Value type handling table
- Override behavior
- Error cases

---

#### Task 5.3: Update Component Documentation
**Files**: `docs/specs/FEAT-073.md`, `docs/manual/stdlib/html.md`
**Estimated effort**: Small (1-2 hours)

Update component documentation with new spread-based APIs.

---

### Phase 6: Edge Cases & Polish

#### Task 6.1: Handle Edge Cases
**Estimated effort**: Medium (3-4 hours)

Implement and test edge case handling:

1. **Spread in wrong context**: `let x = ...attrs` (syntax error)
2. **Spread non-existent identifier**: `<input ...notDefined/>` (runtime error)
3. **Spread expression**: `<input ...{a: 1, b: 2}/>` (support inline dict literals)
4. **Hyphenated keys**: `{"data-id": 123}` → `data-id="123"`
5. **Numeric keys**: `{0: "value"}` → `0="value"`
6. **Special characters in keys**: Need escaping?

---

#### Task 6.2: Performance Testing
**Estimated effort**: Small (2-3 hours)

Benchmark spreading vs. manual attributes to ensure no significant performance degradation.

```go
func BenchmarkTagWithSpread(b *testing.B) {
    input := `
        let attrs = {placeholder: "Name", maxlength: 50}
        <input type="text" ...attrs/>
    `
    // Run benchmark
}

func BenchmarkTagManualAttrs(b *testing.B) {
    input := `
        <input type="text" placeholder="Name" maxlength="50"/>
    `
    // Run benchmark
}
```

---

## Testing Strategy

### Unit Tests
- ✅ Lexer: SPREAD token recognition
- ✅ Parser: spread extraction from tags
- ✅ AST: spread node structure
- ✅ Evaluator: dictionary spreading logic
- ✅ Evaluator: boolean/null handling
- ✅ Evaluator: multiple spreads
- ✅ Error cases

### Integration Tests
- ✅ Components with rest destructuring
- ✅ Conditional spreads
- ✅ Override semantics
- ✅ Mixed regular attrs + spreads

### Example Tests
- Test with actual component refactors
- Test with auth example
- Test with hello example

## Migration Guide

### For Component Authors

**Before:**
```parsley
fn({name, placeholder, disabled, ...}) {
    <input 
        name={name}
        placeholder={if (placeholder) placeholder else null}
        disabled={if (disabled) "disabled" else null}
    />
}
```

**After:**
```parsley
fn({name, ...inputAttrs}) {
    <input name={name} ...inputAttrs/>
}
```

### Backward Compatibility

This is a purely additive feature. All existing code continues to work unchanged. Components can be gradually refactored to use spreading.

## Rollout Plan

1. **Week 1**: Lexer & Parser (Tasks 1.1, 1.2, 2.1, 2.2)
2. **Week 2**: Evaluator core (Tasks 3.1, 3.2, 3.3)
3. **Week 3**: Testing & Components (Tasks 4.1, 4.2)
4. **Week 4**: Documentation & Polish (Tasks 5.1-5.3, 6.1-6.2)

## Success Criteria

- ✅ All unit tests pass
- ✅ All integration tests pass
- ✅ Existing test suite still passes (no regressions)
- ✅ Components refactored and working
- ✅ Documentation complete
- ✅ Performance acceptable (< 5% overhead)

## Open Issues

1. **Parser approach**: String scanning vs. proper tokenization?
   - Recommendation: Start with string scanning, refactor to proper tokenization later

2. **Spread expressions**: Should `...{a: 1}` work (inline dict)?
   - Recommendation: Yes, evaluate expression and check if dict

3. **Attribute ordering**: Should spreads maintain insertion order?
   - Recommendation: Sort keys alphabetically for deterministic output

4. **Error messages**: How verbose should errors be?
   - Recommendation: Clear, actionable messages with file/line info

## Risk Assessment

**Low risk:**
- Additive feature, no breaking changes
- Well-defined semantics from JSX precedent
- Core null omission logic already exists

**Medium risk:**
- Parser changes could be complex if proper tokenization needed
- Integration with existing prop handling

**Mitigation:**
- Comprehensive test coverage
- Incremental implementation
- Fallback to string scanning if tokenization too complex

## Estimated Total Effort

- **Development**: 40-50 hours
- **Testing**: 15-20 hours  
- **Documentation**: 10-12 hours
- **Total**: 65-82 hours (~2-3 weeks)
