# Hello World Example

This is a minimal Basil application demonstrating basic Parsley script handling.

## Files

- `basil.yaml` - Configuration file
- `handlers/index.parsley` - Homepage handler
- `handlers/api/hello.parsley` - JSON API endpoint
- `public/` - Static files directory

## Running

```bash
cd examples/hello
basil --dev
```

Then visit:
- http://localhost:8080/ - Homepage
- http://localhost:8080/api/hello - JSON API
- http://localhost:8080/static/style.css - Static file
