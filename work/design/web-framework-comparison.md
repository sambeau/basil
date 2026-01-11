# Web Framework Feature Analysis

**Date:** 2025-12-07  
**Purpose:** Compare Basil's handler environment against established web frameworks to identify gaps and prioritize improvements.

## Frameworks Compared

| Framework | Language | Style | Notes |
|-----------|----------|-------|-------|
| **PHP** | PHP | Traditional server-side | Built-in superglobals, minimal framework |
| **Express** | Node.js | Minimal, middleware-based | Most popular Node framework |
| **Sinatra** | Ruby | Minimal DSL | Inspiration for Flask, Express |
| **Rails** | Ruby | Full-stack, convention-over-config | Kitchen sink approach |
| **Basil** | Parsley | Minimal, integrated server | Single binary, batteries included |

---

## Feature Comparison Matrix

### Request Handling

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| HTTP method | `$_SERVER['REQUEST_METHOD']` | `req.method` | `request.request_method` | `request.method` | ✅ `basil.http.request.method` | |
| URL path | `$_SERVER['REQUEST_URI']` | `req.path` | `request.path_info` | `request.path` | ✅ `basil.http.request.path` | |
| Query string | `$_GET` | `req.query` | `params` (merged) | `request.query_parameters` | ✅ `basil.http.request.query` | |
| Headers | `$_SERVER['HTTP_*']` | `req.headers` | `request.env` | `request.headers` | ✅ `basil.http.request.headers` | |
| Raw body | `php://input` | `req.body` (middleware) | `request.body.read` | `request.raw_post` | ✅ `basil.http.request.body` | |
| Form data | `$_POST` | `req.body` (middleware) | `params` | `params` | ✅ `basil.http.request.form` | |
| File uploads | `$_FILES` | `req.files` (multer) | `params[:file]` | `params[:file]` | ✅ `basil.http.request.files` | |
| **Cookies** | `$_COOKIE` | `req.cookies` | `request.cookies` | `cookies` | ✅ `basil.http.request.cookies` | FEAT-043 |
| Client IP | `$_SERVER['REMOTE_ADDR']` | `req.ip` | `request.ip` | `request.remote_ip` | ✅ `basil.http.request.remoteAddr` | |
| Host | `$_SERVER['HTTP_HOST']` | `req.hostname` | `request.host` | `request.host` | ✅ `basil.http.request.host` | |

### Response Control

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| Status code | `http_response_code()` | `res.status()` | `status 404` | `head :not_found` | ✅ `basil.http.response.status` | |
| Headers | `header()` | `res.set()` | `headers[]` | `response.headers[]` | ✅ `basil.http.response.headers` | |
| **Set cookies** | `setcookie()` | `res.cookie()` | `response.set_cookie()` | `cookies[]` | ✅ `basil.http.response.cookies` | FEAT-043 |
| **Redirect** | `header('Location:')` | `res.redirect()` | `redirect()` | `redirect_to()` | ⚠️ **Manual** | FEAT-045 |
| Send file | `readfile()` | `res.sendFile()` | `send_file()` | `send_file()` | ⚠️ Via file I/O | |
| JSON response | `json_encode()` | `res.json()` | `json()` | `render json:` | ✅ Auto-detected | |

### Routing

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| Static routes | Manual/.htaccess | `app.get('/path')` | `get '/path'` | `get '/path'` | ✅ Config routes | |
| **Path params** | Manual | `/users/:id` | `/users/:id` | `/users/:id` | ⚠️ **Wildcards only** | FEAT-046 |
| Regex routes | Manual | ✅ | ✅ | ✅ | ❌ | |
| Named routes | ❌ | ❌ | ✅ | ✅ `users_path` | ❌ | Low priority |
| Route groups | ❌ | `Router()` | ❌ | `namespace` | ❌ | |

### Sessions & State

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| **Session store** | `$_SESSION` | `req.session` | `session` | `session` | ❌ **Missing** | Auth has sessions internally |
| Flash messages | ❌ | `req.flash` | `flash` | `flash` | ❌ | Nice to have |
| **CSRF tokens** | Manual | `csurf` | `Rack::Csrf` | Built-in | ❌ **Missing** | FEAT-044 |

### Security

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| XSS escaping | `htmlspecialchars()` | Manual | Manual | Auto in ERB | ✅ Auto in tags | |
| SQL injection | Prepared statements | Prepared | Prepared | ActiveRecord | ✅ Prepared statements | |
| **CSRF protection** | Manual | `csurf` middleware | `Rack::Csrf` | Built-in | ❌ **Missing** | FEAT-044 |
| CORS | Manual | `cors` middleware | Manual | `rack-cors` | ❌ Manual headers | Low priority |
| Rate limiting | Manual | `express-rate-limit` | Manual | `rack-attack` | ✅ Built-in | |
| Security headers | Manual | `helmet` | Manual | `secure_headers` | ✅ Built-in | |

### Database

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| Raw SQL | PDO | pg/mysql | Sequel | ActiveRecord | ✅ `<=?=>` operators | |
| Query builder | ❌ | Knex | Sequel | ActiveRecord | ⚠️ Table type | |
| ORM | Eloquent | Sequelize/Prisma | ActiveRecord | ActiveRecord | ❌ | Out of scope |
| Migrations | ❌ | Knex | Sequel | Built-in | ❌ | Out of scope |
| Connection pool | ❌ | Built-in | Built-in | Built-in | ✅ Managed | |

### Developer Experience

| Feature | PHP | Express | Sinatra | Rails | Basil | Notes |
|---------|-----|---------|---------|-------|-------|-------|
| Hot reload | ✅ (always) | nodemon | rerun | Built-in | ✅ `--dev` | |
| Error pages | ❌ | ❌ | ✅ | ✅ | ✅ Dev mode | |
| Logging | error_log | morgan | logger | Rails.logger | ✅ Structured | |
| **Request ID** | ❌ | `uuid` | ❌ | `X-Request-Id` | ❌ | Nice to have |

---

## Gap Analysis

### Critical Gaps (Security/Functionality)

#### 1. Cookie Support (FEAT-043)

**What's missing:**  
No way to read or write HTTP cookies from Parsley handlers.

**Why it matters:**
- Can't implement "remember me" functionality
- Can't store user preferences (theme, language, timezone)
- Can't implement tracking consent (GDPR compliance)
- Can't implement CSRF protection (needs cookie to store token)
- Can't implement custom session-like features

**What other frameworks provide:**

```php
// PHP
$theme = $_COOKIE['theme'];
setcookie('theme', 'dark', time() + 86400 * 30, '/');
```

```javascript
// Express
const theme = req.cookies.theme;
res.cookie('theme', 'dark', { maxAge: 30 * 24 * 60 * 60 * 1000 });
```

```ruby
# Rails
theme = cookies[:theme]
cookies[:theme] = { value: 'dark', expires: 30.days }
```

**Proposed for Basil:**
```parsley
// Read
let theme = basil.http.request.cookies.theme ?? "light"

// Write (simple)
basil.http.response.cookies.theme = "dark"

// Write (with options)
basil.http.response.cookies.remember = {
    value: token,
    maxAge: @30d,
    httpOnly: true,
    secure: true,
    sameSite: "Strict"
}
```

**Implementation complexity:** Medium  
**Priority:** High — blocks CSRF protection

---

#### 2. CSRF Protection (FEAT-044)

**What's missing:**  
No protection against Cross-Site Request Forgery attacks on forms.

**Why it matters:**
- Malicious sites can trick users into submitting forms to your site
- Attacker can perform actions as the logged-in user
- Common attack vector for state-changing operations
- Required for any production form handling

**Attack scenario:**
```html
<!-- On evil-site.com -->
<form action="https://your-site.com/transfer" method="POST">
    <input type="hidden" name="to" value="attacker">
    <input type="hidden" name="amount" value="1000">
</form>
<script>document.forms[0].submit()</script>
```
If user is logged into your-site.com, this form submits with their session cookie.

**What other frameworks provide:**

```ruby
# Rails - automatic for all forms
<%= form_with do |f| %>
  # Automatically includes: <input type="hidden" name="authenticity_token" value="...">
<% end %>
```

```javascript
// Express with csurf
app.use(csrf({ cookie: true }));
// In template: <input type="hidden" name="_csrf" value="<%= csrfToken %>">
```

**Proposed for Basil:**
```parsley
<form method=POST>
    <input type=hidden name=_csrf value={basil.csrf.token}/>
    <input type=text name=email/>
    <button>Submit</button>
</form>
```

Auto-validation on POST/PUT/PATCH/DELETE for routes with `auth: required` or `auth: optional`.

**Implementation complexity:** Medium  
**Priority:** High — critical for secure forms  
**Depends on:** FEAT-043 (Cookies)

---

### Ergonomic Gaps (Developer Experience)

#### 3. Redirect Helper (FEAT-045)

**What's missing:**  
No simple way to redirect users. Current approach is verbose and error-prone.

**Why it matters:**
- Redirects are extremely common (after form submission, auth flows, etc.)
- Current approach requires 3 lines and knowing HTTP details
- Easy to forget the empty body return
- Easy to use wrong status code

**Current Basil (verbose):**
```parsley
basil.http.response.status = 302
basil.http.response.headers.Location = "/dashboard"
""  // Must return empty body!
```

**What other frameworks provide:**

```php
// PHP
header('Location: /dashboard');
exit;  // Must remember to exit!
```

```javascript
// Express
res.redirect('/dashboard');
res.redirect(301, '/new-location');  // Permanent
```

```ruby
# Rails
redirect_to dashboard_path
redirect_to '/new-location', status: :moved_permanently
```

**Proposed for Basil:**
```parsley
redirect("/dashboard")           // 302 Found (default)
redirect("/new-url", 301)        // 301 Moved Permanently
redirect(@/users/{id}/profile)   // With path literal
```

**Implementation complexity:** Low  
**Priority:** Medium — quality of life improvement

---

#### 4. Path Pattern Matching (FEAT-046)

**What's missing:**  
No way to extract named parameters from URL paths. Must manually parse.

**Why it matters:**
- RESTful APIs need `/users/:id/posts/:postId` patterns
- Manual string parsing is tedious and error-prone
- Config-based routing requires access to basil.yaml
- Site mode's `subpath.segments` is positional, not named

**Current Basil (manual parsing):**
```parsley
let segments = basil.http.request.path.split("/")
// /users/42/posts/99 → ["", "users", "42", "posts", "99"]
if (segments[1] == "users" && segments[3] == "posts") {
    let userId = segments[2]
    let postId = segments[4]
    // ... handle request
}
```

**What other frameworks provide:**

```javascript
// Express
app.get('/users/:userId/posts/:postId', (req, res) => {
    const { userId, postId } = req.params;
});
```

```ruby
# Sinatra
get '/users/:user_id/posts/:post_id' do
    user_id = params[:user_id]
    post_id = params[:post_id]
end
```

**Proposed for Basil:**
```parsley
let params = match(basil.http.request.path, "/users/:userId/posts/:postId")
if (params) {
    let {userId, postId} = params
    // ... handle request
}

// Glob capture
let params = match(path, "/files/*path")
// /files/docs/2025/report.pdf → {path: ["docs", "2025", "report.pdf"]}
```

**Why function over config:**
- No config file access needed for developers
- Can match multiple patterns in one handler
- Works with site mode subpaths
- More flexible (can combine with method checks, query params)
- Testable in isolation

**Implementation complexity:** Low-Medium  
**Priority:** Medium — ergonomic improvement

---

### Nice-to-Have Gaps (Lower Priority)

#### 5. General Session Store

**What's missing:**  
Auth system has internal sessions, but no general-purpose session storage for arbitrary data.

**Use cases:**
- Shopping cart (before login)
- Multi-step form wizards
- Flash messages ("Item saved successfully")
- Temporary user preferences

**Current workarounds:**
- Use cookies directly (limited to 4KB)
- Require login and store in database
- Pass data through URL params or hidden fields

**What Rails provides:**
```ruby
session[:cart] = [item1, item2]
flash[:notice] = "Saved successfully"
```

**Implementation complexity:** Medium-High (needs storage backend)  
**Priority:** Low — can work around with cookies or database

---

#### 6. Request ID / Tracing

**What's missing:**  
No unique identifier per request for logging and debugging.

**Use cases:**
- Correlate log entries across a request
- Debug production issues ("what happened for request abc123?")
- Pass to external services for distributed tracing

**What Rails provides:**
```ruby
# Automatic X-Request-Id header
Rails.logger.info "Processing request #{request.request_id}"
```

**Implementation complexity:** Low  
**Priority:** Low — nice for production debugging

---

#### 7. CORS Configuration

**What's missing:**  
No built-in CORS header management.

**Current workaround:**
```parsley
basil.http.response.headers["Access-Control-Allow-Origin"] = "*"
basil.http.response.headers["Access-Control-Allow-Methods"] = "GET, POST"
// ... tedious for complex CORS
```

**Implementation complexity:** Medium  
**Priority:** Low — manual headers work, most apps don't need complex CORS

---

#### 8. Send File Helper

**What's missing:**  
No optimized file sending with proper headers.

**Current approach:**
```parsley
let content <== bytes(@./files/document.pdf)
basil.http.response.headers["Content-Type"] = "application/pdf"
basil.http.response.headers["Content-Disposition"] = "attachment; filename=doc.pdf"
content
```

**What Express provides:**
```javascript
res.download('/path/to/file.pdf', 'document.pdf');
res.sendFile('/path/to/file.pdf');
```

**Implementation complexity:** Medium  
**Priority:** Low — current approach works, static routes handle most cases

---

## Implementation Roadmap

### Phase 1: Security Foundation
1. **FEAT-043: Cookie Support** — Required for CSRF
2. **FEAT-044: CSRF Protection** — Required for secure forms

### Phase 2: Developer Ergonomics
3. **FEAT-045: Redirect Helper** — Quick win, very common operation
4. **FEAT-046: Path Pattern Matching** — Enables clean RESTful handlers

### Phase 3: Nice-to-Have (Future)
5. General session store
6. Request ID
7. CORS helper
8. Send file helper

---

## Conclusion

Basil has solid fundamentals for request/response handling, but is missing a few critical features for production web applications:

**Must have before production use:**
- Cookie support (can't do stateful web apps without it)
- CSRF protection (security requirement for forms)

**Should have for good DX:**
- Redirect helper (very common operation)
- Path pattern matching (cleaner route handling)

The good news is these are all well-understood features with clear implementations. The dependency chain is:

```
Cookies (FEAT-043)
    └── CSRF (FEAT-044)

Redirect (FEAT-045)     [independent]
Path Matching (FEAT-046) [independent]
```

Cookies should be implemented first since CSRF depends on it. Redirect and path matching can be done in parallel or any order.
