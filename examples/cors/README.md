# CORS Example

This example demonstrates CORS (Cross-Origin Resource Sharing) configuration in Basil.

## Configuration

The `basil.yaml` file configures CORS to allow requests from local development servers:

```yaml
cors:
  origins:
    - http://localhost:3000
    - http://localhost:5173
  methods: [GET, POST, PUT, DELETE]
  headers: [Content-Type, Authorization]
  expose: [X-Total-Count]
  credentials: true
  maxAge: 86400
```

## Testing

### Start the server

```bash
cd examples/cors
basil --dev
```

### Test with curl

#### Same-origin request (no CORS headers)
```bash
curl http://localhost:8080/api/test -v
```

#### Cross-origin request
```bash
curl http://localhost:8080/api/test \
  -H "Origin: http://localhost:3000" \
  -v
```

Should see CORS headers:
- `Access-Control-Allow-Origin: http://localhost:3000`
- `Access-Control-Allow-Credentials: true`
- `Access-Control-Expose-Headers: X-Total-Count`
- `Vary: Origin`

#### Disallowed origin
```bash
curl http://localhost:8080/api/test \
  -H "Origin: http://evil.com" \
  -v
```

No CORS headers (browser would block).

#### Preflight request
```bash
curl -X OPTIONS http://localhost:8080/api/test \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: DELETE" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v
```

Should return 204 with:
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE`
- `Access-Control-Allow-Headers: Content-Type, Authorization`
- `Access-Control-Max-Age: 86400`

### Test with a frontend

Create `test.html`:
```html
<!DOCTYPE html>
<html>
<body>
  <h1>CORS Test</h1>
  <button onclick="test()">Test API</button>
  <pre id="result"></pre>
  
  <script>
    async function test() {
      try {
        const response = await fetch('http://localhost:8080/api/test', {
          method: 'GET',
          credentials: 'include'
        });
        const data = await response.json();
        const totalCount = response.headers.get('X-Total-Count');
        document.getElementById('result').textContent = 
          JSON.stringify({data, totalCount}, null, 2);
      } catch (err) {
        document.getElementById('result').textContent = 'Error: ' + err.message;
      }
    }
  </script>
</body>
</html>
```

Serve it on a different port:
```bash
python3 -m http.server 3000
```

Open http://localhost:3000/test.html and click "Test API".

## Configuration Options

### origins
Single origin or list:
```yaml
# Single origin
origins: "https://app.example.com"

# Multiple origins
origins:
  - https://app.example.com
  - https://staging.example.com

# Wildcard (allows any origin, but cannot use with credentials)
origins: "*"
```

### credentials
Allow cookies and Authorization headers:
```yaml
credentials: true  # Requires specific origins (not *)
```

### methods
HTTP methods to allow (default: GET, HEAD, POST):
```yaml
methods: [GET, POST, PUT, PATCH, DELETE]
```

### headers
Request headers to allow:
```yaml
headers: [Content-Type, Authorization, X-Custom-Header]
```

### expose
Response headers accessible to JavaScript:
```yaml
expose: [X-Total-Count, X-Page-Count, Link]
```

### maxAge
Preflight cache duration in seconds:
```yaml
maxAge: 86400  # 24 hours
```
