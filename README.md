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
- **Rich literals** — These are all native types:
    - dates (`@2024-01-15`),
    - durations (`@2h30m`),
    - money (`$99.99`),
    - paths (`@./config.json`),
    - URLs (`@https://api.example.com`),
    - regex (`/pattern/`)
- **Declarative I/O** — read/write files, query databases, and fetch URLs with operators rather than method chains
- **Data Formats** – slurp CSV in; spit JSON or Markdown out
- **Schemas and records** — define data shapes, validate input, and bind forms with minimal ceremony
- **Batteries included** — string manipulation, date arithmetic, table queries, CSV, JSON, Markdown, SQL, search, SFTP, Git and more without importing anything

**Examples:**

```parsley
// A simple page component

let Page = fn({title, users}) {
    <html>
        <head><title>title</title></head>
        <body>
            <h1>title</h1>
            <ul>
                for (user in users) {
                    {name, email} = user 
                    <li>
                        <b>name +": "</b>email
                    </li>
                }
            </ul>
        </body>
    </html>
}
```

Using an array of dictionaries:

```parsley
 <Page title="Active Users" users={[{name:"Robert Foo", email:"foo@example..com"}]}/>
```

Outputs:

```html
<html><head><title>Active Users</title></head><body><h1>Hello</h1><ul><li><b>Robert Foo: </b>foo@example.com</li></ul></body></html>
```

From a CSV file:

```CSV
name,email
Luke,luke@example.com
Leia,leia@example.com
Han,han@example.com
Chewy,chewy@example.com
```

Load and show:

```parsley
emailList <== CSV(@/path/to/email-list.csv)
<Page title="Active Users" users={emailList}/>
```

Outputs:
```html
<html><head><title>Active Users</title></head><body><h1>Active Users</h1><ul><li><b>Luke: </b>luke@example.com</li><li><b>Leia: </b>leia@example.com</li><li><b>Han: </b>han@example.com</li><li><b>Chewy: </b>chewy@example.com</li></ul></body></html>
```

``emailList`` is loaded in as a table:

```parsley
emailList.toBox()
┌───────┬───────────────────┐
│ name  │ email             │
├───────┼───────────────────┤
│ Luke  │ luke@example.com  │
│ Leia  │ leia@example.com  │
│ Han   │ han@example.com   │
│ Chewy │ chewy@example.com │
└───────┴───────────────────┘
``` 

Loading from database:

```parsley
// Query a database and render the result

let users = @query(
    Users 
    | status == "active" 
    ??-> name, email) // ?? means output a table

<Page title="Active Users" users={users}/>
```

``Users`` is a binding of a ``User`` schema to a database table:

```parsely
@schema User {
    id: int
    name: string
    email: string
    status: string
}

let db = @sqlite(":memory:")       // or a file
db.createTable(User, "users")      // if not already created
let Users = db.bind(User, "users") // 'Users' can now be queried
```

### The `pars` CLI

The `pars` command runs Parsley scripts and includes an interactive REPL for experimentation:

```bash
pars script.pars          # Run a script
pars -e '@now + @7d'      # Evaluate an expression
pars                      # Start the REPL
```

---

## Basil Server

Basil is a web server that runs Parsley handlers. Drop a `.pars` file in a directory and it becomes a route. Single binary install; almost no configuration; no build step.

**Core features:**

- **File-based routing** — or a react-like single file handler
- **Built-in SQL Database** — in-process, super-fast, SQLite database 
- **Hot reload** — edit a handler and your browser refreshes
- **Parts** – dynamic components from reloadable HTML fragments
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
