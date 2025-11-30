# Commit Message Standards

Use [Conventional Commits](https://www.conventionalcommits.org/) format for all commits.

## Format
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

## Types
| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat(cli): add --verbose flag` |
| `fix` | Bug fix | `fix(parser): handle empty input` |
| `docs` | Documentation only | `docs: update README` |
| `style` | Formatting, no code change | `style: fix indentation` |
| `refactor` | Code change, no new feature or fix | `refactor: extract helper function` |
| `test` | Adding or fixing tests | `test: add parser edge cases` |
| `chore` | Maintenance tasks | `chore: update dependencies` |

## Scope (Optional)
Use component or feature area:
- `cli` — Command-line interface
- `parser` — Input parsing
- `config` — Configuration handling
- `docs` — Documentation

## Rules
1. **Subject line**: Max 50 characters, imperative mood ("add" not "added")
2. **Body**: Wrap at 72 characters, explain what and why (not how)
3. **Footer**: Reference issues: `Refs: FEAT-001` or `Fixes: BUG-001`

## Examples

### Simple commit
```
feat(cli): add --dry-run flag
```

### Commit with body
```
fix(parser): handle quoted strings with newlines

The parser was splitting on newlines before processing quotes,
causing multi-line strings to be incorrectly parsed.

Fixes: BUG-003
```

### Breaking change
```
feat(config)!: change config file format to YAML

BREAKING CHANGE: Config files must now be YAML format.
Run `basil migrate-config` to convert existing JSON configs.

Refs: FEAT-015
```

## AI Commits
- AI commits to feature/bug branches only
- Include the feature/bug ID in footer: `Refs: FEAT-XXX`
- Human reviews before merging to main
