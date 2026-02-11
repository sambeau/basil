# Add Parsley language support

## Description

This PR adds support for the [Parsley](https://github.com/sambeau/basil) programming language to GitHub Linguist.

Parsley is a scripting language designed for building web applications. It features:
- JSX-like tag syntax for HTML templating
- Built-in literals for dates, times, durations, money, paths, and URLs
- Query DSL for database operations
- First-class file I/O operators
- String interpolation in multiple string types

## Checklist

- [x] Added entry to `lib/linguist/languages.yml`
- [x] Added sample files to `samples/Parsley/`
- [x] Language has an associated TextMate grammar (`source.parsley`)
- [x] File extensions are unique (`.pars`, `.part`)

## Language Entry

```yaml
Parsley:
  type: programming
  color: "#3B6EA5"
  extensions:
    - ".pars"
    - ".part"
  tm_scope: source.parsley
  ace_mode: text
  codemirror_mode: null
  codemirror_mime_type: null
  language_id: XXXXXX  # To be assigned
```

## Grammar Sources

- **TextMate grammar**: [VS Code Extension](https://github.com/sambeau/basil/tree/main/.vscode-extension/syntaxes/parsley.tmLanguage.json)
- **Tree-sitter grammar**: [tree-sitter-parsley](https://github.com/sambeau/tree-sitter-parsley)

## Sample Code

```parsley
// Web handler with database query and HTML template
export computed users = @query |> "SELECT * FROM users WHERE active = true"

export page = fn(title)
  <html>
    <head><title>{title}</title></head>
    <body>
      <h1>{title}</h1>
      <ul>
        {for user in users {
          <li>{user.name} - {user.email}</li>
        }}
      </ul>
    </body>
  </html>
```

## References

- Language homepage: https://github.com/sambeau/basil
- Documentation: https://github.com/sambeau/basil/tree/main/docs
- VS Code extension: Available in `.vscode-extension/` of the main repo
- Tree-sitter grammar: https://github.com/sambeau/tree-sitter-parsley

## Notes for Maintainers

The `tm_scope` of `source.parsley` corresponds to the TextMate grammar in the VS Code extension. The tree-sitter grammar is available for editors that prefer tree-sitter-based highlighting.

The file extensions are:
- `.pars` - Parsley source files (handlers, scripts)
- `.part` - Parsley partial templates