# CORS (Cross-Origin Resource Sharing)

CORS allows your Basil APIs to be called from JavaScript running on different domains. This is essential when your frontend and backend are hosted separately or during local development.

## Table of Contents

- [Why CORS?](#why-cors)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Common Patterns](#common-patterns)
- [How CORS Works](#how-cors-works)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

---

## Why CORS?

Web browsers block JavaScript from making requests to different origins (domain, protocol, or port) unless the server explicitly allows it. This is called the **Same-Origin Policy**.

**Without CORS:**
```
Frontend: http://localhost:3000
Backend:  http://localhost:8080
Browser:  ❌ BLOCKED
```

**With CORS:**
```
Frontend: http://localhost:3000
Backend:  http://localhost:8080 (with CORS headers)
Browser:  ✅ ALLOWED
```

---

## Quick Start

### Development Setup

Allow requests from your local development server:

**basil.yaml:**
```yaml
server:
  port: 8080

cors:
  origins: http://localhost:3000
  credentials: true

routes:
  - path: /api/users
    handler: ./handlers/users.pars
```

### Production Setup

Allow requests from your production frontend:

```yaml
cors:
  origins: https://app.example.com
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  credentials: true
  maxAge: 86400  # Cache preflight for 24 hours
```

---

## Configuration

### `origins`

**Required.** Which domains can make requests to your API.

```yaml
# Single origin
cors:
  origins: https://app.example.com

# Multiple origins
cors:
  origins:
    - https://app.example.com
    - https://staging.example.com

# Wildcard (public API, no credentials)
cors:
  origins: "*"
```

**Important:** Cannot use `origins: "*"` with `credentials: true` (browsers reject this).

### `methods`

Which HTTP methods are allowed. Default: `[GET, HEAD, POST]`

```yaml
cors:
  methods: [GET, POST, PUT, PATCH, DELETE]
```

### `headers`

Which request headers are allowed. If not specified, Basil echoes back the headers requested by the browser.

```yaml
cors:
  headers: 
    - Content-Type
    - Authorization
    - X-Custom-Header
```

### `expose`

Which response headers JavaScript can access. By default, browsers only expose simple headers like `Content-Type`.

```yaml
cors:
  expose:
    - X-Total-Count
    - X-Page-Count
    - Link
```

Example usage in JavaScript:
```javascript
const response = await fetch('/api/items');
const totalCount = response.headers.get('X-Total-Count'); // Now accessible!
```

### `credentials`

Allow cookies and `Authorization` headers in cross-origin requests. Default: `false`

```yaml
cors:
  credentials: true
```

When enabled:
- Browser sends cookies with requests
- `Authorization` header is sent
- Response must use specific origin (not `*`)

JavaScript must also set `credentials`:
```javascript
fetch('http://localhost:8080/api/data', {
  credentials: 'include'  // Send cookies
})
```

### `maxAge`

How long (in seconds) browsers cache preflight responses. Default: `86400` (24 hours)

```yaml
cors:
  maxAge: 3600  # 1 hour
```

Longer cache = fewer preflight requests = better performance.

---

## Common Patterns

### Pattern 1: Local Development

Allow your local frontend to call your local API:

```yaml
cors:
  origins:
    - http://localhost:3000      # React/Vue dev server
    - http://localhost:5173      # Vite
    - http://127.0.0.1:3000      # Alternative localhost
  credentials: true
```

### Pattern 2: Production with Authenticated API

```yaml
cors:
  origins: https://app.example.com
  methods: [GET, POST, PUT, PATCH, DELETE]
  headers: [Content-Type, Authorization]
  credentials: true
  maxAge: 86400
```

### Pattern 3: Public Read-Only API

```yaml
cors:
  origins: "*"
  methods: [GET]
  # No credentials with wildcard
```

### Pattern 4: Multiple Frontends

```yaml
cors:
  origins:
    - https://app.example.com
    - https://admin.example.com
    - https://mobile.example.com
  credentials: true
```

### Pattern 5: Pagination Headers

Expose pagination metadata to JavaScript:

```yaml
cors:
  origins: https://app.example.com
  expose: [X-Total-Count, X-Page, X-Per-Page, Link]
  credentials: true
```

In your handler:
```parsley
let {basil} = import @std/basil

basil.http.response.headers["X-Total-Count"] = "150"
basil.http.response.headers["X-Page"] = "2"
basil.http.response.headers["X-Per-Page"] = "20"
```

JavaScript:
```javascript
const response = await fetch('/api/items?page=2');
const total = response.headers.get('X-Total-Count'); // "150"
```

---

## How CORS Works

### Simple Requests

For simple requests (GET, POST with basic content types), the browser:

1. Sends the request with an `Origin` header
2. Server responds with CORS headers
3. Browser checks if origin is allowed

**Request:**
```http
GET /api/data HTTP/1.1
Host: localhost:8080
Origin: http://localhost:3000
```

**Response:**
```http
HTTP/1.1 200 OK
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Credentials: true
Content-Type: application/json

{"data": "..."}
```

### Preflight Requests

For "non-simple" requests (PUT, DELETE, custom headers), the browser first sends an OPTIONS request to check if the actual request is allowed.

**Preflight (OPTIONS):**
```http
OPTIONS /api/data HTTP/1.1
Host: localhost:8080
Origin: http://localhost:3000
Access-Control-Request-Method: DELETE
Access-Control-Request-Headers: Authorization
```

**Preflight Response:**
```http
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400
Access-Control-Allow-Credentials: true
```

**Actual Request:**
```http
DELETE /api/data/123 HTTP/1.1
Host: localhost:8080
Origin: http://localhost:3000
Authorization: Bearer token123
```

Basil handles preflight requests automatically based on your configuration.

---

## Security Considerations

### Never Use `*` with Credentials

❌ **Wrong:**
```yaml
cors:
  origins: "*"
  credentials: true  # Browsers reject this!
```

✅ **Correct:**
```yaml
cors:
  origins: https://app.example.com
  credentials: true
```

### Validate Origins Carefully

Only allow origins you control:

```yaml
cors:
  origins:
    - https://app.example.com
    - https://staging.example.com
  # Don't add untrusted domains!
```

### Use HTTPS in Production

```yaml
cors:
  # ❌ Don't use http:// in production
  origins: http://app.example.com
  
  # ✅ Use https://
  origins: https://app.example.com
```

### Limit Exposed Headers

Only expose headers that are safe for JavaScript to access:

```yaml
cors:
  expose: [X-Total-Count]
  # Don't expose: Authorization, Set-Cookie, etc.
```

### Restrict Methods

Only allow methods your API actually uses:

```yaml
cors:
  methods: [GET, POST, PUT, DELETE]
  # Don't add TRACE, CONNECT, etc.
```

---

## Troubleshooting

### CORS error in browser console

**Error:**
```
Access to fetch at 'http://localhost:8080/api/data' from origin 
'http://localhost:3000' has been blocked by CORS policy
```

**Solutions:**

1. **Check your origin is configured:**
   ```yaml
   cors:
     origins: http://localhost:3000  # Must match exactly!
   ```

2. **Check credentials setting:**
   ```javascript
   // JavaScript
   fetch('http://localhost:8080/api/data', {
     credentials: 'include'  // Needed if server has credentials: true
   })
   ```

3. **Verify server is running:**
   ```bash
   curl -H "Origin: http://localhost:3000" http://localhost:8080/api/data -v
   ```
   Should see `Access-Control-Allow-Origin` header.

### Preflight request fails

**Error:**
```
Response to preflight request doesn't pass access control check
```

**Solutions:**

1. **Add custom headers to config:**
   ```yaml
   cors:
     headers: [Content-Type, Authorization, X-Custom-Header]
   ```

2. **Add HTTP method:**
   ```yaml
   cors:
     methods: [GET, POST, PUT, DELETE, PATCH]
   ```

3. **Check request details:**
   ```bash
   curl -X OPTIONS http://localhost:8080/api/data \
     -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: PUT" \
     -H "Access-Control-Request-Headers: Authorization" \
     -v
   ```

### Credentials not working

**Error:**
```
The value of the 'Access-Control-Allow-Origin' header must not be '*' 
when the credentials flag is true
```

**Solution:**

Use specific origin, not wildcard:
```yaml
cors:
  origins: http://localhost:3000  # Not "*"
  credentials: true
```

### Headers not accessible in JavaScript

**Error:**
```javascript
response.headers.get('X-Total-Count') // null
```

**Solution:**

Expose the header:
```yaml
cors:
  expose: [X-Total-Count]
```

### Different port = different origin

Remember:
- `http://localhost:3000` ≠ `http://localhost:8080`
- `http://example.com` ≠ `https://example.com`
- `http://example.com` ≠ `http://www.example.com`

Each needs to be explicitly allowed.

---

## Testing CORS

### Test with curl

**Simple request:**
```bash
curl -H "Origin: http://localhost:3000" \
     http://localhost:8080/api/data \
     -v
```

Look for: `Access-Control-Allow-Origin: http://localhost:3000`

**Preflight:**
```bash
curl -X OPTIONS \
     -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: DELETE" \
     -H "Access-Control-Request-Headers: Authorization" \
     http://localhost:8080/api/data \
     -v
```

Look for: 
- `204 No Content`
- `Access-Control-Allow-Methods`
- `Access-Control-Allow-Headers`

### Test with JavaScript

Create `test.html`:
```html
<!DOCTYPE html>
<html>
<body>
  <button onclick="testCORS()">Test CORS</button>
  <pre id="result"></pre>
  
  <script>
    async function testCORS() {
      try {
        const response = await fetch('http://localhost:8080/api/data', {
          method: 'GET',
          credentials: 'include'
        });
        const data = await response.json();
        document.getElementById('result').textContent = 
          JSON.stringify(data, null, 2);
      } catch (err) {
        document.getElementById('result').textContent = 
          'Error: ' + err.message;
      }
    }
  </script>
</body>
</html>
```

Serve on different port:
```bash
python3 -m http.server 3000
```

Open http://localhost:3000/test.html and click the button.

---

## Example: Complete Setup

**basil.yaml:**
```yaml
server:
  port: 8080

cors:
  origins:
    - http://localhost:3000
    - https://app.example.com
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  expose: [X-Total-Count, X-Page]
  credentials: true
  maxAge: 86400

routes:
  - path: /api/items
    handler: ./handlers/items.pars
```

**handlers/items.pars:**
```parsley
let {basil} = import @std/basil

basil.http.response.headers["Content-Type"] = "application/json"
basil.http.response.headers["X-Total-Count"] = "100"

{
  items: [
    {id: 1, name: "Item 1"},
    {id: 2, name: "Item 2"}
  ]
}.toJSON()
```

**Frontend (JavaScript):**
```javascript
async function getItems() {
  const response = await fetch('http://localhost:8080/api/items', {
    credentials: 'include'
  });
  
  const data = await response.json();
  const totalCount = response.headers.get('X-Total-Count');
  
  console.log('Items:', data.items);
  console.log('Total:', totalCount);
}
```

---

## Related Documentation

- [API Development Guide](./api-table-binding.md)
- [Authentication](./authentication.md)
- [Security Best Practices](#) (coming soon)

## See Also

- [MDN: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [CORS Specification](https://fetch.spec.whatwg.org/#http-cors-protocol)
