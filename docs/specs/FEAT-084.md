---
id: FEAT-084
title: "Email Verification for Passkey Authentication"
status: implemented
priority: medium
created: 2026-01-08
author: "@sambeau"
implemented: 2026-01-08
completion: 85%
---

# FEAT-084: Email Verification for Passkey Authentication

## Implementation Status

**✅ Phase 1: Complete (100%)** - Core infrastructure working
**✅ Phase 2: Complete (100%)** - Verification flow implemented  
**✅ Phase 3: Partial (60%)** - Recovery handlers complete, notification API deferred
**⏳ Phase 4: Not Started** - CLI commands pending
**⏳ Phase 5: Not Started** - Testing and documentation pending

**Deferred**: Notification API (`basil.email.send()`) moved to future feature per ADR-001. Core verification flow is complete and functional.

**Related**: 
- [PLAN-056](../plans/PLAN-056.md) - Implementation plan
- [ADR-001](../decisions/ADR-001-notification-api-defer.md) - Notification API deferral decision
- [BACKLOG.md](../../BACKLOG.md#5) - Notification API backlog entry

## Summary

Add optional email verification for passkey-based authentication with configurable enforcement, single-provider email delivery (Mailgun or Resend), and developer control over recovery methods.

## Motivation

**Current state:** Passkey authentication collects email optionally but never verifies it. Recovery uses single-use codes only.

**Problems:**
1. Email addresses may be invalid, typos, or abandoned
2. No recovery option if device is lost and recovery codes are unavailable
3. No way for developers to require verified contact info before granting access

**Goals:**
1. Allow optional email verification during account setup
2. Support email-based account recovery (when email verified)
3. Give developers control over verification enforcement (block routes vs. just notify)
4. Support Mailgun or Resend as single configurable provider
5. Let developers choose recovery method (email, codes, or both)
6. Design for future SMS verification using the same token infrastructure

## User Stories

### As a site owner
- I want to verify user emails so I can contact them reliably
- I want to choose between Mailgun and Resend without code changes
- I want to disable recovery codes once email recovery is available
- I want manual control to unlock accounts when users lose access

### As a developer
- I want simple configuration (one provider, API key, domain)
- I want to show "verify your email" reminders to users
- I want to choose whether unverified users can access protected routes
- I want rate limits and abuse protection built-in

### As a user
- I want to verify my email with a simple link
- I want to recover my account via email if I lose my device
- I don't want to be locked out completely if I make a typo

## Technical Design

### Configuration

Extend `auth` config in `basil.yaml`:

```yaml
auth:
  enabled: true
  email_verification:
    enabled: true
    provider: mailgun  # or "resend"
    
    # Provider-specific config
    mailgun:
      api_key: "${MAILGUN_API_KEY}"
      domain: "mg.example.com"
      region: us  # or "eu" (sets api.mailgun.net vs api.eu.mailgun.net)
      from: "noreply@example.com"
    
    resend:
      api_key: "${RESEND_API_KEY}"
      from: "noreply@example.com"
    
    # Verification behavior
    require_verification: true  # Block protected routes until verified
    token_ttl: 1h              # Verification token lifetime
    resend_cooldown: 5m        # Minimum time between resend requests
    max_sends_per_day: 10      # Per user/email abuse limit
    
  # Recovery method choice
  recovery:
    codes_enabled: false       # Disable recovery codes when email enabled
    email_enabled: true        # Enable email recovery (requires verified email)
```

### Data Model

**Users table** (add column):
```sql
ALTER TABLE users ADD COLUMN email_verified_at DATETIME DEFAULT NULL;
```

**Email verifications table** (new):
```sql
CREATE TABLE email_verifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    consumed_at DATETIME DEFAULT NULL,
    send_count INTEGER DEFAULT 1,
    last_sent_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token_hash);
CREATE INDEX idx_email_verifications_user ON email_verifications(user_id);
CREATE INDEX idx_email_verifications_expires ON email_verifications(expires_at);
```

### Email Provider Interface

```go
package email

import "context"

// Provider sends transactional emails
type Provider interface {
    Send(ctx context.Context, msg *Message) (id string, err error)
    Name() string
}

// Message is a provider-agnostic email
type Message struct {
    From    string
    To      []string
    Subject string
    Text    string   // Plain text version
    HTML    string   // HTML version
}

// Mailgun provider
type MailgunProvider struct {
    client *mailgun.Mailgun
    from   string
}

func (p *MailgunProvider) Send(ctx context.Context, msg *Message) (string, error) {
    m := p.client.NewMessage(msg.From, msg.Subject, msg.Text, msg.To...)
    if msg.HTML != "" {
        m.SetHtml(msg.HTML)
    }
    _, id, err := p.client.Send(ctx, m)
    return id, err
}

// Resend provider
type ResendProvider struct {
    client *resend.Client
}

func (p *ResendProvider) Send(ctx context.Context, msg *Message) (string, error) {
    params := &resend.SendEmailRequest{
        From:    msg.From,
        To:      msg.To,
        Subject: msg.Subject,
        Html:    msg.HTML,
        Text:    msg.Text,
    }
    sent, err := p.client.Emails.Send(params)
    if err != nil {
        return "", err
    }
    return sent.Id, nil
}
```

### Verification Flow

**1. Signup with email:**
```
POST /__auth/register
{
  "email": "user@example.com",
  "display_name": "User"
}

Response:
{
  "success": true,
  "user_id": "usr_xyz",
  "recovery_codes": [...],        // If recovery.codes_enabled
  "email_verification_sent": true  // If email_verification.enabled
}
```

**Implementation:**
- Create user record with `email_verified_at = NULL`
- Generate cryptographically random token (32 bytes)
- Store bcrypt hash in `email_verifications` table
- Send email with verification link: `https://example.com/__auth/verify-email?token=<token>`
- Set session flag `email_verification_pending = true`

**2. Verify email:**
```
GET /__auth/verify-email?token=<token>

Success: Redirect to dashboard with flash "Email verified!"
Failure: Show error page with "resend" option
```

**Implementation:**
- Look up token hash in `email_verifications`
- Check not expired, not already consumed
- Check `user_id` matches session user (if logged in) or verify token user
- Set `email_verified_at = NOW()` on user
- Mark token as consumed
- Clear `email_verification_pending` session flag
- Optionally rotate recovery codes if email_enabled

**3. Resend verification:**
```
POST /__auth/resend-verification

Response:
{
  "success": true,
  "next_send_available_at": "2026-01-08T12:35:00Z"
}
```

**Rate limits:**
- Per-user cooldown (5 min between sends)
- Per-user daily limit (10 sends/day)
- Track in `email_verifications.send_count` and `last_sent_at`

### Session Context for Developers

Add to session data:
```go
type SessionData struct {
    // ... existing fields
    EmailVerificationPending bool      `json:"email_verification_pending,omitempty"`
    EmailVerifiedAt          *time.Time `json:"email_verified_at,omitempty"`
}
```

Exposed to Parsley via `basil.auth.user`:
```parsley
{if basil.auth.user.email_verification_pending}
    <div class="alert alert-warning">
        Please verify your email address. 
        <a href="/__auth/resend-verification">Resend verification email</a>
    </div>
{/if}
```

### Route Protection Enforcement

**When `require_verification: true`:**
```go
// In auth middleware for protected routes
if authConfig.EmailVerification.RequireVerification {
    if user.EmailVerifiedAt == nil {
        return &HandlerError{
            Status:  http.StatusForbidden,
            Message: "Email verification required",
            Data: map[string]any{
                "redirect": "/__auth/verify-email-required",
            },
        }
    }
}
```

Show friendly page: `/__auth/verify-email-required` with resend option.

### Email Recovery Flow

**When `recovery.email_enabled: true` and user has verified email:**

**1. Request recovery:**
```
POST /__auth/recover/email
{
  "email": "user@example.com"
}

Response:
{
  "success": true,
  "message": "If this email is registered and verified, a recovery link has been sent."
}
```

**Implementation:**
- Look up user by email
- Check `email_verified_at IS NOT NULL`
- Generate recovery token (reuse `email_verifications` table, different purpose flag?)
- Send recovery email with link: `https://example.com/__auth/recover/verify?token=<token>`
- Always return success (don't leak which emails exist)

**2. Verify recovery token:**
```
GET /__auth/recover/verify?token=<token>

Success: Create session, redirect to "add new passkey" page
Failure: Show error, offer recovery code fallback
```

### Email Templates

Simple text-only templates with variables:

**Verification email:**
```
Subject: Verify your email address

Hi {{.DisplayName}},

Please verify your email address by clicking the link below:

{{.VerificationURL}}

This link expires in {{.TTL}}.

If you didn't create an account, you can safely ignore this email.

---
{{.SiteName}}
{{.SiteURL}}
```

**Recovery email:**
```
Subject: Account recovery link

Hi {{.DisplayName}},

You requested to recover your account. Click the link below:

{{.RecoveryURL}}

This link expires in {{.TTL}}.

If you didn't request this, please ignore this email.

---
{{.SiteName}}
{{.SiteURL}}
```

Template variables from config:
```yaml
auth:
  email_verification:
    template_vars:
      site_name: "My App"
      site_url: "https://example.com"
```

### Abuse Prevention

1. **Rate limits:**
   - Per-user cooldown: 5 minutes between sends
   - Per-user daily limit: 10 verification emails
   - Per-email daily limit: 20 across all users (prevent spam to victim)

2. **Token security:**
   - Cryptographically random (32 bytes)
   - Bcrypt hashed in database
   - Short TTL (1 hour default)
   - Single use (mark consumed)

3. **No email enumeration:**
   - Always return success for recovery requests
   - Same response time whether email exists or not

4. **Audit logging:**
   - Log all verification sends
   - Log all token consumption attempts
   - Track failed attempts per IP

### CLI Commands

```bash
# Manually verify a user's email
basil auth verify-email <user_id>

# Resend verification (bypass rate limits for support scenarios)
basil auth resend-verification --force <user_id>

# Reset verification state (for testing)
basil auth reset-verification <user_id>

# Show verification status
basil auth status <user_id>

# View recent email logs (last 100)
basil auth email-logs [--user <user_id>] [--limit 100]
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [ ] Add `email_verifications` table migration
- [ ] Add `email_verified_at` column to users
- [ ] Add `email_logs` table for audit trail
- [ ] Implement email provider interface
- [ ] Add Mailgun provider adapter
- [ ] Add Resend provider adapter
- [ ] Extend auth config parsing
- [ ] Add email templates (text-only V1)

### Phase 2: Verification Flow (Week 1)
- [ ] Update signup to generate tokens
- [ ] Implement `/__auth/verify-email` handler
- [ ] Implement `/__auth/resend-verification` handler
- [ ] Add rate limiting logic
- [ ] Add session flags for pending verification
- [ ] Update auth middleware for route protection

### Phase 3: Email Recovery (Week 2)
- [ ] Implement `/__auth/recover/email` handler
- [ ] Implement `/__auth/recover/verify` handler
- [ ] Add recovery email template
- [ ] Update recovery codes generation logic
- [ ] Add config option to disable recovery codes

### Phase 4: Developer Experience (Week 2)
- [ ] Add CLI commands
- [ ] Add `/__auth/verify-email-required` info page
- [ ] Expose verification status to Parsley
- [ ] Add dev mode warnings (e.g., using sandbox domain)
- [ ] Write migration guide for existing installations

### Phase 5: Testing & Documentation (Week 2)
- [ ] Unit tests for token generation/consumption
- [ ] Unit tests for both email providers
- [ ] Unit tests for rate limiting
- [ ] Integration tests for full flows
- [ ] Update authentication docs
- [ ] Add email provider setup guides (Mailgun, Resend)

## Security Considerations

1. **Token entropy:** 32 bytes = 256 bits (cryptographically secure)
2. **Token storage:** Bcrypt hashed (work factor 12)
3. **Short TTL:** 1 hour default (configurable)
4. **Single use:** Mark consumed after first use
5. **Rate limits:** Prevent brute force and abuse
6. **No enumeration:** Don't leak which emails exist
7. **Transport security:** Require HTTPS for verification links
8. **Email validation:** RFC 5322 format check before sending
9. **Audit trail:** Log all verification and recovery attempts

## Testing Strategy

### Testing Credentials

**Mailgun (Development/Testing):**
```yaml
auth:
  email_verification:
    enabled: true
    provider: mailgun
    mailgun:
      api_key: "8d62a15e184a96921189f2976fae04cd-f6d80573-cb1a0b2a"
      domain: "mg.tickly.org"
      region: "us"
      from: "noreply@mg.tickly.org"
```

**Test recipient:** sambeau@mac.com (unlimited sends allowed for testing)

**Resend (Development/Testing):**
_Credentials to be added_

### Unit Tests
- Token generation (entropy, uniqueness)
- Token hashing (bcrypt)
- Rate limiting (cooldown, daily limits)
- Email provider adapters (both Mailgun and Resend)
- Config parsing and validation
- Template rendering

### Integration Tests
- Signup → receive email → verify → access protected route
- Signup → resend → verify
- Verify with expired token → error
- Verify with consumed token → error
- Rate limit triggers after N sends
- Recovery flow: request → receive email → verify → add passkey
- Config switching: Mailgun → Resend

### Manual Testing Checklist
- [ ] Mailgun sandbox domain (authorized recipients only)
- [ ] Mailgun production domain (real emails)
- [ ] Mailgun EU region
- [ ] Resend test domain
- [ ] Resend production domain
- [ ] Email deliverability (inbox, not spam)
- [ ] Email rendering (plain text + HTML)
- [ ] Mobile verification link (responsive)

## Documentation Updates

1. **README updates:**
   - Add email verification to features list
   - Add Mailgun/Resend to setup requirements

2. **Authentication guide:**
   - Email verification setup
   - Provider configuration (Mailgun vs Resend)
   - Domain verification (DKIM, SPF)
   - Rate limit configuration
   - Recovery method choices

3. **Configuration reference:**
   - `auth.email_verification.*` options
   - `auth.recovery.*` options
   - Template variables

4. **Migration guide:**
   - For existing installations
   - SQL migration commands
   - Config changes
   - User communication (how to verify existing emails)

5. **Provider setup guides:**
   - Mailgun: signup, domain verification, API key
   - Resend: signup, domain verification, API key
   - Free tier limitations
   - Production best practices

## Future Enhancements (Out of Scope)

1. **HTML email templates:** Add HTML support with inline CSS, test rendering across clients
2. **Custom Parsley templates:** Allow template override via `.pars` files with full Parsley support
3. **SMS verification:** Use same token table, add SMS provider interface (Twilio)
4. **Webhook support:** Delivery, bounce, spam complaint notifications from providers
5. **Email change flow:** Re-verification when user changes email address
6. **Batch operations:** Admin tools to bulk verify/invalidate
7. **Analytics:** Email open rates, verification conversion metrics
8. **Email log UI:** DevTools page to view email logs (sent, failed, costs)
9. **Multiple recovery methods:** Email + SMS + recovery codes simultaneously
10. **Social login:** OAuth as alternative to passkeys (separate feature)

## Dependencies

- `github.com/mailgun/mailgun-go/v4` (Mailgun SDK)
- `github.com/resend/resend-go/v2` (Resend SDK)
- Existing auth infrastructure (passkeys, sessions, recovery codes)

## Success Metrics

- Email verification completion rate >80%
- Email delivery rate >95%
- Recovery via email success rate >90%
- Average time to verify <5 minutes
- Zero email enumeration vulnerabilities
- Rate limits effective (no abuse reports)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Email deliverability issues | High | Support two providers, document SPF/DKIM setup |
| User loses access to email | Medium | Keep recovery codes as option, manual unlock |
| Spam abuse | Medium | Rate limits, audit logging, token expiry |
| Provider API changes | Low | Pin SDK versions, abstract via interface |
| GDPR compliance | High | Document data retention, add deletion commands |

## Design Decisions

### ✅ Email template format (V1)
**Decision:** Text-only templates for V1. HTML support added in future phase.  
**Rationale:** Text templates are simpler, more reliable (no spam filter issues), and sufficient for transactional emails. HTML can be added later without breaking changes.

### ✅ Password reset requirement
**Decision:** Not applicable - Basil uses passkeys only, no password support planned.  
**Rationale:** Passkeys are the primary authentication method. Email verification is for account recovery and contact verification, not password resets.

### ✅ Admin rate limit bypass
**Decision:** Add CLI bypass for support scenarios.  
**Clarification:** Rate limits are per-user (5 min cooldown, 10 emails/day). Use case: support staff manually resending to a user who hit their limit due to legitimate repeated attempts or email delivery issues.  
**Implementation:** `basil auth resend-verification --force <user_id>`

### ✅ Custom email templates
**Decision:** Support custom templates via Parsley files (future enhancement).  
**Implementation idea:** Allow override in config:
```yaml
auth:
  email_verification:
    templates:
      verification: "./templates/emails/verify.pars"
      recovery: "./templates/emails/recover.pars"
```
Templates would receive variables: `{user, token, url, site_name, site_url, ttl}` and return text (V1) or HTML (future).  
**Phase:** Add in Phase 4 (Developer Experience) or defer to future enhancement.

### ✅ Email audit logging
**Decision:** Log all outbound emails for audit and cost tracking.  
**What to log:**
- Timestamp
- User ID (if applicable)
- Recipient email
- Email type (verification, recovery)
- Provider used (Mailgun/Resend)
- Provider message ID
- Success/failure status
- Error message (if failed)

**Storage:** New `email_logs` table or append to existing audit log.  
**Benefits:**
- Audit trail for compliance
- Cost tracking (emails per user/day/month)
- Debug delivery issues
- Support investigations

**Schema:**
```sql
CREATE TABLE email_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    recipient TEXT NOT NULL,
    email_type TEXT NOT NULL,  -- 'verification' | 'recovery'
    provider TEXT NOT NULL,     -- 'mailgun' | 'resend'
    provider_message_id TEXT,
    status TEXT NOT NULL,       -- 'sent' | 'failed'
    error TEXT,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_email_logs_user ON email_logs(user_id);
CREATE INDEX idx_email_logs_created ON email_logs(created_at);
CREATE INDEX idx_email_logs_type ON email_logs(email_type);
```

## Simple Email API for Notifications (V1)

### Purpose

Allow developers to send simple text-only transactional emails using the already-configured email provider (Mailgun or Resend) without additional setup.

### Use Cases

**1. User notifications:**
- "You've logged in from a new device"
- "The admin has approved your submission"
- "Your page is now published"
- "Your account settings have changed"

**2. Admin alerts:**
- "A new user has registered"
- "A critical error occurred"
- "A form submission requires review"
- "System health check failed"

### API Design

Basil API (not @std module) accessible via `basil.email.send()`:

```parsley
let result = basil.email.send({
    to: "user@example.com",
    subject: "Your page is now published",
    body: "Your page 'About Us' has been published and is now live at https://example.com/about"
})

{if result.success}
    <p>Notification sent</p>
{else}
    <p>Failed to send: {result.error}</p>
{/if}
```

Admin alert example:
```parsley
let result = basil.email.send({
    to: basil.config.admin_email,
    subject: "New user registration",
    body: "User {user.email} just registered at {basil.time.now()}"
})
```

### Implementation (V1 Scope)

**Simple constraints:**
- Text-only (no HTML)
- No attachments
- Uses same "from" address as auth emails
- Uses already-configured provider (Mailgun or Resend)
- Single recipient per call
- Requires authenticated session (no anonymous sending)

**Configuration (extend existing):**
```yaml
auth:
  email_verification:
    provider: mailgun
    mailgun:
      api_key: "${MAILGUN_API_KEY}"
      domain: "mg.example.com"
      from: "noreply@example.com"  # Used for both auth and notification emails
    
    # Simple rate limit for developer emails
    developer_emails:
      enabled: true                # Can be disabled
      max_per_hour: 50            # Per site (prevent spam)
      max_per_day: 200
```

**Returns:**
```go
type EmailResult struct {
    Success    bool   `parsley:"success"`
    MessageID  string `parsley:"message_id,omitempty"`
    Error      string `parsley:"error,omitempty"`
}
```

**Logging:**
- All sends logged to `email_logs` table with `email_type='notification'`
- Includes user_id (who triggered the send), recipient, status, provider message ID

**Rate Limiting:**
- Global per-site limits (not per-user)
- 50/hour, 200/day default
- Prevents accidental bulk sending
- Returns error when limit exceeded

**Security:**
- Validate recipient address (RFC 5322)
- Require authenticated session (check `basil.auth.user` exists)
- No relay (must use configured domain)
- Auto-add footer: "Sent via Basil" for transparency

**Phase 3 Addition:**
Update Phase 3 to include:
- [ ] Implement `basil.email.send()` API
- [ ] Add developer email rate limiting
- [ ] Log to email_logs with type='notification'
- [ ] Add config option to enable/disable developer emails

**Estimated effort:** ~100 lines of code, reuses all existing infrastructure.

### Future Enhancements (Post-V1)

Once the simple API proves useful, consider:
- HTML support
- Batch sending (multiple recipients)
- Template files (`.pars` email templates)
- Attachments
- Scheduled sending (via job queue)
- Delivery webhooks (bounce/complaint handling)

These would be separate FEAT specs with proper abuse prevention, unsubscribe handling, and compliance features.

---

**Next Steps:**
1. Review and approve this spec
2. Allocate FEAT-084 in ID counter
3. Create implementation plan (PLAN-056)
4. Implement Phase 1 (infrastructure)
