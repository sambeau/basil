---
id: man-pars-money
title: "Money"
system: parsley
type: builtin
name: money
created: 2024-12-14
version: 0.2.0
author: "@sam"
keywords: money, currency, finance, arithmetic, decimal, banking
---

## Money

Money values represent monetary amounts with exact decimal arithmetic. Unlike floating-point numbers, Money values never suffer from rounding errors—they store amounts as integers in the smallest currency unit (e.g., cents for USD). Each Money value carries its currency code and decimal scale, enabling safe arithmetic operations while preventing accidental mixing of different currencies.

```parsley
let price = $19.99
let tax = price * 0.0825
let total = price + tax

total  // $21.64
```

## Literals

Money literals can be written using currency symbols or the `CODE#amount` syntax:

### Symbol Syntax

Common currencies can use their symbol directly:

```parsley
$12.34      // US Dollars
£99.99      // British Pounds  
€50.00      // Euros
¥1000       // Japanese Yen (no decimal places)
```

### Code Syntax

Any ISO 4217 currency code can use the `CODE#amount` format:

```parsley
USD#12.34   // US Dollars
GBP#99.99   // British Pounds
EUR#50.00   // Euros
JPY#1000    // Japanese Yen
CHF#45.50   // Swiss Francs
BTC#1.5     // Bitcoin (custom currency)
```

### The `money()` Function

Create Money values programmatically:

```parsley
money(19.99, "USD")          // $19.99
money(1000, "JPY")           // ¥1000
money(12345, "USD", 2)       // $123.45 (amount in cents with explicit scale)
```

## Operators

### Addition (+)

Add two Money values of the same currency.

```parsley
$10.00 + $5.50  // $15.50
```

Attempting to add different currencies produces an error:

```parsley
$10.00 + £5.00  // Error: cannot perform arithmetic on different currencies
```

### Subtraction (-)

Subtract two Money values of the same currency.

```parsley
$20.00 - $7.50  // $12.50
```

### Multiplication (*)

Multiply a Money value by a number. Uses banker's rounding for exact results.

```parsley
$10.00 * 3      // $30.00
$19.99 * 0.0825 // $1.65 (tax calculation)
2.5 * $10.00    // $25.00 (scalar can be on either side)
```

### Division (/)

Divide a Money value by a number. Uses banker's rounding.

```parsley
$100.00 / 3     // $33.33
$50.00 / 4      // $12.50
```

### Comparison Operators

Compare Money values of the same currency:

```parsley
$10.00 < $20.00   // true
$15.00 > $10.00   // true
$10.00 <= $10.00  // true
$20.00 >= $15.00  // true
$10.00 == $10.00  // true
$10.00 != $20.00  // true
```

## Attributes

### amount

The raw amount in the smallest currency unit (e.g., cents).

```parsley
let price = $19.99
price.amount  // 1999
```

### currency

The ISO 4217 currency code as a string.

```parsley
let price = $19.99
price.currency  // "USD"

let pounds = £50.00
pounds.currency  // "GBP"
```

### scale

The number of decimal places for this currency.

```parsley
let dollars = $19.99
dollars.scale  // 2

let yen = ¥1000
yen.scale  // 0
```

## Methods

### abs()

Returns the absolute value of the Money amount.

```parsley
let loss = $0.00 - $50.00
loss       // -$50.00
loss.abs() // $50.00
```

### format()

#### Usage: format()

Format the Money value using the default locale (en-US).

```parsley
let price = $1234.56
price.format()  // "$1,234.56"
```

#### Usage: format(locale)

Format the Money value using a specific locale for number formatting.

```parsley
let price = €1234.56
price.format("de-DE")  // "1.234,56 €"
price.format("fr-FR")  // "1 234,56 €"
```

### split()

#### Usage: split(n)

Split a Money value into `n` parts that sum exactly to the original amount. This is useful for dividing bills or distributing payments fairly—any remainder cents are distributed one-per-part to the first parts.

```parsley
let bill = $100.00
bill.split(3)  // [$33.34, $33.33, $33.33]

let odd = $10.00
odd.split(3)   // [$3.34, $3.33, $3.33]
```

The parts always sum exactly to the original:

```parsley
let parts = $100.00.split(3)
parts[0] + parts[1] + parts[2]  // $100.00
```

### repr()

Returns a parseable literal representation of the money value:

```parsley
$50.00.repr()       // "$50.00"
EUR#25.50.repr()    // "€25.50"
JPY#1000.repr()     // "¥1000"
```

### toDict()

Returns a clean dictionary for reconstruction (without `__type`):

```parsley
$50.00.toDict()
// {amount: 50.0, currency: "USD"}

EUR#25.50.toDict()
// {amount: 25.5, currency: "EUR"}
```

You can use this for round-trip reconstruction:

```parsley
let original = $50.00
let reconstructed = money(original.toDict())
original == reconstructed  // true
```

### inspect()

Returns a debug dictionary with `__type` and raw values (amount in smallest unit):

```parsley
$50.00.inspect()
// {__type: "money", amount: 5000, currency: "USD", scale: 2}

EUR#25.50.inspect()
// {__type: "money", amount: 2550, currency: "EUR", scale: 2}
```

Note: The `amount` in `inspect()` is in the smallest currency unit (cents), while `toDict()` returns a user-friendly decimal amount.

## Currency Scales

Parsley knows the correct decimal scale for major world currencies per ISO 4217:

| Scale | Currencies |
|-------|------------|
| 0 | JPY, KRW, VND, CLP |
| 2 | USD, EUR, GBP, CHF, CAD, AUD, CNY, and most others |
| 3 | KWD, BHD, OMR, JOD |

For unknown currency codes (e.g., `BTC#1.5`), the scale is inferred from the literal or defaults to 2.

## Best Practices

### Always Use Money for Financial Calculations

```parsley
// ✅ Correct - exact arithmetic
let subtotal = $19.99
let tax = subtotal * 0.0825
let total = subtotal + tax  // $21.64

// ❌ Avoid - floating point errors accumulate
let subtotal = 19.99
let tax = subtotal * 0.0825  // 1.649175 (imprecise)
```

### Use split() for Fair Division

```parsley
// ✅ Correct - all parts sum to original
let shares = $100.00.split(3)  // [$33.34, $33.33, $33.33]

// ❌ Avoid - loses a cent
let share = $100.00 / 3  // $33.33
let total = share * 3    // $99.99 (missing $0.01!)
```

### Keep Currency Consistent

```parsley
// ✅ Correct - same currency
let usd1 = $50.00
let usd2 = $25.00
let total = usd1 + usd2  // $75.00

// ❌ Error - different currencies
let mixed = $50.00 + £25.00  // Error!
```
