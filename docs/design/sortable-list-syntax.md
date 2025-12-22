# Sortable List Syntax Exploration

## The Problem

The JSX-style syntax I suggested doesn't work in Parsley:

```parsley
// ❌ WRONG - This is JSX, not Parsley
<SortableList items={tasks}>
    {task => <div>{task.title}</div>}
</SortableList>
```

The `{task => ...}` is JavaScript arrow function syntax. Parsley doesn't have this construct inside tag content.

## How Parsley Tag Contents Work

Currently, when you write:

```parsley
<Card>
    <p>Hello</p>
    {name}
</Card>
```

The `Card` component receives `contents` as a **string** (the evaluated HTML). The component function sees:

```parsley
fn({contents}) {
    // contents = "<p>Hello</p>Sam"  (a string)
}
```

The evaluator in `evalTagContentsAsArray`:
1. Evaluates each child node to a string
2. Collects them into an array  
3. If single item, passes as string directly
4. Otherwise passes as array of strings

## Current Limitation

There's no way to pass:
- An **unevaluated template** to iterate over later
- A **function** that receives each item
- **Deferred evaluation** of the children

## Possible Approaches

### Approach 1: Explicit Loop (Current Best Option)

**You do the loop, component wraps:**

```parsley
<SortableList endpoint="/api/reorder" group="kanban">
    @for task in tasks {
        <SortableItem id={task.id}>
            <div class="card">{task.title}</div>
        </SortableItem>
    }
</SortableList>
```

**Component definition:**

```parsley
export SortableList = fn({endpoint, group, contents}) {
    <ul 
        class="sortable-list"
        data-sortable
        data-endpoint={endpoint}
        data-group={group}
    >
        (contents)
    </ul>
}

export SortableItem = fn({id, contents}) {
    <li data-sortable-id={id}>
        (contents)
    </li>
}
```

**Pros:**
- Works today with zero language changes
- Explicit and readable
- Full control over item structure
- Natural Parsley idioms

**Cons:**
- More verbose
- Two components needed instead of one

---

### Approach 2: Template Prop (Function Reference)

**Pass a rendering function as prop:**

```parsley
let TaskCard = fn(task) {
    <div class="card">{task.title}</div>
}

<SortableList items={tasks} itemKey="id" render={TaskCard}/>
```

**Component uses `.map()` internally:**

```parsley
export SortableList = fn({items, itemKey, render, endpoint}) {
    <ul data-sortable data-endpoint={endpoint}>
        @for item in items {
            <li data-sortable-id={item[itemKey]}>
                {render(item)}
            </li>
        }
    </ul>
}
```

**Pros:**
- Single component handles everything
- Clean separation of data and presentation
- Works today (functions are first-class)

**Cons:**
- Rendering logic split from usage site
- Need to define named function elsewhere
- Less intuitive for simple cases

---

### Approach 3: Inline Anonymous Function

If Parsley allowed inline functions in props:

```parsley
<SortableList items={tasks} render={fn(t) { <div>{t.title}</div> }}/>
```

**Status:** This syntax might already work! Let me check...

Actually, Parsley supports inline functions:

```parsley
let double = fn(x) { x * 2 }
```

And you can pass functions as props. So this should work:

```parsley
<SortableList 
    items={tasks} 
    itemKey="id"
    render={fn(task) { <div class="card">{task.title}</div> }}
    endpoint="/api/reorder"
/>
```

**Let's verify this is valid Parsley!**

---

### Approach 4: Named Slot Pattern

Inspired by Svelte/Vue slots with let-directives. Would require language extension:

```parsley
// Hypothetical syntax - NOT current Parsley
<SortableList items={tasks} itemKey="id">
    @slot(item) {
        <div class="card">{item.title}</div>
    }
</SortableList>
```

The `@slot(item)` creates a template that receives `item` from the component.

**Pros:**
- Keeps template inline with usage
- Clear scoping

**Cons:**
- Requires new syntax
- Implementation complexity

---

### Approach 5: Named Template Pattern

Define template inline, reference by name:

```parsley
// Hypothetical syntax
<SortableList items={tasks} itemKey="id" template={@taskItem}>
    @template taskItem(item) {
        <div class="card">{item.title}</div>
    }
</SortableList>
```

**Cons:**
- New syntax needed
- Somewhat awkward

---

### Approach 6: Eval-based Templating (Anti-pattern?)

Pass a template string that gets evaluated per-item:

```parsley
<SortableList 
    items={tasks} 
    itemKey="id"
    template="<div class='card'>{item.title}</div>"
/>
```

The component would need to evaluate the string for each item with `item` in scope.

**Cons:**
- Security concerns (string eval)
- No syntax highlighting
- Error-prone

---

## Could Contents Be Non-String?

The user asks: *"is it necessary for the contents handed to a tag-pair function to always be a string?"*

### Current Implementation (evaluator.go)

Looking at `evalCustomTagPair` and `evalTagContentsAsArray`:

```go
// evalTagContentsAsArray evaluates tag contents and returns as an array
func evalTagContentsAsArray(contents []ast.Node, env *Environment) Object {
    elements := make([]Object, 0, len(contents))
    for _, node := range contents {
        obj := Eval(node, env)
        // Convert to string for consistency
        elements = append(elements, &String{Value: objectToTemplateString(obj)})
    }
    return &Array{Elements: elements}
}
```

Then in `evalCustomTagPair`:

```go
if contentsArray, ok := contentsObj.(*Array); ok && len(contentsArray.Elements) == 1 {
    // Single item - pass directly (as string)
    dict.Pairs["contents"] = createLiteralExpression(contentsArray.Elements[0])
} else {
    // Multiple items - pass as array (of strings)
    dict.Pairs["contents"] = createLiteralExpression(contentsObj)
}
```

So contents **is already an array** internally, but each element is stringified.

### What Could Be Changed?

**Option A: Pass AST Nodes (Don't Evaluate)**

Instead of evaluating children, pass the raw AST:

```go
// In evalCustomTagPair - hypothetically
dict.Pairs["contents"] = &ast.ArrayLiteral{Elements: node.Contents}
```

Components could then call a `render(contents, bindings)` function.

**Problems:**
- Exposes AST to user code (leaky abstraction)
- Need new builtin for evaluating AST
- Complex

**Option B: Pass Evaluated Objects (Not Stringified)**

```go
// Change evalTagContentsAsArray to NOT stringify:
func evalTagContentsAsArray(contents []ast.Node, env *Environment) Object {
    elements := make([]Object, 0, len(contents))
    for _, node := range contents {
        obj := Eval(node, env)
        elements = append(elements, obj)  // Keep as Object, don't stringify
    }
    return &Array{Elements: elements}
}
```

This would let contents contain:
- Strings (from text nodes)
- Dictionaries (from `{...}`)
- Arrays
- **Functions** (if passed via `{myFunction}`)

**Use Case:**
```parsley
<TabSet>
    {fn() { <Tab title="First">Content 1</Tab> }}
    {fn() { <Tab title="Second">Content 2</Tab> }}
</TabSet>
```

The `TabSet` could receive an array of functions and call each to render.

**Problems:**
- Breaking change to existing components
- Unclear what non-string contents mean
- How to handle mixed string + function arrays?

**Option C: Pass Functions for Deferred Eval**

Pass a "thunk" - a function that evaluates children when called:

```go
// In evalCustomTagPair  
contentsFn := &Function{
    Params: []*ast.Identifier{},
    Body: &ast.BlockStatement{Statements: contentsAsStatements},
    Env: env,
}
dict.Pairs["contents"] = createLiteralExpression(contentsFn)
```

Usage:
```parsley
export Card = fn({contents}) {
    <div class="card">
        // contents is a function, call it to get HTML
        {contents()}
    </div>
}
```

**Problems:**
- Breaking change (contents becomes callable)
- Awkward API

### Option C: New `@children` Directive

Special syntax for "render my children here":

```parsley
export Card = fn({title}) {
    <div class="card">
        <h2>{title}</h2>
        @children
    </div>
}
```

This is cleaner than `(contents)` but doesn't solve the iteration problem.

---

## Experimental Testing Results

I tested these approaches in the Parsley REPL. Key findings:

### ✅ Named Function Props Work

```parsley
// Define render function externally
let renderTask = fn(t) {
    <li>(t.title)</li>      // Use (parens) not {braces} for interpolation in fn body
}

<Mapper items={tasks} render={renderTask}/>   // ✅ Works
```

### ⚠️ Inline Functions Need String Building

```parsley
// This WORKS - string concatenation
<Mapper items={tasks} render={fn(t) { "<li>" + t.title + "</li>" }}/>

// This WORKS - toString()
<Mapper items={tasks} render={fn(t) { toString("<li>", t.title, "</li>") }}/>

// This FAILS - inline function with HTML tags
<Mapper items={tasks} render={fn(t) { <li>(t.title)</li> }}/>   // ❌ Parser confusion
```

### The Parsing Quirk

Inside function bodies, `{x}` starts a dictionary literal, not interpolation.
Use `(x)` for interpolation in function bodies that return HTML.

But when the function is defined **inline in a tag prop**, the parser gets confused
by nested tags. Workarounds:

1. Define the function externally
2. Use string building instead of tags

---

## Recommendation

For **SortableList specifically**, I recommend **Approach 1 + Approach 2**:

### Simple Pattern (Explicit Loop) - PREFERRED

```parsley
<SortableList endpoint="/api/reorder">
    @for task in tasks {
        <SortableItem id={task.id}>
            <div>{task.title}</div>
        </SortableItem>
    }
</SortableList>
```

### Shorthand Pattern (Named Function Prop)

```parsley
// Define render function first
let TaskCard = fn(task) {
    <div class="card">
        <span>(task.title)</span>    // Note: parens for interpolation in fn body
    </div>
}

// Then use it
<SortableList 
    items={tasks} 
    itemKey="id"
    render={TaskCard}
    endpoint="/api/reorder"
/>
```

**Both work with current Parsley** - the explicit loop is cleaner and avoids the parsing quirk.

### Component Supporting Both

```parsley
export SortableList = fn({items, itemKey, render, endpoint, group, contents}) {
    <ul 
        data-sortable
        data-endpoint={endpoint}
        data-group={group}
    >
        if (items && render) {
            // Function-based: component does the loop
            @for item in items {
                <li data-sortable-id={item[itemKey]}>
                    {render(item)}
                </li>
            }
        } else {
            // Children-based: user provides SortableItems
            (contents)
        }
    </ul>
}

export SortableItem = fn({id, contents}) {
    <li data-sortable-id={id}>
        (contents)
    </li>
}
```

---

## Usage Examples

### Basic (Children Pattern) - RECOMMENDED

```parsley
<SortableList endpoint="/api/reorder">
    @for task in tasks {
        <SortableItem id={task.id}>
            <div class="task-card">
                <span class="task-title">{task.title}</span>
                <span class="task-status">{task.status}</span>
            </div>
        </SortableItem>
    }
</SortableList>
```

### Named Function (Function Pattern)

```parsley
// Define render function (note: use parens for interpolation in fn body)
let TaskCard = fn(t) {
    <div class="task">(t.title)</div>
}

<SortableList 
    items={tasks} 
    itemKey="id"
    render={TaskCard}
    endpoint="/api/tasks/reorder"
/>
```

### Kanban with Multiple Lists

```parsley
let columns = ["todo", "doing", "done"]

<div class="kanban">
    @for col in columns {
        let columnTasks = tasks.filter(fn(t) { t.status == col })
        <div class="column">
            <h3>{col.toUpper()}</h3>
            <SortableList 
                endpoint="/api/tasks/move"
                group="kanban"
                data-column={col}
            >
                @for task in columnTasks {
                    <SortableItem id={task.id}>
                        <div class="card">{task.title}</div>
                    </SortableItem>
                }
            </SortableList>
        </div>
    }
</div>
```

---

## Summary

| Approach | Works Today? | Syntax | Best For |
|----------|--------------|--------|----------|
| Explicit loop with `<SortableItem>` | ✅ Yes | `@for task in tasks { <SortableItem id={task.id}>...</SortableItem> }` | Complex items, recommended |
| Named function prop | ✅ Yes | `let F = fn(t) {...}; <SortableList render={F}/>` | Reusable renderers |
| Inline function (string) | ✅ Yes | `render={fn(t) { "<li>" + t.title + "</li>" }}` | Simple items, hacky |
| Inline function (tags) | ❌ Parser issue | - | Avoid for now |
| `@each` directive | ❌ Not implemented | Future language feature | Would be nice |

**Recommendation: Use the explicit loop pattern with `<SortableList>` and `<SortableItem>` components.**

---

## Future Language Enhancement?

If we wanted cleaner iteration syntax, consider a `@each` directive for components:

```parsley
// Hypothetical future syntax
<SortableList items={tasks} @each="task">
    <div class="card">{task.title}</div>
</SortableList>
```

Where `@each="task"` tells the component to iterate `items` with `task` as the loop variable, evaluating children for each.

This would require:
1. New `@each` attribute syntax in parser
2. Evaluator support for injecting loop variable into children's scope
3. Components opting in to iteration behavior

**Complexity: Medium** - Could be a nice addition but not essential given the working alternatives.
