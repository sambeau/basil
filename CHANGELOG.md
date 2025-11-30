# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Phase 2: Enhanced Features** (FEAT-002)
  - SQLite database support with Parsley operators (`<=?=>`, `<=??=>`, `<=!=>`)
  - Hot reload in dev mode with live browser refresh
  - Request logging (text and JSON formats)
  - Form parsing: URL-encoded, multipart/form-data, JSON body
  - HTTPS with automatic Let's Encrypt certificates (autocert)
  - Security headers (HSTS, CSP, X-Frame-Options, etc.)
  - Reverse proxy support (X-Forwarded-For, X-Real-IP)
  - AST caching for production performance
  - SIGHUP handler for production script/cache reload
  - Route-based response caching with configurable TTL

### Changed
- None

### Fixed
- None

## [0.1.0] - 2025-11-30

### Added
- Development Process Framework (FEAT-001)
  - `AGENTS.md` for AI operational context
  - `ID_COUNTER.md` for automated ID allocation
  - `BACKLOG.md` for deferred items tracking
  - Instruction files for code standards and commit conventions
  - Prompt files for feature, bug, and release workflows
  - Document templates for specs, bugs, and implementation plans
  - Human-friendly guide documentation (quick-start, cheatsheet, FAQ, walkthroughs)

### Changed
- Updated `.github/copilot-instructions.md` with project workflow

## [0.0.0] - 2025-11-30

### Added
- Initial project setup
- Basic Go module structure
- VS Code debug configuration
