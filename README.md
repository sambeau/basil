# Basil & Parsley

**Basil** is a web server. **Parsley** is a programming language. Together they make building web applications surprisingly pleasant.

> ⚠️ **Work in Progress** — This repository has been made public so I can use the [tree-sitter grammar](https://github.com/sambeau/tree-sitter-parsley) in my editor. A proper launch with documentation and a website is coming soon at [herbaceous.net](http://herbaceous.net).
>
> **Please don't post this to Hacker News yet!** This has been months of work and I'd like to present it properly when it's ready. That said, if you've stumbled across this and want to poke around, you're very welcome — the [language manual](docs/parsley/manual/index.md) is pretty much complete.

---

## Parsley Language

Parsley is an expression-oriented scripting language designed for munging data and building web applications. It aims to be expressive, powerful, familiar … and fun.

**Core features:**

- **Everything is an expression** — `if`, `for`, and `try` all return values
- **First-class HTML** — JSX/PHP-like tag syntax is built into the language, not bolted on
- **Rich literals** — dates (`@2024-01-15`), durations (`@2h30m`), money (`$99.99`), paths (`@./config.json`), URLs (`@https://api.example.com`), and regex (`/pattern/`) are all native types
- **Declarative I/O** — read/write files, query databases, and fetch URLs with operators rather than method chains
- **Data Formats** – slurp CSV in; spit JSON or Markdown out
- **Schemas and records** — define data shapes, validate input, and bind forms with minimal ceremony
- **Batteries included** — string manipulation, date arithmetic, table queries, CSV, JSON, Markdown, SQL, search, SFTP, Git and more without importing anything

```parsley
// A simple page component
let Page = fn({title, items}) {
    <html>
        <head><title>title</title></head>
        <body>
            <h1>title</h1>
            <ul>
                for (item in items) {
                    <li>item</li>
                }
            </ul>
        </body>
    </html>
}

// Query a database and render the result
let db = @sqlite("app.db")
let users = db <=??=> "SELECT * FROM users WHERE active = true"

<Page title="Active Users" items={users.map(fn(u) { u.name })}/>
```

### The `pars` CLI

The `pars` command runs Parsley scripts and includes an interactive REPL for experimentation:

```bash
pars script.pars          # Run a script
pars -e 'log(@now)'       # Evaluate an expression
pars                      # Start the REPL
```

---

## Basil Server

Basil is a web server that runs Parsley handlers. Drop a `.pars` file in a directory and it becomes a route. Single binary install; almost no configuration; no build step.

**Core features:**

- **File-based routing** — or a react-like single file handler
- **Hot reload** — edit a handler and your browser refreshes
- **Built-in authentication** — sessions, users, roles, API keys, and ***Passkeys!***
- **Full-text search** — point it at a directory and query your content
- **Built-in Git server** — push-to-deploy
- **Development tools** — database inspector, request logging, and web-based error pages

Basil is still under active development and not *quite* ready for public use.

---

## Embedding Parsley

Parsley can be embedded in any Go application with just a few lines:

```go
import "github.com/sambeau/basil/pkg/parsley"

result, err := parsley.Eval(`"Hello, " ++ name ++ "!"`, parsley.Env{
    "name": "World",
})
```

See the [embedding documentation](pkg/parsley/README.md) for details.

---

## Current Status

- **Parsley**: Final stages of 1.0 alpha release
- **Basil**: Work in progress, coming *very* soon

---

## Want to Know More?

If you're interested in how Parsley and Basil were designed and built, I'd be happy to chat. Feel free to contact me.

---

## License

MIT