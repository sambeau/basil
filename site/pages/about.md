---
title: About
---

The important facts about Basil & Parsley, all in one place.

## Versions

The current version of Basil & Parsley is **1.0.0-alpha.7**.

Basil and Parsley live in the same repository and are released together, so they
share a version number. New versions are cut whenever there's something worth
releasing — see the
[changelog](https://github.com/sambeau/basil/blob/main/CHANGELOG.md) for what's
changed and when.

## How stable is it?

We consider **Parsley to be Beta** and **Basil to be Alpha** — despite the shared
version number.

Parsley (the language) has been stable for a while now: the syntax and standard
library have settled down, and changes at this point are mostly fixes and small
additions. Basil (the server) is younger, and while it works well, some of its
edges are still being sanded — configuration, deployment, and error handling in
particular are still earning their stripes.

## License

Basil & Parsley are open source under the **MIT license**. Use them for anything
you like.

## Where's the code?

On GitHub, at
[github.com/sambeau/basil](https://github.com/sambeau/basil). Everything is in
there: the language, the server, the docs, and the script that builds this very
website.

## What are they written in?

**Go.** Both of them. That's why Basil ships as a single binary with no
dependencies — Go compiles everything, including Parsley, into one file you can
drop on a server and run.

## Do I need Basil to use Parsley?

No. Parsley runs perfectly well on its own — the `pars` command runs Parsley
scripts straight from the command line, no server required. It's handy for
munging data, generating files, and general scripting. In fact, this very
website is a pile of static HTML built by a Parsley script.

Basil is what you reach for when you want to *serve* Parsley pages over HTTP.

## Why do they exist?

There's a whole page about that — see [Why?](why.html) for the full story of
where Basil & Parsley came from and why you might (or might not) want to use
them.

## Found a bug?

Please tell us! We'd gratefully receive bug reports of any kind on the
[GitHub issue tracker](https://github.com/sambeau/basil/issues) — and we're
*especially* keen to hear about errors and error pages. If Parsley ever gives
you an error message that's confusing, unhelpful, or just plain wrong, that's a
bug. Good error messages are something we care a lot about getting right.

## Roadmap

*Coming soon.*
