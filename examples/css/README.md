# Basil CSS Examples

This directory contains example CSS files for use with Basil and Parsley projects.

## Files

### `basil-supplement.css`

A minimal CSS file (~150 lines) that extends [Pico CSS](https://picocss.com) to support Prelude components that Pico doesn't style natively.

**Covers:**
- Skip links (screen reader accessible)
- Toast notifications (positioning and type indicators)
- Pagination (flexbox layout)
- Error summaries (focus styles and visual indicators)
- Screen reader utility class (`.sr-only`)

## Usage

### With Pico CSS (Recommended)

Include Pico CSS first, then the supplement:

```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
<link rel="stylesheet" href="/css/basil-supplement.css">
```

Or in a Parsley template:

```parsley
<Page title="My Page">
    <CSS href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css"/>
    <CSS href="/css/basil-supplement.css"/>
    
    <main class="container">
        // Your content
    </main>
</Page>
```

### Classless Pico

For simpler projects, use the classless version:

```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.classless.min.css">
<link rel="stylesheet" href="/css/basil-supplement.css">
```

## Why Pico CSS?

Basil Prelude components output semantic HTML designed to work with Pico CSS because:

1. **Semantic HTML first** — Pico styles native HTML elements, no class soup required
2. **Tiny footprint** — 3.5KB gzipped for the full framework
3. **Dark mode built-in** — Respects `prefers-color-scheme` automatically
4. **Easy to override** — Uses CSS custom properties throughout
5. **Already in Basil** — The devtools use Pico, so it's battle-tested

## Customization

The supplement uses Pico's CSS custom properties, so colors adapt automatically:

```css
/* These adapt to Pico's theme */
var(--pico-primary)
var(--pico-del-color)      /* Error/danger color */
var(--pico-ins-color)      /* Success color */
var(--pico-background-color)
```

To customize, override Pico's properties in your own CSS:

```css
:root {
    --pico-primary: #your-brand-color;
}
```

## Component Patterns

The supplement expects specific HTML patterns. See the [Pico Compatibility Design Doc](../../../work/design/DESIGN-prelude-pico-compatibility.md) for detailed component HTML structures.

### Quick Examples

**Toast:**
```html
<aside id="toasts" data-position="top-right">
    <article role="status" data-type="success">
        <p>Changes saved!</p>
    </article>
</aside>
```

**Pagination:**
```html
<nav aria-label="Pagination">
    <ul>
        <li><a href="?page=1">1</a></li>
        <li><a href="?page=2" aria-current="page">2</a></li>
        <li><a href="?page=3">3</a></li>
    </ul>
</nav>
```

**Error Summary:**
```html
<aside role="alert" tabindex="-1">
    <h2>There is a problem</h2>
    <ul>
        <li><a href="#email">Enter a valid email</a></li>
    </ul>
</aside>
```

## License

MIT — same as Basil and Pico CSS.