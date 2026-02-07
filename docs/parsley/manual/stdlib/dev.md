---
id: man-pars-std-dev
title: "@std/dev"
system: parsley
type: stdlib
name: dev
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - dev
  - debug
  - logging
  - development
  - server
---

# @std/dev

Development logging utilities for debugging Basil server handlers. Log output is visible at the configured dev log route (typically `/_dev/log`).

> ⚠️ The dev module requires Basil server context. In standalone Parsley scripts, all methods are no-ops (they silently return null).

```parsley
let {dev} = import @std/dev
```

## Methods

| Method | Args | Description |
|---|---|---|
| `dev.log(value)` | any | Log a value |
| `dev.log(label, value)` | string, any | Log a value with a label |
| `dev.log(label, value, opts)` | string, any, dictionary | Log with label and options |
| `dev.clearLog()` | none | Clear all log entries for the current route |
| `dev.logPage(route, value)` | string, any | Log to a specific route's log |
| `dev.logPage(route, label, value)` | string, string, any | Log to a specific route with label |
| `dev.setLogRoute(route)` | string | Set the default log route |
| `dev.clearLogPage(route)` | string | Clear log entries for a specific route |

## Logging

### Basic Logging

```parsley
dev.log(someVariable)
dev.log("user", currentUser)
dev.log("request params", req.params)
```

### Log Levels

Pass an options dictionary with a `level` key to set the log level:

```parsley
dev.log("warning message", someValue, {level: "warn"})
```

Supported levels: `"info"` (default), `"warn"`.

### Route-Scoped Logging

By default, logs are associated with the current handler's route. Use `dev.logPage` to log to a different route's log, or `dev.setLogRoute` to change the default:

```parsley
dev.logPage("admin", "audit", actionDetails)

dev.setLogRoute("dashboard")
dev.log("widget data", data)     // logs to "dashboard" route
```

Route names must be alphanumeric (letters, digits, hyphens, underscores).

### Clearing Logs

```parsley
dev.clearLog()                   // clear current route's logs
dev.clearLogPage("admin")        // clear a specific route's logs
```

## Common Patterns

### Handler Debugging

```parsley
let {dev} = import @std/dev

fn handleRequest(req) {
    dev.log("params", req.params)
    let user = findUser(req.params.id)
    dev.log("found user", user)
    // ...
    return response
}
```

### Conditional Logging

Since dev methods are no-ops outside server context, you don't need to guard calls:

```parsley
// Safe in both server and standalone contexts
dev.log("debug", someValue)
```

## See Also

- [Error Handling](../fundamentals/errors.md) — runtime error handling
- [@std/api](api.md) — HTTP API utilities