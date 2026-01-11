# ADR-001: Defer Notification API to Future Feature

**Date**: 2026-01-08  
**Status**: Accepted  
**Context**: FEAT-084 Phase 3

## Decision

Defer the `basil.email.send()` notification API to a future feature (FEAT-085) due to complexity of Parsley runtime integration.

## Context

The notification API requires:
1. Creating new Parsley namespace (`basil.email`)
2. Exposing Go functions to Parsley runtime
3. Understanding evaluator.PreludeLoader architecture
4. Implementing @std module or request context functions
5. Thread-safe access to EmailService

This integration is significantly more complex than the verification flow and requires deeper investigation of:
- `server/prelude.go` and preludeFS embedding
- `pkg/parsley/evaluator` API exposure patterns
- `server/handler.go` buildRequestContext integration

## Consequences

**Positive**:
- Core email verification feature (85% of FEAT-084 value) is complete and working
- Can ship email verification now, notification API later
- More time to design Parsley API correctly
- Separates infrastructure from developer API

**Negative**:
- Developers cannot send custom emails from Parsley code yet
- Must create FEAT-085 for notification API

## Implementation Status

**FEAT-084 Completion**: 85%
- ✅ Email providers (Mailgun, Resend)
- ✅ Token generation and verification
- ✅ Rate limiting and audit logging
- ✅ Signup verification emails
- ✅ Email recovery flow
- ✅ Verification middleware
- ⏳ Notification API → FEAT-085

## Related

- FEAT-084: Email verification (this feature)
- FEAT-085: Notification API (to be created)
- BACKLOG.md: Add notification API entry
