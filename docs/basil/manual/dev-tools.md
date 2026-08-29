---
id: man-bas-dev-tools
title: "Dev Tools"
system: basil
type: feature
name: dev-tools
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - dev
  - development
  - hot reload
  - live reload
  - error pages
  - logging
  - inspector
  - dev.log
---

# Dev Tools

Run `basil --dev` and the server turns on its development conveniences: live reload, detailed error pages, a log panel, and a database inspector.

## Live Reload

Every page served in dev mode polls for changes; save a handler and the browser refreshes itself. Combined with no build step, the loop is: edit → save → look.

## Error Pages

In dev mode, a failing handler renders a detailed error page — the error class and code, the source line with a caret, and hints. In production, visitors get a generic error page instead (details go to the logs). The generic pages can be replaced with your own via the [`error_pages` config](configuration.md#error-pages).

## The Dev Log

Log from any handler with `@basil/log` — entries land in the dev panel, not your HTML:

```parsley
let {dev} = import @basil/log

dev.log(someValue)
dev.log("user", currentUser)
dev.log("uh oh", err, {level: "warn"})   // "info" (default) | "warn"
```

Every `dev.*` function is a **no-op in production** and in plain `pars`, so it's safe to leave the calls in.

Dev logs are stored in SQLite (configurable):

```yaml
dev:
  log_database: ./logs/dev_logs.db
  log_max_size: 10MB
  log_truncate_pct: 25
```

## Database Inspector

Dev mode serves a web UI at `/__/db` for the [built-in database](database.md): browse tables, run ad-hoc queries, download tables as CSV, and upload CSVs back.

## Request Logging

Requests are logged to the configured output (`logging.output`); use `--quiet` to silence them. Handler `log()` output goes to `logging.parsley.output`.

## Dev Mode vs Production

| | `basil --dev` | `basil` |
|---|---|---|
| Protocol | HTTP on localhost | HTTPS (see [Deployment](deployment.md)) |
| Live reload | ✓ | — |
| Error pages | Detailed | Generic |
| Response caching | Off (`dev.cache` turns it on) | On |
| Edits to handlers and components | Picked up on the next request | On the next deploy or restart |
| Dev log & inspector | ✓ | — |

## See Also

- [Running Basil](running.md) — flags and profiles
- [Configuration](configuration.md) — the `dev` and `logging` sections
