---
title: Performance
---

# Performance

Is Basil fast enough? For most sites, **yes.** 

## How fast is Basil?

Basil is a Go web server with a built-in interpreter. Go serves static files, and does it as fast as you would expect. The Parsley interpreter runs dynamic pages, and that is where the time goes.

We have not published benchmarks yet. Treat these as rough figures from our own testing on a laptop:

| Kind of request | Roughly |
|---|---|
| Static file | Tens of thousands per second |
| Cached dynamic page | Tens of thousands per second |
| Uncached dynamic page | Hundreds to a few thousand per second |

That puts a Parsley page in the same league as PHP, Ruby, or Python.

## Why it is fast enough

Four things keep the common case quick.

**Code caching.** In production, Basil lexes and parses each `.pars` file once, keeps the syntax tree in memory, and reuses it for every request after that — and does the same for every file a page imports. Parsing is the expensive part of running a script, so skipping it matters. Dev mode spends a little of that back for freshness: an edit anywhere, including a component several imports down, shows up on the next request without a restart. See [Dev Tools](basil/dev-tools.html).

**Response caching.** Production mode can cache whole responses. Set a TTL and Basil serves the page from memory until it expires. Most pages on most sites can live with a cache of a minute or two. See [Deployment](basil/deployment.html).

**A database in the same process.** Basil ships with SQLite running inside the server. A query is a function call, not a network trip. More on this below.

**Nothing in between.** No separate application server, no proxy to the interpreter, no ORM. HTML *is* the language, so there is no template engine to run.

Smaller things help too: HTTP/2 and gzip are on by default, and Basil caches image transforms to disk on first request.

## Why a built-in database is faster

Most web scripting languages talk to a database that lives somewhere else. PHP, Ruby, and Python send each query over a socket to Postgres or MySQL, wait, and read the reply back. Every query pays for that round trip, and usually for a connection pool and an ORM on top.

Basil skips all of that. SQLite runs inside the Basil process, on a connection Basil opens at startup and keeps warm. A query is a function call. The rows come back as a Parsley Table, with no ORM in between.

So small queries are cheap, and a page that runs ten costs little more than one that runs one. The "N+1 query" problem that hurts other stacks mostly disappears. You can write the obvious code — find the user, then their orders, then each order's items — and it will be fine.

Reads are super cheap: SQLite in WAL mode lets many readers run while one writer writes. For the sites Basil is aimed at, that is the right trade. See [Database](basil/database.html).

## What it suits, and what it doesn't

Basil fits:

- Personal sites, blogs, and documentation.
- Small business and club sites with a database behind them.
- Internal tools, dashboards, and admin pages.
- Prototypes and side projects you want online today.
- Anything one modest server can carry.

Think first before using it for:

- Very busy sites with thousands of uncached dynamic requests per second. Basil scales out behind a load balancer with a shared session secret, but at that point a compiled stack may serve you better.
- Heavy computation inside a request. Parsley is a scripting language; long loops over large data run slower than in Go.
- Workloads that need Postgres-scale concurrency. SQLite excels at reads and handles moderate writes, but it is one file on one disk. Basil connects to [Postgres or MySQL](manual/features/database.html) when you outgrow it — and then you pay the round trip like everyone else.

In short: one Basil process on a small VPS will comfortably serve the kind of site most people actually build. If you are not sure whether that is you, *it probably is.*
