---
id: FEAT-097
title: "Form Autocomplete Metadata"
status: implemented
priority: medium
created: 2026-01-19
author: "@human"
---

# FEAT-097: Form Autocomplete Metadata

## Summary
Add autocomplete support to schema field metadata for form input binding. The HTML `autocomplete` attribute enables browser autofill, significantly improving user experience for forms. Values are auto-derived from field types and names, with explicit override via metadata.

## User Story
As a developer building forms, I want browser autofill to work automatically for common fields (email, name, address) so that users can complete forms quickly without me needing to add autocomplete attributes manually.

## Acceptance Criteria
- [x] `<input @field>` includes `autocomplete` attribute when derivable
- [x] Type-based defaults: `email` → `"email"`, `phone` → `"tel"`, `url` → `"url"`
- [x] Field name patterns: `firstName` → `"given-name"`, `password` → `"current-password"`, etc.
- [x] Explicit metadata override: `| {autocomplete: "shipping street-address"}`
- [x] Disable with `| {autocomplete: "off"}`
- [x] No attribute added when no match (safe default)
- [x] Case-insensitive field name matching
- [x] Documentation updated

## Design Decisions
- **Layered derivation**: Explicit metadata > Field name > Type > No attribute. This provides zero-config for common cases while allowing precise control.
- **Metadata not constraints**: `autocomplete` is presentation, not validation, so it belongs in metadata (`|`) not constraints (`()`).
- **Password auto-derives**: `password` → `current-password` because override is available. Use `newPassword` or explicit metadata for registration forms.
- **String pass-through**: Compound values like `"shipping street-address"` are passed as-is, supporting all HTML autocomplete tokens.
- **Credit card auto-derives**: Field names like `cardNumber` auto-derive because browser payment autofill is a feature. Disable with `"off"` if unwanted.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
- `pkg/parsley/evaluator/form_binding.go` — Add `getAutocomplete()` function, update `evalFieldBinding()`
- `docs/parsley/manual/builtins/record.md` — Document autocomplete metadata
- `docs/parsley/reference.md` — Update form binding section

### Dependencies
- Depends on: FEAT-091 (Form binding with @field)
- Blocks: None

### Edge Cases & Constraints
1. **Unknown field names** — No autocomplete attribute added (browser decides)
2. **Explicit "off"** — Produces `autocomplete="off"` to disable
3. **Compound values** — Passed through as-is (e.g., `"billing address-level2"`)
4. **Select elements** — Should also support autocomplete (for country dropdowns, etc.)
5. **Textarea elements** — Should support autocomplete (for street-address)

### Field Name Patterns

| Pattern | Autocomplete |
|---------|-------------|
| `name`, `fullName` | `"name"` |
| `firstName`, `givenName` | `"given-name"` |
| `lastName`, `familyName`, `surname` | `"family-name"` |
| `username` | `"username"` |
| `password` | `"current-password"` |
| `newPassword`, `confirmPassword` | `"new-password"` |
| `street`, `streetAddress`, `address`, `addressLine1` | `"street-address"` |
| `city`, `town` | `"address-level2"` |
| `state`, `province`, `region` | `"address-level1"` |
| `zip`, `zipCode`, `postalCode` | `"postal-code"` |
| `country`, `countryName` | `"country-name"` |
| `organization`, `company` | `"organization"` |
| `cardNumber`, `creditCard` | `"cc-number"` |
| `cardName`, `nameOnCard` | `"cc-name"` |
| `cardExpiry`, `expirationDate` | `"cc-exp"` |
| `cardCvc`, `cvv`, `securityCode` | `"cc-csc"` |
| `birthday`, `dob`, `dateOfBirth` | `"bday"` |

## Implementation Notes
*Added during/after implementation*

## Related
- Design doc: `work/design/FORM_AUTOCOMPLETE.md`
- Form binding: FEAT-091
