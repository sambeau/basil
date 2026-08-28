---
title: About
---

## What is this?

- **Parsley** is a scripting language
- **Basil** is a web server that runs Parsley code

When put together, Parsley runs as a server-side programming language for building websites. It does not compile to Javascript, nor run in the browser. Though you can make client-side components with it (see [Parts](https://herbaceous.net/basil/parts.html)).

## Versions

Basil & Parsley currently share a version number:
- **1.0.0-alpha.7**.

See [changelog](https://github.com/sambeau/basil/blob/main/CHANGELOG.md) for recent changes.

## How stable is it?

Despite the shared version number, consider:
- **Parsley** to be **Beta**, 
- **Basil** to be **Alpha**.

**Parsley** (the language) has been stable for a while now: the syntax and standard library have settled down, and changes at this point are mostly fixes and small additions—though, every time I think I’m done, I find another error page to fix.

**Basil** (the server) is younger, and while it works well, some of its edges are still being sanded — configuration, dev-tools, deployment, and error handling in particular are still being worked on.

## License

Basil & Parsley are open source under the **MIT license**. Fill your boots.

## Where's the code?

On GitHub, at [github.com/sambeau/basil](https://github.com/sambeau/basil). 

## What are they written in?

**[Go.](https://go.dev)** It’s plenty fast for this sort of thing.

## Do I need Basil to use Parsley?

No. The `pars` command runs Parsley scripts on the command line, no server required. 

## Why do they exist?

Mostly for fun — see [Why?](why.html) for the full story.

## Found a bug?

Please tell us! We'd gratefully receive bug reports of any kind on the [GitHub issue tracker](https://github.com/sambeau/basil/issues).

We’re *especially* keen to hear about errors and error pages—good error messages are something we care a lot about. If Parsley ever gives
you an error message that's confusing, unhelpful, or points to the wrong line (the most common issue I’ve been chasing), then that's a bug. 

## Found a mistake in the docs?
Please tell us! We'd gratefully receive bug reports about Docs [GitHub issue tracker](https://github.com/sambeau/basil/issues).

Let us know where things are unclear or missing. We’ve tried to be thorough, but I’m sure we’ll have missed something.


## Roadmap

Currently there is no fixed roadmap. Getting to 1.0 is all that’s planned. 

However, we do have thoughts about what we’d like to work on next. But only once we’ve built a few more things with it.

Here’s what I’m thinking about for **Basil**:
- Production Admin tools:
	- Database tools
	- Basic Git tools
	- User management
	- Basic Analytics
- HTML library (maybe):
	- Drag & drop module
	- Text editor module

Here’s what I’m thinking about for **Parsley**:

- Improved formatting for values and tables
- Rest API module (maybe Basil-only)
	- Schema, record and Table integration 
	- Automatic APIs
	- Pager (next/prev)
- Improved Locale support
