---
id: man-pars-stdlib-id
title: "@std/id"
system: parsley
type: stdlib
name: id
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - id
  - uuid
  - ulid
  - nanoid
  - cuid
  - identifier
  - unique
  - generation
---

# @std/id

ID generation functions for creating unique identifiers. All functions return strings and are thread-safe.

```parsley
let id = import @std/id
```

## Functions

| Function | Returns | Description |
|---|---|---|
| `new()` | string | ULID-like ID (26 chars, time-sortable, Crockford Base32) |
| `uuid()` | string | UUID v4 (random, 36 chars with dashes) |
| `uuidv4()` | string | Alias for `uuid()` |
| `uuidv7()` | string | UUID v7 (time-sortable, 36 chars with dashes) |
| `nanoid(length?)` | string | NanoID (URL-safe, compact; default 21 chars) |
| `cuid()` | string | CUID2-like (collision-resistant, 25 chars) |

## Examples

```parsley
let id = import @std/id

id.new()                         // "01KEQAT4553AQS0P93DXYZ"
id.uuid()                        // "550e8400-e29b-41d4-a716-446655440000"
id.uuidv7()                      // "019baead-10a5-734c-8d7e-446655440000"
id.nanoid()                      // "V1StGXR8_Z5jdHi6B-myT"
id.nanoid(10)                    // "IRFa-VaY2b"
id.cuid()                        // "c01keqat4553aqs0p93dxy"
```

## Which ID to Use

| Use Case | Function | Why |
|---|---|---|
| Database primary keys | `new()` or `uuidv7()` | Time-sortable, so B-tree indexes stay efficient |
| Interoperability with external systems | `uuid()` | Standard UUID v4 format recognized everywhere |
| Short URLs, invite codes | `nanoid(10)` | Compact, URL-safe, configurable length |
| Distributed systems, horizontal scaling | `cuid()` | Designed to avoid collisions across multiple machines |

## Format Details

### `new()` — ULID

26-character string using Crockford's Base32 alphabet (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`). The first 10 characters encode the timestamp in milliseconds; the remaining 16 are random. IDs generated in the same millisecond sort randomly within that millisecond.

### `uuid()` / `uuidv4()` — UUID v4

Standard 36-character UUID with dashes: `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx`. All bits except the version (4) and variant markers are random.

### `uuidv7()` — UUID v7

Standard 36-character UUID with dashes. The first 48 bits encode the Unix timestamp in milliseconds, making IDs monotonically increasing over time. Version marker is 7.

### `nanoid(length?)` — NanoID

URL-safe string from a 64-character alphabet (`0-9A-Z_a-z-`). Default length is 21 characters. Pass an integer (1–256) to control length:

```parsley
id.nanoid()                      // 21 chars (default)
id.nanoid(8)                     // 8 chars (shorter, higher collision risk)
id.nanoid(36)                    // 36 chars (very low collision risk)
```

### `cuid()` — CUID2

25-character string prefixed with `c`. Combines a timestamp, a monotonic counter, and random bytes. The counter ensures uniqueness even when multiple IDs are generated in the same millisecond on the same machine.

## See Also

- [Data Model](../fundamentals/data-model.md) — schemas with `id` type fields
- [Database](../features/database.md) — auto-generated IDs in table bindings