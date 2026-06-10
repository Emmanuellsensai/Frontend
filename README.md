# ASCII Art Web — Complete Code Breakdown

A beginner-friendly, line-by-line explanation of your full-stack ASCII art
generator: the Go backend, the HTML template, and the CSS. Read this once
top-to-bottom and you'll be able to rebuild the whole thing from memory.

> The code shown here is the **corrected, working version** with the
> **two-handler design**: one route for showing the page (GET) and a separate
> route for generating art (POST). Each handler does exactly one job.

---

## 1. The big picture: how a web app works

Before any line of code, hold this mental model:

```
  BROWSER (client)                          YOUR GO PROGRAM (server)
  ----------------                          ------------------------
  1. Visit localhost:8080      ──GET /──▶    2. homeHandler builds the page
  4. Browser shows the form    ◀──HTML───    3. and sends it back

  5. Submit the form      ─POST /ascii-art▶  6. asciiHandler reads your text,
                                                runs the engine, and
  8. Browser shows the art ◀──HTML────────   7. sends a new page WITH the art
```

Two kinds of message matter:

- **GET** — "give me the page" (what happens when you first visit `/`).
- **POST** — "here's some data, do something with it" (what the form sends to
  `/ascii-art`).

In this design, **each kind of request has its own handler and its own URL**:

- `GET /` → `homeHandler` → just show the page
- `POST /ascii-art` → `asciiHandler` → generate art and show the result

This is the **single-responsibility** idea: one function, one job. It makes
the code easier to read, test, and debug.

**Static vs dynamic server** (the Live Server lesson): a *static* server
(Live Server, GitHub Pages) just hands files to the browser untouched — it
can't run Go or process `{{ }}` tags. A *dynamic* server (your Go program)
runs code to *build* the page before sending it. Templates and form handling
need the dynamic server.

---

## 2. Project structure

```
Frontend/
├── main.go              ← the web server + TWO handlers (the "brain")
├── ascii/
│   └── generator.go     ← the ASCII art engine (your reusable logic)
├── banners/
│   ├── standard.txt     ← font data files (the letter shapes)
│   ├── shadow.txt
│   └── thinkertoy.txt
├── index.html           ← the page template (HTML + {{ }} placeholders)
├── static/
│   └── styles.css       ← the styling
└── go.mod               ← declares the module name ("stylize")
```

**Why the folders matter:** Go finds files by *path*. `main.go` references
`"index.html"`, `"banners/standard.txt"`, and serves `static/`. If a file
isn't where the code says it is, you get a 404 (missing CSS) or a panic
(missing template). Paths must match exactly — and Linux is case-sensitive,
so `Standard.txt` ≠ `standard.txt`.

---

## 3. The backend — `main.go`

### 3.1 Package and imports

```go
package main

import (
    "html/template"   // renders HTML pages with {{ }} placeholders, safely
    "net/http"        // the web server: handles requests & responses
    "stylize/ascii"   // YOUR ascii engine (stylize = module name in go.mod)
)
```

- `package main` — every runnable Go program starts here. `main` is special:
  it produces an executable.
- `import` — the first two are from Go's standard library (built in). The
  third, `"stylize/ascii"`, is *your own* `ascii` folder — `stylize` is the
  module name from `go.mod`, and `ascii` is the subfolder.

### 3.2 The data carrier — the struct

```go
type PageData struct {
    Art    string
    Text   string
    Banner string
}
```

A **struct** is a custom type that bundles related values. This is the
*package* you hand to the template. Each field maps to a placeholder:

- `Art` → `{{.Art}}` (the generated art)
- `Text` → `{{.Text}}` (refills the textarea after submit)
- `Banner` → `{{.Banner}}` (remembers the dropdown choice)

**Capital first letters matter.** In Go, capitalized = "exported" (visible
outside the package). The template can only read **capitalized** fields. Write
`art` instead of `Art` and the template can't see it.

### 3.3 Parse the template once, at startup

```go
var tmpl = template.Must(template.ParseFiles("index.html"))
```

This is a **package-level variable** — it runs once when the program starts,
*before* any request. Both handlers share this one `tmpl`.

- `template.ParseFiles("index.html")` — load and parse the HTML template from
  disk. The path must match where the file lives.
- `template.Must(...)` — a helper that **panics** if parsing fails (e.g. file
  not found). A loud crash at startup is good: you find the problem instantly
  instead of on the first request.

> Why at startup and not inside the handler? Re-reading the file from disk on
> every single request is wasteful. Parse once, reuse forever.

### 3.4 The art-generating function

```go
func GenerateArt(text, banner string) (string, error) {
    bannerFile := "banners/" + banner + ".txt"   // e.g. "banners/shadow.txt"

    lines, err := ascii.ReadBanner(bannerFile)    // read the font file
    if err != nil {
        return "", err                            // bubble the error up
    }
    asciiMap := ascii.BuildAsciiMap(lines)        // build letter → shape map
    return ascii.PrintAscii(text, asciiMap), nil  // render the text, no error
}
```

This wraps your three engine functions into one call.

- `func GenerateArt(text, banner string) (string, error)` — takes two strings,
  returns the art **and** an error. Returning an error alongside the result is
  *the* Go pattern; the caller decides what to do with it.
- `bannerFile := "banners/" + banner + ".txt"` — builds the file path from the
  chosen banner. This is why the dropdown values must be lowercase and match
  the real filenames.
- `if err != nil { return "", err }` — if the file is missing, stop and pass
  the error back. `nil` means "no error."
- The final line returns the rendered art with a `nil` error = success.

### 3.5 The HOME handler — GET only, shows the page

```go
func homeHandler(w http.ResponseWriter, r *http.Request) {
    // Go sends ALL unmatched paths to "/", so guard against that:
    if r.URL.Path != "/" {
        http.Error(w, "404 - page not found", http.StatusNotFound)
        return
    }
    // This route is GET only.
    if r.Method != http.MethodGet {
        http.Error(w, "405 - method not allowed", http.StatusMethodNotAllowed)
        return
    }
    tmpl.Execute(w, PageData{}) // empty data → no art shown yet
}
```

- `func homeHandler(w http.ResponseWriter, r *http.Request)` — the standard Go
  handler shape. `w` is where you **write the response** (goes to the browser).
  `r` is the **incoming request** (method, path, form data).
- `if r.URL.Path != "/"` — **important Go gotcha:** the default router treats
  `/` as a *catch-all*. Any path that doesn't match another route (like
  `/banana`) falls through to here. This check returns a proper **404** for
  unknown paths instead of wrongly showing the home page.
- `if r.Method != http.MethodGet` — reject anything that isn't a GET with a
  **405 Method Not Allowed**. So a `POST /` is correctly refused.
- `tmpl.Execute(w, PageData{})` — render the page with an **empty** struct.
  `Art` is empty, so `{{if .Art}}` is false and no art box appears. Just the
  form.

### 3.6 The ASCII handler — POST only, generates art

This is the "its own part" you wanted: one handler dedicated to generating art.

```go
func asciiHandler(w http.ResponseWriter, r *http.Request) {
    // This route is POST only.
    if r.Method != http.MethodPost {
        http.Error(w, "405 - method not allowed", http.StatusMethodNotAllowed)
        return
    }

    text := r.FormValue("text")
    banner := r.FormValue("banner")

    valid := map[string]bool{"standard": true, "shadow": true, "thinkertoy": true}
    if !valid[banner] {
        banner = "standard"
    }

    art, err := GenerateArt(text, banner)
    if err != nil {
        http.Error(w, "500 - could not generate art: "+err.Error(), http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, PageData{
        Art:    art,
        Text:   text,
        Banner: banner,
    })
}
```

Line by line:

- `if r.Method != http.MethodPost` — reject non-POST requests with **405**. So
  if someone types `/ascii-art` in the address bar (a GET), they're refused —
  this endpoint only exists to receive form submissions.
- `r.FormValue("text")` / `r.FormValue("banner")` — read the submitted fields
  **by their `name` attribute**. This is *why* every input needs a `name` —
  it's the key Go uses to find the value.
- The `valid` map + `if !valid[banner]` — a safety allowlist. If the banner
  isn't one of the three known ones, fall back to `"standard"`. Stops
  bad/missing values from crashing the app or reading random files.
- `art, err := GenerateArt(...)` — run the engine; catch the error.
- `if err != nil { http.Error(...) ; return }` — on failure, send a **500**
  and stop. `return` is critical — without it, the code keeps running.
- `tmpl.Execute(w, PageData{ Art: art, Text: text, Banner: banner })` — render
  the page, this time with the art **and** the original input filled in (so
  the form stays populated after submitting). The struct you fill in is the
  struct you render.

> 🐛 **The bug you hit earlier:** you once rendered a *different* struct than
> the one you filled in, so the art got thrown away. **The object you fill in
> must be the object you render.**

### 3.7 The `main` function — register routes & start

```go
func main() {
    http.HandleFunc("/", homeHandler)            // GET  → show page
    http.HandleFunc("/ascii-art", asciiHandler)  // POST → generate art

    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    http.ListenAndServe(":8080", nil)
}
```

- `http.HandleFunc("/", homeHandler)` — wire the home page URL to its handler.
- `http.HandleFunc("/ascii-art", asciiHandler)` — wire the generate URL to its
  handler. **Two routes, two handlers** — the separation you wanted.
- `fs := http.FileServer(http.Dir("static"))` — a file server for the
  `static/` folder.
- `http.Handle("/static/", http.StripPrefix("/static/", fs))` — serve files
  under `/static/`. `StripPrefix` removes `/static/` before lookup, so
  `/static/styles.css` maps to `static/styles.css`. *This is why your `<link>`
  points to `/static/styles.css`.*
- `http.ListenAndServe(":8080", nil)` — start listening on port 8080 and wait
  for requests forever. Visit `http://localhost:8080`.

---

## 4. The engine — `ascii/generator.go`

```go
package ascii   // a reusable library package, separate from main

import (
    "os"        // read files from disk
    "strings"   // string helpers (split, replace, builder)
)
```

`package ascii` is a *library* package (not `main`), so it can't run alone,
but `main.go` imports and uses it. Good design: the engine doesn't know it's
used by a web app — it just turns text into art.

### 4.1 `ReadBanner` — load a font file

```go
func ReadBanner(file string) ([]string, error) {
    data, err := os.ReadFile(file)
    if err != nil {
        return nil, err
    }
    lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
    return lines, nil
}
```

- `os.ReadFile(file)` — read the whole file into `data` (raw bytes).
- `if err != nil { return nil, err }` — file missing → return the error.
- `string(data)` — convert raw bytes to a string.
- `strings.ReplaceAll(..., "\r\n", "\n")` — normalize Windows line endings so
  the file parses the same on every OS.
- `strings.Split(..., "\n")` — split into a slice of lines (one per line).

### 4.2 `BuildAsciiMap` — map each character to its shape

```go
func BuildAsciiMap(lines []string) map[rune][]string {
    asciiMap := make(map[rune][]string)
    char := 32
    for i := 1; i < len(lines); i += 9 {
        asciiMap[rune(char)] = lines[i : i+8]
        char++
    }
    return asciiMap
}
```

Banner files store each character as **8 lines**, separated by a blank line —
so each character block is 9 lines (8 + 1 separator).

- `map[rune][]string` — a dictionary from a character (`rune`) to its 8 lines
  of art. A `rune` is Go's type for a single character.
- `make(map[...])` — create an empty map.
- `char := 32` — ASCII code 32 is a space, the first character in the files.
- `for i := 1; i < len(lines); i += 9` — jump 9 lines at a time (each block).
- `asciiMap[rune(char)] = lines[i : i+8]` — store this character's 8 lines.
  `lines[i : i+8]` is a **slice**: lines `i` up to (not including) `i+8`.
- `char++` — advance to the next ASCII character.

Result: a lookup table where `asciiMap['A']` gives the 8 lines that draw "A".

### 4.3 `PrintAscii` — render the text into art

```go
func PrintAscii(text string, asciiMap map[rune][]string) string {
    var result strings.Builder

    text = strings.ReplaceAll(text, "\r\n", "\n")   // normalize newlines
    for i, line := range strings.Split(text, "\n") {
        if line == "" {
            if i != 0 {
                result.WriteString("\n")
            }
            continue
        }
        for row := 0; row < 8; row++ {
            for _, ch := range line {
                if art, ok := asciiMap[ch]; ok {
                    result.WriteString(art[row])
                }
            }
            result.WriteString("\n")
        }
    }
    return result.String()
}
```

- `var result strings.Builder` — an efficient way to build a long string
  piece by piece (faster than `+=` in a loop).
- `text = strings.ReplaceAll(text, "\r\n", "\n")` — normalize input newlines.
- `for i, line := range strings.Split(text, "\n")` — split on **real
  newlines** and loop over each line.
- `if line == "" { ... continue }` — handle blank lines: write a newline, skip.
- `for row := 0; row < 8; row++` — characters are 8 rows tall, so build the
  output **one row at a time across all characters** (row 1 of every letter,
  then row 2, ...).
- `for _, ch := range line` — loop over each character.
- `if art, ok := asciiMap[ch]; ok` — look up the character. `ok` is `true` if
  it exists; this safely skips unknown characters instead of crashing.
- `result.WriteString(art[row])` — add this character's row to the output.
- `return result.String()` — hand back the finished art.

> 🐛 **The newline lesson:** split on `"\n"` (real newline), not `"\\n"` (two
> literal characters). A *command line* passes `\n` as two chars; a *textarea*
> sends one real newline byte. Match the source.

---

## 5. The frontend — `index.html`

A normal HTML page **plus** Go template tags (`{{ }}`) the server fills in.

### 5.1 The head

```html
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ascii Art Gen</title>
    <link rel="stylesheet" href="/static/styles.css">
    <!-- Google Fonts links -->
</head>
```

- `<meta charset="utf-8">` — display all characters correctly.
- `<meta name="viewport" ...>` — scale correctly on phones.
- `<link rel="stylesheet" href="/static/styles.css">` — load your CSS. The
  `/static/` path matches the Go file server route.

### 5.2 Header + nav

```html
<header>
    <nav class="navbar">
        <span class="logo">Ascii Art Gen</span>
        <ul class="navlinks">
            <li><a href="#generate">Generate</a></li>
            <li><a href="#features">Features</a></li>
        </ul>
    </nav>
</header>
```

- `<nav class="navbar">` — CSS makes it a flex row (logo left, links right).
- `<a href="#generate">` — a *same-page* jump link to `id="generate"`.

### 5.3 The hero

```html
<section class="hero">
    <h1>Turn Text Into Ascii Art</h1>
    <p>Type a word, pick a banner type and generate the art instantly</p>
</section>
```

The "hero" is just the styled banner section at the top — no special tag. It
holds the page's single `<h1>`.

### 5.4 The form — note the action points at the ASCII route

```html
<form action="/ascii-art" method="post">
    <label for="text">Enter text here:</label>
    <textarea id="text" name="text">{{.Text}}</textarea>

    <label for="banner">Select banner:</label>
    <select id="banner" name="banner">
        <option value="standard"   {{if eq .Banner "standard"}}selected{{end}}>Standard</option>
        <option value="thinkertoy" {{if eq .Banner "thinkertoy"}}selected{{end}}>Thinkertoy</option>
        <option value="shadow"     {{if eq .Banner "shadow"}}selected{{end}}>Shadow</option>
    </select>

    <button type="submit">Generate</button>
</form>
```

- `<form action="/ascii-art" method="post">` — submitting sends a **POST** to
  `/ascii-art`, which `asciiHandler` catches. **The `action` must match the
  route you registered in `main`.**
- `<textarea ... name="text">{{.Text}}</textarea>` — multi-line input.
  `name="text"` is the key Go reads. `{{.Text}}` refills the box after submit.
- `<select name="banner">` — each `<option value="...">` sends its lowercase
  `value`, matching the filenames.
- `{{if eq .Banner "standard"}}selected{{end}}` — Go template logic: if the
  remembered banner equals this option, add `selected` so the dropdown keeps
  your choice after submit.

> 🐛 **Option fixes:** use `selected` (not `checked` — that's for checkboxes),
> and put **only plain text** inside `<option>` (no `<br>` or `</label>`).

### 5.5 The result area

```html
<div class="artout">
    {{if .Art}}
    <pre class="art-output">{{.Art}}</pre>
    {{end}}
</div>
```

- `{{if .Art}} ... {{end}}` — only render when `Art` isn't empty (after a
  successful generate). On the home page (GET), nothing shows.
- `<pre class="art-output">{{.Art}}</pre>` — **`<pre>` is essential.** Normal
  HTML collapses spaces and ignores line breaks, which destroys ASCII art
  alignment. `<pre>` preserves every space and newline exactly.

> Go's `html/template` auto-escapes `<`, `>`, `&`, so art with those displays
> as literal text instead of breaking your HTML. That safety is on by default.

### 5.6 Features grid + footer

```html
<section id="features" class="grid">
    <div class="card c1"> <h3>...</h3> <p>...</p> </div>
    <div class="card c2"> ... </div>
    <div class="card c3"> ... </div>
</section>

<footer>
    <p>&copy; 2026 Usang Emmanuel</p>
</footer>
```

- `class="grid"` — CSS turns this into a responsive grid (cards reflow by
  screen width).
- `&copy;` — an HTML entity that renders ©.
- `<footer>` sits **outside** `<main>` — a page-level landmark.

---

## 6. The styling — `styles.css` (key concepts)

### 6.1 The reset
```css
* { box-sizing: border-box; }
```
`width` includes padding and border, so sizing behaves intuitively.

### 6.2 Centered, readable body
```css
body { max-width: 900px; margin: 0 auto; padding: 16px; line-height: 1.6; }
```
`margin: 0 auto` centers a block **only with a `max-width`** — they pair up.

### 6.3 Flexbox nav (logo left, links right)
```css
.navbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    position: sticky; top: 0; z-index: 10;
}
```
`space-between` pushes children to opposite ends. `sticky; top: 0` pins it.

### 6.4 Responsive grid
```css
.grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
}
```
Fit as many ≥200px columns as fit; collapse to one on a phone — no media query.

### 6.5 Smooth hover
```css
button { transition: all 0.2s ease; }
button:hover { transform: translateY(-2px); box-shadow: 0 6px 12px rgba(0,0,0,0.2); }
```
`transition` goes on the **resting** state so it animates in *and* out.

### 6.6 The art box
```css
.art-output {
    font-family: 'Courier New', monospace;   /* every char same width */
    white-space: pre;                         /* preserve spaces/newlines */
    overflow-x: auto;                         /* scroll if too wide */
    padding: 16px;
}
```
**Monospace is non-negotiable for ASCII art** — equal-width characters keep
the alignment.

---

## 7. The full request lifecycle (two handlers)

**First visit — `GET /`:**
1. Browser requests `http://localhost:8080/`.
2. `homeHandler` runs. Path is `/` and method is GET → both checks pass.
3. `tmpl.Execute(w, PageData{})` sends the page with an empty struct.
4. You see the form; `{{if .Art}}` is false, so no art box.

**Submitting the form — `POST /ascii-art`:**
1. You type text, pick a banner, click GENERATE.
2. Browser sends a POST to `/ascii-art` with `text` and `banner`.
3. `asciiHandler` runs. Method is POST → passes the check.
4. `r.FormValue` reads your inputs by `name`; the allowlist validates banner.
5. `GenerateArt` reads the font file, builds the map, renders the art.
6. `tmpl.Execute(w, PageData{Art, Text, Banner})` sends a new page — `{{if .Art}}`
   is now true, so the art appears and the form remembers your input.

**Edge cases now handled:**
- `GET /banana` → `homeHandler`'s path guard returns **404**.
- `POST /` → `homeHandler`'s method check returns **405**.
- `GET /ascii-art` → `asciiHandler`'s method check returns **405**.

> After submitting, your address bar shows `/ascii-art` — that's normal (it's
> where the form posted). If you ever want it to stay `/`, look up the
> **POST-redirect-GET** pattern. Not needed now.

---

## 8. Quick-reference cheat sheet

| Concept | What it does | Where |
|---|---|---|
| `package main` | Makes a runnable program | top of `main.go` |
| `struct` | Bundles data for the template | `PageData` |
| Capitalized fields | Visible to the template (`{{.Art}}`) | `PageData` |
| `var tmpl = ...Must(ParseFiles)` | Parse template once at startup | package level |
| `homeHandler` | `GET /` → show the page | `main.go` |
| `asciiHandler` | `POST /ascii-art` → generate art | `main.go` |
| `r.URL.Path != "/"` | 404 for unknown paths (catch-all guard) | homeHandler |
| `r.Method != ...` | Reject wrong method with 405 | both handlers |
| `r.FormValue("x")` | Reads form field named `x` | asciiHandler |
| `tmpl.Execute(w, data)` | Render page, fill `{{ }}`, send it | both handlers |
| `http.HandleFunc(path, fn)` | Wire a URL to a function | `main()` |
| `http.FileServer` | Serves static files (CSS) | `main()` |
| `:8080` | The port the server listens on | `main()` |
| `{{.Field}}` | Insert a struct field into HTML | template |
| `{{if .X}}...{{end}}` | Show block only if `X` is set | template |
| `{{if eq .Banner "x"}}selected{{end}}` | Remember dropdown choice | template |
| `name="..."` | Key the server reads the field by | form inputs |
| `action="/ascii-art"` | Where the form POSTs to | form |
| `<pre>` | Preserves spaces/newlines (art!) | result area |
| `display: flex` | Row/column layout | CSS |
| `repeat(auto-fit, minmax())` | Responsive grid | CSS |
| `transition` | Smooth hover animation | CSS |

### HTTP status codes used

| Code | Name | When |
|---|---|---|
| 200 | OK | Default — page rendered fine |
| 404 | Not Found | Unknown path (e.g. `/banana`) |
| 405 | Method Not Allowed | Wrong method (GET on `/ascii-art`, POST on `/`) |
| 500 | Internal Server Error | Art generation failed (bad banner file) |

---

## 9. How to run it

```bash
# From inside the Frontend/ folder:
go run .

# Then open your browser to:
http://localhost:8080
```

- **Do NOT use Live Server** — it can't process Go templates (you'd see raw
  `{{ }}` tags).
- If it panics about `index.html` → the template path is wrong.
- If generating gives a 500 → banner filenames must match the option values
  (lowercase, exact).
- Test all four paths: visit `/`, submit the form, try `/banana` (404), and
  type `/ascii-art` in the bar (405).
- Stop the server: `Ctrl + C`.

---

## 10. The recurring lesson of this whole project

Almost every bug was the **same class**: a tiny mismatch where two ends of a
connection have to agree.

- `for`/`id` must match → label connects to input
- anchor `#name` must match an `id` → jump link works
- option `value` must match the filename → file is found
- the path in code must match where the file lives → template loads
- the form's `action` must match a registered route → POST reaches the handler
- the object you fill in must be the object you render → art shows
- the splitter (`\n` vs `\\n`) must match the input source → newlines work

**The skill isn't memorizing CSS or Go. It's checking that the two ends of
every connection actually line up.** And now your handlers do one job each —
that's the structure real projects are built on.
