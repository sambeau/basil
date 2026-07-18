<img src="server/prelude/public/logos/basil-logo-shiny.svg" alt="" width="34" align="left"/>

# Basil & Parsley

**Basil** is a web server. **Parsley** is a programming language. Together they make building web applications surprisingly pleasant — no compiler, no npm, no build step, one small binary.

**📖 Website & documentation: [herbaceous.net](https://herbaceous.net)**

> ⚠️ **Early days** — Parsley and Basil are in 1.0 alpha. Things work (and are tested), but expect rough edges and the occasional breaking change before 1.0. **Please don't post this to Hacker News just yet** — a proper launch is coming, and I'd like to present it properly.

---

## Install

```bash
curl -fsSL https://herbaceous.net/install.sh | sh
```

Installs `basil` (the server) and `pars` (the language CLI & REPL). macOS, Linux, and Windows; Apple Silicon and Intel. Or grab a binary from [releases](https://github.com/sambeau/basil/releases).

## What does it look like?

A CSV file becoming a web page — this is the whole program:

```parsley
let Page = fn({title, users}) {
    <html>
        <body>
            <h1>title</h1>
            <ul>
                for (user in users) {
                    <li><b>user.name</b> " — " user.email</li>
                }
            </ul>
        </body>
    </html>
}

let emailList <== CSV(@./email-list.csv)
<Page title="Active Users" users={emailList}/>
```

```bash
pars --raw users.pars
```

And a working website in three commands:

```bash
basil --init myapp
cd myapp
basil --dev        # edit site/index.pars, save, watch the browser refresh
```

## Parsley, the language

An expression-oriented scripting language for munging data and building pages:

- **Everything is an expression** — `if`, `for`, and `try` all return values
- **First-class HTML** — tags are grammar, not templates; components are just functions
- **Rich literals** — dates (`@2024-01-15`), durations (`@2h30m`), money (`$99.99`, exact arithmetic), paths (`@./config.json`), URLs, regex
- **Declarative I/O** — `let users <== CSV(@./users.csv)` reads a file into a table; `<=/=` fetches over the network
- **Batteries included** — strings, dates, tables, CSV/JSON/Markdown, SQL, search, and more without importing anything

→ [Get started in ten minutes](https://herbaceous.net/get-started.html) · [Parsley manual](https://herbaceous.net/manual/index.html)

## Basil, the server

Drop a `.pars` file in a directory and it's a route:

- **File-based routing** with hot reload — edit, save, the browser refreshes
- **Built-in SQL database** (in-process SQLite) with a web inspector
- **Authentication built in** — sessions, users, roles, API keys, and passkeys
- **Full-text search**, an **image server** (resize, smart crop, WebP, srcsets), and a **Git server** for push-to-deploy
- **Production-ready HTTPS** — automatic Let's Encrypt certificates

→ [Basil manual](https://herbaceous.net/basil/index.html)

## Embedding Parsley

Parsley embeds in any Go application via the `pkg/parsley/parsley` package:

```go
import "github.com/sambeau/basil/pkg/parsley/parsley"

result, err := parsley.Eval(`"Hello, " + name + "!"`, parsley.WithVar("name", "World"))
if err != nil {
    // handle error
}
fmt.Println(result.Value) // "Hello, World!"
```

See the [embedding documentation](pkg/parsley/README.md). That package is the
supported public API; the other `pkg/parsley/*` packages are implementation
detail — see [API stability](pkg/parsley/README.md#api-stability).

## Why?

Because making websites in the early 2000s was *fun* — write a file, refresh, smile — and somewhere along the way we buried that under toolchains. This is an attempt to dig it back up. [The whole story](https://herbaceous.net/why.html).

## License

MIT
