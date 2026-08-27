---
id: man-bas-routing
title: "Routing"
system: basil
type: feature
name: routing
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - routing
  - site mode
  - routes
  - handlers
  - index.pars
  - static
  - subpath
  - params
---

# Routing

Basil has two ways to map URLs to handlers: **site mode** (the filesystem is the router) and **explicit routes** (listed in `basil.yaml`). A project can use either or both.

## Site Mode

Point `site.path` at a directory and files in it become routes:

```yaml
site:
  path: ./site
```

```
site/
├── index.pars            → /
├── about/
│   └── about.pars        → /about
└── blog/
    └── index.pars        → /blog, /blog/anything…
```

### How a request finds its handler

For a request to `/blog/2024/hello`, Basil walks back from the deepest path segment looking for a handler:

1. `site/blog/2024/hello/hello.pars`, then `site/blog/2024/hello/index.pars`
2. `site/blog/2024/2024.pars`, then `site/blog/2024/index.pars`
3. `site/blog/blog.pars`, then `site/blog/index.pars` ✓
4. …up to `site/index.pars`

Two conventions, checked in order:

- **Folder-named file** — `blog/blog.pars` serves `/blog`. Easier to tell apart in your editor when you have many folders open.
- **`index.pars`** — the classic. `blog/index.pars` works the same way.

### Subpaths

The portion of the URL the handler didn't consume is its **subpath** — so `site/blog/index.pars` handles `/blog/2024/hello` with subpath `/2024/hello`. One handler can render a whole section, React-router-style, by dispatching on it.

### Query & form parameters

`@params` merges URL query parameters and POST form data (form wins on conflicts):

```parsley
// /search?q=basil
@params.q          // "basil"
```

## Explicit Routes

List routes in `basil.yaml` when you want control over paths, per-route auth, or caching:

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /dashboard
    handler: ./handlers/dashboard.pars
    auth: required        # See Authentication
  - path: /api/*
    handler: ./handlers/api.pars
```

## Static Files

Two ways to serve them:

**`public_dir`** (the usual way) — files under it are served at the web root, and Parsley paths are rewritten to URLs automatically:

```yaml
public_dir: ./public
```

```parsley
<img src={@./public/images/logo.png}/>
// renders as <img src="/images/logo.png"/>
```

**`static` routes** — mount any directory at any prefix:

```yaml
static:
  - path: /static/
    root: ./public
```

## Asset Bundling

Basil automatically bundles every `.css` and `.js` file from your handlers directory into `/__site.css` and `/__site.js` (concatenated depth-first alphabetically, cache-busted). Include them with the built-in tags:

```parsley
<head>
    <Css/>
    <Script/>
</head>
```

Both tags render nothing at all when there is no bundle — no files of that
type, or no bundler in this context — so they are safe to leave in a layout
before you have written any CSS.

`<CSS/>` and `<Javascript/>` work as aliases; `<Css/>` and `<Script/>` are the
names to use.

## See Also

- [Configuration](configuration.md) — `site`, `routes`, `static`, `public_dir`
- [Authentication](authentication.md) — protecting routes
- [Parts](parts.md) — routes that return interactive fragments
