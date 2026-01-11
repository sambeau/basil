# Combined Test Plan: FEAT-035 + FEAT-036

## Overview

This test plan covers both **FEAT-035 (Git over HTTPS)** and **FEAT-036 (CLI User Management)** as they are closely integrated — the CLI creates users and API keys that are used for Git authentication.

## Prerequisites

1. Clean test environment (no existing `.basil-auth.db`)
2. Terminal access
3. A test Basil site with `git init` already run
4. Git client installed

---

## Part 1: CLI User Management (FEAT-036)

### Test 1.1: First User Bootstrap

```bash
# Clean state
rm -f .basil-auth.db

# Create first user
basil users create --name "Admin User" --email "admin@example.com"
```

**Expected:**
- [ ] Output: `✓ Created user usr_...`
- [ ] `basil users list` shows user with `admin` role (auto-assigned)

---

### Test 1.2: Create Editor User

```bash
basil users create --name "Editor User" --email "editor@example.com"
```

**Expected:**
- [ ] User created with `editor` role (default for non-first users)
- [ ] `basil users list` shows both users with correct roles

---

### Test 1.3: Create API Key for Admin

```bash
# Get admin user ID
basil users list

# Create API key (replace <ADMIN_ID> with actual ID)
basil apikey create --user <ADMIN_ID> --name "Git Access"
```

**Expected:**
- [ ] Output shows: `✓ Created API key: bsl_live_...`
- [ ] Key ID shown: `Key ID: key_...`
- [ ] Warning: `(save this now — it won't be shown again)`

**⚠️ SAVE THE PLAINTEXT KEY! You'll need it for Git tests.**

---

### Test 1.4: Create API Key for Editor

```bash
basil apikey create --user <EDITOR_ID> --name "Editor Git"
```

**Expected:**
- [ ] API key created successfully

**⚠️ SAVE THIS KEY TOO!**

---

### Test 1.5: List API Keys

```bash
basil apikey list --user <ADMIN_ID>
```

**Expected:**
- [ ] Shows key with ID, NAME, PREFIX (truncated), CREATED, LAST USED
- [ ] Full plaintext key is NOT shown (only prefix like `bsl_live_abc...xyz`)

---

## Part 2: Git Server Configuration

### Test 2.1: Enable Git in Config

Create or edit `basil.yaml`:

```yaml
server:
  host: localhost
  port: 8080

auth:
  enabled: true
  registration: closed

git:
  enabled: true
  require_auth: true
```

**Expected:**
- [ ] Config is valid YAML

---

### Test 2.2: Start Server with Git Enabled

```bash
basil --dev
```

**Expected:**
- [ ] Server starts without errors
- [ ] Log shows: `[INFO] git server enabled at /.git/`
- [ ] Log shows: `[INFO] authentication enabled`

---

## Part 3: Git Clone Operations

### Test 3.1: Clone Without Auth (Should Fail)

```bash
# In a separate directory
git clone http://localhost:8080/.git test-clone-noauth
```

**Expected:**
- [ ] Clone fails with 401 Unauthorized
- [ ] Error message mentions authentication

---

### Test 3.2: Clone With Invalid API Key (Should Fail)

```bash
git clone http://user:invalid_key@localhost:8080/.git test-clone-badkey
```

**Expected:**
- [ ] Clone fails with 401 Unauthorized

---

### Test 3.3: Clone With Valid Admin API Key

```bash
# Use the API key saved from Test 1.3
git clone http://admin:<ADMIN_API_KEY>@localhost:8080/.git test-clone-admin
```

**Expected:**
- [ ] Clone succeeds
- [ ] Local repo contains site files
- [ ] Server logs show: `[git] GET /.git/info/refs by Admin User (admin)`

---

### Test 3.4: Clone With Valid Editor API Key

```bash
git clone http://editor:<EDITOR_API_KEY>@localhost:8080/.git test-clone-editor
```

**Expected:**
- [ ] Clone succeeds
- [ ] Server logs show user name and `editor` role

---

## Part 4: Git Push Operations

### Test 4.1: Push as Admin

```bash
cd test-clone-admin

# Make a change
echo "<!-- Admin test -->" >> index.pars
git add .
git commit -m "Admin test commit"
git push origin main
```

**Expected:**
- [ ] Push succeeds
- [ ] Server logs: `[git] Push received from ...`
- [ ] Server logs: `[INFO] git push received, reloading handlers...`
- [ ] Caches are cleared (changes visible immediately)

---

### Test 4.2: Push as Editor

```bash
cd test-clone-editor

echo "<!-- Editor test -->" >> index.pars
git add .
git commit -m "Editor test commit"
git push origin main
```

**Expected:**
- [ ] Push succeeds (editors can push)
- [ ] Server reloads handlers

---

### Test 4.3: Create Viewer Role and Try Push (Should Fail)

First, create a user without editor/admin role (if supported), or demote a user:

```bash
# This test verifies role checking
# Currently only admin/editor roles exist, both can push
# This test is for future viewer role implementation
```

**Expected (when viewer role is implemented):**
- [ ] Push fails with 403 Forbidden
- [ ] Error: "editor or admin role required for push"

---

## Part 5: Dev Mode (Unauthenticated Access)

### Test 5.1: Dev Mode Localhost Clone

```bash
# Ensure server started with --dev flag
# And request comes from localhost

git clone http://localhost:8080/.git test-clone-dev
```

**Expected:**
- [ ] Clone succeeds WITHOUT authentication
- [ ] Server logs: `[git] GET /.git/info/refs (dev mode, unauthenticated)`

---

### Test 5.2: Dev Mode Non-Localhost (Should Require Auth)

```bash
# If testing from another machine or using non-localhost address
git clone http://192.168.1.x:8080/.git test-clone-remote
```

**Expected:**
- [ ] Clone fails with 401 (dev mode only bypasses for localhost)

---

## Part 6: API Key Lifecycle

### Test 6.1: API Key Last Used Updates

```bash
# After using a key for git clone/push
basil apikey list --user <USER_ID>
```

**Expected:**
- [ ] LAST USED column shows today's date (was "never" before)

---

### Test 6.2: Revoke API Key

```bash
basil apikey revoke <KEY_ID>
```

**Expected:**
- [ ] Output: `✓ Revoked API key "..."`
- [ ] Key no longer in `basil apikey list`

---

### Test 6.3: Clone With Revoked Key (Should Fail)

```bash
git clone http://user:<REVOKED_KEY>@localhost:8080/.git test-revoked
```

**Expected:**
- [ ] Clone fails with 401 Unauthorized

---

## Part 7: Security & Edge Cases

### Test 7.1: Git Without Auth Enabled (Config Error)

Edit `basil.yaml`:
```yaml
auth:
  enabled: false

git:
  enabled: true
  require_auth: true  # Requires auth but auth is disabled!
```

```bash
basil --dev
```

**Expected:**
- [ ] Server fails to start
- [ ] Error: "git server requires auth but auth is not enabled"

---

### Test 7.2: Git Without Auth Requirement (Warning)

```yaml
git:
  enabled: true
  require_auth: false  # Insecure!
```

```bash
basil  # Production mode (no --dev)
```

**Expected:**
- [ ] Server starts but shows warning
- [ ] Warning: "git server is enabled without authentication - this is insecure!"

---

### Test 7.3: Concurrent CLI and Server Access

**Terminal 1:**
```bash
basil --dev
```

**Terminal 2 (while server running):**
```bash
basil users create --name "Concurrent" --email "concurrent@example.com"
basil apikey create --user <NEW_USER_ID> --name "Concurrent Key"
```

**Expected:**
- [ ] CLI commands succeed without "database is locked" error
- [ ] Server continues running normally
- [ ] New user can immediately use Git with new API key

---

### Test 7.4: Delete User Cascades API Keys

```bash
# Create throwaway user and key
basil users create --name "Throwaway" --email "throwaway@example.com"
basil apikey create --user <THROWAWAY_ID> --name "Will Be Deleted"

# Verify key exists
basil apikey list --user <THROWAWAY_ID>

# Delete user
basil users delete <THROWAWAY_ID> --force

# Try to list keys (user gone)
basil apikey list --user <THROWAWAY_ID>
```

**Expected:**
- [ ] Key exists before deletion
- [ ] User deletion succeeds
- [ ] Key is automatically deleted (cascade)

---

## Part 8: Role Management

### Test 8.1: Promote User to Admin

```bash
basil users set-role <EDITOR_ID> admin
```

**Expected:**
- [ ] Output: `✓ Set role for Editor User to admin`
- [ ] `basil users list` shows updated role

---

### Test 8.2: Demote User to Editor

```bash
# Ensure you have 2+ admins first
basil users set-role <ADMIN_ID> editor
```

**Expected:**
- [ ] Succeeds if not last admin
- [ ] Fails with "cannot remove the last admin user" if last admin

---

### Test 8.3: Cannot Delete Last Admin

```bash
# With only one admin remaining
basil users delete <LAST_ADMIN_ID> --force
```

**Expected:**
- [ ] Error: "cannot delete the last admin user"
- [ ] User still exists

---

## Summary Checklist

### FEAT-036: CLI User Management
| Test | Pass | Notes |
|------|------|-------|
| 1.1 First user bootstrap | ☐ | |
| 1.2 Create editor user | ☐ | |
| 1.3 Create API key (admin) | ☐ | |
| 1.4 Create API key (editor) | ☐ | |
| 1.5 List API keys | ☐ | |

### FEAT-035: Git over HTTPS
| Test | Pass | Notes |
|------|------|-------|
| 2.1 Enable git in config | ☐ | |
| 2.2 Start server with git | ☐ | |
| 3.1 Clone without auth (fail) | ☐ | |
| 3.2 Clone with bad key (fail) | ☐ | |
| 3.3 Clone as admin | ☐ | |
| 3.4 Clone as editor | ☐ | |
| 4.1 Push as admin | ☐ | |
| 4.2 Push as editor | ☐ | |
| 5.1 Dev mode localhost | ☐ | |

### Integration & Security
| Test | Pass | Notes |
|------|------|-------|
| 6.1 Last used updates | ☐ | |
| 6.2 Revoke API key | ☐ | |
| 6.3 Revoked key fails | ☐ | |
| 7.1 Config error (no auth) | ☐ | |
| 7.2 Warning (no require_auth) | ☐ | |
| 7.3 Concurrent access | ☐ | |
| 7.4 Cascade delete | ☐ | |
| 8.1 Promote to admin | ☐ | |
| 8.2 Demote to editor | ☐ | |
| 8.3 Last admin protection | ☐ | |

---

## Clean Up

After testing:
```bash
rm -rf test-clone-*
rm -f .basil-auth.db
```
