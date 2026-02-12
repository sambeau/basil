# Publishing Guide for Parsley Zed Extension

This guide walks through the steps to publish the Parsley extension to the Zed extensions registry.

## Prerequisites

- [x] Extension implemented in `contrib/zed-extension/`
- [x] Local testing completed successfully (see `TESTING.md`)
- [x] All test cases passing
- [ ] GitHub account with access to create repositories

## Step 1: Create Standalone Repository

The extension needs to live in its own repository for publication.

### 1.1 Create GitHub Repository

1. Go to https://github.com/new
2. Repository name: `parsley-zed`
3. Description: "Parsley language support for Zed Editor"
4. Make it **Public**
5. Do NOT initialize with README (we'll copy ours)
6. Click "Create repository"

### 1.2 Clone and Setup

```bash
# Clone the new repository
git clone https://github.com/sambeau/parsley-zed.git
cd parsley-zed

# Copy extension files from Basil repo
cp -r /path/to/basil/contrib/zed-extension/* .

# Verify structure
ls -la
# Should show: extension.toml, LICENSE, README.md, .gitignore, languages/, test/

# Initialize git (if not already done)
git add .
git commit -m "Initial commit: Parsley language extension for Zed"
git push -u origin main
```

### 1.3 Verify Repository

Check that your repository has:
- [x] `extension.toml` at root
- [x] `LICENSE` (MIT)
- [x] `README.md`
- [x] `.gitignore`
- [x] `languages/parsley/` directory with all query files
- [x] Repository is public
- [x] Description is set

## Step 2: Update Grammar Reference (Optional)

If you want Zed to use a specific commit instead of `main`:

1. Get the latest commit SHA from the Basil repo:
   ```bash
   cd /path/to/basil
   git log -1 --format=%H contrib/tree-sitter-parsley/
   ```

2. Update `extension.toml` in `parsley-zed`:
   ```toml
   [grammars.parsley]
   repository = "https://github.com/sambeau/basil"
   commit = "abc123..."  # Replace with actual commit SHA
   path = "contrib/tree-sitter-parsley"
   ```

3. Commit the change:
   ```bash
   git commit -am "Pin grammar to specific commit"
   git push
   ```

**Note**: Using `main` is fine for initial release. Pin to a specific commit for stability later.

## Step 3: Fork Zed Extensions Repository

1. Go to https://github.com/zed-industries/extensions
2. Click "Fork" button (top-right)
3. Select your personal account (NOT an organization)
   - This allows Zed maintainers to push fixes to your PR if needed
4. Wait for fork to complete

## Step 4: Add Extension as Submodule

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/extensions.git
cd extensions

# Initialize existing submodules (may take a while)
git submodule init
git submodule update

# Add Parsley extension as submodule
git submodule add https://github.com/sambeau/parsley-zed.git extensions/parsley

# Stage the new submodule
git add extensions/parsley .gitmodules
```

## Step 5: Update extensions.toml

1. Open `extensions.toml` in your editor

2. Add Parsley entry (it will be sorted in next step):
   ```toml
   [parsley]
   submodule = "extensions/parsley"
   version = "0.1.0"
   ```

3. Save the file

4. Sort extensions (required by Zed):
   ```bash
   # Install pnpm if not already installed
   # npm install -g pnpm

   # Sort extensions
   pnpm install
   pnpm sort-extensions
   ```

5. Verify the changes:
   ```bash
   git diff extensions.toml
   git diff .gitmodules
   ```

## Step 6: Commit and Push

```bash
# Stage all changes
git add extensions.toml .gitmodules extensions/parsley

# Commit with descriptive message
git commit -m "Add Parsley language extension

Parsley is a modern programming language for web development with the Basil framework.

Features:
- Syntax highlighting for .pars and .part files
- Bracket matching
- Code outline
- Auto-indentation
- Support for at-literals, tags, and string interpolation

Repository: https://github.com/sambeau/parsley-zed
Grammar: https://github.com/sambeau/basil/tree/main/contrib/tree-sitter-parsley"

# Push to your fork
git push origin main
```

## Step 7: Create Pull Request

1. Go to your fork: https://github.com/YOUR-USERNAME/extensions
2. Click "Contribute" → "Open pull request"
3. Title: "Add Parsley language extension"
4. Description template:

```markdown
## Extension Information

- **Language**: Parsley
- **Extension Repository**: https://github.com/sambeau/parsley-zed
- **Grammar Repository**: https://github.com/sambeau/basil/tree/main/contrib/tree-sitter-parsley
- **License**: MIT

## What is Parsley?

Parsley is a modern programming language designed for web development with the Basil framework. It features:
- Clean, expressive syntax
- First-class database and file I/O support
- JSX-like templating
- Built-in time and money types
- Powerful string interpolation

Learn more: https://github.com/sambeau/basil

## Extension Features

- Syntax highlighting for `.pars` and `.part` files
- Bracket matching for arrays, dictionaries, functions, and tags
- Code outline with functions and exports
- Smart auto-indentation
- Support for all Parsley syntax elements (at-literals, tags, operators, etc.)

## Testing

Tested locally using Zed's dev extension feature with comprehensive test cases.

## Screenshots (Optional)

[Add screenshots of syntax highlighting if desired]

## Checklist

- [x] Extension ID does not contain "zed" or "Zed"
- [x] Repository uses HTTPS URL (not SSH)
- [x] Extension has valid MIT license
- [x] `extensions.toml` sorted with `pnpm sort-extensions`
- [x] Extension tested locally
- [x] Documentation is complete
```

5. Click "Create pull request"

## Step 8: PR Review Process

### What to Expect

- **Automated Checks**: CI will run to validate your extension
  - License validation
  - TOML syntax
  - Submodule URL format
  - Sorted order

- **Maintainer Review**: Zed team will review your PR
  - Typically takes 1-7 days
  - They may request changes
  - They may push fixes to your PR branch

### Common Review Feedback

1. **Naming**: Extension ID shouldn't contain "Zed"
   - ✅ Good: `parsley`
   - ❌ Bad: `parsley-zed`, `zed-parsley`

2. **License**: Must be one of the accepted licenses
   - ✅ We're using MIT

3. **Grammar Reference**: Should use HTTPS, not SSH
   - ✅ We're using `https://github.com/sambeau/basil`

4. **Sorting**: Files must be sorted
   - ✅ We ran `pnpm sort-extensions`

### Responding to Feedback

If changes are requested:

```bash
# In your parsley-zed repo (if extension needs changes)
# Make changes
git commit -am "Fix: description of change"
git push

# In your extensions fork
cd extensions
git submodule update --remote extensions/parsley
git add extensions/parsley
git commit -m "Update Parsley extension: description"
git push
```

## Step 9: After Merge

Once your PR is merged:

1. **Extension is Published**: It will appear in Zed's extension marketplace
2. **Users Can Install**: Via "zed: extensions" command
3. **Updates**: To release new versions:
   - Update `version` in `parsley-zed/extension.toml`
   - Commit and push to `parsley-zed` repo
   - Submit new PR to `zed-industries/extensions` updating the version number

## Maintenance

### Releasing Updates

1. Make changes in `parsley-zed` repository
2. Update version in `extension.toml` (follow semver):
   - Patch: `0.1.0` → `0.1.1` (bug fixes)
   - Minor: `0.1.0` → `0.2.0` (new features)
   - Major: `0.1.0` → `1.0.0` (breaking changes)
3. Commit and push
4. Submit PR to `zed-industries/extensions` with updated version

### Updating Grammar

If you update the Tree-sitter grammar:

1. Update grammar in Basil repo (`contrib/tree-sitter-parsley/`)
2. Run tests: `tree-sitter test`
3. Commit grammar changes
4. Update `extension.toml` to reference new commit (optional but recommended)
5. Test locally
6. Release new extension version

## Troubleshooting

### CI Fails: License Not Found

**Problem**: PR CI checks fail on license validation
**Solution**: Ensure `LICENSE` file exists at root of `parsley-zed` repo

### CI Fails: Not Sorted

**Problem**: Files not in sorted order
**Solution**: Run `pnpm sort-extensions` in extensions fork

### Submodule Not Updating

**Problem**: Changes to `parsley-zed` don't show in extensions PR
**Solution**:
```bash
cd extensions
git submodule update --remote extensions/parsley
git add extensions/parsley
git commit -m "Update Parsley extension"
git push
```

### Extension Not Loading in Zed

**Problem**: After merge, extension doesn't work
**Solution**: Check Zed's log (`zed: open log`) for errors. Most likely:
- Grammar reference is incorrect
- Query files have syntax errors
- TOML configuration is invalid

## Resources

- [Zed Extension Documentation](https://zed.dev/docs/extensions)
- [Zed Extensions Repository](https://github.com/zed-industries/extensions)
- [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/)
- [Parsley Repository](https://github.com/sambeau/basil)

## Timeline

Expected timeline for publishing:

| Step | Duration |
|------|----------|
| Create standalone repo | 15 minutes |
| Fork and add submodule | 30 minutes |
| Create PR | 15 minutes |
| **Wait for review** | **1-7 days** |
| Address feedback (if any) | 30 minutes |
| **Wait for merge** | **1-2 days** |
| **Total** | **~2-10 days** |

Most of the time is waiting for maintainers. The actual work is ~1-2 hours.

## Success!

Once merged, users can install your extension:
1. Open Zed
2. `Cmd+Shift+P` → "zed: extensions"
3. Search "Parsley"
4. Click "Install"

Congratulations on publishing a Zed extension! 🎉