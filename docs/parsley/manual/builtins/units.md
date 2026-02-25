---
id: man-pars-unit
title: "Units"
system: parsley
type: builtin
name: unit
created: 2026-02-24
version: 0.2.0
author: "@sam"
keywords: unit, measurement, length, mass, volume, temperature, area, data, metric, imperial, SI, US, conversion
---

## Units

Unit values represent physical measurements with automatic unit tracking and conversion. Each Unit carries its numeric value, measurement family (length, mass, volume, etc.), and system (SI or US Customary). Parsley stores units internally using integer arithmetic with high precision, eliminating floating-point errors common in measurement calculations.

```parsley
let height = #5ft + #10in
let area = #5m * #3m
let converted = #100C.to("F")

height     // #5+5/6ft
area       // #15m2
converted  // #212F
```

## Literals

Unit literals use the `#` sigil followed by a number and suffix:

### Decimal Syntax

```parsley
#12.3m      // 12.3 metres
#5.5kg      // 5.5 kilograms  
#100C       // 100 degrees Celsius
#3.785L     // 3.785 litres
```

### Fraction Syntax (US Customary)

US Customary units support exact fraction notation:

```parsley
#3/8in      // three-eighths of an inch
#92+5/8in   // 92 and 5/8 inches
#1+1/2ft    // one and a half feet
#2+1/4lb    // two and a quarter pounds
```

### Negative Values

```parsley
#-6m        // negative 6 metres
#-40C       // negative 40 degrees Celsius
#-3/8in     // negative three-eighths inch
```

## Supported Units

### Length

| Suffix | Unit | System |
|--------|------|--------|
| `mm` | millimetres | SI |
| `cm` | centimetres | SI |
| `m` | metres | SI |
| `km` | kilometres | SI |
| `in` | inches | US |
| `ft` | feet | US |
| `yd` | yards | US |
| `mi` | miles | US |

### Area

| Suffix | Unit | System |
|--------|------|--------|
| `mm2` | square millimetres | SI |
| `cm2` | square centimetres | SI |
| `m2` | square metres | SI |
| `km2` | square kilometres | SI |
| `in2` | square inches | US |
| `ft2` | square feet | US |
| `yd2` | square yards | US |
| `ac` | acres | US |
| `mi2` | square miles | US |

### Mass

| Suffix | Unit | System |
|--------|------|--------|
| `mg` | milligrams | SI |
| `g` | grams | SI |
| `kg` | kilograms | SI |
| `oz` | ounces | US |
| `lb` | pounds | US |

### Volume

| Suffix | Unit | System |
|--------|------|--------|
| `mL` | millilitres | SI |
| `L` | litres | SI |
| `kL` | kilolitres | SI |
| `floz` | fluid ounces | US |
| `cup` | cups | US |
| `pt` | pints | US |
| `qt` | quarts | US |
| `gal` | gallons | US |

### Temperature

| Suffix | Unit | System |
|--------|------|--------|
| `C` | Celsius | SI |
| `K` | Kelvin | SI |
| `F` | Fahrenheit | US |

### Data

| Suffix | Unit | System |
|--------|------|--------|
| `B` | bytes | SI |
| `kB` | kilobytes (decimal) | SI |
| `MB` | megabytes | SI |
| `GB` | gigabytes | SI |
| `TB` | terabytes | SI |
| `KiB` | kibibytes (binary) | SI |
| `MiB` | mebibytes | SI |
| `GiB` | gibibytes | SI |
| `TiB` | tebibytes | SI |

## Constructor Functions

### Named Constructors

Create units programmatically using named constructor functions:

```parsley
metres(5)        // #5m
feet(3)          // #3ft
kilograms(2.5)   // #2.5kg
celsius(100)     // #100C
```

Named constructors also convert existing units:

```parsley
metres(#12in)    // #0.3048m (converts 12 inches to metres)
inches(#1m)      // #39.37...in (converts 1 metre to inches)
```

Available constructors (both British and American spellings):

- **Length**: `millimetres`/`millimeters`, `centimetres`/`centimeters`, `metres`/`meters`, `kilometres`/`kilometers`, `inches`, `feet`, `yards`, `miles`
- **Area**: `millimetres2`/`millimeters2`, `centimetres2`/`centimeters2`, `metres2`/`meters2`, `kilometres2`/`kilometers2`, `inches2`, `feet2`, `yards2`, `acres`, `miles2`
- **Mass**: `milligrams`, `grams`, `kilograms`, `ounces`, `pounds`
- **Volume**: `millilitres`/`milliliters`, `litres`/`liters`, `kilolitres`/`kiloliters`, `fluidounces`, `cups`, `pints`, `quarts`, `gallons`
- **Temperature**: `celsius`, `fahrenheit`, `kelvins`
- **Data**: `bytes`, `kilobytes`, `megabytes`, `gigabytes`, `terabytes`, `kibibytes`, `mebibytes`, `gibibytes`, `tebibytes`

### The `unit()` Function

Create units with a dynamic suffix:

```parsley
unit(123, "m")   // #123m
unit(12, "in")   // #12in
unit(25, "C")    // #25C
```

Convert existing units:

```parsley
unit(#5ft, "m")  // converts 5 feet to metres
```

## Operators

### Addition (+)

Add two units of the same family. Cross-system addition is allowed—the left operand determines the result's system and display unit:

```parsley
#5m + #3m        // #8m
#1ft + #6in      // #1+1/2ft
#1cm + #1in      // #3.54cm (cross-system, result in SI)
#1in + #1cm      // #1.39in (cross-system, result in US)
```

Different families produce an error:

```parsley
#5m + #3kg       // Error: cannot add length and mass
```

### Subtraction (-)

Subtract two units of the same family:

```parsley
#10m - #3m       // #7m
#1ft - #6in      // #1/2ft
#100C - #32C     // #68C
```

### Multiplication (*)

Multiply a unit by a scalar:

```parsley
#10m * 3         // #30m
3 * #10m         // #30m (scalar can be on either side)
#5in * 2         // #10in
```

Multiply two lengths to get area (derived unit arithmetic):

```parsley
#5m * #3m        // #15m2
#2ft * #3ft      // #6ft2
#10cm * #10cm    // #100cm2
```

Cross-system derived multiplication is not allowed:

```parsley
#5m * #3ft       // Error: cannot multiply SI length by US length
```

### Division (/)

Divide a unit by a scalar:

```parsley
#10m / 2         // #5m
#1ft / 2         // #1/2ft
#12in / 4        // #3in
```

Divide two units of the same family to get a dimensionless ratio:

```parsley
#10m / #5m       // 2
#1ft / #6in      // 2
```

Divide area by length to get length (derived unit arithmetic):

```parsley
#15m2 / #3m      // #5m
#6ft2 / #2ft     // #3ft
```

Cross-system derived division is not allowed:

```parsley
#15m2 / #3ft     // Error: cannot divide SI area by US length
```

### Comparison Operators

Compare units of the same family:

```parsley
#5m > #3m        // true
#1ft < #1m       // true (cross-system comparison)
#100cm == #1m    // true
#16oz == #1lb    // true
#212F == #100C   // true (same absolute temperature)
```

## Attributes

### value

The numeric value in the display unit:

```parsley
#12m.value       // 12
#12.5m.value     // 12.5
#3/8in.value     // 0.375
```

### unit

The unit suffix as a string:

```parsley
#12m.unit        // "m"
#5ft.unit        // "ft"
#100C.unit       // "C"
```

### family

The measurement family as a string:

```parsley
#12m.family      // "length"
#5kg.family      // "mass"
#100C.family     // "temperature"
#15m2.family     // "area"
```

### system

The measurement system as a string:

```parsley
#12m.system      // "SI"
#5ft.system      // "US"
#100C.system     // "SI"
#212F.system     // "US"
```

### max

The maximum representable value for this unit:

```parsley
#5m.max          // #9223372036854m (very large)
#100C.max        // extremely large temperature
```

### min

The minimum representable value for this unit. For temperature, this is absolute zero:

```parsley
#5m.min          // #0.000001m (smallest precision)
#100C.min        // #-273.15C (absolute zero)
#32F.min         // #-459.67F (absolute zero)
```

## Methods

### fmt()

The primary formatting method with multiple overloads.

#### Usage: fmt()

Format with default style (medium) and locale (en-US):

```parsley
#5m.fmt()        // "5.00m"
#12.3m.fmt()     // "12.30m"
#5ft.fmt()       // "5.00ft"
```

#### Usage: fmt(precision)

Format with a specific number of decimal places:

```parsley
#12.345m.fmt(2)  // "12.35m"
#3.14159m.fmt(3) // "3.142m"
```

#### Usage: fmt(style)

Format with a named style:

```parsley
let u = #5m

u.fmt("short")   // "5m"
u.fmt("medium")  // "5.00m"
u.fmt("long")    // "5.00 meters"
u.fmt("full")    // "5.00 meters (16.4 ft)"
```

#### Usage: fmt(style, locale)

Format with style and locale:

```parsley
#5m.fmt("long", "en-GB")   // "5.00 metres"
#5m.fmt("long", "en-US")   // "5.00 meters"
```

#### Usage: fmt(options)

Format with an options dictionary:

```parsley
#5m.fmt({style: "long"})                     // "5.00 meters"
#5m.fmt({style: "full", locale: "en-GB"})    // "5.00 metres (16.4 ft)"
#12.345m.fmt({precision: 2})                 // "12.35m"
```

#### Usage: fmt(formatString)

Format using a compound format string:

```parsley
#63in.fmt("ft-in")      // "5' 3\""
#63.375in.fmt("ft-in")  // "5' 3+3/8\""
#37oz.fmt("lb-oz")      // "2lb 5oz"
#1500mL.fmt("L-mL")     // "1L 500mL"
#13pt.fmt("gal-qt-pt")  // "1gal 2qt 1pt"
```

The `"compound"` format auto-detects the appropriate compound format:

```parsley
#63in.fmt("compound")   // "5' 3\"" (feet-inches)
#37oz.fmt("compound")   // "2lb 5oz" (pounds-ounces)
#1500mL.fmt("compound") // "1L 500mL" (litres-millilitres)
```

Cross-system auto-conversion is supported:

```parsley
#1.8288m.fmt("ft-in")   // "6'" (auto-converts to US)
#1gal.fmt("L-mL")       // "3L 785.412mL" (auto-converts to SI)
```

### short()

Compact format with unit suffix only:

```parsley
#5m.short()           // "5m"
#12.3m.short()        // "12.3m"
#5ft.short()          // "5ft"
#3/8in.short()        // "3/8in"

// With locale:
#5m.short("de-DE")    // "5m"
```

### medium()

Standard format with decimal precision (default style):

```parsley
#5m.medium()          // "5.00m"
#12.3m.medium()       // "12.30m"
#5ft.medium()         // "5.00ft"

// With locale:
#5m.medium("de-DE")   // "5,00m"
```

### long()

Verbose format with full unit name:

```parsley
#5m.long()            // "5.00 meters"
#12.3m.long()         // "12.30 meters"
#1m.long()            // "1.00 meter"
#5ft.long()           // "5.00 feet"
#1ft.long()           // "1.00 foot"
#100C.long()          // "100.00 degrees Celsius"

// With locale (affects spelling):
#5m.long("en-GB")     // "5.00 metres"
#5m.long("en-US")     // "5.00 meters"
```

### full()

Maximum context with cross-system conversion:

```parsley
#5m.full()            // "5.00 meters (16.4 ft)"
#5ft.full()           // "5.00 feet (1.52 m)"
#100C.full()          // "100.00 degrees Celsius (212 °F)"
#212F.full()          // "212.00 degrees Fahrenheit (100 °C)"
#5kg.full()           // "5.00 kilograms (11.02 lb)"

// With locale:
#5m.full("en-GB")     // "5.00 metres (16.4 ft)"
```

### format()

Alias for `fmt()`. Retained for backward compatibility:

```parsley
#12.3m.format()       // "12.30m"
#12.3m.format(2)      // "12.30m"
#63in.format("ft-in") // "5' 3\""
```

### abs()

Returns the absolute value of the unit:

```parsley
#-5m.abs()       // #5m
#-40C.abs()      // #40C
```

### to()

#### Usage: to(targetSuffix)

Convert to a different unit within the same family:

```parsley
#1mi.to("km")    // #1.609344km
#1in.to("cm")    // #2.54cm
#100C.to("F")    // #212F
#32F.to("C")     // #0C
#1kg.to("lb")    // #2.205lb
#1gal.to("L")    // #3.785L
```

Converting between incompatible families produces an error:

```parsley
#5m.to("kg")     // Error: cannot convert length to mass
```

### repr()

Returns a parseable literal representation:

```parsley
#12m.repr()      // "#12m"
#3/8in.repr()    // "#3/8in"
#5.5kg.repr()    // "#5.5kg"
```

### toJSON()

Returns the JSON representation:

```parsley
#5m.toJSON()     // "{\"value\":5,\"unit\":\"m\",\"family\":\"length\",\"system\":\"SI\"}"
```

### inspect()

Returns a debug dictionary with internal representation details:

```parsley
#12m.inspect()
// {__type: "unit", amount: 12000000, displayHint: "m", family: "length", scale: 0, system: "SI"}
```

### toDict()

Returns a dictionary with the unit's properties:

```parsley
#12m.toDict()
// {value: 12, unit: "m", family: "length", system: "SI"}

#100C.toDict()
// {value: 100, unit: "C", family: "temperature", system: "SI"}
```

### toFraction()

Returns a fraction string for US Customary values:

```parsley
#3/8in.toFraction()    // "3/8\""
#1+1/2ft.toFraction()  // "1+1/2'"
#12m.toFraction()      // "12m" (SI returns decimal)
```

### toBox()

Renders the unit in a box diagram:

```parsley
#5m.toBox()
```

```
┌───────────────┐
│ 5.00 meters   │
└───────────────┘
```

## Formatting Styles Summary

| Style | Example (`#5m`) | Description |
|-------|-----------------|-------------|
| `short` | `"5m"` | Compact with suffix |
| `medium` (default) | `"5.00m"` | Decimal precision with suffix |
| `long` | `"5.00 meters"` | Full unit name |
| `full` | `"5.00 meters (16.4 ft)"` | With cross-system conversion |

## Temperature Arithmetic

Temperature arithmetic uses special offset-based formulas to ensure physically correct results:

```parsley
// Adding temperatures (represents temperature changes)
#20C + #5C       // #25C

// Subtracting temperatures
#100C - #32C     // #68C

// Comparing temperatures (compares absolute values)
#100C == #212F   // true (both are boiling point of water)
#0C == #32F      // true (both are freezing point of water)
```

Note: Multiplying or dividing two temperature values is not allowed, as it's not physically meaningful.

## Derived Unit Arithmetic

Length multiplication produces area:

```parsley
#5m * #3m        // #15m2 (square metres)
#2ft * #3ft      // #6ft2 (square feet)
```

Area division by length produces length:

```parsley
#15m2 / #3m      // #5m
#6ft2 / #2ft     // #3ft
```

The left operand's display hint determines the result's unit:

```parsley
#5m * #300cm     // #15m2 (left is metres, result in m2)
#100cm * #1m     // #10000cm2 (left is centimetres, result in cm2)
```

Cross-system derived arithmetic requires explicit conversion:

```parsley
// This produces an error:
#5m * #3ft       // Error: cannot multiply SI by US

// Convert first:
#5m * #3ft.to("m")  // Works!
```

## Precision and Storage

Parsley stores unit values using high-precision integer arithmetic:

- **SI units**: Stored in the smallest practical sub-unit (µm for length, mg for mass, nL for volume)
- **US Customary units**: Stored using HCN (Highly Composite Number = 725,760) sub-units, making all common fractions exact

This means operations like `#1/8in + #1/8in + #1/8in` produce exactly `#3/8in` with no floating-point error.

## Schema Integration

Units can be validated in schemas using family names or specific suffixes:

```parsley
@schema Product {
    weight: mass,           // accepts any mass unit
    height: length,         // accepts any length unit
    temp: temperature       // accepts any temperature unit
}

@schema MetricProduct {
    weight: unit(suffix: "kg"),   // must be in kilograms
    height: unit(suffix: "cm")    // must be in centimetres
}
```

## String Interpolation

Units interpolate naturally in template strings:

```parsley
let height = #5ft + #10in
`Height: {height}`  // "Height: 5+5/6ft"

let temp = #100C
`Temperature: {temp}`  // "Temperature: 100C"
```

## Best Practices

### Use Appropriate Units for Your Domain

```parsley
// ✅ SI for scientific work
let distance = #5.5km
let mass = #1.5kg

// ✅ US Customary for US construction
let lumber = #8ft
let nail = #3in
```

### Keep Units Consistent in Calculations

```parsley
// ✅ Same system - simple and clear
let width = #5m
let length = #10m
let area = width * length  // #50m2

// ⚠️ Mixed systems - works but may be confusing
let a = #5m + #10ft  // Works, but consider converting first
```

### Use Style Methods for Display

```parsley
let height = #1.83m

// For compact displays
height.short()       // "1.83m"

// For standard displays
height.medium()      // "1.83m"

// For formal documents
height.long()        // "1.83 meters"

// For international contexts
height.full()        // "1.83 meters (6 ft)"
```

### Use Compound Formatting for User Display

```parsley
let height = #68in
height.fmt("ft-in")  // "5' 8\"" - more readable for users
```

### Convert Early When Mixing Systems

```parsley
// ✅ Convert explicitly for clarity
let siLength = #5ft.to("m")
let area = siLength * #3m  // Now both are SI
```

## See Also

- [Money](money.md) — monetary values with exact arithmetic
- [Numbers](numbers.md) — integer and float arithmetic
- [Schema](schema.md) — validating units in data structures
- [Strings](strings.md) — unit formatting and interpolation