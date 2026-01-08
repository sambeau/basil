---
id: PLAN-056
feature: FEAT-084
title: "Implementation Plan for Email Verification and Notification API"
status: draft
created: 2026-01-08
---

# Implementation Plan: FEAT-084 Email Verification for Passkey Authentication

## Overview
Implement optional email verification for passkey authentication with single-provider support (Mailgun or Resend), email-based account recovery, and a simple notification API for developers. Implementation is organized into 5 phases over 2 weeks.

## Prerequisites
- [x] FEAT-084 specification approved
- [ ] Choose initial email provider for testing (Mailgun or Resend)
- [ ] Obtain test API key and configure test domain
- [ ] Review existing auth infrastructure (`auth/`, `server/session.go`)

## Tasks

### Phase 1: Core Infrastructure (Week 1, Days 1-2)

#### Task 1.1: Database Migrations
**Files**: `auth/migrations.go`, `auth/database.go`
**Estimated effort**: Small

Steps:
1. Add `email_verified_at DATETIME` column to users table
2. Create `email_verifications` table with all columns and indexes
3. Create `email_logs` table with all columns and indexes
4. Write migration rollback functions
5. Test migrations on clean database

Tests:
- Migration runs without errors
- Indexes are created correctly
- Foreign keys work (cascade delete)
- Rollback restores previous state

---

#### Task 1.2: Email Provider Interface
**Files**: `auth/email/provider.go` (new package)
**Estimated effort**: Small

Steps:
1. Create `auth/email` package
2. Define `Provider` interface with `Send()` and `Name()` methods
3. Define `Message` struct (From, To, Subject, Text, HTML)
4. Add provider factory function based on config
5. Add basic error types (ErrProviderNotConfigured, etc.)

Tests:
- Interface definition compiles
- Factory function returns correct provider type
- Error handling for missing config

---

#### Task 1.3: Mailgun Provider Adapter
**Files**: `auth/email/mailgun.go`
**Estimated effort**: Small

Steps:
1. Add `github.com/mailgun/mailgun-go/v4` dependency
2. Implement `MailgunProvider` struct
3. Implement `Send()` method (text + HTML support)
4. Implement `Name()` method
5. Handle EU region configuration (SetAPIBase)
6. Add comprehensive error handling

Tests:
- Unit test with mock Mailgun client
- Text-only email sends successfully
- HTML email sets HTML body
- EU region uses correct base URL
- Errors are propagated correctly

---

#### Task 1.4: Resend Provider Adapter
**Files**: `auth/email/resend.go`
**Estimated effort**: Small

Steps:
1. Add `github.com/resend/resend-go/v2` dependency
2. Implement `ResendProvider` struct
3. Implement `Send()` method (text + HTML support)
4. Implement `Name()` method
5. Add comprehensive error handling

Tests:
- Unit test with mock Resend client
- Text-only email sends successfully
- HTML email sends successfully
- Errors are propagated correctly

---

#### Task 1.5: Configuration Parsing
**Files**: `config/config.go`, `config/auth.go`
**Estimated effort**: Medium

Steps:
1. Add `EmailVerification` struct to auth config
2. Add `Recovery` struct to auth config
3. Add provider-specific structs (Mailgun, Resend)
4. Add validation for required fields
5. Add validation for mutually exclusive providers
6. Add default values (token_ttl: 1h, resend_cooldown: 5m, etc.)
7. Parse environment variables in API keys

Tests:
- Config loads with Mailgun provider
- Config loads with Resend provider
- Validation fails if both providers configured
- Validation fails if provider missing required fields
- Defaults are applied correctly
- Environment variables are expanded

---

#### Task 1.6: Email Templates
**Files**: `auth/email/templates.go`
**Estimated effort**: Small

Steps:
1. Create text templates for verification email
2. Create text templates for recovery email
3. Add template rendering function (Go templates)
4. Add template variable struct (DisplayName, URL, TTL, SiteName, SiteURL)
5. Add template validation

Tests:
- Verification template renders correctly
- Recovery template renders correctly
- All variables are substituted
- Missing variables return error

---

### Phase 2: Verification Flow (Week 1, Days 3-4)

#### Task 2.1: Token Generation and Storage
**Files**: `auth/email_verification.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `GenerateVerificationToken()` function (32 bytes crypto random)
2. Create `StoreVerificationToken()` function (bcrypt hash, save to DB)
3. Create `LookupVerificationToken()` function (find by hash)
4. Create `ConsumeVerificationToken()` function (mark consumed, check expiry)
5. Add cleanup function for expired tokens
6. Add tracking for send count and last sent time

Tests:
- Token is 32 bytes (256 bits)
- Token is cryptographically random (uniqueness)
- Hash is bcrypt with work factor 12
- Lookup finds correct token
- Expired tokens fail validation
- Consumed tokens fail validation
- Send count increments correctly

---

#### Task 2.2: Rate Limiting Logic
**Files**: `auth/rate_limit.go` (new)
**Estimated effort**: Medium

Steps:
1. Create `CheckVerificationRateLimit()` function
2. Check per-user cooldown (5 minutes)
3. Check per-user daily limit (10 emails)
4. Check per-email daily limit (20 emails)
5. Return next available send time on rate limit
6. Add database queries for rate limit checks

Tests:
- Cooldown prevents sends within 5 minutes
- Daily limit prevents 11th send
- Per-email limit prevents spam to victim
- Rate limit resets after cooldown/24h
- Next available time is calculated correctly

---

#### Task 2.3: Update Signup Handler
**Files**: `auth/handlers.go`, `auth/webauthn.go`
**Estimated effort**: Medium

Steps:
1. Update `Register` handler to generate verification token
2. Send verification email after user creation
3. Set `email_verification_pending = true` in session
4. Return `email_verification_sent: true` in response
5. Handle errors gracefully (user created even if email fails)
6. Log to email_logs table

Tests:
- Signup creates user with email_verified_at = NULL
- Verification email is sent
- Session flag is set
- Response includes email_verification_sent
- Email failure doesn't block signup
- Email log entry is created

---

#### Task 2.4: Verify Email Handler
**Files**: `auth/handlers.go`, `auth/email_verification.go`
**Estimated effort**: Medium

Steps:
1. Create `GET /__auth/verify-email` handler
2. Extract token from query string
3. Look up and validate token (not expired, not consumed)
4. Set `email_verified_at = NOW()` on user
5. Mark token as consumed
6. Clear `email_verification_pending` session flag
7. Redirect to dashboard with flash message
8. Show error page on failure with resend option

Tests:
- Valid token verifies email successfully
- Expired token returns error
- Consumed token returns error
- Invalid token returns error
- Session flag is cleared
- Flash message is shown
- Error page offers resend

---

#### Task 2.5: Resend Verification Handler
**Files**: `auth/handlers.go`
**Estimated effort**: Small

Steps:
1. Create `POST /__auth/resend-verification` handler
2. Check authenticated session
3. Check rate limits
4. Generate new token (invalidate old)
5. Send verification email
6. Return success with next_send_available_at
7. Log to email_logs table

Tests:
- Rate limit prevents rapid resends
- New token is generated
- Old token is invalidated
- Email is sent
- Response includes cooldown time
- Unauthenticated request fails

---

#### Task 2.6: Session Context Updates
**Files**: `server/session.go`, `auth/session.go`
**Estimated effort**: Small

Steps:
1. Add `EmailVerificationPending bool` to SessionData
2. Add `EmailVerifiedAt *time.Time` to SessionData
3. Update session creation to populate flags
4. Expose to Parsley via `basil.auth.user`
5. Update session tests

Tests:
- Session data includes new fields
- Flags are populated correctly
- Parsley can access flags
- Existing sessions still work

---

#### Task 2.7: Route Protection Middleware
**Files**: `auth/middleware.go`
**Estimated effort**: Medium

Steps:
1. Update auth middleware to check email verification
2. Add check for `require_verification` config
3. Return 403 if unverified and required
4. Redirect to `/__auth/verify-email-required` page
5. Allow unprotected routes (login, verify, resend)

Tests:
- Verified users access protected routes
- Unverified users blocked when required
- Unverified users allowed when not required
- Redirect works correctly
- Verification pages always accessible

---

#### Task 2.8: Verification Required Info Page
**Files**: `auth/handlers.go`, `auth/templates/` (if using templates)
**Estimated effort**: Small

Steps:
1. Create `GET /__auth/verify-email-required` handler
2. Show friendly message explaining verification requirement
3. Add resend button
4. Add logout option
5. Style consistently with other auth pages

Tests:
- Page renders correctly
- Resend button works
- Logout redirects properly
- Consistent styling

---

### Phase 3: Email Recovery & Notification API (Week 2, Days 1-2)

#### Task 3.1: Email Recovery Request Handler
**Files**: `auth/recovery.go`, `auth/handlers.go`
**Estimated effort**: Medium

Steps:
1. Create `POST /__auth/recover/email` handler
2. Look up user by email (constant-time)
3. Check `email_verified_at IS NOT NULL`
4. Generate recovery token (reuse verification table with type flag)
5. Send recovery email
6. Always return success (no enumeration)
7. Log to email_logs table

Tests:
- Verified email receives recovery link
- Unverified email returns success (no email sent)
- Non-existent email returns success (no email sent)
- Response time is constant
- No enumeration vulnerability

---

#### Task 3.2: Email Recovery Verify Handler
**Files**: `auth/recovery.go`, `auth/handlers.go`
**Estimated effort**: Medium

Steps:
1. Create `GET /__auth/recover/verify` handler
2. Validate recovery token
3. Create authenticated session
4. Redirect to "add new passkey" page
5. Show error with recovery code fallback option
6. Consume token

Tests:
- Valid token creates session
- Expired token shows error
- Consumed token shows error
- Redirect works correctly
- Recovery code fallback is offered

---

#### Task 3.3: Recovery Email Template
**Files**: `auth/email/templates.go`
**Estimated effort**: Small

Steps:
1. Add recovery email template
2. Include recovery link
3. Include expiry time
4. Add security message
5. Test rendering

Tests:
- Template renders correctly
- All variables are substituted
- Link format is correct

---

#### Task 3.4: Recovery Codes Configuration
**Files**: `auth/recovery.go`, `config/auth.go`
**Estimated effort**: Small

Steps:
1. Add `recovery.codes_enabled` config option
2. Add `recovery.email_enabled` config option
3. Update recovery code generation logic to respect config
4. Update recovery handler to check email recovery option
5. Allow both methods simultaneously

Tests:
- Codes disabled when configured
- Email recovery enabled when configured
- Both methods work simultaneously
- Config validation works

---

#### Task 3.5: Simple Notification API - Core
**Files**: `server/email.go` (new), `server/prelude.go`
**Estimated effort**: Medium

Steps:
1. Create `basil.email.send()` function
2. Validate parameters (to, subject, body required)
3. Check authenticated session
4. Validate recipient email (RFC 5322)
5. Check rate limits (50/hour, 200/day per site)
6. Send email via configured provider
7. Log to email_logs with type='notification'
8. Return result struct (success, message_id, error)

Tests:
- Authenticated user can send email
- Unauthenticated request fails
- Invalid recipient fails validation
- Rate limit prevents spam
- Email is sent via provider
- Result includes message ID
- Email log entry created

---

#### Task 3.6: Simple Notification API - Configuration
**Files**: `config/auth.go`
**Estimated effort**: Small

Steps:
1. Add `developer_emails` config block
2. Add `enabled` flag (default: true)
3. Add `max_per_hour` (default: 50)
4. Add `max_per_day` (default: 200)
5. Parse and validate config

Tests:
- Config loads with defaults
- Can disable developer emails
- Rate limits are configurable
- Validation works

---

#### Task 3.7: Simple Notification API - Rate Limiting
**Files**: `server/email.go`, `auth/rate_limit.go`
**Estimated effort**: Small

Steps:
1. Create rate limit tracker for notification emails
2. Track per-site (not per-user)
3. Check hourly limit (50)
4. Check daily limit (200)
5. Return error when exceeded
6. Store in database or in-memory cache

Tests:
- 51st email in hour fails
- 201st email in day fails
- Limits reset correctly
- Error message is clear

---

#### Task 3.8: Simple Notification API - Expose to Parsley
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Add `email` namespace to basil prelude
2. Add `send()` function
3. Parse Parsley table argument {to, subject, body}
4. Call core email sending function
5. Return result as Parsley table {success, message_id, error}

Tests:
- Parsley can call basil.email.send()
- Parameters are parsed correctly
- Result is returned as table
- Errors are handled gracefully

---

### Phase 4: Developer Experience (Week 2, Days 3-4)

#### Task 4.1: CLI Verify Email Command
**Files**: `cmd/basil/auth.go` (or new auth commands file)
**Estimated effort**: Small

Steps:
1. Create `basil auth verify-email <user_id>` command
2. Look up user by ID
3. Set `email_verified_at = NOW()`
4. Clear pending verification tokens
5. Show success message

Tests:
- Command verifies user
- Invalid user ID shows error
- Already verified user shows message
- Success message is clear

---

#### Task 4.2: CLI Resend Verification Command
**Files**: `cmd/basil/auth.go`
**Estimated effort**: Small

Steps:
1. Create `basil auth resend-verification <user_id>` command
2. Add `--force` flag to bypass rate limits
3. Generate new token
4. Send email
5. Show success message with email address

Tests:
- Command sends email
- --force bypasses rate limits
- Invalid user ID shows error
- Success message includes email

---

#### Task 4.3: CLI Reset Verification Command
**Files**: `cmd/basil/auth.go`
**Estimated effort**: Small

Steps:
1. Create `basil auth reset-verification <user_id>` command
2. Set `email_verified_at = NULL`
3. Delete pending tokens
4. Show success message

Tests:
- Command resets verification
- Invalid user ID shows error
- Success message is clear

---

#### Task 4.4: CLI Verification Status Command
**Files**: `cmd/basil/auth.go`
**Estimated effort**: Small

Steps:
1. Create `basil auth status <user_id>` command
2. Show user email
3. Show verification status (verified/unverified)
4. Show verification date if verified
5. Show pending tokens count
6. Show last email sent time

Tests:
- Command shows status
- Invalid user ID shows error
- Output is formatted clearly

---

#### Task 4.5: CLI Email Logs Command
**Files**: `cmd/basil/auth.go`
**Estimated effort**: Small

Steps:
1. Create `basil auth email-logs` command
2. Add `--user <user_id>` filter
3. Add `--limit <n>` option (default: 100)
4. Query email_logs table
5. Display as formatted table (timestamp, user, recipient, type, status)

Tests:
- Command shows recent logs
- User filter works
- Limit works
- Table is formatted clearly

---

#### Task 4.6: Expose Verification Status to Parsley
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Ensure `basil.auth.user.email_verification_pending` is accessible
2. Ensure `basil.auth.user.email_verified_at` is accessible
3. Add helper functions if needed
4. Document in Parsley reference

Tests:
- Parsley can access verification flags
- Values are correct
- Works in conditional expressions

---

#### Task 4.7: Dev Mode Warnings
**Files**: `server/devtools.go`
**Estimated effort**: Small

Steps:
1. Add warning if using Mailgun sandbox domain
2. Add warning if no email provider configured
3. Add warning if HTTPS not enabled (verification links)
4. Show in dev tools panel
5. Show in server startup logs

Tests:
- Warnings appear in dev mode
- Warnings don't appear in production
- Messages are helpful

---

#### Task 4.8: Migration Guide for Existing Installations
**Files**: `docs/guide/migration-email-verification.md` (new)
**Estimated effort**: Small

Steps:
1. Document database migration steps
2. Document config changes required
3. Document how to verify existing users
4. Document provider setup (Mailgun or Resend)
5. Document DKIM/SPF requirements
6. Include troubleshooting section

Tests:
- Guide is clear and complete
- Steps are tested on clean installation
- Links are correct

---

### Phase 5: Testing & Documentation (Week 2, Day 5)

#### Task 5.1: Unit Tests - Token & Verification
**Files**: `auth/email_verification_test.go`
**Estimated effort**: Medium

Steps:
1. Test token generation (entropy, uniqueness)
2. Test token hashing (bcrypt work factor)
3. Test token lookup
4. Test token consumption
5. Test expiry handling
6. Test cleanup of expired tokens

Tests: (All test cases listed above)

---

#### Task 5.2: Unit Tests - Email Providers
**Files**: `auth/email/mailgun_test.go`, `auth/email/resend_test.go`
**Estimated effort**: Medium

Steps:
1. Test Mailgun provider with mock client
2. Test Resend provider with mock client
3. Test text-only emails
4. Test HTML emails
5. Test error handling
6. Test EU region configuration (Mailgun)

Tests: (All test cases listed above)

---

#### Task 5.3: Unit Tests - Rate Limiting
**Files**: `auth/rate_limit_test.go`
**Estimated effort**: Small

Steps:
1. Test per-user cooldown
2. Test per-user daily limit
3. Test per-email daily limit
4. Test rate limit reset
5. Test next available time calculation

Tests: (All test cases listed above)

---

#### Task 5.4: Integration Tests - Full Verification Flow
**Files**: `auth/integration_test.go`
**Estimated effort**: Large

Steps:
1. Test signup → email → verify → access protected route
2. Test signup → resend → verify
3. Test expired token error
4. Test consumed token error
5. Test rate limit triggering
6. Test config switching (Mailgun ↔ Resend)
7. Test verification required enforcement

Tests: (All scenarios listed above)

---

#### Task 5.5: Integration Tests - Recovery Flow
**Files**: `auth/integration_test.go`
**Estimated effort**: Medium

Steps:
1. Test recovery request → email → verify → add passkey
2. Test recovery with unverified email (no email sent)
3. Test recovery with non-existent email (no enumeration)
4. Test expired recovery token
5. Test consumed recovery token

Tests: (All scenarios listed above)

---

#### Task 5.6: Integration Tests - Notification API
**Files**: `server/email_test.go`
**Estimated effort**: Small

Steps:
1. Test authenticated user sends notification
2. Test unauthenticated request fails
3. Test invalid recipient fails
4. Test rate limit enforcement
5. Test email logging

Tests: (All scenarios listed above)

---

#### Task 5.7: Update Authentication Documentation
**Files**: `docs/guide/authentication.md`
**Estimated effort**: Medium

Steps:
1. Add email verification section
2. Document configuration options
3. Add code examples for Parsley
4. Document recovery options
5. Add notification API examples
6. Update table of contents

Tests:
- Documentation is complete
- Code examples work
- Links are correct
- Screenshots are current

---

#### Task 5.8: Create Provider Setup Guides
**Files**: `docs/guide/email-providers.md` (new)
**Estimated effort**: Medium

Steps:
1. Document Mailgun setup (signup, domain, API key, DKIM/SPF)
2. Document Resend setup (signup, domain, API key, DKIM)
3. Document free tier limitations
4. Document production best practices
5. Add troubleshooting section
6. Add deliverability tips

Tests:
- Guides are clear and complete
- Steps are tested
- Screenshots are included
- Links are correct

---

#### Task 5.9: Update Configuration Reference
**Files**: `docs/guide/configuration.md`
**Estimated effort**: Small

Steps:
1. Document `auth.email_verification.*` options
2. Document `auth.recovery.*` options
3. Document template variables
4. Add complete YAML examples
5. Document defaults

Tests:
- Reference is complete
- Examples are tested
- Defaults are correct

---

#### Task 5.10: Update README
**Files**: `README.md`
**Estimated effort**: Small

Steps:
1. Add email verification to features list
2. Add Mailgun/Resend to requirements
3. Update quick start if needed
4. Add link to email provider guide

Tests:
- README is updated
- Links work
- Features list is accurate

---

## Validation Checklist
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build -o basil ./cmd/basil`
- [ ] Build succeeds: `go build -o pars ./cmd/pars`
- [ ] Linter passes: `golangci-lint run`
- [ ] Manual test: Mailgun email sends and verifies
- [ ] Manual test: Resend email sends and verifies
- [ ] Manual test: Recovery flow works end-to-end
- [ ] Manual test: Notification API works
- [ ] Manual test: Rate limits enforce correctly
- [ ] Manual test: Dev warnings show appropriately
- [ ] Documentation is complete and accurate
- [ ] BACKLOG.md updated with any deferrals
- [ ] CHANGELOG.md updated with new features

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-08 | Plan created | ✅ Complete | Ready to begin implementation |

## Deferred Items
Items to add to BACKLOG.md after V1 implementation:
- HTML email templates (requires CSS inlining, testing across clients)
- Custom Parsley email templates (requires template engine integration)
- SMS verification (requires Twilio/similar integration, new provider interface)
- Email change flow (requires re-verification workflow)
- Webhook support for delivery/bounce notifications (requires provider webhook handling)
- Batch email sending (requires queue system)
- Email analytics (open rates, click tracking)
- Email log UI in DevTools (requires UI implementation)
- Attachments support (requires file handling, size limits)
- Multiple recovery methods simultaneously (requires UI/UX design)

## Notes

### Provider Choice
Start with one provider for initial testing (recommend Resend for simpler API), then test Mailgun provider before release. Both must work correctly.

### Security Review Points
- Token entropy verified at 256 bits
- Bcrypt work factor confirmed at 12
- Rate limits tested under load
- No email enumeration vulnerabilities
- HTTPS required in production

### Testing Strategy
- Unit tests for all core functions
- Integration tests for complete flows
- Manual testing with real email providers (sandbox and production)
- Test deliverability (inbox vs spam)
- Test across email clients (Gmail, Outlook, Apple Mail)

### Documentation Priority
1. Email provider setup guides (critical for adoption)
2. Configuration reference (developers need this first)
3. Migration guide (for existing installations)
4. Authentication guide updates
5. README updates

### Timeline Notes
- Phases 1-2 can be done by one developer
- Phase 3 notification API can be parallelized with recovery work
- Phase 4 CLI commands are independent and can be split across developers
- Phase 5 testing should not be rushed - allocate full day for integration tests
