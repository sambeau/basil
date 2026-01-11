# Manual Testing Checklist

This document lists tests that require manual verification — features that can't easily be tested by automated unit tests due to external dependencies, browser interactions, or real-world infrastructure requirements.

## Basil Server

- [ ] **Dev vs Production mode** — Run with `--dev`, verify HTTP works on localhost. Run without `--dev`, verify HTTPS required
- [ ] **HTTPS autocert** — Test Let's Encrypt with a real domain. Verify certificates auto-renew
- [ ] **Hot reload (dev mode)** — Edit .pars file while server running, verify changes appear on next request without restart
- [ ] **SIGHUP reload (prod)** — Send SIGHUP to running server, verify scripts reloaded without dropping connections
- [ ] **Response caching** — Test route caching with TTL config, verify cached responses served, cache expires correctly
- [ ] **Graceful shutdown** — Send SIGTERM, verify in-flight requests complete before exit

## Authentication (Passkeys)

- [ ] **Full passkey flow** — Register new passkey → Login with passkey → Logout → Verify session management across browser restart
- [ ] **Recovery codes** — Generate recovery codes, use one to login, verify used code rejected on second attempt
- [ ] **Route protection** — Test protected routes redirect to login, verify redirect back to original page after auth
- [ ] **Cross-browser passkeys** — Register passkey in Safari, verify login works in Chrome and Firefox
- [ ] **Hardware authenticators** — Test with USB security key (YubiKey), platform authenticator (TouchID/FaceID/Windows Hello)
- [ ] **Mobile passkeys** — Test passkey registration and login on iOS Safari and Android Chrome
- [ ] **Session expiry** — Verify sessions expire after configured TTL, user must re-authenticate

## Dev Tools

- [ ] **Dev tools access control** — Verify `/__/*` routes work in dev mode, return 404 in production mode
- [ ] **Logs viewer** — Generate logs via `dev.log()`, verify they appear at `/__/logs` with correct levels
- [ ] **Log routing** — Test `dev.log("message", {route: "myroute"})`, verify appears at `/__/logs/myroute`
- [ ] **Environment viewer** — Verify `/__/env` shows config without exposing secrets
- [ ] **Database viewer** — Test all `/__/db` functionality (see FEAT-021 tests below)

### FEAT-021: SQLite Dev Tools

- [ ] **Table list** — Create multiple tables, verify all appear with correct names, column counts, row counts
- [ ] **Schema display** — Create tables with INTEGER, TEXT, REAL, BLOB columns, verify PK/NOT NULL constraints shown
- [ ] **Table data view** — Click row count link, verify navigation to `/__/db/view/{table}`
- [ ] **Data display** — Verify columns, rows, NULL display (gray italic), row numbers, sticky header
- [ ] **Empty table** — View table with 0 rows, verify "No data in this table" message
- [ ] **Large table** — Create table with >1000 rows, verify only 1000 shown (LIMIT protection)
- [ ] **CSV download** — Download CSV, open in spreadsheet app, verify headers and data correct
- [ ] **CSV upload - integers** — Upload CSV with integer column, verify INTEGER type inferred
- [ ] **CSV upload - floats** — Upload CSV with decimal numbers, verify REAL type inferred
- [ ] **CSV upload - mixed** — Upload CSV with integers and floats in same column, verify REAL type
- [ ] **CSV upload - text** — Upload CSV with text values, verify TEXT type
- [ ] **CSV upload - empty** — Upload CSV with empty cells, verify stored as NULL
- [ ] **CSV upload - replace** — Upload to existing table, verify old data completely replaced
- [ ] **Create table** — Create new table via form, verify appears with id column and 1 row
- [ ] **Create duplicate** — Try to create table with existing name, verify error message
- [ ] **Create invalid name** — Try names with spaces, special chars, leading digit — verify validation
- [ ] **Delete table** — Delete table, verify confirmation dialog appears, table removed
- [ ] **Delete cancel** — Click delete, cancel confirmation, verify table still exists
- [ ] **Fixed header** — With many tables, scroll page, verify header stays fixed at top
- [ ] **Safari forms** — In Safari: create table, upload CSV — verify no phantom file downloads

## Parsley Language Features

### SFTP Operations (requires SFTP server)

- [ ] **SFTP connect** — Connect to SFTP server with password authentication
- [ ] **SFTP key auth** — Connect using SSH key file
- [ ] **SFTP read** — Read file from remote server
- [ ] **SFTP write** — Write file to remote server
- [ ] **SFTP list** — List directory contents
- [ ] **SFTP mkdir/rmdir** — Create and remove directories
- [ ] **SFTP error handling** — Verify proper errors for connection failures, permission denied

### HTTP Operations

- [ ] **fetch() GET** — Fetch from real external API (httpbin.org, jsonplaceholder.typicode.com)
- [ ] **fetch() headers** — Send custom headers, verify server receives them
- [ ] **fetch() POST JSON** — POST JSON body, verify response parsing
- [ ] **fetch() POST form** — POST form-encoded data
- [ ] **fetch() timeout** — Test request timeout handling
- [ ] **fetch() errors** — Verify proper errors for network failures, 4xx/5xx responses

### Database Operations

- [ ] **SQLite concurrency** — Multiple concurrent requests hitting database, verify connection pooling
- [ ] **Database transactions** — Test transaction with intentional error, verify rollback
- [ ] **Large queries** — Query returning 10k+ rows, verify memory usage reasonable
- [ ] **Database locking** — Concurrent writes, verify proper locking behavior

### File Operations

- [ ] **Large files** — Read/write files >100MB, verify memory usage reasonable
- [ ] **Unicode files** — Read/write files with unicode characters, emoji, RTL text
- [ ] **Binary files** — Read binary files, verify no corruption
- [ ] **File permissions** — Write file, verify correct permissions set

### Process Execution

- [ ] **process() basic** — Run simple command, capture output
- [ ] **process() timeout** — Long-running command with timeout, verify killed
- [ ] **process() stdin** — Pass input to command via stdin
- [ ] **process() errors** — Command that fails, verify error captured

### Date/Time

- [ ] **Timezone handling** — Operations across timezone boundaries
- [ ] **DST transitions** — Date arithmetic across daylight saving transitions
- [ ] **Locale formatting** — Format dates in different locales (en-US, de-DE, ja-JP)

### Locale & Formatting

- [ ] **Currency formatting** — Format currency in different locales
- [ ] **Number formatting** — Thousands separators, decimal points by locale
- [ ] **Date formatting** — Date patterns vary by locale

### Security

- [ ] **Path traversal blocked** — Attempt `../../../etc/passwd`, verify blocked
- [ ] **Symlink outside sandbox** — Create symlink pointing outside handler dir, verify blocked
- [ ] **Write outside sandbox** — Attempt write outside allowed directory, verify blocked

### Markdown

- [ ] **Tables** — Markdown tables render correctly
- [ ] **Code blocks** — Fenced code blocks with syntax highlighting
- [ ] **Images** — Image references resolve correctly
- [ ] **Links** — Internal and external links work

### Table Module

- [ ] **Large datasets** — Table() with 10k+ rows, verify acceptable performance
- [ ] **Complex queries** — Chained where().orderBy().select().limit()
- [ ] **toHTML output** — Verify clean HTML table output
- [ ] **toCSV output** — Verify valid CSV output

### Module System

- [ ] **Circular imports** — Module A imports B, B imports A — verify handled gracefully
- [ ] **Deep nesting** — Deeply nested imports (5+ levels)
- [ ] **Module caching** — Same module imported twice, verify cached (not re-evaluated)

## Example Apps

- [ ] **hello example** — Run `examples/hello`, test all pages in browser
- [ ] **auth example** — Run `examples/auth`, test full registration/login/logout flow

## Browser Compatibility

- [ ] **Safari** — Test key features (forms, passkeys, dev tools)
- [ ] **Chrome** — Test key features
- [ ] **Firefox** — Test key features
- [ ] **iOS Safari** — Test on iPhone/iPad
- [ ] **Android Chrome** — Test on Android device

---

## External Setup Required

### 1. SFTP Server (for SFTP tests)

**Option A: Docker (recommended)**
```bash
docker run -d \
  --name sftp-test \
  -p 2222:22 \
  -e SFTP_USERS=testuser:testpass:::upload \
  atmoz/sftp
```

**Option B: Local SSH**
Enable SFTP in `/etc/ssh/sshd_config`:
```
Subsystem sftp /usr/lib/openssh/sftp-server
```

**Test connection:**
```bash
sftp -P 2222 testuser@localhost
```

### 2. Real Domain (for HTTPS autocert)

1. Point a domain's DNS A record to your test machine's public IP
2. Open port 443 (firewall/router configuration)
3. Run Basil without `--dev` flag to test Let's Encrypt certificate issuance

### 3. Hardware Authenticators (for passkey tests)

- **YubiKey** — USB security key (any FIDO2-compatible key)
- **TouchID** — Mac with Touch ID sensor
- **Windows Hello** — Windows PC with biometrics or PIN
- **Mobile device** — Phone as passkey via QR code scanning

### 4. Multiple Browsers

Install for cross-browser testing:
- Safari (macOS default)
- Chrome
- Firefox
- Mobile: iOS Safari, Android Chrome

---

## What's Already Covered by Unit Tests

These do **NOT** need manual testing (automated coverage exists):

- Array methods (pick, take, shuffle, includes)
- Dictionary operations
- String manipulation
- Path templates (`@({...})`)
- Trailing commas
- `in` operator
- Type inference in CSV import
- Basic database CRUD operations
- Error message formatting
- Regex operations
- YAML/JSON/CSV parsing (basic cases)
- Table module operations (basic)
- Root path alias (`@~/`)
- Public dir path rewriting
- Valid/invalid table name validation

## Email Verification (FEAT-084)

### Test Environment Setup

Use provided test credentials:

**Mailgun:**
```yaml
email_verification:
  provider: mailgun
  mailgun:
    api_key: "8d62a15e184a96921189f2976fae04cd-f6d80573-cb1a0b2a"
    domain: "mg.tickly.org"
    from: "test@mg.tickly.org"
```

**Resend:**
```yaml
email_verification:
  provider: resend
  resend:
    api_key: "re_CTmuvKDZ_Hs8QFRGe2hnRQWmPt3uq9nxJ"
    from: "onboarding@resend.dev"
```

**Test email:** `sambeau@mac.com`

### Manual Test Cases

- [ ] **Basic verification flow** — Register with email, receive verification link, click link, verify email marked as verified
- [ ] **Parsley context** — Test `basil.auth.user.email_verified_at` and `email_verification_pending` in templates
- [ ] **Resend with cooldown** — Register, try immediate resend (blocked), wait 5min, resend (succeeds)
- [ ] **Force resend** — Use `basil auth resend-verification --force` to bypass cooldown
- [ ] **Daily rate limit** — Send 10 verification emails, verify 11th blocked with "daily limit exceeded"
- [ ] **Manual verification** — Use `basil auth verify-email <user_id>` to verify without clicking link
- [ ] **Reset verification** — Use `basil auth reset-verification <user_id>` to unverify user
- [ ] **Email audit logs** — Use `basil auth email-logs` to view sent emails, filter by user
- [ ] **Require verification (blocking)** — Set `require_verification: true`, verify unverified users redirected
- [ ] **Token expiry** — Set `token_ttl: 1m`, wait 2min, click link (should fail)
- [ ] **Dev mode warnings** — Test sandbox domain, missing HTTPS, incomplete config warnings
- [ ] **Provider switch** — Start with Mailgun, switch to Resend, verify emails send correctly
- [ ] **Multiple users** — Register 3 users, verify each gets separate emails and status
- [ ] **Real provider** — Test with your own Mailgun or Resend account and domain
- [ ] **Spam folder** — Check if verification emails land in spam (deliverability test)
- [ ] **Email formatting** — Verify email template renders correctly in multiple email clients
- [ ] **Mobile email** — Click verification link from mobile device email client
- [ ] **Link expiry message** — Click expired token, verify clear error message shown
- [ ] **Invalid token** — Manually edit token in URL, verify error message
- [ ] **DNS configuration** — Set up custom domain with SPF/DKIM, test deliverability
- [ ] **Production HTTPS** — Test verification with HTTPS in production (link must use HTTPS)

### CLI Commands to Test

```bash
# Check user status
./basil auth status <user_id>

# Manually verify email
./basil auth verify-email <user_id>

# Resend verification (bypass rate limits)
./basil auth resend-verification --force <user_id>

# Reset verification state
./basil auth reset-verification <user_id>

# View email audit logs
./basil auth email-logs --limit 10
./basil auth email-logs --user <user_id>
```

### Documentation to Review

- [Email Verification Guide](./guide/email-verification.md)
- [Mailgun Setup Guide](./guide/email-providers/mailgun.md)
- [Resend Setup Guide](./guide/email-providers/resend.md)
- [FEAT-084 Specification](./specs/FEAT-084.md)
