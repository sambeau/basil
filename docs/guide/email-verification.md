# Email Verification Guide

Email verification adds an optional layer of account security to Basil's passkey authentication. When enabled, users must verify their email addresses before accessing protected routes (configurable).

## Overview

**Features:**
- Optional email verification during signup
- Configurable enforcement (block unverified users or just notify)
- Single-provider configuration (Mailgun or Resend)
- Built-in rate limiting and abuse protection
- Email-based account recovery (when verified)
- CLI commands for manual administration

**Not included:**
- Multi-provider failover (use one provider at a time)
- Notification API for developers (deferred to future feature)

## Quick Start

### 1. Choose a Provider

Basil supports two email providers:

- **Mailgun**: Established provider, good deliverability, pay-as-you-go
- **Resend**: Developer-friendly, modern API, competitive pricing

### 2. Configure basil.yaml

```yaml
auth:
  enabled: true
  
  email_verification:
    enabled: true
    provider: mailgun  # or "resend"
    
    # Mailgun configuration
    mailgun:
      api_key: "${MAILGUN_API_KEY}"
      domain: "mg.example.com"
      region: us  # or "eu"
      from: "noreply@example.com"
    
    # Resend configuration (use instead of mailgun)
    resend:
      api_key: "${RESEND_API_KEY}"
      from: "noreply@example.com"
    
    # Verification behavior
    require_verification: true  # Block unverified users
    token_ttl: 1h              # Link expires after 1 hour
    resend_cooldown: 5m        # Minimum between resends
    max_sends_per_day: 10      # Per-user daily limit
    
    # Email templates
    template_vars:
      site_name: "My App"
      site_url: "https://example.com"
```

### 3. Set Environment Variables

**Mailgun:**
```bash
export MAILGUN_API_KEY="key-your-mailgun-api-key"
```

**Resend:**
```bash
export RESEND_API_KEY="re_your-resend-api-key"
```

### 4. Start Basil

```bash
basil
```

Users will receive verification emails automatically after registering.

## Configuration Options

### Email Verification Settings

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable email verification |
| `provider` | string | — | Email provider: `mailgun` or `resend` |
| `require_verification` | bool | `false` | Block unverified users from protected routes |
| `token_ttl` | duration | `1h` | Verification link lifetime |
| `resend_cooldown` | duration | `5m` | Minimum time between resend requests |
| `max_sends_per_day` | int | `10` | Per-user daily verification email limit |

### Mailgun Settings

| Option | Required | Description |
|--------|----------|-------------|
| `api_key` | Yes | Mailgun API key |
| `domain` | Yes | Verified sending domain |
| `region` | No | `us` (default) or `eu` |
| `from` | Yes | Sender email address |

### Resend Settings

| Option | Required | Description |
|--------|----------|-------------|
| `api_key` | Yes | Resend API key |
| `from` | Yes | Sender email address |

### Template Variables

| Option | Default | Description |
|--------|---------|-------------|
| `site_name` | `"Basil"` | Your application name |
| `site_url` | Server URL | Link to your homepage |

## Provider Setup

See detailed setup guides:
- [Mailgun Setup Guide](./email-providers/mailgun.md)
- [Resend Setup Guide](./email-providers/resend.md)

## User Flow

### Signup with Email Verification

1. User registers with passkey and email
2. Basil sends verification email automatically
3. User clicks link in email
4. Email is verified, user can access protected routes

### Without Verification Required

If `require_verification: false`:
- Users can access routes immediately
- Verification emails still sent
- `basil.auth.user.email_verification_pending` flag available in templates

### With Verification Required

If `require_verification: true`:
- Unverified users redirected to `/__auth/verify-email-required`
- Access granted after verification
- CLI can manually verify users

## Parsley Context

Email verification status is exposed in Parsley handlers:

```parsley
<html>
<body>
  {if basil.auth.user}
    {if basil.auth.user.email_verified_at}
      <p>Email verified: {basil.auth.user.email_verified_at}</p>
    {else if basil.auth.user.email_verification_pending}
      <p>Please verify your email address.</p>
    {/if}
  {/if}
</body>
</html>
```

**Available fields:**
- `basil.auth.user.email_verified_at` - Timestamp or `nil`
- `basil.auth.user.email_verification_pending` - Boolean

## CLI Commands

### Verify Email Manually

```bash
basil auth verify-email <user_id>
```

Marks a user's email as verified without requiring them to click the link.

### Show Verification Status

```bash
basil auth status <user_id>
```

Shows user details including verification status and pending tokens.

### Resend Verification Email

```bash
basil auth resend-verification <user_id>
```

Sends a new verification email. Use `--force` to bypass rate limits.

### Reset Verification State

```bash
basil auth reset-verification <user_id>
```

Clears email verification (sets email_verified_at to NULL). Useful for testing or fixing issues.

### View Email Logs

```bash
basil auth email-logs [--user <id>] [--limit N]
```

Shows email audit logs. Filter by user ID or limit results.

## Rate Limiting

Built-in protection against abuse:

| Limit | Default | Description |
|-------|---------|-------------|
| Cooldown | 5 minutes | Minimum between sends to same user |
| Per-user daily | 10 emails | Maximum verifications per user per day |
| Per-email daily | 20 emails | Maximum to same email (spam protection) |
| Developer hourly | 50 emails | Site-wide notification API limit (future) |
| Developer daily | 200 emails | Site-wide notification API limit (future) |

Rate limits are enforced at the database level and cannot be bypassed by users.

## Security

### Token Security

- **Entropy:** 32 bytes (256 bits) cryptographically secure random
- **Storage:** Bcrypt hashed (work factor 12)
- **Lifetime:** 1 hour default (configurable)
- **Single-use:** Marked consumed after first verification
- **Transport:** HTTPS required for verification links

### Email Validation

- RFC 5322 format validation
- No email enumeration (same response for invalid emails)
- Audit logging for all attempts

### Warnings

Development mode shows warnings for:
- Sandbox domains (Mailgun sandbox, Resend onboarding@)
- Missing HTTPS configuration
- Incomplete provider configuration
- Unknown provider specified

## Account Recovery

When email is verified, users can recover accounts via email:

1. User visits `/__auth/recovery`
2. Enters email address
3. Receives recovery link
4. Adds new passkey to regain access

**Requirements:**
- Email must be verified
- Recovery only available for verified emails
- Recovery codes can be disabled when email recovery is enabled

## Testing

### Development Testing

Use the provided test credentials:

**Mailgun:**
```yaml
mailgun:
  api_key: "8d62a15e184a96921189f2976fae04cd-f6d80573-cb1a0b2a"
  domain: "mg.tickly.org"
  from: "test@mg.tickly.org"
```

**Resend:**
```yaml
resend:
  api_key: "re_CTmuvKDZ_Hs8QFRGe2hnRQWmPt3uq9nxJ"
  from: "onboarding@resend.dev"
```

**Test recipient:** `sambeau@mac.com`

### Check Warnings

```bash
basil --check-config
```

Shows configuration warnings including email verification issues.

## Troubleshooting

### Emails Not Sending

1. Check provider API keys are correct
2. Verify domain is configured with provider
3. Check logs for errors: `basil auth email-logs`
4. Test with development credentials

### Users Not Receiving Emails

1. Check spam folder
2. Verify domain SPF/DKIM records
3. Check provider dashboard for delivery status
4. Use test credentials to isolate issue

### Rate Limit Issues

1. Check cooldown period hasn't passed
2. View status: `basil auth status <user_id>`
3. Use `--force` flag to bypass for testing
4. Adjust limits in config if needed

### Token Expired

Tokens expire after 1 hour by default. User must request new verification email.

## Migration Guide

### Existing Installations

1. Update `basil.yaml` with email verification config
2. Choose and configure a provider
3. Restart Basil
4. Existing users: email is unverified
5. Optionally manually verify: `basil auth verify-email <user_id>`

### Disabling Email Verification

Set `enabled: false` in config. No migration needed - verification status is preserved.

## See Also

- [Authentication Guide](./authentication.md)
- [Passkey Guide](./passkeys.md)
- [Configuration Reference](../parsley/reference.md)
- [Mailgun Setup](./email-providers/mailgun.md)
- [Resend Setup](./email-providers/resend.md)
