---
title: Get Started
---

# Get started

Ten minutes, one install, and a working website at the end. Let's go.

## 1. Install

One line installs both `basil` (the server) and `pars` (the language):

```bash
curl -fsSL https://herbaceous.net/install.sh | sh
```

Prefer to see what you're running first? [Read the script](https://github.com/sambeau/basil/blob/main/scripts/install.sh), or grab a binary from [the releases page](https://github.com/sambeau/basil/releases) — macOS, Linux, and Windows, Apple Silicon and Intel. (On Windows, download the `.zip` and add it to your `PATH`.)

## 2. Sixty seconds of Parsley

Type `pars` to open the REPL, then try a few lines:

```parsley
>> @now + @7d
@2026-07-19T18:06:16Z

>> $19.99 * 3
$59.97

>> [1, 2, 3].map(fn(x){ x * x })
[1, 4, 9]
```

Yes — dates, durations, and money are all real types you can just *write*. `@7d` is seven days. `$19.99` is money (and it won't grow floating-point crumbs when you multiply it). Type `exit` when you're done.

## 3. Your first script

Parsley was made for turning data into web pages. Save this as `users.pars`:

```parsley
let Page = fn({title, users}) {
    <html>
        <body>
            <h1>title</h1>
            <ul>
                for (user in users) {
                    <li><b>user.name</b> " — " user.email</li>
                }
            </ul>
        </body>
    </html>
}

let emailList <== CSV(@./email-list.csv)
<Page title="Active Users" users={emailList}/>
```

And this as `email-list.csv`:

```csv
name,email
Luke,luke@example.com
Leia,leia@example.com
```

Run it:

```bash
pars --raw users.pars
```

```html
<html><body><h1>Active Users</h1><ul><li><b>Luke</b> — luke@example.com</li><li><b>Leia</b> — leia@example.com</li></ul></body></html>
```

That's the whole program. HTML is part of the language, `<==` slurps the CSV into a table, and the `for` loop runs right inside the markup. No template language, no imports, no build.

## 4. Your first website

```bash
basil --init myapp
cd myapp
basil --dev
```

Open [http://localhost:8080](http://localhost:8080). There's your site.

Now open `site/index.pars` in your editor and change it:

```parsley
<h1>"My first Basil site"</h1>
<p>"Today is " + @now.dayName + "."</p>
```

Save, and the browser refreshes itself. That's the loop: edit, save, look. There is no step four.

## 5. Add a page

Routes are folders. To make `/about`, create `site/about/` with a file named after the folder:

```parsley
// site/about/about.pars
<h1>"About"</h1>
<p>"This page is one file."</p>
```

Visit [http://localhost:8080/about](http://localhost:8080/about) — it's already live. (A file called `index.pars` works too; the folder-named version is just easier to spot in your editor when you have ten tabs open.)

## 6. Put it on the internet

When the folder is ready to be a website, it does not have to be restructured to
get there. Run one command on a server you can SSH into:

```bash
basil --init /srv/mysite --server --host mysite.example.com --admin [your name]
```

…point your domain at that machine, add it as a Git remote from your folder, and
publish:

```bash
git remote add origin https://[your name]@mysite.example.com/.git
git push -u origin main
basil publish
```

Basil fetches its own HTTPS certificate, checks each release before it goes live,
and keeps every release so you can roll back. From then on it is `git push` to
share and `basil publish` to go live.

Both ways round — starting on the server, or starting on your laptop like this —
are written out step by step in [Deployment](basil/deployment.html#two-ways-to-start).

## Where next?

- [Getting Started with Parsley](manual/getting-started.html) — a fuller tour of the language: variables, functions, dictionaries, components
- [The Parsley Manual](manual/index.html) — every type, operator, and builtin
- [The REPL](manual/features/repl.html) — output modes, tab completion, and `:help`
- [File I/O](manual/features/file-io.html) — reading and writing JSON, CSV, and Markdown
- [Deployment](basil/deployment.html) — HTTPS, push-to-deploy over Git, rolling back
- [Why does this exist?](why.html) — the story, and why the quirks are quirks

Stuck? Something confusing? [Open an issue](https://github.com/sambeau/basil/issues) — early feedback is a gift.
