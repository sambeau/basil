# Testing Basil Applications

## Testing with curl

### GET Requests

```bash
# Basic GET
curl http://localhost:8080/

# With query parameters
curl "http://localhost:8080/search?q=test&page=2"

# With headers
curl http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123"

# Verbose output (see request/response headers)
curl -v http://localhost:8080/

# Follow redirects
curl -L http://localhost:8080/old-path
```

### POST Requests

```bash
# Form data (application/x-www-form-urlencoded)
curl -X POST http://localhost:8080/contact \
  -d "name=Alice" \
  -d "email=alice@example.com" \
  -d "message=Hello"

# JSON data
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# File upload (multipart/form-data)
curl -X POST http://localhost:8080/upload \
  -F "file=@./photo.jpg" \
  -F "title=My Photo"
```

### Cookies & Sessions

```bash
# Save cookies to file
curl -c cookies.txt http://localhost:8080/login \
  -d "username=admin" \
  -d "password=secret"

# Use saved cookies
curl -b cookies.txt http://localhost:8080/dashboard

# Send specific cookie
curl http://localhost:8080/profile \
  -H "Cookie: session=abc123; theme=dark"

# Show response headers (to see Set-Cookie)
curl -i http://localhost:8080/login -d "user=admin&pass=secret"
```

### Other HTTP Methods

```bash
# PUT
curl -X PUT http://localhost:8080/api/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated"}'

# DELETE
curl -X DELETE http://localhost:8080/api/users/123

# PATCH
curl -X PATCH http://localhost:8080/api/users/123 \
  -H "Content-Type: application/json" \
  -d '{"active":false}'
```

### Debugging

```bash
# Verbose output (see full request/response)
curl -v http://localhost:8080/

# Show only response headers
curl -I http://localhost:8080/

# Show response headers with body
curl -i http://localhost:8080/

# Trace request (very verbose)
curl --trace - http://localhost:8080/

# Time the request
curl -w "\nTime: %{time_total}s\n" http://localhost:8080/

# Save response to file
curl -o response.html http://localhost:8080/
```

## Testing in Browser

### 1. Start Dev Server

```bash
./basil --dev
```

### 2. Visit URLs

- Homepage: http://localhost:8080/
- Dev tools: http://localhost:8080/__dev/log
- Specific route: http://localhost:8080/users/123

### 3. Use Browser DevTools

**Network Tab:**
- See all requests (including Part updates)
- Inspect request/response headers
- View form data and JSON payloads
- Check status codes

**Console:**
- Run JavaScript
- See console.log output
- Check for errors

**Application Tab:**
- Inspect cookies
- View localStorage/sessionStorage

## Testing Parts

### Test Part Directly

```bash
# Visit part URL with props as query params
curl "http://localhost:8080/parts/counter.part?count=5"

# Trigger part action
curl "http://localhost:8080/parts/counter.part?count=5&__action=increment"
```

### Test in Browser

1. Load page with part
2. Open Network tab
3. Click part button
4. See part HTTP request
5. Inspect response HTML

### Debug Part Issues

```bash
# Check if part is routed
curl -I http://localhost:8080/parts/counter.part
# Should return 200, not 404

# Check part renders
curl http://localhost:8080/parts/counter.part
# Should return HTML

# Check with props
curl "http://localhost:8080/parts/counter.part?count=10"
# Should render with count=10
```

## Testing Databases

### Setup Test Database

```yaml
# basil-test.yaml
sqlite: ./test.db
```

```bash
# Run with test config
./basil --dev --config basil-test.yaml
```

### Test Queries

```bash
# Create SQLite test database
sqlite3 test.db << EOF
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
EOF

# Test handler that queries database
curl http://localhost:8080/users
```

### Reset Database Between Tests

```bash
# Remove and recreate
rm test.db
sqlite3 test.db < schema.sql
sqlite3 test.db < test-data.sql
```

## Testing Sessions

### Test Login Flow

```bash
# 1. Login (save cookies)
curl -c cookies.txt http://localhost:8080/login \
  -d "username=admin" \
  -d "password=secret"

# 2. Access protected page
curl -b cookies.txt http://localhost:8080/dashboard

# 3. Logout
curl -b cookies.txt http://localhost:8080/logout

# 4. Try protected page again (should fail)
curl -b cookies.txt http://localhost:8080/dashboard
```

### Test Flash Messages

```bash
# 1. Trigger flash message
curl -c cookies.txt http://localhost:8080/submit \
  -d "data=test"

# 2. Load page that shows flash (message should appear)
curl -b cookies.txt http://localhost:8080/

# 3. Refresh (message should be gone)
curl -b cookies.txt http://localhost:8080/
```

## Testing Forms

### Test Form Submission

```bash
# Simple form
curl -X POST http://localhost:8080/contact \
  -d "name=Alice" \
  -d "email=alice@example.com" \
  -d "message=Hello"

# With CSRF token (if enabled)
# 1. Get form (save cookies)
curl -c cookies.txt -i http://localhost:8080/contact | grep csrf

# 2. Extract token and submit
curl -b cookies.txt -X POST http://localhost:8080/contact \
  -d "name=Alice" \
  -d "_csrf=TOKEN_HERE"
```

### Test Validation

```bash
# Missing required field
curl -X POST http://localhost:8080/contact \
  -d "name=Alice"
# Should return error

# Invalid format
curl -X POST http://localhost:8080/contact \
  -d "name=Alice" \
  -d "email=invalid"
# Should return validation error
```

## Common Test Scenarios

### Test Routing

```bash
# Test explicit route
curl -I http://localhost:8080/api/users
# Should return 200

# Test 404
curl -I http://localhost:8080/nonexistent
# Should return 404

# Test route params
curl http://localhost:8080/users/123
# Should pass id=123 to handler
```

### Test API Endpoints

```bash
# GET list
curl http://localhost:8080/api/users

# GET one
curl http://localhost:8080/api/users/123

# CREATE
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# UPDATE
curl -X PUT http://localhost:8080/api/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated"}'

# DELETE
curl -X DELETE http://localhost:8080/api/users/123
```

### Test Error Handling

```bash
# Test 400 Bad Request
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"invalid":"data"}'

# Test 401 Unauthorized
curl http://localhost:8080/admin

# Test 403 Forbidden
curl -b "session=user-session" http://localhost:8080/admin

# Test 500 Internal Server Error
# (trigger error in handler)
```

## Automated Testing

### Test Script Example

```bash
#!/bin/bash
# test.sh

BASE_URL="http://localhost:8080"

# Test homepage
echo "Testing homepage..."
curl -f -s "$BASE_URL/" > /dev/null || { echo "Homepage failed"; exit 1; }

# Test API
echo "Testing API..."
curl -f -s "$BASE_URL/api/users" > /dev/null || { echo "API failed"; exit 1; }

# Test login
echo "Testing login..."
curl -f -s -c cookies.txt "$BASE_URL/login" \
  -d "user=admin&pass=secret" > /dev/null || { echo "Login failed"; exit 1; }

echo "All tests passed!"
```

```bash
# Run tests
chmod +x test.sh
./basil --dev &
sleep 2  # Wait for server to start
./test.sh
kill %1  # Stop server
```

## Best Practices

1. **Use dev mode** - Get detailed error messages
2. **Check dev logs** - Visit `/__dev/log` for debugging
3. **Save cookies** - Use `-c cookies.txt` and `-b cookies.txt`
4. **Use -v flag** - See full request/response for debugging
5. **Test incrementally** - Test each endpoint as you build
6. **Use test database** - Don't test against production data
7. **Test error cases** - Not just happy paths
8. **Automate** - Write shell scripts for common test flows
