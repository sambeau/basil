---
id: man-pars-std-hash
title: "@std/hash"
system: parsley
type: stdlib
name: hash
created: 2026-02-26
version: 1.0.0
author: Basil Team
keywords:
  - hash
  - md5
  - sha1
  - sha256
  - sha512
  - checksum
  - digest
  - hex
---

# @std/hash

Cryptographic hash functions for checksums, cache keys, ETags, and non-security hashing. All functions take a string and return a lowercase hex-encoded hash string.

```parsley
let hash = import @std/hash
```

> **⚠️ Not for passwords.** These are raw hash functions with no salting or key stretching. For password hashing, use Basil's auth system.

## Functions

| Function | Arguments | Returns | Description |
|---|---|---|---|
| `md5(s)` | `s: string` | 32-char hex string | MD5 hash |
| `sha1(s)` | `s: string` | 40-char hex string | SHA1 hash |
| `sha256(s)` | `s: string` | 64-char hex string | SHA256 hash |
| `sha512(s)` | `s: string` | 128-char hex string | SHA512 hash |

```parsley
hash.md5("hello")      // "5d41402abc4b2a76b9719d911017c592"
hash.sha1("hello")     // "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
hash.sha256("hello")   // "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
hash.sha512("hello")   // "9b71d224bd62f3785d96d46ad3ea3d73..."
```

Empty strings produce the hash of zero bytes:

```parsley
hash.md5("")           // "d41d8cd98f00b204e9800998ecf8427e"
hash.sha256("")        // "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
```

Unicode strings are hashed as their UTF-8 byte representation:

```parsley
hash.md5("日本語")     // "00110af8b4393ef3f72c50be5b332bec"
```

## Common Use Cases

### ETags for Caching

```parsley
let hash = import @std/hash

let content = loadPageContent(slug)
let etag = hash.md5(content)
```

### Cache Keys

```parsley
let hash = import @std/hash

let cacheKey = hash.sha256(userId + ":" + query)
```

### Content Deduplication

```parsley
let hash = import @std/hash

let digest = hash.sha256(fileContent)
if (digest in knownHashes) {
    "Duplicate detected"
}
```

### Checksums for Data Integrity

```parsley
let hash = import @std/hash

let checksum = hash.sha256(payload)
// Send checksum alongside payload for verification
```

## Destructuring Import

```parsley
let { md5, sha256 } = import @std/hash

md5("test")            // "098f6bcd4621d373cade4e832627b4f6"
sha256("test")         // "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
```

## Errors

All functions require exactly one string argument:

```parsley
hash.md5()             // Error: wrong number of arguments
hash.md5(123)          // Error: expected a string
hash.md5("a", "b")    // Error: wrong number of arguments
```

## Choosing a Hash Function

| Function | Speed | Output Size | Use For |
|---|---|---|---|
| `md5` | Fastest | 128-bit (32 hex) | ETags, cache keys, non-critical checksums |
| `sha1` | Fast | 160-bit (40 hex) | Git-style content addressing |
| `sha256` | Moderate | 256-bit (64 hex) | Data integrity, deduplication |
| `sha512` | Moderate | 512-bit (128 hex) | When you need a longer digest |

> MD5 and SHA1 are cryptographically broken for collision resistance but remain fine for checksums and cache keys where adversarial collisions are not a concern.

## See Also

- [Strings](../builtins/strings.md) — `.toBase64()` and `.fromBase64()` for encoding
- [@std/valid](valid.md) — validation predicates
- [@std/id](id.md) — ID generation