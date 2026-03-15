package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// componentSource caches loaded component source files.
var (
	componentSources   = make(map[string]string)
	componentSourcesMu sync.Mutex
)

// loadComponent loads a prelude component source file by its filename (e.g. "button.pars").
// Results are cached since the files don't change during a test run.
func loadComponent(t *testing.T, filename string) string {
	t.Helper()
	componentSourcesMu.Lock()
	defer componentSourcesMu.Unlock()

	if src, ok := componentSources[filename]; ok {
		return src
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to determine test file path")
	}
	componentPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "server", "prelude", "components", filename)
	data, err := os.ReadFile(componentPath)
	if err != nil {
		t.Fatalf("failed to load component %s: %v", filename, err)
	}
	componentSources[filename] = string(data)
	return string(data)
}

// evalComponent evaluates one or more component source files followed by test code.
// Returns the output string and any error encountered (parse or evaluation).
func evalComponent(t *testing.T, files []string, code string) (string, error) {
	t.Helper()
	var sb strings.Builder
	for _, f := range files {
		sb.WriteString(loadComponent(t, f))
		sb.WriteString("\n")
	}
	sb.WriteString(code)

	result, err := parsley.Eval(sb.String())
	if err != nil {
		return "", err
	}
	if result.Value.Type() == evaluator.ERROR_OBJ {
		return "", &parsley.RuntimeError{Message: result.Value.Inspect()}
	}
	return result.Value.Inspect(), nil
}

func TestPreludeSmoke(t *testing.T) {
	tests := []struct {
		name     string
		files    []string // component files to load
		code     string   // test code to evaluate after loading component(s)
		contains []string // strings the output must contain
		wantErr  string   // if non-empty, expect an error containing this substring (known component bugs)
	}{
		// ── A ──────────────────────────────────────────────────────────
		{
			name:  "A",
			files: []string{"a.pars"},
			code:  `<A href="/about">"About"</A>`,
			contains: []string{
				"<a",
				`href="/about"`,
				"About",
			},
		},
		{
			name:  "A/external",
			files: []string{"a.pars"},
			code:  `<A href="https://example.com" external={true}>"Ext"</A>`,
			contains: []string{
				"noopener noreferrer",
				`target="_blank"`,
			},
		},
		// ── Abbr ───────────────────────────────────────────────────────
		{
			name:  "Abbr",
			files: []string{"abbr.pars"},
			code:  `<Abbr title="HyperText Markup Language">"HTML"</Abbr>`,
			contains: []string{
				"<abbr",
				`title="HyperText Markup Language"`,
				"HTML",
			},
		},
		// ── Accordion ──────────────────────────────────────────────────
		{
			name:  "Accordion",
			files: []string{"accordion.pars"},
			code:  `<Accordion name="faq" items={[{title: "Q1", content: "A1"}, {title: "Q2", content: "A2"}]}/>`,
			contains: []string{
				"<details",
				`name="faq"`,
				"<summary>",
				"Q1",
				"A1",
				"Q2",
				"A2",
			},
		},
		{
			name:  "Accordion/empty",
			files: []string{"accordion.pars"},
			code:  `<Accordion name="faq" items={[]}/>`,
			contains: []string{
				"null",
			},
		},
		// ── Blockquote ─────────────────────────────────────────────────
		{
			name:  "Blockquote",
			files: []string{"blockquote.pars"},
			code:  `<Blockquote author="Oscar Wilde">"Be yourself."</Blockquote>`,
			contains: []string{
				"<blockquote",
				"Be yourself.",
				"<cite>",
				"Oscar Wilde",
			},
		},
		// ── Breadcrumb ─────────────────────────────────────────────────
		{
			name:  "Breadcrumb",
			files: []string{"breadcrumb.pars"},
			code:  `<Breadcrumb items={[{label: "Home", href: "/"}, {label: "Products"}]}/>`,
			contains: []string{
				"<nav",
				`aria-label="breadcrumb"`,
				"<ol",
				"Home",
				"Products",
				`aria-current="page"`,
			},
		},
		// ── Button ─────────────────────────────────────────────────────
		{
			name:  "Button",
			files: []string{"button.pars"},
			code:  `<Button>"Click Me"</Button>`,
			contains: []string{
				"<button",
				`type="button"`,
				"Click Me",
			},
		},
		{
			name:  "Button/submit",
			files: []string{"button.pars"},
			code:  `<Button type="submit">"Save"</Button>`,
			contains: []string{
				`type="submit"`,
			},
		},
		// ── Checkbox ───────────────────────────────────────────────────
		// BUG: checkbox.pars uses `.length` (property syntax) on arrays,
		// but Parsley requires `.length()` (method call) for arrays.
		{
			name:    "Checkbox",
			files:   []string{"checkbox.pars"},
			code:    `<Checkbox name="agree" label="I agree"/>`,
			wantErr: "Dot notation can only be used on dictionaries",
		},
		// ── CheckboxGroup ──────────────────────────────────────────────
		// BUG: checkbox_group.pars uses `.length` (property syntax) on arrays.
		{
			name:  "CheckboxGroup",
			files: []string{"checkbox_group.pars"},
			code: `<CheckboxGroup
				name="toppings"
				label="Toppings"
				options={[{value: "cheese", label: "Cheese"}, {value: "peppers", label: "Peppers"}]}
			/>`,
			wantErr: "Dot notation can only be used on dictionaries",
		},
		// ── DataTable ──────────────────────────────────────────────────
		{
			name:  "DataTable",
			files: []string{"data_table.pars"},
			code: `
				let t = table([{name: "Alice", age: 30}])
				<DataTable data={t}/>
			`,
			contains: []string{
				"<table",
				"<thead>",
				"<tbody>",
				"Name",
				"Age",
				"Alice",
				"30",
			},
		},
		{
			name:  "DataTable/empty",
			files: []string{"data_table.pars"},
			code: `
				let t = table([])
				<DataTable data={t}/>
			`,
			contains: []string{
				"No data",
			},
		},
		// ── Details ────────────────────────────────────────────────────
		{
			name:  "Details",
			files: []string{"details.pars"},
			code:  `<Details title="Click to expand">"Hidden content"</Details>`,
			contains: []string{
				"<details",
				"<summary>",
				"Click to expand",
				"Hidden content",
			},
		},
		// ── Dialog ─────────────────────────────────────────────────────
		{
			name:  "Dialog",
			files: []string{"dialog.pars"},
			code:  `<Dialog id="confirm" title="Confirm">"Are you sure?"</Dialog>`,
			contains: []string{
				"<dialog",
				`id="confirm"`,
				"<article>",
				"Confirm",
				"Are you sure?",
				`aria-label="Close"`,
			},
		},
		// ── ErrorSummary ───────────────────────────────────────────────
		{
			name:  "ErrorSummary",
			files: []string{"error_summary.pars"},
			code:  `<ErrorSummary errors={[{field: "email", message: "Enter a valid email"}]}/>`,
			contains: []string{
				`role="alert"`,
				"There is a problem",
				"Enter a valid email",
				`href="#email"`,
			},
		},
		{
			name:  "ErrorSummary/empty",
			files: []string{"error_summary.pars"},
			code:  `<ErrorSummary errors={[]}/>`,
			contains: []string{
				"null",
			},
		},
		// ── Figure ─────────────────────────────────────────────────────
		{
			name:  "Figure",
			files: []string{"figure.pars"},
			code:  `<Figure caption="A photo">"image here"</Figure>`,
			contains: []string{
				"<figure",
				"<figcaption>",
				"A photo",
				"image here",
			},
		},
		// ── Form ───────────────────────────────────────────────────────
		// Form imports @basil/http; without server context, request.csrf is null
		// so the CSRF hidden input is skipped. The form still renders.
		{
			name:  "Form",
			files: []string{"form.pars"},
			code:  `<Form action="/submit">"fields here"</Form>`,
			contains: []string{
				"<form",
				`method="POST"`,
				`action="/submit"`,
				"fields here",
			},
		},
		// ── Icon ───────────────────────────────────────────────────────
		{
			name:  "Icon",
			files: []string{"icon.pars"},
			code:  `<Icon name="search" label="Search"/>`,
			contains: []string{
				"icon icon-search",
				`aria-hidden="true"`,
				`class="sr-only"`,
				"Search",
			},
		},
		// ── Iframe ─────────────────────────────────────────────────────
		{
			name:  "Iframe",
			files: []string{"iframe.pars"},
			code:  `<Iframe src="https://example.com" title="Example"/>`,
			contains: []string{
				"<iframe",
				`src="https://example.com"`,
				`title="Example"`,
				`loading="lazy"`,
			},
		},
		// ── Img ────────────────────────────────────────────────────────
		{
			name:  "Img",
			files: []string{"img.pars"},
			code:  `<Img src="/photo.jpg" alt="A sunset"/>`,
			contains: []string{
				"<img",
				`src="/photo.jpg"`,
				`alt="A sunset"`,
				`loading="lazy"`,
				`decoding="async"`,
			},
		},
		// ── LocalTime ──────────────────────────────────────────────────
		// BUG: local_time.pars calls datetime.format("iso"), but "iso" is not
		// a valid format style. Should use .iso property or .toJSON() instead.
		{
			name:    "LocalTime",
			files:   []string{"local_time.pars"},
			code:    `<LocalTime datetime={datetime("2025-03-15T10:30:00Z")}/>`,
			wantErr: `Invalid style iso for formatDate`,
		},
		// ── Meta ───────────────────────────────────────────────────────
		{
			name:  "Meta",
			files: []string{"meta.pars"},
			code:  `<Meta image="/og.png" url="https://example.com"/>`,
			contains: []string{
				`og:image`,
				"/og.png",
				`rel="canonical"`,
				"https://example.com",
			},
		},
		// ── Nav ────────────────────────────────────────────────────────
		{
			name:  "Nav",
			files: []string{"nav.pars"},
			code:  `<Nav label="Main">"nav items"</Nav>`,
			contains: []string{
				"<nav",
				`aria-label="Main"`,
				"nav items",
			},
		},
		// ── Page ───────────────────────────────────────────────────────
		// Page depends on SkipLink. CSS/Javascript/BasilJS are evaluator
		// built-in tags that return empty strings when no asset bundle is set.
		{
			name:  "Page",
			files: []string{"skip_link.pars", "page.pars"},
			code: `<Page title="Test Page" lang="en">
				<main id="main">"Hello"</main>
			</Page>`,
			contains: []string{
				"<!DOCTYPE html>",
				"<html",
				"<head>",
				"<title>Test Page</title>",
				"<body",
				"Hello",
				"skip-link",
			},
		},
		// ── Pagination ─────────────────────────────────────────────────
		// BUG: pagination.pars calls .floor() on the result of integer division,
		// which is already an integer. .floor() is only defined for floats.
		{
			name:    "Pagination",
			files:   []string{"pagination.pars"},
			code:    `<Pagination current={3} total={100} perPage={10} href="/items?page={page}"/>`,
			wantErr: `Unknown method`,
		},
		{
			name:    "Pagination/singlePage",
			files:   []string{"pagination.pars"},
			code:    `<Pagination current={1} total={5} perPage={10} href="/items?page={page}"/>`,
			wantErr: `Unknown method`,
		},
		// ── RadioGroup ─────────────────────────────────────────────────
		// BUG: radio_group.pars uses `.length` (property syntax) on arrays.
		{
			name:  "RadioGroup",
			files: []string{"radio_group.pars"},
			code: `<RadioGroup
				name="size"
				label="Select size"
				options={[{value: "s", label: "Small"}, {value: "m", label: "Medium"}]}
				value="s"
			/>`,
			wantErr: "Dot notation can only be used on dictionaries",
		},
		// ── RelativeTime ───────────────────────────────────────────────
		// BUG: relative_time.pars calls datetime.format("iso"), which is
		// not a valid format style.
		{
			name:    "RelativeTime",
			files:   []string{"relative_time.pars"},
			code:    `<RelativeTime datetime={datetime("2025-01-01T00:00:00Z")}/>`,
			wantErr: `Invalid style iso for formatDate`,
		},
		// ── SelectField ────────────────────────────────────────────────
		{
			name:  "SelectField",
			files: []string{"select_field.pars"},
			code: `<SelectField
				name="country"
				label="Country"
				options={[{value: "us", label: "USA"}, {value: "uk", label: "UK"}]}
				placeholder="Choose..."
			/>`,
			contains: []string{
				"<label",
				"Country",
				"<select",
				`name="country"`,
				"<option",
				"USA",
				"UK",
				"Choose...",
			},
		},
		// ── SkipLink ───────────────────────────────────────────────────
		{
			name:  "SkipLink",
			files: []string{"skip_link.pars"},
			code:  `<SkipLink/>`,
			contains: []string{
				"<a",
				`href="#main"`,
				"skip-link",
				"Skip to main content",
			},
		},
		// ── SrOnly ─────────────────────────────────────────────────────
		{
			name:  "SrOnly",
			files: []string{"sr_only.pars"},
			code:  `<SrOnly>"Open menu"</SrOnly>`,
			contains: []string{
				`class="sr-only"`,
				"Open menu",
			},
		},
		// ── TextField ──────────────────────────────────────────────────
		{
			name:  "TextField",
			files: []string{"text_field.pars"},
			code:  `<TextField name="email" label="Email" type="email"/>`,
			contains: []string{
				"<label",
				"Email",
				"<input",
				`type="email"`,
				`name="email"`,
			},
		},
		// ── TextareaField ──────────────────────────────────────────────
		{
			name:  "TextareaField",
			files: []string{"textarea_field.pars"},
			code:  `<TextareaField name="bio" label="Biography"/>`,
			contains: []string{
				"<label",
				"Biography",
				"<textarea",
				`name="bio"`,
			},
		},
		// ── Time ───────────────────────────────────────────────────────
		// BUG: time.pars calls datetime.format("iso"), which is not a valid
		// format style. Should use .iso property or .toJSON() instead.
		{
			name:    "Time",
			files:   []string{"time.pars"},
			code:    `<Time value={datetime("2025-03-15T10:30:00Z")}/>`,
			wantErr: `Invalid style iso for formatDate`,
		},
		// ── TimeRange ──────────────────────────────────────────────────
		// BUG: time_range.pars calls datetime.format("iso"), which is not a
		// valid format style.
		{
			name:    "TimeRange",
			files:   []string{"time_range.pars"},
			code:    `<TimeRange start={datetime("2025-03-15T09:00:00Z")} end={datetime("2025-03-15T17:00:00Z")}/>`,
			wantErr: `Invalid style iso for formatDate`,
		},
		// ── Toast ──────────────────────────────────────────────────────
		{
			name:  "Toast",
			files: []string{"toast.pars"},
			code:  `<Toast message="Changes saved" type="success"/>`,
			contains: []string{
				"<article",
				`data-type="success"`,
				"Changes saved",
				`aria-label="Dismiss"`,
			},
		},
		// ── Toasts ─────────────────────────────────────────────────────
		{
			name:  "Toasts",
			files: []string{"toast.pars", "toasts.pars"},
			code:  `<Toasts><Toast message="Hello"/></Toasts>`,
			contains: []string{
				`id="toasts"`,
				`aria-live="polite"`,
				"Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := evalComponent(t, tt.files, tt.code)

			if tt.wantErr != "" {
				// Component has a known bug — verify it fails as expected.
				if err == nil {
					t.Fatalf("expected error containing %q but got none\nOutput: %s", tt.wantErr, output)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q\nGot: %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q\nGot: %s", want, output)
				}
			}
		})
	}
}
