---
id: man-pars-globals
title: Globals
system: parsley
type: fundamentals
name: globals
created: 2026-07-15
version: 0.2.0
author: Basil Team
keywords:
  - globals
  - env
  - args
  - environment variables
  - command-line arguments
---

# Globals

Every Parsley program runs with a couple of ambient globals already in scope — no `import` required. They give a script the process environment (`@env`) and its command-line arguments (`@args`). Both are available wherever Parsley runs: the `pars` CLI, the REPL, and inside [Basil](../../../basil/manual/index.md) server handlers.

> These are the only two globals the language itself provides. The Basil server injects a few more that are request-scoped (`@params`) or connection-scoped (`@DB`, `@SEARCH`) — see [Server Globals](../../../basil/manual/globals.md).

## `@env`

Environment variables, as a dictionary.

**Type:** `Dictionary`

```parsley
@env.HOME                       // "/Users/alice"
@env["DATABASE_URL"]            // connection string
@env.PATH                       // system PATH
```

The dictionary is populated once, when the environment is created (startup), from the process environment. It is read-only — you read configuration from it, you don't write to it. Keys are held in sorted order, so iterating `@env` is deterministic.

Missing keys follow the usual dictionary rules — access returns `null`, so supply a default with `??`:

```parsley
let port = @env.PORT ?? "8080"
```

## `@args`

Command-line arguments, as an array of strings.

**Type:** `Array` of `String`

```parsley
// pars script.pars Alice Bob
@args[0]                        // "Alice"
@args[1]                        // "Bob"
@args.length()                  // 2
```

`@args` carries the arguments passed after the script name (or after the expression with `-e`). It is primarily useful in `pars` CLI scripts — see the [CLI page](../features/cli.md#command-line-arguments) for how arguments are passed. In a Basil server handler `@args` is typically empty: a request handler is invoked by the server, not from a command line.

## See Also

- [Command Line Interface](../features/cli.md) — how `pars` passes arguments to `@args`
- [Server Globals](../../../basil/manual/globals.md) — the request-scoped globals Basil adds (`@params`, `publicUrl()`, CSRF)
- [Dictionaries](../builtins/dictionary.md) — working with `@env`
- [Arrays](../builtins/array.md) — working with `@args`
