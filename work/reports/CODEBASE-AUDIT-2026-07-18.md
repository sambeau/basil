# Basil & Parsley — Pre-1.0 Codebase Audit

**Date:** 2026-07-18
**Scope:** Full codebase — `pkg/parsley` (language), `server/` (web framework), `cmd/`.
**Lens:** security, efficiency, clarity, idiomatic Go, incomplete/dead code.
**Goal:** a clean, well-structured, easy-to-read codebase we can be proud to open-source.

---

## TL;DR

The codebase is in genuinely good shape for a 1.0. `go build ./...` and `go test ./...`
both pass, the security-critical machinery (SQL-injection defence, path sandboxing,
WebAuthn, CSRF, session encryption) is well-designed and consistently applied, and secure
defaults are the norm. New readers will mostly be impressed.

There is no single "showstopper". The work to raise the bar splits into:

- **A handful of real robustness/security hardening items** — the most important is that
  the server has **no panic-recovery middleware and no interpreter recursion limit**, so a
  single bad request can drop a connection uncleanly and a deeply-recursive script can crash
  the whole process. This is the one finding I'd fix before inviting scrutiny.
- **Mechanical hygiene** that costs almost nothing and pays off the moment strangers read
  the code: run `gofmt`, apply `staticcheck`/`golangci-lint` autofixes, delete ~15 dead
  functions, and remove a block of abandoned scratch code in the smart-crop detector.
- **A few structural notes** (very large files, an enormous exported API surface on the
  evaluator package) that are worth a decision now because they define the 1.0 compatibility
  surface, even if the refactor lands later.

Severity legend: 🔴 fix before release · 🟡 should fix · 🟢 polish / nice-to-have · ℹ️ note.

---

## 1. Automated baseline

| Tool | Result |
|------|--------|
| `go build ./...` | ✅ clean |
| `go test ./...` | ✅ all packages pass |
| `go vet ./...` | 1 issue — `server/database_test.go` context-leak (test only) |
| `gofmt -l` | **4 production files unformatted** + several test files |
| `staticcheck` | 7 issues (2 unused prod funcs, 2 dead assignments, 1 simplification, 2 unused test funcs) |
| `golangci-lint` | ~48 `modernize` + ~30 `gocritic` + 7 `errcheck` in production code (rest test-only) |
| `govulncheck` | 36 advisories — **all upstream** (Go stdlib on go1.26 + 4 modules); none in Basil's own logic |
| `deadcode -test ./...` | 31 unreachable funcs; ~15 genuinely removable after triage |

Reproduce:

```sh
go build ./... && go test ./...
gofmt -l pkg server cmd
staticcheck ./...
golangci-lint run ./...
govulncheck ./...
deadcode -test ./...
```

---

## 2. Security

### What's already good (keep doing this)

These are worth stating plainly because they're the parts that will earn trust:

- **SQL-injection defence is thorough and consistent.** Every dynamic identifier path is
  guarded. `pkg/parsley/evaluator/sql_security.go` provides `validateSQLIdentifier` /
  `quoteSQLIdentifier`, and the query builders in `stdlib_schema_table_binding.go` and
  `stdlib_dsl_query.go` validate table names, column names, `orderBy` columns, and
  `select` lists against `identifierRegex` **before** any `fmt.Sprintf` interpolation, with
  all values passed as `?` placeholders. The devtools DB browser
  (`server/devtools_db.go`) independently validates table names too.
- **Path sandboxing is correct.** `Environment.checkPathAccess`
  (`pkg/parsley/evaluator/eval_helpers.go:383`) resolves with `filepath.Abs` → `Clean` →
  `EvalSymlinks`, and `isPathAllowed` / `isPathRestricted` compare with a trailing
  `os.PathSeparator` so `/foo` cannot match `/foobar`. This defeats `../` traversal and
  symlink escapes.
- **WebAuthn is textbook** (`server/auth/webauthn.go`): challenges are single-use
  (deleted on consumption), expire after 5 minutes, there's a cleanup sweep, and
  **sign-count replay protection** is enforced on login.
- **CSRF**: `SameSite=Strict`, `HttpOnly`, `Secure` in production, constant-time token
  comparison, validated only on mutating methods (`server/csrf.go`).
- **Sessions**: AES-256-GCM with random nonces (`server/session_crypto.go`); PLN transport
  is HMAC-SHA256 signed.
- **Secure defaults**: `git.require_auth` defaults to `true`, devtools (`/__/*`) return 404
  outside dev mode, cookies go `Secure` in production, HSTS is available, and the HTTP server
  sets `ReadHeaderTimeout`/`WriteTimeout`/`IdleTimeout`.

### 🔴 S-1 · No panic recovery and no interpreter recursion limit

There is **no `recover()` anywhere** in `server/` or `pkg/parsley/` (grep confirms zero
occurrences), and the middleware chain in `server.Run` (`server/server.go:1013–1037`) has no
recovery layer. Separately, the evaluator has **no recursion-depth guard** (no `maxDepth`/
call-depth counter anywhere in `pkg/parsley/evaluator`).

Consequences:

1. **Stack-overflow → whole-process crash.** A Parsley script that recurses without bound
   (or recurses to a depth driven by request input, e.g. deeply-nested JSON processed
   recursively) overflows the goroutine stack. Stack overflow is a *fatal* runtime error
   that `recover()` cannot catch — it takes down the entire server and every concurrent
   user. This is a denial-of-service.
2. **Ordinary handler panics drop the connection.** For non-fatal panics (nil deref, index
   out of range in some corner of 75k lines of evaluator), Go's `net/http` recovers
   per-connection so the *process* survives, but the client gets an aborted connection
   instead of a clean `500`, and nothing flows through Basil's own logging.

Recommended fix (two parts):
- Add a recovery middleware as the outermost wrap in `server.Run` that `recover()`s, logs
  through the existing logger, and writes a `500` (a dev-mode variant can show the stack).
- Add a call-depth counter to the evaluator (increment on function-application / `Eval`
  entry, return a Parsley error like `RECURSION-0001` past a configurable limit, e.g.
  10 000). This converts a fatal crash into a catchable script error.

This is the one item I'd treat as release-blocking, because "one request can kill the
server" is exactly what new eyes probe for first.

### 🟡 S-2 · O(n) bcrypt scans on the auth path (extends a known issue)

`ValidateAPIKey` (`server/auth/apikeys.go:166`) loads **all** API keys and runs a bcrypt
comparison against each on every request — O(n) bcrypt operations per auth. This is already
tracked (see `work/reports/BACKLOG-INVESTIGATION-2026-03-16.md`, item #44: "prefix-based
lookup would be a better first optimization").

**New:** the same pattern exists in `LookupVerificationToken`
(`server/auth/email_verification.go:71–110`) — it scans every unconsumed, non-expired
verification token and bcrypt-compares each. An attacker submitting bad tokens forces a
full bcrypt sweep per attempt.

Root cause worth calling out for the report's audience: **bcrypt is the wrong primitive for
high-entropy secrets.** API keys (256-bit) and verification tokens (256-bit) are not
guessable, so the offline-brute-force resistance bcrypt buys is unnecessary. A fast keyed
hash (HMAC-SHA-256 with a server secret) stored in an indexed column enables an **O(1)
lookup** and removes the DoS surface entirely — faster *and* safer. If bcrypt is kept,
the interim mitigation is to filter by the stored `key_prefix` first so only one candidate
is bcrypt-checked.

### 🟢 S-3 · Unbounded response read in `fetch`

`fetchUrlContentFull` uses `io.ReadAll(resp.Body)` with no cap
(`pkg/parsley/evaluator/eval_network_io.go:582`). A large or malicious upstream response can
exhaust memory. Wrap with `io.LimitReader` (or `http.MaxBytesReader`) at a generous default,
overridable per-request. The 30 s client timeout is good and already present.

### 🟢 S-4 · Recovery-code generation has modulo bias, and ignores the RNG error

`generateRecoveryCode` (`server/auth/recovery.go:124–135`) does
`idx := randByte() % byte(len(recoveryCodeChars))` where `len` is 31. Since
256 = 8·31 + 8, the first 8 characters are slightly more likely — a small, avoidable bias in
a security token. Use `crypto/rand`'s rejection sampling (or `rand.Int(rand.Reader, big)`).
Also `randByte` (`recovery.go:143`) ignores the error from `rand.Read`; on a (rare) RNG
failure it would silently emit zero bytes.

### 🟢 S-5 · Minor hardening notes

- **Session tokens stored in plaintext** as the primary key in the `sessions` table
  (`server/auth/session.go`). A DB read leak yields directly usable session tokens. Storing
  a SHA-256 of the token (and looking up by hash) limits the blast radius. Low priority —
  tokens are high-entropy and server-side.
- **`secureCompare`** (`server/csrf.go:92`) is a correct hand-rolled constant-time compare,
  but `crypto/subtle.ConstantTimeCompare` is the idiomatic choice and self-documents intent.
- **`warnedIPs` map** (`server/git.go`) grows one entry per distinct HTTP-with-auth client
  IP and is never bounded. Only reachable under the insecure-HTTP misconfiguration; trivial.

### ℹ️ S-6 · Dependency advisories (all upstream)

`govulncheck` reports 36 advisories, none in Basil's own code:

- The bulk are **Go standard library** issues fixed in **go1.26.1** — the module targets
  `go 1.25.0` and is being built with a go1.26 toolchain. Bumping the build toolchain to
  ≥ go1.26.1 clears them.
- Four third-party modules have called-code advisories: `golang.org/x/net`,
  `golang.org/x/crypto`, `golang.org/x/image`, `github.com/yuin/goldmark`.
  `go get -u` those four + `go mod tidy`.

Worth doing right before tagging so the first `govulncheck` a curious visitor runs is clean.

---

## 3. Incomplete implementations & stubs

Exactly the "stubs that were never completed" category to clear before release.

### 🟡 I-1 · `sftpFileHandle.rmdir({recursive: true})` silently does nothing

`sftpRmdir` (`pkg/parsley/evaluator/eval_network_io.go:87–111`) parses the `recursive`
option, then discards it (`_ = recursive`) and calls `RemoveDirectory`, which only removes
*empty* directories. A user passing `recursive: true` gets no error and no recursion — a
silent footgun. Either implement recursive removal, or return a clear "recursive rmdir not
supported" error so the option can't be silently ignored.

### 🟡 I-2 · Abandoned scratch code in the smart-crop detector

`server/images/smartcrop/detect.go:133–143` computes `r1`/`c1`, then a
`// Wait - looking at Pigo more carefully:` comment, then **recomputes** them — so lines
134–135 are dead assignments (this is the `staticcheck` SA4006 + `ineffassign` hit). This
reads as a developer thinking out loud, left in a shipping file. Delete the first
computation and the scratch comment; keep the corrected block at 142–145.

### 🟢 I-3 · TODO markers to resolve or convert to backlog items

10 markers in production code. Most are benign, but a few are user-visible:

- `pkg/parsley/errors/errors.go:1264` — `"CSV write not yet implemented for SFTP"` — confirm
  this is a deliberately unsupported combination with a clean error (it appears to be).
- `server/search.go:591` and `server/search/scanner.go:137` — `// TODO: Add proper logging`
  swallow scan errors. Wire these into the logger before release.
- The rest (`evaluator.go:1066` import-dependency tracking, `stdlib_dsl_query.go:2523` bulk
  insert) are genuine future-work; move to the backlog and drop the inline TODO.

---

## 4. Dead code

`deadcode` run *without* `-test` reports 105 functions, but that count is misleading: it
only treats `cmd/basil` and `cmd/pars` as roots, so it flags the entire embedding API
(`pkg/parsley/parsley`) and every test-only helper as "dead". Re-running with `-test ./...`
gives the accurate list — **31 unreachable functions** — which I've triaged into three
buckets.

### 🟢 Bucket A — genuinely removable (~12 functions)

Safe to delete; not part of the public embedding API, not interface-required, unused even
by tests:

| Location | Function |
|----------|----------|
| `pkg/parsley/evaluator/eval_errors.go:297` | `newUndefinedMethodError` (also `staticcheck` U1000) |
| `pkg/parsley/evaluator/form_components.go:445` | `evalMetaComponent` (also `staticcheck` U1000) |
| `pkg/parsley/evaluator/stdlib_table.go:594` | `TableFromDict` |
| `pkg/parsley/lexer/lexer.go:2442` | `Lexer.EnterTagContentMode` |
| `pkg/parsley/pln/parser.go:56` | `Parser.Errors` |
| `pkg/parsley/pln/pln.go:57` | `SerializeWithEnv` |
| `server/auth/auth.go:73` | `DefaultConfig` |
| `server/auth/middleware.go:79` | `GetUserFromContext` |
| `server/search/fts5.go:164` | `FTS5Index.Weights` |
| `server/search/watcher.go:165` | `UpdateStats.String` |
| `server/images/seamcarve/seamcarve_test.go:36` | `createHorizontalStripeImage` (test) |
| `server/images/smartcrop/sobel_test.go:383` | `approxEqual` (test) |

### ℹ️ Bucket B — public embedding API, unused internally (keep, but decide)

`pkg/parsley/parsley` exposes an options/logger facade for embedders. A subset is currently
exercised by nothing — `StdoutLogger`, `WriterLogger`, `NullLogger` (logger.go),
`WithEnv`, `WithSecurity`, `WithFilename` (options.go), `EvalFile` (parsley.go). These are
legitimate public surface for anyone embedding Parsley, but untested public API bit-rots.
Recommendation: add one example/test that exercises each, **or** trim to the options you
actually intend to support in 1.0. Either way, make it a conscious choice — it's part of the
compatibility promise.

### ℹ️ Bucket C — deadcode false positives (do not remove)

The 8 `TokenLiteral`/`String` methods on `QueryOneStatement`, `QueryManyStatement`,
`ExecuteStatement`, `QueryCTERef` (`pkg/parsley/ast/ast.go`) are required to satisfy the
`ast.Statement`/`ast.Expression` interfaces; the node types themselves are live (handled in
`evaluator.go`'s type switch at 4638). `deadcode` flags them only because they're never
reached via dynamic dispatch. Leave them.

---

## 5. Clarity & idiomatic Go

### 🟢 C-1 · `gofmt` these 4 production files

`gofmt -l` flags production code — the first thing many Go developers run:
`pkg/parsley/evaluator/sql_security.go`, `pkg/parsley/format/constants.go`,
`server/auth/email_verification.go`, `server/auth/rate_limit.go`. Fix:
`gofmt -w pkg server cmd`. Consider a CI check (`gofmt -l` must be empty) so this never
regresses.

### 🟢 C-2 · Apply the mechanical `golangci-lint` / `gocritic` fixes

Low-risk, high-signal polish (production-code counts):

- `modernize` (~48): `min`/`max` builtins, `maps.Copy`, `range over int`, `slices.Contains`,
  `strings.SplitSeq`. Many are auto-fixable with `golangci-lint run --fix`.
- `gocritic` (~30): `ifElseChain` → `switch` (`eval_regex.go:95`, `form_components.go:115`,
  `transform.go:109`), `octalLiteral` → `0o755` (`methods_file_http.go:148/150`),
  `sprintfQuotedString` → `%q` (`form_components.go`), `builtinShadow` of `max`
  (`eval_operators.go:188/360` — rename the local), `importShadow` of `format`
  (`cmd/pars/main.go:264`).
- `errcheck` (7 in production, e.g. `eval_network_io.go:40/43` unchecked `Close()`,
  `server/search.go:375`): at minimum assign to `_` with a reason, or log.

The `staticcheck` `S1017` at `pkg/parsley/help/compose.go:98` (`strings.TrimSuffix` instead
of a manual `if`+slice) is a nice readability one.

### ℹ️ C-3 · Very large files

15 non-test files exceed 1 500 lines; the extremes are `evaluator.go` (6 269),
`parser.go` (5 333), `lexer.go` (3 319), `stdlib_dsl_query.go` (3 315),
`stdlib_table.go` (3 254), `eval_tags.go` (2 651), `ast/format/ast_format.go` (2 551).
These are navigable (good naming, clear section comments) but daunting for a first-time
reader. Splitting `evaluator.go` along the seams that already exist (the file has clear
functional regions) would lower the barrier to contribution. Not urgent; worth a
post-1.0 pass.

### ℹ️ C-4 · The evaluator's public API surface is very large

`pkg/parsley/evaluator` exports ~540 identifiers. Because it's a non-`internal` package,
every one of those becomes part of the 1.0 compatibility surface the moment you tag. Most
(the `Object` zoo, `eval*` helpers) look like implementation detail that happens to live in
an exported package. For a clean public story, consider moving the interpreter internals
under `pkg/parsley/internal/...` and keeping the curated facade in `pkg/parsley/parsley`.
This is the highest-leverage *structural* decision to make **before** 1.0, precisely because
it's hard to change after. If a full move is too much now, at least document which packages
are "public API" vs "internal, no compatibility guarantee" in the README.

### 🟢 C-5 · Minor naming friction

`pkg/parsley/format` (Parsley value pretty-printing) and `pkg/parsley/formatter` (HTML
pretty-printing) sit side by side with near-identical names — an easy source of confusion
for newcomers. A rename (e.g. `format` → `pretty`/`inspect`, or `formatter` → `htmlfmt`)
would remove the ambiguity. Cosmetic.

---

## 6. Efficiency

### 🟡 E-1 · Regexes compiled per call on the render/hot path

Several `regexp.MustCompile` calls live *inside* functions, recompiling on every
invocation. The worst offenders are on the form/tag render path:

- `pkg/parsley/evaluator/form_binding.go:152` (`removeAtRecord`), `:279`
  (`removeFieldAttribute`), `:285` (`removeComponentAttributes`) — these run during
  component rendering.
- `pkg/parsley/evaluator/eval_datetime.go:624, 684, 700, 708` — recompiled per datetime
  parse.
- `pkg/parsley/evaluator/methods.go:1355, 1380` — per phrase-highlight / paragraph split.

Fix: hoist to package-level `var … = regexp.MustCompile(…)`, exactly as the file already
does for `whitespaceRegex` et al. (`methods.go:27–30`). Pure win, no behaviour change.

### 🟡 E-2 · O(n) bcrypt scans

See **S-2** — this is as much an efficiency problem as a security one. Prefix-indexed
lookup (or HMAC + indexed column) turns per-request auth from O(n) bcrypt into O(1).

### 🟢 E-3 · Unbounded `io.ReadAll`

See **S-3**. Also relevant to memory efficiency under load.

---

## 7. Suggested order of work

1. **🔴 S-1** — recovery middleware + evaluator recursion limit. *(robustness; the one I'd
   gate the release on)*
2. **🟡 S-2 / E-2** — prefix-indexed API-key and verification-token lookup.
3. **🟡 I-1, I-2** — fix the silent `rmdir` no-op; delete the abandoned smart-crop block.
4. **🟢 Mechanical sweep** — `gofmt -w`; `golangci-lint run --fix`; delete Bucket-A dead
   code; resolve the two error-swallowing search TODOs; add a `gofmt`/`vet` CI gate.
5. **🟢 S-3/E-1** — bound `fetch` reads; hoist per-call regexes.
6. **🟢 S-4, S-5** — recovery-code bias, `subtle.ConstantTimeCompare`, session-token hashing.
7. **ℹ️ Before tag** — bump toolchain to ≥ go1.26.1, `go get -u` the four flagged modules,
   re-run `govulncheck`.
8. **ℹ️ Decide (structural)** — evaluator public-API surface (C-4) and Bucket-B embedding
   API. Document the public/internal boundary in the README even if the refactor waits.

Items 1–5 are a few focused days and would leave the codebase genuinely polished for public
eyes. Everything in §2's "What's already good" is worth being proud of as-is.
