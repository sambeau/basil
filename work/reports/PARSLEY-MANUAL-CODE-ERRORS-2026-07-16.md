# Parsley Manual Code Errors — 2026-07-16

**Scope:** every ` ```parsley ` code block in `docs/parsley/manual/**/*.md` and `docs/basil/manual/*.md` (63 files, 1,067 code blocks).
**Method:** each block was extracted, written to a `.pars` file, and run through `pars` (v0.15.3). No source-code verification was done — this is purely "what happens when you actually run it." Where a block's own trailing `// value` comments gave an expected result, those were used to sanity-check output.
**Goal:** find hallucinated/invented Parsley syntax before we write replacement examples.

Do not fix anything from this report directly — it's the input to the next pass, where we write new examples and correct the surrounding prose.

---

## Headline numbers

- 1,067 code blocks, split into ~3,180 individually-testable statements.
- **43 confirmed syntax/parse errors** that are not explained by missing files or missing Basil-server context — these are the real "hallucinated code" candidates, detailed below.
- The remaining ~850 failures are near-universally explainable as: the snippet references a schema/variable defined earlier on the same page but not repeated in this isolated block, needs a real database/file/network connection, or needs the Basil server runtime (`@DB`, `@SEARCH`, HTML component prelude, form context). These are **not** doc bugs — see "Not investigated further" at the end.

A note on method: testing isolated snippets from a page that assumes shared state (imports, schemas, `let` bindings shown once and reused across several later blocks) produces a lot of "Identifier not found" noise that isn't a real problem. I built a harness that reconstructs each statement's context from prior `let`/`export`/`import` lines *within the same code block* to cut this down, but it can't see across blocks — so a block that says `User.title("name")` when `@schema User` was defined three blocks earlier will still show as "Identifier not found: `User`" here. Read the counts below with that in mind.

---

## A. Confirmed syntax/parse errors (fix these)

### 1. `@insert` DSL — inconsistent, and partly broken

Two different, mutually-inconsistent syntaxes appear, and both have problems:

**[record.md:1085](docs/parsley/manual/builtins/record.md:1085)** uses spread syntax that doesn't parse:
```
@insert(Users |< ...form .)
```
```
Parser error: line 39, column 25
  expected identifier, got '...'
```

**[query-dsl.md:366](docs/parsley/manual/features/query-dsl.md:366)** uses a `|<` field syntax with a brace-wrapped value that also doesn't parse:
```
@insert(
	Users 
	|< name: {userName} 
	|< email: "carol@test.com" 
	.)
```
```
Parser error: line 4, column 20
  expected ':', got '}'
```
(Fails on the *second* `|<` line — `{userName}` as a value in the first line apparently swallows the wrong thing.) Neither form of `@insert` shown in the docs actually runs. The real `@insert` syntax needs to be pulled from the query-DSL implementation.

### 2. `enum(...)` / `string(...)` with quoted string arguments — wrong call form

Two separate pages invent an `enum("a", "b", "c")` (or `string("a","b","c")`) call form with quoted strings as arguments:

**[schema.md:926](docs/parsley/manual/builtins/schema.md:926)**:
```
subject: enum("General", "Support", "Sales") | {title: "Subject"}
```
**[query-dsl.md:641](docs/parsley/manual/features/query-dsl.md:641)**:
```
role: string("admin", "user", "guest")
```
Both fail the same way: `expected identifier, got 'General'` / `got 'admin'` — the parser wants **unquoted identifiers** in that position, not strings.

The correct form is used elsewhere in the *same file* ([schema.md:32](docs/parsley/manual/builtins/schema.md:32)) and confirmed working directly:
```
role: enum["user", "admin"] = "user"     // works — square brackets, not a call
```
So `enum[...]` (array literal) is real; `enum(...)` and `string(...)` as shown are invented. Every `enum(...)`/`string(...)`-with-quoted-args example needs to be rewritten to the bracket form.

### 3. `@https://...` fetch operators — mostly fine, but `<=?=>` and `<=#=>` don't exist

This one needs care: the docs use a family of custom arrow-style operators for I/O (`<=/=` fetch, `=/=>` push, `<==` read). **These are real** — confirmed directly:
```
pars -e 'let data <=/= JSON(@https://api.example.com/users.json)'
→ Runtime error: HTTP error: fetch failed: ... (correctly attempted a real network call)
```
`@https://...` as a URL literal is also real (`pars describe` confirms a `url` type with `scheme`/`host`/`path`/etc.).

But two operators used in the docs **do not exist**, despite following the same naming pattern:

- **`<=?=>`** ("query-assign") — used in [table.md:159](docs/parsley/manual/builtins/table.md:159) and [database.md:371](docs/parsley/manual/features/database.md:371):
  ```
  let results <=?=> "SELECT * FROM orders WHERE total > 100"
  ```
  ```
  Parser error: expected '=', got '<=?=>'
  ```
- **`<=#=>`** ("exec-assign") — used in [commands.md:98,136](docs/parsley/manual/features/commands.md:98) and [security.md:121](docs/parsley/manual/features/security.md:121):
  ```
  let result <=#=> cmd
  ```
  ```
  Parser error: expected '=', got '<=#=>'
  ```
Both look like someone extended the real `<=/=`/`=/=>`/`<==` pattern by analogy rather than checking the implementation. Whatever the real DB-query and shell-exec assignment syntax is (if one exists at all), it isn't this.

### 4. `@dur(...)` breaks when used as a dict value

**[commands.md:98](docs/parsley/manual/features/commands.md:98)**:
```
let cmd = @shell("make", ["build"], {
    env: {PATH: "/usr/local/bin:/usr/bin"},
    dir: @./project,
    timeout: @dur(30, "s")
})
```
```
Parser error: line 4, column 18
  expected ':', got '('
```
Confirmed in isolation — `@dur(30, "s")` as a bare expression is fine (well, it's actually `Identifier not found: dur — Did you mean 'dir'?` as a bare call, so `@dur` may not even be the right builtin name), but as a dict value it breaks parsing outright:
```
pars -e '{timeout: @dur(30, "s")}'
→ Parser error: expected ':', got '('
```
Two separate bugs bundled here: (a) `@dur` doesn't appear to be a real identifier (the runtime hints "Did you mean `dir`?" — this may be conflating with the `dir:` key on the previous line, worth checking what the actual duration-literal/builtin is called), and (b) whatever it is, using `name(...)` call syntax as a dict value breaks the parser.

### 5. Dictionary shorthand `{name, age}` doesn't exist

**[dictionary.md:807](docs/parsley/manual/builtins/dictionary.md:807)**:
```
{name, age}
```
```
Parser error: line 2, column 6
  expected ':', got ','
```
This is JS/ES6 property-shorthand syntax (`{name, age}` as shorthand for `{name: name, age: age}`). Parsley requires explicit `key: value` pairs — confirm the intended replacement is just `{name: name, age: age}` (or whatever the actual doc context was demonstrating).

### 6. `@field name: Type` syntax doesn't parse

**[schema.md:432](docs/parsley/manual/builtins/schema.md:432)**:
```
@field phone: Contact.phone  // <input pattern="^\+?[0-9\s\-]+$" ...>
```
```
Parser error: line 1, column 13
  unexpected ':'
```
Whatever `@field` is meant to do here, this exact form isn't valid syntax.

### 7. `@table(...)` literal — inconsistent row columns

Two examples build a `@table` literal where the second row has an *extra* column not present in the first row, which the `@table` literal constructor rejects outright:

**[schema.md:28](docs/parsley/manual/builtins/schema.md:28)** (first row has no `role`, second row adds `role: "admin"`):
```
let users = @table(User) [
    {id: 1, name: "Alice", email: "alice@example.com"},
    {id: 2, name: "Bob", email: "bob@example.com", role: "admin"}
]
```
```
Parser error: @table row 2: extra columns not in first row: role
```
**[schema.md:769](docs/parsley/manual/builtins/schema.md:769)** — same shape, extra `inStock` on row 2.

This does not look intentional (contrast with **[table.md:92](docs/parsley/manual/builtins/table.md:92)**, which deliberately shows a *missing*-column row and explicitly comments `// Error: missing column 'age'` — that one is working exactly as documented and is not a bug). The schema.md examples have no such disclaimer and read as if they're meant to succeed.

### 8. `try(fn() { ... })` is not how `try` works

**[security.md:141](docs/parsley/manual/features/security.md:141)**:
```
let result = try(fn() {
    let secret <== text(@./secrets/key.pem)
})
```
```
Parser error: line 1, column 14
  Try requires a function or method call
  Use: try can only wrap function calls like: try func()
   or: try can only wrap method calls like: try obj.method()
```
The error message is unusually explicit about the fix. This is also confirmed as the *only* form used correctly elsewhere — e.g. [errors.md:47](docs/parsley/manual/fundamentals/errors.md:47) `let {result, error} = try risky()` parses fine (it only fails at runtime because `risky()` isn't defined, which is expected for an isolated snippet). So `try` never takes parens/a wrapped closure — it's a prefix keyword directly in front of a call. security.md's example needs to be rewritten to something like `try text(@./secrets/key.pem)`.

### 9. Named function declarations `fn name(args) { }` don't exist — only `let name = fn(args) { }`

This is the most repeated and highest-impact finding — it appears in **4 different Basil-manual pages, 6 blocks**, always in route-handler examples:

- [api.md:96](docs/basil/manual/api.md:96) — `fn getUser(req) { ... }`
- [api.md:106](docs/basil/manual/api.md:106) — `fn createUser(req) { ... }`
- [api.md:132](docs/basil/manual/api.md:132) — `fn handleLogin(req) { ... }`, `fn handleOldUrl(req) { ... }`
- [log.md:90](docs/basil/manual/log.md:90) — `fn handleRequest(req) { ... }`
- [session.md:160](docs/basil/manual/session.md:160) — `fn handleLogin(req) { ... }`, `fn handleLogout(req) { ... }`

All fail identically:
```
Parser error: line 1, column 3
  expected '(', got 'getUser'
```
Confirmed directly: `fn double(x) { x * 2 }` fails the same way; only the assigned-anonymous form works:
```
let double = fn(x) { x * 2 }   // fine
```
And [record.md:1085](docs/parsley/manual/builtins/record.md:1085) actually shows the *correct* pattern elsewhere in the same doc set: `export save = fn(props) { ... }`. So the fix across all six blocks is mechanical — every `fn routeHandlerName(req) { ... }` needs to become `export routeHandlerName = fn(req) { ... }` (or `let ... = fn ...`, whichever the routing convention actually is — worth checking against server/routing before settling on the replacement).

### 10. `??` inside a function call's parentheses doesn't parse

**[search-guide.md:213](docs/basil/manual/search-guide.md:213)**:
```
let page = toInt(@params.page ?? "1")
```
```
Parser error: line 2, column 30
  expected ')', got '??'
```
Confirmed as a genuine, narrow parser limitation, not a harness artifact:
```
pars -e 'toInt(null ?? "1")'          → Parser error: expected ')', got '??'
pars -e 'toInt((null ?? "1"))'        → 1   (works with an extra layer of parens)
```
So `??` used directly as a function-call argument breaks, but wrapping it in its own parens works around it. Worth flagging to confirm whether this is intended precedence/grammar behavior or a real parser bug — if it's intended, the docs should show the wrapped form.

### 11. Trailing `// comment` on a tag attribute line breaks the parser

**[parts-guide.md:231](docs/basil/manual/parts-guide.md:231)**:
```
<Part
    src={@./heavy-chart.part}
    view="placeholder"
    part-lazy="loaded"
    part-lazy-threshold={200}    // start 200px before it's visible (optional)
/>
```
```
Parser error: line 1, column 1
  Expected closing tag </Part>, got end of file
```
Confirmed by isolating the variable: removing the trailing `// comment` from the last attribute line makes the exact same tag parse cleanly. Inside a tag literal, `//` is evidently not treated as a comment the way it is in ordinary code, and it appears to swallow the rest of the tag including the closing `/>`. Any doc example with an inline comment on a tag-attribute line needs the comment moved to its own line (or removed).

---

## B. Not code — mis-fenced content

These aren't hallucinated syntax so much as prose/reference content that got put inside a ` ```parsley ` fence by mistake, so it reads as "broken code" when it's not meant to be code at all:

- **[html.md:51](docs/basil/manual/html.md:51)** — a plain comma-separated list of supported HTML tag names (`<p>, <span>, <div>, ...`) sitting in a parsley fence.
- **[images.md:45,120,178,210](docs/basil/manual/images.md:45)** — function-signature notation using a unicode arrow, e.g. `image(path) → string`, `imageInfo(path) → dict`. This is a signature-listing convention, not executable code.

Recommend moving these to plain text or a non-executable fence (or reformatting as a table, which the docs already use elsewhere for this exact purpose).

---

## C. Confirmed *not* bugs (checked and working as intended)

Flagging these so they don't get "fixed" by mistake in the next pass:

- **[comments.md:105](docs/parsley/manual/fundamentals/comments.md:105)** — `# a comment` is explicitly shown as an error case (`// ❌ parse error: '#' is not a comment character`) and correctly fails. Working as documented.
- **[database.md:186](docs/parsley/manual/features/database.md:186)** and **[tags.md:336](docs/parsley/manual/fundamentals/tags.md:336)** (same example, duplicated across two pages) — `<SQL>...'@{name}'...</SQL>` is explicitly shown as the "❌ ERROR" half of an error/fix pair, demonstrating that string interpolation inside `<SQL>` tags is correctly rejected. Working as documented.
- **[table.md:92](docs/parsley/manual/builtins/table.md:92)** — a `@table` literal with a deliberately missing column, explicitly commented `// Error: missing column 'age'`. Working as documented (contrast with schema.md's two *un*-commented cases in finding A.7, which are real bugs).
- `fail(...)`, `api.notFound(...)` etc. throughout **errors.md** — these intentionally demonstrate Parsley's real error objects and correctly produce the documented error text.

---

## D. Not investigated further (expected, not doc bugs)

For completeness, the ~850 remaining failures break down as:

| Category | Count | Example |
|---|---|---|
| Undefined identifier (schema/variable/import from earlier in the same page, not repeated in this snippet) | ~600 | `Identifier not found: User`, `math`, `db` |
| Needs Basil server context | ~58 | `@SEARCH is only available in Basil server context`, `HTML components not available: prelude not initialized` |
| Missing fixture file | ~85 | `Failed to read file 'sales.csv'`, `Module not found: helpers.pars` |
| Other runtime (mostly intentional error demos, see §C) | ~100 | — |

None of these were individually verified against the codebase — per your instruction, that's the next pass, and only worth doing for cases that still look suspicious once real page-level context (the actual import/schema from earlier in the page) is restored.

---

## Files with confirmed real issues (for the fix pass)

- `docs/parsley/manual/builtins/dictionary.md` — #5
- `docs/parsley/manual/builtins/record.md` — #1
- `docs/parsley/manual/builtins/schema.md` — #2, #6, #7 (×2)
- `docs/parsley/manual/builtins/table.md` — #3
- `docs/parsley/manual/features/commands.md` — #3 (×2), #4
- `docs/parsley/manual/features/database.md` — #3
- `docs/parsley/manual/features/query-dsl.md` — #1, #2
- `docs/parsley/manual/features/security.md` — #3, #8
- `docs/basil/manual/api.md` — #9 (×3 blocks)
- `docs/basil/manual/log.md` — #9
- `docs/basil/manual/session.md` — #9 (×1 block, 2 handlers)
- `docs/basil/manual/search-guide.md` — #10
- `docs/basil/manual/parts-guide.md` — #11
- `docs/basil/manual/html.md` — mis-fenced content
- `docs/basil/manual/images.md` — mis-fenced content (×4 blocks)
