# Form Autocomplete Metadata Design

**Status:** Proposal  
**Date:** 2026-01-19  
**Author:** AI Assistant with human review

## Overview

This document proposes adding autocomplete support to schema field metadata for form input binding. The HTML `autocomplete` attribute enables browser autofill, significantly improving user experience for forms.

## Goals

1. **Zero configuration for common cases** — email, phone, name fields "just work"
2. **Explicit override when needed** — billing vs shipping addresses, disabling
3. **Simple mental model** — "it guesses, you can override"
4. **Safe fallback** — no autocomplete attribute if unknown (browser decides)

## Non-Goals

1. Supporting every obscure autocomplete token
2. Structured autocomplete values (use strings)
3. JavaScript-based autofill (browser-native only)

---

## Design

### Layered Auto-Derivation

Autocomplete values are derived in priority order:

1. **Explicit metadata** — Always wins
2. **Field name pattern** — Common naming conventions  
3. **Schema type** — Type-based defaults
4. **Default** — No attribute (browser decides)

### Type-Based Defaults

These types always get an autocomplete attribute unless overridden:

| Schema Type | Autocomplete Value |
|-------------|-------------------|
| `email` | `"email"` |
| `phone` | `"tel"` |
| `url` | `"url"` |

### Field Name Patterns

Field names are matched case-insensitively with common variations:

| Pattern | Autocomplete Value |
|---------|-------------------|
| `name`, `fullName`, `fullname` | `"name"` |
| `firstName`, `firstname`, `givenName`, `givenname` | `"given-name"` |
| `lastName`, `lastname`, `familyName`, `familyname`, `surname` | `"family-name"` |
| `username`, `userName` | `"username"` |
| `password`, `passwd` | `"current-password"` |
| `newPassword`, `newpassword` | `"new-password"` |
| `confirmPassword`, `confirmpassword`, `passwordConfirm` | `"new-password"` |
| `street`, `streetAddress`, `address`, `addressLine1`, `address1` | `"street-address"` |
| `addressLine2`, `address2`, `apt`, `suite`, `unit` | `"address-line2"` |
| `city`, `town` | `"address-level2"` |
| `state`, `province`, `region`, `county` | `"address-level1"` |
| `zip`, `zipCode`, `zipcode`, `postalCode`, `postalcode`, `postCode`, `postcode` | `"postal-code"` |
| `country`, `countryName` | `"country-name"` |
| `countryCode` | `"country"` |
| `organization`, `company`, `org`, `companyName` | `"organization"` |
| `jobTitle`, `title`, `position`, `role` | `"organization-title"` |
| `creditCard`, `cardNumber`, `ccNumber` | `"cc-number"` |
| `cardName`, `ccName`, `nameOnCard` | `"cc-name"` |
| `cardExpiry`, `ccExpiry`, `expiry`, `expirationDate` | `"cc-exp"` |
| `cardCvc`, `cvc`, `cvv`, `securityCode` | `"cc-csc"` |
| `birthday`, `birthdate`, `dob`, `dateOfBirth` | `"bday"` |
| `language`, `preferredLanguage` | `"language"` |

### Explicit Metadata Override

Use the `autocomplete` key in field metadata:

```parsley
@schema Checkout {
    // Auto-derived from type: autocomplete="email"
    email: email
    
    // Auto-derived from field name: autocomplete="given-name"
    firstName: string
    
    // Explicit override for shipping context
    shippingStreet: string | {autocomplete: "shipping street-address"}
    shippingCity: string | {autocomplete: "shipping address-level2"}
    
    // Explicit override for billing context  
    billingStreet: string | {autocomplete: "billing street-address"}
    billingCity: string | {autocomplete: "billing address-level2"}
    
    // Disable autocomplete entirely
    captcha: string | {autocomplete: "off"}
    
    // One-time code (SMS/email verification)
    verificationCode: string | {autocomplete: "one-time-code"}
}
```

### Priority Examples

```parsley
@schema User {
    // 1. Type wins: email type → autocomplete="email"
    email: email
    
    // 2. Field name: "firstName" → autocomplete="given-name"
    firstName: string
    
    // 3. Explicit overrides all: autocomplete="username"
    firstName: string | {autocomplete: "username"}
    
    // 4. No match: no autocomplete attribute
    favoriteColor: string
    
    // 5. Explicit disable: autocomplete="off"
    temporaryCode: string | {autocomplete: "off"}
}
```

---

## HTML Output

### Before (current)

```parsley
<form @record={user}>
    <input @field="email"/>
    <input @field="firstName"/>
</form>
```

Generates:
```html
<form>
    <input name="email" type="email" value="..." />
    <input name="firstName" type="text" value="..." />
</form>
```

### After (proposed)

Same Parsley code generates:
```html
<form>
    <input name="email" type="email" value="..." autocomplete="email" />
    <input name="firstName" type="text" value="..." autocomplete="given-name" />
</form>
```

---

## Special Cases

### Password Fields

Password fields auto-derive based on common naming:

```parsley
@schema Login {
    password: string    // → autocomplete="current-password"
}

@schema Registration {
    password: string           // → autocomplete="current-password" (ambiguous)
    newPassword: string        // → autocomplete="new-password"
    confirmPassword: string    // → autocomplete="new-password"
}
```

**Recommendation:** For registration forms, use explicit field names like `newPassword` or add explicit metadata:

```parsley
@schema Registration {
    password: string | {autocomplete: "new-password"}
}
```

### Section Prefixes

HTML autocomplete supports section prefixes for multiple address blocks:

```parsley
@schema Order {
    // Shipping address
    shippingName: string | {autocomplete: "shipping name"}
    shippingStreet: string | {autocomplete: "shipping street-address"}
    shippingCity: string | {autocomplete: "shipping address-level2"}
    
    // Billing address  
    billingName: string | {autocomplete: "billing name"}
    billingStreet: string | {autocomplete: "billing street-address"}
    billingCity: string | {autocomplete: "billing address-level2"}
}
```

The autocomplete value is passed through as-is, supporting all valid HTML autocomplete tokens including compound values.

### Disabling Autocomplete

Two ways to disable:

```parsley
// Disable for sensitive/temporary data
verificationCode: string | {autocomplete: "off"}

// Also valid (explicit string)
searchQuery: string | {autocomplete: "off"}
```

### Credit Card Fields

Credit card fields require explicit metadata for security reasons (no auto-derivation from type):

```parsley
@schema Payment {
    // These will NOT auto-derive (too sensitive)
    cardNumber: string        // No autocomplete (safe default)
    
    // Explicit opt-in required
    cardNumber: string | {autocomplete: "cc-number"}
    cardExpiry: string | {autocomplete: "cc-exp"}
    cardCvc: string | {autocomplete: "cc-csc"}
}
```

**Wait, this contradicts the table above.** Let me reconsider...

**Decision:** Credit card fields WILL auto-derive from field names like `cardNumber`, `creditCard`, etc. The browser's payment autofill is a feature, not a bug. Users can disable with `autocomplete: "off"` if needed.

---

## Implementation

### Data Flow

1. `evalFieldBinding()` processes `<input @field="..."/>`
2. Looks up field in schema
3. Calls `getAutocomplete(fieldName, fieldType, fieldMetadata)`
4. Adds `autocomplete="..."` attribute if value returned

### Pseudocode

```go
func getAutocomplete(fieldName string, fieldType string, metadata map[string]Object) string {
    // 1. Check explicit metadata
    if ac, ok := metadata["autocomplete"]; ok {
        return ac.String()
    }
    
    // 2. Check type-based defaults
    switch fieldType {
    case "email":
        return "email"
    case "phone":
        return "tel"
    case "url":
        return "url"
    }
    
    // 3. Check field name patterns
    nameLower := strings.ToLower(fieldName)
    if ac, ok := autocompletePatterns[nameLower]; ok {
        return ac
    }
    
    // 4. No match - return empty (no attribute)
    return ""
}

var autocompletePatterns = map[string]string{
    "name":            "name",
    "fullname":        "name",
    "firstname":       "given-name",
    "givenname":       "given-name",
    "lastname":        "family-name",
    "familyname":      "family-name",
    "surname":         "family-name",
    "username":        "username",
    "password":        "current-password",
    "newpassword":     "new-password",
    "confirmpassword": "new-password",
    // ... etc
}
```

### Files to Modify

1. **`pkg/parsley/evaluator/form_binding.go`** — Add `getAutocomplete()` function
2. **`pkg/parsley/evaluator/form_binding.go`** — Update `evalFieldBinding()` to include autocomplete
3. **`docs/parsley/manual/builtins/record.md`** — Document autocomplete metadata
4. **`docs/parsley/reference.md`** — Update form binding section

---

## Complete Autocomplete Token Reference

For reference, here are all standard HTML autocomplete tokens:

### Name Tokens
- `name` — Full name
- `given-name` — First name
- `additional-name` — Middle name
- `family-name` — Last name
- `nickname` — Nickname
- `honorific-prefix` — Title (Mr., Dr., etc.)
- `honorific-suffix` — Suffix (Jr., PhD, etc.)

### Contact Tokens
- `email` — Email address
- `tel` — Full phone number
- `tel-country-code` — Country code
- `tel-national` — National phone number
- `tel-area-code` — Area code
- `tel-local` — Local phone number
- `tel-extension` — Extension
- `url` — URL/website

### Address Tokens
- `street-address` — Full street address (multiline)
- `address-line1` — Address line 1
- `address-line2` — Address line 2
- `address-line3` — Address line 3
- `address-level1` — State/province/region
- `address-level2` — City/town
- `address-level3` — District/suburb
- `address-level4` — Neighborhood
- `postal-code` — ZIP/postal code
- `country` — Country code (ISO 3166-1 alpha-2)
- `country-name` — Country name

### Account Tokens
- `username` — Username
- `current-password` — Current password (login)
- `new-password` — New password (registration/change)
- `one-time-code` — OTP/verification code

### Payment Tokens
- `cc-name` — Name on card
- `cc-given-name` — First name on card
- `cc-additional-name` — Middle name on card
- `cc-family-name` — Last name on card
- `cc-number` — Card number
- `cc-exp` — Expiration date (MM/YY)
- `cc-exp-month` — Expiration month
- `cc-exp-year` — Expiration year
- `cc-csc` — Security code (CVC/CVV)
- `cc-type` — Card type (Visa, etc.)

### Other Tokens
- `bday` — Full birthday
- `bday-day` — Birthday day
- `bday-month` — Birthday month
- `bday-year` — Birthday year
- `sex` — Gender
- `language` — Preferred language
- `organization` — Company/organization
- `organization-title` — Job title
- `photo` — Photo URL
- `impp` — Instant messaging URL

### Section Prefixes
- `shipping` — Shipping address context
- `billing` — Billing address context
- `section-*` — Custom section (e.g., `section-blue`)

---

## Examples

### Login Form

```parsley
@schema Login {
    email: email           // → autocomplete="email"
    password: string       // → autocomplete="current-password"
}

<form @record={login} method="POST">
    <label @field="email"/>
    <input @field="email"/>
    
    <label @field="password"/>
    <input @field="password" type="password"/>
    
    <button type="submit">"Sign In"</button>
</form>
```

### Registration Form

```parsley
@schema Registration {
    firstName: string                                    // → autocomplete="given-name"
    lastName: string                                     // → autocomplete="family-name"
    email: email                                         // → autocomplete="email"
    newPassword: string                                  // → autocomplete="new-password"
    confirmPassword: string                              // → autocomplete="new-password"
}
```

### Checkout Form

```parsley
@schema Checkout {
    // Contact
    email: email                                         // → autocomplete="email"
    phone: phone                                         // → autocomplete="tel"
    
    // Shipping (explicit section)
    shippingName: string | {autocomplete: "shipping name"}
    shippingStreet: string | {autocomplete: "shipping street-address"}
    shippingCity: string | {autocomplete: "shipping address-level2"}
    shippingState: string | {autocomplete: "shipping address-level1"}
    shippingZip: string | {autocomplete: "shipping postal-code"}
    shippingCountry: string | {autocomplete: "shipping country-name"}
    
    // Billing (explicit section)
    billingName: string | {autocomplete: "billing name"}
    billingStreet: string | {autocomplete: "billing street-address"}
    billingCity: string | {autocomplete: "billing address-level2"}
    billingState: string | {autocomplete: "billing address-level1"}
    billingZip: string | {autocomplete: "billing postal-code"}
    billingCountry: string | {autocomplete: "billing country-name"}
    
    // Payment
    cardNumber: string | {autocomplete: "cc-number"}
    cardName: string | {autocomplete: "cc-name"}
    cardExpiry: string | {autocomplete: "cc-exp"}
    cardCvc: string | {autocomplete: "cc-csc"}
}
```

### Profile Form (No Autocomplete)

```parsley
@schema Profile {
    // These don't need autocomplete (not common autofill targets)
    bio: text
    website: url           // → autocomplete="url" (type-based)
    favoriteColor: string  // No autocomplete (no match)
    
    // Explicitly disable for search
    searchQuery: string | {autocomplete: "off"}
}
```

---

## Testing

### Unit Tests

1. Type-based derivation (`email` → `"email"`)
2. Field name pattern matching (`firstName` → `"given-name"`)
3. Case-insensitive matching (`FIRSTNAME` → `"given-name"`)
4. Explicit override wins over auto-derivation
5. Explicit `"off"` produces `autocomplete="off"`
6. Unknown fields produce no autocomplete attribute
7. Compound values pass through (`"shipping street-address"`)

### Integration Tests

1. Full form rendering with multiple field types
2. Checkout form with shipping/billing sections
3. Password fields with correct tokens

---

## Open Questions

1. **Should we support `autocomplete` on `<select @field>`?**
   - HTML supports it for country dropdowns, etc.
   - Recommendation: Yes, same logic applies

2. **Should textarea support autocomplete?**
   - Less common but valid for `street-address`
   - Recommendation: Yes, for consistency

3. **What about `autofocus`?**
   - Different feature but related UX
   - Recommendation: Separate proposal if needed

---

## References

- [MDN: HTML autocomplete attribute](https://developer.mozilla.org/en-US/docs/Web/HTML/Attributes/autocomplete)
- [WHATWG: Autofill](https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#autofill)
- [Chrome Autofill](https://developer.chrome.com/docs/extensions/reference/autofillPrivate/)
