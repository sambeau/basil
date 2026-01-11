# Parsley Standard Library Inventory

> Generated from source code audit of `pkg/parsley/evaluator/stdlib_*.go`

## Module Registry

From `getStdlibModules()` in stdlib_table.go:

| Module | Loader Function | Description |
|--------|-----------------|-------------|
| `table` | loadTableModule | Table module access |
| `dev` | loadDevModule | Development logging |
| `math` | loadMathModule | Math functions |
| `valid` | loadValidModule | Validation functions |
| `schema` | loadSchemaModule | Schema definition and validation |
| `id` | loadIDModule | ID generation |
| `api` | loadAPIModule | API auth wrappers and helpers |
| `mdDoc` | loadMdDocModule | Markdown documentation |
| `html` | loadHTMLModule | HTML components (from prelude) |

---

## @std/math Module

### Constants

| Name | Value | Description |
|------|-------|-------------|
| `PI` | 3.14159... | Pi |
| `E` | 2.71828... | Euler's number |
| `TAU` | 6.28318... | 2π |

### Rounding Functions

| Function | Arity | Description |
|----------|-------|-------------|
| `floor` | 1 | Round down to integer |
| `ceil` | 1 | Round up to integer |
| `round` | 1 | Round to nearest integer |
| `trunc` | 1 | Truncate to integer |

### Comparison & Clamping

| Function | Arity | Description |
|----------|-------|-------------|
| `abs` | 1 | Absolute value |
| `sign` | 1 | Sign of number (-1, 0, 1) |
| `clamp` | 3 | Clamp value between min and max |

### Aggregation (2 args OR array)

| Function | Arity | Description |
|----------|-------|-------------|
| `min` | 1-2+ | Minimum of values |
| `max` | 1-2+ | Maximum of values |
| `sum` | 1 | Sum of array |
| `avg` / `mean` | 1 | Average of array |
| `product` | 1 | Product of array |
| `count` | 1 | Count of array |

### Statistics (array only)

| Function | Arity | Description |
|----------|-------|-------------|
| `median` | 1 | Median of array |
| `mode` | 1 | Mode of array |
| `stddev` | 1 | Standard deviation |
| `variance` | 1 | Variance |
| `range` | 1 | Range (max - min) |

### Random

| Function | Arity | Description |
|----------|-------|-------------|
| `random` | 0 | Random float 0-1 |
| `randomInt` | 1-2 | Random integer in range |
| `seed` | 1 | Seed the RNG |

### Powers & Logarithms

| Function | Arity | Description |
|----------|-------|-------------|
| `sqrt` | 1 | Square root |
| `pow` | 2 | Power (base^exponent) |
| `exp` | 1 | e^x |
| `log` | 1 | Natural logarithm |
| `log10` | 1 | Base-10 logarithm |

### Trigonometry

| Function | Arity | Description |
|----------|-------|-------------|
| `sin` | 1 | Sine |
| `cos` | 1 | Cosine |
| `tan` | 1 | Tangent |
| `asin` | 1 | Arc sine |
| `acos` | 1 | Arc cosine |
| `atan` | 1 | Arc tangent |
| `atan2` | 2 | Arc tangent of y/x |

### Angular Conversion

| Function | Arity | Description |
|----------|-------|-------------|
| `degrees` | 1 | Radians to degrees |
| `radians` | 1 | Degrees to radians |

### Geometry & Interpolation

| Function | Arity | Description |
|----------|-------|-------------|
| `hypot` | 2 | Hypotenuse √(x² + y²) |
| `dist` | 4 | Distance between points |
| `lerp` | 3 | Linear interpolation |
| `map` | 5 | Map value from one range to another |

---

## @std/valid Module

### Type Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `string` | 1 | Check if string |
| `number` | 1 | Check if number (int or float) |
| `integer` | 1 | Check if integer |
| `boolean` | 1 | Check if boolean |
| `array` | 1 | Check if array |
| `dict` | 1 | Check if dictionary |

### String Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `empty` | 1 | Check if empty or whitespace |
| `minLen` | 2 | Check minimum length |
| `maxLen` | 2 | Check maximum length |
| `length` | 3 | Check length in range |
| `matches` | 2 | Check regex match |
| `alpha` | 1 | Check letters only |
| `alphanumeric` | 1 | Check letters/numbers only |
| `numeric` | 1 | Check digits only |

### Number Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `min` | 2 | Check minimum value |
| `max` | 2 | Check maximum value |
| `between` | 3 | Check value in range |
| `positive` | 1 | Check positive |
| `negative` | 1 | Check negative |

### Format Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `email` | 1 | Validate email format |
| `url` | 1 | Validate URL format |
| `uuid` | 1 | Validate UUID format |
| `phone` | 1 | Validate phone format |
| `creditCard` | 1 | Validate credit card (Luhn) |
| `date` | 1-2 | Validate date format |
| `time` | 1 | Validate time format |

### Locale-aware Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `postalCode` | 2 | Validate postal code by locale |
| `parseDate` | 2 | Parse date by locale |

### Collection Validators

| Function | Arity | Description |
|----------|-------|-------------|
| `contains` | 2 | Check array contains value |
| `oneOf` | 2 | Check value is one of options |

---

## @std/id Module

| Function | Arity | Description |
|----------|-------|-------------|
| `new` | 0 | Generate ULID-like ID (26 chars, sortable) |
| `uuid` / `uuidv4` | 0 | Generate UUID v4 (random) |
| `uuidv7` | 0 | Generate UUID v7 (time-sortable) |
| `nanoid` | 0-1 | Generate NanoID (default 21 chars) |
| `cuid` | 0 | Generate CUID2-like ID |

---

## @std/api Module

### Auth Wrappers

| Function | Arity | Description |
|----------|-------|-------------|
| `public` | 1-2 | Mark function as public (no auth) |
| `adminOnly` | 1 | Require admin role |
| `roles` | 2 | Require specific roles |
| `auth` | 1-2 | Custom auth options |

### Error Helpers

| Function | Arity | Description |
|----------|-------|-------------|
| `notFound` | 0-1 | Return 404 error |
| `forbidden` | 0-1 | Return 403 error |
| `badRequest` | 0-1 | Return 400 error |
| `unauthorized` | 0-1 | Return 401 error |
| `conflict` | 0-1 | Return 409 error |
| `serverError` | 0-1 | Return 500 error |

### Redirect Helper

| Function | Arity | Description |
|----------|-------|-------------|
| `redirect` | 1-2 | Return redirect response |

---

## @std/schema Module

### Type Factories

| Function | Arity | Description |
|----------|-------|-------------|
| `string` | 0-1 | String type spec |
| `email` | 0-1 | Email type spec |
| `url` | 0-1 | URL type spec |
| `phone` | 0-1 | Phone type spec |
| `integer` | 0-1 | Integer type spec |
| `number` | 0-1 | Number type spec |
| `boolean` | 0-1 | Boolean type spec |
| `enum` | 1+ | Enum type spec with values |
| `date` | 0-1 | Date type spec |
| `datetime` | 0-1 | Datetime type spec |
| `money` | 0-1 | Money type spec |
| `array` | 0-1 | Array type spec |
| `object` | 0-1 | Object type spec |
| `id` | 0-1 | ID type spec (default ULID) |

### Schema Operations

| Function | Arity | Description |
|----------|-------|-------------|
| `define` | 2 | Create schema from name and fields dict |
| `table` | ? | Create table-bound schema |

### Schema Methods (on schema objects)

| Method | Description |
|--------|-------------|
| `.validate(data)` | Validate data against schema |

---

## @std/dev Module

| Function | Arity | Description |
|----------|-------|-------------|
| `log` | 1-3 | Log to dev panel |
| `clearLog` | 0 | Clear dev log |
| `logPage` | 2-4 | Log to specific page route |
| `setLogRoute` | 1 | Set default log route |
| `clearLogPage` | 1 | Clear log for specific page |

---

## @std/html Module

Components loaded from prelude (if available):

### Layout Components

| Component | Description |
|-----------|-------------|
| `Page` | Page wrapper |
| `Head` | HTML head |

### Form Components

| Component | Description |
|-----------|-------------|
| `TextField` | Text input field |
| `TextareaField` | Textarea field |
| `SelectField` | Select dropdown |
| `RadioGroup` | Radio button group |
| `CheckboxGroup` | Checkbox group |
| `Checkbox` | Single checkbox |
| `Button` | Button |
| `Form` | Form wrapper |

### Navigation Components

| Component | Description |
|-----------|-------------|
| `Nav` | Navigation |
| `Breadcrumb` | Breadcrumb trail |
| `SkipLink` | Accessibility skip link |

### Media Components

| Component | Description |
|-----------|-------------|
| `Img` | Image |
| `Iframe` | Iframe |
| `Figure` | Figure with caption |
| `Blockquote` | Blockquote |

### Utility Components

| Component | Description |
|-----------|-------------|
| `SrOnly` | Screen reader only |
| `Abbr` | Abbreviation |
| `A` | Anchor link |
| `Icon` | Icon |

### Time Components

| Component | Description |
|-----------|-------------|
| `Time` | Time element |
| `LocalTime` | Localized time |
| `TimeRange` | Time range |
| `RelativeTime` | Relative time (e.g., "2 hours ago") |

### Table Components

| Component | Description |
|-----------|-------------|
| `DataTable` | Data table |

---

## Summary

| Module | Export Count | Primary Use |
|--------|--------------|-------------|
| math | 37 | Mathematical operations |
| valid | 28 | Data validation |
| id | 5 | ID generation |
| api | 11 | API helpers |
| schema | 17 | Schema definition/validation |
| dev | 5 | Development tools |
| html | 22 | HTML components |

**Note**: `table` module provides access to `Table` type. `mdDoc` module provides markdown documentation utilities.
