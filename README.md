# Basil

A web server for the Parsley programming language.

## Prerequisites

- [Go](https://golang.org/dl/) 1.24 or later

## Quick Start

Create a new project:

```bash
basil --init myproject
cd myproject
basil
```

Your site will be running at http://localhost:8080

## Getting Started

### Build

```bash
go build -o basil .
```

### Run

```bash
go run .
```

Or after building:

```bash
./basil
```

### Test

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Current test coverage:
- **Server package**: 60.7% (25 implementation files, 26 test files)
- **Auth package**: Comprehensive coverage of authentication and authorization
- **Config package**: Full configuration loading and validation coverage

The test suite includes:
- Unit tests for core functionality
- HTTP handler tests using httptest
- Concurrent access tests for thread safety
- Security component tests (rate limiting, CSRF, sessions)
- Integration tests for server lifecycle

### Security Features

Basil includes production-ready security features:

**Authentication & Authorization**
- Database-backed user authentication
- Session management with secure cookies
- Role-based access control (RBAC)
- API key authentication
- WebAuthn support for passwordless authentication
- Git HTTP server with role-based repository access

**Request Protection**
- CSRF protection with token validation
- Rate limiting with per-user token buckets
- CORS configuration with credential support
- Secure session encryption (AES-256-GCM)
- HTTP security headers (Content-Security-Policy, etc.)

**Audit & Monitoring**
- Per-IP tracking for insecure HTTP requests
- Comprehensive request logging
- Development tools with database inspection
- Git authentication audit trail

## Documentation

Documentation is organized by audience:

- **[docs/](docs/)** — User-facing documentation
  - **[docs/guide/](docs/guide/)** — Basil framework user guides (quick start, FAQ, etc.)
  - **[docs/parsley/](docs/parsley/)** — Parsley language reference and manual
- **[work/](work/)** — Workflow and contributor documentation
  - Feature specifications, implementation plans, bug reports
  - Design documents, audits, and Parsley implementation notes
  - See [work/README.md](work/README.md) for details

For AI agents and contributors, see [AGENTS.md](AGENTS.md) for workflow rules and project structure.

## Project Structure

```
basil/
├── cmd/               # Command-line tools
│   ├── basil/        # Basil server
│   └── pars/         # Parsley REPL
├── server/           # Server implementation
│   ├── auth/         # Authentication
│   ├── config/       # Configuration
│   └── search/       # Full-text search
├── pkg/              # Reusable packages
│   └── parsley/      # Parsley language implementation
├── docs/             # User documentation
│   ├── guide/        # Framework guides
│   └── parsley/      # Language reference
├── work/             # Contributor documentation
│   ├── specs/        # Feature specifications
│   ├── plans/        # Implementation plans
│   └── ...           # Design docs, bug reports, etc.
├── examples/         # Example projects
├── .github/          # AI instructions and templates
├── AGENTS.md         # AI workflow rules
├── CHANGELOG.md      # Release history
├── go.mod
└── README.md
```

## License

MIT
