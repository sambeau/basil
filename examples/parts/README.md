# Parts Example

This example demonstrates the Parts feature in Basil - reloadable HTML fragments with multiple view functions.

## What are Parts?

Parts are interactive components that can update themselves without a full page reload. They're perfect for:

- Counters and interactive widgets
- Real-time data displays
- Form submissions with inline updates
- Progressive enhancement

## Running the Example

```bash
./basil examples/parts
```

Then visit `http://localhost:3000` in your browser.

## How It Works

### 1. Part Files

Parts are defined in `.part` files (e.g., `counter.part`). Each Part exports view functions:

```parsley
export default = fn(props) {
  // Default view
}

export increment = fn(props) {
  // Increment view
}

export decrement = fn(props) {
  // Decrement view
}
```

### 2. Using Parts in Pages

Include a Part in your page with the `<Part/>` component:

```parsley
<Part src={@./handlers/counter.part} view="default" count={0}/>
```

### 3. Interactive Elements

Use `part-click` and `part-submit` attributes for interactivity:

```html
<button part-click="increment" part-count={count}>+</button>
<button part-click="decrement" part-count={count}>-</button>
<form part-submit="reset">
  <button type="submit">Reset</button>
</form>
```

### 4. Prop Passing

Props can be passed as attributes and are accessible in view functions:

- Part attributes become props (e.g., `count={5}` → `props.count`)
- Element attributes with `part-` prefix pass data (e.g., `part-count={count}`)
- Form data is automatically included on `part-submit`

## Architecture

1. **Server-side**: 
   - Part files are loaded and executed like regular Parsley modules
   - View functions return HTML fragments
   - Props are type-coerced (integers, booleans, strings)

2. **Client-side**: 
   - JavaScript runtime handles event delegation
   - Fetches Part URLs with `?_view=viewName&prop=value`
   - Updates innerHTML with returned HTML
   - Re-initializes nested Parts after updates

3. **Error handling**:
   - Failed updates leave old content visible
   - Loading states with `.part-loading` class
   - Console logs for debugging

## File Structure

```
examples/parts/
├── basil.yaml          # Basil configuration
├── README.md           # This file
├── handlers/
│   ├── index.pars      # Main page
│   └── counter.part    # Counter Part with multiple views
└── public/
    └── styles.css      # Optional styles including loading states
```

## Features Demonstrated

- Multiple view functions in a single Part
- State management through props
- Click handlers with `part-click`
- Form submission with `part-submit`
- Loading states
- Error handling

## Important: Standards Mode CSS

If you include `<!DOCTYPE html>` in your page (recommended), add this CSS to ensure Parts render correctly:

```css
[data-part-src] {
  display: contents;
}
```

Without this, Part elements will collapse to inline elements in standards mode. The `display: contents` makes the Part wrapper transparent so the actual content determines the layout.
