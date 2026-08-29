# Parsley for people who know JavaScript or Python — the gotchas

Every entry was hit during a full audit of the Parsley manual, where the docs' own examples
had been written from JS/Python memory instead of run. Hit counts are the distinct defects
found for that pattern; every wrong/right pair below was run against `pars` first. "Silent"
means the code runs and produces something wrong; "loud" means it errors — still worth
knowing, because an error aborts the whole block, so nothing after the mistake runs.

## 1. `let` is immutable, and nothing declares itself — 23 hits
```parsley
let count = 0
count = count + 1     // ✗ cannot reassign immutable binding 'count'
results = query(q)    // ✗ cannot assign to undeclared variable 'results'
let user = a
let user = b          // ✗ 'user' is already declared in this scope

var count = 0         // ✓ var is the mutable one
let results = query(q)
let existing = b      // ✓ one name per binding per scope
```

Loud: JS `let` is mutable and freely shadowed, Parsley `let` is a constant and a scope allows
one binding per name — which breaks closure counters, accumulators, and side-by-side variants.

## 2. Invented functions and modules — 22 hits
```parsley
fetch(url)   execute(cmd("ls"))   fs.read("/etc/passwd")   http.get(url)   now()
dur(30, "s")   range(1, 10)   len(xs)   print(x)   number("42")   SQLITE(":memory:")

let r <=/= JSON(@https://example.com/x)     // fetch: the <=/= operator
let out = @shell("ls", ["-la"]) <=#=> null
let data <== text(@/etc/passwd)             // file read: <== over a format handle
@now   @30s   1..10   xs.length()   log(x)   toNumber("42")   @sqlite(":memory:")
```

Loud (`Identifier not found`, usually with the right name suggested): there is no `fs`,
`http`, or `fetch` at all, because I/O is operators over handles rather than functions.

## 3. The phantom DB API — 20 hits
```parsley
let Users = db.table("users", schema: User)   // ✗ no such method
Users.create({name: "Bob"})                   // ✗ Unknown method `create` for table binding
db.insert(user)                               // ✗ insert lives on the binding, not the db
Users.where({name: "Alice"}).first()          // ✗ Unknown method `first` for table

db.createTable(User, "users")                 // ✓ bind() alone leaves "no such table"
let Users = db.bind(User, "users")
let _ = Users.insert(User({name: "Bob"}))
let alices = Users.where({name: "Alice"})     // returns a Table — not chainable
```

Loud: bindings expose `.all()`, `.where({…})`, `.find(id)`, `.first()` and
`.insert/.update/.save/.delete(record)`, and raw SQL is `db <=?=>`/`<=??=>`/`<=!=>`, so
ORM-shaped names invented from Rails or Prisma never resolve.

## 4. A tag body is expression position, not text — 18 hits
```parsley
<p>Hello</p>                    // ✗ Identifier not found: `Hello`
<h1>Welcome to {title}!</h1>    // ✗ expected ':' — {…} parses as a dictionary literal

<p>"Hello"</p>                  // ✓ literal text is quoted
<p>user.name</p>                // ✓ a bare expression is fine
<h1>`Welcome to {title}!`</h1>  // ✓ interpolation needs a backtick string
```

Mixed (unquoted words are a runtime error, `{expr}` a parse error): the JSX habit is exactly
inverted, because in a tag body braces mean "dictionary" and quotes mean "text".

## 5. Method names do not transliterate from JS — 17 hits
```parsley
"abc".toUpperCase()   "abcd".startsWith("ab")   "abcd".substring(0, 2)   "abcd".length
[1,2,3].contains(2)   "a1".match(/\d/)   d.get("a")   42.toString()   t.select("name")

"abc".toUpper()       "abcd" ~ /^ab/            "abcdefg".truncate(4)   "abcd".length()
2 in [1,2,3]          "a1" ~ /\d/        d["a"]  toString(42)   t.select(["name"])
```

Loud, but the right name is rarely the JS one: regex testing is `~`, membership is `in`,
dictionaries have no `.get()`, and `.length` without parens gives a misleading error.

## 6. A Table is not an array — 13 hits
```parsley
t.length()    // ✗ Unknown method `length` for table    t.reduce(f, 0)   // ✗ not in registry
t.all()       // ✗ .all(pred) is a quantifier           t.sortBy(fn(r){…}) // ✗ arrays only
t.groupBy(fn(r) {…})                 // ✗ groupBy takes column names
@table [["name","age"], ["A", 1]]    // ✗ expected dictionary literal, got LBRACKET

t.count()   t.sum("age")   t.orderBy("age", "desc")
t.groupBy("dept", fn(rows) { {staff: rows.length()} })
@table [{name: "A", age: 1}]         // dictionary rows, identical columns in every row
```

Loud: the Table registry is column-oriented rather than callback-oriented, so lodash-shaped
APIs are the ones most often invented for it (`.length()` is right for arrays and strings).

## 7. The line-start paren trap — 12 hits
```parsley
(-5).abs()
(-3.14).abs()      // ✗ Cannot call integer as a function

let a = -5   a.abs()
let b = -3.14   b.abs()      // ✓
```

Loud but baffling: a line beginning `(` continues the previous expression as a call, and a
blank line between them does not help — only a non-paren start does.

## 8. Stdlib modules that were removed or renamed — 12 hits
```parsley
let {basil} = import @std/basil   // ✗ removed — use @basil/http or @basil/auth
let t = import @std/table         // ✗ removed — use the @table literal / table()
let api = import @std/api         // deprecated alias; canonical is @basil/api
```

Mixed: removed modules are loud, but deprecated aliases still work and only warn, so they
survive every test (`@std/valid` stayed importable after losing most of its v0 surface).

## 9. Schemas and Records — 12 hits
```parsley
@schema User { id: integer  name: string }   // every field required; id is not auto
let u = User({name: "A"})
u.isValid()        // ✗ false — construction does not validate
u.valid            // ✗ silently null
u.errors           // ✗ `errors` is a method on record, not a property
u.set("name", "B") // ✗ Unknown method `set` for record

@schema User { id: int(auto)  name: string  nick: string? }   // ? = optional/nullable
let u = User({name: "A"}).validate()
u.isValid()   u.errors()   u.update({name: "B"})
```

Silent then loud: a field is required unless marked `auto` or `type?` and `createTable` emits
NOT NULL for every non-`?` field, so one tidy boilerplate schema invalidates every record
example after it. (`Schema.name` also returns the schema's own name, shadowing that field.)

## 10. The Query DSL lives entirely inside `@query(...)` — 11 hits
```parsley
@query(Users).where({email: x}) ?-> *                     // ✗ arrow outside the parens
@query(Authors | with posts | status == "p" ??-> *)       // ✗ filters the OUTER table
@insert(Users |< name: {u.name}, email: {u.email} ?-> *)  // ✗ two defects

@query(Users | email == {x} ??-> *)
@query(Authors | with posts(status == "p" | order created_at desc) ??-> *)
@insert(Users |< name: u.name |< email: u.email ?-> *)
```

Loud: `{expr}` applies to query *conditions* only — `|<` write values are plain expressions,
each needs its own `|<`, relation filters go in parens, and boolean columns need `== true`.

## 11. Zero-arity accessors are methods, not properties — 9 hits
```parsley
let r <=/= JSON(@https://api.example.com/x)
r.status   // ✗ silently null      r.ok   // ✗ silently null
doc.ast    // ✗ Dot notation can only be used on dictionaries, got mddoc

r.data()   r.response().status   r.response().ok   doc.ast()
```

Silent, and the worst entry here: a fetch result is a typed dict holding `__data`/`__format`/
`__response`, so `response.status` evaluates to `null` (verified against a live local server).

## 12. Basil-only code in standalone snippets — 9 hits
```parsley
<Page title="x"> … </Page>       // ✗ Undefined component: `Page`
p.public()   image(@./a.jpg)   @params.id   @DB   @SEARCH   request.query
```

Loud: Prelude components are module exports (`let {Page} = import @basil/html`) and the rest
needs the server, so on a Parsley page mark it `// Basil only — needs the running server`.

## 13. Double-quoted strings do not interpolate — 5 hits
```parsley
"Hello, {name}!"     // → the literal text  Hello, {name}!
`Hello, {name}!`     // → Hello, Alice!
'raw @{name}'        // raw strings use @{expr}
```

Silent: the braces render as themselves and nothing complains, so the fix is to change the
quotes, never the braces (`${x}` is never Parsley).

## 14. `{expr}` *does* interpolate inside a double-quoted attribute — 3 hits (new)
```parsley
<button onclick="fetch('/a', {method: 'POST'})">"Go"</button>   // ✗ unexpected ':'
<button onclick='fetch("/a", {method: "POST"})'>"Go"</button>   // ✓ raw single-quoted
<div class="user-{id}"></div>                                   // ✓ really does interpolate
```

Loud, and the exact inverse of #13: attribute values *are* interpolated, so an inline JS object
literal is a parse error — carry JS in a raw attribute, using `@{expr}` for dynamic values.

## 15. Integer division truncates — 4 hits
```parsley
5 / 2      // 2   — not 2.5
5 / 2.0    // 2.5 — one float operand gives a float
```

Silent: this Python-3/JS-ism survives review because the snippet still runs cleanly — only
the `// result` comment is wrong.

## 16. `import @path` is the only import form — 4 hits
```parsley
import {activeUsers} from @./data.pars   // ✗ Expected path after import
let {Page} = import(@./Page.pars)        // ✗ same — no call parentheses
{api} = import @std/api                  // ✗ and no bare destructuring either

let {activeUsers} = import @./data.pars
let api = import @std/api                // some modules ARE the object; check the page
```

Loud: destructuring cannot rename either (`let {search: s} = …` is a parse error — pull the
field off afterwards), and module scope is isolated, so an imported file needs its own imports.

## 17. A line starting `!` or `not` continues the previous line — 3 hits (new)
```parsley
let a = 1
!true              // ✗ Expected 'in' after 'not', got true
!(3 in xs)         // ✗ Expected 'in' after 'not', got (

let inverted = !true      // ✓ bind it
3 not in xs               // ✓ negated membership is `not in`
```

Loud but confusing: `!` and `not` lex to the same token, and after a complete expression the
parser reads `expr not …` as the `not in` operator — same family as #7.

## Smaller traps, one hit each
- `` `func main() {}` `` — a backtick string swallows `{}` as an empty interpolation; use double quotes for sample code containing braces.
- `let a = [...]` — ellipsis as a placeholder is a parse error (`unexpected '...'`), and reads as a JS spread.
- `{if x} … {/if}` — Handlebars/Liquid template tags are not Parsley; use `if (x) { … } else if (y) { … }`.
- `toInt(d["p"] ?? "1")` — `??`/`||` unparenthesised in a call argument is a parse error; bind first.
- `@2024-12-25T14:30` — datetime literals need seconds; `@14:30` (time-only) is fine.
- `fn double(x) { … }` — no named function declarations; use `let double = fn(x) { … }`.
- `{("key"): v}` — computed dictionary keys use square brackets, `{[k]: v}`; a literal key needs no wrapper.
- `var sum = 0` then `sum = sum + $5.00` — money and numbers cannot be added; seed with `$0`.
- `db <=??=> "SELECT …"` alone on a line — a statement cannot start `ident <operator>`; bind it.
- `x || "default"` — `||` is boolean OR and returns `true`/`false`; the coalescing operator is `??`.
- `#63in.fmt("ft-in")` — `fmt` silently ignores an unrecognised format string; the compound form is `.format("ft-in")`.

## Also worth knowing — patterns the audit never hit
The JS/Python habits the docs got right everywhere; all are loud parse errors.

- **Ternary**: `c ? a : b` → `let x = if (c) { a } else { b }` (`if` is an expression).
- **Strict equality**: `===` / `!==` → `==` / `!=`.
- **Arrow functions**: `x => x * 2` → `fn(x) { x * 2 }`.
- **Array mutation**: no `.push()`, `.append()`, `.concat()`, or `[...spread]` — arrays are
  immutable; join with `++` (note `"a" ++ "b"` builds an array; string join is `+`).
- **Python literals and keywords**: `True`/`False`/`None`/`elif` → `true`/`false`/`null`/`else if`
  (`and`/`or`/`not` do exist, alongside `&&`/`||`/`!`).
- **JS/Python loop forms**: `for (const x of y)` and `for x in y:` → `for (x in y) { … }`.
