/*
Language: Parsley
Description: Parsley language syntax highlighting for highlight.js
Author: Basil Contributors
Website: https://github.com/sambeau/basil
Category: web
*/

function hljsDefineParsley(hljs) {
  const IDENT_RE = "[a-zA-Z_][a-zA-Z0-9_]*";

  const KEYWORDS = {
    $pattern: /\b[a-zA-Z_][a-zA-Z0-9_]*\b/,
    keyword: [
      "fn",
      "function",
      "let",
      "for",
      "in",
      "as",
      "if",
      "else",
      "return",
      "export",
      "try",
      "import",
      "check",
      "stop",
      "skip",
      "computed",
      "and",
      "or",
      "not",
    ],
    literal: ["true", "false", "null"],
    built_in: [
      // File/Data Loading
      "JSON",
      "YAML",
      "CSV",
      "MD",
      "markdown",
      "lines",
      "text",
      "bytes",
      "SVG",
      "file",
      "dir",
      "fileList",
      // Time
      "time",
      "now",
      // URLs
      "url",
      // Type Conversion
      "toInt",
      "toFloat",
      "toNumber",
      "toString",
      "toArray",
      "toDict",
      // Introspection
      "inspect",
      "describe",
      "repr",
      "builtins",
      // Output
      "print",
      "println",
      "printf",
      "log",
      "logLine",
      "toDebug",
      // Control Flow
      "fail",
      // Formatting
      "format",
      "tag",
      // Regex
      "regex",
      "match",
      // Money
      "money",
      // Assets
      "asset",
      // Connection (used with @ literals)
      "sqlite",
      "postgres",
      "mysql",
      "sftp",
      "shell",
    ],
  };

  // At-literals (@-prefixed values)
  const AT_LITERAL = {
    scope: "symbol",
    variants: [
      // DateTime literals: @2024-12-25T14:30:00Z
      {
        match:
          /@\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})?)?/,
      },
      // Time literals: @14:30:00
      { match: /@\d{1,2}:\d{2}(:\d{2})?/ },
      // Special time literals
      { match: /@(now|today|timeNow|dateNow)\b/ },
      // Duration literals: @2h30m, @7d, @1y6mo (including negative durations)
      { match: /@-?\d+[yMwdhms]([0-9yMwdhms]|mo)*/ },
      // Database connection literals
      { match: /@(sqlite|postgres|mysql|DB)\b/ },
      // Other special literals
      { match: /@(sftp|shell)\b/ },
      // Schema and table literals
      { match: /@(schema|table)\b/ },
      // Query DSL literals
      { match: /@(query|insert|update|delete|transaction)\b/ },
      // Runtime context literals
      { match: /@(SEARCH|env|args|params)\b/ },
      // Standard streams
      { match: /@(stdin|stdout|stderr|stdio|-)\b/ },
      // Stdlib imports: @std/module
      {
        match:
          /@std\/(table|dev|math|valid|schema|id|api|markdown|mdDoc|html)\b/,
      },
      // Just @std
      { match: /@std\b/ },
      // Basil namespace: @basil, @basil/http, @basil/auth
      { match: /@basil(\/http|\/auth)?\b/ },
      // URL literals: @https://example.com (must come before path literals)
      { match: /@https?:\/\/[^\s<>"{}|\\^`\[\]]*/ },
      { match: /@ftp:\/\/[^\s<>"{}|\\^`\[\]]*/ },
      { match: /@file:\/\/[^\s<>"{}|\\^`\[\]]*/ },
      // Path templates: @(./path/{expr})
      { match: /@\([^)]+\)/ },
      // Path literals: @./config, @../config, @/usr/local, @~/home, @.config (dotfiles)
      { match: /@\.\.?\/[^\s<>"{}|\\^`\[\]]*/ },
      { match: /@\/[^\s<>"{}|\\^`\[\]]*/ },
      { match: /@~\/[^\s<>"{}|\\^`\[\]]*/ },
      { match: /@\.[a-zA-Z0-9_.-]+/ }, // dotfiles like @.config, @.bashrc
    ],
  };

  // Money literals: $99.99, £50, €25, ¥1000, EUR#100.00
  const MONEY = {
    scope: "number",
    match: /([$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?/,
  };

  // Operators - special I/O and database operators
  const SPECIAL_OPERATORS = {
    scope: "operator",
    match: /<=\?\?=>|<=\?=>|<=!=>|<=#=>|<==|<=\/=|=\/=>>?|==>>?/,
  };

  // Query DSL operators
  const QUERY_OPERATORS = {
    scope: "operator",
    match: /\?\?->|\?->|\.->|\|<|\|>|<-/,
  };

  // Regex literals
  const REGEX = {
    scope: "regexp",
    begin: /\/(?![*/\s])/,
    end: /\/[gimsuvy]*/,
    contains: [hljs.BACKSLASH_ESCAPE],
  };

  // Double-quoted strings with interpolation
  const STRING = {
    scope: "string",
    begin: '"',
    end: '"',
    contains: [
      hljs.BACKSLASH_ESCAPE,
      {
        scope: "subst",
        begin: /\{/,
        end: /\}/,
        keywords: KEYWORDS,
        contains: [], // Will be filled below
      },
    ],
  };

  // Template literals (backticks)
  const TEMPLATE = {
    scope: "string",
    begin: "`",
    end: "`",
    contains: [
      hljs.BACKSLASH_ESCAPE,
      {
        scope: "subst",
        begin: /\{/,
        end: /\}/,
        keywords: KEYWORDS,
        contains: [],
      },
    ],
  };

  // Raw template literals (single quotes with @{} interpolation)
  const RAW_TEMPLATE = {
    scope: "string",
    begin: "'",
    end: "'",
    contains: [
      hljs.BACKSLASH_ESCAPE,
      {
        scope: "subst",
        begin: /@\{/,
        end: /\}/,
        keywords: KEYWORDS,
        contains: [],
      },
    ],
  };

  // JSX-like tags
  const TAG = {
    scope: "tag",
    begin: /<\/?/,
    end: /\/?>/,
    contains: [
      {
        scope: "name",
        match: /[A-Za-z][A-Za-z0-9-]*/,
        starts: {
          endsWithParent: true,
          contains: [
            {
              scope: "attr",
              match: /[a-zA-Z][a-zA-Z0-9_-]*(?=\s*=)/,
            },
            STRING,
            {
              scope: "subst",
              begin: /\{/,
              end: /\}/,
              keywords: KEYWORDS,
            },
            {
              scope: "operator",
              match: /\.\.\.[a-zA-Z_][a-zA-Z0-9_]*/,
            },
          ],
        },
      },
    ],
  };

  // Fill in recursive contains for interpolations
  const INTERPOLATION_CONTAINS = [hljs.C_NUMBER_MODE, AT_LITERAL, STRING];
  STRING.contains[1].contains = INTERPOLATION_CONTAINS;
  TEMPLATE.contains[1].contains = INTERPOLATION_CONTAINS;
  RAW_TEMPLATE.contains[1].contains = INTERPOLATION_CONTAINS;

  return {
    name: "Parsley",
    aliases: ["pars", "parsley"],
    keywords: KEYWORDS,
    contains: [
      hljs.C_LINE_COMMENT_MODE,
      SPECIAL_OPERATORS,
      QUERY_OPERATORS,
      AT_LITERAL,
      MONEY,
      REGEX,
      STRING,
      TEMPLATE,
      RAW_TEMPLATE,
      TAG,
      hljs.C_NUMBER_MODE,
      {
        // Function definitions: myFunc = fn(...)
        scope: "function",
        match: new RegExp(IDENT_RE + "(?=\\s*=\\s*fn\\b)"),
      },
      {
        // Range operator
        scope: "operator",
        match: /\.\./,
      },
      {
        // Spread operator
        scope: "operator",
        match: /\.\.\./,
      },
      {
        // Nullish coalescing
        scope: "operator",
        match: /\?\?/,
      },
      {
        // Regex match operators
        scope: "operator",
        match: /!~|~/,
      },
      {
        // Concatenation
        scope: "operator",
        match: /\+\+/,
      },
      {
        // Underscore as discard
        scope: "variable.language",
        match: /\b_\b/,
      },
    ],
  };
}

// Export for different module systems
if (typeof module !== "undefined" && module.exports) {
  module.exports = hljsDefineParsley;
}
if (typeof window !== "undefined" && window.hljs) {
  window.hljs.registerLanguage("parsley", hljsDefineParsley);
}
