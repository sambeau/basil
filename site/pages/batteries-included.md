---
title: Batteries Included
---

# Batteries Included

Basil and Parsley come with the tools you need to build a website. 

The hard parts — databases, forms, login, images, live updates — work out of the box with little or no configuration. When you come to build something tricky, we have, hopefully, already thought of what you need and included it.

## What Parsley gives you

Most of what makes a website awkward is data: reading it, checking it, storing it, showing it. Parsley puts that in the language rather than in a library.

#### Types for real things
Dates, times, durations, money, and units are values, not strings. `@2024-01-15 + @1w` is a date. `$19.99 * 3` is money, rounded correctly. `#5ft + #10in` is a length. You write them directly and print them properly. See [dates and times](manual/builtins/datetime.html), [money](manual/builtins/money.html), and [units](manual/builtins/units.html).

#### Data formats
CSV, JSON, and Markdown are built in. Read a CSV file and you get a table with numbers as numbers. Read a Markdown file and you get HTML plus its frontmatter as a dictionary. Write any of them back out with one method. See [data formats](manual/features/data-formats.html).

#### Table data type
Filter, sort, group, and total rows with `.where()`, `.orderBy()`, `.groupBy()`, and `.sum()`. The same methods work on a CSV file, a database result, and a list you typed in. Output as HTML, CSV, Markdown, or JSON. See [tables](manual/builtins/table.html).

#### Schemas and records
A `@schema` declares your data's shape once. From it, Parsley validates input, creates the database table, and renders the form. A record carries its data and its errors together, so a form can show what went wrong next to the field that went wrong. See [the data model](manual/fundamentals/data-model.html).

#### Forms from schemas
Bind a form to a record with `@record` and its fields with `@field`. Labels, input types, placeholders, and error messages come from the schema. Submit, validate, re-render with errors, save — the whole loop without writing it by hand. See [tags](manual/fundamentals/tags.html) and [records](manual/builtins/record.html).

#### A query DSL
Read and write database rows with `@query`, `@insert`, `@update`, and `@delete` instead of SQL strings. The DSL generates parameterised SQL for you. When you want SQL, the `<SQL>` tag is there and is parameterised too. See [the query DSL](manual/features/query-dsl.html).

#### HTML as part of the language
A page is a function that returns markup. No template engine, no separate syntax. See [tags](manual/fundamentals/tags.html).

## What Basil gives you

Basil runs Parsley files as pages and comes with the services a site usually has to bolt on.

#### A built-in database
SQLite runs inside the server. Every handler can query it as `@DB` with no setup. A schema becomes a table. Dev mode includes a browser-based inspector for browsing, querying, and importing CSV. See [Database](basil/database.html).

#### Parts
Parsley runs on the server only. Parts add the dynamic layer: a section of the page that re-renders on the server and swaps in place without a full reload, in the spirit of Hotwire or htmx. You write them in Parsley. See [Parts](basil/parts.html).

#### Authentication and sessions
Passkey login with WebAuthn, users, roles, protected paths, recovery codes, and API keys. Sessions with key-value storage and flash messages. CSRF protection is automatic. See [Authentication](basil/authentication.html) and [Sessions](basil/session.html).

#### Images
Resize, crop, blur, and serve responsive `srcset`s from one source file. Basil transforms on first request and caches the result. See [Images](basil/images.html).

#### JavaScript and CSS bundling
Basil gathers every `.css` and `.js` file in your handlers directory into one cache-busted stylesheet and one script. Include them with a tag. No bundler to configure. See [Routing](basil/routing.html#asset-bundling).

#### Search
Full-text search over your own content, with ranking and a query syntax. See [Search](basil/search.html).

#### HTML components
Accessible form, navigation, and media components, ready to drop in. See [@basil/html](basil/html.html).

#### Live reload
In dev mode, save a file and the browser refreshes itself. The loop is edit, save, look. See [Dev Tools](basil/dev-tools.html).

#### A dev log panel
Log a value from any handler with `dev.log()` and it appears in a panel in the browser, not in your HTML. The calls do nothing in production, so you can leave them in. See [Dev Tools](basil/dev-tools.html#the-dev-log).

#### Readable error pages
A failing handler in dev mode shows the error class, the source line with a caret, and a hint. Production shows a plain page and writes the detail to the logs. See [Dev Tools](basil/dev-tools.html#error-pages).

#### HTTPS and deployment
Basil fetches and renews Let's Encrypt certificates itself. Push to its built-in Git server and the site reloads. See [Deployment](basil/deployment.html) and [Git Deploy](basil/git.html).

## Why this matters

Every one of these is something you would otherwise choose, install, configure, and keep up to date. A typical stack needs a database driver, an ORM, a validation library, a form library, an auth service, an image pipeline, a bundler, a dev server, and a deploy script — from nine different projects, with nine different ideas about how things should work.

Basil and Parsley are one project with one idea. The schema that validates your form is the schema that made your table. The table that came out of a CSV file answers to the same methods as the one that came from the database. The server that serves the page is the one that logs the error, bundles the CSS, and fetches the certificate.

That is what we mean by batteries included: you open the box and everything you need should be there.
