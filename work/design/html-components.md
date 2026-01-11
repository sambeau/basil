# HTML Components Library

Design document for Basil's accessible HTML component library.

## Philosophy

**"Just enough to be more convenient than looking it up on MDN."**

These components render semantic, accessible HTML with correct ARIA attributes, proper structure, and modern best practices. They're unstyled by default - just the right HTML that developers would otherwise get wrong or have to look up.

Goals:
- Every component should be more convenient than writing raw HTML
- Accessibility baked in, not bolted on
- Server-side rendered, progressive enhancement
- No styling opinions - works with any CSS approach
- Minimal JS, only where it adds genuine value

Non-goals:
- Replacing simple HTML tags that are already easy to use
- Complex client-side interactivity
- Styling or theming

---

## Component Categories

### 1. Skip These (Use Native HTML)

These tags gain nothing from being Parsley components. Just use them directly:

```
<p>, <span>, <div>, <strong>, <em>, <small>, <b>, <i>, <u>, <s>
<sup>, <sub>, <br/>, <hr>, <pre>, <code>
<ul>, <ol>, <li>, <dl>, <dt>, <dd>
<thead>, <tbody>, <tfoot>, <tr>, <td>
<header>, <footer>, <main>, <article>, <section>, <aside>
<h1>, <h2>, <h3>, <h4>, <h5>, <h6>
<address>, <mark>, <kbd>, <samp>, <var>, <wbr>
```

### 2. High-Level Compound Components (Most Value)

These wrap multiple elements and save significant boilerplate:

| Component | What It Generates |
|-----------|-------------------|
| `<Page>` | `<!DOCTYPE>` + `<html>` + `<head>` + `<body>` + `<SkipLink>` |
| `<Head>` | All meta tags (charset, viewport, OG, Twitter, favicon) |
| `<Form>` | `<form>` with CSRF, error summary, auto-disable on submit |
| `<TextField>` | Field wrapper + label + input + hint + error |
| `<TextareaField>` | Field wrapper + label + textarea + hint + error |
| `<SelectField>` | Field wrapper + label + select + hint + error |
| `<RadioGroup>` | Fieldset + legend + radio inputs |
| `<CheckboxGroup>` | Fieldset + legend + checkbox inputs |
| `<DataTable>` | Table + caption + thead + tbody with proper th scope |
| `<Nav>` | `<nav>` landmark with aria-label |
| `<Breadcrumb>` | nav + ol + Schema.org markup |
| `<Tabs>` | TabList + Tabs + TabPanels with ARIA + keyboard nav |
| `<Dialog>` | `<dialog>` with focus trap and escape handling |
| `<Blockquote>` | figure + blockquote + figcaption + cite |
| `<Figure>` | figure + figcaption |

### 3. Simple Enhanced Elements (Medium Value)

Single elements that add required attributes or smart defaults:

| Component | Enhancement |
|-----------|-------------|
| `<Img>` | Requires `alt`, adds `loading="lazy"` |
| `<Iframe>` | Requires `title`, adds `loading="lazy"` |
| `<A>` | Auto `rel="noopener noreferrer"` for `target="_blank"` |
| `<Time>` | Formats `datetime` attribute from value |
| `<LocalTime>` | Client-side localization of UTC datetime |
| `<TimeRange>` | Smart display of datetime spans |
| `<RelativeTime>` | "2 hours ago" / "in 3 days" with optional auto-refresh |
| `<Abbr>` | Requires `title` (expansion) |
| `<Button>` | Defaults to `type="button"` (not submit) |
| `<Th>` | Infers `scope` from position |

### 4. Convenience/Utility

| Component | Use |
|-----------|-----|
| `<SrOnly>` | Screen reader only text (visually hidden) |
| `<SkipLink>` | Skip to main content link |
| `<Icon>` | SVG wrapper with `aria-hidden` + sr-only label |

---

## Component Specifications

### Page Structure

#### `<Page>`

Complete HTML document wrapper with automatic asset inclusion.

```parsley
<Page lang="en" title="My Site" description="About my site">
  <Nav>...</Nav>
  <main>...</main>
  <footer>...</footer>
</Page>
```

Renders:
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>My Site</title>
  <meta name="description" content="About my site"/>
  <link rel="stylesheet" href="/__site.css?v=abc123">
</head>
<body>
  <a href="#main" class="skip-link">Skip to main content</a>
  <nav>...</nav>
  <main id="main">...</main>
  <footer>...</footer>
  <script src="/__site.js?v=def456"></script>
  <script src="/__/js/basil.abc1234.js"></script>
</body>
</html>
```

Props:
- `lang` - Language code (default: "en")
- `title` - Page title (required)
- `description` - Meta description
- `class` - Body class
- `id` - Body id
- `head` - Additional `<head>` content (extra CSS, meta tags, etc.)
- `noBasilJS` - Omit basil.js script (for pages that don't use enhanced components)

Automatic inclusions:
- `<CSS/>` - Site CSS bundle (all `.css` files from handlers directory)
- `<Javascript/>` - Site JS bundle (all `.js` files from handlers directory)
- `<BasilJS/>` - Basil component enhancement JavaScript
- `<SkipLink/>` - Accessibility skip link


#### `<Head>`

All the meta tags nobody wants to remember. Includes Open Graph, Twitter Cards, and favicons.

```parsley
<Head 
  title="My Page"
  description="About things"
  image="/og-image.png"
  url="https://example.com/page"
  type="article"
  author="Sam Phillips"
  published={@2024-01-15}
  twitter="@handle"
/>
```

Renders:
```html
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>My Page</title>
  
  <!-- SEO -->
  <meta name="description" content="About things"/>
  <meta name="author" content="Sam Phillips"/>
  
  <!-- Canonical -->
  <link rel="canonical" href="https://example.com/page"/>
  
  <!-- Open Graph -->
  <meta property="og:title" content="My Page"/>
  <meta property="og:description" content="About things"/>
  <meta property="og:image" content="/og-image.png"/>
  <meta property="og:url" content="https://example.com/page"/>
  <meta property="og:type" content="article"/>
  <meta property="article:published_time" content="2024-01-15"/>
  <meta property="article:author" content="Sam Phillips"/>
  
  <!-- Twitter -->
  <meta name="twitter:card" content="summary_large_image"/>
  <meta name="twitter:site" content="@handle"/>
  <meta name="twitter:creator" content="@handle"/>
  <meta name="twitter:title" content="My Page"/>
  <meta name="twitter:description" content="About things"/>
  <meta name="twitter:image" content="/og-image.png"/>
  
  <!-- Favicon -->
  <link rel="icon" href="/favicon.ico" sizes="any"/>
  <link rel="icon" href="/favicon.svg" type="image/svg+xml"/>
  <link rel="apple-touch-icon" href="/apple-touch-icon.png"/>
  
  <!-- Site CSS -->
  <link rel="stylesheet" href="/__site.css?v=abc123">
</head>
```

Props:
- `title` - Page title (required)
- `description` - Meta description
- `image` - Open Graph/Twitter image URL
- `url` - Canonical URL
- `type` - og:type (default: "website", use "article" for blog posts)
- `author` - Author name
- `published` - Published date (for articles, expects datetime value)
- `modified` - Modified date (for articles)
- `twitter` - Twitter handle (with @)
- `favicon` - Custom favicon path (default: /favicon.ico)
- `faviconSvg` - SVG favicon path (default: /favicon.svg)
- `appleTouchIcon` - Apple touch icon path (default: /apple-touch-icon.png)
- `noIndex` - Add robots noindex meta tag
- `contents` - Additional head content (extra meta tags, scripts, etc.)

Automatic inclusions:
- `<CSS/>` - Site CSS bundle

**Note:** `<Head>` is for when you need full control over the `<head>` section. For most cases, use `<Page>` which wraps everything including `<head>`.

---

### Forms

#### `<Form>`

Form with CSRF protection, error handling, and submit protection.

```parsley
<Form action="/register" method=POST confirm="Submit registration?">
  <TextField name="email" label="Email" type=email required/>
  <Button type=submit>Register</Button>
</Form>
```

Renders:
```html
<form action="/register" method="POST" data-confirm="Submit registration?">
  <input type="hidden" name="_csrf" value="..."/>
  <!-- fields -->
</form>
```

Props:
- `action` - Form action URL
- `method` - HTTP method (default: POST)
- `confirm` - Confirmation message before submit
- `class`, `id` - Standard hooks

Automatic behaviors:
- CSRF token injected
- Submit buttons disabled on submit (prevents double-click)
- Error summary rendered if `errors` prop provided

#### `<TextField>`

Complete form field with label, input, hint, and error.

```parsley
<TextField 
  name="email" 
  label="Email address" 
  type=email
  value={form.email}
  hint="We'll never share your email"
  error={errors.email}
  required
/>
```

Renders:
```html
<div class="field">
  <label for="field-email">
    Email address
    <span class="field-required" aria-hidden="true">*</span>
  </label>
  <input 
    type="email"
    id="field-email"
    name="email"
    value="..."
    required
    aria-required="true"
    aria-describedby="field-email-hint field-email-error"
    aria-invalid="true"
  />
  <p id="field-email-hint" class="field-hint">We'll never share your email</p>
  <p id="field-email-error" class="field-error" role="alert">Invalid email address</p>
</div>
```

Props:
- `name` - Input name (required)
- `label` - Label text (required)
- `type` - Input type (default: text)
- `value` - Current value
- `hint` - Help text
- `error` - Error message
- `required` - Required field
- `class` - Wrapper class

#### `<TextareaField>`

Like TextField but for multi-line text.

```parsley
<TextareaField 
  name="bio" 
  label="Biography"
  value={form.bio}
  maxlength=280
  counter
  autoresize
/>
```

Additional props:
- `counter` - Show character count
- `autoresize` - Grow with content

#### `<SelectField>`

Dropdown select with label and error handling.

```parsley
<SelectField 
  name="country"
  label="Country"
  value={form.country}
  options={countries}
  valueKey="code"
  labelKey="name"
  autosubmit
/>
```

Props:
- `options` - Array of options
- `valueKey` - Property for option value (default: "value")
- `labelKey` - Property for option label (default: "label")
- `autosubmit` - Submit form on change

#### `<RadioGroup>`

Group of radio buttons with proper fieldset/legend.

```parsley
<RadioGroup 
  name="size"
  label="Select size"
  value={form.size}
  options={[
    {value: "s", label: "Small"},
    {value: "m", label: "Medium"},
    {value: "l", label: "Large"}
  ]}
/>
```

Renders:
```html
<fieldset class="radio-group">
  <legend>Select size</legend>
  <label><input type="radio" name="size" value="s"/> Small</label>
  <label><input type="radio" name="size" value="m" checked/> Medium</label>
  <label><input type="radio" name="size" value="l"/> Large</label>
</fieldset>
```

#### `<CheckboxGroup>`

Group of checkboxes for multi-select.

```parsley
<CheckboxGroup
  name="toppings"
  label="Select toppings"
  values={form.toppings}
  options={toppingOptions}
/>
```

#### `<Checkbox>`

Single checkbox for boolean values.

```parsley
<Checkbox name="agree" label="I agree to the terms" checked={form.agree}/>
```

---

### Navigation

#### `<Nav>`

Navigation landmark with proper labeling.

```parsley
<Nav label="Main navigation">
  <a href="/">Home</a>
  <a href="/about">About</a>
</Nav>
```

Renders:
```html
<nav aria-label="Main navigation">
  <a href="/">Home</a>
  <a href="/about">About</a>
</nav>
```

#### `<Breadcrumb>`

Breadcrumb navigation with Schema.org markup.

```parsley
<Breadcrumb items={[
  {label: "Home", href: "/"},
  {label: "Products", href: "/products"},
  {label: "Shoes"}
]}/>
```

Renders full breadcrumb with Schema.org BreadcrumbList markup and `aria-current="page"` on the last item.

#### `<SkipLink>`

Accessibility skip link (usually in Page component).

```parsley
<SkipLink/>
```

Renders:
```html
<a href="#main" class="skip-link">Skip to main content</a>
```

---

### Tables

#### `<DataTable>`

Data table with proper semantics.

```parsley
<DataTable
  caption="User list"
  columns={["Name", "Email", "Role"]}
  rows={users}
  keys={["name", "email", "role"]}
/>
```

Renders:
```html
<table>
  <caption>User list</caption>
  <thead>
    <tr>
      <th scope="col">Name</th>
      <th scope="col">Email</th>
      <th scope="col">Role</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Alice</td>
      <td>alice@example.com</td>
      <td>Admin</td>
    </tr>
    <!-- ... -->
  </tbody>
</table>
```

---

### Interactive

#### `<Dialog>`

Modal dialog with focus management.

```parsley
<Button toggle="#confirm-dialog">Open Dialog</Button>

<Dialog id="confirm-dialog" title="Confirm Action">
  <p>Are you sure?</p>
  <Button>Cancel</Button>
  <Button type=submit>Confirm</Button>
</Dialog>
```

Features:
- Focus trapped inside when open
- Escape key closes
- Focus returns to trigger on close
- Proper `aria-labelledby`

#### `<Tabs>`

Tabbed interface with keyboard navigation.

```parsley
<Tabs>
  <Tab id="one" label="First Tab">
    Content for first tab
  </Tab>
  <Tab id="two" label="Second Tab">
    Content for second tab
  </Tab>
</Tabs>
```

Features:
- Arrow key navigation between tabs
- Proper `role="tablist"`, `role="tab"`, `role="tabpanel"`
- `aria-selected`, `aria-controls`, `aria-labelledby`

---

### Media & Content

#### `<Img>`

Image with required alt and lazy loading.

```parsley
<Img src="/photo.jpg" alt="A sunset over mountains" width=800 height=600/>
```

Renders:
```html
<img src="/photo.jpg" alt="A sunset over mountains" width="800" height="600" loading="lazy"/>
```

Props:
- `alt` - Alt text (required, enforced)
- `loading` - Loading strategy (default: "lazy")

#### `<Iframe>`

Iframe with required title.

```parsley
<Iframe src="https://youtube.com/embed/..." title="Product demo video"/>
```

Props:
- `title` - Frame title (required, enforced)
- `loading` - Loading strategy (default: "lazy")

#### `<Figure>`

Figure with caption.

```parsley
<Figure caption="A beautiful sunset">
  <Img src="/sunset.jpg" alt="Sun setting over mountains"/>
</Figure>
```

Renders:
```html
<figure>
  <img src="/sunset.jpg" alt="Sun setting over mountains" loading="lazy"/>
  <figcaption>A beautiful sunset</figcaption>
</figure>
```

#### `<Blockquote>`

Blockquote with proper citation.

```parsley
<Blockquote author="Oscar Wilde" cite="https://...">
  Be yourself; everyone else is already taken.
</Blockquote>
```

Renders:
```html
<figure class="blockquote">
  <blockquote cite="https://...">
    <p>Be yourself; everyone else is already taken.</p>
  </blockquote>
  <figcaption>— <cite>Oscar Wilde</cite></figcaption>
</figure>
```

#### `<Time>`

Time element with proper datetime attribute.

```parsley
<Time value={post.createdAt}>December 7, 2024</Time>
<Time value={post.createdAt}/>  // Auto-formats content
```

Renders:
```html
<time datetime="2024-12-07T10:30:00Z">December 7, 2024</time>
```

#### `<LocalTime>`

Client-side localized datetime display. Renders a `<local-time>` custom element that JavaScript enhances to show the datetime in the user's browser timezone and locale.

```parsley
// With server-rendered fallback
<LocalTime datetime={event.startTime}>
    event.startTime.format("long")
</LocalTime>

// Format options
<LocalTime datetime={post.createdAt} format="short"/>
<LocalTime datetime={meeting.time} format="time"/>   // Time only
<LocalTime datetime={deadline} format="date"/>       // Date only
<LocalTime datetime={event.date} format="long" weekday="long"/>
```

Renders:
```html
<local-time datetime="2024-12-25T14:30:00Z">December 25, 2024 at 2:30 PM UTC</local-time>
```

After JS enhancement (user in EST):
```html
<local-time datetime="2024-12-25T14:30:00Z">December 25, 2024 at 9:30 AM</local-time>
```

Props:
- `datetime` - The UTC datetime to display (required)
- `format` - `"short"`, `"long"` (default), `"full"`, `"date"`, `"time"`
- `weekday` - `"short"`, `"long"` (adds weekday to output)

Format examples:

| Format | Example Output |
|--------|----------------|
| `"short"` | 12/25/24, 9:30 AM |
| `"long"` | December 25, 2024 at 9:30 AM |
| `"full"` | Wednesday, December 25, 2024 at 9:30 AM EST |
| `"date"` | December 25, 2024 |
| `"time"` | 9:30 AM |

The JavaScript uses `Intl.DateTimeFormat` with the browser's locale (`navigator.language`).

#### `<TimeRange>`

Smart display of datetime spans that collapses redundant information. Renders a `<time-range>` custom element.

```parsley
// Same-day event
<TimeRange start={session.start} end={session.end}>
    session.start.format("long") + " – " + session.end.time
</TimeRange>
// → December 25, 2024, 9:00 AM – 11:00 AM

// Multi-day event
<TimeRange start={conference.start} end={conference.end}>
    conference.start.date + " – " + conference.end.date
</TimeRange>
// → December 25 – 27, 2024
```

Renders:
```html
<time-range start="2024-12-25T14:00:00Z" end="2024-12-25T16:00:00Z">
    December 25, 2024 at 2:00 PM – 4:00 PM UTC
</time-range>
```

After JS enhancement (user in EST):
```html
<time-range start="2024-12-25T14:00:00Z" end="2024-12-25T16:00:00Z">
    December 25, 2024, 9:00 AM – 11:00 AM
</time-range>
```

Props:
- `start` - Start datetime (required)
- `end` - End datetime (required)
- `format` - `"short"`, `"long"` (default)
- `separator` - Text between start/end (default: `" – "`)

Smart formatting rules:

| Scenario | Example Output |
|----------|----------------|
| Same day | December 25, 2024, 9:00 AM – 11:00 AM |
| Same month | December 25 – 27, 2024 |
| Same year | December 25 – January 3, 2024 |
| Different years | December 25, 2024 – January 3, 2025 |
| All day (midnight to midnight) | December 25, 2024 |
| Multi-day all day | December 25 – 27, 2024 |

#### `<RelativeTime>`

Human-readable relative time display ("2 hours ago", "in 3 days"). Renders a `<relative-time>` custom element with optional auto-refresh.

```parsley
// Basic usage
<RelativeTime datetime={comment.createdAt}/>
// → "5 minutes ago"

// Live countdown (auto-refreshes)
<RelativeTime datetime={auction.ends} live/>
// → "2 hours 34 minutes" (updates every minute)

// Threshold: show relative within 7 days, then absolute
<RelativeTime datetime={post.createdAt} threshold=@7d/>
// Recent: "3 days ago"
// Older: "December 18, 2024"
```

Renders:
```html
<relative-time datetime="2024-12-25T14:30:00Z">2 hours ago</relative-time>

<!-- With live updates -->
<relative-time datetime="2024-12-25T16:00:00Z" live>in 2 hours 34 minutes</relative-time>
```

Props:
- `datetime` - The UTC datetime (required)
- `live` - Auto-refresh the display (for countdowns/countups)
- `threshold` - Duration after which to show absolute date instead
- `format` - Absolute date format when threshold exceeded (default: `"long"`)

Relative formatting examples:

| Time Difference | Output |
|-----------------|--------|
| < 1 minute | just now |
| 1-59 minutes | 5 minutes ago |
| 1-23 hours | 2 hours ago |
| 1-6 days | 3 days ago |
| 1-4 weeks | 2 weeks ago |
| 1-11 months | 3 months ago |
| 1+ years | 2 years ago |
| Future | in 3 days |

The JavaScript uses `Intl.RelativeTimeFormat` for localized output. When `live` is set, the component updates every minute (or every second when < 1 minute away).

**Auto-refresh behavior:**
- Updates DOM only when displayed text would change
- Pauses when tab is hidden (via `document.visibilityState`)
- Cleans up interval on element disconnect

---

### Time Component Accessibility

The time components (`<LocalTime>`, `<TimeRange>`, `<RelativeTime>`) have specific accessibility considerations:

#### Machine-Readable Datetime
All components preserve the ISO datetime in the `datetime` attribute, providing machine-readable data for assistive technologies and tools.

#### Screen Reader Announcements
- **Static times**: No special handling needed — screen readers read the text content naturally
- **Live updates**: Use `aria-live="off"` by default to prevent constant announcements during countdowns
- **Critical countdowns**: Add `announce` prop to enable `aria-live="polite"` for important deadlines

```parsley
// Silent countdown (default) - no announcements
<RelativeTime datetime={auction.ends} live/>

// Announced countdown - speaks changes at key intervals
<RelativeTime datetime={auction.ends} live announce/>
```

When `announce` is set, the component only announces at meaningful intervals (1 hour, 30 min, 10 min, 5 min, 1 min, 30 sec) rather than every update.

#### Timezone Clarity
For `<LocalTime>`, consider whether users need to know which timezone is displayed:

```parsley
// Show timezone abbreviation
<LocalTime datetime={event.time} format="full"/>
// → "December 25, 2024 at 9:30 AM EST"

// Or add explicit timezone indicator
<LocalTime datetime={flight.departure} showZone/>
// → "9:30 AM (EST)"
```

#### Relative Time Comprehension
Relative times ("3 days ago") can be harder to interpret than absolute dates for some users. The `threshold` prop addresses this by switching to absolute dates after a duration:

```parsley
// After 7 days, show the actual date
<RelativeTime datetime={post.createdAt} threshold=@7d/>
```

Consider using shorter thresholds for important dates, or providing both:

```parsley
<span>
    <RelativeTime datetime={event.date}/>
    (<LocalTime datetime={event.date} format="date"/>)
</span>
// → "in 3 days (December 28, 2024)"
```

#### Title Attribute for Precision
All time components include the full UTC datetime in a `title` attribute for hover/focus:

```html
<local-time datetime="2024-12-25T14:30:00Z" title="December 25, 2024 at 2:30 PM UTC">
    December 25, 2024 at 9:30 AM
</local-time>
```

This provides access to the original UTC time for users who need precision.

#### Focus and Interaction
Time elements are not interactive by default (no focus, no click handling). If a time needs to be actionable (e.g., click to copy, click for details), wrap in a `<button>` or `<a>`:

```parsley
<button type="button" data-copy-datetime={event.time.iso}>
    <LocalTime datetime={event.time}>event.time.format("long")</LocalTime>
    <SrOnly>Click to copy datetime</SrOnly>
</button>
```

---

#### `<Abbr>`

Abbreviation with required expansion.

```parsley
<Abbr title="HyperText Markup Language">HTML</Abbr>
```

---

### Utility

#### `<Button>`

Button with sensible defaults.

```parsley
<Button>Click Me</Button>           // type="button" (not submit!)
<Button type=submit>Save</Button>   // type="submit"
<Button toggle="#menu">Menu</Button>
<Button copy="#api-key">Copy</Button>
```

Props:
- `type` - Button type (default: "button", not "submit")
- `toggle` - ID of element to toggle visibility
- `copy` - ID of element to copy text from

#### `<A>`

Link with automatic safety for external links.

```parsley
<A href="/about">About</A>
<A href="https://external.com" external>External Site</A>
```

External links automatically get `target="_blank" rel="noopener noreferrer"`.

#### `<SrOnly>`

Screen reader only text (visually hidden).

```parsley
<button>
  <Icon name="menu"/>
  <SrOnly>Open menu</SrOnly>
</button>
```

Renders:
```html
<button>
  <svg aria-hidden="true">...</svg>
  <span class="sr-only">Open menu</span>
</button>
```

#### `<Icon>`

Accessible icon wrapper.

```parsley
<Icon name="search" label="Search"/>
```

Renders:
```html
<svg aria-hidden="true" class="icon icon-search">...</svg>
<span class="sr-only">Search</span>
```

If no label, just `aria-hidden="true"` (decorative).

---

## Prop Conventions

### Standard Props (All Components)

| Prop | Target | Purpose |
|------|--------|---------|
| `class` | Outer wrapper | CSS class |
| `id` | Outer wrapper | Element ID |

### Pass-through Props

- `data-*` attributes pass to outer wrapper
- `aria-*` attributes pass to semantically relevant element

### ID Generation

When `id` is provided, inner elements get derived IDs:
- `id="foo"` → input gets `id="foo-input"`, hint gets `id="foo-hint"`

When `id` is omitted but `name` exists:
- `name="email"` → `id="field-email"`, input gets `id="field-email-input"`

---

## JavaScript Enhancements

### Philosophy

- Progressive enhancement only - everything works without JS
- Tiny footprint (~55 lines total)
- Only behaviors that genuinely improve UX
- Automatic - no user configuration

### Automatic Behaviors (No Prop)

These are always enabled:

#### Disable Submit on Submit
Prevents double-click/double-submit on all `<Form>` components.

#### Focus First Error
On page load, focuses the first `[aria-invalid="true"]` element.

### Opt-In Behaviors (Via Props)

| Prop | Component | Behavior |
|------|-----------|----------|
| `confirm="..."` | Form | Shows native confirm dialog before submit |
| `autosubmit` | Select, RadioGroup | Submits form on change |
| `counter` | TextareaField | Shows live character count |
| `autoresize` | TextareaField | Grows textarea with content |
| `toggle="#id"` | Button | Toggles visibility of target element |
| `copy="#id"` | Button | Copies target element text to clipboard |

### Implementation

```javascript
// Confirm before submit
document.querySelectorAll('form[data-confirm]').forEach(f => 
  f.addEventListener('submit', e => 
    confirm(f.dataset.confirm) || e.preventDefault()))

// Auto-submit on change
document.querySelectorAll('[data-autosubmit]').forEach(el =>
  el.addEventListener('change', () => el.form.submit()))

// Character counter
document.querySelectorAll('[data-counter]').forEach(ta => {
  const counter = document.getElementById(ta.dataset.counter)
  const max = ta.maxLength
  const update = () => counter.textContent = `${ta.value.length} / ${max}`
  ta.addEventListener('input', update)
  update()
})

// Toggle visibility
document.querySelectorAll('[data-toggle]').forEach(btn => {
  const target = document.querySelector(btn.dataset.toggle)
  btn.setAttribute('aria-controls', target.id)
  btn.setAttribute('aria-expanded', !target.hidden)
  btn.addEventListener('click', () => {
    target.hidden = !target.hidden
    btn.setAttribute('aria-expanded', !target.hidden)
  })
})

// Copy to clipboard
document.querySelectorAll('[data-copy]').forEach(btn => {
  const originalText = btn.textContent
  btn.addEventListener('click', async () => {
    try {
      const text = document.querySelector(btn.dataset.copy).textContent
      await navigator.clipboard.writeText(text)
      btn.textContent = 'Copied!'
    } catch (e) {
      btn.textContent = 'Failed'
    }
    setTimeout(() => btn.textContent = originalText, 2000)
  })
})

// Disable submit button on submit
document.querySelectorAll('form').forEach(f =>
  f.addEventListener('submit', () =>
    f.querySelectorAll('[type=submit]').forEach(b => b.disabled = true)))

// Auto-resize textarea (CSS fallback)
if (!CSS.supports('field-sizing', 'content')) {
  document.querySelectorAll('[data-autoresize]').forEach(ta => {
    const resize = () => { ta.style.height = 'auto'; ta.style.height = ta.scrollHeight + 'px' }
    ta.addEventListener('input', resize)
    resize()
  })
}

// Focus first invalid field
const firstError = document.querySelector('[aria-invalid="true"]')
if (firstError) firstError.focus()
```

### Delivery

Three special tags are available for including JavaScript and CSS bundles:

| Tag | Output | Purpose |
|-----|--------|---------|
| `<CSS/>` | `<link rel="stylesheet" href="/__site.css?v={hash}">` | Site CSS bundle (all `.css` files from handlers directory) |
| `<Javascript/>` | `<script src="/__site.js?v={hash}"></script>` | Site JS bundle (all `.js` files from handlers directory) |
| `<BasilJS/>` | `<script src="/__/js/basil.{hash}.js"></script>` | Basil component enhancement JavaScript |

**When using `<Page>`**, all three are automatically included:
- `<CSS/>` in the `<head>`
- `<Javascript/>` and `<BasilJS/>` before `</body>`

**When building custom page layouts**, include them manually:

```parsley
<html>
<head>
  <CSS/>
</head>
<body>
  // ... content ...
  <Javascript/>
  <BasilJS/>
</body>
</html>
```

The tags output empty strings if there are no files to bundle (e.g., `<CSS/>` outputs nothing if you have no `.css` files).

---

## CSS Considerations

### Class Naming

Components use predictable BEM-style classes:

```
.field                 - Field wrapper
.field-label           - Label element
.field-input           - Input element
.field-hint            - Hint text
.field-error           - Error message
.field-required        - Required indicator

.skip-link             - Skip navigation link
.sr-only               - Screen reader only

.radio-group           - Radio group fieldset
.checkbox-group        - Checkbox group fieldset

.blockquote            - Blockquote figure
.counter               - Character counter
```

### Required CSS

Only the skip link and sr-only need CSS to function:

```css
.skip-link {
  position: absolute;
  left: -9999px;
}
.skip-link:focus {
  position: static;
  left: auto;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
```

Basil could include these automatically, or users add them to their stylesheet.

---

## Implementation Notes

### Library Location

Components live in `std/html` (imported) or are built-in to Basil.

Options:
1. **Built-in** - Always available, zero friction
2. **Import** - `import "std/html"` - explicit dependency

Recommendation: Built-in for Basil projects. They're Basil components, not generic Parsley.

### Error Handling

Components should fail gracefully:
- `<Img>` without `alt` → compile/runtime error (enforced)
- `<Iframe>` without `title` → compile/runtime error (enforced)
- `<TextField>` without `name` → compile/runtime error (enforced)
- `<TextField>` without `label` → compile/runtime error (enforced)

### Future Considerations

- **Dark mode**: Components could accept `colorScheme` prop
- **RTL support**: Components could accept `dir` prop
- **i18n**: Error messages, button text could be configurable
- **Form `target=`**: See `rails-inspired-ux.md` for partial updates design

---

## Example: Complete Form

```parsley
<Page lang="en" title="Register">
  <main>
    <h1>Create Account</h1>
    
    <Form action="/register" method=POST>
      <TextField 
        name="name" 
        label="Full name" 
        value={form.name}
        error={errors.name}
        required
      />
      
      <TextField 
        name="email" 
        label="Email address" 
        type=email
        value={form.email}
        error={errors.email}
        hint="We'll never share your email"
        required
      />
      
      <TextareaField
        name="bio"
        label="Bio"
        value={form.bio}
        maxlength=280
        counter
        autoresize
      />
      
      <SelectField
        name="country"
        label="Country"
        value={form.country}
        options={countries}
        valueKey="code"
        labelKey="name"
      />
      
      <Checkbox 
        name="terms" 
        label="I agree to the terms of service"
        checked={form.terms}
        error={errors.terms}
      />
      
      <Button type=submit>Create Account</Button>
    </Form>
  </main>
</Page>
```

This produces fully accessible HTML with:
- Proper document structure
- Skip link
- CSRF protection
- Label/input associations
- Error announcements
- Required field indicators
- Character counter
- Auto-resize textarea
- Double-submit prevention
- Focus management on errors

All from ~40 lines of Parsley.
