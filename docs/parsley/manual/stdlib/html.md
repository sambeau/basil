---
id: man-pars-std-html
title: "Std/HTML"
system: parsley
type: stdlib
name: "@std/html"
created: 2025-12-20
version: 0.2.0
author: "@copilot"
keywords: 
            - html
            - components
            - forms
            - accessibility
            - aria
            - semantic
            - a11y
---

## HTML Components Library

The `@std/html` module provides accessible, semantic HTML components that save you from looking up correct ARIA attributes, proper element structure, and modern best practices. Every component renders server-side with progressive enhancement—they work without JavaScript but can be enhanced by client-side scripts.

```parsley
let {TextField, Button, Form} = import @std/html

<Form action="/contact" method="POST">
    <TextField name="email" label="Email" type="email" required={true}/>
    <TextField name="message" label="Message" hint="We'll respond within 24 hours"/>
    <Button type="submit">"Send Message"</Button>
</Form>
```

## Philosophy

**"Just enough to be more convenient than looking it up on MDN."**

These components:
- Render semantic, accessible HTML with correct ARIA attributes
- Are unstyled by default—works with any CSS approach
- Use progressive enhancement—JavaScript optional
- Follow consistent patterns for labels, hints, and errors

### What to Use Native HTML For

These tags gain nothing from being components. Just use them directly:

```parsley
<p>, <span>, <div>, <strong>, <em>, <ul>, <ol>, <li>
<h1>, <h2>, <h3>, <header>, <footer>, <main>, <article>
```

---

## Form Components

### TextField

Complete text input with label, hint text, error message, and full accessibility support.

```parsley
let {TextField} = import @std/html

// Basic text field
<TextField name="username" label="Username"/>

// Email with validation hint
<TextField 
    name="email" 
    label="Email Address" 
    type="email"
    hint="We'll never share your email"
    required={true}
/>

// Field with error
<TextField 
    name="password" 
    label="Password" 
    type="password"
    error="Password must be at least 8 characters"
    minlength={8}
/>
```

Renders (for the error example):
```html
<div class="field" id="field-password">
    <label for="field-password-input">
        Password
        <span class="field-required" aria-hidden="true"> *</span>
    </label>
    <input 
        type="password"
        id="field-password-input"
        name="password"
        minlength="8"
        required
        aria-required="true"
        aria-describedby="field-password-error"
        aria-invalid="true"
    />
    <p id="field-password-error" class="field-error" role="alert">
        Password must be at least 8 characters
    </p>
</div>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required) |
| `label` | string | Label text (required) |
| `type` | string | Input type: "text", "email", "password", "tel", "url", etc. Default: "text" |
| `value` | string | Current value |
| `hint` | string | Help text shown below input |
| `error` | string | Error message (triggers invalid state) |
| `required` | boolean | Mark as required field |
| `placeholder` | string | Placeholder text |
| `autocomplete` | string | Autocomplete hint ("email", "name", etc.) |
| `disabled` | boolean | Disable the input |
| `readonly` | boolean | Make read-only |
| `minlength` | number | Minimum character length |
| `maxlength` | number | Maximum character length |
| `pattern` | string | Regex validation pattern |
| `id` | string | Override generated ID |
| `class` | string | Additional CSS classes |

---

### TextareaField

Multi-line text input with optional character counter and auto-resize.

```parsley
let {TextareaField} = import @std/html

// Basic textarea
<TextareaField name="bio" label="Biography" rows={4}/>

// With character counter
<TextareaField 
    name="description" 
    label="Product Description"
    maxlength={500}
    counter={true}
    hint="Describe your product in detail"
/>

// Auto-resizing textarea
<TextareaField 
    name="notes" 
    label="Notes"
    autoresize={true}
/>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required) |
| `label` | string | Label text (required) |
| `value` | string | Current value |
| `hint` | string | Help text |
| `error` | string | Error message |
| `required` | boolean | Mark as required |
| `rows` | number | Number of visible rows. Default: 4 |
| `cols` | number | Visible width in characters |
| `minlength` | number | Minimum character length |
| `maxlength` | number | Maximum character length |
| `counter` | boolean | Show character count (requires maxlength) |
| `autoresize` | boolean | Auto-grow with content (requires JS) |
| `placeholder` | string | Placeholder text |
| `disabled` | boolean | Disable the textarea |
| `readonly` | boolean | Make read-only |

---

### SelectField

Dropdown select with support for both simple arrays and object arrays.

```parsley
let {SelectField} = import @std/html

// Simple array of options
<SelectField 
    name="color" 
    label="Favorite Color"
    options={["Red", "Green", "Blue"]}
    placeholder="Choose a color..."
/>

// Array of objects with custom keys
let countries = [
    {code: "US", name: "United States"},
    {code: "GB", name: "United Kingdom"},
    {code: "FR", name: "France"}
]

<SelectField 
    name="country" 
    label="Country"
    options={countries}
    valueKey="code"
    labelKey="name"
    value="GB"
/>

// Auto-submit on change
<SelectField 
    name="sort" 
    label="Sort by"
    options={["Newest", "Oldest", "Popular"]}
    autosubmit={true}
/>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required) |
| `label` | string | Label text (required) |
| `options` | array | Array of options (strings or objects) |
| `value` | string | Currently selected value |
| `valueKey` | string | Object property for option value. Default: "value" |
| `labelKey` | string | Object property for option label. Default: "label" |
| `placeholder` | string | Placeholder option text |
| `hint` | string | Help text |
| `error` | string | Error message |
| `required` | boolean | Mark as required |
| `disabled` | boolean | Disable the select |
| `autosubmit` | boolean | Submit form on change (requires JS) |

---

### RadioGroup

Group of mutually exclusive radio buttons with proper fieldset/legend structure.

```parsley
let {RadioGroup} = import @std/html

// Simple options
<RadioGroup 
    name="size" 
    label="Select Size"
    options={["Small", "Medium", "Large"]}
    value="Medium"
/>

// Object options
let plans = [
    {value: "free", label: "Free Plan"},
    {value: "pro", label: "Pro Plan ($10/mo)"},
    {value: "enterprise", label: "Enterprise (Contact us)"}
]

<RadioGroup 
    name="plan" 
    label="Choose a Plan"
    options={plans}
    required={true}
    hint="You can change your plan anytime"
/>
```

Renders:
```html
<fieldset class="radio-group" id="field-size">
    <legend>Select Size</legend>
    <div class="radio-group-options">
        <label class="radio-option">
            <input type="radio" name="size" value="Small"/>
            <span class="radio-label">Small</span>
        </label>
        <label class="radio-option">
            <input type="radio" name="size" value="Medium" checked/>
            <span class="radio-label">Medium</span>
        </label>
        <label class="radio-option">
            <input type="radio" name="size" value="Large"/>
            <span class="radio-label">Large</span>
        </label>
    </div>
</fieldset>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required) |
| `label` | string | Legend text (required) |
| `options` | array | Array of options (strings or objects) |
| `value` | string | Currently selected value |
| `valueKey` | string | Object property for option value. Default: "value" |
| `labelKey` | string | Object property for option label. Default: "label" |
| `hint` | string | Help text |
| `error` | string | Error message |
| `required` | boolean | Mark as required |
| `disabled` | boolean | Disable all options |

---

### CheckboxGroup

Group of checkboxes for multi-select scenarios.

```parsley
let {CheckboxGroup} = import @std/html

let toppings = [
    {value: "cheese", label: "Extra Cheese"},
    {value: "pepperoni", label: "Pepperoni"},
    {value: "mushrooms", label: "Mushrooms"},
    {value: "olives", label: "Olives"}
]

<CheckboxGroup 
    name="toppings" 
    label="Select Toppings"
    options={toppings}
    values={["cheese", "pepperoni"]}
/>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required). Renders as `name[]` |
| `label` | string | Legend text (required) |
| `options` | array | Array of options (strings or objects) |
| `values` | array | Array of currently selected values |
| `valueKey` | string | Object property for option value. Default: "value" |
| `labelKey` | string | Object property for option label. Default: "label" |
| `hint` | string | Help text |
| `error` | string | Error message |
| `required` | boolean | Mark as required |
| `disabled` | boolean | Disable all options |

---

### Checkbox

Single checkbox for boolean values like terms acceptance.

```parsley
let {Checkbox} = import @std/html

<Checkbox 
    name="terms" 
    label="I agree to the Terms of Service"
    required={true}
/>

<Checkbox 
    name="newsletter" 
    label="Subscribe to our newsletter"
    checked={true}
    hint="We send updates about once a month"
/>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Input name (required) |
| `label` | string | Label text (required) |
| `checked` | boolean | Whether checkbox is checked |
| `value` | string | Value when checked. Default: "true" |
| `hint` | string | Help text |
| `error` | string | Error message |
| `required` | boolean | Mark as required |
| `disabled` | boolean | Disable the checkbox |

---

### Button

Button with sensible defaults—`type="button"` by default (not submit!), plus support for toggle and copy behaviors.

```parsley
let {Button} = import @std/html

// Regular button (won't accidentally submit forms)
<Button>"Click Me"</Button>

// Submit button
<Button type="submit">"Save Changes"</Button>

// Toggle button for showing/hiding content
<Button toggle="#menu">"Toggle Menu"</Button>

// Copy-to-clipboard button
<Button copy="#api-key">"Copy API Key"</Button>

// Disabled button
<Button disabled={true}>"Not Available"</Button>
```

| Prop | Type | Description |
|------|------|-------------|
| `type` | string | Button type: "button", "submit", "reset". Default: "button" |
| `toggle` | string | CSS selector for element to toggle (requires JS) |
| `copy` | string | CSS selector for element whose text to copy (requires JS) |
| `disabled` | boolean | Disable the button |
| `name` | string | Button name for form submission |
| `value` | string | Button value for form submission |
| `form` | string | ID of form to associate with |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

### Form

Form wrapper with automatic CSRF protection and optional confirmation dialog.

```parsley
let {Form, TextField, Button} = import @std/html

<Form action="/register" method="POST">
    <TextField name="email" label="Email" type="email" required={true}/>
    <TextField name="password" label="Password" type="password" required={true}/>
    <Button type="submit">"Register"</Button>
</Form>

// With confirmation dialog
<Form action="/delete" method="POST" confirm="Are you sure you want to delete this?">
    <Button type="submit">"Delete Account"</Button>
</Form>
```

Renders:
```html
<form action="/register" method="POST" class="form">
    <input type="hidden" name="_csrf" value="..."/>
    <!-- fields -->
</form>
```

| Prop | Type | Description |
|------|------|-------------|
| `action` | string | Form action URL (required) |
| `method` | string | HTTP method. Default: "POST" |
| `confirm` | string | Confirmation message before submit (requires JS) |
| `enctype` | string | Encoding type (e.g., "multipart/form-data") |
| `target` | string | Target frame/window |
| `novalidate` | boolean | Disable browser validation |
| `autocomplete` | string | Autocomplete behavior |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

## Navigation Components

### Nav

Navigation landmark with proper ARIA labeling.

```parsley
let {Nav} = import @std/html

<Nav label="Main navigation">
    <a href="/">"Home"</a>
    <a href="/products">"Products"</a>
    <a href="/about">"About"</a>
    <a href="/contact">"Contact"</a>
</Nav>

// Secondary navigation
<Nav label="Account menu">
    <a href="/settings">"Settings"</a>
    <a href="/logout">"Sign Out"</a>
</Nav>
```

Renders:
```html
<nav aria-label="Main navigation">
    <a href="/">Home</a>
    ...
</nav>
```

| Prop | Type | Description |
|------|------|-------------|
| `label` | string | Accessible name for the navigation region |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### Breadcrumb

Breadcrumb navigation with Schema.org structured data for SEO.

```parsley
let {Breadcrumb} = import @std/html

<Breadcrumb items={[
    {label: "Home", href: "/"},
    {label: "Products", href: "/products"},
    {label: "Electronics", href: "/products/electronics"},
    {label: "Headphones"}
]}/>

// Custom separator
<Breadcrumb 
    items={[{label: "Home", href: "/"}, {label: "About"}]}
    separator=" → "
/>
```

Renders:
```html
<nav class="breadcrumb" aria-label="Breadcrumb">
    <ol class="breadcrumb-list" itemscope itemtype="https://schema.org/BreadcrumbList">
        <li class="breadcrumb-item" itemprop="itemListElement" itemscope itemtype="https://schema.org/ListItem">
            <a href="/" itemprop="item"><span itemprop="name">Home</span></a>
            <meta itemprop="position" content="1"/>
        </li>
        <li class="breadcrumb-item" ...>
            <span class="breadcrumb-separator" aria-hidden="true"> / </span>
            ...
        </li>
        <li class="breadcrumb-item" ...>
            <span class="breadcrumb-separator" aria-hidden="true"> / </span>
            <span itemprop="name" aria-current="page">Headphones</span>
            <meta itemprop="position" content="4"/>
        </li>
    </ol>
</nav>
```

| Prop | Type | Description |
|------|------|-------------|
| `items` | array | Array of `{label, href?}` objects (required) |
| `separator` | string | Separator between items. Default: " / " |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

### SkipLink

Accessibility skip link for keyboard users to bypass navigation.

```parsley
let {SkipLink} = import @std/html

// Usually at the very top of the page
<SkipLink/>

// Custom target and text
<SkipLink target="#content" text="Skip to main content"/>
```

Renders:
```html
<a href="#main" class="skip-link">Skip to main content</a>
```

| Prop | Type | Description |
|------|------|-------------|
| `target` | string | Target element ID. Default: "#main" |
| `text` | string | Link text. Default: "Skip to main content" |

---

## Media Components

### Img

Image with required `alt` attribute and lazy loading by default.

```parsley
let {Img} = import @std/html

// Basic image
<Img src="/hero.jpg" alt="Mountain landscape at sunset" width={1200} height={800}/>

// Decorative image (empty alt)
<Img src="/divider.svg" alt="" width={100} height={2}/>

// Responsive image
<Img 
    src="/photo.jpg" 
    alt="Team photo"
    srcset="/photo-400.jpg 400w, /photo-800.jpg 800w, /photo-1200.jpg 1200w"
    sizes="(max-width: 600px) 400px, (max-width: 1000px) 800px, 1200px"
/>

// Eager loading for above-fold images
<Img src="/logo.png" alt="Company Logo" loading="eager"/>
```

| Prop | Type | Description |
|------|------|-------------|
| `src` | string | Image source URL (required) |
| `alt` | string | Alternative text (required for accessibility) |
| `width` | number | Image width |
| `height` | number | Image height |
| `loading` | string | Loading strategy: "lazy", "eager". Default: "lazy" |
| `decoding` | string | Decoding hint: "async", "sync", "auto". Default: "async" |
| `srcset` | string | Responsive image sources |
| `sizes` | string | Responsive size hints |
| `crossorigin` | string | CORS setting |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### Iframe

Iframe with required `title` for accessibility and lazy loading.

```parsley
let {Iframe} = import @std/html

<Iframe 
    src="https://www.youtube.com/embed/dQw4w9WgXcQ" 
    title="Product demo video"
    width={560}
    height={315}
/>

// Map embed with specific permissions
<Iframe 
    src="https://maps.google.com/..."
    title="Store location map"
    allow="geolocation"
/>
```

| Prop | Type | Description |
|------|------|-------------|
| `src` | string | Frame source URL (required) |
| `title` | string | Accessible title (required) |
| `width` | number | Frame width |
| `height` | number | Frame height |
| `loading` | string | Loading strategy. Default: "lazy" |
| `allow` | string | Permissions policy |
| `sandbox` | string | Sandbox restrictions |
| `referrerpolicy` | string | Referrer policy |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### Figure

Figure with caption—proper semantic structure for images, diagrams, or code.

```parsley
let {Figure, Img} = import @std/html

<Figure caption="Annual revenue growth 2020-2024">
    <Img src="/chart.png" alt="Bar chart showing 15% year-over-year growth"/>
</Figure>
```

Renders:
```html
<figure>
    <img src="/chart.png" alt="..." loading="lazy"/>
    <figcaption>Annual revenue growth 2020-2024</figcaption>
</figure>
```

| Prop | Type | Description |
|------|------|-------------|
| `caption` | string | Figure caption |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### Blockquote

Blockquote with proper citation structure.

```parsley
let {Blockquote} = import @std/html

<Blockquote author="Oscar Wilde">
    "Be yourself; everyone else is already taken."
</Blockquote>

<Blockquote 
    author="Marie Curie" 
    cite="https://example.com/curie-quotes"
>
    "Nothing in life is to be feared, it is only to be understood."
</Blockquote>
```

Renders:
```html
<figure class="blockquote">
    <blockquote cite="https://example.com/curie-quotes">
        Nothing in life is to be feared, it is only to be understood.
    </blockquote>
    <figcaption>— <cite>Marie Curie</cite></figcaption>
</figure>
```

| Prop | Type | Description |
|------|------|-------------|
| `author` | string | Quote attribution |
| `cite` | string | Source URL |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

## Time Components

### Time

Semantic time element with proper `datetime` attribute.

```parsley
let {Time} = import @std/html

// Auto-formatted display
<Time value={post.createdAt}/>
// → <time datetime="2024-12-07T10:30:00Z">December 7, 2024</time>

// Custom display text
<Time value={event.date}>"This Saturday"</Time>

// Different format
<Time value={post.createdAt} format="short"/>
// → <time datetime="2024-12-07T10:30:00Z">12/7/24</time>
```

| Prop | Type | Description |
|------|------|-------------|
| `value` | datetime | The datetime value (required) |
| `format` | string | Display format: "short", "long" (default), "full" |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### LocalTime

Client-side localized datetime. Renders a custom element that JavaScript can enhance to show the user's local timezone.

```parsley
let {LocalTime} = import @std/html

// Shows server time, JS updates to local
<LocalTime datetime={event.startTime}/>

// With format options
<LocalTime datetime={post.createdAt} format="short"/>
<LocalTime datetime={meeting.time} format="time"/>  // Time only
<LocalTime datetime={deadline} weekday="long"/>     // Include weekday
```

| Prop | Type | Description |
|------|------|-------------|
| `datetime` | datetime | The UTC datetime (required) |
| `format` | string | "short", "long" (default), "full", "date", "time" |
| `weekday` | string | "short" or "long" to include weekday |
| `showZone` | boolean | Show timezone abbreviation |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### TimeRange

Smart display of datetime spans that collapses redundant information.

```parsley
let {TimeRange} = import @std/html

// Same-day event
<TimeRange start={session.start} end={session.end}/>
// → December 25, 2024, 9:00 AM – 11:00 AM

// Multi-day event
<TimeRange start={conference.start} end={conference.end}/>
// → December 25 – 27, 2024
```

| Prop | Type | Description |
|------|------|-------------|
| `start` | datetime | Start datetime (required) |
| `end` | datetime | End datetime (required) |
| `format` | string | "short" or "long" (default) |
| `separator` | string | Text between times. Default: " – " |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### RelativeTime

Human-readable relative time ("2 hours ago", "in 3 days").

```parsley
let {RelativeTime} = import @std/html

// Basic relative time
<RelativeTime datetime={comment.createdAt}/>
// → "5 minutes ago"

// Live countdown (updates automatically)
<RelativeTime datetime={auction.ends} live={true}/>
// → "2 hours 34 minutes" (updates every minute)

// Threshold: relative within 7 days, then absolute
<RelativeTime datetime={post.createdAt} threshold={@7d}/>
// Recent: "3 days ago"
// Older: "December 18, 2024"
```

| Prop | Type | Description |
|------|------|-------------|
| `datetime` | datetime | The datetime (required) |
| `live` | boolean | Auto-refresh display (requires JS) |
| `threshold` | duration | Show absolute date after this duration |
| `format` | string | Absolute format when threshold exceeded |
| `announce` | boolean | Announce updates to screen readers |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

## Data Components

### DataTable

Data table with proper header semantics and accessibility.

```parsley
let {DataTable} = import @std/html

let users = [
    {name: "Alice", email: "alice@example.com", role: "Admin"},
    {name: "Bob", email: "bob@example.com", role: "User"},
    {name: "Charlie", email: "charlie@example.com", role: "User"}
]

<DataTable 
    caption="User Accounts"
    columns={["Name", "Email", "Role"]}
    rows={users}
    keys={["name", "email", "role"]}
/>
```

Renders:
```html
<table class="data-table">
    <caption>User Accounts</caption>
    <thead>
        <tr>
            <th scope="col">Name</th>
            <th scope="col">Email</th>
            <th scope="col">Role</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <th scope="row">Alice</th>
            <td>alice@example.com</td>
            <td>Admin</td>
        </tr>
        <tr>
            <th scope="row">Bob</th>
            <td>bob@example.com</td>
            <td>User</td>
        </tr>
        ...
    </tbody>
</table>
```

| Prop | Type | Description |
|------|------|-------------|
| `caption` | string | Table caption |
| `columns` | array | Array of column header strings |
| `rows` | array | Array of row data objects |
| `keys` | array | Object keys corresponding to columns |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

## Utility Components

### A

Link with automatic safety for external links.

```parsley
let {A} = import @std/html

// Internal link (no changes)
<A href="/about">"About Us"</A>

// External link (automatically adds rel="noopener noreferrer")
<A href="https://example.com" external={true}>"External Site"</A>

// target="_blank" also triggers safety attributes
<A href="https://docs.example.com" target="_blank">"Documentation"</A>
```

| Prop | Type | Description |
|------|------|-------------|
| `href` | string | Link URL (required) |
| `external` | boolean | Mark as external link (adds safety attributes, opens in new tab) |
| `target` | string | Link target |
| `rel` | string | Override rel attribute |
| `download` | string | Download filename |
| `hreflang` | string | Language of linked resource |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### Abbr

Abbreviation with required expansion.

```parsley
let {Abbr} = import @std/html

<p>
    "The "<Abbr title="World Wide Web Consortium">"W3C"</Abbr>
    " sets web standards."
</p>
```

Renders:
```html
<p>The <abbr title="World Wide Web Consortium">W3C</abbr> sets web standards.</p>
```

| Prop | Type | Description |
|------|------|-------------|
| `title` | string | Full expansion of abbreviation (required) |
| `id` | string | Element ID |
| `class` | string | CSS classes |

---

### SrOnly

Screen reader only text—visually hidden but accessible.

```parsley
let {SrOnly, Icon} = import @std/html

// Add context for screen readers
<button>
    <Icon name="trash"/>
    <SrOnly>"Delete item"</SrOnly>
</button>
```

Renders:
```html
<button>
    <span class="icon icon-trash" aria-hidden="true"></span>
    <span class="sr-only">Delete item</span>
</button>
```

---

### Icon

Accessible icon wrapper with screen reader label.

```parsley
let {Icon} = import @std/html

// Icon with visible label nearby (decorative)
<button><Icon name="save"/>" Save"</button>

// Icon-only button (needs label for accessibility)
<button><Icon name="close" label="Close dialog"/></button>
```

Renders (with label):
```html
<span class="icon icon-close" aria-hidden="true"></span>
<span class="sr-only">Close dialog</span>
```

| Prop | Type | Description |
|------|------|-------------|
| `name` | string | Icon name (becomes class `icon-{name}`) |
| `label` | string | Accessible label (creates sr-only text) |
| `id` | string | Element ID |
| `class` | string | Additional CSS classes |

---

## Complete Example

Here's a realistic contact form using multiple components together:

```parsley
let {Form, TextField, TextareaField, SelectField, Checkbox, Button, Nav, Breadcrumb} = import @std/html

// Navigation
<Nav label="Main">
    <a href="/">"Home"</a>
    <a href="/contact" aria-current="page">"Contact"</a>
</Nav>

// Breadcrumb
<Breadcrumb items={[
    {label: "Home", href: "/"},
    {label: "Contact"}
]}/>

<h1>"Contact Us"</h1>

// Contact form with validation
<Form action="/contact" method="POST">
    <TextField 
        name="name" 
        label="Your Name" 
        required={true}
        autocomplete="name"
    />
    
    <TextField 
        name="email" 
        label="Email Address" 
        type="email"
        required={true}
        autocomplete="email"
        hint="We'll respond to this address"
    />
    
    <SelectField 
        name="subject"
        label="Subject"
        required={true}
        placeholder="Select a topic..."
        options={[
            "General Inquiry",
            "Technical Support",
            "Sales Question",
            "Partnership"
        ]}
    />
    
    <TextareaField 
        name="message" 
        label="Message"
        required={true}
        maxlength={2000}
        counter={true}
        hint="Please be as detailed as possible"
    />
    
    <Checkbox 
        name="newsletter" 
        label="Subscribe to our newsletter"
    />
    
    <Button type="submit">"Send Message"</Button>
</Form>
```

---

## CSS Classes Reference

All components emit semantic CSS classes for styling:

| Component | Classes |
|-----------|---------|
| TextField, TextareaField, SelectField | `.field`, `.field-hint`, `.field-error`, `.field-required` |
| RadioGroup | `.radio-group`, `.radio-group-options`, `.radio-option`, `.radio-label` |
| CheckboxGroup | `.checkbox-group`, `.checkbox-group-options`, `.checkbox-option`, `.checkbox-label` |
| Checkbox | `.field-checkbox`, `.checkbox-label`, `.checkbox-text` |
| Button | `.button` |
| Form | `.form` |
| Breadcrumb | `.breadcrumb`, `.breadcrumb-list`, `.breadcrumb-item`, `.breadcrumb-separator` |
| DataTable | `.data-table` |
| Blockquote | `.blockquote` |
| Icon | `.icon`, `.icon-{name}` |
| SrOnly | `.sr-only` |
| SkipLink | `.skip-link` |

---

## Data Attributes for JavaScript Enhancement

Components emit data attributes that JavaScript can use for progressive enhancement:

| Attribute | Component | Purpose |
|-----------|-----------|---------|
| `data-confirm` | Form | Show confirmation dialog before submit |
| `data-toggle` | Button | Toggle visibility of target element |
| `data-copy` | Button | Copy text content of target element |
| `data-autosubmit` | SelectField | Submit form when value changes |
| `data-autoresize` | TextareaField | Auto-grow textarea with content |
| `data-counter` | TextareaField | ID of character counter element |

These attributes are inert without JavaScript—the components work without it, but can be enhanced.
