# Design Investigation: Schema-Based HTML Form Validation

**Status:** Investigation  
**Date:** 2025-01-14  
**Related:** BACKLOG #21, FEAT-002

## 1. Problem Statement

Basil/Parsley currently has:
- `@schema{}` DSL for defining data structures with validation rules
- `ValidateSchemaFields()` for server-side validation (returns errors dict)
- Parts system with `part-submit` for interactive form handling
- Database binding (`db.bind()`) tied to schemas

But there's no cohesive way to:
1. Bind HTML forms to schemas for automatic validation
2. Display validation errors elegantly
3. Re-populate forms after validation failure
4. Generate HTML5 validation attributes from schema constraints

## 2. Prior Art Analysis

### 2.1 Rails (form_with + ActiveRecord)

```ruby
# Model-driven: form binds to model, auto-populates
<%= form_with model: @book do |form| %>
  <%= form.text_field :title %>   # name="book[title]" value="..."
  <%= form.submit %>
<% end %>

# Errors available via @book.errors
@error('title') { display error }
```

**Key insights:**
- Form bound to model object, not just schema
- Same form used for create (empty) and edit (populated)
- Validation runs on model, errors attached to model
- `accepts_nested_attributes_for` handles related objects

### 2.2 Django (Form class)

```python
class ContactForm(forms.Form):
    email = forms.EmailField()
    message = forms.CharField(max_length=100)

# View
if request.method == 'POST':
    form = ContactForm(request.POST)  # bound form
    if form.is_valid():
        # process form.cleaned_data
else:
    form = ContactForm()  # unbound form

# Template
{{ form.email.errors }}  # field errors
{{ form.email }}         # renders <input>
```

**Key insights:**
- Form class is like a schema (fields + validation)
- "Bound" vs "unbound" forms (with/without data)
- Form object renders itself, including errors
- `cleaned_data` gives validated/typed data

### 2.3 Laravel (Form Requests + Blade)

```php
// Validation in controller
$validated = $request->validate([
    'title' => 'required|max:255',
    'email' => 'required|email',
]);

// Blade template
<input name="title" value="{{ old('title') }}" 
       class="@error('title') is-invalid @enderror">
@error('title')
    <span class="error">{{ $message }}</span>
@enderror
```

**Key insights:**
- Validation rules in controller, not model
- `old()` helper re-populates from flash storage
- `@error` directive for inline error display
- Automatic redirect on failure with errors flashed

### 2.4 Phoenix LiveView (Changesets)

```elixir
# Schema + changeset for validation
def changeset(user, attrs) do
  user
  |> cast(attrs, [:name, :email])
  |> validate_required([:name, :email])
  |> validate_format(:email, ~r/@/)
end

# In LiveView - real-time validation
def handle_event("validate", %{"user" => params}, socket) do
  changeset = User.changeset(%User{}, params)
  {:noreply, assign(socket, :changeset, changeset)}
end
```

**Key insights:**
- Changeset is the validated "diff" between old and new state
- Same changeset for create and update
- Real-time validation on change events

## 3. What Basil/Parsley Already Has

### 3.1 Schema Definition

```parsley
let UserSchema = @schema {
    name: string(min: 2, max: 50),
    email: email(required),
    age: int(min: 18),
    role: enum("admin", "user", "guest")
}
```

### 3.2 Schema Validation (Go)

```go
// ValidateSchemaFields validates a map against schema
// Returns: {error: "VALIDATION_ERROR", message: "...", fields: [{field, code, message}...]}

// Supports:
// - Types: email, url, phone, slug, enum
// - Constraints: min/max length, min/max value, required, nullable
```

### 3.3 Parts Form Handling (Real Syntax)

Part files (`.part`) export functions. Each function is a "view" that receives props and returns HTML:

```parsley
// user-form.part

export default = fn(props) {
    let data = props.data ?? {}
    
    <form part-submit="save">
        <input name="name" value={data.name}/>
        <input name="email" value={data.email}/>
        <button type="submit">Save</button>
    </form>
}

export save = fn(props) {
    // props contains form data, automatically type-coerced
    // props.name, props.email from form inputs
    
    // Process and return new HTML
    <div>Saved: {props.name}</div>
}
```

**Key Part behaviors:**
- `part-submit="viewName"` calls view on form submit
- `part-click="viewName"` calls view on click
- `part-*` attributes pass additional props
- Props are type-coerced (strings → Integer, Float, Boolean)
- Views return HTML that replaces the Part's content

## 4. Design Options

### Option A: Schema-Only Validation (Minimal)

Just connect `schema.validate()` to form handling in Parts:

```parsley
// user-form.part

let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
    email: email(required),
    age: int(min: 18)
}

export default = fn(props) {
    let data = props.data ?? {}
    let errors = props.errors ?? {}
    
    <form part-submit="save">
        <div class="field">
            <label>Name</label>
            <input name="name" value={data.name}
                   class={errors.name ? "error" : ""}/>
            {errors.name && <span class="error">{errors.name}</span>}
        </div>
        
        <div class="field">
            <label>Email</label>
            <input name="email" value={data.email}
                   class={errors.email ? "error" : ""}/>
            {errors.email && <span class="error">{errors.email}</span>}
        </div>
        
        <button type="submit">Save</button>
    </form>
}

export save = fn(props) {
    let result = UserSchema.validate(props)
    
    if (result.ok) {
        // Success - save and show confirmation
        let _ = @DB.users.insert(result.data)
        <div class="success">User saved!</div>
    } else {
        // Validation failed - re-render form with errors
        // Pass form data back to preserve input
        default({data: props, errors: result.errors})
    }
}
```

**Pros:** Simple, explicit, no magic  
**Cons:** Verbose, error handling pattern repeated everywhere

### Option B: Schema Methods + Helpers

Add methods to schema for HTML generation:

```parsley
// user-form.part

let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
    email: email(required)
}

export default = fn(props) {
    let data = props.data ?? {}
    let errors = props.errors ?? {}
    
    <form part-submit="save">
        <div class="field">
            <label>Name</label>
            // schema.attrs() returns HTML5 validation attributes
            <input name="name" 
                   type={UserSchema.attrs("name").type}
                   minlength={UserSchema.attrs("name").minlength}
                   maxlength={UserSchema.attrs("name").maxlength}
                   required={UserSchema.attrs("name").required}
                   value={data.name}
                   class={errors.name ? "error" : ""}/>
            {errors.name && <span class="error">{errors.name}</span>}
        </div>
        
        <div class="field">
            <label>Email</label>
            <input name="email"
                   type="email"
                   required
                   value={data.email}
                   class={errors.email ? "error" : ""}/>
            {errors.email && <span class="error">{errors.email}</span>}
        </div>
        
        <button type="submit">Save</button>
    </form>
}

export save = fn(props) {
    let result = UserSchema.validate(props)
    
    if (result.ok) {
        let _ = @DB.users.insert(result.data)
        <div class="success">User saved!</div>
    } else {
        default({data: props, errors: result.errors})
    }
}
```

**schema.attrs("fieldName")** returns:
```parsley
// For string(min: 2, max: 50, required)
{type: "text", minlength: 2, maxlength: 50, required: true, name: "name"}

// For email(required)  
{type: "email", required: true, name: "email"}

// For int(min: 18)
{type: "number", min: 18, name: "age"}

// For enum("a", "b", "c")
{type: "text", pattern: "^(a|b|c)$", name: "role"}
```

**Pros:** Schema drives HTML5 validation, DRY  
**Cons:** Still verbose for error display

### Option C: Field Helper Function

Add a helper that generates input + label + error as a unit:

```parsley
// user-form.part

let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
    email: email(required),
    age: int(min: 18)
}

// Helper function for consistent field rendering
let Field = fn(schema, fieldName, data, errors, opts) {
    let attrs = schema.attrs(fieldName)
    let value = data[fieldName] ?? ""
    let error = errors[fieldName]
    let label = opts.label ?? fieldName
    
    <div class="field">
        <label for={fieldName}>{label}</label>
        <input id={fieldName}
               name={fieldName}
               type={attrs.type ?? "text"}
               minlength={attrs.minlength}
               maxlength={attrs.maxlength}
               min={attrs.min}
               max={attrs.max}
               required={attrs.required}
               value={value}
               class={error ? "error" : ""}/>
        {error && <span class="error">{error}</span>}
    </div>
}

export default = fn(props) {
    let data = props.data ?? {}
    let errors = props.errors ?? {}
    
    <form part-submit="save">
        <Field schema={UserSchema} fieldName="name" data={data} errors={errors} label="Full Name"/>
        <Field schema={UserSchema} fieldName="email" data={data} errors={errors} label="Email Address"/>
        <Field schema={UserSchema} fieldName="age" data={data} errors={errors} label="Age"/>
        <button type="submit">Save</button>
    </form>
}

export save = fn(props) {
    let result = UserSchema.validate(props)
    
    if (result.ok) {
        let _ = @DB.users.insert(result.data)
        <div class="success">User {result.data.name} saved!</div>
    } else {
        default({data: props, errors: result.errors})
    }
}
```

**Pros:** Reusable, consistent, schema-driven  
**Cons:** Custom component per project (could be stdlib)

## 5. Key Design Questions

### Q1: Server-side only, or also client-side?

**Recommendation:** Server-first with progressive enhancement

- Server validation is authoritative (security)
- `schema.attrs()` generates HTML5 attributes (free client-side UX)
- Parts already do server-side re-render on submit
- No custom JavaScript required beyond Parts runtime

### Q2: Generate full forms or bind to inputs?

**Recommendation:** Bind to inputs with optional generation

- `schema.attrs("field")` for HTML5 attributes → flexible
- Optional `Field` helper for convenience → user-defined or stdlib
- Don't force a specific form layout
- Let developers control markup

### Q3: How to handle validation errors in Parts?

**Recommendation:** Pass errors through props, re-render with same view

```parsley
export save = fn(props) {
    let result = UserSchema.validate(props)
    
    if (result.ok) {
        // Success path
        <div>Saved!</div>
    } else {
        // Re-render form with errors and preserved data
        default({data: props, errors: result.errors})
    }
}
```

This pattern:
- Keeps form data via `data: props`
- Passes validation errors via `errors: result.errors`
- Calls `default()` to re-render the form view
- Works within existing Parts architecture

### Q4: Schema-Database binding integration?

Current: Schema defines structure, `db.bind(schema, "table")` creates binding

**Recommendation:** Keep separate but composable

```parsley
let Users = @DB.bind(UserSchema, "users")

export save = fn(props) {
    let result = UserSchema.validate(props)
    if (result.ok) {
        Users.insert(result.data)  // validated data is safe
    }
}
```

### Q5: How to reduce form input verbosity?

The explicit `schema.attrs()` approach is verbose:

```parsley
let nameAttrs = UserSchema.attrs("name")
<input name="name"
       type={nameAttrs.type}
       minlength={nameAttrs.minlength}
       maxlength={nameAttrs.maxlength}
       required={nameAttrs.required}
       value={data.name ?? ""}/>
```

**Option A: Schema.input() method**

Schema returns complete input HTML:

```parsley
{UserSchema.input("name", {value: data.name, class: errors.name ? "invalid" : ""})}
// Outputs: <input name="name" type="text" minlength="2" maxlength="50" required value="..." class="..."/>
```

*Pros:* Simple, no language changes needed  
*Cons:* Less control over markup, harder to add custom attributes

**Option B: User-defined Input component**

```parsley
let Input = fn(schema, name, data, errors, opts) {
    let attrs = schema.attrs(name)
    <input name={name}
           type={attrs.type ?? "text"}
           minlength={attrs.minlength}
           maxlength={attrs.maxlength}
           min={attrs.min}
           max={attrs.max}
           required={attrs.required}
           value={data[name] ?? ""}
           class={errors[name] ? "invalid" : ""}
           placeholder={opts.placeholder}/>
}

// Usage - still needs schema passed each time
<Input schema={UserSchema} name="name" data={data} errors={errors}/>
```

*Pros:* Works today, customizable  
*Cons:* Repetitive schema parameter

**Option C: Form context with @schema (Language Feature)**

```parsley
<form @schema={UserSchema} part-submit="save">
    <input @name="name" value={data.name} class={errors.name ? "invalid" : ""}/>
    <input @name="email" value={data.email}/>
    <input @name="age" value={data.age}/>
    <button type="submit">Save</button>
</form>
```

The `@schema` attribute establishes context. The `@name` attribute triggers schema-aware expansion:
- `<input @name="name"/>` becomes `<input name="name" type="text" minlength="2" maxlength="50" required/>`

**Implementation:** This doesn't require HTML parsing because Parsley already parses tags as AST nodes. During tag evaluation:
1. `<form @schema={X}>` sets schema in evaluation context
2. `<input @name="field">` checks for schema context, expands attributes from schema.attrs()

*Pros:* Very concise, declarative, feels native  
*Cons:* New language feature, "magic" behavior

**Option D: SchemaForm wrapper component**

```parsley
let SchemaForm = fn(schema, action, children) {
    // Wrap children with schema context somehow?
    <form part-submit={action}>
        {children}
    </form>
}

<SchemaForm schema={UserSchema} action="save">
    <input @name="name"/>
</SchemaForm>
```

*Problem:* Parsley doesn't have a mechanism for passing context to children. Would need new language feature anyway.

**Option E: Bound form object**

```parsley
let form = UserSchema.form({data: data, errors: errors})

<form part-submit="save">
    {form.input("name")}
    {form.input("email")}  
    {form.input("age")}
    <button type="submit">Save</button>
</form>
```

The bound form knows about data and errors, so inputs are fully self-contained:
- Includes value from data
- Includes validation attributes from schema
- Includes error class if field has error

*Pros:* Clean API, self-contained  
*Cons:* Less control over HTML structure, new Form type needed

**Recommendation:** Start with **Option A** (schema.input method) or **Option E** (bound form object) for MVP. Consider **Option C** (@schema context) as future enhancement if the pattern proves valuable.

Option C is the most elegant but requires careful design to avoid too much magic. Key questions:
- Should `@name` auto-include `value` from context? (More magic)
- Or just attributes, leaving value explicit? (Less magic, more explicit)
- How to handle error styling?

Minimal magic version of Option C:
```parsley
<form @schema={UserSchema} part-submit="save">
    <!-- @name adds HTML5 attrs from schema, nothing else -->
    <input @name="name" value={data.name} class={errors.name ? "invalid" : ""}/>
</form>
```

Maximum magic version:
```parsley
<form @schema={UserSchema} @data={data} @errors={errors} part-submit="save">
    <!-- @name adds attrs, value, and error class automatically -->
    <input @name="name"/>
    {errors.name && <span class="error">{errors.name}</span>}
</form>
```

## 6. Recommended MVP

Based on Parsley's aesthetic (simplicity, minimalism, completeness, composability):

### Phase 1: Schema Validation API

Ensure `schema.validate()` returns ergonomic result:

```parsley
let result = UserSchema.validate(formData)

result.ok       // true/false
result.data     // cleaned/validated data (if ok)
result.errors   // {fieldName: "Error message", ...} (if not ok)
```

### Phase 2: Bound Form Object

Add `schema.form(opts)` that creates a form helper with data/errors bound:

```parsley
let form = UserSchema.form({data: data, errors: errors})

form.input("name")              // <input name="name" type="text" minlength="2" maxlength="50" required value="Alice" class="invalid"/>
form.input("name", {class: "custom"})  // Override/add attributes
form.error("name")              // "Name must be at least 2 characters" or null
form.value("name")              // "Alice"
form.hasError("name")           // true
```

The bound form:
- Knows the schema (for HTML5 attributes)
- Knows the data (for values)
- Knows the errors (for error class + messages)

### Phase 3: Complete Example with Bound Form

```parsley
// user-form.part

let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
    email: email(required),
    age: int(min: 18)
}

export default = fn(props) {
    let data = props.data ?? {}
    let errors = props.errors ?? {}
    let form = UserSchema.form({data: data, errors: errors})
    
    <form part-submit="save">
        <div class="field">
            <label>Name</label>
            {form.input("name")}
            {form.error("name") && <span class="error">{form.error("name")}</span>}
        </div>
        
        <div class="field">
            <label>Email</label>
            {form.input("email")}
            {form.error("email") && <span class="error">{form.error("email")}</span>}
        </div>
        
        <div class="field">
            <label>Age</label>
            {form.input("age")}
            {form.error("age") && <span class="error">{form.error("age")}</span>}
        </div>
        
        <button type="submit">Save User</button>
    </form>
}

export save = fn(props) {
    let result = UserSchema.validate(props)
    
    if (result.ok) {
        let Users = @DB.bind(UserSchema, "users")
        Users.insert(result.data)
        
        <div class="success">
            <p>User {result.data.name} saved successfully!</p>
            <button part-click="default">Add Another</button>
        </div>
    } else {
        default({data: props, errors: result.errors})
    }
}
```

### Phase 4 (Future): @schema Context Syntax

If the bound form pattern proves valuable, consider the more magical but concise syntax:

```parsley
export default = fn(props) {
    let data = props.data ?? {}
    let errors = props.errors ?? {}
    
    <form @schema={UserSchema} @data={data} @errors={errors} part-submit="save">
        <div class="field">
            <label>Name</label>
            <input @name="name"/>
            {errors.name && <span class="error">{errors.name}</span>}
        </div>
        
        <div class="field">
            <label>Email</label>
            <input @name="email"/>
            {errors.email && <span class="error">{errors.email}</span>}
        </div>
        
        <div class="field">
            <label>Age</label>
            <input @name="age"/>
            {errors.age && <span class="error">{errors.age}</span>}
        </div>
        
        <button type="submit">Save User</button>
    </form>
}
```

**Implementation notes for @schema:**

The architecture for this is favorable. Key insights from the evaluator:

1. **Tags are AST nodes**: Parsley parses `<form>` and `<input>` as `ast.TagPairExpression` and `ast.TagLiteral` nodes during parsing, not as opaque strings. The evaluator walks the AST tree.

2. **Evaluation is tree-walking**: `evalTagPair` → `evalStandardTagPair` → `evalTagContents` passes the `*Environment` through. Each child node is evaluated in sequence with access to the environment.

3. **Environment already supports context**: `Environment` is a scoped store that chains to outer environments via `NewEnclosedEnvironment()`. It already carries contextual state like `BasilCtx`, `FragmentCache`, `DevMode`.

4. **Spread syntax exists**: Tags already support `{...dict}` spread for attributes, evaluated at render time via `node.Spreads`.

**Implementation approach:**

```go
// In evalStandardTagPair for <form>:
if node.Name == "form" {
    // Check for @schema prop
    if schemaExpr, ok := props["@schema"]; ok {
        schema := Eval(schemaExpr, env)
        // Create enclosed environment with form context
        formEnv := NewEnclosedEnvironment(env)
        formEnv.Set("__form_schema__", schema)
        formEnv.Set("__form_data__", Eval(props["@data"], env))
        formEnv.Set("__form_errors__", Eval(props["@errors"], env))
        // Evaluate children in formEnv instead of env
        contentsObj := evalTagContents(node.Contents, formEnv)
    }
}

// In evalTagLiteral for <input @name="field">:
if hasAtName {
    if schema, ok := env.Get("__form_schema__"); ok {
        // Expand @name to full attributes from schema
        attrs := schema.attrs(fieldName)
        // Merge with explicit attrs, @name attrs have lower precedence
    }
}
```

**Context is scoped naturally**: When `evalTagContents` walks the children of `<form>`, it passes the enclosed `formEnv`. When the form closes, the context disappears. No special cleanup needed.

**Magic level:** Medium - the expansion is predictable and the pattern is declarative. Users can still add custom attributes alongside `@name`.

### Alternative: Even More Concise Field Helper

If we want the error message included:

```parsley
let form = UserSchema.form({data: data, errors: errors})

// form.field() returns input + error message together
<div class="field">
    <label>Name</label>
    {form.field("name")}
</div>

// Outputs:
// <input name="name" type="text" minlength="2" maxlength="50" required value="Alice" class="invalid"/>
// <span class="error">Name must be at least 2 characters</span>
```

Or with label included:

```parsley
{form.field("name", {label: "Full Name"})}

// Outputs:
// <label for="name">Full Name</label>
// <input id="name" name="name" type="text" .../>
// <span class="error">...</span>
```

This is similar to Django's `{{ form.field }}` approach.

## 7. Implementation Considerations

### 7.1 Go Changes Needed

1. Add `Schema.Attrs(fieldName)` method returning `map[string]interface{}`
2. Ensure `ValidateSchemaFields` returns `{ok, data, errors}` structure
3. Expose `.validate()` and `.attrs()` methods in Parsley runtime

### 7.2 Parsley Changes Needed

1. Expose schema methods: `.validate()`, `.attrs()`
2. Ensure conditional attributes work: `required={attrs.required}`
3. Consider spread syntax for attributes: `<input {...attrs}/>` (future)

### 7.3 Documentation Needed

- Guide: "Form Validation with Schemas"
- Reference: Schema validation methods
- Example: Complete CRUD form with validation

## 8. Future Enhancements (Not MVP)

- **Spread attributes:** `<input {...schema.attrs("name")}/>` for cleaner syntax
- **Field component in stdlib:** `<@std/Field schema={...} name="..."/>`
- **Form generation:** `schema.toForm(opts)` generates entire form
- **Custom validators:** User-defined validation functions
- **Async validation:** Check uniqueness against DB
- **Multi-step forms:** Wizard/stepper pattern with Part state
- **File uploads:** File validation integrated with schema

## 9. Alternative Direction: Record Type (Phoenix Changeset Pattern)

### 9.1 The Insight

In the `@schema` + `@data` + `@errors` approach, we pass three separate values that are conceptually one thing: **the form state**. What if data could carry its schema?

```parsley
// Instead of three separate bindings...
<form @schema={UserSchema} @data={data} @errors={errors}>

// ...what if data already knew its schema?
<form @bind={user}>
```

This is essentially **Phoenix's changeset pattern**: a single value that combines schema + data + validation state.

### 9.2 The Record Type

A new type that wraps data with its schema:

```parsley
// Schema with field metadata
let UserSchema = @schema {
    name: string(min: 2, max: 50, required, label: "Full Name"),
    email: email(required, label: "Email Address", placeholder: "you@example.com"),
    age: int(min: 18, label: "Age")
}

// Create empty record (for "new" forms)
let user = UserSchema.new()           // Record with schema, empty data, no errors

// Create record from data (for "edit" forms)
let user = UserSchema.new({name: "Alice", email: "alice@example.com", age: 30})

// Record properties
user.name                             // "Alice" (data access)
user.schema                           // The schema itself
user.errors                           // {} (no errors yet)
user.ok                               // true (no errors)
user.dirty                            // false (not modified)
```

### 9.3 Validation Returns Records

```parsley
// Validation produces a record with errors attached
let result = UserSchema.validate(formData)

result.ok           // false
result.name         // "A" (the submitted value)
result.errors       // {name: "Must be at least 2 characters"}
result.errors.name  // "Must be at least 2 characters"

// The record IS the form state - pass it directly
default({form: result})
```

### 9.4 Database Returns Records

```parsley
let Users = @DB.bind(UserSchema, "users")

// Find returns a record (data + schema)
let user = Users.find(1)
user.name           // "Alice"
user.schema         // UserSchema

// Insert/update accepts records
Users.insert(validatedRecord)
Users.update(validatedRecord)
```

### 9.5 Form Binding with Records

With records, form binding becomes trivial:

```parsley
export default = fn(props) {
    let form = props.form ?? UserSchema.new()
    
    <form @bind={form} part-submit="save">
        <Field @name="name"/>
        <Field @name="email"/>
        <Field @name="age"/>
        <button type="submit">Save User</button>
    </form>
}

export save = fn(props) {
    let form = UserSchema.validate(props)
    
    if (form.ok) {
        let Users = @DB.bind(UserSchema, "users")
        Users.insert(form)
        <div class="success">Saved {form.name}!</div>
    } else {
        default({form: form})
    }
}
```

The `<Field @name="name"/>` component has access to everything:
- **Label text**: from `form.schema.fields.name.label`
- **Input type/validation**: from `form.schema.fields.name`  
- **Current value**: from `form.name`
- **Error message**: from `form.errors.name`

### 9.6 Component Expansion

```parsley
<Field @name="name"/>

// Expands to (conceptually):
<div class="field">
    <label for="name">{form.schema.fields.name.label}</label>
    <input id="name" 
           name="name"
           type={form.schema.fields.name.htmlType}
           minlength={form.schema.fields.name.min}
           maxlength={form.schema.fields.name.max}
           required={form.schema.fields.name.required}
           placeholder={form.schema.fields.name.placeholder}
           value={form.name}
           class={form.errors.name ? "invalid" : ""}/>
    {form.errors.name && <span class="error">{form.errors.name}</span>}
</div>
```

### 9.7 Comparison to Prior Art

| Framework | Concept | Contains |
|-----------|---------|----------|
| **Phoenix** | Changeset | schema + data + errors + changes |
| **Django** | BoundForm | form class + data + errors |
| **Rails** | ActiveRecord | schema (via model) + data + errors |
| **Parsley** | Record (proposed) | schema + data + errors |

Phoenix changesets are the closest analog. Key insight: the changeset is **the unit of form state** that flows through the system.

### 9.8 Implementation Considerations

**New Record type in Go:**

```go
type Record struct {
    Schema *DSLSchema
    Data   map[string]Object
    Errors map[string]string
    // Optionally: Changes map[string]Object for tracking modifications
}

func (r *Record) Type() ObjectType { return RECORD_OBJ }
func (r *Record) Get(key string) Object { return r.Data[key] }
func (r *Record) HasError(key string) bool { return r.Errors[key] != "" }
func (r *Record) IsOk() bool { return len(r.Errors) == 0 }
```

**Schema methods:**

```go
// Schema.new(data?) → Record
// Schema.validate(data) → Record (with errors populated)
```

**Property access:**

Records would need special handling for property access:
- `record.name` → `record.Data["name"]`
- `record.schema` → `record.Schema`
- `record.errors` → Dictionary of errors
- `record.ok` → Boolean

**Form context:**

`<form @bind={record}>` sets `__form_record__` in environment. Children access via context.

### 9.9 Trade-offs

**Pros:**
- Single value captures entire form state
- Natural flow: schema → record → validate → record with errors → render
- Database integration is seamless (queries return records)
- Less ceremony in templates
- Similar to proven patterns (Phoenix, Django)

**Cons:**
- New type (adds complexity to language)
- Property access has dual meaning (data vs metadata)
- Records are "heavier" than plain dictionaries
- Migration path for existing `@schema` code

### 9.10 Alternative: Lightweight "Form" Wrapper

Instead of a full Record type, a lighter approach:

```parsley
// Form wraps data + schema + errors without being a new type
let form = UserSchema.form({
    data: data,
    errors: errors
})

// form is still a Dictionary, but with special structure
form.data.name      // "Alice"
form.errors.name    // "Too short"
form.schema         // UserSchema
form.field("name")  // Returns {value, error, attrs, label, ...}
```

This is less elegant but doesn't require a new type. The `form.field()` method provides the bundled access.

### 9.11 Questions to Resolve

1. **New type or wrapper?** Record type is cleaner but bigger change
2. **Property access semantics?** `record.name` vs `record.data.name`
3. **Mutability?** Are records immutable (new record on change) or mutable?
4. **Schema field metadata?** What metadata do we need? (label, placeholder, help text, ...)
5. **Database integration scope?** Does `db.bind` return records, or is that separate?
6. **Backward compatibility?** How do plain dicts interact with record-expecting code?
7. **How to handle i18n for labels?** See section 9.11.1

#### 9.11.1 The i18n Problem with Static Labels

If labels are defined in the schema:

```parsley
let UserSchema = @schema {
    name: string(min: 2, max: 50, required, label: "Full Name"),
}
```

The label "Full Name" is bound at schema definition time (essentially compile-time). This makes translation problematic:

- You'd need separate schemas per language
- Or override every label at render time anyway
- Schema becomes a mix of validation logic and display concerns

**Solution Options:**

**Option A: Labels are i18n keys, not display strings**

```parsley
let UserSchema = @schema {
    name: string(min: 2, max: 50, required, label: "user.name"),  // i18n key
}

// Resolution happens at render time
form.label("name")  // Internally calls @i18n("user.name")
```

*Pros:* Single schema, labels resolved at runtime
*Cons:* Requires i18n infrastructure, keys look like labels but aren't

**Option B: No labels in schema, provide at render time**

```parsley
// Schema is purely validation
let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
}

// Labels passed to Field component
<Field @name="name" label={@i18n("user.name")}/>

// Or: labels dictionary on form
<form @bind={form} @labels={{name: @i18n("user.name")}}>
```

*Pros:* Clean separation (schema = validation, template = display)
*Cons:* More verbose, labels repeated across forms using same schema

**Option C: Label resolver function on form**

```parsley
let form = UserSchema.form({
    data: data,
    errors: errors,
    labels: fn(field) { @i18n("user." + field) }
})

// Or with a namespace shorthand
let form = UserSchema.form({
    data: data,
    errors: errors,
    labelNamespace: "user"  // Implies fn(f) { @i18n("user." + f) }
})

form.label("name")  // Resolves via the function → "Full Name" (translated)
```

*Pros:* Centralized, works with any i18n system
*Cons:* Another thing to pass to form()

**Option D: Convention-based default + override**

```parsley
// No label in schema → derive from field name
// "name" → "Name", "firstName" → "First Name", "email_address" → "Email Address"

// Override when needed
<Field @name="name" label={@i18n("user.name")}/>
```

*Pros:* Zero config for prototyping, explicit override for i18n
*Cons:* Magic derivation may not match desired labels

**Recommendation:** Combine **Option B** (no labels in schema) with **Option C** (label resolver):

```parsley
// Schema is pure validation
let UserSchema = @schema {
    name: string(min: 2, max: 50, required),
    email: email(required),
}

// Form with label resolver
let form = UserSchema.form({
    data: data,
    errors: errors,
    labels: fn(field) { @i18n("user." + field) }
})

// <Field> uses form's label resolver by default
<Field @name="name"/>  // Label from form.labels("name")

// Explicit override when needed
<Field @name="name" label="Custom Label"/>
```

This gives us:
- **Pure schemas** (validation only, no display concerns)
- **Centralized i18n** (one resolver per form)
- **Flexibility** (override per-field when needed)
- **Separation of concerns** (schema authors don't think about i18n)

### 9.12 Recommendation

**Start with the lightweight form wrapper (9.10)** as it doesn't require language changes:

```parsley
let form = UserSchema.form({data: props, errors: {}})

<form @bind={form} part-submit="save">
    <Field @name="name"/>
</form>
```

If this pattern proves valuable, **graduate to a full Record type** in a future release. This lets us:
1. Validate the UX before committing to language changes
2. Gather feedback on what metadata fields need
3. Design the database integration carefully

## 10. Decision Summary

| Question | Recommendation |
|----------|---------------|
| Server vs client validation | Server-first + HTML5 attributes |
| Generate forms vs bind inputs | Bind inputs (attrs helper), optional Field component |
| Error handling | Pass errors via props, re-render same view |
| Schema-DB coupling | Keep separate, compose at usage |
| MVP scope | `validate()` + `attrs()` + documentation |
| Form state pattern | Start with `schema.form()` wrapper, consider Record type later |

## 11. Next Steps

1. Review this design with human
2. If approved, create FEAT-XXX for implementation
3. Implement in phases:
   - Phase 1: Polish `schema.validate()` API
   - Phase 2: Add `schema.attrs()` method  
   - Phase 3: Add `schema.form()` wrapper with `@bind` support
   - Phase 4: Add schema field metadata (label, placeholder)
   - Phase 5: Implement `<Field @name="..."/>` component
   - Phase 6: (Future) Consider Record type if pattern proves valuable
