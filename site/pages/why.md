---
title: Why?
---

# Why does this exist?

Because making websites used to be *fun*, and I wanted it back.

## The itch

I started building for the web in the early 2000s. You'd write a file, push it to a server, and hit refresh. That was the whole workflow. View Source was the documentation. You could go from idea to a thing your friends could visit in an afternoon, and the distance between "I wonder if…" and "look what I made!" was about as short as it has ever been in the history of making things.

Somewhere between then and now, we buried that under toolchains. To put a paragraph on the internet today you're expected to install a package manager, configure a bundler, pick a framework, annotate your types, and wait for a compile step — and half of it will be deprecated by autumn. All of that machinery exists for good reasons at Google scale. Most of us are not Google. Most of us just want to make a website.

So I spent a couple of years building the tool I wished existed: **Parsley**, a small language where HTML is part of the grammar, and **Basil**, the server it lives in. One binary. No compiler, no npm, no build step. Write a file, save, look.

## What it is

**Parsley** is an expression-oriented scripting language for munging data and making pages. Everything returns a value — `if`, `for`, even `try`. HTML tags are expressions too, so a component is just a function that returns markup. Reading a CSV file into a table is one line. So is querying a database, or fetching a URL.

**Basil** is the web server. Drop a `.pars` file in a folder and it's a route. It has the things a real site eventually needs already inside it — an SQL database, sessions, users and passkeys, full-text search, an image server, a Git server you can push to deploy with — so "I'll just add login" doesn't turn into a weekend of integration work.

## About those quirks

If you skim the manual you'll notice Parsley does some things differently. None of it is different for the sake of it — every quirk is there for a reason. Sometimes the reason is hard-nosed computer science; sometimes the reason is that it made me smile.

**Dates, money, durations, paths, and URLs are literals.** You write `@2024-01-15`, `$19.99`, `@7d`, `@./config.json`. The comp-sci reason: these are the values web apps actually traffic in, and giving them real types catches whole families of bugs — money is exact arithmetic, not floating point, so `$0.10 + $0.20` is thirty cents, full stop. The human reason: look how nice they are to type.

**Reading a file is an arrow.** `let users <== CSV(@./users.csv)` — the data visibly flows from the file into the variable. I/O is one of the most consequential things a program does; it deserves to be visible at a glance, not hidden in a method chain.

**HTML is grammar, not strings.** Tags are parsed, nested, and checked by the language itself. There is no template dialect to learn and no escaping dance — and a component is just `let Card = fn({title}) { <div>…</div> }`.

**Everything is an expression.** The last value in a file is what a page renders. An `if` returns its branch. It sounds academic until you notice your handlers have almost no plumbing in them.

**Batteries included, imports optional.** CSV, JSON, Markdown, SQL, string tools, date arithmetic — built in. The early web felt fast because PHP shipped with everything; Parsley takes that lesson seriously.

## Honestly, though

This is early days. Parsley is in its 1.0 alpha; Basil is close behind. It's the work of one person having a very good time, which means it moves fast but it isn't Google-scale battle-tested, and you'll find rough edges. When you do, [tell me](https://github.com/sambeau/basil/issues) — early feedback shapes a language more than almost anything else.

If any of this sounds like the web you remember — or the web you wish you'd been there for — [give it ten minutes](get-started.html). Worst case, you spent ten minutes and got a working website out of it.

— Sam
