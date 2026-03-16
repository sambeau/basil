# Design: FileField Component

**Date:** 2026-03-15
**Status:** Draft
**Related:**
- `work/design/DESIGN-prelude-1.0-components.md` §7 (deferred placeholder)
- `work/design/server-enhanced-components.md` §8 (early sketch)
- `work/reports/STANDARD-PRELUDE-REVIEW.md` §12, §14
- `server/prelude/components/text_field.pars` (pattern to follow)
- `server/handler.go` — `parseMultipartForm` (existing multipart support)

---

## 1. Overview

FileField is a form component for uploading files in Basil applications. It is the last major gap in the form component set — every other common input type (text, textarea, select, radio, checkbox) already has a prelude component.

This document proposes a **phased design**: a basic Level 1 component for 1.0 that follows the exact patterns of the existing form fields, and progressive enhancements (drag-drop, previews, chunked uploads, progress) for 1.1+.

### 1.1 Design Goals

1. **Consistent** — Looks and feels like `TextField` and the other `*Field` components
2. **Semantic HTML** — Uses `<input type="file">` with proper `<label>`, hint/error, and ARIA
3. **Unstyled** — Works beautifully with Pico CSS (and any other classless/utility framework) with no component-supplied CSS
4. **Progressive** — Basic version works with zero JavaScript; enhancements layer on top
5. **Simple API** — Basic use case is a single prop (`name`); complexity is opt-in
6. **Secure by default** — Server-side validation, size limits, type checking; client-side validation is a convenience, not a trust boundary

### 1.2 Scope

| Phase | Target | What |
|-------|--------|------|
| **Phase 1** | 1.0 | Basic `<FileField>` — semantic HTML, accessibility, Pico CSS compatible, auto `enctype` fix in `basil.js` |
| **Phase 2** | 1.1 | Client-side validation, file list display, current file display, remove button |
| **Phase 3** | 1.1+ | Drag-drop zone, image preview thumbnails, upload progress |
| **Phase 4** | 1.2+ | Managed upload endpoint, chunked uploads, resumable uploads |

This document covers all four phases but only Phase 1 is proposed for 1.0.

---

## 2. Prior Art

### 2.1 HTML Standard

The native `<input type="file">` element is well-supported and accessible. Key attributes:

| Attribute | Purpose |
|-----------|---------|
| `accept` | MIME types or extensions (e.g., `image/*`, `.pdf,.docx`) |
| `multiple` | Allow selecting multiple files |
| `capture` | Camera/microphone on mobile (`user`, `environment`) |
| `required` | Form validation |

The `<input type="file">` must be inside a `<form>` with `enctype="multipart/form-data"` for the file contents to be transmitted. This is the single most common mistake users make with file uploads.

### 2.2 Framework Survey

| Framework | Approach | Key Ideas |
|-----------|----------|-----------|
| **Rails** Active Storage | `file_field` helper + server-managed direct uploads | Auto-generates upload endpoint; hidden field stores blob ID; JS handles direct-to-storage upload |
| **Laravel** Livewire | `<input type="file" wire:model="photo">` | Automatic temp upload + preview; server stores to configurable disk; progress events |
| **Django** | `ClearableFileInput` widget | Shows current file + "Clear" checkbox + new file input; handles replace-or-keep semantics |
| **SvelteKit** / **Remix** | Native `<form enctype="multipart/form-data">` | Lean on the platform; file available as `File` object in form action |
| **Dropzone.js** | Standalone JS library | Drag-drop zone; thumbnail previews; chunked uploads; extensive events API |
| **Uppy** (Transloadit) | Modular upload framework | Plugin architecture; resumable uploads (tus protocol); dashboard UI; provider integrations |
| **FilePond** | JS library with framework adapters | Drop-in replacement for `<input type="file">`; async uploads; image editing; plugins |
| **Shoelace** `<sl-file-upload>` | Web component | Drag-drop + browse; max-files; custom icons |

### 2.3 Key Takeaways from Prior Art

1. **Start native** — Rails, Django, SvelteKit all start with plain `<input type="file">` and enhance from there. We should too.
2. **The `enctype` trap** — Every framework either auto-sets it or prominently documents it. We must handle this.
3. **Server-managed uploads are the gold standard** — Rails Active Storage and Laravel both auto-generate upload endpoints. This is the best UX (upload starts immediately, not on form submit) but also the most complex to implement.
4. **Progressive enhancement layers** — The mature libraries (Dropzone, Uppy, FilePond) all support a "basic fallback" mode. Our Phase 1 IS that fallback; later phases add the enhancements.
5. **Django's "current file" pattern** — When editing an existing record, showing the current file with a "remove" option is essential UX. We should plan for this.

---

## 3. Phase 1: Basic FileField (1.0)

### 3.1 API

```parsley
// Minimal — just a file input with a label
<FileField name="avatar" label="Profile Photo"/>

// With validation hints
<FileField
    name="resume"
    label="Upload Resume"
    accept=".pdf,.docx"
    hint="PDF or Word document, max 5MB"
    required={true}
/>

// Multiple files
<FileField
    name="attachments"
    label="Attachments"
    multiple={true}
    accept="image/*,.pdf"
/>

// With error from server-side validation
<FileField
    name="avatar"
    label="Profile Photo"
    accept="image/*"
    error={errors.avatar}
/>

// Mobile camera capture
<FileField
    name="photo"
    label="Take a Photo"
    accept="image/*"
    capture="environment"
/>
```

### 3.2 Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `name` | string | — | Field name (required) |
| `label` | string | — | Label text |
| `accept` | string | `null` | Accepted MIME types/extensions |
| `multiple` | bool | `false` | Allow multiple files |
| `capture` | string | `null` | Camera direction: `"user"` or `"environment"` |
| `required` | bool | `false` | Required field |
| `hint` | string | `null` | Help text below the input |
| `error` | string | `null` | Error message (replaces hint) |
| `id` | string | `name` | Override generated ID |
| `disabled` | bool | `false` | Disable the input |
| `...inputAttrs` | — | — | Passed through to `<input>` |

### 3.3 Rendered HTML

```html
<!-- <FileField name="avatar" label="Profile Photo" accept="image/*" hint="Max 5MB" required={true}/> -->

<label for="avatar">
    Profile Photo
    <span aria-hidden="true"> *</span>
</label>
<input
    type="file"
    id="avatar"
    name="avatar"
    accept="image/*"
    required="required"
    aria-required="true"
    aria-describedby="avatar-hint"
/>
<small id="avatar-hint">Max 5MB</small>
```

With error:

```html
<label for="avatar">
    Profile Photo
    <span aria-hidden="true"> *</span>
</label>
<input
    type="file"
    id="avatar"
    name="avatar"
    accept="image/*"
    required="required"
    aria-required="true"
    aria-describedby="avatar-error"
    aria-invalid="true"
/>
<small id="avatar-error">File must be an image</small>
```

This is identical in structure to `TextField` — `<label>` → `<input>` → `<small>` — which means Pico CSS styles it correctly with zero custom CSS.

### 3.4 Implementation

```parsley
// FileField - File upload input with label, hint, error, and accessibility
// Compatible with Pico CSS - uses <small> for hints/errors (inherits validation color)
// Usage: <FileField name="avatar" label="Profile Photo" accept="image/*"/>

export FileField = fn(props) {
    // Extract component-specific props, spread the rest to the input
    let {name, label, accept, multiple, capture, hint, error, required, id, ...inputAttrs} = props

    // Generate IDs for accessibility
    let inputId = id ?? name
    let hintId = if (hint) inputId + "-hint" else null
    let errorId = if (error) inputId + "-error" else null

    // Build aria-describedby from hint and error IDs
    let describedByParts = [hintId, errorId].filter(fn(x) { x != null })
    let describedBy = if (describedByParts.length() > 0) { describedByParts.join(" ") } else { null }

    <label for={inputId}>
        label
        if (required) {
            <span aria-hidden="true">" *"</span>
        }
    </label>
    <input
        type="file"
        id={inputId}
        name={name}
        accept={accept}
        multiple={if (multiple) "multiple" else null}
        capture={capture}
        required={if (required) "required" else null}
        aria-required={if (required) "true" else null}
        aria-describedby={describedBy}
        aria-invalid={if (error) "true" else null}
        ...inputAttrs
    />
    if (hint && !error) {
        <small id={hintId}>hint</small>
    }
    if (error) {
        <small id={errorId}>error</small>
    }
}
```

This follows the exact same pattern as `TextField` — destructure with rest spread, generate IDs, build `aria-describedby`, emit `<label>` → `<input>` → `<small>`.

### 3.5 Registration

Add to `componentFiles` in `pkg/parsley/evaluator/stdlib_html.go`:

```go
// Form components
{"text_field.pars", "TextField"},
{"textarea_field.pars", "TextareaField"},
{"select_field.pars", "SelectField"},
{"radio_group.pars", "RadioGroup"},
{"checkbox_group.pars", "CheckboxGroup"},
{"checkbox.pars", "Checkbox"},
{"file_field.pars", "FileField"},    // ← new
{"button.pars", "Button"},
{"form.pars", "Form"},
```

### 3.6 Form `enctype` Handling

**Problem:** `<input type="file">` silently sends only the filename (no file data) unless the `<form>` has `enctype="multipart/form-data"`. This is the #1 file upload pitfall. Users will write:

```parsley
<Form action="/upload" method="POST">
    <FileField name="avatar" label="Photo"/>
    <Button type="submit">"Upload"</Button>
</Form>
```

…and wonder why `request.files` is empty.

**Solution (Phase 1):** Two layers of defense:

**1. Documentation** — Clearly document that the user should set `enctype`:

```parsley
<Form action="/upload" method="POST" enctype="multipart/form-data">
    <FileField name="avatar" label="Photo"/>
    <Button type="submit">"Upload"</Button>
</Form>
```

The `enctype` prop already passes through via `...formAttrs` on the `<Form>` component — no changes to `Form` are needed.

**2. Auto-fix in `basil.js`** — Since `basil.js` already runs on every prelude page and already handles progressive enhancements (`data-confirm`, `data-counter`, auto-resize, focus-first-invalid, etc.), adding a one-liner safety net here is consistent with what Phase 1 already ships:

```javascript
// In basil.js — auto-fix enctype for forms containing file inputs
document.querySelectorAll('form:has(input[type="file"])').forEach(form => {
    if (!form.enctype || form.enctype === 'application/x-www-form-urlencoded') {
        form.enctype = 'multipart/form-data';
    }
});
```

This isn't introducing JavaScript to FileField — it's adding a line to the existing enhancement script that catches the #1 file upload mistake for *any* `<input type="file">`, whether it comes from `<FileField>` or hand-written HTML. It's a safety net, not a feature.

> **Note:** A Parsley-level auto-detect (where `Form` inspects its `contents` for FileField children) is architecturally infeasible — contents are opaque rendered output, not introspectable AST nodes. The JS approach is the right tool here.

### 3.7 Server-Side Handling

The existing `parseMultipartForm` in `handler.go` already parses multipart requests and exposes file metadata via `request.files`. A basic upload handler looks like:

```parsley
// In a route handler:
let {request} = import @basil/http

if (request.method == "POST") {
    let file = request.files.avatar
    // file = {filename: "photo.jpg", size: 102400, headers: {...}}
    // Actual file saving handled by the Basil runtime (see §3.8)
}
```

### 3.8 File Persistence Gap

**Current state:** `parseMultipartForm` extracts file metadata but does not expose file contents or provide a save mechanism to Parsley code. The multipart form data IS parsed by Go's `r.ParseMultipartForm`, so the temp files exist during request processing — but there is no Parsley API to persist them.

**Phase 1 approach:** Add a `saveFile` builtin (or `request.saveFile(fieldName, destPath)`) that saves an uploaded file to an allowed directory. This is pure Go code:

```go
// Proposed: server/upload.go
func saveUploadedFile(r *http.Request, fieldName, destPath string, allowedDirs []string) error {
    // 1. Validate destPath is within an allowed directory (security.allow_write)
    // 2. Open the multipart file: r.FormFile(fieldName)
    // 3. Create destination file
    // 4. Copy contents
    // 5. Close both
}
```

Exposed to Parsley as:

```parsley
let result = request.saveFile("avatar", "./uploads/" + file.filename)
// result = {path: "./uploads/photo.jpg", size: 102400} or an error
```

This uses the existing `security.allow_write` config to control which directories are writable:

```yaml
# basil.yaml
security:
  allow_write:
    - ./uploads
    - ./data/attachments
```

**Detailed design of `saveFile` is deferred to a spec (FEAT-xxx) since it involves runtime/evaluator changes.** Phase 1 of FileField (the UI component) can ship without it — users can build their own upload endpoint in Go as a Basil plugin or handle uploads through a proxy.

### 3.9 Estimated Effort

- **Component file:** ~30 minutes (it's almost identical to TextField)
- **Registration:** ~5 minutes
- **Auto-enctype in `basil.js`:** ~15 minutes
- **Tests:** ~1-2 hours (render tests, accessibility, edge cases)
- **Documentation:** ~1 hour

**Total: ~3-4 hours** — well within 1.0 budget.

---

## 4. Phase 2: Client-Side Validation & File List (1.1)

### 4.1 Client-Side File Validation

Add `data-*` attributes to the `<input>` that `basil.js` reads for client-side validation:

```parsley
<FileField
    name="avatar"
    label="Profile Photo"
    accept="image/*"
    maxSize={5 * 1024 * 1024}    // 5MB — new prop
    maxFiles={3}                  // new prop (only meaningful with multiple)
    multiple={true}
/>
```

Rendered HTML:

```html
<input
    type="file"
    id="avatar"
    name="avatar"
    accept="image/*"
    multiple="multiple"
    data-max-size="5242880"
    data-max-files="3"
    aria-describedby="avatar-hint"
/>
```

JavaScript enhancement in `basil.js`:

```javascript
// File validation on change
document.addEventListener('change', (e) => {
    if (e.target.type !== 'file') return;
    const input = e.target;
    const maxSize = parseInt(input.dataset.maxSize, 10);
    const maxFiles = parseInt(input.dataset.maxFiles, 10);

    const errors = [];
    const files = Array.from(input.files);

    if (maxFiles && files.length > maxFiles) {
        errors.push(`Maximum ${maxFiles} files allowed`);
    }
    if (maxSize) {
        const tooBig = files.filter(f => f.size > maxSize);
        if (tooBig.length > 0) {
            const limit = formatBytes(maxSize);
            errors.push(`${tooBig.length} file(s) exceed the ${limit} limit`);
        }
    }

    // Show/clear inline error via aria-invalid + sibling <small>
    showFieldError(input, errors.join('. '));
});
```

**This is ~30 lines added to `basil.js`** — consistent with how it already handles `data-confirm`, `data-counter`, etc.

### 4.2 New Props for Phase 2

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `maxSize` | integer | `null` | Maximum file size in bytes |
| `maxFiles` | integer | `null` | Maximum number of files (with `multiple`) |

### 4.3 Selected File List Display

After a user selects files, show the file names below the input. This is purely JS-driven — the server-rendered HTML doesn't change:

```javascript
// After file selection, render a file list below the input
function renderFileList(input) {
    let list = input.nextElementSibling;
    if (!list || !list.classList.contains('file-field-list')) {
        list = document.createElement('ul');
        list.className = 'file-field-list';
        list.setAttribute('role', 'status');
        list.setAttribute('aria-live', 'polite');
        input.insertAdjacentElement('afterend', list);
    }
    list.innerHTML = '';
    Array.from(input.files).forEach(file => {
        const li = document.createElement('li');
        li.textContent = `${file.name} (${formatBytes(file.size)})`;
        list.appendChild(li);
    });
}
```

This uses `aria-live="polite"` so screen readers announce the selected files.

### 4.4 "Current File" Display (Edit Mode)

When editing an existing record, the user needs to see the currently uploaded file. This is a prop:

```parsley
<FileField
    name="avatar"
    label="Profile Photo"
    current="photo.jpg"           // filename or display text
    currentUrl="/uploads/photo.jpg"   // optional link to current file
/>
```

Rendered HTML:

```html
<label for="avatar">Profile Photo</label>
<div class="file-field-current">
    <a href="/uploads/photo.jpg">photo.jpg</a>
    <label>
        <input type="checkbox" name="avatar_remove" value="1"/>
        Remove
    </label>
</div>
<input type="file" id="avatar" name="avatar" aria-describedby="avatar-hint"/>
<small id="avatar-hint">Select a new file to replace the current one</small>
```

This follows Django's `ClearableFileInput` pattern: show the current file, offer a remove checkbox, and allow selecting a replacement. The `<div class="file-field-current">` is the only structural deviation from TextField — it sits between the label and the input.

### 4.5 Estimated Effort (Phase 2)

- **Client validation JS:** ~2 hours
- **File list display:** ~1 hour
- **Current file display:** ~2 hours
- **Tests:** ~3 hours

**Total: ~8 hours**

---

## 5. Phase 3: Drag-Drop, Previews, Progress (1.1+)

### 5.1 Design Approach

Phase 3 moves beyond simple `<input type="file">` styling into interactive territory. The key architectural decision is: **do we enhance the existing `<input>` or replace it with a custom dropzone?**

**Recommendation: Enhance, don't replace.**

The `<input type="file">` remains in the DOM (it's the actual form control and what gets submitted). The dropzone is a visual overlay that triggers the hidden input. This is the pattern used by Dropzone.js, FilePond, and most modern upload UIs. It means:

- Form submission still works without JS (progressive enhancement)
- The native file picker still works
- Accessibility is handled by the native input
- We don't reinvent form mechanics

### 5.2 Drag-Drop Zone

Activated by a `dragDrop` prop:

```parsley
<FileField
    name="attachments"
    label="Upload Files"
    accept="image/*,.pdf"
    multiple={true}
    dragDrop={true}
/>
```

Rendered HTML with `dragDrop`:

```html
<label for="attachments">Upload Files</label>
<div class="file-field-drop" data-file-drop="true" tabindex="0" role="button"
     aria-label="Upload Files. Drag files here or press Enter to browse.">
    <input type="file" id="attachments" name="attachments"
           accept="image/*,.pdf" multiple="multiple" tabindex="-1"/>
    <p>Drag files here or click to browse</p>
</div>
```

JavaScript behavior:
- `dragenter`/`dragover` → add `drag-over` class (for CSS hover styling)
- `dragleave`/`drop` → remove `drag-over` class
- `drop` → assign files to the hidden `<input>` via `DataTransfer`
- Click on dropzone → trigger `input.click()`
- Keyboard: Enter/Space on dropzone → trigger `input.click()`

The `<p>` inside the dropzone is plain text that Pico CSS (or any framework) can style. We don't provide the styling — just the semantic structure.

### 5.3 Image Preview Thumbnails

Activated by a `preview` prop:

```parsley
<FileField
    name="photos"
    label="Photos"
    accept="image/*"
    multiple={true}
    preview={true}
/>
```

JavaScript reads selected files via `FileReader` and renders thumbnails:

```javascript
function renderPreviews(input, listElement) {
    Array.from(input.files).forEach(file => {
        if (!file.type.startsWith('image/')) return;
        const li = document.createElement('li');
        const img = document.createElement('img');
        img.alt = file.name;
        img.style.maxWidth = '120px';
        img.style.maxHeight = '120px';
        const reader = new FileReader();
        reader.onload = (e) => { img.src = e.target.result; };
        reader.readAsDataURL(file);
        li.appendChild(img);
        li.appendChild(document.createTextNode(` ${file.name}`));
        listElement.appendChild(li);
    });
}
```

Previews use `FileReader.readAsDataURL` — no server round-trip needed. For non-image files, a filename-only display is shown (same as Phase 2 file list).

### 5.4 Upload Progress

Upload progress requires **async upload** — files are uploaded to an endpoint as they're selected, rather than waiting for form submission. This is the Phase 3/4 boundary: it fundamentally changes the upload model from "synchronous with form submit" to "async with ID reference."

**Approach:** XHR with `upload.onprogress`:

```javascript
function uploadFile(file, endpoint, csrf, onProgress, onComplete, onError) {
    const xhr = new XMLHttpRequest();
    const formData = new FormData();
    formData.append('file', file);

    xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
            onProgress(e.loaded / e.total);
        }
    });
    xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
            onComplete(JSON.parse(xhr.responseText));
        } else {
            onError(xhr.statusText);
        }
    });
    xhr.addEventListener('error', () => onError('Upload failed'));

    xhr.open('POST', endpoint);
    if (csrf) xhr.setRequestHeader('X-CSRF-Token', csrf);
    xhr.send(formData);
}
```

Progress is displayed as a `<progress>` element per file — fully semantic HTML:

```html
<li>
    <span>photo.jpg (2.1 MB)</span>
    <progress value="0.45" max="1">45%</progress>
    <button type="button" aria-label="Cancel upload" class="file-field-cancel">×</button>
</li>
```

Pico CSS styles `<progress>` elements natively, so this renders well with no custom CSS.

### 5.5 Estimated Effort

- **Drag-drop zone:** ~3 hours (HTML changes + JS)
- **Image previews:** ~2 hours
- **Upload progress:** ~4 hours (XHR upload, progress UI, cancel)
- **Tests:** ~4 hours

**Total: ~13 hours**

---

## 6. Phase 4: Managed Upload Endpoint (1.2+)

### 6.1 The Problem

Phases 1-3 require the user to provide their own upload endpoint. This is fine for experienced developers, but the Basil philosophy is "just do the right thing behind the scenes." The ideal experience:

```parsley
<FileField name="avatar" label="Photo" accept="image/*"/>
```

…and the file just gets saved. No endpoint to write, no storage to configure (beyond a directory path in `basil.yaml`).

### 6.2 Auto-Generated Upload Endpoint

Basil can auto-generate a temporary upload endpoint per form instance. This follows the pattern proposed in `server-enhanced-components.md` and is similar to Rails Active Storage direct uploads.

**Server-side (`basil.yaml` config):**

```yaml
uploads:
  directory: ./uploads           # Where files are stored
  max_file_size: 10MB            # Per-file limit
  max_request_size: 50MB         # Per-request limit
  allowed_types:                 # Whitelist (empty = all)
    - image/*
    - application/pdf
    - text/plain
  url_prefix: /uploads           # URL path for serving uploaded files
```

**How it works:**

1. When `FileField` renders with `upload={true}` (or auto-detected in async mode), Basil registers a temporary endpoint: `POST /_basil/upload/{token}`
2. The `token` is a HMAC-signed, time-limited identifier scoping the upload to this form session
3. The endpoint validates the upload (size, type, token) and saves to the configured directory
4. It returns JSON: `{"id": "abc123", "name": "photo.jpg", "size": 102400, "url": "/uploads/abc123-photo.jpg"}`
5. The client JS stores the returned `id` in a hidden `<input name="avatar" value="abc123">`
6. On form submit, the handler receives the file ID (not the file itself) and can look it up

**Rendered HTML:**

```html
<input type="file" id="avatar" name="avatar_file" accept="image/*"
       data-upload-endpoint="/_basil/upload/hmac-signed-token"
       data-upload-csrf="csrf-token-here"/>
<input type="hidden" name="avatar" value=""/>
```

### 6.3 Upload Endpoint Contract

```
POST /_basil/upload/{token}
Content-Type: multipart/form-data
X-CSRF-Token: {csrf}

Field: file (the uploaded file)

Success Response (200):
{
    "id": "abc123",
    "name": "photo.jpg",
    "size": 102400,
    "type": "image/jpeg",
    "url": "/uploads/abc123-photo.jpg"
}

Error Response (400/413/415):
{
    "error": "File exceeds maximum size of 10MB"
}
```

### 6.4 Security Considerations

| Concern | Mitigation |
|---------|------------|
| **Unauthorized uploads** | Endpoint token is HMAC-signed with expiry; tied to the user's session |
| **Path traversal** | Filenames are sanitized; stored with generated IDs, not user-supplied names |
| **File type spoofing** | Content-Type sniffing via `http.DetectContentType` in addition to extension checking |
| **Denial of service** | Per-file and per-request size limits enforced in Go before buffering to disk |
| **Orphaned files** | Uploaded files without form submission are cleaned up by a periodic sweep (configurable retention, default 24h) |
| **Directory limits** | Uploads only go to the configured `uploads.directory`; independent of `security.allow_write` |

### 6.5 Chunked / Resumable Uploads

For large files, chunked uploads prevent timeouts and allow resumption after network failures. This is a significant feature and should be evaluated against the **tus protocol** (https://tus.io), an open standard for resumable uploads.

**tus benefits:**
- Open standard with wide client library support
- Resume after connection loss
- Chunked by design
- Well-defined protocol (HTTP-based)

**tus costs:**
- Adds a protocol dependency
- More complex server implementation
- May be over-engineered for most Basil use cases

**Recommendation:** Defer chunked/resumable uploads to 1.2+ or later. The simple XHR upload from Phase 3 handles files up to the configured limit without issues. If user demand materializes for large file support, evaluate tus protocol adoption as a separate design.

### 6.6 Estimated Effort

- **Upload endpoint (Go):** ~8-12 hours
- **Config schema:** ~2 hours
- **Token signing/validation:** ~3 hours
- **Orphan cleanup:** ~3 hours
- **Client JS integration:** ~4 hours
- **Tests:** ~8 hours

**Total: ~30-40 hours** — this is a significant feature.

---

## 7. Full API Reference (All Phases)

### 7.1 Props

| Prop | Phase | Type | Default | Description |
|------|-------|------|---------|-------------|
| `name` | 1 | string | — | Field name (required) |
| `label` | 1 | string | — | Label text |
| `accept` | 1 | string | `null` | Accepted MIME types/extensions |
| `multiple` | 1 | bool | `false` | Allow multiple files |
| `capture` | 1 | string | `null` | Camera direction (`"user"`, `"environment"`) |
| `required` | 1 | bool | `false` | Required indicator |
| `hint` | 1 | string | `null` | Help text |
| `error` | 1 | string | `null` | Error message |
| `id` | 1 | string | `name` | Override ID |
| `disabled` | 1 | bool | `false` | Disabled state |
| `maxSize` | 2 | integer | `null` | Max file size in bytes |
| `maxFiles` | 2 | integer | `null` | Max file count |
| `current` | 2 | string | `null` | Current file display text |
| `currentUrl` | 2 | string | `null` | Link to current file |
| `dragDrop` | 3 | bool | `false` | Enable drag-drop zone |
| `preview` | 3 | bool | `false` | Show image thumbnails |
| `upload` | 4 | bool | `false` | Use managed async upload |

### 7.2 Server-Side Request Context

After multipart form submission, the handler has access to:

```parsley
let {request} = import @basil/http

// File metadata (Phase 1 — already available)
request.files.avatar           // {filename: "photo.jpg", size: 102400, headers: {...}}
request.files.attachments      // [{filename: ...}, {filename: ...}] (multiple)

// Save file to disk (Phase 1 — new builtin needed)
request.saveFile("avatar", "./uploads/")   // Returns {path, size} or error

// File ID from managed upload (Phase 4)
request.form.avatar            // "abc123" (the file ID, not the file itself)
```

---

## 8. Accessibility

### 8.1 Requirements

FileField meets the same accessibility standards as all other prelude form fields:

| Requirement | Implementation |
|-------------|---------------|
| **Label association** | `<label for={id}>` linked to `<input id={id}>` |
| **Required indication** | Visual `*` with `aria-hidden="true"` + `aria-required="true"` on input |
| **Error association** | `aria-describedby` pointing to error `<small>` + `aria-invalid="true"` |
| **Hint association** | `aria-describedby` pointing to hint `<small>` |
| **File list announcement** | `aria-live="polite"` on dynamically populated file list (Phase 2+) |
| **Drag-drop keyboard access** | Dropzone has `tabindex="0"`, `role="button"`, Enter/Space triggers file picker (Phase 3) |
| **Progress announcement** | Upload progress uses `<progress>` element (natively accessible) (Phase 3) |
| **Cancel button labeling** | Cancel buttons have `aria-label="Cancel upload for {filename}"` (Phase 3) |

### 8.2 Screen Reader Flow

1. User tabs to field → "Profile Photo, required, file input"
2. User activates input → native file picker opens
3. User selects file → file list announced via live region: "photo.jpg, 2.1 megabytes"
4. If validation error → "Profile Photo, invalid, File must be an image"

---

## 9. Pico CSS Compatibility

### 9.1 Phase 1: Native Styling

Pico CSS styles `<input type="file">` within a `<label>` → `<input>` structure. Our output matches this exactly. No custom CSS needed.

Pico CSS also styles `<small>` after inputs as hint/validation text with automatic color based on `aria-invalid`. This is the same pattern as TextField — it works out of the box.

### 9.2 Phase 2+: Minimal Structural Classes

When adding the dropzone and file list, we use semantic elements that Pico CSS handles:

- `<progress>` — styled natively by Pico
- `<ul>` / `<li>` — basic list styling
- `<button>` — styled natively by Pico

The only class names introduced are for JS targeting (`file-field-drop`, `file-field-list`, `file-field-current`) and optional CSS hooks. These are **not required for the component to look correct** — they're entry points for custom styling.

---

## 10. Testing Strategy

### 10.1 Phase 1 Tests

```go
// Render tests
func TestFileField_Basic(t *testing.T)           // name + label → correct HTML
func TestFileField_WithAccept(t *testing.T)       // accept prop renders on input
func TestFileField_Multiple(t *testing.T)         // multiple prop renders correctly
func TestFileField_WithCapture(t *testing.T)      // capture prop renders on input
func TestFileField_Required(t *testing.T)         // required indicator + aria-required
func TestFileField_WithHint(t *testing.T)         // hint rendered with correct ID linkage
func TestFileField_WithError(t *testing.T)        // error replaces hint, aria-invalid set
func TestFileField_WithHintAndError(t *testing.T) // error wins over hint
func TestFileField_CustomId(t *testing.T)         // id prop overrides generated ID
func TestFileField_Disabled(t *testing.T)         // disabled prop passes through
func TestFileField_ExtraAttrs(t *testing.T)       // ...inputAttrs spread to <input>

// Accessibility tests
func TestFileField_AriaDescribedBy(t *testing.T)  // correct aria-describedby linkage
func TestFileField_LabelFor(t *testing.T)         // label[for] matches input[id]
```

### 10.2 Phase 2+ Tests

- Client-side validation (max size, max files) — unit tests for the JS functions
- File list rendering — DOM integration tests
- Drag-drop — interaction tests (may need browser testing framework)
- Upload progress — mock XHR tests

---

## 11. Implementation Order

### Phase 1 (1.0) — Recommended

1. Create `server/prelude/components/file_field.pars`
2. Register in `pkg/parsley/evaluator/stdlib_html.go`
3. Add auto-enctype fix to `server/prelude/js/basil.js`
4. Write render tests
5. Add documentation to `docs/parsley/reference.md` and update `CHEATSHEET.md`
6. Add usage example in FAQ: "How do I handle file uploads?"

### Phase 2 (1.1)

1. Add `maxSize` / `maxFiles` props and `data-*` attributes
2. Add client-side validation JS to `basil.js`
3. Add file list display JS
4. Add `current` / `currentUrl` props for edit mode
5. Write integration tests

### Phase 3 (1.1+)

1. Add `dragDrop` prop and dropzone HTML
2. Add drag-drop JS to `basil.js`
3. Add `preview` prop and thumbnail JS
4. Add XHR upload with progress
5. Write interaction tests

### Phase 4 (1.2+)

1. Design `uploads` config schema
2. Implement managed upload endpoint in Go
3. Implement token signing/validation
4. Implement orphan file cleanup
5. Add `upload` prop to FileField
6. Integrate with Parts runtime
7. Write end-to-end tests

---

## 12. Open Questions

1. **`saveFile` builtin scope:** Should `request.saveFile()` be part of the `@basil/http` module, a standalone `@basil/fs` module, or a method on the request object? Needs a FEAT spec.

2. **Image processing:** Should Basil provide thumbnail generation server-side (e.g., for `preview` in non-JS contexts)? This would add an `imaging` dependency. Probably defer to plugins.

3. **Storage backends:** Phase 4 assumes local filesystem storage. Should the upload system be abstracted to support S3/GCS/Azure Blob? Probably not for 1.x — local-first, cloud via plugins.

4. **Virus scanning hook:** Should the managed upload endpoint support a pre-save hook for virus scanning? Useful for production apps. Could be a config option: `uploads.scan_command: "clamdscan --no-summary"`.

5. **`<field/>` tag integration:** The Level 4 `<field/>` auto-generation system (FEAT-145) currently has no mapping for a `file` schema type. When should this be added?

6. **Form-level vs field-level `enctype`:** Should `FileField` be able to signal to a parent `Form` that `enctype` is needed? This requires a component communication mechanism that doesn't exist yet.

---

## 13. Recommendation

**Ship Phase 1 with 1.0.** It is ~3 hours of work, follows the exact existing patterns, and closes the most visible gap in the form component set. Users who need file uploads today can use the component immediately with `enctype="multipart/form-data"` on their form.

Phase 2 is a natural follow-up for 1.1 — it adds the quality-of-life features (validation, file list, current file display) that make file uploads pleasant without architectural changes.

Phases 3 and 4 are more substantial and should be evaluated against 1.1/1.2 priorities. Phase 3 (drag-drop, previews, progress) is high user value but moderate effort. Phase 4 (managed uploads) is the "magic" experience but requires significant Go implementation.