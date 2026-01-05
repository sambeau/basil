# Folder-Named Index Example

This example demonstrates the new folder-named index convention where:
1. `{foldername}/{foldername}.pars` is checked first
2. `{foldername}/index.pars` is used as fallback

This makes it easier to identify files in your editor when working with multiple folders.

## Structure

```
site/
├── index.pars              → handles / (root)
├── edit/
│   └── edit.pars          → handles /edit/ (folder-named)
├── admin/
│   ├── admin.pars         → handles /admin/ (takes precedence)
│   └── index.pars         → ignored (folder-named file exists)
└── posts/
    ├── index.pars         → handles /posts/ (no folder-named file)
    └── post.pars          → not used (doesn't match folder name)
```

## Why?

When you have many folders open in an editor, tabs like:
- `admin.pars`
- `edit.pars`
- `posts/index.pars`

...are much clearer than having 10 tabs all named `index.pars`!

## Configuration

```yaml
# basil.yaml
server:
  port: 8080
  dev: true

site: ./site
```

## Running

```bash
basil --dev
```

Visit:
- http://localhost:8080/ → uses `index.pars`
- http://localhost:8080/edit/ → uses `edit/edit.pars`
- http://localhost:8080/admin/ → uses `admin/admin.pars` (not `admin/index.pars`)
- http://localhost:8080/posts/ → uses `posts/index.pars` (no `posts/posts.pars`)
