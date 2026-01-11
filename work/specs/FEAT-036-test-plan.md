# FEAT-036 Manual Test Plan

## Prerequisites

1. Clean test environment (no existing `.basil-auth.db`)
2. Terminal access to run `basil` commands
3. Browser for passkey registration tests
4. Two terminal windows for concurrency tests

---

## Test 1: First User Creation (Bootstrap)

**Objective:** Verify first user is always admin

```bash
# Ensure clean state
rm -f .basil-auth.db

# Create first user without --role flag
basil users create --name "Admin User" --email "admin@example.com"
```

**Expected:**
- [ ] Output shows `✓ Created user usr_...`
- [ ] No errors

```bash
# Verify role is admin
basil users list
```

**Expected:**
- [ ] User shows `admin` role (not `editor`)
- [ ] Table displays ID, NAME, EMAIL, ROLE, CREATED columns

---

## Test 2: First User Role Override Ignored

**Objective:** Verify first user can't be demoted during creation

```bash
rm -f .basil-auth.db

# Try to create first user as editor
basil users create --name "First User" --email "first@example.com" --role editor
```

**Expected:**
- [ ] Warning message: `Note: First user is always admin (ignoring --role editor)`
- [ ] User created successfully
- [ ] `basil users list` shows user as `admin`

---

## Test 3: Subsequent Users Default to Editor

**Objective:** Verify second+ users get editor role by default

```bash
# Create second user (assumes first user exists)
basil users create --name "Editor User" --email "editor@example.com"
```

**Expected:**
- [ ] User created successfully
- [ ] `basil users list` shows new user as `editor`

---

## Test 4: Create User with Explicit Admin Role

**Objective:** Verify admin role can be assigned to subsequent users

```bash
basil users create --name "Second Admin" --email "admin2@example.com" --role admin
```

**Expected:**
- [ ] User created successfully
- [ ] `basil users list` shows user as `admin`

---

## Test 5: Invalid Role Rejected

**Objective:** Verify invalid roles are rejected

```bash
basil users create --name "Test" --email "test@example.com" --role superuser
```

**Expected:**
- [ ] Error: `invalid role: superuser (use: admin or editor)`
- [ ] No user created

---

## Test 6: User Update

**Objective:** Verify name and email can be updated

```bash
# Get a user ID from list
basil users list

# Update both name and email (use actual user ID)
basil users update <USER_ID> --name "Updated Name" --email "updated@example.com"
```

**Expected:**
- [ ] Output: `✓ Updated user usr_...`

```bash
basil users show <USER_ID>
```

**Expected:**
- [ ] Name shows "Updated Name"
- [ ] Email shows "updated@example.com"

---

## Test 7: User Update Validation

**Objective:** Verify at least one field required

```bash
basil users update <USER_ID>
```

**Expected:**
- [ ] Error: `at least one of --name or --email must be provided`

---

## Test 8: Set Role (Promote to Admin)

**Objective:** Verify role can be changed

```bash
# Get an editor user ID
basil users list

# Promote to admin
basil users set-role <EDITOR_USER_ID> admin
```

**Expected:**
- [ ] Output: `✓ Set role for <Name> to admin`
- [ ] `basil users list` shows updated role

---

## Test 9: Set Role (Demote to Editor)

**Objective:** Verify admin can be demoted (if not last admin)

```bash
# Ensure you have 2+ admins first
basil users set-role <ADMIN_USER_ID> editor
```

**Expected:**
- [ ] Output: `✓ Set role for <Name> to editor`
- [ ] `basil users list` shows updated role

---

## Test 10: Cannot Remove Last Admin (set-role)

**Objective:** Verify last admin protection

```bash
# Ensure only one admin exists (demote others first)
basil users set-role <LAST_ADMIN_ID> editor
```

**Expected:**
- [ ] Error: `cannot remove the last admin user`
- [ ] Role unchanged

---

## Test 11: User Delete with Confirmation

**Objective:** Verify delete prompts for confirmation

```bash
# Create a throwaway user
basil users create --name "Delete Me" --email "delete@example.com"

# Try to delete (answer 'n')
basil users delete <USER_ID>
```

**Expected:**
- [ ] Prompt: `⚠ This will delete user Delete Me and all their credentials. Continue? [y/N]`
- [ ] Entering `n` or nothing cancels deletion
- [ ] Output: `Cancelled.`

---

## Test 12: User Delete with --force

**Objective:** Verify --force skips confirmation

```bash
basil users delete <USER_ID> --force
```

**Expected:**
- [ ] No prompt
- [ ] Output: `✓ Deleted user usr_...`
- [ ] User no longer in `basil users list`

---

## Test 13: Cannot Delete Last Admin

**Objective:** Verify last admin can't be deleted

```bash
# Ensure only one admin exists
basil users delete <LAST_ADMIN_ID> --force
```

**Expected:**
- [ ] Error: `cannot delete the last admin user`
- [ ] User still exists

---

## Test 14: API Key Creation

**Objective:** Verify API key creation and format

```bash
basil apikey create --user <USER_ID> --name "Test Key"
```

**Expected:**
- [ ] Output shows: `✓ Created API key: bsl_live_...`
- [ ] Key ID shown: `Key ID: key_...`
- [ ] Warning: `(save this now — it won't be shown again)`
- [ ] Plaintext key starts with `bsl_live_`

**Save the plaintext key for later tests!**

---

## Test 15: API Key List

**Objective:** Verify API key listing shows prefix only

```bash
basil apikey list --user <USER_ID>
```

**Expected:**
- [ ] Table shows ID, NAME, PREFIX, CREATED, LAST USED
- [ ] PREFIX shows truncated key (e.g., `bsl_live_abc...xyz`)
- [ ] LAST USED shows `never` for new key
- [ ] Full plaintext key is NOT shown

---

## Test 16: API Key for Non-existent User

**Objective:** Verify error for invalid user ID

```bash
basil apikey create --user usr_nonexistent --name "Bad Key"
```

**Expected:**
- [ ] Error message about user not found

---

## Test 17: API Key Revoke

**Objective:** Verify API key can be revoked

```bash
# Get key ID from list
basil apikey list --user <USER_ID>

basil apikey revoke <KEY_ID>
```

**Expected:**
- [ ] Output: `✓ Revoked API key "Test Key"`
- [ ] Key no longer appears in `basil apikey list`

---

## Test 18: Revoke Non-existent Key

**Objective:** Verify error for invalid key ID

```bash
basil apikey revoke key_nonexistent
```

**Expected:**
- [ ] Error: `API key not found: key_nonexistent`

---

## Test 19: User Deletion Cascades to API Keys

**Objective:** Verify deleting user removes their API keys

```bash
# Create user and API key
basil users create --name "Cascade Test" --email "cascade@example.com"
basil apikey create --user <NEW_USER_ID> --name "Will Be Deleted"
basil apikey list --user <NEW_USER_ID>  # Verify key exists

# Delete user
basil users delete <NEW_USER_ID> --force

# Try to list keys (should show none or error gracefully)
basil apikey list --user <NEW_USER_ID>
```

**Expected:**
- [ ] Key exists before user deletion
- [ ] After user deletion, key is gone
- [ ] No orphaned keys in database

---

## Test 20: Concurrent CLI and Server Access

**Objective:** Verify WAL mode allows concurrent access

**Terminal 1:**
```bash
# Start the server
basil --dev
```

**Terminal 2 (while server is running):**
```bash
basil users create --name "Concurrent Test" --email "concurrent@example.com"
basil users list
basil apikey create --user <USER_ID> --name "Concurrent Key"
```

**Expected:**
- [ ] Server continues running without interruption
- [ ] CLI commands succeed without "database is locked" errors
- [ ] User and API key are created successfully

---

## Test 21: Passkey Registration for CLI-Created User

**Objective:** Verify CLI-created users can register passkeys via web

**Setup:**
```bash
# Create user without passkey
basil users create --name "Web User" --email "webuser@example.com"
```

**In Browser:**
1. Navigate to your registration page (e.g., `http://localhost:8080/register`)
2. Enter email: `webuser@example.com`
3. Enter name: `Web User` (or any name)
4. Click register

**Expected:**
- [ ] Registration flow begins (browser prompts for passkey/biometric)
- [ ] Does NOT show "Email already registered" error
- [ ] After completing registration, user can log in via web

**Verify:**
```bash
basil users show <USER_ID>
```

**Expected:**
- [ ] Passkeys count is now `1` (was `0`)

---

## Test 22: Passkey Registration Blocked for User WITH Passkey

**Objective:** Verify users who already have passkeys can't re-register

**In Browser:**
1. Log out if logged in
2. Navigate to registration page
3. Enter email of user who already registered via web (has passkey)

**Expected:**
- [ ] Error: "Email already registered"
- [ ] Registration does not proceed

---

## Test 23: Users Show Command

**Objective:** Verify detailed user view

```bash
basil users show <USER_ID>
```

**Expected output includes:**
- [ ] User ID
- [ ] Name
- [ ] Email (or "(none)")
- [ ] Role
- [ ] Created timestamp
- [ ] Passkeys count
- [ ] Recovery codes remaining
- [ ] API Keys count

---

## Test 24: Users Reset Command (Recovery Codes)

**Objective:** Verify recovery code regeneration

```bash
basil users reset <USER_ID>
```

**Expected:**
- [ ] Shows 8 new recovery codes
- [ ] Warning to save codes
- [ ] Old codes are invalidated (if any existed)

---

## Summary Checklist

| Test | Pass | Notes |
|------|------|-------|
| 1. First user is admin | ☐ | |
| 2. First user role override ignored | ☐ | |
| 3. Default editor role | ☐ | |
| 4. Explicit admin role | ☐ | |
| 5. Invalid role rejected | ☐ | |
| 6. User update | ☐ | |
| 7. Update validation | ☐ | |
| 8. Promote to admin | ☐ | |
| 9. Demote to editor | ☐ | |
| 10. Last admin protection (set-role) | ☐ | |
| 11. Delete with confirmation | ☐ | |
| 12. Delete with --force | ☐ | |
| 13. Last admin protection (delete) | ☐ | |
| 14. API key creation | ☐ | |
| 15. API key list | ☐ | |
| 16. API key invalid user | ☐ | |
| 17. API key revoke | ☐ | |
| 18. Revoke invalid key | ☐ | |
| 19. Cascade delete | ☐ | |
| 20. Concurrent access | ☐ | |
| 21. Passkey for CLI user | ☐ | |
| 22. Passkey blocked if exists | ☐ | |
| 23. Users show | ☐ | |
| 24. Users reset | ☐ | |

---

## Clean Up

After testing:
```bash
rm -f .basil-auth.db
```
