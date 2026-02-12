---
id: FEAT-116
title: "PLN Native Money and DateTime Serialization"
status: complete
priority: medium
created: 2026-02-12
completed: 2026-02-12
author: "@ai"
related: FEAT-098, FEAT-115
blocking: false
---

# FEAT-116: PLN Native Money and DateTime Serialization

## Summary

Extend PLN to support native `Money` literals, enabling true round-trip serialization where Money values written to PLN files are read back as native Money objects (not dictionaries). This uses the same literal syntax as Parsley's main lexer (`USD#19.99`, `JPY#500`, etc.).

## User Story

As a Parsley developer, I want to write native Money and DateTime values to PLN files so that I don't have to manually convert them to dictionaries before serialization.

## Problem Statement

### Previous Behavior

Before this feature, Money values would either:
1. Fail to serialize with "cannot serialize type *evaluator.Money"
2. Serialize as dictionaries `{amount: 1000, currency: "USD", scale: 2}` which read back as dictionaries, not Money

This defeated the purpose of PLN—if you're just getting dictionaries back, you might as well use JSON.

### Solution

Use **literal notation** for Money in PLN, matching Parsley's existing money literal syntax:

```parsley
// Write a Money value
let price = money(19.99, "USD")
price ==> PLN(@./data.pln)

// PLN file contains: USD#19.99

// Read it back - returns native Money, not a dictionary!
let loaded <== PLN(@./data.pln)
loaded + loaded  // $39.98 - arithmetic works because it's a real Money object
```

## Acceptance Criteria

### Core Functionality
- [x] Native `*evaluator.Money` values serialize to PLN literal format (`USD#19.99`)
- [x] Money values round-trip correctly: write then read returns equivalent value
- [x] DateTime dict values continue to serialize correctly (already works)
- [x] Serialized Money format is compatible with existing PLN deserializer

### Output Format

Money serializes to literal notation matching Parsley's money syntax:

| Money Value | PLN Literal |
|-------------|-------------|
| `money(19.99, "USD")` | `USD#19.99` |
| `money(500, "JPY")` | `JPY#500` |
| `money(-10.50, "EUR")` | `EUR#-10.50` |
| `money(1000.00, "GBP")` | `GBP#1000.00` |

Nested in structures:
```pln
{items: [{name: "Widget", price: USD#19.99}], tax: USD#5.60}
```

### Testing
- [x] Unit test: serialize native Money
- [x] Unit test: round-trip Money through PLN file
- [x] Unit test: serialize Money in nested structures (arrays, dicts)
- [x] Integration test: Money field in Record serializes correctly

## Design Decisions

### Money Literal Format

Use `CODE#amount` format (e.g., `USD#19.99`) matching Parsley's internal money literal representation:

**Rationale:**
- Consistent with Parsley's main lexer which uses this format internally
- Unambiguous—no confusion with other numeric types
- Supports all ISO 4217 currency codes
- Handles currencies with different decimal places (JPY=0, USD=2, KWD=3)

### DateTime Already Works

DateTime values serialize to `@YYYY-MM-DD` or `@YYYY-MM-DDTHH:MM:SS` literal format (already implemented in FEAT-098).

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components

| Component | Location | Change Type | Notes |
|-----------|----------|-------------|-------|
| Lexer | `pkg/parsley/pln/lexer.go` | Add MONEY token | Parse `$`, `£`, `€`, `¥`, `CODE#` formats |
| Parser | `pkg/parsley/pln/parser.go` | Add parseMoney() | Return `*evaluator.Money` |
| Serializer | `pkg/parsley/pln/serializer.go` | Add case | Output `CODE#amount` format |
| Tests | `pkg/parsley/pln/serializer_test.go` | New tests | Serialization and round-trip tests |

### Implementation

### Lexer Changes (`pkg/parsley/pln/lexer.go`)

Added `MONEY` token type and `readMoneyLiteral()` to handle:
- Currency symbols: `$`, `£`, `€`, `¥`
- CODE# format: `USD#19.99`, `JPY#500`
- Negative amounts: `EUR#-10.50`

### Parser Changes (`pkg/parsley/pln/parser.go`)

Added `parseMoney()` that:
- Parses the `CODE#amount` literal format
- Returns a native `*evaluator.Money` object
- Handles negative amounts and varying decimal scales

### Serializer Changes (`pkg/parsley/pln/serializer.go`)

Added `serializeMoney()` that outputs literal format:

```go
func (s *Serializer) serializeMoney(m *evaluator.Money) (string, error) {
    // Output: CODE#amount (e.g., USD#19.99, JPY#500)
    if m.Scale == 0 {
        return fmt.Sprintf("%s#%d", m.Currency, m.Amount), nil
    }
    // ... handle decimals and negative amounts
}
```

### Dependencies

- Depends on: FEAT-098 (PLN infrastructure)
- Related: FEAT-115 (PLN write support)

### Effort Estimate

- Implementation: 2 hours (lexer + parser + serializer)
- Testing: 1 hour
- **Total: 3 hours**

## Testing Plan

### Unit Tests (implemented in `serializer_test.go`)

- `TestSerializeMoney` - Tests literal output format for USD, JPY, EUR, GBP, KWD
- `TestSerializeMoneyPretty` - Verifies literals are compact even in pretty mode
- `TestSerializeMoneyInArray` - Tests `[USD#1.00, EUR#2.00]`
- `TestSerializeMoneyInDict` - Tests `{name: "Widget", price: USD#19.99}`
- `TestMoneyRoundTrip` - Verifies serialize→parse returns `*evaluator.Money` with correct values

## Notes

- DateTime already uses literal format (`@2024-01-15`) and round-trips correctly
- The PLN lexer now supports the same money literal formats as Parsley's main lexer
- Currency scales are validated (e.g., JPY doesn't allow decimal places)

## Complete PLN Type Support

All PLN-serializable types now use **literal notation** instead of dictionaries:

| Type | PLN Literal | Round-trips | Notes |
|------|-------------|-------------|-------|
| Integer | `42` | ✅ | |
| Float | `3.14` | ✅ | |
| String | `"hello"` | ✅ | |
| Boolean | `true` / `false` | ✅ | |
| Null | `null` | ✅ | |
| Array | `[1, 2, 3]` | ✅ | |
| Dictionary | `{a: 1, b: 2}` | ✅ | |
| **Money** | `USD#19.99` | ✅ | New in FEAT-116 |
| **Date** | `@2024-01-15` | ✅ | Fixed `__type` detection |
| **DateTime** | `@2024-01-15T10:30:00` | ✅ | Fixed `__type` detection |
| **Path** | `@./config/file.json` | ✅ | Fixed serializer + parser |
| **URL** | `@https://example.com/api` | ✅ | Fixed serializer + parser |
| Record | `@Person({name: "Alice"})` | ✅ | Already worked |
| Table | `[{...}, {...}]` | ✅ | Serializes as array of dicts |

### Types That Cannot Be Serialized (by design)

- `Function` / `Builtin` - Code cannot be serialized
- `DBConnection` / `SFTPConnection` - Runtime resources
- `Schema` - Type definitions belong in code, not data files

### Schema Handling

Schemas are **not** serialized into PLN files. Records include the schema name:
```pln
@Person({name: "Alice", age: 30})
```

When deserializing, PLN looks up the schema by name in the current environment. If not found, it creates a minimal stub schema to preserve the data.

## Implementation Notes

**Completed 2026-02-12**

### Files Changed

1. `pkg/parsley/pln/lexer.go`:
   - Added `MONEY` token type
   - Added `readMoneyLiteral()` with support for `$`, `£`, `€`, `¥`, and `CODE#` formats
   - Added `CurrencyScales` map for ISO 4217 decimal places
   - Added helper functions for parsing money amounts

2. `pkg/parsley/pln/parser.go`:
   - Added `case MONEY` in `parseValue()` switch
   - Added `parseMoney()` that returns `*evaluator.Money`

3. `pkg/parsley/pln/serializer.go`:
   - Updated `serializeMoney()` to output `CODE#amount` format instead of dict

4. `pkg/parsley/pln/serializer_test.go`:
   - Updated test expectations for literal format
   - Added `TestMoneyRoundTrip` verifying `*evaluator.Money` is returned

### Manual Verification

```parsley
let price = money(19.99, "USD")
price ==> PLN(@/tmp/money_test.pln)
// File contains: USD#19.99

let read_back <== PLN(@/tmp/money_test.pln)
// Returns: $19.99 (native Money object)

read_back + read_back  // $39.98 - arithmetic works!
```

Nested structure test:
```parsley
let cart = {
    items: [{name: "Widget", price: money(19.99, "USD")}],
    tax: money(5.60, "USD")
}
cart ==> PLN(@./cart.pln)
// File: {items: [{name: "Widget", price: USD#19.99}], tax: USD#5.60}

let loaded <== PLN(@./cart.pln)
loaded.items[0].price + loaded.tax  // $25.59 - Money operations work
```

All PLN tests pass (`go test ./pkg/parsley/pln/...`).