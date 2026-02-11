# GitHub Linguist Submission Guide

This directory contains the files and instructions needed to submit Parsley to [GitHub Linguist](https://github.com/github-linguist/linguist) for `.pars` file recognition on GitHub.

## Prerequisites

Before submitting:

1. **Tree-sitter grammar published**: The grammar at `github.com/sambeau/tree-sitter-parsley` should be live
2. **VS Code extension available**: The TextMate grammar should be accessible
3. **Sample code ready**: Representative Parsley code samples

## Files in This Directory

| File | Purpose |
|------|---------|
| `languages.yml.patch` | Entry to add to `lib/linguist/languages.yml` |
| `sample.pars` | Sample file to add to `samples/Parsley/` |
| `PR_DESCRIPTION.md` | Template for the pull request description |

## Submission Steps

### 1. Fork Linguist

```bash
# Fork github/linguist on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/linguist.git
cd linguist
```

### 2. Create a Branch

```bash
git checkout -b add-parsley-language
```

### 3. Add Language Entry

Edit `lib/linguist/languages.yml` and add the Parsley entry in alphabetical order (after "ParseTree", before "Pascal"):

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
  language_id: 1234567890  # Will be assigned by maintainers
```

**Note**: Leave `language_id` as a placeholder. Linguist maintainers will assign the actual ID.

### 4. Add Sample Files

```bash
mkdir -p samples/Parsley
cp /path/to/sample.pars samples/Parsley/
```

You can add multiple sample files to showcase different language features.

### 5. Run Tests

```bash
# Install dependencies
bundle install

# Run the test suite
bundle exec rake test
```

### 6. Commit and Push

```bash
git add lib/linguist/languages.yml samples/Parsley/
git commit -m "Add Parsley language support"
git push origin add-parsley-language
```

### 7. Open Pull Request

Open a PR against `github/linguist` using the content from `PR_DESCRIPTION.md`.

## After Merge

Once the PR is merged:

1. **Wait for deployment**: GitHub deploys Linguist updates periodically
2. **Verify on GitHub**: Push a `.pars` file to a repo and check that it's recognized
3. **Update Basil docs**: Note that Parsley is now recognized on GitHub

## Troubleshooting

### "Unknown extension" error

Make sure the extensions aren't already claimed by another language. Check `languages.yml` for conflicts.

### Sample files not detected

Ensure sample files have valid Parsley syntax and are representative of the language.

### TextMate scope not found

The `tm_scope: source.parsley` must match the scope in the TextMate grammar. This is defined in the VS Code extension at `.vscode-extension/syntaxes/parsley.tmLanguage.json`.

## References

- [Linguist Contributing Guide](https://github.com/github-linguist/linguist/blob/master/CONTRIBUTING.md)
- [Adding a Language](https://github.com/github-linguist/linguist/blob/master/CONTRIBUTING.md#adding-a-language)
- [Language YAML Schema](https://github.com/github-linguist/linguist/blob/master/lib/linguist/languages.yml)