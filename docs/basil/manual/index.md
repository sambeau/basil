---
id: man-bas-index
title: "Basil Server Manual"
system: basil
type: tutorial
name: index
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - manual
  - index
  - basil
  - server
---

# Basil Server Manual

Basil is a web server that runs [Parsley](../../parsley/manual/index.md) handlers. Drop a `.pars` file in a directory and it becomes a route. One binary, almost no configuration, no build step — with a database, authentication, search, and an image server already inside.

> **New to Basil?** The [Get Started tutorial](https://herbaceous.net/get-started.html) takes you from install to a working site in ten minutes.

## The Server

| Page | Description |
|------|-------------|
| [Running Basil](running.md) | Install, `--init`, dev mode, CLI commands, and signals |
| [Configuration](configuration.md) | The `basil.yaml` file — every section explained |
| [Routing](routing.md) | Site mode, folder-named handlers, explicit routes, and static files |

## Features

| Page | Description |
|------|-------------|
| [Database](database.md) | The built-in SQLite database, `@DB`, and the database inspector |
| [Authentication](authentication.md) | Passkey login, users, roles, protected paths, and API keys |
| [Session Management](session.md) | `basil.session` — key-value storage and flash messages |
| [Parts](parts.md) | Interactive components that update without page reloads |
| [Parts JavaScript API](parts-js.md) | Script your Parts — `window.Parts`, events, and cross-Part targeting |
| [Search](search.md) | Full-text search over your content with `@SEARCH` |
| [Images](images.md) | Image transformation, smart crop, and responsive srcsets |
| [Git Deploy](git.md) | Push-to-deploy over HTTPS with the built-in Git server |

## Guides

In-depth walkthroughs of Basil's larger features.

| Page | Description |
|------|-------------|
| [The Authentication Guide](authentication-guide.md) | Auth in depth — sessions, recovery, email verification, and CSRF |
| [The Parts Guide](parts-guide.md) | Parts in depth — nesting, lazy loading, loading states, and error handling |
| [The Search Guide](search-guide.md) | Search in depth — indexing, ranking, query syntax, and manual documents |

## Handler API

Globals, modules, and objects available inside server handlers.

| Page | Description |
|------|-------------|
| [Server Globals](globals.md) | `@params`, `@env`, `@args`, `publicUrl()`, and the CSRF token |
| [@basil/http](http.md) | The request/response context — `request`, `response`, `route`, `method` |
| [@basil/auth](auth.md) | The per-request `session`, `auth`, and `user` |
| [@basil/api](api.md) | Auth wrappers, error helpers, and redirect for handlers |
| [@basil/log](log.md) | Development logging utilities for handlers |
| [@basil/html](html.md) | Accessible HTML form and UI components |

## Development & Production

| Page | Description |
|------|-------------|
| [Dev Tools](dev-tools.md) | Hot reload, the dev log panel, and error pages |
| [Deployment](deployment.md) | Putting a site on a server: the two Git workflows, HTTPS, rolling back, Fly.io |

## Reference

- [Basil Server Reference](../reference.md) — a map of every server-side module, global, and function, with a feature-availability matrix
- [Parsley Language Manual](../../parsley/manual/index.md) — the language itself
