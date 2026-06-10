package main

import (
    "html/template"
    "net/http"
    "stylize/ascii"
)

type PageData struct {
    Art    string
    Text   string
    Banner string
}

// Parse the template ONCE at startup, not on every request.
// If index.html is missing, this panics immediately so you notice.
var tmpl = template.Must(template.ParseFiles("index.html"))

func GenerateArt(text, banner string) (string, error) {
    bannerFile := "banners/" + banner + ".txt"
    lines, err := ascii.ReadBanner(bannerFile)
    if err != nil {
        return "", err
    }
    asciiMap := ascii.BuildAsciiMap(lines)
    return ascii.PrintAscii(text, asciiMap), nil
}

// HOME HANDLER — only handles GET on exactly "/"
func homeHandler(w http.ResponseWriter, r *http.Request) {
    // Go's default router sends ALL unmatched paths to "/", so guard it:
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

// ASCII HANDLER — its own part, only handles POST on "/ascii-art"
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

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", homeHandler)            // GET  → show page
    mux.HandleFunc("/ascii-art", asciiHandler)  // POST → generate art

    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    http.ListenAndServe(":8083", mux)
}