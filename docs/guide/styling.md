# Styling Prelude Components

Prelude components output semantic HTML that works with any CSS framework—or no framework at all. This guide covers the recommended approach using Pico CSS.

## Quick Start

Add Pico CSS and the Basil supplement to your page:

```parsley
<Page lang="en" title="My App" head={
    <>
        <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css"/>
        <link rel="stylesheet" href="/css/basil-supplement.css"/>
    </>
}>
    <main id="main" class="container">
        <h1>"Welcome"</h1>
    </main>
</Page>
```

**Note:** Put `id="main"` on your `<main>` element so the skip link works correctly.

---

## Why Pico CSS?

| Feature | Benefit |
|---------|---------|
| Semantic | Styles HTML elements directly—minimal classes needed |
| Small | ~3.5KB gzipped |
| Accessible | Built-in focus states, color contrast |
| Dark mode | Automatic via `prefers-color-scheme` |
| Already in Basil | DevTools use Pico at `/__/css/pico.min.css` |

---

## The Basil Supplement

Pico CSS covers most components, but a few need extra styles. Copy `examples/css/basil-supplement.css` to your project's static files.

**What the supplement provides:**

| Component | Styles |
|-----------|--------|
| `SkipLink` | Visually hidden until focused |
| `Toast` / `Toasts` | Positioning and type-based border colors |
| `Pagination` | Flexbox layout for page links |
| `ErrorSummary` | Alert border and focus ring |

---

## Component Examples

### Forms

Prelude form components work directly with Pico:

```parsley
<Form action="/register" method="POST">
    <TextField name="email" label="Email" type="email" required={true}/>
    <TextField name="password" label="Password" type="password" required={true}/>
    <TextareaField name="bio" label="Bio" hint="Tell us about yourself"/>
    <SelectField 
        name="country" 
        label="Country" 
        options={["USA", "UK", "Canada"]}
        placeholder="Select a country"
    />
    <Button type="submit">"Register"</Button>
</Form>
```

**Validation states:** Add `error` prop to show validation errors. Pico automatically styles inputs with `aria-invalid="true"`.

```parsley
<TextField 
    name="email" 
    label="Email" 
    value={form.email}
    error={if (errors.email) { errors.email } else { null }}
/>
```

### Error Summary

Show all form errors at the top with links to each field:

```parsley
<ErrorSummary errors={[
    {field: "email", message: "Enter a valid email address"},
    {field: "password", message: "Password must be at least 8 characters"}
]}/>
```

### Dialog (Modal)

Native HTML5 dialog with Pico styling:

```parsley
<Dialog id="confirm-delete" title="Delete Item" footer={
    <>
        <Button class="secondary" onclick="this.closest('dialog').close()">"Cancel"</Button>
        <Button onclick="deleteItem()">"Delete"</Button>
    </>
}>
    <p>"Are you sure you want to delete this item? This action cannot be undone."</p>
</Dialog>

// Open with JavaScript:
<Button onclick="document.getElementById('confirm-delete').showModal()">
    "Delete"
</Button>
```

### Accordion

Native HTML5 exclusive accordion—only one section open at a time:

```parsley
<Accordion name="faq" items={[
    {title: "What is Basil?", content: <p>"A web framework for Parsley."</p>},
    {title: "What is Parsley?", content: <p>"A language for building web pages."</p>},
    {title: "Is it production ready?", content: <p>"Yes!"</p>}
]}/>
```

For individual expandable sections:

```parsley
<Details title="Click to expand">
    <p>"This content is hidden by default."</p>
</Details>

<Details title="Open by default" open={true}>
    <p>"This content is visible immediately."</p>
</Details>
```

### Toasts

Notification messages with positioning:

```parsley
<Toasts position="top-right">
    <Toast message="Changes saved" type="success"/>
    <Toast message="Connection lost" type="error"/>
</Toasts>
```

**Toast types:** `info` (default), `success`, `warning`, `error`

**Positions:** `top-right` (default), `top-left`, `top-center`, `bottom-right`, `bottom-left`, `bottom-center`

### Pagination

Page navigation with accessibility:

```parsley
<Pagination 
    current={3} 
    total={150} 
    perPage={10} 
    href="/articles?page={page}"
/>
```

**Props:**
- `current` — Current page number
- `total` — Total item count
- `perPage` — Items per page (default: 20)
- `href` — URL template with `{page}` placeholder
- `window` — Pages to show around current (default: 2)

### Breadcrumb

Navigation with Schema.org markup:

```parsley
<Breadcrumb items={[
    {label: "Home", href: "/"},
    {label: "Products", href: "/products"},
    {label: "Shoes"}  // No href = current page
]}/>
```

---

## Layout with Pico

Pico provides a `.container` class for centered, max-width content:

```parsley
<Page lang="en" title="My Site">
    <header class="container">
        <Nav>...</Nav>
    </header>
    <main id="main" class="container">
        <h1>"Page Title"</h1>
        contents
    </main>
    <footer class="container">
        <p>"© 2025"</p>
    </footer>
</Page>
```

For full-width sections, omit the class:

```parsley
<section style="background: var(--pico-secondary-background)">
    <div class="container">
        <h2>"Featured"</h2>
    </div>
</section>
```

---

## Dark Mode

Pico automatically supports dark mode via `prefers-color-scheme`. To force a theme:

```parsley
<html lang="en" data-theme="dark">
```

Or let users toggle:

```parsley
<html lang="en" data-theme={userPreference ?? "auto"}>
```

---

## Button Variants

Pico provides button variants via classes:

```parsley
<Button>"Primary"</Button>
<Button class="secondary">"Secondary"</Button>
<Button class="contrast">"Contrast"</Button>
<Button class="outline">"Outline"</Button>
<Button class="secondary outline">"Secondary Outline"</Button>
```

---

## Without Pico (Classless)

Prelude components output semantic HTML that's readable without any CSS. If you prefer a different framework or custom styles:

1. Components use standard HTML elements (`<label>`, `<input>`, `<nav>`, `<dialog>`, etc.)
2. ARIA attributes are included for accessibility
3. No framework-specific classes are required

The supplement CSS is optional—it only adds polish for components Pico doesn't cover.

---

## Migration from Previous Versions

If you were using custom CSS for Prelude components:

### Form Fields

**Before:** Components wrapped in `<div class="field">` with `<p class="field-hint">`.

**After:** No wrapper div. Hints/errors use `<small>` elements.

```parsley
// Output is now:
// <label for="email">Email</label>
// <input type="email" id="email" name="email"/>
// <small id="email-hint">We'll never share this</small>
```

### SkipLink

**Before:** Inline `<style>` tag with `#skip` styles.

**After:** Uses `class="skip-link"`. Add `basil-supplement.css` for styling.

### Page Body ID

**Before:** `<body id="main">` by default.

**After:** No default body ID. Put `id="main"` on your `<main>` element:

```parsley
<Page lang="en" title="My Site">
    <main id="main">  // Add this for skip link target
        contents
    </main>
</Page>
```

---

## Resources

- [Pico CSS Documentation](https://picocss.com/docs)
- [Pico CSS Examples](https://picocss.com/examples)
- [Basil Supplement CSS](../../examples/css/basil-supplement.css)
- [WCAG Accessibility Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)