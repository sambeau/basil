# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0-alpha.2] - 2026-07-12

### Added

#### Release & CI Infrastructure
- **goreleaser configuration** — Push a version tag and GitHub Actions builds cross-compiled `basil` + `pars` binaries (macOS/Linux/Windows, amd64 + arm64, CGo-free), packages them with checksums, and publishes a GitHub Release.
- **CI workflow** — Build and test on every push and pull request.
- **`scripts/install.sh`** — One-line installer: detects OS/arch, fetches the newest release (prereleases included), verifies the SHA-256 checksum, and installs both binaries.

#### Image Transformation (FEAT-148)
- **`image()` builtin** — Transform images and serve them at content-hashed, immutable URLs. Auto-rotates via EXIF, strips metadata, resizes (width/height/fit or center-crop), converts format, and caches results to disk. No upscaling; concurrent transforms are deduplicated via singleflight.
- **`imageInfo()` builtin** — Returns `{width, height, format, orientation}` for an image without transforming it. Results are cached in memory (keyed by path + modification time) for efficient gallery-page use.
- **`imageBlur()` builtin** — Generates a Low Quality Image Placeholder (LQIP) as an inline `data:image/jpeg;base64,...` URI (~600 bytes). Use as an instant CSS background while the full image loads.
- **`imageSrcset()` builtin** — Generates multiple resized variants and returns `{src, srcset, width, height}` for responsive `<img>` tags. Supports both width descriptor mode (`[400, 800, 1200]`) and density descriptor mode (`[1, 2, 3], "x"`).
- **WebP output encoding** — All four builtins support WebP output via `gen2brain/webp` (pure-Go, no CGo). Uses native `libwebp` when available on the host, with a WASM fallback (`wazero`) otherwise. WebP typically produces 67–72% smaller files than JPEG for photos.
- **Automatic sharpening on downscale** — A subtle unsharp mask (σ=0.5) is applied after resize to recover edge detail. Disable with `{sharpen: false}` or set an explicit sigma with `{sharpen: 1.2}`.
- **Disk cache at `/__img/`** — Transformed images are served at `/__img/{hash}.{ext}` with immutable `Cache-Control` headers. Cache is stored at `./cache/images/` by default (configurable via `images.cache_dir`).
- **Binary size note** — The `basil` binary grows by ~4 MB (35 MB → 39 MB) due to the bundled WASM runtime. Install `libwebp` on your production host (`brew install webp` / `apt install libwebp-dev`) to use the faster native encoder and confirm the runtime path at startup.

#### Content-Aware Image Operations (FEAT-149)
- **Smart crop** (`{crop: "smart"}`) — Analyses images for faces and high-interest regions (edges, saturation, skin tones, composition) and crops to preserve them. Uses a two-pass architecture: PICO face detection at 640px followed by heuristic scoring at 256px. Pure Go, no new dependencies — the PICO cascade file (234 KB, MIT-licensed) is embedded via `go:embed`.
- **Focal point option** (`{focal: {x, y}}`) — Guide smart crop toward a specific point or region using normalised (0–1) coordinates. Supports both point `{x, y}` and rectangle `{x, y, w, h}` forms.
- **Seam carving** (`{scale: "smart"}`) — Content-aware resizing that removes or inserts low-energy pixel paths (seams) instead of uniformly scaling. Preserves faces, text, and important content when changing aspect ratio. Supports both reduction and enlargement. Based on Avidan & Shamir, SIGGRAPH 2007.
- **Validation** — `crop: "smart"` requires both width and height; `focal` requires `crop: "smart"`; `crop` and `scale` are mutually exclusive. Changes beyond 30% via seam carving log a warning.

#### Language Features
- **Short-circuit evaluation for `&&` and `||`** (BUG-025) — Logical operators now short-circuit correctly: `&&` skips the right operand when the left is falsy, `||` skips when the left is truthy. Collection overloads (intersection/union) are preserved for non-boolean operands.

#### Standard Library
- **Consistent string coercion for typed values** (FEAT-146) — DateTime, Duration, and Unit types now render consistently across tables, templates, and `toString()`:
  - Tables: Duration renders as "2 hours 30 minutes" (was raw dict), Unit renders as "5km" (was `<UNIT>` inspect format)
  - Templates: DateTime uses `.medium()` for human-friendly output ("Jun 15, 2025")
  - `toString()`: DateTime remains ISO format for programmatic use
- **`<field/>` tag and `record.fieldProps()`** (FEAT-145) — Declarative form field rendering with automatic label, type, validation, and accessibility attributes derived from schema metadata.
- **DataTable redesign** (FEAT-144) — Rebuilt DataTable component with sortable columns, pagination, row selection, footer aggregates, empty state, and responsive layout.
- **Meta component and Page restructure** (FEAT-142) — Extracted `<Meta/>` from `<Page/>` for flexible `<head>` management; Page now supports `dir` prop for RTL layouts.
- **Accessibility components** (FEAT-143) — New `<SkipLink/>`, `<VisuallyHidden/>`, `<LiveRegion/>`, `<FocusTrap/>` components; plus `<Details/>`, `<Accordion/>`, `<Dialog/>`, `<Toast/>`, `<Toasts/>`, `<Pagination/>`, `<ErrorSummary/>` with Pico CSS compatibility.
- **Prelude smoke test** — Automated rendering test for all 33 prelude components to catch syntax and runtime errors.
- **Money formatting in tables** — Money values in table cells now render with `.medium()` formatting.

#### Namespace & Organisation
- **`@basil/` namespace for server modules** — `@std/dev` → `@basil/log`, `@std/html` → `@basil/html`, `@std/api` → `@basil/api`. Old `@std/` names kept as deprecated aliases (removal tracked in backlog #121).
- **`@std/mddoc`** — Renamed from `@std/mdDoc` for naming consistency. Old camelCase name kept as deprecated alias.
- **Method registry migration complete** (FEAT-137) — All ~20 remaining types migrated to declarative `MethodRegistry`. `pars describe` now works correctly for all types including array, dictionary, datetime, duration, path, url, regex, file, dir, boolean, null, request, response, DBConnection, SFTP, session, record, table, and more.

### Changed
- **Config YAML consistency** — `cors.maxAge` renamed to `cors.max_age`; top-level `sqlite` key moved to `database.path`; `SessionConfig.HttpOnly` renamed to `HTTPOnly` (YAML tag `http_only`). These are breaking changes to config files from pre-alpha formats.
- **`@std/schema` docs** — Slimmed to deprecation notice pointing to `@schema { ... }` DSL syntax.

### Fixed
- **Helpful errors for non-destructured imports (BUG-005)** — Using a module dictionary as a function (`let Logo = import @./logo.pars` then `Logo()` or `<Logo/>`) now explains that `import` returns a dictionary of exports and shows the destructuring fix (`let { Logo } = import ...`). New error codes CALL-0006 and COMP-0003.
- **Component error wrapper never fired** — A case-sensitive string match meant "Cannot use '<X/>' because 'X' is not a function" (COMP-0002) was unreachable; users saw the raw "Cannot call dictionary as a function" instead. Now matched by error code.
- **Module tests now run outside this machine** — Test fixtures for module imports lived at an absolute path outside the repository; moved to `pkg/parsley/tests/test_fixtures/modules/` so CI can run them.
- **9 prelude component bugs** discovered by smoke test:
  - `.length` used as property instead of `.length()` on arrays (Checkbox, CheckboxGroup, RadioGroup, Pagination)
  - `.format("iso")` vs `.iso` differences in time-related components
  - `.floor()` called on integer division results (unnecessary)
- **DataTable footer null guard** — Simplified using short-circuit `&&` (enabled by BUG-025).
- **`array.join()` skips null and empty strings** — No longer inserts extra separators for nil/empty elements.
- **Search debug output removed** — Removed `fmt.Printf("[DEBUG]...")` statements from search option parsing that were printing to stdout in production.
- **Search text file size limit** — Added 50MB size guard for Markdown/HTML files in search indexer, matching existing PDF/DOCX limits, preventing OOM on oversized files.
- **Git handler path traversal guard** — Added explicit `..` check in `GitHandler.ServeHTTP` before delegating to `go-git-http`, returning 400 Bad Request for traversal attempts.

## [1.0.0-alpha.1] - 2026-02-26

**Parsley 1.0 Alpha** — The first alpha release of the Parsley programming language.

### Breaking Changes
- **`let`/`var` variable declarations** (FEAT-122) — Swift-style immutable/mutable semantics replace bare assignment. `let` for immutable bindings, `var` for mutable.
- **`print`/`println`/`printf` removed** (FEAT-120) — Parsley uses expression-based output; values ARE the output. Use `log()` for debugging.
- **`bytes()` renamed to `raw()`** (FEAT-119) — `bytes()` is now a unit constructor for data sizes.
- **Area constructors renamed** (FEAT-118) — Renamed to match unit suffixes.
- **Global formatting/serialization functions removed** — Replaced with unified `.fmt()` and style methods (FEAT-121).
- **`==>` and `==>>` no longer accept network targets** — Must use dedicated `=/=>` and `=/=>>` for network I/O, enforcing a visible distinction between local and remote operations.
- **`@std/table` module removed** (FEAT-128) — Use `@table` literal syntax instead: `@table [{name: "Alice", age: 30}]`.
- **`format(array, style)` removed** (FEAT-128) — Use the method form: `["a", "b"].format("and")`.
- **Uppercase form components `<Label>`, `<Error>`, `<Meta>` removed** (FEAT-128) — Use lowercase `<label>`, `<error>`, `<val>`.
- **`migrate-let-var` CLI command removed** (FEAT-128).
- **Deprecation warning infrastructure removed** (FEAT-128) — No longer needed for 1.0.

### Added

#### Language Features
- **Measurement units** (FEAT-118) — Length, weight, temperature, volume, area, and data size units with cross-system conversion and derived unit arithmetic (e.g., length × length → area).
- **`with` expression** (FEAT-123) — Scoped field access for cleaner dictionary/record manipulation.
- **Unified formatter API** (FEAT-121) — `.fmt()` method and style methods for all types.
- **Remote write operators `=/=>` and `=/=>>`** — Send data to HTTP endpoints or SFTP servers. Counterpart to the fetch operator `<=/=`.
- **PLN (Parsley Literal Notation)** — Native serialization format with full read/write support (FEAT-115), including Money and DateTime literals (FEAT-116).
- **Named capture groups** (FEAT-127) — Regex named captures return dictionaries.
- **Custom `failIfInvalid` messages** (FEAT-127) — Optional string parameter for record validation.
- **Unicode identifiers and HTML tags** (FEAT-103) — Full Unicode support in variable names and tag names.
- **SQL tag raw text content** (FEAT-117) — Direct attribute syntax for SQL tags.
- **`string.title()` Unicode-aware** — Now uses `golang.org/x/text/cases` for proper title casing.

#### Standard Library
- **`@std/hash` module** — Hashing functions for strings and data.
- **`@std/valid` refactored** — Cleaner validation API.
- **`@basil/` namespace** — Server-specific modules moved to `@basil/http` and `@basil/auth`.
- **`@std/schema` deprecated** in favor of `@schema { ... }` DSL syntax (still works for compatibility).
- **`.reorder()` method** — For dictionaries and arrays.

#### CLI & Tooling
- **`pars -e` expression evaluation** (FEAT-113) — Quick expression testing with PLN output.
- **`pars --raw` / `-r` flag** — File-like output mode for scripts.
- **`pars describe`** — Introspection API for types, methods, builtins, and modules.
- **`pars reference`** — Generate reference documentation in multiple formats.
- **`pars --check` flag** — Syntax checking without execution.
- **Help system** (FEAT-112) — Self-describing modules with unified help infrastructure.
- **Declarative method registry** (FEAT-111) — Methods described via metadata for introspection.

#### Code Formatter
- **AST-based formatter** — Full Parsley code formatter with intelligent line breaking, comment preservation, method chain formatting, multiline tag attributes, and tab indentation.

#### DevTools
- **Browser error pages** — Parse and runtime errors displayed with source line context.
- **Live reload** — Development mode with live browser refresh for templates and CSS.
- **Logs page** — Real-time log display with copy button.
- **Environment page** — Configuration display with secret masking.
- **Component loading infrastructure** — Shared CSS and component system.

#### REPL
- **Raw output mode** — Script-style output for piping.
- **AST formatter for function output** — Formatted function definitions.
- **Table repr** — Formatted table display.
- **Improved indentation** — Correct nesting for functions in arrays/dicts.

#### Database
- **PostgreSQL and MySQL support** (FEAT-107) — In addition to existing SQLite support.

#### Editor Support
- **Tree-sitter grammar** — Full Parsley grammar for syntax highlighting (FEAT-109, FEAT-114).
- **Zed Editor extension** — Language support for Zed.
- **VS Code syntax updates** — Updated for all 1.0 language features.
- **Highlight.js grammar** — Updated for web-based highlighting.

### Changed
- `string.title()` uses Unicode-aware title casing (minor behavior change with apostrophes)
- Structured error model with error codes, hints, and consistent formatting (FEAT-105, FEAT-125)
- Comprehensive documentation overhaul (FEAT-131) — Reference, cheatsheet, manual, and API docs verified against `pars describe`

### Fixed
- Duration multiplication is now commutative (BUG-021)
- Module imports no longer require execute permission (BUG-022)
- Phantom operators and wrong metadata in introspection (BUG-023)
- Dictionary insertion order preserved in `toCSV` (BUG-019)
- ISO date format accepts single-digit months and days
- Record repr uses valid Parsley syntax
- Updated deprecated mailgun API calls
- Updated deprecated goldmark `Text` property usage
- Fixed unchecked error returns on database close in CLI commands
- Acronym handling in case conversion methods

### Removed
- `print`/`println`/`printf` builtins (FEAT-120)
- `@std/table` module (FEAT-128) — use `@table` literal
- `format(array, style)` global function (FEAT-128) — use `array.format(style)`
- `<Label>`, `<Error>`, `<Meta>` uppercase components (FEAT-128)
- `migrate-let-var` CLI command (FEAT-128)
- Deprecation warning infrastructure (FEAT-128)
- 12 unused internal functions (no public API changes)

### Security
- Upgraded to Go 1.26.0 to fix TLS and URL parsing vulnerabilities (GO-2026-4337, GO-2026-4340, GO-2026-4341)
- Updated chi router dependency to v5.2.2 to fix host header injection vulnerability (GO-2025-3770)

## [0.2.0] - 2025-06-20

### Added
- **Basil Web Server** (FEAT-002)
  - Full HTTP/HTTPS server with route-based request handling
  - HTTPS with automatic Let's Encrypt certificates (autocert)
  - Development mode (`--dev`) with HTTP and live browser refresh
  - SQLite database support with Parsley operators (`<=?=>`, `<=??=>`, `<=!=>`)
  - Request logging (text and JSON formats)
  - Form parsing: URL-encoded, multipart/form-data, JSON body
  - Security headers (HSTS, CSP, X-Frame-Options, etc.)
  - Reverse proxy support (X-Forwarded-For, X-Real-IP)
  - AST caching for production performance
  - SIGHUP handler for production script/cache reload
  - Route-based response caching with configurable TTL
- **Semantic Versioning** (FEAT-005)
  - Git tag-based versioning with build-time injection
  - Makefile with `build`, `dev`, `test`, `clean`, `check` targets
  - `--version` flag shows version and commit hash

### Fixed
- Live reload no longer creates excessive log noise (BUG-004)

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
